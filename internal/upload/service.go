// Package upload provides services for generating signed URLs for direct R2 uploads.
package upload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// Allowed MIME types for uploads
const (
	MIMEImageJPEG = "image/jpeg"
	MIMEImagePNG  = "image/png"
	MIMEAudioMPEG = "audio/mpeg"
	MIMEAudioWAV  = "audio/wav"
)

// Validation errors
var (
	ErrUnsupportedType = errors.New("unsupported content type")
	ErrFileTooLarge    = errors.New("file size exceeds maximum allowed")
	ErrInvalidPostID   = errors.New("invalid post ID")
)

// AllowedMIMETypes maps allowed MIME types to their file extensions
var AllowedMIMETypes = map[string]string{
	MIMEImageJPEG: ".jpg",
	MIMEImagePNG:  ".png",
	MIMEAudioMPEG: ".mp3",
	MIMEAudioWAV:  ".wav",
}

// SignedURLRequest represents a request for a signed upload URL.
type SignedURLRequest struct {
	ContentType string  // MIME type of the file
	SizeBytes   int64   // Size of the file in bytes
	PostID      *string // Optional post ID; if nil, uses "temp"
}

// SignedURLResponse represents the response containing the signed URL and metadata.
type SignedURLResponse struct {
	URL       string    `json:"url"`        // Pre-signed PUT URL
	Key       string    `json:"key"`        // Object key in R2
	ExpiresAt time.Time `json:"expires_at"` // URL expiration time
}

// Service handles generating signed URLs for R2 uploads.
type Service struct {
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
	maxSizeBytes  int64
	urlExpiry     time.Duration
	timeNow       func() time.Time // For testability
}

// ServiceConfig holds configuration for the upload service.
type ServiceConfig struct {
	BucketName       string
	AccessKeyID      string
	SecretAccessKey  string
	Endpoint         string
	MaxSizeMB        int
	URLExpiryMinutes int // Default: 5 minutes
}

// NewService creates a new upload service with the given configuration.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.BucketName == "" {
		return nil, errors.New("bucket name is required")
	}
	if cfg.AccessKeyID == "" {
		return nil, errors.New("access key ID is required")
	}
	if cfg.SecretAccessKey == "" {
		return nil, errors.New("secret access key is required")
	}
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	// Default values
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 15
	}
	if cfg.URLExpiryMinutes <= 0 {
		cfg.URLExpiryMinutes = 5
	}

	// Create S3 client with R2-compatible configuration
	s3Client := s3.New(s3.Options{
		Region: "auto", // R2 uses auto region
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"", // No session token for R2
		)),
		BaseEndpoint: aws.String(cfg.Endpoint),
		UsePathStyle: true, // R2 requires path-style addressing
	})

	presignClient := s3.NewPresignClient(s3Client)

	return &Service{
		s3Client:      s3Client,
		presignClient: presignClient,
		bucketName:    cfg.BucketName,
		maxSizeBytes:  int64(cfg.MaxSizeMB) * 1024 * 1024,
		urlExpiry:     time.Duration(cfg.URLExpiryMinutes) * time.Minute,
		timeNow:       time.Now,
	}, nil
}

// ValidateContentType checks if the content type is allowed.
func ValidateContentType(contentType string) error {
	if _, ok := AllowedMIMETypes[contentType]; !ok {
		return ErrUnsupportedType
	}
	return nil
}

// ValidateFileSize checks if the file size is within limits.
func (s *Service) ValidateFileSize(sizeBytes int64) error {
	if sizeBytes > s.maxSizeBytes {
		return ErrFileTooLarge
	}
	if sizeBytes <= 0 {
		return errors.New("file size must be positive")
	}
	return nil
}

// GenerateObjectKey creates a unique object key for the upload.
// Pattern: posts/{postId or temp}/uuid.ext
func GenerateObjectKey(contentType string, postID *string) (string, error) {
	ext, ok := AllowedMIMETypes[contentType]
	if !ok {
		return "", ErrUnsupportedType
	}

	// Generate UUID for uniqueness
	objectUUID := uuid.New().String()

	// Use postID if provided, otherwise use "temp"
	prefix := "temp"
	if postID != nil && *postID != "" {
		// Sanitize postID: only alphanumeric, hyphens, underscores
		sanitized := sanitizePathComponent(*postID)
		if sanitized == "" {
			return "", ErrInvalidPostID
		}
		prefix = sanitized
	}

	key := fmt.Sprintf("posts/%s/%s%s", prefix, objectUUID, ext)
	return key, nil
}

// sanitizePathComponent removes potentially dangerous characters from path components.
func sanitizePathComponent(s string) string {
	// Only allow alphanumeric, hyphens, and underscores
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GenerateSignedURL generates a pre-signed PUT URL for direct upload to R2.
func (s *Service) GenerateSignedURL(ctx context.Context, req SignedURLRequest) (*SignedURLResponse, error) {
	// Validate content type
	if err := ValidateContentType(req.ContentType); err != nil {
		return nil, err
	}

	// Validate file size
	if err := s.ValidateFileSize(req.SizeBytes); err != nil {
		return nil, err
	}

	// Generate object key
	key, err := GenerateObjectKey(req.ContentType, req.PostID)
	if err != nil {
		return nil, err
	}

	// Create presigned PUT request
	putObjectInput := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(key),
		ContentType:   aws.String(req.ContentType),
		ContentLength: aws.Int64(req.SizeBytes),
	}

	presignedReq, err := s.presignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = s.urlExpiry
	})
	if err != nil {
		return nil, fmt.Errorf("failed to presign request: %w", err)
	}

	expiresAt := s.timeNow().Add(s.urlExpiry)

	return &SignedURLResponse{
		URL:       presignedReq.URL,
		Key:       key,
		ExpiresAt: expiresAt,
	}, nil
}

// GetS3Client returns the S3 client used by the service.
// This can be used by other services that need to interact with R2.
func (s *Service) GetS3Client() *s3.Client {
	return s.s3Client
}

// GetBucketName returns the bucket name used by the service.
func (s *Service) GetBucketName() string {
	return s.bucketName
}
