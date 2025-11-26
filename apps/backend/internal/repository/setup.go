package repository

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/surrealdb/surrealdb.go"
)

// SchemaInitOptions controls schema resolve behaviour and admin seeding.
type SchemaInitOptions struct {
	SchemaPath    string
	AppEnv        string
	AdminUsername string
	AdminPassword string
	JWTSecret     string
}

// InitSchema applies the SurrealDB schema defined in db/schema.surql (or DB_SCHEMA_PATH),
// seeds an admin user when configured, and in development starts a watcher that reapplies
// the schema + seed whenever the file changes.
func InitSchema(db *DB, opts SchemaInitOptions) error {
	schemaPath, err := resolveSchemaPath(opts.SchemaPath)
	if err != nil {
		return err
	}

	if err := applySchemaAndSeed(db, schemaPath, opts); err != nil {
		return err
	}

	if strings.ToLower(opts.AppEnv) != "production" {
		startSchemaWatcher(db, schemaPath, opts)
	}

	return nil
}

func resolveSchemaPath(preferred string) (string, error) {
	candidates := make([]string, 0, 4)
	if preferred != "" {
		candidates = append(candidates, preferred)
	}

	if wd, err := os.Getwd(); err == nil {
		dir := wd
		visited := map[string]struct{}{}
		for {
			if _, ok := visited[dir]; ok {
				break
			}
			visited[dir] = struct{}{}
			candidates = append(candidates, filepath.Join(dir, "db", "schema.surql"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			if abs, err := filepath.Abs(candidate); err == nil {
				candidate = abs
			}
		}
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", errors.New("schema file not found; set DB_SCHEMA_PATH or place db/schema.surql alongside the project")
}

func applySchemaAndSeed(db *DB, schemaPath string, opts SchemaInitOptions) error {
	if err := applySchema(db, schemaPath, opts); err != nil {
		return err
	}
	if err := seedAdminUser(db, opts.AdminUsername, opts.AdminPassword); err != nil {
		return err
	}
	return nil
}

func applySchema(db *DB, schemaPath string, opts SchemaInitOptions) error {
	ctx := db.Context()
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file (%s): %w", schemaPath, err)
	}

	schema := strings.TrimSpace(string(content))
	if opts.JWTSecret != "" {
		schema = strings.ReplaceAll(schema, "${JWT_SECRET}", opts.JWTSecret)
	}
	if schema == "" {
		return fmt.Errorf("schema file %s is empty", schemaPath)
	}

	if _, err := surrealdb.Query[interface{}](ctx, db.Client, schema, nil); err != nil {
		if isAlreadyDefinedErr(err) {
			log.Printf("schema already applied (%s): %v", schemaPath, err)
			return nil
		}
		return fmt.Errorf("failed to apply schema (%s): %w", schemaPath, err)
	}

	log.Printf("surreal schema applied from %s", schemaPath)
	return nil
}

func startSchemaWatcher(db *DB, schemaPath string, opts SchemaInitOptions) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("schema watcher disabled: %v", err)
		return
	}

	dir := filepath.Dir(schemaPath)
	if err := watcher.Add(dir); err != nil {
		log.Printf("schema watcher disabled: %v", err)
		watcher.Close()
		return
	}

	cleanPath := filepath.Clean(schemaPath)
	go func() {
		defer watcher.Close()

		var (
			mu    sync.Mutex
			timer *time.Timer
		)

		trigger := func() {
			mu.Lock()
			defer mu.Unlock()
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(250*time.Millisecond, func() {
				if err := applySchemaAndSeed(db, cleanPath, opts); err != nil {
					log.Printf("failed to reload schema: %v", err)
					return
				}
				log.Printf("reload applied for %s", cleanPath)
			})
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Clean(event.Name) != cleanPath {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					trigger()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("schema watcher error: %v", err)
			}
		}
	}()
}

func seedAdminUser(db *DB, username, password string) error {
	ctx := db.Context()
	username = strings.TrimSpace(strings.ToLower(username))
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		return nil
	}

	checkVars := map[string]interface{}{"email": username}
	checkSQL := "SELECT * FROM user WHERE email = $email LIMIT 1"

	check, err := surrealdb.Query[[]map[string]any](ctx, db.Client, checkSQL, checkVars)
	if err != nil {
		return fmt.Errorf("failed to verify admin user: %w", err)
	}

	if len(*check) > 0 && len((*check)[0].Result) > 0 {
		return nil
	}

	createSQL := `
		CREATE user CONTENT {
			email: $email,
			pass: crypto::argon2::generate($password)
		};
	`
	createVars := map[string]interface{}{
		"email":    username,
		"password": password,
	}

	if _, err := surrealdb.Query[interface{}](ctx, db.Client, createSQL, createVars); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	log.Printf("admin user '%s' created for initial access", username)
	return nil
}

func isAlreadyDefinedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists")
}
