package response

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

// Response represents a standard API response envelope
type Response struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error represents a standard error response
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ValidationError represents a single field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Success sends a successful response with data
func Success(c *gin.Context, status int, data any) {
	c.JSON(status, Response{Data: data})
}

// Fail sends an error response
func Fail(c *gin.Context, status int, code string, message string, details any) {
	logger := log.Ctx(c.Request.Context())

	// Log internal server errors
	if status >= 500 {
		logger.Error().
			Str("code", code).
			Str("message", message).
			Interface("details", details).
			Msg("internal error")
	} else if status >= 400 {
		logger.Warn().
			Str("code", code).
			Str("message", message).
			Msg("client error")
	}

	c.JSON(status, Response{
		Error: &Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// ValidationFailed sends a validation error response
func ValidationFailed(c *gin.Context, err error) {
	if ve, ok := err.(validator.ValidationErrors); ok {
		items := make([]ValidationError, 0, len(ve))
		for _, fe := range ve {
			items = append(items, ValidationError{
				Field:   toJSONField(fe.Field()),
				Message: validationMessage(fe),
			})
		}
		Fail(c, http.StatusBadRequest, "VALIDATION_FAILED", "Request validation failed", items)
		return
	}

	Fail(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
}

// NotFound sends a not found error
func NotFound(c *gin.Context, resource string) {
	Fail(c, http.StatusNotFound, "NOT_FOUND", resource+" not found", nil)
}

// Unauthorized sends an unauthorized error
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

// Forbidden sends a forbidden error
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Access denied"
	}
	Fail(c, http.StatusForbidden, "FORBIDDEN", message, nil)
}

// InternalError sends an internal server error
func InternalError(c *gin.Context, err error) {
	logger := log.Ctx(c.Request.Context())
	logger.Error().Err(err).Msg("internal server error")

	Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred", nil)
}

// BadRequest sends a bad request error
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "BAD_REQUEST", message, nil)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Helper functions

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "gte":
		return "must be greater than or equal to " + fe.Param()
	case "lte":
		return "must be less than or equal to " + fe.Param()
	case "max":
		return "maximum value is " + fe.Param()
	case "min":
		return "minimum value is " + fe.Param()
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "len":
		return "must be exactly " + fe.Param() + " characters long"
	default:
		return "validation failed: " + fe.Tag()
	}
}

func toJSONField(field string) string {
	if field == "" {
		return field
	}
	// Convert PascalCase to camelCase
	return strings.ToLower(field[:1]) + field[1:]
}
