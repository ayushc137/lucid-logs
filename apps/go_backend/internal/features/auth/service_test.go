package auth

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/shared/database"
)

func TestRegisterThenLoginWithArgon2Password(t *testing.T) {
	db, err := database.New(context.Background(), database.Config{
		URL: filepath.Join(t.TempDir(), "auth.db"), MigrationsPath: "../../../../../db/migrations",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", Issuer: "test", ExpirationHours: 1}}
	service := NewService(db, cfg)
	registered, err := service.Register(context.Background(), &RegisterRequest{Username: "User@Example.com", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if registered.Token == "" || registered.User == "" {
		t.Fatalf("invalid registration response: %#v", registered)
	}
	loggedIn, err := service.Login(context.Background(), &LoginRequest{Username: "user@example.com", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if loggedIn.User != registered.User {
		t.Fatalf("login user=%q, want %q", loggedIn.User, registered.User)
	}
	if _, err := service.Login(context.Background(), &LoginRequest{Username: "user@example.com", Password: "wrong"}); err == nil {
		t.Fatal("wrong password unexpectedly authenticated")
	}
}
