package timeutil

import (
	"time"
)

// ParseDateTime parses a datetime string in multiple formats accepted by the API.
//
// Supported formats:
//   - RFC3339 (2006-01-02T15:04:05Z07:00)
//   - RFC3339Nano (2006-01-02T15:04:05.999999999Z07:00)
//   - Date only (2006-01-02) -> assumes midnight UTC
//   - Datetime without timezone (2006-01-02T15:04:05) -> assumes UTC
func ParseDateTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t, nil
	}

	return time.Time{}, &time.ParseError{
		Value:   s,
		Message: "invalid datetime format, expected ISO8601 (2025-11-24T09:00:00Z) or date (2025-11-24)",
	}
}
