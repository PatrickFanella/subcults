package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onnwee/subcults/internal/middleware"
)

// TestFullIntegration_404Handler demonstrates a complete end-to-end usage
// of the error handling system with middleware integration.
func TestFullIntegration_404Handler(t *testing.T) {
	// Create a simple handler that returns 404 for non-root paths
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "The requested resource was not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"success"}`))
	})

	tests := []struct {
		name          string
		path          string
		wantStatus    int
		wantErrorCode string
		wantMessage   string
		wantIsError   bool
	}{
		{
			name:        "root_path_succeeds",
			path:        "/",
			wantStatus:  http.StatusOK,
			wantIsError: false,
		},
		{
			name:          "non_existent_path_returns_404",
			path:          "/api/v1/scenes",
			wantStatus:    http.StatusNotFound,
			wantErrorCode: ErrCodeNotFound,
			wantMessage:   "The requested resource was not found",
			wantIsError:   true,
		},
		{
			name:          "nested_path_returns_404",
			path:          "/api/v1/scenes/123",
			wantStatus:    http.StatusNotFound,
			wantErrorCode: ErrCodeNotFound,
			wantMessage:   "The requested resource was not found",
			wantIsError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantIsError {
				// Verify it returns JSON error structure
				var resp ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse error response: %v, body: %s", err, w.Body.String())
				}

				if resp.Error.Code != tt.wantErrorCode {
					t.Errorf("expected error code %s, got %s", tt.wantErrorCode, resp.Error.Code)
				}

				if resp.Error.Message != tt.wantMessage {
					t.Errorf("expected message %s, got %s", tt.wantMessage, resp.Error.Message)
				}

				// Verify Content-Type is JSON
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json; charset=utf-8" {
					t.Errorf("expected Content-Type application/json; charset=utf-8, got %s", contentType)
				}
			}
		})
	}
}

// TestFullIntegration_WithMiddleware demonstrates complete middleware integration
// including RequestID and Logging middleware.
func TestFullIntegration_WithMiddleware(t *testing.T) {
	// Create a handler that returns various errors
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/validation":
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid request parameters")
		case "/auth":
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
			WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		case "/forbidden":
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
			WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "Access denied")
		case "/internal":
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
			WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Internal server error")
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}
	})

	// Wrap with middleware
	withMiddleware := middleware.RequestID(handler)

	tests := []struct {
		path          string
		wantStatus    int
		wantErrorCode string
	}{
		{"/validation", http.StatusBadRequest, ErrCodeValidation},
		{"/auth", http.StatusUnauthorized, ErrCodeAuthFailed},
		{"/forbidden", http.StatusForbidden, ErrCodeForbidden},
		{"/internal", http.StatusInternalServerError, ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			withMiddleware.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}

			if resp.Error.Code != tt.wantErrorCode {
				t.Errorf("expected error code %s, got %s", tt.wantErrorCode, resp.Error.Code)
			}

			// Verify request ID was added by middleware
			requestID := w.Header().Get("X-Request-ID")
			if requestID == "" {
				t.Error("expected X-Request-ID header to be set by middleware")
			}
		})
	}
}
