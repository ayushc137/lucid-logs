// Package database provides SurrealDB connection and query utilities.
//
// This package wraps the SurrealDB Go SDK to provide:
//   - Type-safe CRUD operations using generics
//   - Connection lifecycle management
//   - Query logging for debugging
//   - Record ID utilities with models.RecordID support
//
// # RecordID Usage Convention
//
// Use models.RecordID for any struct field that stores SurrealDB record IDs
// in database-facing structs (e.g., taskDB, categoryDB). Convert to string
// only when crossing API boundaries using ToStringID().
//
// SDK Methods Used:
//   - surrealdb.Select[T]() - Type-safe record selection
//   - surrealdb.Create[T]() - Type-safe record creation
//   - surrealdb.Update[T]() - Type-safe full record updates
//   - surrealdb.Merge[T]()  - Type-safe partial record updates
//   - surrealdb.Delete[T]() - Type-safe record deletion
//   - surrealdb.Query[T]()  - Type-safe raw query execution
//
// See: https://surrealdb.com/docs/sdk/golang
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// =============================================================================
// DATABASE CLIENT
// =============================================================================

// DB wraps the SurrealDB client with logging and type-safe utilities.
//
// Features:
//   - Automatic query logging with duration tracking
//   - Generic methods for type-safe operations
//   - Error wrapping with context
type DB struct {
	client     *surrealdb.DB
	logger     zerolog.Logger
	logQueries bool
	namespace  string
	database   string
}

// Config holds database connection configuration.
type Config struct {
	URL        string // WebSocket URL (e.g., "ws://localhost:8000/rpc")
	Namespace  string // SurrealDB namespace
	Database   string // SurrealDB database
	Username   string // Root username for initial connection
	Password   string // Root password for initial connection
	LogQueries bool   // Whether to log all queries (dev mode)
}

// =============================================================================
// CONNECTION
// =============================================================================

// New creates a new database connection using the SurrealDB SDK.
//
// Connection flow:
//  1. Connect via WebSocket
//  2. Authenticate as root
//  3. Select namespace and database
func New(ctx context.Context, cfg Config) (*DB, error) {
	logger := log.With().Str("component", "database").Logger()

	// Connect to SurrealDB using SDK's New function
	client, err := surrealdb.New(cfg.URL) //nolint:staticcheck // deprecated but functional
	if err != nil {
		logger.Error().Err(err).Str("url", cfg.URL).Msg("failed to connect to SurrealDB")
		return nil, errors.ErrDatabase.Wrap(fmt.Errorf("connection failed: %w", err))
	}

	// Authenticate as root using SDK's SignIn method
	// See: https://surrealdb.com/docs/sdk/golang/methods/signin
	if _, err := client.SignIn(ctx, &surrealdb.Auth{
		Username: cfg.Username,
		Password: cfg.Password,
	}); err != nil {
		logger.Error().Err(err).Msg("failed to authenticate with SurrealDB")
		return nil, errors.ErrDatabase.Wrap(fmt.Errorf("authentication failed: %w", err))
	}

	// SurrealDB v3 requires namespace/database to exist before USE.
	// Define them as root before switching context.
	if _, err := surrealdb.Query[any](ctx, client,
		fmt.Sprintf("DEFINE NAMESPACE IF NOT EXISTS %s", cfg.Namespace), nil); err != nil {
		logger.Warn().Err(err).Str("namespace", cfg.Namespace).Msg("failed to define namespace (may already exist)")
	}
	if _, err := surrealdb.Query[any](ctx, client,
		fmt.Sprintf("DEFINE DATABASE IF NOT EXISTS %s ON NAMESPACE %s", cfg.Database, cfg.Namespace), nil); err != nil {
		logger.Warn().Err(err).Str("database", cfg.Database).Msg("failed to define database (may already exist)")
	}

	// Select namespace and database using SDK's Use method
	// See: https://surrealdb.com/docs/sdk/golang/methods/use
	if err := client.Use(ctx, cfg.Namespace, cfg.Database); err != nil {
		logger.Error().Err(err).
			Str("namespace", cfg.Namespace).
			Str("database", cfg.Database).
			Msg("failed to select namespace/database")
		return nil, errors.ErrDatabase.Wrap(fmt.Errorf("use namespace failed: %w", err))
	}

	logger.Info().
		Str("url", cfg.URL).
		Str("namespace", cfg.Namespace).
		Str("database", cfg.Database).
		Msg("connected to SurrealDB")

	return &DB{
		client:     client,
		logger:     logger,
		logQueries: cfg.LogQueries,
		namespace:  cfg.Namespace,
		database:   cfg.Database,
	}, nil
}

// Client returns the underlying SurrealDB client.
//
// Use this for SDK operations not wrapped by this package, such as:
//   - SignIn/SignUp for user authentication
//   - Live queries
//   - Custom operations
func (db *DB) Client() *surrealdb.DB {
	return db.client
}

// Namespace returns the configured namespace.
func (db *DB) Namespace() string {
	return db.namespace
}

// Database returns the configured database name.
func (db *DB) Database() string {
	return db.database
}

// Close closes the database connection.
func (db *DB) Close(ctx context.Context) {
	db.client.Close(ctx)
	db.logger.Info().Msg("database connection closed")
}

// =============================================================================
// TYPED SDK OPERATIONS
// =============================================================================

// Select retrieves a single record using the SurrealDB SDK's Select method.
//
// This is a type-safe wrapper around surrealdb.Select[T]().
// See: https://surrealdb.com/docs/sdk/golang/methods/select
//
// Example:
//
//	task, err := Select[Task](ctx, db, "tasks:abc123")
func Select[T any](ctx context.Context, db *DB, what string) (*T, error) {
	start := time.Now()

	// Use SDK's generic Select function
	result, err := surrealdb.Select[T](ctx, db.client, what)

	if db.logQueries {
		db.logger.Debug().
			Str("select", what).
			Dur("duration", time.Since(start)).
			Msg("SDK Select executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("select", what).Msg("SDK Select failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result, nil
}

// SelectAll retrieves all records from a table using the SurrealDB SDK.
//
// This returns a slice of records, useful for listing operations.
//
// Example:
//
//	tasks, err := SelectAll[Task](ctx, db, "tasks")
func SelectAll[T any](ctx context.Context, db *DB, table string) ([]T, error) {
	start := time.Now()

	// Use SDK's generic Select function with slice type
	result, err := surrealdb.Select[[]T](ctx, db.client, table)

	if db.logQueries {
		db.logger.Debug().
			Str("select_all", table).
			Dur("duration", time.Since(start)).
			Msg("SDK SelectAll executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("select_all", table).Msg("SDK SelectAll failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	if result == nil {
		return []T{}, nil
	}

	return *result, nil
}

// Create creates a new record using the SurrealDB SDK's Create method.
//
// This is a type-safe wrapper around surrealdb.Create[T]().
// See: https://surrealdb.com/docs/sdk/golang/methods/create
//
// Examples:
//
//	// Create with auto-generated ID
//	task, err := Create[Task](ctx, db, "tasks", data)
//
//	// Create with specific ID
//	task, err := Create[Task](ctx, db, "tasks:myid", data)
func Create[T any](ctx context.Context, db *DB, what string, data any) (*T, error) {
	start := time.Now()

	// Use SDK's generic Create function
	result, err := surrealdb.Create[T](ctx, db.client, what, data)

	if db.logQueries {
		db.logger.Debug().
			Str("create", what).
			Dur("duration", time.Since(start)).
			Msg("SDK Create executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("create", what).Msg("SDK Create failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result, nil
}

// Update replaces a record entirely using the SurrealDB SDK's Update method.
//
// This is a type-safe wrapper around surrealdb.Update[T]().
// See: https://surrealdb.com/docs/sdk/golang/methods/update
//
// NOTE: This replaces the entire record. Use Merge for partial updates.
//
// Example:
//
//	task, err := Update[Task](ctx, db, "tasks:abc123", updatedData)
func Update[T any](ctx context.Context, db *DB, what string, data any) (*T, error) {
	start := time.Now()

	// Use SDK's generic Update function
	result, err := surrealdb.Update[T](ctx, db.client, what, data)

	if db.logQueries {
		db.logger.Debug().
			Str("update", what).
			Dur("duration", time.Since(start)).
			Msg("SDK Update executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("update", what).Msg("SDK Update failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result, nil
}

// Merge partially updates a record using the SurrealDB SDK's Merge method.
//
// This is a type-safe wrapper around surrealdb.Merge[T]().
// See: https://surrealdb.com/docs/sdk/golang/methods/merge
//
// Only provided fields are updated; other fields remain unchanged.
//
// Example:
//
//	task, err := Merge[Task](ctx, db, "tasks:abc123", map[string]any{
//	    "completed": true,
//	})
func Merge[T any](ctx context.Context, db *DB, what string, data any) (*T, error) {
	start := time.Now()

	// Use SDK's generic Merge function
	result, err := surrealdb.Merge[T](ctx, db.client, what, data)

	if db.logQueries {
		db.logger.Debug().
			Str("merge", what).
			Dur("duration", time.Since(start)).
			Msg("SDK Merge executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("merge", what).Msg("SDK Merge failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result, nil
}

// Delete removes a record using the SurrealDB SDK's Delete method.
//
// This is a type-safe wrapper around surrealdb.Delete[T]().
// See: https://surrealdb.com/docs/sdk/golang/methods/delete
//
// Example:
//
//	task, err := Delete[Task](ctx, db, "tasks:abc123")
func Delete[T any](ctx context.Context, db *DB, what string) (*T, error) {
	start := time.Now()

	// Use SDK's generic Delete function
	result, err := surrealdb.Delete[T](ctx, db.client, what)

	if db.logQueries {
		db.logger.Debug().
			Str("delete", what).
			Dur("duration", time.Since(start)).
			Msg("SDK Delete executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("delete", what).Msg("SDK Delete failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result, nil
}

// =============================================================================
// TYPED QUERY OPERATIONS
// =============================================================================

// Query executes a SurrealQL query using the SDK's Query method.
//
// This is a type-safe wrapper around surrealdb.Query[T]().
// See: https://surrealdb.com/docs/sdk/golang/methods/query
//
// The type parameter T is the type of individual result items.
// For queries returning multiple rows, use QueryAll[T].
//
// Example:
//
//	results, err := Query[Task](ctx, db, "SELECT * FROM tasks WHERE completed = $completed", map[string]any{
//	    "completed": true,
//	})
func Query[T any](ctx context.Context, db *DB, sql string, vars map[string]any) (*[]surrealdb.QueryResult[[]T], error) {
	start := time.Now()

	// Use SDK's generic Query function with slice type for result
	// The SDK returns QueryResult where Result is the type parameter
	result, err := surrealdb.Query[[]T](ctx, db.client, sql, vars)

	if db.logQueries {
		db.logger.Debug().
			Str("query", truncateQuery(sql, 150)).
			Dur("duration", time.Since(start)).
			Msg("SDK Query executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("query", truncateQuery(sql, 150)).Msg("SDK Query failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result, nil
}

// QueryFirst executes a query and returns the first result.
//
// Useful for queries expected to return a single record.
//
// Example:
//
//	task, err := QueryFirst[Task](ctx, db, "SELECT * FROM tasks:abc123", nil)
func QueryFirst[T any](ctx context.Context, db *DB, sql string, vars map[string]any) (*T, error) {
	results, err := Query[T](ctx, db, sql, vars)
	if err != nil {
		return nil, err
	}

	if results == nil || len(*results) == 0 {
		return nil, nil
	}

	// Get first result set
	firstResult := (*results)[0]
	if firstResult.Status != "OK" {
		return nil, errors.ErrDatabase.Wrap(fmt.Errorf("query failed: %s", firstResult.Status))
	}

	// Result is []T, get first item
	if len(firstResult.Result) == 0 {
		return nil, nil
	}

	return &firstResult.Result[0], nil
}

// QueryAll executes a query and returns all results from the first result set.
//
// Example:
//
//	tasks, err := QueryAll[Task](ctx, db, "SELECT * FROM tasks WHERE completed = true", nil)
func QueryAll[T any](ctx context.Context, db *DB, sql string, vars map[string]any) ([]T, error) {
	results, err := Query[T](ctx, db, sql, vars)
	if err != nil {
		return nil, err
	}

	if results == nil || len(*results) == 0 {
		return []T{}, nil
	}

	// Get first result set
	firstResult := (*results)[0]
	if firstResult.Status != "OK" {
		return nil, errors.ErrDatabase.Wrap(fmt.Errorf("query failed: %s", firstResult.Status))
	}

	return firstResult.Result, nil
}

// QueryScalar executes a query and returns a scalar value.
//
// Useful for COUNT queries and functions that return single values.
//
// Example:
//
//	count, err := QueryScalar[int64](ctx, db, "RETURN fn::task::count_for_user($user)", map[string]any{"user": userID})
func QueryScalar[T any](ctx context.Context, db *DB, sql string, vars map[string]any) (T, error) {
	var zero T
	start := time.Now()

	// Use SDK's generic Query function with any type
	result, err := surrealdb.Query[any](ctx, db.client, sql, vars)

	if db.logQueries {
		db.logger.Debug().
			Str("query_scalar", truncateQuery(sql, 150)).
			Dur("duration", time.Since(start)).
			Msg("SDK QueryScalar executed")
	}

	if err != nil {
		db.logger.Error().Err(err).Str("query", truncateQuery(sql, 150)).Msg("SDK QueryScalar failed")
		return zero, errors.ErrDatabase.Wrap(err)
	}

	if result == nil || len(*result) == 0 {
		return zero, nil
	}

	firstResult := (*result)[0]
	if firstResult.Status != "OK" {
		return zero, errors.ErrDatabase.Wrap(fmt.Errorf("query failed: %s", firstResult.Status))
	}

	// Result is the scalar value directly
	val := firstResult.Result

	// Handle direct type match
	if v, ok := val.(T); ok {
		return v, nil
	}

	// Handle numeric conversions (SurrealDB often returns float64 or int64)
	if f, ok := val.(float64); ok {
		switch any(zero).(type) {
		case int64:
			return any(int64(f)).(T), nil //nolint:errcheck // type assertion
		case int:
			return any(int(f)).(T), nil //nolint:errcheck // type assertion
		case float32:
			return any(float32(f)).(T), nil //nolint:errcheck // type assertion
		case float64:
			return any(f).(T), nil //nolint:errcheck // type assertion
		}
	}

	// Handle int64 (SurrealDB SDK may return int64 for integers)
	if i, ok := val.(int64); ok {
		switch any(zero).(type) {
		case int64:
			return any(i).(T), nil //nolint:errcheck // type assertion
		case int:
			return any(int(i)).(T), nil //nolint:errcheck // type assertion
		case float32:
			return any(float32(i)).(T), nil //nolint:errcheck // type assertion
		case float64:
			return any(float64(i)).(T), nil //nolint:errcheck // type assertion
		}
	}

	// Handle int
	if i, ok := val.(int); ok {
		switch any(zero).(type) {
		case int64:
			return any(int64(i)).(T), nil //nolint:errcheck // type assertion
		case int:
			return any(i).(T), nil //nolint:errcheck // type assertion
		case float32:
			return any(float32(i)).(T), nil //nolint:errcheck // type assertion
		case float64:
			return any(float64(i)).(T), nil //nolint:errcheck // type assertion
		}
	}

	// Try JSON marshaling as fallback
	data, err := json.Marshal(val)
	if err != nil {
		return zero, fmt.Errorf("cannot convert result: %w", err)
	}
	var t T
	if err := json.Unmarshal(data, &t); err != nil {
		return zero, fmt.Errorf("cannot unmarshal result: %w", err)
	}
	return t, nil
}

// =============================================================================
// RECORD ID UTILITIES
// =============================================================================

// ToStringID converts a models.RecordID to its string representation.
//
// This is the primary way to convert SurrealDB record IDs to strings
// when crossing API boundaries (e.g., in toTask(), toCategory() converters).
//
// Example:
//
//	rid := models.NewRecordID("tasks", "abc123")
//	str := ToStringID(rid) // "tasks:abc123"
func ToStringID(rid models.RecordID) string {
	return rid.String()
}

// MustRecordID creates a models.RecordID from a table name and raw ID string.
//
// Use this when you have a string ID and need to convert it to RecordID
// for database operations. The raw string can be either:
//   - Just the ID portion: "abc123"
//   - Full record ID: "tasks:abc123" (table will be extracted)
//
// Example:
//
//	rid := MustRecordID("tasks", "abc123")           // tasks:abc123
//	rid := MustRecordID("tasks", "tasks:abc123")    // tasks:abc123 (extracts ID)
func MustRecordID(table, raw string) models.RecordID {
	// If raw already contains table prefix, extract just the ID
	if strings.Contains(raw, ":") {
		_, raw = ParseRecordID(raw)
	}
	return models.NewRecordID(table, raw)
}

// NewRecordID is an alias for models.NewRecordID for convenience.
//
// Use this to create record links for relationships (e.g., task.category).
//
// Example:
//
//	categoryLink := NewRecordID("categories", catID)
func NewRecordID(table string, id any) models.RecordID {
	return models.NewRecordID(table, id)
}

// RecordIDFromString parses a string record ID into models.RecordID.
//
// Returns an error if the string cannot be parsed.
//
// Example:
//
//	rid, err := RecordIDFromString("tasks:abc123")
func RecordIDFromString(s string) (models.RecordID, error) {
	rid, err := models.ParseRecordID(s)
	if err != nil {
		return models.RecordID{}, fmt.Errorf("invalid record ID %q: %w", s, err)
	}
	return *rid, nil
}

// RecordID creates a SurrealDB record ID string from table and ID parts.
//
// Deprecated: Use MustRecordID() to get models.RecordID for type safety,
// or NewRecordID() for creating record links.
//
// Examples:
//
//	RecordID("tasks", "abc123")      // "tasks:abc123"
//	RecordID("tasks", "tasks:xyz")   // "tasks:xyz" (unchanged)
func RecordID(table, id string) string {
	if strings.Contains(id, ":") {
		return id
	}
	return table + ":" + id
}

// ParseRecordID splits a record ID string into table and ID parts.
//
// Example:
//
//	table, id := ParseRecordID("tasks:abc123")
//	// table = "tasks", id = "abc123"
func ParseRecordID(recordID string) (table, id string) {
	parts := strings.SplitN(recordID, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", recordID
}

// ExtractID extracts just the ID portion from a record ID string.
//
// Example:
//
//	id := ExtractID("tasks:abc123") // "abc123"
func ExtractID(recordID string) string {
	_, id := ParseRecordID(recordID)
	return id
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// truncateQuery truncates a query string for logging.
func truncateQuery(s string, maxLen int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
