# Attachment API Documentation

## POST /posts - Create Post with Attachments

Posts can include media attachments with enriched metadata.

### Request Body

```json
{
  "scene_id": "scene-uuid",
  "text": "Check out this photo!",
  "attachments": [
    {
      "key": "posts/temp/abc123-def456.jpg"
    }
  ]
}
```

### Attachment Upload Flow

1. **Client requests signed URL**:
   ```http
   POST /uploads/sign
   {
     "content_type": "image/jpeg",
     "size_bytes": 1024000,
     "post_id": null
   }
   ```

2. **Server returns signed URL**:
   ```json
   {
     "url": "https://bucket.r2.cloudflarestorage.com/...",
     "key": "posts/temp/abc123-def456.jpg",
     "expires_at": "2024-01-01T00:05:00Z"
   }
   ```

3. **Client uploads directly to R2** using the signed URL

4. **Client submits post with attachment key**:
   ```http
   POST /posts
   {
     "scene_id": "scene-123",
     "text": "My post",
     "attachments": [{ "key": "posts/temp/abc123-def456.jpg" }]
   }
   ```

5. **Server enriches attachment** (strips EXIF, extracts metadata)

6. **Server returns enriched post**:
   ```json
   {
     "id": "post-uuid",
     "scene_id": "scene-123",
     "text": "My post",
     "attachments": [
       {
         "key": "posts/temp/abc123-def456.jpg",
         "type": "image/jpeg",
         "size_bytes": 1020000,
         "width": 1920,
         "height": 1080
       }
     ],
     "created_at": "2024-01-01T00:00:00Z"
   }
   ```

### Attachment Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Yes* | R2 object key (from signed URL response) |
| `type` | string | No | MIME type (auto-detected from R2) |
| `size_bytes` | integer | No | File size in bytes (auto-detected) |
| `width` | integer | No | Image width in pixels (images only) |
| `height` | integer | No | Image height in pixels (images only) |
| `duration_seconds` | float | No | Audio duration in seconds (audio only, future) |
| `url` | string | Yes* | Legacy: direct URL to attachment |

\* Either `key` (new format) or `url` (legacy format) is required

### Privacy Guarantees

**All image attachments are automatically sanitized**:
- ✅ GPS coordinates removed
- ✅ Camera make/model removed
- ✅ Timestamps removed
- ✅ Software metadata removed
- ✅ Original orientation preserved
- ✅ Image quality maintained

The sanitization happens server-side during post creation. The original uploaded file is replaced with the sanitized version.

### Supported Content Types

**Images**:
- `image/jpeg`
- `image/png`

**Audio**:
- `audio/mpeg`
- `audio/wav`

Maximum file size: 15MB (configurable via `R2_MAX_UPLOAD_SIZE_MB`)
Maximum attachments per post: 6

### Error Handling

If attachment enrichment fails:
- Post is still created successfully
- Attachment is stored with client-provided data
- Error is logged server-side
- Client receives 201 Created with basic attachment info

This ensures attachment processing doesn't block post creation.

### Example: Multiple Attachments

```json
{
  "scene_id": "scene-123",
  "text": "Photo dump from the show!",
  "attachments": [
    { "key": "posts/post-abc/image1.jpg" },
    { "key": "posts/post-abc/image2.jpg" },
    { "key": "posts/post-abc/image3.png" }
  ]
}
```

Response includes enriched metadata for each:

```json
{
  "id": "post-uuid",
  "text": "Photo dump from the show!",
  "attachments": [
    {
      "key": "posts/post-abc/image1.jpg",
      "type": "image/jpeg",
      "size_bytes": 1020000,
      "width": 1920,
      "height": 1080
    },
    {
      "key": "posts/post-abc/image2.jpg",
      "type": "image/jpeg",
      "size_bytes": 850000,
      "width": 1280,
      "height": 720
    },
    {
      "key": "posts/post-abc/image3.png",
      "type": "image/png",
      "size_bytes": 2048000,
      "width": 2560,
      "height": 1440
    }
  ],
  "created_at": "2024-01-01T00:00:00Z"
}
```
