package middleware

import (
	"net/http"
	"strings"

	"github.com/daily-journal/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates SurrealDB-issued JWTs and injects claims into the Gin context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must start with 'Bearer '"})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))

		claims, err := auth.ParseToken(tokenString, jwtSecret)
		if err != nil || claims.ID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.ID)
		c.Set("surreal_namespace", claims.NS)
		c.Set("surreal_database", claims.DB)
		c.Set("surreal_token", tokenString)
		c.Next()
	}
}
