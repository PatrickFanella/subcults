// Package middleware provides HTTP middleware components for the API server.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// requestIDKey is the context key for request ID.
type requestIDKey struct{}

// RequestIDHeader is the HTTP header name for request ID.
const RequestIDHeader = "X-Request-ID"

// maxRequestIDLength is the maximum allowed length for a request ID.
const maxRequestIDLength = 128

// isValidRequestID checks if a request ID is valid.
// Valid request IDs are non-empty, at most 128 characters, and contain only
// alphanumeric characters, hyphens, and underscores.
func isValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// RequestID is a middleware that injects a request ID into the context.
// If the request already has a valid X-Request-ID header, it uses that value.
// Otherwise, it generates a new UUID.
// Request IDs from headers are validated to prevent injection attacks.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if !isValidRequestID(requestID) {
			requestID = uuid.New().String()
		}

		// Set the header in the response
		w.Header().Set(RequestIDHeader, requestID)

		// Add request ID to context
		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID returns the request ID from context. Returns empty string if not present.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
