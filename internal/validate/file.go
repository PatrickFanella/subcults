package validate

import (
	"errors"
	"fmt"
	"strings"
)

// File validation errors
var (
	ErrInvalidMIMEType = errors.New("invalid MIME type")
	ErrFileTooLarge    = errors.New("file too large")
	ErrFileTooSmall    = errors.New("file too small")
)

// Common MIME type categories
const (
	MIMEImageJPEG = "image/jpeg"
	MIMEImagePNG  = "image/png"
	MIMEImageGIF  = "image/gif"
	MIMEImageWebP = "image/webp"
	MIMEAudioMPEG = "audio/mpeg"
	MIMEAudioWAV  = "audio/wav"
	MIMEAudioOGG  = "audio/ogg"
	MIMEVideoMP4  = "video/mp4"
	MIMEVideoWebM = "video/webm"
)

// AllowedImageTypes defines allowed image MIME types.
var AllowedImageTypes = []string{
	MIMEImageJPEG,
	MIMEImagePNG,
	MIMEImageGIF,
	MIMEImageWebP,
}

// AllowedAudioTypes defines allowed audio MIME types.
var AllowedAudioTypes = []string{
	MIMEAudioMPEG,
	MIMEAudioWAV,
	MIMEAudioOGG,
}

// AllowedVideoTypes defines allowed video MIME types.
var AllowedVideoTypes = []string{
	MIMEVideoMP4,
	MIMEVideoWebM,
}

// FileConstraints defines validation constraints for file uploads.
type FileConstraints struct {
	AllowedTypes []string // Allowed MIME types
	MaxSizeBytes int64    // Maximum file size in bytes
	MinSizeBytes int64    // Minimum file size in bytes (0 = no minimum)
}

// MIMEType validates a MIME type against allowed types.
// Returns the normalized MIME type (lowercased) and an error if invalid.
func MIMEType(mimeType string, allowedTypes []string) (string, error) {
	// Normalize: trim whitespace and lowercase
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))

	if mimeType == "" {
		return "", ErrEmpty
	}

	// Check if in allowed list
	for _, allowed := range allowedTypes {
		if mimeType == strings.ToLower(allowed) {
			return mimeType, nil
		}
	}

	return "", fmt.Errorf("%w: %q not in allowed types", ErrInvalidMIMEType, mimeType)
}

// FileSize validates a file size against constraints.
func FileSize(sizeBytes int64, constraints FileConstraints) error {
	if sizeBytes <= 0 {
		return errors.New("file size must be positive")
	}

	if constraints.MinSizeBytes > 0 && sizeBytes < constraints.MinSizeBytes {
		return fmt.Errorf("%w: got %d bytes, minimum is %d", ErrFileTooSmall, sizeBytes, constraints.MinSizeBytes)
	}

	if constraints.MaxSizeBytes > 0 && sizeBytes > constraints.MaxSizeBytes {
		return fmt.Errorf("%w: got %d bytes, maximum is %d", ErrFileTooLarge, sizeBytes, constraints.MaxSizeBytes)
	}

	return nil
}

// File validates both MIME type and file size.
func File(mimeType string, sizeBytes int64, constraints FileConstraints) (string, error) {
	// Validate MIME type
	validatedType, err := MIMEType(mimeType, constraints.AllowedTypes)
	if err != nil {
		return "", err
	}

	// Validate size
	if err := FileSize(sizeBytes, constraints); err != nil {
		return "", err
	}

	return validatedType, nil
}

// ImageFile validates an image file upload.
// Uses default image constraints: allowed image types, max 10MB.
func ImageFile(mimeType string, sizeBytes int64) (string, error) {
	return File(mimeType, sizeBytes, FileConstraints{
		AllowedTypes: AllowedImageTypes,
		MaxSizeBytes: 10 * 1024 * 1024, // 10MB
		MinSizeBytes: 0,
	})
}

// AudioFile validates an audio file upload.
// Uses default audio constraints: allowed audio types, max 50MB.
func AudioFile(mimeType string, sizeBytes int64) (string, error) {
	return File(mimeType, sizeBytes, FileConstraints{
		AllowedTypes: AllowedAudioTypes,
		MaxSizeBytes: 50 * 1024 * 1024, // 50MB
		MinSizeBytes: 0,
	})
}

// VideoFile validates a video file upload.
// Uses default video constraints: allowed video types, max 500MB.
func VideoFile(mimeType string, sizeBytes int64) (string, error) {
	return File(mimeType, sizeBytes, FileConstraints{
		AllowedTypes: AllowedVideoTypes,
		MaxSizeBytes: 500 * 1024 * 1024, // 500MB
		MinSizeBytes: 0,
	})
}
