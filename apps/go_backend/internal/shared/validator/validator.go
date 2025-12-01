// Package validator provides request validation utilities.
//
// This package wraps go-playground/validator with:
//   - Custom validation rules (datetime_flexible, etc.)
//   - JSON field name support in error messages
//   - Human-readable error messages
//
// Usage:
//
//	val := validator.New()
//	if errs := val.Validate(&req); errs != nil {
//	    response.ValidationFailed(w, errs)
//	    return
//	}
package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/timeutil"
)

// =============================================================================
// VALIDATOR TYPE
// =============================================================================

// Validator wraps go-playground/validator with custom validations.
type Validator struct {
	v *validator.Validate
}

// New creates a new Validator instance with custom validations registered.
//
// Custom validations included:
//   - datetime_flexible: Accepts ISO8601 or YYYY-MM-DD formats
func New() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())

	// Use JSON tag names in error messages instead of struct field names
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register custom validations
	_ = v.RegisterValidation("datetime_flexible", validateDateTimeFlexible)

	return &Validator{v: v}
}

// =============================================================================
// VALIDATION METHODS
// =============================================================================

// Validate validates a struct and returns formatted validation errors.
//
// Returns nil if validation passes.
//
// Example:
//
//	type CreateTaskRequest struct {
//	    Title string `json:"title" validate:"required,min=1,max=500"`
//	}
//
//	if errs := val.Validate(&req); errs != nil {
//	    response.ValidationFailed(w, errs)
//	    return
//	}
func (val *Validator) Validate(s any) []response.ValidationErrorDetail {
	err := val.v.Struct(s)
	if err == nil {
		return nil
	}

	var errors []response.ValidationErrorDetail
	for _, e := range err.(validator.ValidationErrors) {
		errors = append(errors, response.ValidationErrorDetail{
			Field:   e.Field(),
			Message: formatValidationMessage(e),
		})
	}
	return errors
}

// ValidateVar validates a single variable against a tag.
//
// Example:
//
//	err := val.ValidateVar(email, "required,email")
func (val *Validator) ValidateVar(field any, tag string) error {
	return val.v.Var(field, tag)
}

// =============================================================================
// ERROR MESSAGE FORMATTING
// =============================================================================

// formatValidationMessage returns a human-readable validation error message.
func formatValidationMessage(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return field + " is required"
	case "min":
		if e.Type().Kind() == reflect.String {
			return field + " must be at least " + e.Param() + " characters"
		}
		return field + " must be at least " + e.Param()
	case "max":
		if e.Type().Kind() == reflect.String {
			return field + " must be at most " + e.Param() + " characters"
		}
		return field + " must be at most " + e.Param()
	case "email":
		return field + " must be a valid email address"
	case "datetime_flexible":
		return field + " must be a valid datetime (ISO8601 or YYYY-MM-DD)"
	case "gtefield":
		return field + " must be greater than or equal to " + e.Param()
	case "oneof":
		return field + " must be one of: " + e.Param()
	case "url":
		return field + " must be a valid URL"
	case "uuid":
		return field + " must be a valid UUID"
	default:
		return field + " failed validation: " + e.Tag()
	}
}

// =============================================================================
// CUSTOM VALIDATORS
// =============================================================================

// validateDateTimeFlexible validates datetime in ISO8601 or YYYY-MM-DD format.
func validateDateTimeFlexible(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Empty is OK, use 'required' tag for mandatory fields
	}
	_, err := timeutil.ParseDateTime(value)
	return err == nil
}
