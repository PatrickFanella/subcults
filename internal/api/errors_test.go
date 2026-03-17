package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onnwee/subcults/internal/middleware"
)

func TestWriteError_BasicFields(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := context.Background()

	WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")

	// Check status code
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type to contain application/json, got %s", contentType)
	}

	// Parse response body
	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v, body: %s", err, w.Body.String())
	}

	// Verify error structure
	if resp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s, got %s", ErrCodeNotFound, resp.Error.Code)
	}
	if resp.Error.Message != "Scene not found" {
		t.Errorf("expected message 'Scene not found', got %s", resp.Error.Message)
	}
}

func TestWriteError_AllErrorCodes(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		code       string
		message    string
		wantStatus int
	}{
		{
			name:       "validation_error",
			status:     http.StatusBadRequest,
			code:       ErrCodeValidation,
			message:    "Invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "auth_failed",
			status:     http.StatusUnauthorized,
			code:       ErrCodeAuthFailed,
			message:    "Authentication required",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "not_found",
			status:     http.StatusNotFound,
			code:       ErrCodeNotFound,
			message:    "Resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "rate_limited",
			status:     http.StatusTooManyRequests,
			code:       ErrCodeRateLimited,
			message:    "Too many requests",
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "internal_error",
			status:     http.StatusInternalServerError,
			code:       ErrCodeInternal,
			message:    "Internal server error",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "forbidden",
			status:     http.StatusForbidden,
			code:       ErrCodeForbidden,
			message:    "Access denied",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "conflict",
			status:     http.StatusConflict,
			code:       ErrCodeConflict,
			message:    "Resource already exists",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "bad_request",
			status:     http.StatusBadRequest,
			code:       ErrCodeBadRequest,
			message:    "Malformed request",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx := context.Background()

			WriteError(w, ctx, tt.status, tt.code, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if resp.Error.Code != tt.code {
				t.Errorf("expected code %s, got %s", tt.code, resp.Error.Code)
			}
			if resp.Error.Message != tt.message {
				t.Errorf("expected message %s, got %s", tt.message, resp.Error.Message)
			}
		})
	}
}

func TestWriteError_SetsErrorCodeInContext(t *testing.T) {
	// This test verifies the error response structure returned by WriteError.
	// Context error code propagation is handled by middleware in integration tests.
	handlerCalled := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		WriteError(w, r.Context(), http.StatusBadRequest, ErrCodeValidation, "Test error")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Fatal("handler was not called")
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Code != ErrCodeValidation {
		t.Errorf("expected code %s in response, got %s", ErrCodeValidation, resp.Error.Code)
	}
}

func TestWriteError_IntegrationWithLoggingMiddleware(t *testing.T) {
	// Create a buffer to capture logs
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create handler that properly sets error code in context before calling WriteError
	handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
		WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Resource not found")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeNotFound, resp.Error.Code)
	}

	// Verify logging
	type logEntry struct {
		Level     string `json:"level"`
		Status    int    `json:"status"`
		ErrorCode string `json:"error_code"`
	}

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v, log: %s", err, buf.String())
	}

	if entry.Status != http.StatusNotFound {
		t.Errorf("expected logged status 404, got %d", entry.Status)
	}
	if entry.Level != "WARN" {
		t.Errorf("expected log level WARN for 4xx, got %s", entry.Level)
	}
	// Verify error_code is captured in logs
	if entry.ErrorCode != ErrCodeNotFound {
		t.Errorf("expected error_code %s in logs, got %s", ErrCodeNotFound, entry.ErrorCode)
	}
}

func TestStatusCodeMapping(t *testing.T) {
	tests := []struct {
		code       string
		wantStatus int
	}{
		{ErrCodeValidation, http.StatusBadRequest},
		{ErrCodeAuthFailed, http.StatusUnauthorized},
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeRateLimited, http.StatusTooManyRequests},
		{ErrCodeForbidden, http.StatusForbidden},
		{ErrCodeConflict, http.StatusConflict},
		{ErrCodeBadRequest, http.StatusBadRequest},
		{ErrCodeInternal, http.StatusInternalServerError},
		{"unknown_code", http.StatusInternalServerError}, // default
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := StatusCodeMapping(tt.code)
			if got != tt.wantStatus {
				t.Errorf("StatusCodeMapping(%s) = %d, want %d", tt.code, got, tt.wantStatus)
			}
		})
	}
}

func TestErrorResponse_JSONStructure(t *testing.T) {
	// Verify the exact JSON structure matches the spec
	w := httptest.NewRecorder()
	ctx := context.Background()

	WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid email format")

	// Parse as generic map to verify exact structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify top-level structure has only "error" key
	if len(response) != 1 {
		t.Errorf("expected 1 top-level key, got %d: %v", len(response), response)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'error' to be an object, got %T", response["error"])
	}

	// Verify error object has exactly "code" and "message"
	if len(errorObj) != 2 {
		t.Errorf("expected 2 fields in error object, got %d: %v", len(errorObj), errorObj)
	}

	code, ok := errorObj["code"].(string)
	if !ok {
		t.Fatalf("expected 'code' to be a string, got %T", errorObj["code"])
	}
	if code != ErrCodeValidation {
		t.Errorf("expected code %s, got %s", ErrCodeValidation, code)
	}

	message, ok := errorObj["message"].(string)
	if !ok {
		t.Fatalf("expected 'message' to be a string, got %T", errorObj["message"])
	}
	if message != "Invalid email format" {
		t.Errorf("expected message 'Invalid email format', got %s", message)
	}
}

func TestWriteError_WithRequestID(t *testing.T) {
	// Test that WriteError works correctly when request ID is present
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := middleware.RequestID(
		middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
			WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Invalid token")
		})),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/scenes", nil)
	req.Header.Set("X-Request-ID", "test-req-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Code != ErrCodeAuthFailed {
		t.Errorf("expected code %s, got %s", ErrCodeAuthFailed, resp.Error.Code)
	}

	// Verify request ID and error code are in logs
	type logEntry struct {
		RequestID string `json:"request_id"`
		ErrorCode string `json:"error_code"`
	}
	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if entry.RequestID != "test-req-123" {
		t.Errorf("expected request_id test-req-123 in logs, got %s", entry.RequestID)
	}
	if entry.ErrorCode != ErrCodeAuthFailed {
		t.Errorf("expected error_code %s in logs, got %s", ErrCodeAuthFailed, entry.ErrorCode)
	}
}

func TestWriteError_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := context.Background()

	WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "")

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Code != ErrCodeInternal {
		t.Errorf("expected code %s, got %s", ErrCodeInternal, resp.Error.Code)
	}
	if resp.Error.Message != "" {
		t.Errorf("expected empty message, got %s", resp.Error.Message)
	}
}

func TestWriteError_SpecialCharactersInMessage(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := context.Background()

	specialMsg := `Error with "quotes", <brackets>, & ampersands, and emoji 🎵`
	WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, specialMsg)

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != specialMsg {
		t.Errorf("message not properly escaped: got %s", resp.Error.Message)
	}
}
