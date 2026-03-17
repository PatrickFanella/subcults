# Feed Aggregation API Reference

## Overview

Feed aggregation endpoints provide paginated access to posts associated with scenes and events. These endpoints support cursor-based pagination for stable, efficient traversal of large post collections.

## Endpoints

### Scene Feed

**GET** `/scenes/{id}/feed`

Retrieves a paginated list of posts for a specific scene.

#### Path Parameters

| Parameter | Type   | Required | Description           |
|-----------|--------|----------|-----------------------|
| `id`      | string | Yes      | Scene UUID            |

#### Query Parameters

| Parameter | Type    | Required | Default | Description                                    |
|-----------|---------|----------|---------|------------------------------------------------|
| `limit`   | integer | No       | 20      | Number of posts to return (max: 100)          |
| `cursor`  | string  | No       | -       | Pagination cursor from previous response       |

#### Response

**Status**: `200 OK`

```json
{
  "posts": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "scene_id": "scene-uuid",
      "event_id": null,
      "author_did": "did:example:alice",
      "text": "Check out this new track!",
      "attachments": [
        {
          "url": "https://storage.example.com/media/track.mp3",
          "type": "audio/mpeg"
        }
      ],
      "labels": [],
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "next_cursor": {
    "created_at": "2024-01-15T09:15:00Z",
    "id": "660e8400-e29b-41d4-a716-446655440001"
  }
}
```

#### Response Fields

| Field         | Type            | Description                                    |
|---------------|-----------------|------------------------------------------------|
| `posts`       | array           | Array of post objects                          |
| `next_cursor` | object \| null  | Cursor for next page, `null` if no more pages  |

#### Post Object

| Field          | Type       | Description                              |
|----------------|------------|------------------------------------------|
| `id`           | string     | Post UUID                                |
| `scene_id`     | string     | Scene UUID (nullable)                    |
| `event_id`     | string     | Event UUID (nullable)                    |
| `author_did`   | string     | Author's decentralized identifier        |
| `text`         | string     | Post content                             |
| `attachments`  | array      | Array of attachment objects              |
| `labels`       | array      | Moderation labels (e.g., `["nsfw"]`)     |
| `created_at`   | string     | ISO 8601 timestamp                       |
| `updated_at`   | string     | ISO 8601 timestamp                       |

#### Error Responses

**400 Bad Request** - Invalid parameters
```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid limit parameter"
  }
}
```

**404 Not Found** - Scene does not exist
```json
{
  "error": {
    "code": "not_found",
    "message": "Scene not found"
  }
}
```

**500 Internal Server Error** - Server error
```json
{
  "error": {
    "code": "internal_error",
    "message": "Failed to retrieve posts"
  }
}
```

#### Filtering Behavior

The feed endpoint automatically filters posts to:
- **Exclude soft-deleted posts**: Posts with `deleted_at != NULL` are never returned
- **Exclude hidden posts**: Posts with the `"hidden"` moderation label are excluded from all public feeds
- **Order by recency**: Posts are returned in reverse chronological order (`created_at DESC`)

#### Pagination

This endpoint uses **cursor-based pagination** for stable, efficient traversal:

1. **First Request**: Omit the `cursor` parameter to get the most recent posts
2. **Subsequent Requests**: Pass the `next_cursor` value from the previous response
3. **End of Feed**: When `next_cursor` is `null`, there are no more posts

**Example Pagination Flow**:

```typescript
// First page
const page1 = await fetch('/scenes/scene-123/feed?limit=20');
const data1 = await page1.json();

// Second page (if next_cursor exists)
if (data1.next_cursor) {
  // Convert ISO 8601 timestamp to Unix nanoseconds for cursor encoding
  const timestampNano = new Date(data1.next_cursor.created_at).getTime() * 1000000;
  const cursorStr = `${timestampNano}:${data1.next_cursor.id}`;
  const page2 = await fetch(`/scenes/scene-123/feed?limit=20&cursor=${encodeURIComponent(cursorStr)}`);
  const data2 = await page2.json();
}
```

**Cursor Format**: `{unix_timestamp_nano}:{post_id}`

Example: `1705315800000000000:550e8400-e29b-41d4-a716-446655440000`

**Note**: The API returns `next_cursor.created_at` as an ISO 8601 timestamp string (e.g., `"2024-01-15T10:30:00Z"`), but the cursor query parameter expects Unix nanoseconds. Use `new Date(created_at).getTime() * 1000000` to convert.

##### Cursor Structure and Invariants

**Cursor Composition**: Each cursor consists of two components:
1. **`created_at`** (timestamp): ISO 8601 timestamp of the last post returned on the current page
2. **`id`** (UUID string): Unique identifier of the last post returned on the current page

**Ordering Guarantee**: Posts are ordered by:
1. `created_at DESC` (newest first)
2. `id ASC` (lexicographic, for tie-breaking when timestamps are identical)

This composite ordering ensures stable, deterministic pagination even when multiple posts have the same creation timestamp.

**Pagination Invariants**:

1. **No Duplicates**: A post will never appear on multiple pages within the same pagination session (using the same sequence of cursors).

2. **Cursor Stability Under Mutations**:
   - **Deletions**: If a post is deleted after a cursor is captured, pagination continues without duplicates or unintended skips. The deleted post simply doesn't appear in subsequent pages.
   - **Label Changes (Hiding)**: If a post is hidden (via label change) after a cursor is captured, it's excluded from subsequent pages without affecting pagination integrity.
   - **Insertions**: New posts inserted with timestamps that would place them *before* the cursor in the `created_at DESC` feed (i.e., with `created_at > cursor.created_at`) will **not** appear in ongoing pagination. They only appear when refreshing from the beginning (no cursor). This prevents "page tearing" where new content disrupts the user's scroll position.

3. **Timestamp Boundaries**: The cursor marks a position in the chronologically-sorted feed. Subsequent pages return only posts with:
   - `created_at < cursor.created_at` (older posts), OR
   - `created_at == cursor.created_at AND id > cursor.id` (same timestamp but lexicographically greater ID)

4. **Idempotency**: Calling the same endpoint with the same cursor multiple times returns the same results (assuming no posts were deleted/hidden between calls).

5. **Session Independence**: Cursors from different pagination sessions can be used interchangeably as long as they reference valid posts. However, refreshing from the beginning (no cursor) may show different results if new posts were added.

**Expected Behaviors**:

- **New posts earlier in the feed order**: If a post is created after the cursor is captured but has a `created_at` timestamp greater than the cursor's timestamp (placing it earlier in the DESC-ordered feed), it will **not** appear in ongoing pagination. Users must refresh (start from the beginning with no cursor) to see it. This is intentional to prevent disrupting infinite scroll UX.

- **Consistent ordering**: The same set of posts (excluding deleted/hidden ones) will always appear in the same order, regardless of when pagination occurs.

- **Hidden posts stay hidden**: Posts with the `hidden` label are excluded from all feed responses, even if they were visible when the cursor was created.

**Testing**: Cursor integrity is validated through comprehensive test suites covering:
- Post deletion during pagination
- Post hiding during pagination  
- New post insertion during pagination
- Multiple concurrent mutations
- Event feed pagination (same semantics as scene feeds)

See `internal/post/feed_test.go` for detailed test scenarios.

---

### Event Feed

**GET** `/events/{id}/feed`

Retrieves a paginated list of posts for a specific event.

#### Path Parameters

| Parameter | Type   | Required | Description           |
|-----------|--------|----------|-----------------------|
| `id`      | string | Yes      | Event UUID            |

#### Query Parameters

Same as Scene Feed (see above).

#### Response

Same structure as Scene Feed (see above), but posts are filtered by `event_id` instead of `scene_id`.

#### Error Responses

Same as Scene Feed (see above).

---

## Usage Examples

### JavaScript/TypeScript

```typescript
import { apiClient } from '@/lib/api-client';

interface FeedResponse {
  posts: Post[];
  next_cursor: { created_at: string; id: string } | null;
}

// Fetch first page of scene feed
async function loadSceneFeed(sceneId: string, limit = 20) {
  return await apiClient.get<FeedResponse>(
    `/scenes/${sceneId}/feed?limit=${limit}`
  );
}

// Fetch next page using cursor
async function loadNextPage(sceneId: string, cursor: { created_at: string; id: string }) {
  // Convert ISO 8601 timestamp to Unix nanoseconds
  const timestampNano = new Date(cursor.created_at).getTime() * 1000000;
  const cursorStr = `${timestampNano}:${cursor.id}`;
  return await apiClient.get<FeedResponse>(
    `/scenes/${sceneId}/feed?limit=20&cursor=${encodeURIComponent(cursorStr)}`
  );
}

// Load all posts (careful with large feeds!)
async function loadAllPosts(sceneId: string): Promise<Post[]> {
  const allPosts: Post[] = [];
  let cursor: { created_at: string; id: string } | null = null;

  while (true) {
    const response = cursor 
      ? await loadNextPage(sceneId, cursor)
      : await loadSceneFeed(sceneId, 100); // Use max limit

    allPosts.push(...response.posts);

    if (!response.next_cursor) break;
    cursor = response.next_cursor;
  }

  return allPosts;
}
```

### cURL

```bash
# Get first page of scene feed
curl -X GET "http://localhost:8080/scenes/550e8400-e29b-41d4-a716-446655440000/feed?limit=20"

# Get second page with cursor
curl -X GET "http://localhost:8080/scenes/550e8400-e29b-41d4-a716-446655440000/feed?limit=20&cursor=1705315800000000000%3A550e8400-e29b-41d4-a716-446655440001"

# Get event feed
curl -X GET "http://localhost:8080/events/660e8400-e29b-41d4-a716-446655440002/feed?limit=50"
```

---

## Performance Considerations

### Database Indexes

The following indexes optimize feed queries:

```sql
-- Scene feed index
CREATE INDEX idx_posts_scene_feed 
    ON posts(scene_id, created_at DESC, id ASC) 
    WHERE deleted_at IS NULL AND scene_id IS NOT NULL;

-- Event feed index
CREATE INDEX idx_posts_event_feed 
    ON posts(event_id, created_at DESC, id ASC) 
    WHERE deleted_at IS NULL AND event_id IS NOT NULL;
```

### Best Practices

1. **Use appropriate limits**: Default is 20, max is 100. Higher limits may increase latency.
2. **Cache results**: Posts are immutable once created (except updates). Consider caching feed pages.
3. **Cursor stability**: Cursors remain valid even if new posts are added, ensuring stable pagination.
4. **Avoid loading all posts**: For large feeds, use infinite scroll or "Load More" patterns instead of fetching all posts at once.

---

## Security & Privacy

### Authentication

Feed endpoints do **not** currently require authentication. All scene and event feeds are publicly accessible.

Future versions may add:
- Authentication requirements for private scenes/events
- User-specific filtering (e.g., showing posts from blocked users)

### Privacy Compliance

- **No location data**: Post objects do not include precise geographic coordinates
- **Minimal metadata**: Only essential post data is returned (no IP addresses, device info, etc.)
- **Moderation labels**: Hidden posts are automatically excluded

### Rate Limiting

API rate limits apply to feed endpoints:
- **Anonymous users**: 100 requests per minute
- **Authenticated users**: 300 requests per minute

Exceeding the rate limit returns `429 Too Many Requests`.

---

## Migration Notes

Migration `000014_add_feed_indexes.up.sql` creates the necessary indexes:

```sql
-- Apply migration
./scripts/migrate.sh up

-- Verify indexes
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'posts' 
AND indexname LIKE '%feed%';
```

---

## Related Documentation

- [Post Management API](./API_REFERENCE.md#posts) - Creating and managing posts
- [Moderation Guidelines](./MODERATION.md) - Post moderation labels and filtering
- [Privacy Policy](./PRIVACY.md) - Privacy-first design principles
- [Architecture Overview](./ARCHITECTURE.md) - System architecture
