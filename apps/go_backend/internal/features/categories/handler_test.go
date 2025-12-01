package categories

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	apperrors "github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
	"github.com/lucid-logs/go-backend/internal/test/apitest"
)

type categoryServiceStub struct {
	listFn   func(context.Context, string, pagination.Params) (*pagination.Response[*Category], error)
	getFn    func(context.Context, string, string) (*Category, error)
	createFn func(context.Context, *CreateRequest, string) (*Category, error)
	updateFn func(context.Context, string, *UpdateRequest, string) (*Category, error)
	deleteFn func(context.Context, string, string) error
}

func (s *categoryServiceStub) List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Category], error) {
	return s.listFn(ctx, userID, params)
}
func (s *categoryServiceStub) Get(ctx context.Context, id, userID string) (*Category, error) {
	return s.getFn(ctx, id, userID)
}
func (s *categoryServiceStub) Create(ctx context.Context, req *CreateRequest, userID string) (*Category, error) {
	return s.createFn(ctx, req, userID)
}
func (s *categoryServiceStub) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Category, error) {
	return s.updateFn(ctx, id, req, userID)
}
func (s *categoryServiceStub) Delete(ctx context.Context, id, userID string) error {
	return s.deleteFn(ctx, id, userID)
}

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/"), svc, validator.New())
	return r
}

func TestCategoriesListSuccess(t *testing.T) {
	svc := &categoryServiceStub{
		listFn: func(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Category], error) {
			resp := pagination.NewResponse([]*Category{
				{ID: "categories:1", Name: "Work", Color: "#fff"},
			}, 1, params)
			return &resp, nil
		},
	}
	router := setupRouter(svc)

	req := apitest.JSONRequest(t, http.MethodGet, "/?limit=10", nil, apitest.WithUser("user:1"))
	rr := apitest.Do(router, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data pagination.Response[*Category] `json:"data"`
	}
	apitest.Decode(t, rr, &resp)
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected items")
	}
}

func TestCategoriesCreateConflict(t *testing.T) {
	svc := &categoryServiceStub{
		createFn: func(ctx context.Context, req *CreateRequest, userID string) (*Category, error) {
			return nil, apperrors.ErrCategoryNameExists
		},
		listFn: func(context.Context, string, pagination.Params) (*pagination.Response[*Category], error) {
			return nil, nil
		},
		getFn: func(context.Context, string, string) (*Category, error) { return nil, nil },
		updateFn: func(context.Context, string, *UpdateRequest, string) (*Category, error) {
			return nil, nil
		},
		deleteFn: func(context.Context, string, string) error { return nil },
	}

	router := setupRouter(svc)
	req := apitest.JSONRequest(t, http.MethodPost, "/", map[string]string{
		"name":  "Work",
		"color": "#fff",
	}, apitest.WithUser("user:1"))

	rr := apitest.Do(router, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d", rr.Code)
	}

	var resp response.APIResponse
	apitest.Decode(t, rr, &resp)
	if resp.Error == nil || resp.Error.Code != apperrors.ErrCategoryNameExists.Code {
		t.Fatalf("expected category exists error")
	}
}
