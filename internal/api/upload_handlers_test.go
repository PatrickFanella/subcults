package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onnwee/subcults/internal/upload"
)

// TestSignUpload_InvalidJSON tests handling of malformed JSON.
func TestSignUpload_InvalidJSON(t *testing.T) {
	// Create a minimal upload service configuration for testing
	// Note: This won't be used for actual S3 calls in this test
	service, err := upload.NewService(upload.ServiceConfig{
		BucketName:      "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "https://test.r2.cloudflarestorage.com",
		MaxSizeMB:       15,
	})
	if err != nil {
		t.Fatalf("failed to create upload service: %v", err)
	}

	handlers := NewUploadHandlers(service)

	req := httptest.NewRequest(http.MethodPost, "/uploads/sign", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.SignUpload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeBadRequest {
		t.Errorf("expected error code %s, got %s", ErrCodeBadRequest, errResp.Error.Code)
	}
}

// TestSignUpload_MissingContentType tests validation when contentType is missing.
func TestSignUpload_MissingContentType(t *testing.T) {
	service, err := upload.NewService(upload.ServiceConfig{
		BucketName:      "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "https://test.r2.cloudflarestorage.com",
		MaxSizeMB:       15,
	})
	if err != nil {
		t.Fatalf("failed to create upload service: %v", err)
	}

	handlers := NewUploadHandlers(service)

	reqBody := SignUploadRequest{
		ContentType: "",
		SizeBytes:   1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploads/sign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.SignUpload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code %s, got %s", ErrCodeValidation, errResp.Error.Code)
	}
}

// TestSignUpload_InvalidSize tests validation when sizeBytes is invalid.
func TestSignUpload_InvalidSize(t *testing.T) {
	service, err := upload.NewService(upload.ServiceConfig{
		BucketName:      "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "https://test.r2.cloudflarestorage.com",
		MaxSizeMB:       15,
	})
	if err != nil {
		t.Fatalf("failed to create upload service: %v", err)
	}

	handlers := NewUploadHandlers(service)

	tests := []struct {
		name      string
		sizeBytes int64
	}{
		{"zero size", 0},
		{"negative size", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := SignUploadRequest{
				ContentType: "image/jpeg",
				SizeBytes:   tt.sizeBytes,
			}

			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/uploads/sign", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handlers.SignUpload(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != ErrCodeValidation {
				t.Errorf("expected error code %s, got %s", ErrCodeValidation, errResp.Error.Code)
			}
		})
	}
}

// TestSignUpload_UnsupportedType tests handling of unsupported MIME types.
func TestSignUpload_UnsupportedType(t *testing.T) {
	service, err := upload.NewService(upload.ServiceConfig{
		BucketName:      "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "https://test.r2.cloudflarestorage.com",
		MaxSizeMB:       15,
	})
	if err != nil {
		t.Fatalf("failed to create upload service: %v", err)
	}

	handlers := NewUploadHandlers(service)

	reqBody := SignUploadRequest{
		ContentType: "image/gif", // Unsupported type
		SizeBytes:   1024 * 1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploads/sign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.SignUpload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeUnsupportedType {
		t.Errorf("expected error code %s, got %s", ErrCodeUnsupportedType, errResp.Error.Code)
	}
}

// TestSignUpload_FileTooLarge tests handling of oversized files.
func TestSignUpload_FileTooLarge(t *testing.T) {
	service, err := upload.NewService(upload.ServiceConfig{
		BucketName:      "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "https://test.r2.cloudflarestorage.com",
		MaxSizeMB:       15,
	})
	if err != nil {
		t.Fatalf("failed to create upload service: %v", err)
	}

	handlers := NewUploadHandlers(service)

	reqBody := SignUploadRequest{
		ContentType: "image/jpeg",
		SizeBytes:   20 * 1024 * 1024, // 20MB - exceeds 15MB limit
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploads/sign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.SignUpload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code %s, got %s", ErrCodeValidation, errResp.Error.Code)
	}

	if errResp.Error.Message != "File size exceeds maximum allowed" {
		t.Errorf("unexpected error message: %s", errResp.Error.Message)
	}
}
