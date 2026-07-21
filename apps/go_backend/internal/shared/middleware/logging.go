// Package middleware provides HTTP middleware for the application.
//
// Middleware included:
//   - Auth: JWT authentication and user context injection
//   - Logger: Request/response logging
//   - Recovery: Panic recovery with logging
package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// REQUEST LOGGING
// =============================================================================

// Logger is a middleware that logs HTTP requests using zerolog.
//
// Log fields include:
//   - method: HTTP method
//   - path: Request path
//   - status: Response status code
//   - bytes: Response size
//   - duration: Request duration
//   - remote_addr: Client IP
//   - user_agent: Client user agent
//   - request_id: Unique request ID (if present)
//
// Log levels:
//   - INFO: Successful requests (2xx, 3xx)
//   - WARN: Client errors (4xx)
//   - ERROR: Server errors (5xx)
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get trace/request ID if available
		traceID := TraceID(c.Request.Context())
		if traceID == "" {
			traceID = c.GetHeader(traceHeader)
		}

		// Determine log level based on status
		status := c.Writer.Status()
		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		// Build log entry
		event.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", status).
			Int("bytes", c.Writer.Size()).
			Dur("duration", duration).
			Str("remote_addr", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent())

		if traceID != "" {
			event.Str("trace_id", traceID)
		}

		// Log user ID if authenticated
		if userID := UserID(c.Request.Context()); userID != "" {
			event.Str("user_id", userID)
		}

		event.Msg("request")
	}
}

// =============================================================================
// PANIC RECOVERY
// =============================================================================

// Recovery is a middleware that recovers from panics and logs them.
//
// When a panic occurs:
//   - Logs the panic with stack trace context
//   - Returns a 500 Internal Server Error
//   - Prevents the server from crashing
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("panic", err).
					Bytes("stack", debug.Stack()).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Str("remote_addr", c.ClientIP()).
					Msg("panic recovered")

				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}
