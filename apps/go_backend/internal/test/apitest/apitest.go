package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lucid-logs/go-backend/internal/shared/middleware"
)

// RequestOption mutates an http.Request before it is sent.
type RequestOption func(*http.Request)

// JSONRequest builds an HTTP request with a JSON body and Content-Type header.
func JSONRequest(t *testing.T, method, path string, body any, opts ...RequestOption) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for _, opt := range opts {
		opt(req)
	}
	return req
}

// WithHeader sets a header on the request.
func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithUser injects an authenticated user into the request context.
func WithUser(userID string) RequestOption {
	return func(req *http.Request) {
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, &middleware.AuthenticatedUser{
			UserID: userID,
		})
		*req = *req.Clone(ctx)
	}
}

// Do executes the request against the handler and returns the recorder.
func Do(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// Decode decodes the JSON response body into v.
func Decode(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
