package image

import (
	"bytes"
	"fmt"
	"io"

	"github.com/h2non/bimg"
)

// ProcessorConfig holds configuration for image processing.
type ProcessorConfig struct {
	// Quality for JPEG/WebP encoding (1-100, default: 85)
	Quality int
	// OutputFormat specifies the output format (jpeg, webp, png)
	OutputFormat string
	// StripMetadata removes all EXIF/metadata (default: true)
	StripMetadata bool
	// MaxWidth limits image width (0 = no limit)
	MaxWidth int
	// MaxHeight limits image height (0 = no limit)
	MaxHeight int
}

// DefaultConfig returns sensible defaults for image processing.
func DefaultConfig() ProcessorConfig {
	return ProcessorConfig{
		Quality:       85,
		OutputFormat:  "jpeg",
		StripMetadata: true,
		MaxWidth:      0,
		MaxHeight:     0,
	}
}

// Processor handles image sanitization and re-encoding.
type Processor struct {
	config ProcessorConfig
}

// NewProcessor creates a new image processor with the given config.
func NewProcessor(config ProcessorConfig) *Processor {
	return &Processor{config: config}
}

// Process takes an image file (as io.Reader) and returns sanitized bytes.
// This function:
// 1. Reads the input image
// 2. Strips all EXIF metadata (GPS, camera details, timestamps)
// 3. Re-encodes to specified format with quality settings
// 4. Applies orientation correction if needed
func Process(r io.Reader) ([]byte, error) {
	return ProcessWithConfig(r, DefaultConfig())
}

// ProcessWithConfig processes an image with custom configuration.
func ProcessWithConfig(r io.Reader, config ProcessorConfig) ([]byte, error) {
	// Read the input image into memory
	inputBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input image: %w", err)
	}

	// Validate that we have a valid image
	img := bimg.NewImage(inputBytes)
	metadata, err := img.Metadata()
	if err != nil {
		return nil, fmt.Errorf("failed to read image metadata: %w", err)
	}

	// Build processing options
	options := bimg.Options{
		Quality:       config.Quality,
		StripMetadata: config.StripMetadata,
		// Auto-orient based on EXIF orientation tag before stripping
		// This ensures images display correctly after EXIF removal
		Rotate: bimg.Angle(0), // Will use EXIF orientation
	}

	// Set output format
	switch config.OutputFormat {
	case "jpeg", "jpg":
		options.Type = bimg.JPEG
	case "webp":
		options.Type = bimg.WEBP
	case "png":
		options.Type = bimg.PNG
	default:
		// Keep original format if not specified
		options.Type = determineImageType(metadata.Type)
	}

	// Apply size constraints if specified
	if config.MaxWidth > 0 || config.MaxHeight > 0 {
		width := metadata.Size.Width
		height := metadata.Size.Height

		// Calculate resize dimensions while maintaining aspect ratio
		if config.MaxWidth > 0 && width > config.MaxWidth {
			options.Width = config.MaxWidth
		}
		if config.MaxHeight > 0 && height > config.MaxHeight {
			options.Height = config.MaxHeight
		}
	}

	// Process the image (this strips EXIF and re-encodes)
	outputBytes, err := img.Process(options)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %w", err)
	}

	return outputBytes, nil
}

// ProcessBytes is a convenience wrapper for processing image bytes directly.
func ProcessBytes(inputBytes []byte) ([]byte, error) {
	return ProcessWithConfig(bytes.NewReader(inputBytes), DefaultConfig())
}

// determineImageType maps bimg's string type to bimg.ImageType constant.
func determineImageType(typeStr string) bimg.ImageType {
	switch typeStr {
	case "jpeg":
		return bimg.JPEG
	case "png":
		return bimg.PNG
	case "webp":
		return bimg.WEBP
	case "gif":
		return bimg.GIF
	case "svg":
		return bimg.SVG
	default:
		// Default to JPEG for unknown types
		return bimg.JPEG
	}
}

// VerifyNoEXIF checks if the image has EXIF metadata.
// Returns true if no EXIF data is present, false otherwise.
func VerifyNoEXIF(imageBytes []byte) (bool, error) {
	img := bimg.NewImage(imageBytes)
	metadata, err := img.Metadata()
	if err != nil {
		return false, fmt.Errorf("failed to read image metadata: %w", err)
	}

	// bimg metadata will not include EXIF data if it was stripped
	// Check if critical EXIF fields are empty (GPS, camera info)
	exif := metadata.EXIF
	hasEXIF := exif.Make != "" || exif.Model != "" ||
		exif.GPSLatitude != "" || exif.GPSLongitude != "" ||
		exif.DateTimeOriginal != "" || exif.Software != ""

	return !hasEXIF, nil
}
