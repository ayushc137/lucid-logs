package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
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
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code and bytes
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Process request
		next.ServeHTTP(ww, r)

		// Calculate duration
		duration := time.Since(start)

		// Get request ID if available
		reqID := middleware.GetReqID(r.Context())

		// Determine log level based on status
		status := ww.Status()
		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		// Build log entry
		event.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", status).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", duration).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent())

		if reqID != "" {
			event.Str("request_id", reqID)
		}

		// Log user ID if authenticated
		if userID := UserID(r.Context()); userID != "" {
			event.Str("user_id", userID)
		}

		event.Msg("request")
	})
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
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("panic", err).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Msg("panic recovered")

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
