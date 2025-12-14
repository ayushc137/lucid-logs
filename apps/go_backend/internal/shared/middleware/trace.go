package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const traceHeader = "X-Request-ID"

// Trace injects a trace ID into the request/response lifecycle.
//
// Flow:
//  1. Reuse incoming X-Request-ID header if provided (allows external correlation)
//  2. Otherwise generate a UUIDv4
//  3. Store the value in both the gin.Context and the std context
//  4. Echo the trace ID back in the response header so clients can correlate
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := strings.TrimSpace(c.GetHeader(traceHeader))
		if traceID == "" {
			traceID = uuid.NewString()
		}

		// Expose to downstream handlers and responses
		c.Set(string(TraceContextKey), traceID)
		c.Writer.Header().Set(traceHeader, traceID)

		ctx := context.WithValue(c.Request.Context(), TraceContextKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// TraceID extracts the trace ID from context.
func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(TraceContextKey).(string); ok {
		return val
	}
	return ""
}



