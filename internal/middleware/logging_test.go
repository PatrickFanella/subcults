package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testLogEntry represents a parsed JSON log entry for testing.
type testLogEntry struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Size      int    `json:"size"`
	RequestID string `json:"request_id"`
	UserDID   string `json:"user_did"`
	ErrorCode string `json:"error_code"`
}

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func TestLogging_BasicFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Parse log entry
	var entry testLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v, log: %s", err, buf.String())
	}

	// Verify required fields
	if entry.Method != "GET" {
		t.Errorf("expected method GET, got %s", entry.Method)
	}
	if entry.Path != "/api/v1/test" {
		t.Errorf("expected path /api/v1/test, got %s", entry.Path)
	}
	if entry.Status != 200 {
		t.Errorf("expected status 200, got %d", entry.Status)
	}
	if entry.LatencyMS < 0 {
		t.Errorf("expected latency_ms >= 0, got %d", entry.LatencyMS)
	}
	if entry.Size != 5 { // "hello" = 5 bytes
		t.Errorf("expected size 5, got %d", entry.Size)
	}
	if entry.Level != "INFO" {
		t.Errorf("expected level INFO, got %s", entry.Level)
	}
}

func TestLogging_WithRequestID(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	// Chain RequestID middleware with Logging middleware
	handler := RequestID(Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.Header.Set(RequestIDHeader, "test-request-id-456")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var entry testLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if entry.RequestID != "test-request-id-456" {
		t.Errorf("expected request_id test-request-id-456, got %s", entry.RequestID)
	}
}

// userDIDMiddleware simulates authentication middleware that sets user DID.
func userDIDMiddleware(did string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := SetUserDID(r.Context(), did)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// errorCodeMiddleware simulates error handling middleware that sets error code.
func errorCodeMiddleware(code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := SetErrorCode(r.Context(), code)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestLogging_WithUserDID(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	// Simulate auth middleware setting user DID before logging middleware
	handler := userDIDMiddleware("did:web:example.com:user123")(
		Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var entry testLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if entry.UserDID != "did:web:example.com:user123" {
		t.Errorf("expected user_did did:web:example.com:user123, got %s", entry.UserDID)
	}
}

func TestLogging_ErrorResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	// Error code is typically set by error handling middleware that runs before logging
	handler := errorCodeMiddleware("VALIDATION_ERROR")(
		Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"validation failed"}`))
		})),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var entry testLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if entry.Status != 400 {
		t.Errorf("expected status 400, got %d", entry.Status)
	}
	if entry.ErrorCode != "VALIDATION_ERROR" {
		t.Errorf("expected error_code VALIDATION_ERROR, got %s", entry.ErrorCode)
	}
	if entry.Level != "WARN" {
		t.Errorf("expected level WARN for 4xx, got %s", entry.Level)
	}
}

func TestLogging_ServerError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	handler := errorCodeMiddleware("INTERNAL_ERROR")(
		Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var entry testLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if entry.Status != 500 {
		t.Errorf("expected status 500, got %d", entry.Status)
	}
	if entry.ErrorCode != "INTERNAL_ERROR" {
		t.Errorf("expected error_code INTERNAL_ERROR, got %s", entry.ErrorCode)
	}
	if entry.Level != "ERROR" {
		t.Errorf("expected level ERROR for 5xx, got %s", entry.Level)
	}
}

func TestLogging_DefaultStatus(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	// Handler that doesn't explicitly set status code
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var entry testLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	// Default status should be 200
	if entry.Status != 200 {
		t.Errorf("expected default status 200, got %d", entry.Status)
	}
}

func TestNewLogger_Production(t *testing.T) {
	buf := &bytes.Buffer{}

	// Create logger with production settings using test helper
	logger := newLoggerWithWriter("production", buf)
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Test that production logger outputs JSON format
	logger.Info("test message", "key", "value", "count", 42)

	// Verify JSON format
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("production logger should output valid JSON: %v, log: %s", err, buf.String())
	}

	// Verify standard fields
	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got %v", logEntry["msg"])
	}
	if logEntry["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", logEntry["level"])
	}
	if logEntry["key"] != "value" {
		t.Errorf("expected key 'value', got %v", logEntry["key"])
	}
	if logEntry["count"] != float64(42) {
		t.Errorf("expected count 42, got %v", logEntry["count"])
	}
	// Verify timestamp exists
	if _, ok := logEntry["time"]; !ok {
		t.Error("expected 'time' field in JSON log")
	}
}

func TestNewLogger_Development(t *testing.T) {
	buf := &bytes.Buffer{}

	// Create logger with development settings using test helper
	logger := newLoggerWithWriter("development", buf)
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Test that development logger outputs text format and logs debug messages
	logger.Debug("debug message", "component", "test")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Errorf("expected log to contain 'debug message', got: %s", output)
	}
	if !strings.Contains(output, "component=test") {
		t.Errorf("expected log to contain 'component=test', got: %s", output)
	}
}

func TestSetUserDID_GetUserDID(t *testing.T) {
	ctx := context.Background()

	// Test with empty context
	if did := GetUserDID(ctx); did != "" {
		t.Errorf("expected empty DID, got %q", did)
	}

	// Test with DID set
	ctx = SetUserDID(ctx, "did:web:test.com")
	if did := GetUserDID(ctx); did != "did:web:test.com" {
		t.Errorf("expected did:web:test.com, got %q", did)
	}
}

func TestSetErrorCode_GetErrorCode(t *testing.T) {
	ctx := context.Background()

	// Test with empty context
	if code := GetErrorCode(ctx); code != "" {
		t.Errorf("expected empty error code, got %q", code)
	}

	// Test with error code set
	ctx = SetErrorCode(ctx, "NOT_FOUND")
	if code := GetErrorCode(ctx); code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %q", code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w, context.Background())

	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected status code 201, got %d", rw.statusCode)
	}
	if w.Code != http.StatusCreated {
		t.Errorf("expected underlying writer status 201, got %d", w.Code)
	}
}

func TestResponseWriter_MultipleWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w, context.Background())

	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusBadRequest) // Second call should not change statusCode

	// Should capture the first status code
	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected status code 201, got %d", rw.statusCode)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w, context.Background())

	data := []byte("test response body")
	n, err := rw.Write(data)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}
	if rw.size != len(data) {
		t.Errorf("expected size %d, got %d", len(data), rw.size)
	}
}

func TestLogging_AllFieldsPresent(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	// Chain: RequestID -> UserDID -> ErrorCode -> Logging -> Handler
	handler := RequestID(
		userDIDMiddleware("did:plc:abcd1234")(
			errorCodeMiddleware("FORBIDDEN")(
				Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"error":"forbidden"}`))
				})),
			),
		),
	)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/scenes/123", nil)
	req.Header.Set(RequestIDHeader, "req-id-789")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var entry testLogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	// Verify all fields
	if entry.Method != "DELETE" {
		t.Errorf("expected method DELETE, got %s", entry.Method)
	}
	if entry.Path != "/api/v1/scenes/123" {
		t.Errorf("expected path /api/v1/scenes/123, got %s", entry.Path)
	}
	if entry.Status != 403 {
		t.Errorf("expected status 403, got %d", entry.Status)
	}
	if entry.RequestID != "req-id-789" {
		t.Errorf("expected request_id req-id-789, got %s", entry.RequestID)
	}
	if entry.UserDID != "did:plc:abcd1234" {
		t.Errorf("expected user_did did:plc:abcd1234, got %s", entry.UserDID)
	}
	if entry.ErrorCode != "FORBIDDEN" {
		t.Errorf("expected error_code FORBIDDEN, got %s", entry.ErrorCode)
	}
	if entry.Size != 21 { // `{"error":"forbidden"}` = 21 bytes
		t.Errorf("expected size 21, got %d", entry.Size)
	}
}

func TestLogging_NoErrorCodeFor2xx(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	// Set error code even for success (shouldn't be logged)
	handler := errorCodeMiddleware("SOME_CODE")(
		Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// The error_code should not be present for 2xx responses
	logStr := buf.String()
	if strings.Contains(logStr, "error_code") {
		t.Error("error_code should not be logged for 2xx responses")
	}
}

// TestLogging_StandardFieldsVerification verifies all standard fields are present
// in structured logs across the application.
func TestLogging_StandardFieldsVerification(t *testing.T) {
	tests := []struct {
		name            string
		setupHandler    func(w http.ResponseWriter, r *http.Request)
		setupMiddleware func(http.Handler) http.Handler
		expectedFields  map[string]bool // fields that should be present
		statusCode      int
		requestID       string
	}{
		{
			name: "successful request with all fields",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			},
			setupMiddleware: func(next http.Handler) http.Handler {
				return RequestID(userDIDMiddleware("did:plc:test123")(next))
			},
			expectedFields: map[string]bool{
				"time":       true,
				"level":      true,
				"msg":        true,
				"method":     true,
				"path":       true,
				"status":     true,
				"latency_ms": true,
				"size":       true,
				"request_id": true,
				"user_did":   true,
			},
			statusCode: http.StatusOK,
			requestID:  "req-abc-123",
		},
		{
			name: "error request with error code",
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"bad request"}`))
			},
			setupMiddleware: func(next http.Handler) http.Handler {
				return RequestID(errorCodeMiddleware("VALIDATION_ERROR")(next))
			},
			expectedFields: map[string]bool{
				"time":       true,
				"level":      true,
				"msg":        true,
				"method":     true,
				"path":       true,
				"status":     true,
				"latency_ms": true,
				"size":       true,
				"request_id": true,
				"error_code": true,
			},
			statusCode: http.StatusBadRequest,
			requestID:  "req-err-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := newTestLogger(buf)

			handler := tt.setupMiddleware(Logging(logger)(http.HandlerFunc(tt.setupHandler)))

			req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
			if tt.requestID != "" {
				req.Header.Set(RequestIDHeader, tt.requestID)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Parse log entry
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("failed to parse log entry: %v, log: %s", err, buf.String())
			}

			// Verify expected fields are present
			for field, shouldExist := range tt.expectedFields {
				_, exists := logEntry[field]
				if shouldExist && !exists {
					t.Errorf("expected field %q to be present in log entry", field)
				}
			}

			// Verify status code
			if status, ok := logEntry["status"].(float64); ok {
				if int(status) != tt.statusCode {
					t.Errorf("expected status %d, got %d", tt.statusCode, int(status))
				}
			} else {
				t.Errorf("status field missing or wrong type")
			}
		})
	}
}
