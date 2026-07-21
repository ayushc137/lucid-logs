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
