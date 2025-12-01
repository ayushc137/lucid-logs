package tasks

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	apperrors "github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
	"github.com/lucid-logs/go-backend/internal/test/apitest"
)

type taskServiceStub struct {
	listFn   func(context.Context, string, pagination.Params) (*pagination.Response[*Task], error)
	getFn    func(context.Context, string, string) (*Task, error)
	createFn func(context.Context, *CreateRequest, string) (*Task, error)
	updateFn func(context.Context, string, *UpdateRequest, string) (*Task, error)
	deleteFn func(context.Context, string, string) error
}

func (s *taskServiceStub) List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Task], error) {
	return s.listFn(ctx, userID, params)
}
func (s *taskServiceStub) Get(ctx context.Context, id, userID string) (*Task, error) {
	return s.getFn(ctx, id, userID)
}
func (s *taskServiceStub) Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error) {
	return s.createFn(ctx, req, userID)
}
func (s *taskServiceStub) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error) {
	return s.updateFn(ctx, id, req, userID)
}
func (s *taskServiceStub) Delete(ctx context.Context, id, userID string) error {
	return s.deleteFn(ctx, id, userID)
}

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/"), svc, validator.New())
	return r
}

func TestTasksListSuccess(t *testing.T) {
	svc := &taskServiceStub{
		listFn: func(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Task], error) {
			resp := pagination.NewResponse([]*Task{
				{ID: "tasks:1", Title: "Plan"},
			}, 1, params)
			return &resp, nil
		},
	}
	router := setupRouter(svc)

	req := apitest.JSONRequest(t, http.MethodGet, "/", nil, apitest.WithUser("user:1"))
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}

func TestTasksCreateCategoryMissing(t *testing.T) {
	svc := &taskServiceStub{
		createFn: func(ctx context.Context, req *CreateRequest, userID string) (*Task, error) {
			return nil, apperrors.ErrCategoryNotFound
		},
		listFn: func(context.Context, string, pagination.Params) (*pagination.Response[*Task], error) {
			return nil, nil
		},
		getFn: func(context.Context, string, string) (*Task, error) { return nil, nil },
		updateFn: func(context.Context, string, *UpdateRequest, string) (*Task, error) {
			return nil, nil
		},
		deleteFn: func(context.Context, string, string) error { return nil },
	}
	router := setupRouter(svc)

	req := apitest.JSONRequest(t, http.MethodPost, "/", map[string]any{
		"title":      "Plan",
		"start_date": "2024-11-30T09:00:00Z",
		"end_date":   "2024-11-30T10:00:00Z",
	}, apitest.WithUser("user:1"))
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rr.Code)
	}
}
