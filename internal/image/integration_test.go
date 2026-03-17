package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/h2non/bimg"
)

// TestProcess_RealImage_WithEXIF creates a real image and verifies EXIF stripping.
// This test creates an actual image buffer and processes it through bimg.
func TestProcess_RealImage_WithEXIF(t *testing.T) {
	// Create a simple 100x100 test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Fill with a gradient pattern
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := color.RGBA{
				R: uint8(x * 255 / 100),
				G: uint8(y * 255 / 100),
				B: 128,
				A: 255,
			}
			img.Set(x, y, c)
		}
	}

	// Encode to JPEG buffer
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	originalBytes := buf.Bytes()
	t.Logf("Original JPEG size: %d bytes", len(originalBytes))

	// Process the image to strip EXIF (even though this image has minimal metadata)
	processedBytes, err := ProcessBytes(originalBytes)
	if err != nil {
		t.Fatalf("ProcessBytes failed: %v", err)
	}

	if len(processedBytes) == 0 {
		t.Fatal("Processed image is empty")
	}

	t.Logf("Processed JPEG size: %d bytes", len(processedBytes))

	// Verify the processed image has no EXIF
	noEXIF, err := VerifyNoEXIF(processedBytes)
	if err != nil {
		t.Fatalf("VerifyNoEXIF failed: %v", err)
	}

	if !noEXIF {
		t.Error("EXIF metadata found in processed image")
	}

	// Verify we can still read the processed image
	processedImg := bimg.NewImage(processedBytes)
	metadata, err := processedImg.Metadata()
	if err != nil {
		t.Fatalf("Failed to read processed image metadata: %v", err)
	}

	// Verify dimensions are preserved
	if metadata.Size.Width != 100 || metadata.Size.Height != 100 {
		t.Errorf("Image dimensions changed: expected 100x100, got %dx%d",
			metadata.Size.Width, metadata.Size.Height)
	}

	t.Logf("Processed image metadata: %dx%d, type=%s, channels=%d",
		metadata.Size.Width, metadata.Size.Height, metadata.Type, metadata.Channels)
}

// TestProcess_WebP_Conversion tests converting JPEG to WebP format.
func TestProcess_WebP_Conversion(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255}) // Red square
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	originalBytes := buf.Bytes()
	originalSize := len(originalBytes)

	// Convert to WebP
	config := DefaultConfig()
	config.OutputFormat = "webp"
	config.Quality = 80

	processedBytes, err := ProcessWithConfig(bytes.NewReader(originalBytes), config)
	if err != nil {
		t.Fatalf("ProcessWithConfig failed: %v", err)
	}

	processedSize := len(processedBytes)

	// WebP should typically be more efficient than JPEG for most images
	t.Logf("JPEG size: %d bytes, WebP size: %d bytes (%.1f%% of original)",
		originalSize, processedSize, float64(processedSize)/float64(originalSize)*100)

	// Verify the output is valid WebP
	processedImg := bimg.NewImage(processedBytes)
	metadata, err := processedImg.Metadata()
	if err != nil {
		t.Fatalf("Failed to read processed image metadata: %v", err)
	}

	if metadata.Type != "webp" {
		t.Errorf("Expected type webp, got %s", metadata.Type)
	}

	// Verify no EXIF
	noEXIF, err := VerifyNoEXIF(processedBytes)
	if err != nil {
		t.Fatalf("VerifyNoEXIF failed: %v", err)
	}

	if !noEXIF {
		t.Error("EXIF metadata found in WebP output")
	}
}

// TestProcess_PreserveQuality tests that quality settings work correctly.
func TestProcess_PreserveQuality(t *testing.T) {
	// Create a detailed test image with gradients
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			c := color.RGBA{
				R: uint8((x + y) * 255 / 400),
				G: uint8((200 - x) * 255 / 200),
				B: uint8((200 - y) * 255 / 200),
				A: 255,
			}
			img.Set(x, y, c)
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	originalBytes := buf.Bytes()

	// Test different quality levels
	qualities := []int{95, 85, 70, 50}
	var prevSize int

	for _, quality := range qualities {
		config := DefaultConfig()
		config.Quality = quality

		processedBytes, err := ProcessWithConfig(bytes.NewReader(originalBytes), config)
		if err != nil {
			t.Fatalf("ProcessWithConfig with quality %d failed: %v", quality, err)
		}

		size := len(processedBytes)
		t.Logf("Quality %d: %d bytes", quality, size)

		// Generally, lower quality should result in smaller files
		// (though not guaranteed for all images)
		if prevSize > 0 && size > prevSize*2 {
			t.Logf("Warning: Quality %d produced larger file than previous quality", quality)
		}
		prevSize = size

		// Verify no EXIF regardless of quality
		noEXIF, err := VerifyNoEXIF(processedBytes)
		if err != nil {
			t.Fatalf("VerifyNoEXIF failed for quality %d: %v", quality, err)
		}

		if !noEXIF {
			t.Errorf("EXIF metadata found with quality %d", quality)
		}
	}
}

// BenchmarkProcess benchmarks the image processing performance.
func BenchmarkProcess(b *testing.B) {
	// Create a realistic test image
	img := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	for y := 0; y < 768; y++ {
		for x := 0; x < 1024; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / 1024),
				G: uint8((y * 255) / 768),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		b.Fatalf("Failed to encode test image: %v", err)
	}

	imageBytes := buf.Bytes()
	b.Logf("Test image size: %d bytes", len(imageBytes))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ProcessBytes(imageBytes)
		if err != nil {
			b.Fatalf("ProcessBytes failed: %v", err)
		}
	}
}

// BenchmarkProcess_WebP benchmarks WebP conversion performance.
func BenchmarkProcess_WebP(b *testing.B) {
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	for y := 0; y < 768; y++ {
		for x := 0; x < 1024; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / 1024),
				G: uint8((y * 255) / 768),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		b.Fatalf("Failed to encode test image: %v", err)
	}

	imageBytes := buf.Bytes()
	config := DefaultConfig()
	config.OutputFormat = "webp"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ProcessWithConfig(bytes.NewReader(imageBytes), config)
		if err != nil {
			b.Fatalf("ProcessWithConfig failed: %v", err)
		}
	}
}
