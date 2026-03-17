# Attachment Metadata Package

This package provides services for enriching media attachment metadata while ensuring privacy through EXIF stripping.

## Overview

The `attachment` package handles the extraction and enrichment of attachment metadata for posts. When a user uploads a file to R2 and submits the attachment key with a post, this service:

1. **Fetches basic metadata** using S3 HeadObject (content type, size)
2. **Processes images** to strip EXIF data (GPS, camera info, timestamps)
3. **Extracts dimensions** for images (width, height)
4. **Re-uploads sanitized images** to R2, replacing the original
5. **Returns enriched attachment** with metadata

## Privacy & Security

This package is critical for user privacy and implements the following safeguards:

### EXIF Stripping

All image attachments are processed through the `internal/image` package which:
- Removes GPS coordinates
- Strips camera make/model
- Deletes original timestamps
- Removes software metadata
- Preserves image quality and orientation

### No Raw EXIF Persistence

Images are:
1. Downloaded from R2
2. Processed to strip EXIF
3. Re-uploaded to replace the original
4. Only sanitized versions are stored

### Metadata Extraction

Only safe metadata is extracted and stored:
- **Images**: width, height, size, content type
- **Audio**: duration (placeholder), size, content type
- **No PII** is extracted or stored

## Usage

### Initialization

The metadata service requires an S3 client (configured for Cloudflare R2):

```go
import (
    "github.com/onnwee/subcults/internal/attachment"
    "github.com/onnwee/subcults/internal/upload"
)

// Create upload service first
uploadService, err := upload.NewService(upload.ServiceConfig{
    BucketName:      "my-bucket",
    AccessKeyID:     "key",
    SecretAccessKey: "secret",
    Endpoint:        "https://account.r2.cloudflarestorage.com",
})

// Create metadata service using upload service's S3 client
metadataService, err := attachment.NewMetadataService(
    attachment.MetadataServiceConfig{
        S3Client:   uploadService.GetS3Client(),
        BucketName: uploadService.GetBucketName(),
    },
)
```

### Enriching Attachments

```go
// After user uploads to R2 via signed URL
attachment, err := metadataService.EnrichAttachment(ctx, "posts/uuid/image.jpg")
if err != nil {
    // Handle error (object not found, invalid format, etc.)
}

// attachment now has:
// - Key: "posts/uuid/image.jpg"
// - Type: "image/jpeg"
// - SizeBytes: 1024000
// - Width: 1920 (pointer)
// - Height: 1080 (pointer)
// And the image in R2 no longer has EXIF data
```

## Integration with Post Handlers

The attachment metadata service is integrated into post creation:

```go
// In CreatePost handler
for _, att := range req.Attachments {
    enriched, err := h.metadataService.EnrichAttachment(ctx, att.Key)
    if err != nil {
        // Log warning but don't fail request
        // Use client-provided attachment as fallback
        continue
    }
    enrichedAttachments = append(enrichedAttachments, *enriched)
}
```

## Attachment Structure

Attachments support both legacy URL-based and new key-based formats:

```go
type Attachment struct {
    // Legacy
    URL string `json:"url,omitempty"`
    
    // New enriched format
    Key       string   `json:"key,omitempty"`
    Type      string   `json:"type,omitempty"`
    SizeBytes int64    `json:"size_bytes,omitempty"`
    Width     *int     `json:"width,omitempty"`     // Images only
    Height    *int     `json:"height,omitempty"`    // Images only
    DurationSeconds *float64 `json:"duration_seconds,omitempty"` // Audio only
}
```

## Performance Considerations

### Blocking Operations

The `EnrichAttachment` method performs the following operations synchronously:
1. HeadObject (~100ms)
2. GetObject (depends on file size)
3. EXIF stripping + dimension extraction (~200ms for typical image)
4. PutObject (depends on file size)

**Total**: ~500ms-2s per image depending on size and network latency

### Non-Blocking Error Handling

Post creation does NOT fail if attachment enrichment fails:
- Errors are logged
- Client-provided attachment data is used as fallback
- Request completes successfully

This ensures attachment enrichment doesn't block post creation.

### Optimization Opportunities

Future optimizations:
- [ ] Async processing with post-creation enrichment
- [ ] Caching dimension data in metadata (e.g., R2 object metadata)
- [ ] Batch processing for multiple attachments
- [ ] Skip re-upload if EXIF already stripped

## Error Handling

| Error | Cause | Handler Behavior |
|-------|-------|------------------|
| `ErrInvalidObjectKey` | Empty key | Log warning, use fallback |
| `ErrObjectNotFound` | Key doesn't exist in R2 | Log warning, use fallback |
| Image processing error | Invalid image data | Log warning, use fallback |
| S3 PutObject error | Upload failure | Log warning, use fallback |

All errors result in warnings logged but do not fail the request.

## Testing

### Unit Tests

```bash
go test ./internal/attachment/... -v
```

Tests cover:
- Service initialization
- Content type detection (image/audio)
- Invalid key handling
- Graceful error handling

### Integration Tests

Integration tests require:
- R2 credentials configured
- Test bucket available
- libvips installed

See `internal/image/integration_test.go` for EXIF stripping verification.

## Dependencies

- **AWS SDK v2**: S3 client for R2 interaction
- **bimg/libvips**: Image processing and EXIF stripping
- **internal/image**: EXIF stripping service

## Configuration

The service is automatically initialized when R2 credentials are configured:

```bash
# Required environment variables
R2_BUCKET_NAME=my-bucket
R2_ACCESS_KEY_ID=your-access-key
R2_SECRET_ACCESS_KEY=your-secret-key
R2_ENDPOINT=https://account-id.r2.cloudflarestorage.com
```

If R2 is not configured, the metadata service is `nil` and attachments are used as-is.

## Future Enhancements

- [ ] Audio duration extraction (requires audio codec library)
- [ ] Video thumbnail generation
- [ ] Async post-processing pipeline
- [ ] CDN integration for optimized delivery
- [ ] Image compression/resizing options
- [ ] Support for additional formats (HEIF, AVIF)

## Related Documentation

- [Image Processing Package](../image/README.md)
- [Privacy Documentation](../../docs/PRIVACY.md)
- [Upload Service](../upload/README.md)
- [Post API Handlers](../api/post_handlers.go)
