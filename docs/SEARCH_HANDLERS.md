# Search Handlers

The search handlers package provides HTTP endpoints for searching scenes, events, and posts with privacy-aware location handling and optional trust-weighted ranking.

## Endpoints

### Post Search

**Endpoint:** `GET /search/posts`

**Description:** Search for posts by text query with optional scene filter. Results are ranked by text relevance.

**Query Parameters:**

| Parameter | Type   | Required | Default | Max | Description |
|-----------|--------|----------|---------|-----|-------------|
| `q`       | string | Yes      | -       | -   | Text search query (cannot be empty) |
| `scene_id`| string | No       | -       | -   | Filter results to a specific scene |
| `cursor`  | string | No       | -       | -   | Pagination cursor (format: "score:id") |
| `limit`   | int    | No       | 20      | 50  | Number of results per page |

**Response Format:**

```json
{
  "results": [
    {
      "id": "uuid",
      "excerpt": "First 160 characters of post text...",
      "scene_id": "scene-uuid",
      "created_at": "2024-01-24T12:00:00Z"
    }
  ],
  "next_cursor": "0.875000:post-uuid",
  "count": 20
}
```

**Response Fields:**

- `id` (string): Post UUID
- `excerpt` (string): First 160 characters of post text, truncated at word boundary when possible
- `scene_id` (string, optional): Scene UUID if post is associated with a scene
- `created_at` (string): ISO 8601 timestamp
- `next_cursor` (string): Cursor for next page (empty on last page)
- `count` (int): Number of results in this response

**Ranking Algorithm:**

Currently, post search uses text relevance only:

```
score = text_relevance * 1.0
```

(Trust-weighted ranking integration is planned for a future update)

Where:
- `text_relevance`: 1.0 for exact substring match, proportional for partial word matches (e.g., 2/3 words matched = 0.67)
- Results are ordered by `(score DESC, id ASC)` for stable pagination

**Moderation Filtering:**

The following posts are automatically excluded from search results:
- Soft-deleted posts (`deleted_at IS NOT NULL`)
- Posts with `hidden` label
- Posts with `spam` label
- Posts with `flagged` label

**Error Responses:**

| Status | Error Code         | Condition |
|--------|-------------------|-----------|
| 400    | `validation_error` | Missing `q` parameter |
| 400    | `validation_error` | Invalid `limit` (negative, zero, or not an integer) |
| 500    | `internal_error`   | Database or search failure |

**Example Request:**

```bash
# Basic search
GET /search/posts?q=electronic+music

# With scene filter
GET /search/posts?q=techno&scene_id=scene-uuid

# With pagination
GET /search/posts?q=festival&limit=10&cursor=0.875000:post-uuid
```

**Example Response:**

```json
{
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "excerpt": "Electronic music festival happening next week! Join us for amazing techno, house, and ambient performances...",
      "scene_id": "650e8400-e29b-41d4-a716-446655440000",
      "created_at": "2024-01-24T15:30:00Z"
    },
    {
      "id": "750e8400-e29b-41d4-a716-446655440000",
      "excerpt": "Just discovered this incredible electronic music scene. The community here is so welcoming and the sound is pure...",
      "scene_id": "650e8400-e29b-41d4-a716-446655440000",
      "created_at": "2024-01-24T14:20:00Z"
    }
  ],
  "next_cursor": "1.000000:750e8400-e29b-41d4-a716-446655440000",
  "count": 2
}
```

### Scene Search

**Endpoint:** `GET /search/scenes`

**Description:** Search for scenes within a bounding box with optional text query. Results include jittered coordinates for privacy.

**Query Parameters:**

| Parameter | Type   | Required | Default | Description |
|-----------|--------|----------|---------|-------------|
| `bbox`    | string | Yes      | -       | Bounding box (format: `minLng,minLat,maxLng,maxLat`) |
| `q`       | string | No       | -       | Text search query |
| `cursor`  | string | No       | -       | Pagination cursor |
| `limit`   | int    | No       | 20      | Number of results per page (max 50) |

See existing scene search documentation for details.

## Privacy Considerations

### Location Privacy

All scene coordinates in search results are **automatically jittered** using deterministic geohash-based noise:
- Jitter range: 500m to 1.5km at equator
- Same coordinates always produce the same jittered result (deterministic)
- Prevents exact location exposure while maintaining discoverability
- Implemented via `applyJitter()` function using geohash precision 8

### Content Privacy

Post search respects moderation labels:
- `hidden`: Never appears in public search
- `spam`: Excluded from search results
- `flagged`: Excluded from search results
- `nsfw`: Posts are returned but should be filtered client-side based on user preferences

## Trust-Weighted Ranking

Trust-weighted ranking is controlled by the `RANK_TRUST_ENABLED` environment variable (default: `false`).

**When Enabled:**
- Scene search includes trust scores in ranking calculation
- Post search is currently ranked by text relevance only (trust score integration pending)
- Response includes `trust_score` field for scene search results

**When Disabled:**
- Trust scores are not fetched or included in ranking
- `trust_score` field is omitted from responses
- Scene and post search ranking is based purely on text relevance

**Setting the Flag:**

```bash
# Enable trust ranking
export RANK_TRUST_ENABLED=true

# Disable trust ranking (default)
export RANK_TRUST_ENABLED=false
```

Accepted values: `true/false`, `1/0`, `yes/no`, `on/off` (case-insensitive)

## Performance Considerations

### Pagination

- Use cursor-based pagination for consistent results
- Cursors are stable: same query always returns same results in same order
- Cursor format: `"score:id"` for posts, implementation-specific for scenes
- Empty `next_cursor` indicates no more results

### Limits

- Default limit: 20 results
- Maximum limit: 50 results (requests above this are capped)
- Larger limits increase response time and memory usage
- Consider using smaller limits for mobile clients

### Bounding Box (Scene Search)

- Maximum bbox area: 10 square degrees (~1000km x 1000km at equator)
- Larger areas are rejected with `validation_error`
- Use smaller bboxes for faster queries and more relevant results

## Implementation Details

### Text Search

The current in-memory implementation uses simple substring matching:
- Exact substring match: relevance = 1.0
- Partial word match: relevance = (matched words) / (total words)
- Case-insensitive matching
- No stemming or fuzzy matching (future enhancement)

For production PostgreSQL implementation, use:
- `tsvector` for full-text search indexing
- `ts_rank()` for relevance scoring
- GIN index for performance

### Moderation Filtering

Moderation filtering is applied at the repository layer:
- `HasLabel()` method checks for moderation labels
- Excluded posts never reach the handler
- No post-processing required
- Consistent with feed endpoints

## Testing

Comprehensive test coverage includes:
- Basic text search with multiple matches
- Scene filter reducing scope
- Moderation label filtering (hidden, spam, flagged)
- Cursor-based pagination mechanics
- Missing/invalid query parameter validation
- Limit validation and capping
- Excerpt generation for long/short text

Run tests:

```bash
# All search tests
go test -v ./internal/api -run TestSearch

# Post search tests only
go test -v ./internal/api -run TestSearchPosts
```

## Future Enhancements

### Short-term
- [ ] PostgreSQL repository implementation with `tsvector`
- [ ] GIN indexes for post text search
- [ ] Query performance benchmarks

### Long-term
- [ ] Fuzzy matching and stemming
- [ ] Multi-language support
- [ ] Search result highlighting
- [ ] Advanced filters (date range, tags, author)
- [ ] Search analytics and trending queries
- [ ] Elasticsearch integration for complex queries
