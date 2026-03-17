package attachment

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockS3Client is a simple mock for testing S3 operations
type mockS3Client struct {
	headObjectFunc func(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	getObjectFunc  func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	putObjectFunc  func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func (m *mockS3Client) HeadObject(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectFunc != nil {
		return m.headObjectFunc(ctx, input, opts...)
	}
	return nil, fmt.Errorf("HeadObject not mocked")
}

func (m *mockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, input, opts...)
	}
	return nil, fmt.Errorf("GetObject not mocked")
}

func (m *mockS3Client) PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, input, opts...)
	}
	return nil, fmt.Errorf("PutObject not mocked")
}

// TestEnrichAttachment_ImageProcessing tests image metadata extraction with mocked S3.
func TestEnrichAttachment_ImageProcessing(t *testing.T) {
	// Create a simple 1x1 PNG image
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE,
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54,
		0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00, 0x00,
		0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D, 0xB4,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
		0xAE, 0x42, 0x60, 0x82,
	}

	contentType := "image/png"
	contentLength := int64(len(pngData))

	mock := &mockS3Client{
		headObjectFunc: func(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			return &s3.HeadObjectOutput{
				ContentType:   &contentType,
				ContentLength: &contentLength,
			}, nil
		},
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body: io.NopCloser(bytes.NewReader(pngData)),
			}, nil
		},
		putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return &s3.PutObjectOutput{}, nil
		},
	}

	service := &MetadataService{
		s3Client:   mock,
		bucketName: "test-bucket",
	}

	ctx := context.Background()
	attachment, err := service.EnrichAttachment(ctx, "posts/test/image.png")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attachment.Key != "posts/test/image.png" {
		t.Errorf("expected key 'posts/test/image.png', got %s", attachment.Key)
	}
	if attachment.Type != "image/png" {
		t.Errorf("expected type 'image/png', got %s", attachment.Type)
	}
	if attachment.SizeBytes <= 0 {
		t.Errorf("expected positive size, got %d", attachment.SizeBytes)
	}
	if attachment.Width == nil {
		t.Error("expected width to be set")
	} else if *attachment.Width != 1 {
		t.Errorf("expected width 1, got %d", *attachment.Width)
	}
	if attachment.Height == nil {
		t.Error("expected height to be set")
	} else if *attachment.Height != 1 {
		t.Errorf("expected height 1, got %d", *attachment.Height)
	}
}

// TestEnrichAttachment_NonImageAttachment tests audio attachment handling.
func TestEnrichAttachment_NonImageAttachment(t *testing.T) {
	contentType := "audio/mpeg"
	contentLength := int64(1000)

	mock := &mockS3Client{
		headObjectFunc: func(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			return &s3.HeadObjectOutput{
				ContentType:   &contentType,
				ContentLength: &contentLength,
			}, nil
		},
	}

	service := &MetadataService{
		s3Client:   mock,
		bucketName: "test-bucket",
	}

	ctx := context.Background()
	attachment, err := service.EnrichAttachment(ctx, "posts/test/audio.mp3")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachment.Type != "audio/mpeg" {
		t.Errorf("expected type 'audio/mpeg', got %s", attachment.Type)
	}
	if attachment.SizeBytes != 1000 {
		t.Errorf("expected size 1000, got %d", attachment.SizeBytes)
	}
	if attachment.Width != nil {
		t.Errorf("audio should not have width, got %v", attachment.Width)
	}
	if attachment.Height != nil {
		t.Errorf("audio should not have height, got %v", attachment.Height)
	}
}

// TestEnrichAttachment_S3Errors tests error handling for S3 failures.
func TestEnrichAttachment_S3Errors(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *mockS3Client
		expectedError string
	}{
		{
			name: "HeadObject error",
			setupMock: func() *mockS3Client {
				return &mockS3Client{
					headObjectFunc: func(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
						return nil, fmt.Errorf("object not found")
					},
				}
			},
			expectedError: "object not found",
		},
		{
			name: "GetObject error for image - graceful degradation",
			setupMock: func() *mockS3Client {
				contentType := "image/png"
				contentLength := int64(100)
				return &mockS3Client{
					headObjectFunc: func(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
						return &s3.HeadObjectOutput{
							ContentType:   &contentType,
							ContentLength: &contentLength,
						}, nil
					},
					getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						return nil, fmt.Errorf("network error")
					},
				}
			},
			expectedError: "", // No error - gracefully falls back to basic metadata
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MetadataService{
				s3Client:   tt.setupMock(),
				bucketName: "test-bucket",
			}

			ctx := context.Background()
			_, err := service.EnrichAttachment(ctx, "posts/test/image.png")

			if tt.expectedError == "" {
				// Expecting success (graceful degradation)
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				// Expecting error
				if err == nil {
					t.Error("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			}
		})
	}
}

// TestDetermineOutputFormat tests format preservation logic.
func TestDetermineOutputFormat(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		bimgType    string
		expected    string
	}{
		{
			name:        "PNG from content type",
			contentType: "image/png",
			bimgType:    "png",
			expected:    "png",
		},
		{
			name:        "JPEG from content type",
			contentType: "image/jpeg",
			bimgType:    "jpeg",
			expected:    "jpeg",
		},
		{
			name:        "WebP from content type",
			contentType: "image/webp",
			bimgType:    "webp",
			expected:    "webp",
		},
		{
			name:        "PNG from bimg when content type unknown",
			contentType: "application/octet-stream",
			bimgType:    "png",
			expected:    "png",
		},
		{
			name:        "Default to JPEG when both unknown",
			contentType: "unknown",
			bimgType:    "unknown",
			expected:    "jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineOutputFormat(tt.contentType, tt.bimgType)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
