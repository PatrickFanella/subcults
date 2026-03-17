package attachment

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestNewMetadataService tests metadata service initialization.
func TestNewMetadataService(t *testing.T) {
	tests := []struct {
		name        string
		config      MetadataServiceConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: MetadataServiceConfig{
				S3Client:   &s3.Client{},
				BucketName: "test-bucket",
			},
			expectError: false,
		},
		{
			name: "missing s3 client",
			config: MetadataServiceConfig{
				BucketName: "test-bucket",
			},
			expectError: true,
			errorMsg:    "s3 client is required",
		},
		{
			name: "missing bucket name",
			config: MetadataServiceConfig{
				S3Client: &s3.Client{},
			},
			expectError: true,
			errorMsg:    "bucket name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewMetadataService(tt.config)

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
			}
		})
	}
}

// TestIsImageType tests image content type detection.
func TestIsImageType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "image/jpeg",
			contentType: "image/jpeg",
			expected:    true,
		},
		{
			name:        "image/png",
			contentType: "image/png",
			expected:    true,
		},
		{
			name:        "image/webp",
			contentType: "image/webp",
			expected:    true,
		},
		{
			name:        "audio/mpeg",
			contentType: "audio/mpeg",
			expected:    false,
		},
		{
			name:        "video/mp4",
			contentType: "video/mp4",
			expected:    false,
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "empty string",
			contentType: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageType(tt.contentType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestIsAudioType tests audio content type detection.
func TestIsAudioType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "audio/mpeg",
			contentType: "audio/mpeg",
			expected:    true,
		},
		{
			name:        "audio/wav",
			contentType: "audio/wav",
			expected:    true,
		},
		{
			name:        "audio/ogg",
			contentType: "audio/ogg",
			expected:    true,
		},
		{
			name:        "image/jpeg",
			contentType: "image/jpeg",
			expected:    false,
		},
		{
			name:        "video/mp4",
			contentType: "video/mp4",
			expected:    false,
		},
		{
			name:        "empty string",
			contentType: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAudioType(tt.contentType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestEnrichAttachment_InvalidKey tests error handling for invalid keys.
func TestEnrichAttachment_InvalidKey(t *testing.T) {
	service := &MetadataService{
		s3Client:   &s3.Client{},
		bucketName: "test-bucket",
	}

	ctx := context.Background()

	// Test empty key - should return error before calling S3
	_, err := service.EnrichAttachment(ctx, "")
	if err == nil {
		t.Errorf("expected error for empty key, got nil")
	}
	if err != ErrInvalidObjectKey {
		t.Errorf("expected ErrInvalidObjectKey, got %v", err)
	}

	// Note: For valid key tests, we'd need to mock the S3 client
	// which is beyond the scope of unit tests - those are integration tests
	// The actual HeadObject and GetObject calls would be tested with mocks or in integration tests
}
