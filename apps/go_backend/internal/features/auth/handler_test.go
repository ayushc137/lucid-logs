package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	apperrors "github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
	"github.com/lucid-logs/go-backend/internal/test/apitest"
)

type authServiceStub struct {
	loginFn    func(context.Context, *LoginRequest) (*AuthResponse, error)
	registerFn func(context.Context, *RegisterRequest) (*AuthResponse, error)
}

func (s *authServiceStub) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	return s.loginFn(ctx, req)
}

func (s *authServiceStub) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	return s.registerFn(ctx, req)
}

func (s *authServiceStub) ValidateToken(string) (*SurrealClaims, error) {
	return nil, nil
}

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/"), svc, validator.New())
	return r
}

func TestAuthLoginSuccess(t *testing.T) {
	svc := &authServiceStub{
		loginFn: func(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
			if req.Username != "user@example.com" {
				t.Fatalf("unexpected username %s", req.Username)
			}
			return &AuthResponse{Token: "token", User: "user:123"}, nil
		},
		registerFn: func(context.Context, *RegisterRequest) (*AuthResponse, error) {
			return nil, errors.New("unused")
		},
	}

	router := setupRouter(svc)
	req := apitest.JSONRequest(t, http.MethodPost, "/login", map[string]string{
		"username": "user@example.com",
		"password": "password123",
	})
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var resp struct {
		Data AuthResponse `json:"data"`
	}
	apitest.Decode(t, rr, &resp)
	if resp.Data.Token != "token" {
		t.Fatalf("expected token, got %s", resp.Data.Token)
	}
}

func TestAuthLoginValidationError(t *testing.T) {
	router := setupRouter(&authServiceStub{})

	req := apitest.JSONRequest(t, http.MethodPost, "/login", map[string]string{
		"username": "",
	})
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAuthRegisterConflict(t *testing.T) {
	svc := &authServiceStub{
		registerFn: func(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
			return nil, apperrors.ErrUserExists
		},
		loginFn: func(context.Context, *LoginRequest) (*AuthResponse, error) {
			return nil, errors.New("unused")
		},
	}

	router := setupRouter(svc)
	req := apitest.JSONRequest(t, http.MethodPost, "/register", map[string]string{
		"username": "used@example.com",
		"password": "password123",
	})
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}

	var resp response.APIResponse
	apitest.Decode(t, rr, &resp)
	if resp.Error == nil || resp.Error.Code != apperrors.ErrConflict.Code {
		t.Fatalf("expected conflict error")
	}
}
