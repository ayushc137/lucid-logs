// Package errors provides standardized error handling for the application.
//
// This package defines:
//   - AppError: A structured error type with HTTP status codes and error codes
//   - Predefined errors for common scenarios (NotFound, Unauthorized, etc.)
//   - Error wrapping utilities for consistent error handling
//
// Usage:
//
//	if user == nil {
//	    return errors.ErrNotFound.WithMessage("user not found")
//	}
//
//	if err := db.Query(...); err != nil {
//	    return errors.ErrDatabase.Wrap(err)
//	}
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// =============================================================================
// ERROR CODES
// =============================================================================

// ErrorCode represents a machine-readable error code for API consumers.
// These codes are stable and can be used for programmatic error handling.
type ErrorCode string

const (
	// Client errors (4xx)
	CodeBadRequest       ErrorCode = "BAD_REQUEST"
	CodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodeForbidden        ErrorCode = "FORBIDDEN"
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeTooManyRequests  ErrorCode = "TOO_MANY_REQUESTS"

	// Server errors (5xx)
	CodeInternalError ErrorCode = "INTERNAL_ERROR"
	CodeDatabaseError ErrorCode = "DATABASE_ERROR"
	CodeServiceError  ErrorCode = "SERVICE_UNAVAILABLE"
)

// =============================================================================
// APP ERROR TYPE
// =============================================================================

// AppError represents a structured application error with HTTP semantics.
//
// Fields:
//   - Code: Machine-readable error code (e.g., "NOT_FOUND")
//   - Message: Human-readable error message
//   - Status: HTTP status code
//   - Details: Optional additional error details (validation errors, etc.)
//   - Err: The underlying error (if any)
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Status  int       `json:"-"`
	Details any       `json:"details,omitempty"`
	Err     error     `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithMessage returns a copy of the error with a custom message.
//
// Example:
//
//	return ErrNotFound.WithMessage("task not found")
func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: msg,
		Status:  e.Status,
		Details: e.Details,
		Err:     e.Err,
	}
}

// WithDetails returns a copy of the error with additional details.
//
// Example:
//
//	return ErrValidationFailed.WithDetails(validationErrors)
func (e *AppError) WithDetails(details any) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Status:  e.Status,
		Details: details,
		Err:     e.Err,
	}
}

// Wrap wraps an underlying error with this AppError.
//
// Example:
//
//	if err := db.Query(...); err != nil {
//	    return ErrDatabase.Wrap(err)
//	}
func (e *AppError) Wrap(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Status:  e.Status,
		Details: e.Details,
		Err:     err,
	}
}

// =============================================================================
// PREDEFINED ERRORS
// =============================================================================

// Client errors (4xx)
var (
	// ErrBadRequest indicates a malformed or invalid request.
	ErrBadRequest = &AppError{
		Code:    CodeBadRequest,
		Message: "Bad request",
		Status:  http.StatusBadRequest,
	}

	// ErrValidationFailed indicates request validation failure.
	// Use WithDetails() to include validation error details.
	ErrValidationFailed = &AppError{
		Code:    CodeValidationFailed,
		Message: "Request validation failed",
		Status:  http.StatusBadRequest,
	}

	// ErrUnauthorized indicates missing or invalid authentication.
	ErrUnauthorized = &AppError{
		Code:    CodeUnauthorized,
		Message: "Authentication required",
		Status:  http.StatusUnauthorized,
	}

	// ErrForbidden indicates the user lacks permission for the resource.
	ErrForbidden = &AppError{
		Code:    CodeForbidden,
		Message: "Access denied",
		Status:  http.StatusForbidden,
	}

	// ErrNotFound indicates the requested resource doesn't exist.
	ErrNotFound = &AppError{
		Code:    CodeNotFound,
		Message: "Resource not found",
		Status:  http.StatusNotFound,
	}

	// ErrConflict indicates a conflict with existing data.
	ErrConflict = &AppError{
		Code:    CodeConflict,
		Message: "Resource already exists",
		Status:  http.StatusConflict,
	}

	// ErrTooManyRequests indicates rate limiting.
	ErrTooManyRequests = &AppError{
		Code:    CodeTooManyRequests,
		Message: "Too many requests",
		Status:  http.StatusTooManyRequests,
	}
)

// Server errors (5xx)
var (
	// ErrInternal indicates an unexpected server error.
	// The actual error is logged but not exposed to clients.
	ErrInternal = &AppError{
		Code:    CodeInternalError,
		Message: "An internal error occurred",
		Status:  http.StatusInternalServerError,
	}

	// ErrDatabase indicates a database operation failure.
	// Use Wrap() to include the underlying error for logging.
	ErrDatabase = &AppError{
		Code:    CodeDatabaseError,
		Message: "Database operation failed",
		Status:  http.StatusInternalServerError,
	}

	// ErrServiceUnavailable indicates the service is temporarily unavailable.
	ErrServiceUnavailable = &AppError{
		Code:    CodeServiceError,
		Message: "Service temporarily unavailable",
		Status:  http.StatusServiceUnavailable,
	}
)

// =============================================================================
// DOMAIN-SPECIFIC ERRORS
// =============================================================================

// Domain-specific errors that can be used across features.
var (
	// ErrInvalidCredentials for auth failures.
	ErrInvalidCredentials = &AppError{
		Code:    CodeUnauthorized,
		Message: "Invalid credentials",
		Status:  http.StatusUnauthorized,
	}

	// ErrUserExists for duplicate user registration.
	ErrUserExists = &AppError{
		Code:    CodeConflict,
		Message: "User with this email already exists",
		Status:  http.StatusConflict,
	}

	// ErrInvalidDateRange for date validation.
	ErrInvalidDateRange = &AppError{
		Code:    CodeBadRequest,
		Message: "end_date must be on or after start_date",
		Status:  http.StatusBadRequest,
	}

	// ErrCategoryNotFound for category lookups.
	ErrCategoryNotFound = &AppError{
		Code:    CodeBadRequest,
		Message: "Category not found or has been deleted",
		Status:  http.StatusBadRequest,
	}

	// ErrCategoryNameExists for duplicate category names.
	ErrCategoryNameExists = &AppError{
		Code:    CodeConflict,
		Message: "Category with this name already exists",
		Status:  http.StatusConflict,
	}
)

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// Is checks if the target error is an AppError with the given code.
//
// Example:
//
//	if errors.Is(err, ErrNotFound) {
//	    // handle not found
//	}
func Is(err error, target *AppError) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == target.Code
	}
	return false
}

// AsAppError attempts to convert an error to an AppError.
// Returns nil if the error is not an AppError.
func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// ToAppError converts any error to an AppError.
// If the error is already an AppError, it returns it unchanged.
// Otherwise, it wraps the error as an internal error.
func ToAppError(err error) *AppError {
	if appErr := AsAppError(err); appErr != nil {
		return appErr
	}
	return ErrInternal.Wrap(err)
}
