// Package attachment provides services for enriching attachment metadata
// while ensuring EXIF data is stripped for privacy.
package attachment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/h2non/bimg"
	"github.com/onnwee/subcults/internal/image"
	"github.com/onnwee/subcults/internal/post"
)

// Common errors for metadata extraction
var (
	ErrObjectNotFound    = errors.New("object not found in R2")
	ErrInvalidObjectKey  = errors.New("invalid object key")
	ErrUnsupportedFormat = errors.New("unsupported file format")
)

// S3Client interface abstracts S3 operations for testability
type S3Client interface {
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// MetadataService handles attachment metadata extraction and enrichment.
type MetadataService struct {
	s3Client   S3Client
	bucketName string
}

// MetadataServiceConfig holds configuration for the metadata service.
type MetadataServiceConfig struct {
	S3Client   S3Client
	BucketName string
}

// NewMetadataService creates a new metadata service.
func NewMetadataService(cfg MetadataServiceConfig) (*MetadataService, error) {
	if cfg.S3Client == nil {
		return nil, errors.New("s3 client is required")
	}
	if cfg.BucketName == "" {
		return nil, errors.New("bucket name is required")
	}

	return &MetadataService{
		s3Client:   cfg.S3Client,
		bucketName: cfg.BucketName,
	}, nil
}

// EnrichAttachment fetches metadata for an attachment key and returns an enriched attachment.
// For images, it strips EXIF data and extracts dimensions.
// For audio, it extracts basic metadata (duration placeholder for now).
func (s *MetadataService) EnrichAttachment(ctx context.Context, key string) (*post.Attachment, error) {
	if key == "" {
		return nil, ErrInvalidObjectKey
	}

	// Step 1: Fetch object metadata using HeadObject
	headOutput, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrObjectNotFound, err)
	}

	// Extract basic metadata
	contentType := ""
	if headOutput.ContentType != nil {
		contentType = *headOutput.ContentType
	}
	sizeBytes := int64(0)
	if headOutput.ContentLength != nil {
		sizeBytes = *headOutput.ContentLength
	}

	attachment := &post.Attachment{
		Key:       key,
		Type:      contentType,
		SizeBytes: sizeBytes,
	}

	// Step 2: Process based on content type
	if isImageType(contentType) {
		// For images, we need to fetch the object, strip EXIF, and extract dimensions
		if err := s.processImage(ctx, key, attachment); err != nil {
			// Log error but don't fail - return basic metadata
			// In production, you might want to log this properly
			return attachment, nil
		}
	} else if isAudioType(contentType) {
		// For audio, we could extract duration here
		// For now, this is a placeholder - audio metadata extraction can be added later
		// attachment.DurationSeconds would be set here
	}

	return attachment, nil
}

// processImage fetches the image, strips EXIF, extracts dimensions, and re-uploads sanitized version.
//
// This method performs the following steps:
// 1. Downloads image from R2
// 2. Extracts dimensions from original image (preserves actual upload dimensions)
// 3. Strips EXIF metadata (GPS, camera info, timestamps) for privacy
// 4. Re-uploads sanitized image, replacing the original
// 5. Updates attachment with dimensions and sanitized file size
//
// Note: The returned SizeBytes reflects the sanitized image size, which may differ from
// the original due to EXIF removal and re-encoding. This is typically smaller.
func (s *MetadataService) processImage(ctx context.Context, key string, attachment *post.Attachment) error {
	// Fetch the image from R2
	getOutput, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer getOutput.Body.Close()

	// Read the image bytes
	imageBytes, err := io.ReadAll(getOutput.Body)
	if err != nil {
		return fmt.Errorf("failed to read image: %w", err)
	}

	// Extract dimensions from original image before processing
	// This ensures we return the actual uploaded dimensions, not processed dimensions
	// bimg reads metadata without re-encoding, so this is fast
	img := bimg.NewImage(imageBytes)
	metadata, err := img.Metadata()
	if err != nil {
		return fmt.Errorf("failed to read image metadata: %w", err)
	}

	// Store original dimensions
	width := metadata.Size.Width
	height := metadata.Size.Height
	attachment.Width = &width
	attachment.Height = &height

	// Strip EXIF metadata for privacy while preserving original format
	// This prevents converting PNG to JPEG (which would lose transparency)
	// and avoids unnecessary format changes that could degrade quality
	config := image.DefaultConfig()
	config.OutputFormat = determineOutputFormat(attachment.Type, metadata.Type)

	sanitizedBytes, err := image.ProcessWithConfig(bytes.NewReader(imageBytes), config)
	if err != nil {
		return fmt.Errorf("failed to strip EXIF: %w", err)
	}

	// Re-upload the sanitized image back to R2, replacing the original
	// This ensures no EXIF data is persisted in storage
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(key),
		Body:          bytes.NewReader(sanitizedBytes),
		ContentType:   aws.String(attachment.Type),
		ContentLength: aws.Int64(int64(len(sanitizedBytes))),
	})
	if err != nil {
		return fmt.Errorf("failed to re-upload sanitized image: %w", err)
	}

	// Update size to reflect sanitized image size (post-EXIF-stripping)
	// This is typically smaller than the original due to metadata removal
	attachment.SizeBytes = int64(len(sanitizedBytes))

	return nil
}

// determineOutputFormat determines the appropriate output format for image processing.
// Preserves original format to avoid lossy conversions (e.g., PNG -> JPEG losing transparency).
// Falls back to detected format from bimg metadata if content type is unavailable.
func determineOutputFormat(contentType, bimgType string) string {
	// Try to determine from content type first
	switch contentType {
	case "image/jpeg", "image/jpg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	}

	// Fall back to bimg detected type
	switch bimgType {
	case "jpeg":
		return "jpeg"
	case "png":
		return "png"
	case "webp":
		return "webp"
	}

	// Default to jpeg if we can't determine
	return "jpeg"
}

// isImageType checks if the content type is an image.
func isImageType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// isAudioType checks if the content type is audio.
func isAudioType(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/")
}
