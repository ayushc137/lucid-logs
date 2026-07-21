package database

import (
	"context"
	"testing"
)

func TestNewRunsVersionedMigrationsAndEnforcesForeignKeys(t *testing.T) {
	db, err := New(context.Background(), Config{URL: "file::memory:?cache=shared", MigrationsPath: "../../../../../db/migrations"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())

	var migrations int
	if err := db.SQL().QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatal(err)
	}
	if migrations == 0 {
		t.Fatal("expected at least one applied migration")
	}
	var foreignKeys int
	if err := db.SQL().QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys=%d, want 1", foreignKeys)
	}
}

func TestNamedQueriesScanJSONTaggedStructs(t *testing.T) {
	db, err := New(context.Background(), Config{URL: "file::memory:?cache=shared", MigrationsPath: "../../../../../db/migrations"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())

	if _, err := db.SQL().Exec(`INSERT INTO users (id,email,pass,is_admin,preferences,created_at,updated_at) VALUES ('u1','a@example.com','hash',0,'{}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	type row struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	got, err := QueryFirst[row](context.Background(), db, "SELECT id,email FROM users WHERE id=$id", map[string]any{"id": "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "u1" || got.Email != "a@example.com" {
		t.Fatalf("unexpected row: %#v", got)
	}
}

func TestReferenceMigrationSeedsEmotionsAndSystemUnits(t *testing.T) {
	db, err := New(context.Background(), Config{URL: t.TempDir() + "/seed.db", MigrationsPath: "../../../../../db/migrations"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())

	for table, want := range map[string]int{"emotions": 100, "units": 17} {
		var got int
		if err := db.SQL().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s count = %d, want %d", table, got, want)
		}
	}
}

func TestQueryAllDecodesJSONColumns(t *testing.T) {
	db, err := New(context.Background(), Config{URL: t.TempDir() + "/json.db", MigrationsPath: "../../../../../db/migrations"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	if _, err := db.SQL().Exec(`INSERT INTO users(id,email,pass,preferences,created_at,updated_at) VALUES('users:u1','json@example.com','x','{"timezone":"Asia/Kolkata"}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	type preferences struct {
		Timezone string `json:"timezone"`
	}
	type row struct {
		Preferences preferences `json:"preferences"`
	}
	got, err := QueryFirst[row](context.Background(), db, `SELECT preferences FROM users WHERE id=$id`, map[string]any{"id": "users:u1"})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Preferences.Timezone != "Asia/Kolkata" {
		t.Fatalf("unexpected JSON column: %#v", got)
	}
}

func TestQueryAllDecodesSQLiteBooleans(t *testing.T) {
	db, err := New(context.Background(), Config{URL: t.TempDir() + "/bool.db", MigrationsPath: "../../../../../db/migrations"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	if _, err := db.SQL().Exec(`INSERT INTO users(id,email,pass,is_admin,created_at,updated_at) VALUES('users:u1','admin@example.com','x',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	type row struct {
		IsAdmin bool `json:"is_admin"`
	}
	got, err := QueryFirst[row](context.Background(), db, `SELECT is_admin FROM users WHERE id=$id`, map[string]any{"id": "users:u1"})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.IsAdmin {
		t.Fatalf("unexpected boolean column: %#v", got)
	}
}
