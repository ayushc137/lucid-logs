package handler

import (
	"net/http"

	"github.com/daily-journal/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
// @Summary      Health Check
// @Description  Checks if the API is running
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func HealthCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, gin.H{
		"status":  "ok",
		"service": "daily-journal-backend",
	})
}
