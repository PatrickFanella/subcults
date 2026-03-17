// Integration tests demonstrating Request ID middleware usage
package middleware_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onnwee/subcults/internal/middleware"
)

// TestRequestID_BasicUsage demonstrates basic usage of the RequestID middleware
func TestRequestID_BasicUsage(t *testing.T) {
	// Create a simple handler that echoes the request ID
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("expected request ID in context")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request ID: " + requestID))
	})

	// Wrap with RequestID middleware
	wrappedHandler := middleware.RequestID(handler)

	// Test without providing a request ID
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr1, req1)

	// Verify response has X-Request-ID header
	if rr1.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}

	// Test with a provided request ID
	customID := "my-custom-id-123"
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Request-ID", customID)
	rr2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr2, req2)

	// Verify the custom ID was preserved
	if rr2.Header().Get("X-Request-ID") != customID {
		t.Errorf("expected X-Request-ID %q, got %q", customID, rr2.Header().Get("X-Request-ID"))
	}
}

// TestIntegration_RequestIDWithLogging demonstrates the Request ID middleware
// integrated with logging middleware
func TestIntegration_RequestIDWithLogging(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("expected request ID in context, got empty string")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware in correct order: RequestID first, then Logging
	wrappedHandler := middleware.RequestID(
		middleware.Logging(logger)(handler),
	)

	// Make a request without X-Request-ID header
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Verify response has X-Request-ID header
	responseID := rr.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Error("expected X-Request-ID header in response")
	}

	// Verify log contains request_id field
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "request_id=") {
		t.Errorf("expected log to contain request_id field, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, responseID) {
		t.Errorf("expected log to contain request ID %s, got: %s", responseID, logOutput)
	}
}

// TestIntegration_RequestIDPreservation demonstrates that valid request IDs
// are preserved through the middleware chain
func TestIntegration_RequestIDPreservation(t *testing.T) {
	customID := "test-request-12345"
	var capturedID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", customID)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Verify the custom ID was preserved
	if capturedID != customID {
		t.Errorf("expected request ID %q, got %q", customID, capturedID)
	}

	// Verify response header matches
	responseID := rr.Header().Get("X-Request-ID")
	if responseID != customID {
		t.Errorf("expected response header %q, got %q", customID, responseID)
	}
}

// TestIntegration_RequestIDSecurity demonstrates that invalid request IDs
// are replaced with generated UUIDs for security
func TestIntegration_RequestIDSecurity(t *testing.T) {
	tests := []struct {
		name       string
		incomingID string
		wantDiff   bool // whether we expect a different ID in response
	}{
		{
			name:       "malicious injection attempt",
			incomingID: "test\nmalicious-log-entry",
			wantDiff:   true,
		},
		{
			name:       "special characters",
			incomingID: "test@#$%^&*()",
			wantDiff:   true,
		},
		{
			name:       "too long",
			incomingID: strings.Repeat("a", 200),
			wantDiff:   true,
		},
		{
			name:       "valid UUID",
			incomingID: "550e8400-e29b-41d4-a716-446655440000",
			wantDiff:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := middleware.RequestID(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Request-ID", tt.incomingID)
			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			responseID := rr.Header().Get("X-Request-ID")
			if responseID == "" {
				t.Error("expected X-Request-ID in response")
			}

			if tt.wantDiff {
				if responseID == tt.incomingID {
					t.Errorf("expected invalid ID %q to be replaced, but got same ID", tt.incomingID)
				}
			} else {
				if responseID != tt.incomingID {
					t.Errorf("expected valid ID %q to be preserved, got %q", tt.incomingID, responseID)
				}
			}
		})
	}
}

// TestIntegration_CompleteMiddlewareStack demonstrates a realistic middleware
// stack with RequestID, Logging, and other middleware
func TestIntegration_CompleteMiddlewareStack(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a handler that returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is available
		requestID := middleware.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("request ID not available in handler")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	// Build middleware stack: RequestID -> Logging -> Handler
	stack := middleware.RequestID(
		middleware.Logging(logger)(handler),
	)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
	rr := httptest.NewRecorder()
	stack.ServeHTTP(rr, req)

	// Verify response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	// Verify X-Request-ID header
	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header")
	}

	// Verify log output
	logOutput := logBuf.String()
	expectedFields := []string{
		"method=GET",
		"path=/api/users/123",
		"status=200",
		"request_id=",
	}

	for _, field := range expectedFields {
		if !strings.Contains(logOutput, field) {
			t.Errorf("expected log to contain %q, got: %s", field, logOutput)
		}
	}
}

// BenchmarkRequestID_NewID benchmarks request ID generation
func BenchmarkRequestID_NewID(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrappedHandler := middleware.RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}

// BenchmarkRequestID_ExistingID benchmarks request ID validation
func BenchmarkRequestID_ExistingID(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrappedHandler := middleware.RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "550e8400-e29b-41d4-a716-446655440000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}

// TestHTTPIntegration demonstrates actual HTTP server usage
// This test is skipped by default but can be run manually to see the middleware in action
func TestHTTPIntegration(t *testing.T) {
	t.Skip("Manual test - run with 'go test -v -run TestHTTPIntegration' to see output")

	// Create logger
	logger := middleware.NewLogger("development")

	// Create handler
	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r.Context())
		logger.Info("handling request", "request_id", requestID, "path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Request ID: "+requestID+"\n")
	})

	// Apply middleware
	handler := middleware.RequestID(
		middleware.Logging(logger)(mux),
	)

	// Create test server
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Make requests
	resp1, err := http.Get(ts.URL + "/api/test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp1.Body.Close()

	t.Logf("Response 1 - X-Request-ID: %s", resp1.Header.Get("X-Request-ID"))

	// Make request with custom ID
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id-123")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	t.Logf("Response 2 - X-Request-ID: %s", resp2.Header.Get("X-Request-ID"))
}
