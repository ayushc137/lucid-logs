// Package database provides the application's database/sql based libSQL connection.
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	turso "turso.tech/database/tursogo"
)

// Config configures a local Turso database and optional cloud synchronization.
type Config struct {
	URL            string
	RemoteURL      string
	AuthToken      string
	MigrationsPath string
	LogQueries     bool
}

// DB owns the database/sql pool and, for synchronized databases, the sync handle.
type DB struct {
	sql        *sql.DB
	syncDB     *turso.TursoSyncDb
	logger     zerolog.Logger
	logQueries bool
}

// New opens the database, configures SQLite-compatible safety pragmas, and applies migrations.
func New(ctx context.Context, cfg Config) (*DB, error) {
	logger := log.With().Str("component", "database").Logger()
	path := cfg.URL
	if path == "" {
		path = "./data/lucid-logs.db"
	}
	// The old tests and callers used SQLite's file::memory: URI. Turso's local
	// driver expects :memory: and otherwise treats the URI as a literal filename.
	if strings.HasPrefix(path, "file::memory:") {
		path = ":memory:"
	}
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	var (
		pool   *sql.DB
		syncDB *turso.TursoSyncDb
		err    error
	)
	if cfg.RemoteURL != "" || cfg.AuthToken != "" {
		if cfg.RemoteURL == "" || cfg.AuthToken == "" {
			return nil, fmt.Errorf("remote URL and auth token must be configured together")
		}
		syncDB, err = turso.NewTursoSyncDb(ctx, turso.TursoSyncDbConfig{
			Path: path, RemoteUrl: cfg.RemoteURL, AuthToken: cfg.AuthToken,
		})
		if err == nil {
			pool, err = syncDB.Connect(ctx)
		}
	} else {
		pool, err = sql.Open("turso", path)
	}
	if err != nil {
		return nil, fmt.Errorf("open libSQL database: %w", err)
	}
	db := &DB{sql: pool, syncDB: syncDB, logger: logger, logQueries: cfg.LogQueries}
	if err := db.configure(ctx); err != nil {
		_ = pool.Close()
		return nil, err
	}
	if cfg.MigrationsPath != "" {
		if err := db.applyMigrations(ctx, cfg.MigrationsPath); err != nil {
			_ = pool.Close()
			return nil, err
		}
	}
	return db, nil
}

func (db *DB) configure(ctx context.Context) error {
	db.sql.SetMaxOpenConns(1)
	for _, statement := range []string{"PRAGMA foreign_keys = ON", "PRAGMA busy_timeout = 5000"} {
		if _, err := db.sql.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure database (%s): %w", statement, err)
		}
	}
	return db.sql.PingContext(ctx)
}

// SQL exposes the standard pool for repositories and transactional operations.
func (db *DB) SQL() *sql.DB { return db.sql }

// Close closes the underlying pool. The context is retained for API compatibility.
func (db *DB) Close(_ context.Context) { _ = db.sql.Close() }

// Push uploads local commits when this is a synchronized database.
func (db *DB) Push(ctx context.Context) error {
	if db.syncDB == nil {
		return nil
	}
	return db.syncDB.Push(ctx)
}

// Pull downloads remote commits when this is a synchronized database.
func (db *DB) Pull(ctx context.Context) error {
	if db.syncDB == nil {
		return nil
	}
	_, err := db.syncDB.Pull(ctx)
	return err
}

func (db *DB) applyMigrations(ctx context.Context, dir string) error {
	if _, err := db.sql.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations %q: %w", dir, err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		sum := sha256.Sum256(body)
		checksum := hex.EncodeToString(sum[:])
		var existing string
		err = db.sql.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version = ?", name).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration %s checksum changed", name)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("inspect migration %s: %w", name, err)
		}
		tx, err := db.sql.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err = tx.ExecContext(ctx, string(body)); err == nil {
			_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum,applied_at) VALUES(?,?,?)", name, checksum, time.Now().UTC().Format(time.RFC3339Nano))
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

var namedParameter = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)

func bindNamed(query string, vars map[string]any) (string, []any, error) {
	if len(vars) == 0 {
		return query, nil, nil
	}
	args := make([]any, 0, len(vars))
	seen := map[string]bool{}
	query = namedParameter.ReplaceAllStringFunc(query, func(token string) string {
		name := token[1:]
		if !seen[name] {
			args = append(args, sql.Named(name, vars[name]))
			seen[name] = true
		}
		return ":" + name
	})
	for name := range seen {
		if _, ok := vars[name]; !ok {
			return "", nil, fmt.Errorf("missing query parameter %q", name)
		}
	}
	return query, args, nil
}

// QueryAll executes parameterized SQL and maps columns to json-tagged struct fields.
func QueryAll[T any](ctx context.Context, db *DB, query string, vars map[string]any) ([]T, error) {
	query, args, err := bindNamed(query, vars)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	rows, err := db.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	if db.logQueries {
		db.logger.Debug().Dur("duration", time.Since(started)).Msg("SQL query executed")
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]T, 0)
	boolColumns := booleanJSONFields[T]()
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		mapped := make(map[string]any, len(columns))
		for i, column := range columns {
			value := values[i]
			if b, ok := value.([]byte); ok {
				value = string(b)
			}
			if text, ok := value.(string); ok && json.Valid([]byte(text)) {
				var decoded any
				if err := json.Unmarshal([]byte(text), &decoded); err == nil {
					value = decoded
				}
			}
			if boolColumns[column] {
				switch number := value.(type) {
				case int64:
					value = number != 0
				case float64:
					value = number != 0
				}
			}
			mapped[column] = value
		}
		encoded, err := json.Marshal(mapped)
		if err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal(encoded, &item); err != nil {
			return nil, fmt.Errorf("decode SQL row: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func booleanJSONFields[T any]() map[string]bool {
	result := map[string]bool{}
	typ := reflect.TypeOf((*T)(nil)).Elem()
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return result
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldType := field.Type
		for fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() != reflect.Bool {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" {
			name = field.Name
		}
		if name != "-" {
			result[name] = true
		}
	}
	return result
}

func QueryFirst[T any](ctx context.Context, db *DB, query string, vars map[string]any) (*T, error) {
	rows, err := QueryAll[T](ctx, db, query, vars)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func QueryScalar[T any](ctx context.Context, db *DB, query string, vars map[string]any) (T, error) {
	var zero T
	query, args, err := bindNamed(query, vars)
	if err != nil {
		return zero, err
	}
	var raw any
	if err := db.sql.QueryRowContext(ctx, query, args...).Scan(&raw); err != nil {
		return zero, err
	}
	encoded, _ := json.Marshal(raw)
	var out T
	if err := json.Unmarshal(encoded, &out); err != nil {
		target := reflect.ValueOf(&out).Elem()
		value := reflect.ValueOf(raw)
		if value.IsValid() && value.Type().ConvertibleTo(target.Type()) {
			target.Set(value.Convert(target.Type()))
			return out, nil
		}
		return zero, err
	}
	return out, nil
}

// Select retrieves a record by its table:value primary key.
func Select[T any](ctx context.Context, db *DB, recordID string) (*T, error) {
	table, _ := ParseRecordID(recordID)
	if !identifier.MatchString(table) {
		return nil, fmt.Errorf("invalid table %q", table)
	}
	return QueryFirst[T](ctx, db, "SELECT * FROM "+table+" WHERE id = :record_id", map[string]any{"record_id": recordID})
}

// Delete deletes a record by primary key. Entity repositories should prefer soft deletes.
func Delete[T any](ctx context.Context, db *DB, recordID string) (*T, error) {
	item, err := Select[T](ctx, db, recordID)
	if err != nil || item == nil {
		return item, err
	}
	table, _ := ParseRecordID(recordID)
	if _, err := db.sql.ExecContext(ctx, "DELETE FROM "+table+" WHERE id = ?", recordID); err != nil {
		return nil, err
	}
	return item, nil
}

// Merge updates allowlisted column names and returns the resulting record.
func Merge[T any](ctx context.Context, db *DB, recordID string, values map[string]any) (*T, error) {
	table, _ := ParseRecordID(recordID)
	if !identifier.MatchString(table) {
		return nil, fmt.Errorf("invalid table %q", table)
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if !identifier.MatchString(key) {
			return nil, fmt.Errorf("invalid column %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assignments := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+1)
	for _, key := range keys {
		value := values[key]
		switch value.(type) {
		case nil, string, []byte, bool, int, int64, float64, time.Time:
		default:
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			value = string(encoded)
		}
		if timestamp, ok := value.(time.Time); ok {
			value = timestamp.UTC().Format(time.RFC3339Nano)
		}
		assignments = append(assignments, key+" = ?")
		args = append(args, value)
	}
	if len(assignments) == 0 {
		return Select[T](ctx, db, recordID)
	}
	args = append(args, recordID)
	if _, err := db.sql.ExecContext(ctx, "UPDATE "+table+" SET "+strings.Join(assignments, ", ")+" WHERE id = ?", args...); err != nil {
		return nil, err
	}
	return Select[T](ctx, db, recordID)
}

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ExtractID strips the optional table prefix from an API record ID.
func ExtractID(value string) string {
	_, id := ParseRecordID(value)
	return id
}

// Record ID helpers remain during the repository migration to preserve table:value API IDs.
func ToStringID(rid models.RecordID) string { return rid.String() }
func MustRecordID(table, raw string) models.RecordID {
	if strings.Contains(raw, ":") {
		if rid, err := models.ParseRecordID(raw); err == nil {
			return *rid
		}
	}
	return models.NewRecordID(table, raw)
}
func NewRecordID(table string, id any) models.RecordID { return models.NewRecordID(table, id) }
func RecordIDFromString(value string) (models.RecordID, error) {
	rid, err := models.ParseRecordID(value)
	if err != nil {
		return models.RecordID{}, fmt.Errorf("invalid record ID %q: %w", value, err)
	}
	return *rid, nil
}
func RecordID(table, raw string) string {
	if strings.Contains(raw, ":") {
		return raw
	}
	return table + ":" + raw
}

// ParseRecordID separates the optional table prefix from an API record ID.
func ParseRecordID(value string) (table, id string) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", value
}

// Create inserts a row using validated identifiers and returns it by its id.
func Create[T any](ctx context.Context, db *DB, table string, data map[string]any) (*T, error) {
	if !identifier.MatchString(table) {
		return nil, fmt.Errorf("invalid table %q", table)
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		if !identifier.MatchString(key) {
			return nil, fmt.Errorf("invalid column %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, key := range keys {
		placeholders[i] = "?"
		args[i] = data[key]
	}
	if _, err := db.sql.ExecContext(ctx, "INSERT INTO "+table+"("+strings.Join(keys, ",")+") VALUES("+strings.Join(placeholders, ",")+")", args...); err != nil {
		return nil, err
	}
	id, ok := data["id"]
	if !ok {
		return nil, fmt.Errorf("create %s requires an id", table)
	}
	return Select[T](ctx, db, fmt.Sprint(id))
}
