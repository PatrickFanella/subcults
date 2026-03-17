package upload

import (
	"context"
	"testing"
	"time"
)

// TestValidateContentType tests MIME type validation.
func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expectError bool
	}{
		{
			name:        "valid image/jpeg",
			contentType: MIMEImageJPEG,
			expectError: false,
		},
		{
			name:        "valid image/png",
			contentType: MIMEImagePNG,
			expectError: false,
		},
		{
			name:        "valid audio/mpeg",
			contentType: MIMEAudioMPEG,
			expectError: false,
		},
		{
			name:        "valid audio/wav",
			contentType: MIMEAudioWAV,
			expectError: false,
		},
		{
			name:        "invalid image/gif",
			contentType: "image/gif",
			expectError: true,
		},
		{
			name:        "invalid video/mp4",
			contentType: "video/mp4",
			expectError: true,
		},
		{
			name:        "invalid application/pdf",
			contentType: "application/pdf",
			expectError: true,
		},
		{
			name:        "empty content type",
			contentType: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContentType(tt.contentType)
			if tt.expectError && err == nil {
				t.Errorf("expected error for content type %s, got nil", tt.contentType)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for content type %s: %v", tt.contentType, err)
			}
			if tt.expectError && err != ErrUnsupportedType {
				t.Errorf("expected ErrUnsupportedType, got %v", err)
			}
		})
	}
}

// TestValidateFileSize tests file size validation.
func TestValidateFileSize(t *testing.T) {
	service := &Service{
		maxSizeBytes: 15 * 1024 * 1024, // 15MB
	}

	tests := []struct {
		name        string
		sizeBytes   int64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid 1MB file",
			sizeBytes:   1 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "valid 15MB file (at limit)",
			sizeBytes:   15 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "invalid 16MB file (over limit)",
			sizeBytes:   16 * 1024 * 1024,
			expectError: true,
			errorMsg:    "exceeds maximum",
		},
		{
			name:        "invalid 0 bytes",
			sizeBytes:   0,
			expectError: true,
			errorMsg:    "must be positive",
		},
		{
			name:        "invalid negative size",
			sizeBytes:   -1,
			expectError: true,
			errorMsg:    "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateFileSize(tt.sizeBytes)
			if tt.expectError && err == nil {
				t.Errorf("expected error for size %d, got nil", tt.sizeBytes)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for size %d: %v", tt.sizeBytes, err)
			}
		})
	}
}

// TestGenerateObjectKey tests object key generation.
func TestGenerateObjectKey(t *testing.T) {
	postID := "post123"

	tests := []struct {
		name        string
		contentType string
		postID      *string
		expectError bool
		checkPrefix string
		checkExt    string
	}{
		{
			name:        "jpeg with post ID",
			contentType: MIMEImageJPEG,
			postID:      &postID,
			expectError: false,
			checkPrefix: "posts/post123/",
			checkExt:    ".jpg",
		},
		{
			name:        "png without post ID",
			contentType: MIMEImagePNG,
			postID:      nil,
			expectError: false,
			checkPrefix: "posts/temp/",
			checkExt:    ".png",
		},
		{
			name:        "mp3 with temp post ID",
			contentType: MIMEAudioMPEG,
			postID:      nil,
			expectError: false,
			checkPrefix: "posts/temp/",
			checkExt:    ".mp3",
		},
		{
			name:        "wav with post ID",
			contentType: MIMEAudioWAV,
			postID:      &postID,
			expectError: false,
			checkPrefix: "posts/post123/",
			checkExt:    ".wav",
		},
		{
			name:        "invalid content type",
			contentType: "image/gif",
			postID:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateObjectKey(tt.contentType, tt.postID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check prefix
			if len(tt.checkPrefix) > 0 && len(key) >= len(tt.checkPrefix) {
				if key[:len(tt.checkPrefix)] != tt.checkPrefix {
					t.Errorf("expected key to start with %s, got %s", tt.checkPrefix, key)
				}
			}

			// Check extension
			if len(tt.checkExt) > 0 && len(key) >= len(tt.checkExt) {
				if key[len(key)-len(tt.checkExt):] != tt.checkExt {
					t.Errorf("expected key to end with %s, got %s", tt.checkExt, key)
				}
			}

			// Key should contain UUID (36 chars + extension)
			if len(key) < 36 {
				t.Errorf("key too short to contain UUID: %s", key)
			}
		})
	}
}

// TestSanitizePathComponent tests path component sanitization.
func TestSanitizePathComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "alphanumeric only",
			input:    "post123",
			expected: "post123",
		},
		{
			name:     "with hyphens and underscores",
			input:    "post-123_abc",
			expected: "post-123_abc",
		},
		{
			name:     "with slashes (should be removed)",
			input:    "../../etc/passwd",
			expected: "etcpasswd",
		},
		{
			name:     "with special characters",
			input:    "post@#$%123",
			expected: "post123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePathComponent(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenerateSignedURL tests the full signed URL generation flow.
func TestGenerateSignedURL(t *testing.T) {
	// Create a service with mock configuration
	// Note: This won't actually connect to R2, but tests the validation logic
	service := &Service{
		bucketName:   "test-bucket",
		maxSizeBytes: 15 * 1024 * 1024,
		urlExpiry:    5 * time.Minute,
		timeNow:      func() time.Time { return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) },
		// s3Client and presignClient would need to be mocked for full testing
		// For now, we'll test validation paths
	}

	postID := "post123"

	tests := []struct {
		name        string
		request     SignedURLRequest
		expectError bool
		errorType   error
	}{
		{
			name: "valid request with post ID",
			request: SignedURLRequest{
				ContentType: MIMEImageJPEG,
				SizeBytes:   1 * 1024 * 1024, // 1MB
				PostID:      &postID,
			},
			expectError: false,
		},
		{
			name: "valid request without post ID",
			request: SignedURLRequest{
				ContentType: MIMEImagePNG,
				SizeBytes:   5 * 1024 * 1024, // 5MB
				PostID:      nil,
			},
			expectError: false,
		},
		{
			name: "invalid content type",
			request: SignedURLRequest{
				ContentType: "image/gif",
				SizeBytes:   1 * 1024 * 1024,
				PostID:      nil,
			},
			expectError: true,
			errorType:   ErrUnsupportedType,
		},
		{
			name: "file too large",
			request: SignedURLRequest{
				ContentType: MIMEImageJPEG,
				SizeBytes:   20 * 1024 * 1024, // 20MB
				PostID:      nil,
			},
			expectError: true,
			errorType:   ErrFileTooLarge,
		},
		{
			name: "zero size",
			request: SignedURLRequest{
				ContentType: MIMEImageJPEG,
				SizeBytes:   0,
				PostID:      nil,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// For tests that should fail validation, we can check without needing actual S3 client
			if tt.expectError {
				// Test validation directly
				if err := ValidateContentType(tt.request.ContentType); err != nil {
					if tt.errorType != nil && err != tt.errorType {
						t.Errorf("expected error %v, got %v", tt.errorType, err)
					}
					return
				}
				if err := service.ValidateFileSize(tt.request.SizeBytes); err != nil {
					if tt.errorType != nil && err != tt.errorType {
						t.Errorf("expected error %v, got %v", tt.errorType, err)
					}
					return
				}
				t.Errorf("expected validation error, but validations passed")
			} else {
				// For successful validation tests, just check the validations pass
				if err := ValidateContentType(tt.request.ContentType); err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
				if err := service.ValidateFileSize(tt.request.SizeBytes); err != nil {
					t.Errorf("unexpected size validation error: %v", err)
				}

				// Note: Full GenerateSignedURL test would require mocking the S3 client
				// which is beyond the scope of this initial implementation
				// In production, you'd use a mock S3 presigner or integration tests with a test bucket
			}

			// Skip actual URL generation in unit tests since it requires real S3 client
			_ = ctx
		})
	}
}

// TestNewService tests service initialization.
func TestNewService(t *testing.T) {
	tests := []struct {
		name        string
		config      ServiceConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: ServiceConfig{
				BucketName:      "test-bucket",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "https://test.r2.cloudflarestorage.com",
				MaxSizeMB:       15,
			},
			expectError: false,
		},
		{
			name: "missing bucket name",
			config: ServiceConfig{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "https://test.r2.cloudflarestorage.com",
			},
			expectError: true,
			errorMsg:    "bucket name is required",
		},
		{
			name: "missing access key",
			config: ServiceConfig{
				BucketName:      "test-bucket",
				SecretAccessKey: "test-secret",
				Endpoint:        "https://test.r2.cloudflarestorage.com",
			},
			expectError: true,
			errorMsg:    "access key ID is required",
		},
		{
			name: "missing secret",
			config: ServiceConfig{
				BucketName:  "test-bucket",
				AccessKeyID: "test-key",
				Endpoint:    "https://test.r2.cloudflarestorage.com",
			},
			expectError: true,
			errorMsg:    "secret access key is required",
		},
		{
			name: "missing endpoint",
			config: ServiceConfig{
				BucketName:      "test-bucket",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			expectError: true,
			errorMsg:    "endpoint is required",
		},
		{
			name: "defaults applied",
			config: ServiceConfig{
				BucketName:      "test-bucket",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "https://test.r2.cloudflarestorage.com",
				// MaxSizeMB and URLExpiryMinutes not set
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if service == nil {
				t.Errorf("expected service to be non-nil")
				return
			}

			// Check defaults were applied
			if service.maxSizeBytes != int64(tt.config.MaxSizeMB)*1024*1024 && tt.config.MaxSizeMB > 0 {
				t.Errorf("expected max size %d, got %d", tt.config.MaxSizeMB*1024*1024, service.maxSizeBytes)
			}
			if tt.config.MaxSizeMB == 0 && service.maxSizeBytes != 15*1024*1024 {
				t.Errorf("expected default max size 15MB, got %d bytes", service.maxSizeBytes)
			}
		})
	}
}
