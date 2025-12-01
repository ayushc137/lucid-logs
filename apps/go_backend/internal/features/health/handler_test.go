package health

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lucid-logs/go-backend/internal/test/apitest"
)

func TestHealthRoutesCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/"), nil)

	req := apitest.JSONRequest(t, http.MethodGet, "/", nil)
	rr := apitest.Do(r, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}
