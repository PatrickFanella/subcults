# Image Processing Package

This package provides image sanitization and re-encoding functionality to strip EXIF metadata and prevent privacy leakage.

## Overview

The `image` package uses [bimg](https://github.com/h2non/bimg) (libvips binding) to:

1. Strip all EXIF metadata (GPS coordinates, camera details, timestamps)
2. Re-encode images to JPEG, WebP, or PNG
3. Apply orientation correction based on EXIF data before stripping
4. Optionally resize images with quality controls

## Privacy & Safety

This package is a core component of Subcult's **Privacy & Safety Epic #6**. It prevents passive data leakage by removing:

- **GPS coordinates** embedded in photos
- **Device identifiers** (camera make, model, serial numbers)
- **Timestamps** (original capture time, modification time)
- **Camera metadata** (exposure, ISO, aperture, etc.)
- **Software information** (editing software, versions)

## Usage

### Basic Usage

```go
import "github.com/onnwee/subcults/internal/image"

// Process with defaults (JPEG, quality 85, strip metadata)
sanitizedBytes, err := image.Process(fileReader)
if err != nil {
    log.Fatal(err)
}

// Verify EXIF was removed
noEXIF, err := image.VerifyNoEXIF(sanitizedBytes)
if err != nil {
    log.Fatal(err)
}
if !noEXIF {
    log.Warn("EXIF data still present!")
}
```

### Custom Configuration

```go
config := image.ProcessorConfig{
    Quality:       90,
    OutputFormat:  "webp",
    StripMetadata: true,
    MaxWidth:      2048,
    MaxHeight:     2048,
}

sanitizedBytes, err := image.ProcessWithConfig(fileReader, config)
```

### Processing Bytes Directly

```go
sanitizedBytes, err := image.ProcessBytes(inputBytes)
```

## Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Quality` | int | 85 | JPEG/WebP quality (1-100) |
| `OutputFormat` | string | "jpeg" | Output format: jpeg, webp, png |
| `StripMetadata` | bool | true | Remove all EXIF/metadata |
| `MaxWidth` | int | 0 | Max width in pixels (0 = no limit) |
| `MaxHeight` | int | 0 | Max height in pixels (0 = no limit) |

## Implementation Details

### EXIF Stripping

The package uses bimg's `StripMetadata` option which:
- Removes all EXIF tags including GPS, camera, and software metadata
- Preserves image quality and dimensions
- Applies EXIF orientation correction before stripping (ensures correct display)

### Orientation Handling

Images with EXIF orientation tags are automatically rotated to match their intended display orientation before the EXIF data is removed. This prevents images from appearing rotated incorrectly after metadata stripping.

### Output Size

Processed images typically have:
- **Smaller file size** due to metadata removal (varies by original EXIF size)
- **Comparable quality** at default settings (quality 85)
- **Optional compression** via quality settings or format conversion (WebP)

## Testing

Run the test suite:

```bash
go test -v ./internal/image/...
```

Tests use a sample JPEG image with embedded EXIF data to verify:
- Metadata is stripped
- GPS coordinates are removed
- Image dimensions are preserved
- Quality is acceptable
- Orientation is corrected

## Dependencies

- **[bimg](https://github.com/h2non/bimg)**: Go binding for libvips (fast image processing)
- **libvips**: Required system library (must be installed on host)

### Installing libvips

#### Ubuntu/Debian
```bash
sudo apt-get install libvips-dev
```

#### macOS
```bash
brew install vips
```

#### Docker
```dockerfile
RUN apt-get update && apt-get install -y libvips-dev
```

## Security Considerations

1. **Input Validation**: Always validate file types before processing
2. **Size Limits**: Set `MaxWidth`/`MaxHeight` to prevent memory exhaustion
3. **Error Handling**: Handle processing errors gracefully (malformed images)
4. **Content-Type Verification**: Validate MIME types match actual image data

## Future Enhancements

- [ ] Support for additional formats (HEIF, AVIF)
- [ ] Batch processing API
- [ ] Streaming processing for large files
- [ ] Metadata allowlist (preserve specific safe tags)
- [ ] Image fingerprinting for duplicate detection

## References

- **Epic**: [Privacy & Safety #6](https://github.com/subculture-collective/subcults/issues/6)
- **Issue**: Image EXIF Stripping Service
- **Documentation**: [PRIVACY.md](../../docs/PRIVACY.md)
