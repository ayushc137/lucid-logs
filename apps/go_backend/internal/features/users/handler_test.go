package users

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
	"github.com/lucid-logs/go-backend/internal/test/apitest"
)

type userServiceStub struct {
	getFn    func(context.Context, string, string) (*User, error)
	updateFn func(context.Context, string, string, *UpdateRequest) (*User, error)
	deleteFn func(context.Context, string, string) error
}

func (s *userServiceStub) Get(ctx context.Context, requesterID, targetID string) (*User, error) {
	return s.getFn(ctx, requesterID, targetID)
}
func (s *userServiceStub) Update(ctx context.Context, requesterID, targetID string, req *UpdateRequest) (*User, error) {
	return s.updateFn(ctx, requesterID, targetID, req)
}
func (s *userServiceStub) Delete(ctx context.Context, requesterID, targetID string) error {
	return s.deleteFn(ctx, requesterID, targetID)
}

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/"), svc, validator.New())
	return r
}

func TestUsersMeSuccess(t *testing.T) {
	svc := &userServiceStub{
		getFn: func(ctx context.Context, requesterID, targetID string) (*User, error) {
			return &User{ID: requesterID, Email: "me@example.com"}, nil
		},
		updateFn: func(context.Context, string, string, *UpdateRequest) (*User, error) { return nil, nil },
		deleteFn: func(context.Context, string, string) error { return nil },
	}

	router := setupRouter(svc)
	req := apitest.JSONRequest(t, http.MethodGet, "/me", nil, apitest.WithUser("user:1"))
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}

func TestUsersDeleteNoContent(t *testing.T) {
	svc := &userServiceStub{
		deleteFn: func(ctx context.Context, requesterID, targetID string) error {
			if requesterID != "user:1" || targetID != "user:1" {
				t.Fatalf("unexpected ids")
			}
			return nil
		},
		getFn:    func(context.Context, string, string) (*User, error) { return nil, nil },
		updateFn: func(context.Context, string, string, *UpdateRequest) (*User, error) { return nil, nil },
	}
	router := setupRouter(svc)

	req := apitest.JSONRequest(t, http.MethodDelete, "/user:1", nil, apitest.WithUser("user:1"))
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", rr.Code)
	}
}
