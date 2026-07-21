package config

import "testing"

func TestLoadDatabaseConfigForLocalTurso(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_PATH", "/tmp/lucid-test.db")
	t.Setenv("TURSO_DATABASE_URL", "")
	t.Setenv("TURSO_AUTH_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Path != "/tmp/lucid-test.db" {
		t.Fatalf("Database.Path = %q, want %q", cfg.Database.Path, "/tmp/lucid-test.db")
	}
	if cfg.Database.IsSynced() {
		t.Fatal("Database.IsSynced() = true for local-only configuration")
	}
}

func TestLoadDatabaseConfigForSyncedTurso(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_PATH", "/tmp/lucid-replica.db")
	t.Setenv("TURSO_DATABASE_URL", "libsql://example.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Database.IsSynced() {
		t.Fatal("Database.IsSynced() = false for complete remote configuration")
	}
	if cfg.Database.URL != "libsql://example.turso.io" {
		t.Fatalf("Database.URL = %q", cfg.Database.URL)
	}
	if cfg.Database.AuthToken != "test-token" {
		t.Fatalf("Database.AuthToken = %q", cfg.Database.AuthToken)
	}
}

func TestLoadRejectsIncompleteTursoCredentials(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_PATH", "/tmp/lucid-replica.db")
	t.Setenv("TURSO_DATABASE_URL", "libsql://example.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want incomplete Turso credential error")
	}
}

func TestLoadDatabaseConfigFromLibSQLEnvVars(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("TURSO_DATABASE_URL", "")
	t.Setenv("TURSO_AUTH_TOKEN", "")
	t.Setenv("LIBSQL_LOCAL_PATH", "/tmp/lucid-libsql.db")
	t.Setenv("LIBSQL_URL", "libsql://libsql-named.turso.io")
	t.Setenv("LIBSQL_AUTH_TOKEN", "libsql-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Path != "/tmp/lucid-libsql.db" {
		t.Fatalf("Database.Path = %q, want LIBSQL_LOCAL_PATH value", cfg.Database.Path)
	}
	if cfg.Database.URL != "libsql://libsql-named.turso.io" {
		t.Fatalf("Database.URL = %q, want LIBSQL_URL value", cfg.Database.URL)
	}
	if cfg.Database.AuthToken != "libsql-token" {
		t.Fatalf("Database.AuthToken = %q, want LIBSQL_AUTH_TOKEN value", cfg.Database.AuthToken)
	}
	if !cfg.Database.IsSynced() {
		t.Fatal("Database.IsSynced() = false for complete LIBSQL_URL + LIBSQL_AUTH_TOKEN configuration")
	}
}

func TestLoadLibSQLEnvVarsTakePrecedenceOverLegacy(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_PATH", "/tmp/legacy.db")
	t.Setenv("LIBSQL_LOCAL_PATH", "/tmp/primary.db")
	t.Setenv("LIBSQL_URL", "")
	t.Setenv("LIBSQL_AUTH_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Path != "/tmp/primary.db" {
		t.Fatalf("Database.Path = %q, want LIBSQL_LOCAL_PATH to win over DATABASE_PATH", cfg.Database.Path)
	}
}
