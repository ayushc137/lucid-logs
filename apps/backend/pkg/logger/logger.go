package logger

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init initializes the global logger with pretty console output for development
func Init(env string) {
	zerolog.TimeFieldFormat = time.RFC3339

	if env == "development" || env == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if env == "development" || env == "dev" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// FromContext retrieves a logger from the context, falling back to the global logger
func FromContext(ctx context.Context) *zerolog.Logger {
	if ctx == nil {
		return &log.Logger
	}

	if logger := zerolog.Ctx(ctx); logger != nil && logger.GetLevel() != zerolog.Disabled {
		return logger
	}

	return &log.Logger
}

// WithContext adds a logger to the context
func WithContext(ctx context.Context, logger *zerolog.Logger) context.Context {
	return logger.WithContext(ctx)
}

// Middleware creates a Gin middleware for request logging
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Create request-scoped logger
		logger := log.With().
			Str("method", c.Request.Method).
			Str("path", path).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Logger()

		// Add logger to context
		c.Request = c.Request.WithContext(logger.WithContext(c.Request.Context()))

		// Process request
		c.Next()

		// Log after request
		latency := time.Since(start)
		status := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		event := logger.Info()
		if status >= 500 {
			event = logger.Error()
		} else if status >= 400 {
			event = logger.Warn()
		}

		event.
			Int("status", status).
			Dur("latency", latency).
			Int("size", c.Writer.Size()).
			Msg("request completed")

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.Error().Err(err.Err).Msg(err.Error())
			}
		}
	}
}
