# Post CRUD Handlers

## Overview

The post handlers provide HTTP endpoints for creating, updating, and soft-deleting posts with XSS protection, content validation, and support for attachments and labels.

## Endpoints

### POST /posts

Creates a new post associated with a scene or event.

**Request Body:**
```json
{
  "scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "text": "Check out this amazing underground show tonight!",
  "attachments": [
    {
      "url": "https://example.com/flyer.jpg",
      "type": "image"
    }
  ],
  "labels": ["announcement", "live-music"]
}
```

**Alternative with event_id:**
```json
{
  "event_id": "660e8400-e29b-41d4-a716-446655440001",
  "text": "This event is going to be epic!",
  "labels": ["hype"]
}
```

**Response:** `201 Created`
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "author_did": "did:plc:abc123",
  "text": "Check out this amazing underground show tonight!",
  "attachments": [
    {
      "url": "https://example.com/flyer.jpg",
      "type": "image"
    }
  ],
  "labels": ["announcement", "live-music"],
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Validation:**
- At least one of `scene_id` or `event_id` must be provided
- `text`: Required, non-empty after trimming, maximum 5000 characters
- `attachments`: Optional, maximum 6 items
- `labels`: Optional, sanitized to prevent XSS
- All text fields are sanitized using HTML escaping to prevent XSS attacks

**Error Responses:**
- `400 Bad Request` with code `missing_target` - Neither scene_id nor event_id provided
- `400 Bad Request` with code `validation_error` - Text validation failure
- `400 Bad Request` with code `validation_error` - Too many attachments (>6)

### PATCH /posts/{id}

Updates an existing post. Only provided fields are updated.

**Request Body:**
```json
{
  "text": "Updated post content",
  "attachments": [
    {
      "url": "https://example.com/updated.jpg",
      "type": "image"
    }
  ],
  "labels": ["updated", "moderated"]
}
```

**Response:** `200 OK`
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "author_did": "did:plc:abc123",
  "text": "Updated post content",
  "attachments": [
    {
      "url": "https://example.com/updated.jpg",
      "type": "image"
    }
  ],
  "labels": ["updated", "moderated"],
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

**Validation:**
- Same validation rules as create for provided fields
- `text`: If provided, must be non-empty and ≤5000 characters
- `attachments`: If provided, maximum 6 items
- Cannot update soft-deleted posts

**Error Responses:**
- `400 Bad Request` - Invalid JSON or validation failure
- `404 Not Found` - Post not found or already deleted

### DELETE /posts/{id}

Soft-deletes a post by setting the `deleted_at` timestamp.

**Response:** `204 No Content`

**Behavior:**
- Sets `deleted_at` timestamp on the post
- Soft-deleted posts:
  - Return 404 on direct fetch via GetByID
  - Cannot be updated
  - Should be excluded from feeds and searches
- Idempotent: deleting an already-deleted post returns 404

**Error Responses:**
- `404 Not Found` - Post not found or already deleted

## Security & Privacy

### XSS Prevention

All text fields (text, labels) are sanitized using `html.EscapeString` to prevent XSS attacks:

**Input:**
```json
{
  "text": "<script>alert('xss')</script>Hello",
  "labels": ["<b>bold</b>"]
}
```

**Stored:**
```json
{
  "text": "&lt;script&gt;alert('xss')&lt;/script&gt;Hello",
  "labels": ["&lt;b&gt;bold&lt;/b&gt;"]
}
```

### Content Limits

- **Text**: 5000 characters maximum
- **Attachments**: 6 items maximum
- **Labels**: No explicit limit, but each label is sanitized

### Soft Delete

Deleted posts are not physically removed from the database. Instead:
1. `deleted_at` timestamp is set
2. Repository's `GetByID` excludes deleted posts
3. Updates to deleted posts are rejected
4. Feeds and searches should filter out deleted posts using `WHERE deleted_at IS NULL`

## Database Schema

Posts table includes:
- `id` (UUID, primary key)
- `scene_id` (UUID, nullable, foreign key to scenes)
- `event_id` (UUID, nullable, foreign key to events)
- `author_did` (TEXT, NOT NULL)
- `text` (TEXT, NOT NULL)
- `attachments` (JSONB, default '[]')
- `labels` (TEXT[], default '{}')
- `deleted_at` (TIMESTAMPTZ, nullable)
- `created_at` (TIMESTAMPTZ, NOT NULL)
- `updated_at` (TIMESTAMPTZ, NOT NULL)

**Constraints:**
- `chk_post_association`: At least one of scene_id or event_id must be non-null

**Indexes:**
- `idx_posts_author` on author_did WHERE deleted_at IS NULL
- `idx_posts_scene` on scene_id WHERE deleted_at IS NULL AND scene_id IS NOT NULL
- `idx_posts_event` on event_id WHERE deleted_at IS NULL AND event_id IS NOT NULL
- `idx_posts_labels` (GIN) on labels for moderation filtering
- `idx_posts_created` on created_at DESC WHERE deleted_at IS NULL

## Example Usage

### Creating a Scene Post

```bash
curl -X POST http://localhost:8080/posts \
  -H "Content-Type: application/json" \
  -d '{
    "scene_id": "550e8400-e29b-41d4-a716-446655440000",
    "text": "New show announcement!",
    "labels": ["announcement"]
  }'
```

### Creating an Event Post

```bash
curl -X POST http://localhost:8080/posts \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "660e8400-e29b-41d4-a716-446655440001",
    "text": "See you tonight!",
    "attachments": [
      {"url": "https://example.com/map.jpg", "type": "image"}
    ]
  }'
```

### Updating a Post

```bash
curl -X PATCH http://localhost:8080/posts/770e8400-e29b-41d4-a716-446655440002 \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Updated: Show cancelled due to weather",
    "labels": ["cancelled"]
  }'
```

### Deleting a Post

```bash
curl -X DELETE http://localhost:8080/posts/770e8400-e29b-41d4-a716-446655440002
```

## Future Enhancements

- **Authentication**: Integrate with JWT middleware to enforce author authorization
- **Moderation**: Implement label-based filtering for content moderation workflows
- **Attachments**: Add signed URL validation for attachment uploads
- **Full-text Search**: Enable GIN indexing on text field for FTS queries
- **Feed APIs**: Implement paginated feed endpoints filtered by scene/event/author
