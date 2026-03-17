package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	var contextID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		contextID = GetRequestID(r.Context())
		if contextID == "" {
			t.Error("expected request ID in context, got empty string")
		}
		// Verify it matches response header
		if responseID := w.Header().Get(RequestIDHeader); responseID != contextID {
			t.Errorf("context ID %q doesn't match response header ID %q", contextID, responseID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify X-Request-ID header is set in response
	responseID := rr.Header().Get(RequestIDHeader)
	if responseID == "" {
		t.Error("expected X-Request-ID header in response, got empty string")
	}

	// Verify context ID matches response header
	if responseID != contextID {
		t.Errorf("context ID %q doesn't match response header ID %q", contextID, responseID)
	}
}

func TestRequestID_UsesExistingHeader(t *testing.T) {
	existingID := "existing-request-id-123"
	var capturedID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, existingID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify existing ID is preserved
	if capturedID != existingID {
		t.Errorf("expected request ID %q, got %q", existingID, capturedID)
	}

	// Verify response header has the same ID
	responseID := rr.Header().Get(RequestIDHeader)
	if responseID != existingID {
		t.Errorf("expected response header %q, got %q", existingID, responseID)
	}
}

func TestRequestID_RejectsInvalidHeader(t *testing.T) {
	tests := []struct {
		name      string
		headerID  string
		shouldUse bool // whether the header ID should be used or a new one generated
	}{
		{"empty string", "", false},
		{"too long", strings.Repeat("a", 129), false},
		{"with spaces", "invalid id", false},
		{"with special chars", "id@#$%", false},
		{"valid alphanumeric", "abc123", true},
		{"valid with hyphens", "abc-123-def", true},
		{"valid with underscores", "abc_123_def", true},
		{"valid UUID format", "550e8400-e29b-41d4-a716-446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedID string
			handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedID = GetRequestID(r.Context())
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.headerID != "" {
				req.Header.Set(RequestIDHeader, tt.headerID)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tt.shouldUse {
				if capturedID != tt.headerID {
					t.Errorf("expected request ID %q, got %q", tt.headerID, capturedID)
				}
			} else {
				if capturedID == tt.headerID {
					t.Errorf("expected new request ID to be generated, but got header ID %q", capturedID)
				}
				if capturedID == "" {
					t.Error("expected request ID to be generated, got empty string")
				}
			}
		})
	}
}

func TestGetRequestID_EmptyContextReturnsEmptyString(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	requestID := GetRequestID(req.Context())
	if requestID != "" {
		t.Errorf("expected empty string, got %q", requestID)
	}
}
