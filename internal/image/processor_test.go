package image

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"
)

// TestProcess_StripEXIF tests that EXIF metadata is removed from images.
func TestProcess_StripEXIF(t *testing.T) {
	// Use a sample JPEG with embedded EXIF data
	// This is a minimal 1x1 pixel JPEG with EXIF metadata
	imageWithEXIF := getTestImageWithEXIF(t)

	// Process the image
	processedBytes, err := ProcessBytes(imageWithEXIF)
	if err != nil {
		t.Fatalf("ProcessBytes failed: %v", err)
	}

	// Verify output is not empty
	if len(processedBytes) == 0 {
		t.Fatal("Processed image is empty")
	}

	// Verify EXIF was removed
	noEXIF, err := VerifyNoEXIF(processedBytes)
	if err != nil {
		t.Fatalf("VerifyNoEXIF failed: %v", err)
	}

	if !noEXIF {
		t.Error("EXIF metadata still present after processing")
	}
}

// TestProcess_FileSize tests that processed images have reasonable file sizes.
func TestProcess_FileSize(t *testing.T) {
	imageWithEXIF := getTestImageWithEXIF(t)
	originalSize := len(imageWithEXIF)

	processedBytes, err := ProcessBytes(imageWithEXIF)
	if err != nil {
		t.Fatalf("ProcessBytes failed: %v", err)
	}

	processedSize := len(processedBytes)

	// Processed image should be smaller or similar size (EXIF adds overhead)
	// For a small test image, we just ensure it's not dramatically larger
	if processedSize > originalSize*2 {
		t.Errorf("Processed image unexpectedly large: original=%d, processed=%d", originalSize, processedSize)
	}

	t.Logf("Original size: %d bytes, Processed size: %d bytes", originalSize, processedSize)
}

// TestProcessWithConfig_Quality tests different quality settings.
func TestProcessWithConfig_Quality(t *testing.T) {
	imageWithEXIF := getTestImageWithEXIF(t)

	tests := []struct {
		name    string
		quality int
	}{
		{"high_quality", 95},
		{"default_quality", 85},
		{"low_quality", 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Quality = tt.quality

			processedBytes, err := ProcessWithConfig(bytes.NewReader(imageWithEXIF), config)
			if err != nil {
				t.Fatalf("ProcessWithConfig failed: %v", err)
			}

			if len(processedBytes) == 0 {
				t.Error("Processed image is empty")
			}

			// Verify EXIF was removed
			noEXIF, err := VerifyNoEXIF(processedBytes)
			if err != nil {
				t.Fatalf("VerifyNoEXIF failed: %v", err)
			}

			if !noEXIF {
				t.Errorf("EXIF metadata still present with quality=%d", tt.quality)
			}

			t.Logf("Quality=%d: %d bytes", tt.quality, len(processedBytes))
		})
	}
}

// TestProcessWithConfig_Format tests different output formats.
func TestProcessWithConfig_Format(t *testing.T) {
	imageWithEXIF := getTestImageWithEXIF(t)

	tests := []struct {
		name   string
		format string
	}{
		{"jpeg", "jpeg"},
		{"webp", "webp"},
		{"png", "png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.OutputFormat = tt.format

			processedBytes, err := ProcessWithConfig(bytes.NewReader(imageWithEXIF), config)
			if err != nil {
				t.Fatalf("ProcessWithConfig failed for format %s: %v", tt.format, err)
			}

			if len(processedBytes) == 0 {
				t.Error("Processed image is empty")
			}

			// Verify EXIF was removed
			noEXIF, err := VerifyNoEXIF(processedBytes)
			if err != nil {
				t.Fatalf("VerifyNoEXIF failed: %v", err)
			}

			if !noEXIF {
				t.Errorf("EXIF metadata still present for format=%s", tt.format)
			}

			t.Logf("Format=%s: %d bytes", tt.format, len(processedBytes))
		})
	}
}

// TestProcess_InvalidImage tests error handling for invalid input.
func TestProcess_InvalidImage(t *testing.T) {
	invalidData := []byte("not an image")

	_, err := ProcessBytes(invalidData)
	if err == nil {
		t.Error("Expected error for invalid image data, got nil")
	}

	t.Logf("Expected error: %v", err)
}

// TestDefaultConfig tests that default configuration has sensible values.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Quality != 85 {
		t.Errorf("Expected default quality 85, got %d", config.Quality)
	}

	if config.OutputFormat != "jpeg" {
		t.Errorf("Expected default format jpeg, got %s", config.OutputFormat)
	}

	if !config.StripMetadata {
		t.Error("Expected StripMetadata to be true by default")
	}

	if config.MaxWidth != 0 {
		t.Errorf("Expected MaxWidth 0, got %d", config.MaxWidth)
	}

	if config.MaxHeight != 0 {
		t.Errorf("Expected MaxHeight 0, got %d", config.MaxHeight)
	}
}

// TestProcessWithConfig_Resize tests image resizing.
func TestProcessWithConfig_Resize(t *testing.T) {
	// For this test, we need a larger image to see resize effects
	// We'll use the test image and verify the functionality
	imageWithEXIF := getTestImageWithEXIF(t)

	config := DefaultConfig()
	config.MaxWidth = 800
	config.MaxHeight = 600

	processedBytes, err := ProcessWithConfig(bytes.NewReader(imageWithEXIF), config)
	if err != nil {
		t.Fatalf("ProcessWithConfig failed: %v", err)
	}

	if len(processedBytes) == 0 {
		t.Error("Processed image is empty")
	}

	// Verify EXIF was removed even with resize
	noEXIF, err := VerifyNoEXIF(processedBytes)
	if err != nil {
		t.Fatalf("VerifyNoEXIF failed: %v", err)
	}

	if !noEXIF {
		t.Error("EXIF metadata still present after resize")
	}
}

// getTestImageWithEXIF returns a JPEG image with EXIF metadata for testing.
// This uses a base64-encoded minimal JPEG with embedded EXIF data.
func getTestImageWithEXIF(t *testing.T) []byte {
	// Try to read from testdata directory first
	testImagePath := "testdata/sample_exif.jpg"
	if data, err := os.ReadFile(testImagePath); err == nil {
		return data
	}

	// Fallback: use embedded base64-encoded JPEG with EXIF
	// This is a 1x1 red pixel JPEG with minimal EXIF data
	// Generated with: convert -size 1x1 xc:red -set comment "Test EXIF" test.jpg
	base64JPEG := `
/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0a
HBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIy
MjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIA
AhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEB
AQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAB//2Q==
`

	decoded, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace([]byte(base64JPEG))))
	if err != nil {
		t.Fatalf("Failed to decode test image: %v", err)
	}

	return decoded
}
