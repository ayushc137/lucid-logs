package apitest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/lucid-logs/go-backend/internal/test/apitest"
)

// TestAuthRegisterThenLoginCoversFullUserFlow exercises the real user journey:
// register → receive JWT → use JWT to access a protected route → login again with same credentials.
func TestAuthRegisterThenLoginCoversFullUserFlow(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	// Register a new user
	registerReq := map[string]string{
		"username": "Alice@Example.COM", // mixed case to test normalization
		"password": "supersecret123",
	}
	req := apitest.JSONRequest(t, "POST", "/api/v1/auth/register", registerReq)
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var registerResp struct {
		Data struct {
			Token   string `json:"token"`
			User    string `json:"user"`
			IsAdmin bool   `json:"is_admin"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &registerResp)

	if registerResp.Data.Token == "" {
		t.Fatal("register response missing token")
	}
	if registerResp.Data.User == "" {
		t.Fatal("register response missing user ID")
	}
	if registerResp.Data.IsAdmin {
		t.Error("new user should not be admin")
	}

	// User ID should have the "users:" prefix
	if got := registerResp.Data.User; got[:6] != "users:" {
		t.Errorf("user ID %q does not have 'users:' prefix", got)
	}

	// Use the token to access a protected route
	req = apitest.JSONRequest(t, "GET", "/api/v1/tasks", nil, apitest.WithToken(registerResp.Data.Token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	// Login with lowercase email (normalization check)
	loginReq := map[string]string{
		"username": "alice@example.com",
		"password": "supersecret123",
	}
	req = apitest.JSONRequest(t, "POST", "/api/v1/auth/login", loginReq)
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
			User  string `json:"user"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &loginResp)

	if loginResp.Data.User != registerResp.Data.User {
		t.Errorf("login user %q != register user %q", loginResp.Data.User, registerResp.Data.User)
	}
}

// TestAuthRegisterRejectsDuplicateEmail ensures registering the same email twice fails.
func TestAuthRegisterRejectsDuplicateEmail(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	creds := map[string]string{
		"username": "dup@example.com",
		"password": "password123",
	}

	// First register succeeds
	req := apitest.JSONRequest(t, "POST", "/api/v1/auth/register", creds)
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	// Second register with same email fails
	req = apitest.JSONRequest(t, "POST", "/api/v1/auth/register", creds)
	rr = apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusConflict, "CONFLICT")

	// Also fails with different case
	creds["username"] = "DUP@EXAMPLE.COM"
	req = apitest.JSONRequest(t, "POST", "/api/v1/auth/register", creds)
	rr = apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusConflict, "CONFLICT")
}

// TestAuthLoginRejectsWrongPassword verifies wrong password returns 401.
func TestAuthLoginRejectsWrongPassword(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("pwtest@example.com", "correctpassword")
	if token == "" {
		t.Fatal("failed to register test user")
	}

	// Wrong password
	req := apitest.JSONRequest(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "pwtest@example.com",
		"password": "wrongpassword",
	})
	rr := apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")

	// Non-existent user
	req = apitest.JSONRequest(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "nonexistent@example.com",
		"password": "whatever123",
	})
	rr = apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")
}

// TestAuthRegisterValidatesInput ensures malformed requests are rejected.
func TestAuthRegisterValidatesInput(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	tests := []struct {
		name    string
		payload map[string]string
	}{
		{"missing email", map[string]string{"password": "password123"}},
		{"missing password", map[string]string{"username": "test@example.com"}},
		{"invalid email", map[string]string{"username": "not-an-email", "password": "password123"}},
		{"short password", map[string]string{"username": "test@example.com", "password": "12345"}},
		{"empty payload", map[string]string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := apitest.JSONRequest(t, "POST", "/api/v1/auth/register", tc.payload)
			rr := apitest.Do(app.Router, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestAuthProtectedRouteRequiresToken verifies that protected routes reject unauthenticated requests.
func TestAuthProtectedRouteRequiresToken(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	// No token at all
	req := apitest.JSONRequest(t, "GET", "/api/v1/tasks", nil)
	rr := apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")

	// Malformed header (no Bearer prefix)
	req = apitest.JSONRequest(t, "GET", "/api/v1/tasks", nil, apitest.WithHeader("Authorization", "sometoken"))
	rr = apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")

	// Invalid token
	req = apitest.JSONRequest(t, "GET", "/api/v1/tasks", nil, apitest.WithToken("invalid-token-here"))
	rr = apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")
}

// TestAuthTokenExpiry verifies that expired tokens are rejected.
func TestAuthTokenExpiry(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	// Create a user directly so we can generate an expired token for them
	userID := "users:expiry-test"
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := app.DB.SQL().Exec(
		`INSERT INTO users(id,email,pass,is_admin,preferences,created_at,updated_at) VALUES(?,?,?,0,'{}',?,?)`,
		userID, "expiry@example.com", "hash", now, now,
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Manually craft an expired JWT
	expiredClaims := jwt.MapClaims{
		"ID":  userID,
		"sub": userID,
		"iss": app.Cfg.JWT.Issuer,
		"iat": time.Now().Add(-48 * time.Hour).Unix(),
		"nbf": time.Now().Add(-48 * time.Hour).Unix(),
		"exp": time.Now().Add(-24 * time.Hour).Unix(), // expired 24h ago
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, expiredClaims)
	tokenString, err := token.SignedString([]byte(app.Cfg.JWT.Secret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	req := apitest.JSONRequest(t, "GET", "/api/v1/tasks", nil, apitest.WithToken(tokenString))
	rr := apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")
}

// TestAuthTokenSignedWithWrongSecretRejected ensures tokens from other issuers are rejected.
func TestAuthTokenSignedWithWrongSecretRejected(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("wrongsecret@example.com", "password123")
	if token == "" {
		t.Fatal("failed to register")
	}

	// Decode the token, re-sign with wrong secret
	parsed, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	badToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	badTokenString, err := badToken.SignedString([]byte("wrong-secret"))
	if err != nil {
		t.Fatalf("sign bad token: %v", err)
	}

	req := apitest.JSONRequest(t, "GET", "/api/v1/tasks", nil, apitest.WithToken(badTokenString))
	rr := apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusUnauthorized, "UNAUTHORIZED")
}
