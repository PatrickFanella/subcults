# Upload API Endpoint

## POST /uploads/sign

Generates a pre-signed URL for direct upload to Cloudflare R2 storage. This endpoint supports secure client-side uploads of media attachments (images and audio) without proxying large files through the API server.

### Authentication

**Required.** Clients must provide a valid access token in the `Authorization: Bearer <token>` header. The signed upload URL is scoped to the authenticated user and subject to standard rate limiting and authorization checks.

> **Note:** In the current implementation, authentication will be enforced once the authentication middleware is integrated into the endpoint handler.

### Request

**Method:** `POST`  
**Endpoint:** `/uploads/sign`  
**Content-Type:** `application/json`

#### Request Body

```json
{
  "contentType": "image/jpeg",
  "sizeBytes": 1048576,
  "postId": "optional-post-id"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `contentType` | string | Yes | MIME type of the file. Allowed values: `image/jpeg`, `image/png`, `audio/mpeg`, `audio/wav` |
| `sizeBytes` | number | Yes | Size of the file in bytes. Maximum: 15MB (15728640 bytes) by default |
| `postId` | string | No | Optional post ID to associate with the upload. If omitted, file is stored under `posts/temp/` |

### Response

#### Success (200 OK)

```json
{
  "url": "https://account-id.r2.cloudflarestorage.com/bucket/posts/temp/uuid.jpg?signature=...",
  "key": "posts/temp/uuid.jpg",
  "expiresAt": "2024-01-01T00:05:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Pre-signed PUT URL for uploading the file. Valid for 5 minutes |
| `key` | string | Object key in R2 storage. Use this key to reference the uploaded file |
| `expiresAt` | string | ISO 8601 timestamp when the signed URL expires |

#### Error Responses

##### 400 Bad Request - Invalid JSON

```json
{
  "error": {
    "code": "bad_request",
    "message": "Invalid JSON in request body"
  }
}
```

##### 400 Bad Request - Missing Content Type

```json
{
  "error": {
    "code": "validation_error",
    "message": "contentType is required"
  }
}
```

##### 400 Bad Request - Invalid Size

```json
{
  "error": {
    "code": "validation_error",
    "message": "sizeBytes must be positive"
  }
}
```

##### 400 Bad Request - Unsupported Type

```json
{
  "error": {
    "code": "unsupported_type",
    "message": "Unsupported content type. Allowed types: image/jpeg, image/png, audio/mpeg, audio/wav"
  }
}
```

##### 400 Bad Request - File Too Large

```json
{
  "error": {
    "code": "validation_error",
    "message": "File size exceeds maximum allowed"
  }
}
```

##### 500 Internal Server Error

```json
{
  "error": {
    "code": "internal_error",
    "message": "Failed to generate signed URL"
  }
}
```

### Usage Example

#### 1. Request Signed URL

```typescript
const fileToUpload = fileInput.files[0];

const signRequest = {
  contentType: fileToUpload.type,
  sizeBytes: fileToUpload.size,
  postId: "my-post-123" // optional
};

const signResponse = await fetch('/uploads/sign', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(signRequest)
});

const { url, key, expiresAt } = await signResponse.json();
```

#### 2. Upload File to R2

```typescript
const uploadResponse = await fetch(url, {
  method: 'PUT',
  headers: {
    'Content-Type': fileToUpload.type,
    'Content-Length': fileToUpload.size.toString()
  },
  body: fileToUpload
});

if (uploadResponse.ok) {
  console.log('Upload successful!');
  console.log('File key:', key);
  // Now you can reference this file using the 'key' in your post
}
```

#### 3. Include in Post

When creating or updating a post, reference the uploaded file using its key:

```typescript
const post = await fetch('/posts', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    scene_id: "scene-123",
    text: "Check out this photo!",
    attachments: [
      {
        type: "image",
        url: key, // Use the key from the upload response
        mime_type: "image/jpeg",
        size_bytes: fileToUpload.size
      }
    ]
  })
});
```

### Upload Workflow

```
┌──────────────┐
│   Client     │
└──────┬───────┘
       │ 1. Request signed URL
       │    POST /uploads/sign
       ▼
┌──────────────────┐
│   API Server     │
└──────┬───────────┘
       │ 2. Generate signed URL
       │    (valid for 5 minutes)
       ▼
┌──────────────────┐
│   Client         │
└──────┬───────────┘
       │ 3. Direct upload to R2
       │    PUT <signed-url>
       ▼
┌──────────────────┐
│   Cloudflare R2  │
└──────────────────┘
       │ 4. Upload complete
       ▼
┌──────────────────┐
│   Client         │
└──────┬───────────┘
       │ 5. Create/update post
       │    POST /posts (with key)
       ▼
┌──────────────────┐
│   API Server     │
└──────────────────┘
```

### Object Key Format

Uploaded files are stored with the following key format:

- **With postId**: `posts/{sanitized-postId}/{uuid}.{ext}`
- **Without postId**: `posts/temp/{uuid}.{ext}`

The UUID is automatically generated to ensure uniqueness. The file extension is derived from the content type:

| Content Type | Extension |
|-------------|-----------|
| image/jpeg | .jpg |
| image/png | .png |
| audio/mpeg | .mp3 |
| audio/wav | .wav |

### Security Considerations

1. **Size Limits**: The endpoint enforces a maximum file size (default 15MB) to prevent resource exhaustion
2. **Content Type Validation**: Only allowed MIME types are accepted to reduce attack surface
3. **Time-Limited URLs**: Pre-signed URLs expire after 5 minutes to limit exposure window
4. **Path Sanitization**: Post IDs are sanitized to prevent path traversal attacks
5. **Single Use**: While URLs can technically be reused within the 5-minute window, they should be treated as single-use

### Configuration

The upload service is configured via environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `R2_BUCKET_NAME` | R2 bucket name | - | No* |
| `R2_ACCESS_KEY_ID` | R2 access key ID | - | No* |
| `R2_SECRET_ACCESS_KEY` | R2 secret access key | - | No* |
| `R2_ENDPOINT` | R2 S3-compatible endpoint URL | - | No* |
| `R2_MAX_UPLOAD_SIZE_MB` | Maximum upload size in MB | 15 | No |

\* If any of the R2 credential variables (bucket name, access key ID, secret access key, or endpoint) are not configured, the API will log a warning and the `/uploads/sign` endpoint will not be available. All other API functionality continues to operate normally.

### Performance Characteristics

- **Signed URL Generation**: < 10ms (in-memory operation)
- **URL Expiry**: 5 minutes (configurable)
- **Direct Upload**: No API bandwidth consumed (client → R2 direct)
- **Throughput**: Limited by R2 capabilities, not API server

### Troubleshooting

#### URL Expired

If you receive a 403 Forbidden error when uploading to R2, the signed URL may have expired. Request a new signed URL.

#### Upload Failed

If the upload to R2 fails:
1. Check that you're using the correct Content-Type header
2. Verify the Content-Length matches the file size
3. Ensure you haven't modified the signed URL
4. Confirm the file size hasn't changed since requesting the URL

#### Invalid Key Reference

If your post creation fails with an invalid attachment reference:
1. Verify you're using the exact `key` value returned from the sign endpoint
2. Ensure the file was successfully uploaded to R2 before creating the post
3. Check that the key format matches the expected pattern
