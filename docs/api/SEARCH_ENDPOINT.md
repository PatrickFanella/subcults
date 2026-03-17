# Scene Search Endpoint

## Overview

The scene search endpoint provides discovery of scenes through text search, geographic filtering, and trust-weighted ranking. It implements cursor-based pagination for efficient browsing and enforces privacy controls by always returning jittered coordinates.

## Endpoint

```
GET /search/scenes
```

## Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `q` | string | No | - | Text search query matching scene name, description, or tags |
| `bbox` | string | Conditionally | - | Bounding box in format `minLng,minLat,maxLng,maxLat` (required when `lat`/`lon` are not provided) |
| `lat` | float | Conditionally | - | Reference latitude for proximity scoring (must be paired with `lon`) |
| `lon` | float | Conditionally | - | Reference longitude for proximity scoring (must be paired with `lat`) |
| `genres` | string | No | - | Comma-separated tag/genre filter (e.g. `techno,jazz`) |
| `limit` | integer | No | 20 | Max results per page (1-50) |
| `offset` | integer | No | 0 | Offset pagination (non-negative) |
| `cursor` | string | No | - | Pagination cursor from previous response |

## Response Format

```json
{
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Electronic Music Scene",
      "description": "Underground techno parties in Brooklyn",
      "jittered_centroid": {
        "lat": 40.7150,
        "lng": -74.0080
      },
      "coarse_geohash": "dr5regw",
      "tags": ["electronic", "techno", "underground"],
      "visibility": "public",
      "trust_score": 0.85
    }
  ],
  "next_cursor": "eyJzY29yZSI6MC44NSwiaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ==",
  "count": 1
}
```

## Response Fields

### SceneSearchResult

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | UUID of the scene |
| `name` | string | Scene name |
| `description` | string | Scene description (optional) |
| `jittered_centroid` | Point | **Privacy-protected** coordinates (always jittered) |
| `coarse_geohash` | string | Coarse location geohash for privacy |
| `tags` | array | Categorization tags |
| `visibility` | string | Visibility mode (`public`, `private`, `unlisted`). `unlisted` scenes are hidden and excluded from search results. |
| `trust_score` | float | Trust score (0.0-1.0, only if trust ranking enabled) |

### Point

| Field | Type | Description |
|-------|------|-------------|
| `lat` | float | Latitude (jittered for privacy) |
| `lng` | float | Longitude (jittered for privacy) |

## Ranking Algorithm

Scenes are ranked by a composite score combining multiple factors:

```
composite_score = (text_match * 0.6) + (proximity * 0.25) + (trust * 0.15)
```

### Ranking Components

- **Text Match (60%)**: Query relevance
  - Name match: 1.0
  - Description match: 0.7
  - Tag match: 0.5
  - No match: 0.0
  
- **Proximity (25%)**: Distance from bbox center
  - Uses decay function: `1 / (1 + distance)`
  - Closer scenes rank higher
  
- **Trust (15%)**: Scene reputation
  - Only applied when `RANK_TRUST_ENABLED=true`
  - Based on alliance graph and owner reputation

Results are sorted by score descending, then by ID ascending for stable ordering.

## Examples

### Text Search with Bounding Box

```bash
curl "http://localhost:8080/search/scenes?q=electronic&bbox=-74.1,40.6,-73.9,40.8"
```

### Geographic Search Only

```bash
curl "http://localhost:8080/search/scenes?bbox=-74.1,40.6,-73.9,40.8"
```

### Combined Search with Pagination

```bash
# First page
curl "http://localhost:8080/search/scenes?q=music&bbox=-74.1,40.6,-73.9,40.8&limit=20"

# Next page using cursor from previous response
curl "http://localhost:8080/search/scenes?q=music&bbox=-74.1,40.6,-73.9,40.8&limit=20&cursor=eyJzY29yZSI6MC44NSwiaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ=="
```

## Validation Rules

### Bounding Box

- **Format**: `minLng,minLat,maxLng,maxLat`
- **Coordinate Ranges**:
  - Longitude: -180 to 180
  - Latitude: -90 to 90
- **Constraints**:
  - `minLng < maxLng`
  - `minLat < maxLat`
  - Area ≤ 10 square degrees (prevents wide scans)

### Limit

- **Range**: 1-50
- **Default**: 20
- **Behavior**: Values > 50 are capped to 50

## Error Responses

### Missing Bounding Box

```json
{
  "error": {
    "code": "validation_error",
    "message": "bbox parameter is required"
  }
}
```

**Status**: 400 Bad Request

### Invalid Bounding Box

```json
{
  "error": {
    "code": "validation_error",
    "message": "bbox area too large (max 10.0 square degrees)"
  }
}
```

**Status**: 400 Bad Request

### Invalid Cursor

```json
{
  "error": {
    "code": "internal_error",
    "message": "Failed to search scenes"
  }
}
```

**Status**: 500 Internal Server Error

## Privacy Controls

### Location Jittering

All coordinates returned in `jittered_centroid` are **privacy-protected**:

- Applies deterministic offset based on original coordinates
- Prevents exact location tracking
- Maintains stability (same scene = same jittered point)
- Sufficient for map display without revealing precise venue

### Visibility Filtering

Only scenes the requester is authorized to discover are returned:

- **public**: Included in search results for all callers
- **private**: **Currently included in search results** (membership-based filtering not yet implemented - see Security Note below)
- **unlisted**: **Always excluded** from search results

**Security Note**: The current implementation does not enforce membership-based access control for private scenes in search results. This means private scenes are discoverable by unauthenticated users and non-members through the search endpoint. This is a known limitation that will be addressed in a future update to apply proper authorization checks based on scene membership before including private scenes in search results.

### Deleted Scenes

Soft-deleted scenes (with `deleted_at` timestamp) are automatically excluded.

## Pagination

### Cursor Format

Cursors are base64-encoded JSON containing:

```json
{
  "score": 0.85,
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Stable Ordering

Results use deterministic ordering:
1. Primary: Score (descending)
2. Secondary: ID (ascending)

This ensures:
- No duplicates across pages
- Consistent ordering with same query
- Stable pagination even with score ties

### End of Results

When `next_cursor` is empty, there are no more results.

## Performance Considerations

### Bbox Area Limit

The 10 square degree limit prevents expensive wide scans:

- Roughly 1000km × 1000km at equator
- Encourages focused geographic queries
- Prevents database overload

### Result Limits

Default of 20 results balances:
- User experience (reasonable page size)
- Server load (efficient queries)
- Network bandwidth (smaller payloads)

## Feature Flags

### Trust Ranking

**Environment Variable**: `RANK_TRUST_ENABLED`

**Values**: `true`, `false`, `1`, `0`, `yes`, `no`, `on`, `off`

**Default**: `false`

**Behavior**:
- When **enabled**: Trust score contributes 15% to ranking
- When **disabled**: Trust score excluded from composite score

## Future Enhancements

Planned improvements:

1. **Query Caching**: Redis cache for popular searches
2. **Autocomplete**: Prefix matching for scene names
3. **Faceted Search**: Additional filters by visibility and metadata
4. **Relevance Tuning**: ML-based ranking optimization

## Related Documentation

- [Privacy Architecture](/docs/PRIVACY.md)
- [Trust Graph System](/docs/api/TRUST_HANDLERS.md)
- [Scene Visibility](/docs/api/SCENE_VISIBILITY.md)
