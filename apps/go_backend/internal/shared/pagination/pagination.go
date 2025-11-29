// Package pagination provides standardized pagination utilities.
//
// This package defines:
//   - Params: Pagination query parameters with validation
//   - Response: Generic paginated response wrapper
//   - Parsing utilities for HTTP requests
//
// Usage:
//
//	func ListTasks(w http.ResponseWriter, r *http.Request) {
//	    params := pagination.FromRequest(r)
//	    tasks, total, err := repo.FindPaginated(ctx, params.Limit, params.Offset)
//	    response.OK(w, pagination.NewResponse(tasks, total, params))
//	}
package pagination

import (
	"net/http"
	"strconv"
)

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// DefaultLimit is the default number of items per page.
	DefaultLimit = 20

	// MaxLimit is the maximum allowed items per page.
	MaxLimit = 100

	// DefaultOffset is the default starting position.
	DefaultOffset = 0
)

// =============================================================================
// PAGINATION PARAMS
// =============================================================================

// Params contains validated pagination parameters.
//
// Fields:
//   - Limit: Number of items to return (1-100, default 20)
//   - Offset: Number of items to skip (>= 0, default 0)
type Params struct {
	Limit  int `json:"limit" validate:"min=1,max=100"`
	Offset int `json:"offset" validate:"min=0"`
}

// Default returns the default pagination parameters.
func Default() Params {
	return Params{
		Limit:  DefaultLimit,
		Offset: DefaultOffset,
	}
}

// FromRequest extracts and validates pagination params from HTTP request.
//
// Query parameters:
//   - limit: Number of items (default 20, max 100)
//   - offset: Items to skip (default 0)
//
// Example URL: /api/v1/tasks?limit=50&offset=100
func FromRequest(r *http.Request) Params {
	params := Default()

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	// Clamp limit to max
	if params.Limit > MaxLimit {
		params.Limit = MaxLimit
	}

	return params
}

// =============================================================================
// PAGINATED RESPONSE
// =============================================================================

// Response is a generic paginated response wrapper.
//
// Fields:
//   - Items: The page of data items
//   - Total: Total count of all matching items
//   - Limit: Number of items requested
//   - Offset: Starting position
//   - HasMore: Whether more items exist after this page
//
// Example response:
//
//	{
//	    "items": [...],
//	    "total": 150,
//	    "limit": 20,
//	    "offset": 0,
//	    "has_more": true
//	}
type Response[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	HasMore bool  `json:"has_more"`
}

// NewResponse creates a new paginated response.
//
// Example:
//
//	tasks, total, _ := repo.FindPaginated(ctx, params.Limit, params.Offset)
//	resp := pagination.NewResponse(tasks, total, params)
func NewResponse[T any](items []T, total int64, params Params) Response[T] {
	// Ensure items is never nil (return empty array instead)
	if items == nil {
		items = []T{}
	}

	hasMore := int64(params.Offset+len(items)) < total

	return Response[T]{
		Items:   items,
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: hasMore,
	}
}

// =============================================================================
// SUREALDB QUERY HELPERS
// =============================================================================

// ToSurrealQL returns the LIMIT and START clauses for SurrealDB queries.
//
// Example:
//
//	query := "SELECT * FROM tasks " + params.ToSurrealQL()
//	// Result: "SELECT * FROM tasks LIMIT 20 START 0"
func (p Params) ToSurrealQL() string {
	return "LIMIT " + strconv.Itoa(p.Limit) + " START " + strconv.Itoa(p.Offset)
}

// ToVars returns pagination parameters as a map for SurrealDB query binding.
//
// Example:
//
//	vars := params.ToVars()
//	db.Query("SELECT * FROM tasks LIMIT $limit START $offset", vars)
func (p Params) ToVars() map[string]any {
	return map[string]any{
		"limit":  p.Limit,
		"offset": p.Offset,
	}
}
