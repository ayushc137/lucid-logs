package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/server"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
)

// TestApp is a fully wired test application with a real database and HTTP router.
type TestApp struct {
	T       *testing.T
	Router  *gin.Engine
	DB      *database.DB
	Cfg     *config.Config
	BaseURL string
}

// NewTestApp spins up a fresh in-memory libSQL database, runs all migrations,
// and returns a ready-to-test application with the real HTTP router.
func NewTestApp(t *testing.T) *TestApp {
	t.Helper()

	// Use a temp file for the database to avoid cross-test pollution from shared in-memory caches
	dbPath := t.TempDir() + "/test.db"
	db, err := database.New(context.Background(), database.Config{
		URL:            dbPath,
		MigrationsPath: "../../../../../db/migrations",
	})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cfg := &config.Config{
		App: config.AppConfig{
			Env:     "development",
			Name:    "Lucid Logs Test",
			Version: "1.0.0-test",
		},
		Server: config.ServerConfig{
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Path:           dbPath,
			MigrationsPath: "../../../../../db/migrations",
		},
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-integration-tests",
			ExpirationHours: 24,
			Issuer:          "lucid-logs-test",
		},
		Admin: config.AdminConfig{
			Username: "admin@example.com",
			Password: "adminadmin",
		},
	}

	router := server.NewRouter(server.Config{
		Cfg:       cfg,
		DB:        db,
		Validator: validator.New(),
	})

	return &TestApp{
		T:       t,
		Router:  router,
		DB:      db,
		Cfg:     cfg,
		BaseURL: "http://localhost:8080",
	}
}

// Close cleans up test resources.
func (app *TestApp) Close() {
	if app.DB != nil {
		app.DB.Close(context.Background())
	}
}

// RequestOption mutates an http.Request before it is sent.
type RequestOption func(*http.Request)

// JSONRequest builds an HTTP request with a JSON body and Content-Type header.
func JSONRequest(t *testing.T, method, path string, body any, opts ...RequestOption) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for _, opt := range opts {
		opt(req)
	}
	return req
}

// WithHeader sets a header on the request.
func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithUser injects an authenticated user into the request context.
// This bypasses the JWT middleware and directly sets the user context.
func WithUser(userID string) RequestOption {
	return func(req *http.Request) {
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, &middleware.AuthenticatedUser{
			UserID: userID,
		})
		*req = *req.Clone(ctx)
	}
}

// WithToken sets the Authorization header with a Bearer token.
func WithToken(token string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// Do executes the request against the handler and returns the recorder.
func Do(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// Decode decodes the JSON response body into v.
func Decode(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// DecodeError decodes an API error response.
func DecodeError(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return resp
}

// AssertStatus checks the HTTP status code.
func AssertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Errorf("status = %d, want %d; body: %s", rr.Code, want, rr.Body.String())
	}
}

// AssertError checks that the response is an error with the expected code.
func AssertError(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	AssertStatus(t, rr, wantStatus)
	resp := DecodeError(t, rr)
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response, got: %v", resp)
	}
	if code, _ := errObj["code"].(string); code != wantCode {
		t.Errorf("error code = %q, want %q", code, wantCode)
	}
}

// RegisterAndLogin registers a new user and returns the auth token.
func (app *TestApp) RegisterAndLogin(email, password string) string {
	app.T.Helper()

	// Register
	req := JSONRequest(app.T, "POST", "/api/v1/auth/register", map[string]string{
		"username": email,
		"password": password,
	})
	rr := Do(app.Router, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		app.T.Fatalf("register failed: status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Token string `json:"token"`
			User  string `json:"user"`
		} `json:"data"`
	}
	Decode(app.T, rr, &resp)
	return resp.Data.Token
}

// CreateTestUser creates a user directly in the database and returns the user ID.
// This is faster than going through the API when you just need a user to exist.
func (app *TestApp) CreateTestUser(email string) string {
	app.T.Helper()
	userID := "users:" + email
	now := "2025-01-01T00:00:00Z"
	_, err := app.DB.SQL().Exec(
		`INSERT INTO users(id,email,pass,is_admin,preferences,created_at,updated_at) VALUES(?,?,?,0,'{}',?,?)`,
		userID, email, "hash-not-needed", now, now,
	)
	if err != nil {
		app.T.Fatalf("create test user: %v", err)
	}
	return userID
}

// AuthHeader returns a header map with the Authorization Bearer token.
func AuthHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

// Path builds a URL path with format args.
func Path(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
