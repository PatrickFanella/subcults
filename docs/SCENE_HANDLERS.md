# Scene CRUD Handlers

## Overview

The scene handlers provide HTTP endpoints for creating, updating, and deleting scenes with privacy-first location handling and comprehensive validation.

## Endpoints

### POST /scenes

Creates a new scene.

**Request Body:**
```json
{
  "name": "Underground Jazz Club",
  "description": "Weekly jazz sessions in the basement",
  "owner_did": "did:plc:abc123",
  "allow_precise": true,
  "precise_point": {
    "lat": 40.7128,
    "lng": -74.0060
  },
  "coarse_geohash": "dr5regw",
  "tags": ["jazz", "live-music"],
  "visibility": "public",
  "palette": {
    "primary": "#1a1a1a",
    "secondary": "#ff6b35"
  }
}
```

**Response:** `201 Created`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Underground Jazz Club",
  "description": "Weekly jazz sessions in the basement",
  "owner_did": "did:plc:abc123",
  "allow_precise": true,
  "precise_point": {
    "lat": 40.7128,
    "lng": -74.0060
  },
  "coarse_geohash": "dr5regw",
  "tags": ["jazz", "live-music"],
  "visibility": "public",
  "palette": {
    "primary": "#1a1a1a",
    "secondary": "#ff6b35"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Validation:**
- `name`: Required, 3-64 characters, letters/numbers/spaces and limited punctuation (-, _, ', ., &)
- `owner_did`: Required
- `coarse_geohash`: Required (NOT NULL in database)
- `visibility`: Optional, defaults to "public", must be one of: "public", "private", "unlisted"

**Error Responses:**
- `400 Bad Request` - Invalid JSON or validation failure
- `409 Conflict` - Scene name already exists for this owner

### PATCH /scenes/{id}

Updates an existing scene. Only provided fields are updated.

**Request Body:**
```json
{
  "name": "Updated Scene Name",
  "description": "New description",
  "tags": ["updated", "tags"],
  "visibility": "unlisted",
  "palette": {
    "primary": "#000000",
    "secondary": "#ffffff"
  },
  "allow_precise": false
}
```

**Response:** `200 OK` - Returns updated scene

**Notes:**
- `owner_did` is immutable and cannot be updated
- Name uniqueness is checked excluding the current scene
- Privacy consent is enforced on update

**Error Responses:**
- `400 Bad Request` - Invalid JSON or validation failure
- `404 Not Found` - Scene not found or soft-deleted
- `409 Conflict` - Updated name conflicts with another scene

### DELETE /scenes/{id}

Soft-deletes a scene by setting `deleted_at` timestamp.

**Response:** `204 No Content`

**Error Responses:**
- `404 Not Found` - Scene not found or already deleted

### GET /scenes/owned

Lists all scenes owned by the authenticated user with summary statistics.

**Authentication:** Required (JWT token with user DID in context)

**Response:** `200 OK`
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Underground Jazz Club",
    "description": "Weekly jazz sessions in the basement",
    "coarse_geohash": "dr5regw",
    "tags": ["jazz", "live-music"],
    "visibility": "public",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "members_count": 15,
    "has_active_stream": true
  },
  {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "Electronic Music Collective",
    "description": "Experimental electronic music sessions",
    "coarse_geohash": "dr5rex1",
    "tags": ["electronic", "experimental"],
    "visibility": "private",
    "created_at": "2024-01-20T14:00:00Z",
    "updated_at": "2024-01-22T18:45:00Z",
    "members_count": 8,
    "has_active_stream": false
  }
]
```

**Response Fields:**
- `members_count`: Number of active memberships (status="active")
- `has_active_stream`: Boolean indicating if there's an active stream (ended_at IS NULL)
- Excludes heavy fields: `palette`, `precise_point`
- Excludes soft-deleted scenes (deleted_at IS NULL)

**Performance:**
- Uses batch queries to avoid N+1 query problem
- Single query for all scenes: `ListByOwner(userDID)`
- Single query for all membership counts: `CountByScenes(sceneIDs, "active")`
- Single query for all active stream checks: `HasActiveStreamsForScenes(sceneIDs)`
- Total: 3 queries regardless of number of scenes owned

**Error Responses:**
- `401 Unauthorized` - Authentication required (no user DID in context)

## Privacy Enforcement

All endpoints enforce location privacy:
- When `allow_precise=false`, `precise_point` is automatically cleared before storage
- Repository layer enforces this constraint via `EnforceLocationConsent()`
- Responses exclude `precise_point` when consent is not granted

## Security

### XSS Prevention
- Scene names are sanitized using `html.EscapeString()` after validation
- Validation runs before sanitization to allow legitimate punctuation

### Duplicate Prevention
- Scene names must be unique per owner
- Enforced via `ExistsByOwnerAndName()` repository method
- Update operations exclude current scene ID when checking duplicates

## Testing

Comprehensive test coverage includes:
- Success cases for all CRUD operations
- Privacy enforcement validation
- Duplicate name rejection
- Invalid name validation (length, character restrictions)
- Soft-delete behavior
- Missing required fields
- HTML injection prevention

Run tests:
```bash
go test -v ./internal/api/... -run Scene
```

## Usage Example

```go
// Initialize handlers
sceneRepo := scene.NewInMemorySceneRepository()
membershipRepo := membership.NewInMemoryMembershipRepository()
streamRepo := stream.NewInMemorySessionRepository()
handlers := api.NewSceneHandlers(sceneRepo, membershipRepo, streamRepo)

// Register routes (example with http.ServeMux)
mux := http.NewServeMux()
mux.HandleFunc("POST /scenes", handlers.CreateScene)
mux.HandleFunc("PATCH /scenes/", handlers.UpdateScene)
mux.HandleFunc("DELETE /scenes/", handlers.DeleteScene)
mux.HandleFunc("GET /scenes/owned", handlers.ListOwnedScenes)
```

## Future Enhancements

- Integration with chi router for cleaner URL parameter extraction
- GET endpoints for public scene listing and searching
- Batch operations
- Filtering by visibility, tags, location
- Pagination support for /scenes/owned endpoint
