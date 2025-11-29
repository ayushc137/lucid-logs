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
	"encoding/json"
	"net/http"

	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/rs/zerolog/log"
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
	Data  any       `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
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
// RESPONSE WRITERS
// =============================================================================

// JSON writes a JSON response with the given status code.
//
// This is the low-level function used by other response helpers.
// Prefer using the higher-level helpers (OK, Created, Error, etc.)
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Error().Err(err).Msg("failed to encode JSON response")
		}
	}
}

// =============================================================================
// SUCCESS RESPONSES
// =============================================================================

// OK sends a 200 OK response with data.
//
// Example:
//
//	response.OK(w, task)
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, APIResponse{Data: data})
}

// Created sends a 201 Created response with data.
//
// Example:
//
//	response.Created(w, newTask)
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, APIResponse{Data: data})
}

// NoContent sends a 204 No Content response.
//
// Example:
//
//	response.NoContent(w)
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Message sends a success response with a simple message.
//
// Example:
//
//	response.Message(w, http.StatusOK, "Task deleted")
func Message(w http.ResponseWriter, status int, message string) {
	JSON(w, status, APIResponse{Data: OperationMessage{Message: message}})
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
//	    response.Error(w, errors.ErrNotFound)
//	    return
//	}
func Error(w http.ResponseWriter, appErr *errors.AppError) {
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

	JSON(w, appErr.Status, APIResponse{
		Error: &APIError{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}

// ErrorFromErr converts a generic error to an AppError and sends it.
//
// If the error is already an AppError, it's used directly.
// Otherwise, it's wrapped as an internal error.
//
// Example:
//
//	if err := service.DoSomething(); err != nil {
//	    response.ErrorFromErr(w, err)
//	    return
//	}
func ErrorFromErr(w http.ResponseWriter, err error) {
	Error(w, errors.ToAppError(err))
}

// =============================================================================
// CONVENIENCE ERROR RESPONSES
// =============================================================================

// BadRequest sends a 400 Bad Request response.
//
// Example:
//
//	response.BadRequest(w, "Invalid JSON body")
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, errors.ErrBadRequest.WithMessage(message))
}

// ValidationFailed sends a 400 response with validation error details.
//
// Example:
//
//	response.ValidationFailed(w, []ValidationErrorDetail{
//	    {Field: "title", Message: "title is required"},
//	})
func ValidationFailed(w http.ResponseWriter, details []ValidationErrorDetail) {
	Error(w, errors.ErrValidationFailed.WithDetails(details))
}

// Unauthorized sends a 401 Unauthorized response.
//
// Example:
//
//	response.Unauthorized(w, "Invalid token")
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required"
	}
	Error(w, errors.ErrUnauthorized.WithMessage(message))
}

// Forbidden sends a 403 Forbidden response.
//
// Example:
//
//	response.Forbidden(w, "Cannot modify this resource")
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Access denied"
	}
	Error(w, errors.ErrForbidden.WithMessage(message))
}

// NotFound sends a 404 Not Found response.
//
// Example:
//
//	response.NotFound(w)
func NotFound(w http.ResponseWriter) {
	Error(w, errors.ErrNotFound)
}

// Conflict sends a 409 Conflict response.
//
// Example:
//
//	response.Conflict(w, "Category name already exists")
func Conflict(w http.ResponseWriter, message string) {
	Error(w, errors.ErrConflict.WithMessage(message))
}

// InternalError sends a 500 Internal Server Error response.
//
// The actual error is logged but not exposed to the client.
//
// Example:
//
//	response.InternalError(w, err)
func InternalError(w http.ResponseWriter, err error) {
	Error(w, errors.ErrInternal.Wrap(err))
}
