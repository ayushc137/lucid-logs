// Package response provides standardized HTTP response utilities.
//
// This package ensures consistent API responses across all endpoints:
//   - Success responses with data
//   - Error responses with structured error details
//   - Pagination support
//   - Logging integration
//
// Response Format:
//
//	Success: { "data": <payload> }
//	Error:   { "error": { "code": "...", "message": "...", "details": [...] } }
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// APIResponse is the standard wrapper for all API responses.
//
// All successful responses return:
//
//	{ "data": <payload> }
//
// All error responses return:
//
//	{ "error": { "code": "...", "message": "...", "details": [...] } }
type APIResponse struct {
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
	TraceID string    `json:"trace_id,omitempty"`
}

// APIError represents the error portion of an API response.
type APIError struct {
	Code    errors.ErrorCode `json:"code"`
	Message string           `json:"message"`
	Details any              `json:"details,omitempty"`
}

// ValidationErrorDetail represents a single field validation error.
type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// OperationMessage is a simple success message response.
type OperationMessage struct {
	Message string `json:"message"`
}

// =============================================================================
// SUCCESS RESPONSES
// =============================================================================

// OK sends a 200 OK response with data.
//
// Example:
//
//	response.OK(c, task)
func OK(c *gin.Context, data any) {
	resp := APIResponse{Data: data}
	attachTraceID(c, &resp)
	c.JSON(http.StatusOK, resp)
}

// Created sends a 201 Created response with data.
//
// Example:
//
//	response.Created(c, newTask)
func Created(c *gin.Context, data any) {
	resp := APIResponse{Data: data}
	attachTraceID(c, &resp)
	c.JSON(http.StatusCreated, resp)
}

// NoContent sends a 204 No Content response.
//
// Example:
//
//	response.NoContent(c)
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Message sends a success response with a simple message.
//
// Example:
//
//	response.Message(c, http.StatusOK, "Task deleted")
func Message(c *gin.Context, status int, message string) {
	resp := APIResponse{Data: OperationMessage{Message: message}}
	attachTraceID(c, &resp)
	c.JSON(status, resp)
}

// =============================================================================
// ERROR RESPONSES
// =============================================================================

// Error sends an error response based on an AppError.
//
// This is the primary way to send error responses. It:
//   - Extracts status code, error code, and message from AppError
//   - Logs server errors (5xx) automatically
//   - Hides internal error details from clients
//
// Example:
//
//	if err != nil {
//	    response.Error(c, errors.ErrNotFound)
//	    return
//	}
func Error(c *gin.Context, appErr *errors.AppError) {
	// Log server errors
	if appErr.Status >= 500 {
		if appErr.Err != nil {
			log.Error().
				Err(appErr.Err).
				Str("code", string(appErr.Code)).
				Msg(appErr.Message)
		} else {
			log.Error().
				Str("code", string(appErr.Code)).
				Msg(appErr.Message)
		}
	}

	resp := APIResponse{
		Error: &APIError{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
	}
	attachTraceID(c, &resp)
	c.JSON(appErr.Status, resp)
}

// ErrorFromErr converts a generic error to an AppError and sends it.
//
// If the error is already an AppError, it's used directly.
// Otherwise, it's wrapped as an internal error.
//
// Example:
//
//	if err := service.DoSomething(); err != nil {
//	    response.ErrorFromErr(c, err)
//	    return
//	}
func ErrorFromErr(c *gin.Context, err error) {
	Error(c, errors.ToAppError(err))
}

// =============================================================================
// CONVENIENCE ERROR RESPONSES
// =============================================================================

// BadRequest sends a 400 Bad Request response.
//
// Example:
//
//	response.BadRequest(c, "Invalid JSON body")
func BadRequest(c *gin.Context, message string) {
	Error(c, errors.ErrBadRequest.WithMessage(message))
}

// ValidationFailed sends a 400 response with validation error details.
//
// Example:
//
//	response.ValidationFailed(c, []ValidationErrorDetail{
//	    {Field: "title", Message: "title is required"},
//	})
func ValidationFailed(c *gin.Context, details []ValidationErrorDetail) {
	Error(c, errors.ErrValidationFailed.WithDetails(details))
}

// Unauthorized sends a 401 Unauthorized response.
//
// Example:
//
//	response.Unauthorized(c, "Invalid token")
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	Error(c, errors.ErrUnauthorized.WithMessage(message))
}

// Forbidden sends a 403 Forbidden response.
//
// Example:
//
//	response.Forbidden(c, "Cannot modify this resource")
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Access denied"
	}
	Error(c, errors.ErrForbidden.WithMessage(message))
}

// NotFound sends a 404 Not Found response.
//
// Example:
//
//	response.NotFound(c)
func NotFound(c *gin.Context) {
	Error(c, errors.ErrNotFound)
}

// Conflict sends a 409 Conflict response.
//
// Example:
//
//	response.Conflict(c, "Category name already exists")
func Conflict(c *gin.Context, message string) {
	Error(c, errors.ErrConflict.WithMessage(message))
}

// InternalError sends a 500 Internal Server Error response.
//
// The actual error is logged but not exposed to the client.
//
// Example:
//
//	response.InternalError(c, err)
func InternalError(c *gin.Context, err error) {
	Error(c, errors.ErrInternal.Wrap(err))
}

func attachTraceID(c *gin.Context, resp *APIResponse) {
	if c == nil || resp == nil {
		return
	}
	traceID := c.Writer.Header().Get("X-Request-ID")
	if traceID == "" {
		traceID = c.GetHeader("X-Request-ID")
	}
	if traceID != "" {
		resp.TraceID = traceID
	}
}
