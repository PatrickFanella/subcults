# Event Handlers Documentation

This document describes the HTTP handlers for event CRUD operations in the Subcults API.

## Overview

Event handlers provide endpoints for creating, updating, and retrieving events within scenes. Events drive temporal discovery on the platform, enabling users to schedule and discover time-based activities in music scenes.

## Handlers

### EventHandlers

The `EventHandlers` struct manages all event-related HTTP endpoints.

```go
type EventHandlers struct {
    eventRepo scene.EventRepository
    sceneRepo scene.SceneRepository
    auditRepo audit.Repository
}
```

**Constructor:**

```go
func NewEventHandlers(eventRepo scene.EventRepository, sceneRepo scene.SceneRepository, auditRepo audit.Repository) *EventHandlers
```

## Endpoints

### POST /events - Create Event

Creates a new event within a scene.

**Request Body:**

```json
{
  "scene_id": "uuid",
  "title": "Event Title",
  "description": "Optional description",
  "allow_precise": true,
  "precise_point": {
    "lat": 40.7128,
    "lng": -74.0060
  },
  "coarse_geohash": "dr5regw",
  "tags": ["tag1", "tag2"],
  "starts_at": "2024-12-25T20:00:00Z",
  "ends_at": "2024-12-25T23:00:00Z"
}
```

**Required Fields:**
- `scene_id`: UUID of the parent scene
- `title`: Event title (3-80 characters)
- `coarse_geohash`: Geohash for approximate location
- `starts_at`: Event start time (RFC3339 format)

**Optional Fields:**
- `description`: Event description
- `allow_precise`: Privacy consent for precise location (default: false)
- `precise_point`: Precise GPS coordinates (only stored if `allow_precise` is true)
- `tags`: Array of categorization tags
- `ends_at`: Event end time (must be after `starts_at`)

**Authorization:**
- Requires authentication (JWT token)
- User must be the owner of the parent scene

**Validations:**
- Title length: 3-80 characters
- `coarse_geohash` is required and non-empty
- If `ends_at` is provided, `starts_at` must be before `ends_at`
- `scene_id` must reference an existing, non-deleted scene
- HTML sanitization applied to `title`, `description`, and `tags`

**Privacy Enforcement:**
- If `allow_precise` is false, `precise_point` is cleared before storage
- Repository automatically enforces location consent

**Success Response (201 Created):**

```json
{
  "id": "event-uuid",
  "scene_id": "scene-uuid",
  "title": "Event Title",
  "description": "Description",
  "allow_precise": true,
  "precise_point": {
    "lat": 40.7128,
    "lng": -74.0060
  },
  "coarse_geohash": "dr5regw",
  "tags": ["tag1", "tag2"],
  "status": "scheduled",
  "starts_at": "2024-12-25T20:00:00Z",
  "ends_at": "2024-12-25T23:00:00Z",
  "created_at": "2024-12-09T18:00:00Z",
  "updated_at": "2024-12-09T18:00:00Z"
}
```

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `bad_request` | Invalid JSON in request body |
| 400 | `validation_error` | Title length invalid, or missing required field |
| 400 | `invalid_time_range` | Start time is not before end time |
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the parent scene |
| 404 | `not_found` | Parent scene not found or deleted |
| 500 | `internal_error` | Server error during creation |

### PATCH /events/{id} - Update Event

Updates an existing event.

**URL Parameters:**
- `id`: Event UUID

**Request Body (all fields optional):**

```json
{
  "title": "Updated Title",
  "description": "Updated description",
  "tags": ["new", "tags"],
  "allow_precise": false,
  "coarse_geohash": "dr5regx",
  "starts_at": "2024-12-26T20:00:00Z",
  "ends_at": "2024-12-26T23:00:00Z"
}
```

**Immutable Fields:**
- `scene_id`: Cannot be changed after creation

**Update Restrictions:**
- `starts_at` can only be updated if the event is still in the future
- Time window validation applies: `starts_at` must be before `ends_at`

**Authorization:**
- Requires authentication (JWT token)
- User must be the owner of the parent scene

**Validations:**
- If `title` is provided, must be 3-80 characters
- If `coarse_geohash` is provided, must be non-empty
- Time window validation: `starts_at` < `ends_at`
- Cannot update `starts_at` for past events
- HTML sanitization applied to updated fields

**Success Response (200 OK):**

```json
{
  "id": "event-uuid",
  "scene_id": "scene-uuid",
  "title": "Updated Title",
  "description": "Updated description",
  "allow_precise": false,
  "coarse_geohash": "dr5regx",
  "tags": ["new", "tags"],
  "status": "scheduled",
  "starts_at": "2024-12-26T20:00:00Z",
  "ends_at": "2024-12-26T23:00:00Z",
  "created_at": "2024-12-09T18:00:00Z",
  "updated_at": "2024-12-09T18:30:00Z"
}
```

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `bad_request` | Invalid JSON or missing event ID |
| 400 | `validation_error` | Validation failed or cannot update past event |
| 400 | `invalid_time_range` | Start time is not before end time |
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the parent scene |
| 404 | `not_found` | Event or parent scene not found |
| 500 | `internal_error` | Server error during update |

### GET /events/{id} - Get Event

Retrieves a single event by ID.

**URL Parameters:**
- `id`: Event UUID

**Authorization:**
- Public endpoint (no authentication required)

**Privacy Enforcement:**
- If `allow_precise` is false, `precise_point` is excluded from response
- Repository automatically enforces location consent

**Success Response (200 OK):**

```json
{
  "id": "event-uuid",
  "scene_id": "scene-uuid",
  "title": "Event Title",
  "description": "Description",
  "allow_precise": true,
  "precise_point": {
    "lat": 40.7128,
    "lng": -74.0060
  },
  "coarse_geohash": "dr5regw",
  "tags": ["tag1", "tag2"],
  "status": "scheduled",
  "starts_at": "2024-12-25T20:00:00Z",
  "ends_at": "2024-12-25T23:00:00Z",
  "created_at": "2024-12-09T18:00:00Z",
  "updated_at": "2024-12-09T18:00:00Z"
}
```

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `bad_request` | Missing or invalid event ID |
| 404 | `not_found` | Event not found |
| 500 | `internal_error` | Server error during retrieval |

### POST /events/{id}/cancel - Cancel Event

Cancels an event by updating its status and storing cancellation metadata. This endpoint is idempotent: cancelling an already-cancelled event returns success without modification.

**URL Parameters:**
- `id`: Event UUID

**Request Body (optional):**

```json
{
  "reason": "Venue unavailable"
}
```

**Optional Fields:**
- `reason`: Text explanation for cancellation (sanitized for HTML safety)

**Authorization:**
- Requires authentication (JWT token)
- User must be the owner of the parent scene

**Behavior:**
- Sets `status` to `"cancelled"`
- Stores `cancelled_at` timestamp (current time)
- Stores `cancellation_reason` if provided
- Emits audit log entry with action `"event_cancel"`
- **Idempotent:** Second cancellation of same event returns 200 OK with no changes and no duplicate audit log

**Success Response (200 OK):**

```json
{
  "id": "event-uuid",
  "scene_id": "scene-uuid",
  "title": "Event Title",
  "description": "Description",
  "allow_precise": true,
  "precise_point": {
    "lat": 40.7128,
    "lng": -74.0060
  },
  "coarse_geohash": "dr5regw",
  "tags": ["tag1", "tag2"],
  "status": "cancelled",
  "starts_at": "2024-12-25T20:00:00Z",
  "ends_at": "2024-12-25T23:00:00Z",
  "created_at": "2024-12-09T18:00:00Z",
  "updated_at": "2024-12-09T18:30:00Z",
  "cancelled_at": "2024-12-09T18:30:00Z",
  "cancellation_reason": "Venue unavailable"
}
```

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `bad_request` | Missing or invalid event ID |
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the parent scene |
| 404 | `not_found` | Event or parent scene not found |
| 500 | `internal_error` | Server error during cancellation |

**Audit Logging:**
- Entity Type: `"event"`
- Entity ID: Event UUID
- Action: `"event_cancel"`
- Only logged on first cancellation (not on idempotent retries)

**Search/List Behavior:**
- Cancelled events are excluded from upcoming event searches/listings
- Existing database indexes use `WHERE cancelled_at IS NULL` for filtering

## Validation Rules

### Title Validation

```go
// Title length constraints
const (
    MinEventTitleLength = 3
    MaxEventTitleLength = 80
)
```

- Minimum length: 3 characters
- Maximum length: 80 characters
- Whitespace is trimmed before validation
- HTML sanitization applied after validation

### Time Window Validation

```go
func validateTimeWindow(startsAt time.Time, endsAt *time.Time) string
```

- If `ends_at` is provided, `starts_at` must be strictly before `ends_at`
- Equal times are rejected (error code: `invalid_time_range`)
- Past events cannot have `starts_at` updated

### Coarse Geohash Validation

- Required on creation
- Must be non-empty string
- Used for approximate location-based discovery

## Security Considerations

### HTML Sanitization

All user-provided text fields are sanitized using `html.EscapeString()`:
- `title`
- `description`
- `tags` (each element individually)

### Authorization

- Event creation and updates require scene ownership verification
- Uses `isSceneOwner()` helper to check authorization
- Uniform error messages prevent user enumeration

### Privacy Enforcement

- Location consent enforced at multiple layers:
  1. Repository level (automatic)
  2. Handler level (explicit check in GET handler)
- `precise_point` automatically cleared if `allow_precise` is false
- Prevents precise location exposure without explicit consent

## Testing

Comprehensive test coverage includes:

- **Success Cases:**
  - Event creation with all fields
  - Event updates (title, description, tags, times)
  - Event retrieval

- **Validation Tests:**
  - Title too short (< 3 chars)
  - Title too long (> 80 chars)
  - Missing coarse_geohash
  - Invalid time window (end before start, equal times)
  - Cannot update past event start time

- **Authorization Tests:**
  - Unauthorized creation (non-scene-owner)
  - Unauthorized update (non-scene-owner)

- **Privacy Tests:**
  - Privacy enforcement on creation (allow_precise=false)
  - Privacy enforcement on retrieval (precise_point hidden)

- **Cancellation Tests:**
  - Successful cancellation with reason
  - Successful cancellation without reason
  - Unauthorized cancellation (non-scene-owner)
  - Idempotent cancellation (already cancelled)
  - Audit log emission on first cancel
  - No duplicate audit log on second cancel

Run tests:

```bash
go test -v ./internal/api/event_handlers_test.go
```

## Error Codes

| Code | Usage |
|------|-------|
| `invalid_time_range` | Start time is not before end time |
| `validation_error` | Generic input validation failure |
| `auth_failed` | Authentication required |
| `forbidden` | User lacks permission (not scene owner) |
| `not_found` | Event or scene not found |
| `bad_request` | Malformed request (invalid JSON, missing ID) |
| `internal_error` | Server error |

## Database Schema

### Event Cancellation Fields

The events table includes the following cancellation-related fields:

- `status` TEXT - Event lifecycle status (scheduled, live, ended, cancelled)
- `cancelled_at` TIMESTAMPTZ - Timestamp when event was cancelled (NULL if not cancelled)
- `cancellation_reason` TEXT - Optional reason for cancellation (NULL if not provided)

**Indexes:**
Database indexes automatically filter out cancelled events using `WHERE cancelled_at IS NULL` clause, ensuring cancelled events are excluded from performance-critical queries.

## Future Enhancements

- Event search and filtering endpoints
- Event listing by scene
- Event status transitions (scheduled → live → ended)
- Recurring events support
- Event attendance/RSVP functionality
- Integration with LiveKit for live streaming events

## Search Events Endpoint

### GET /search/events - Search Events

Searches for events within a geographic bounding box and time window with optional text search and trust-weighted ranking.

**Query Parameters:**

- `bbox` (required): Bounding box in format `minLng,minLat,maxLng,maxLat` (e.g., `-74.1,40.6,-73.9,40.8`)
- `from` (required): Start of time window (RFC3339 format)
- `to` (required): End of time window (RFC3339 format)
- `q` (optional): Text search query (searches title, description, and tags)
- `limit` (optional): Results per page (1-100, default: 50)
- `cursor` (optional): Pagination cursor from previous response

**Validations:**

- `bbox` must have exactly 4 coordinates
- Longitude must be between -180 and 180
- Latitude must be between -90 and 90
- `minLng` must be less than `maxLng`
- `minLat` must be less than `maxLat`
- `from` and `to` must be valid RFC3339 timestamps
- `from` must be before `to`
- Time window cannot exceed 30 days
- `limit` must be between 1 and 100

**Filtering:**

- Only returns events with `status != "cancelled"`
- Only returns events with `deleted_at IS NULL`
- Only returns events within the specified time window
- Only returns events within the geographic bounding box
- If `q` is provided, only returns events matching the text query

**Ranking Formula:**

Events are ranked by composite score:

```
composite_score = (recency_weight * 0.3) + (text_match_score * 0.4) + 
                  (proximity_score * 0.2) + (trust_score * 0.1)
```

**Ranking Components:**

1. **Recency Weight (30%):** Time-based scoring favoring events happening sooner
   - Formula: `1 - ((event_start - now) / window_span)` clamped to [0, 1]
   - Events happening now or in the past: 1.0
   - Events at the end of the time window: ~0.0

2. **Text Match Score (40%):** Relevance to search query
   - Title match: 1.0
   - Description match: 0.8
   - Tag match: 0.6
   - No match: 0.0
   - Empty query: 1.0 for all events

3. **Proximity Score (20%):** Distance from bbox center
   - Formula: `1 / (1 + distance)`
   - Center of bbox: 1.0
   - Further away: approaches 0.0
   - No location: 0.5

4. **Trust Score (10%):** Scene reputation (when enabled)
   - Requires trust score store configured
   - Based on scene memberships and alliances
   - Range: 0.0 to 1.0
   - If trust ranking is disabled: weight is 0.0

**Trust-Weighted Ranking:**

Trust scores are only included in ranking when:
1. A trust score store is provided to `EventHandlers`
2. Trust scores are available for the event's scenes

Trust ranking can be enabled/disabled via the `RANK_TRUST_ENABLED` environment variable.

**Success Response (200 OK):**

```json
{
  "events": [
    {
      "event": {
        "id": "event-uuid",
        "scene_id": "scene-uuid",
        "title": "Event Title",
        "description": "Description",
        "allow_precise": false,
        "coarse_geohash": "dr5regw",
        "tags": ["tag1", "tag2"],
        "status": "scheduled",
        "starts_at": "2024-12-25T20:00:00Z",
        "ends_at": "2024-12-25T23:00:00Z",
        "created_at": "2024-12-09T18:00:00Z",
        "updated_at": "2024-12-09T18:00:00Z"
      },
      "rsvp_counts": {
        "going": 42,
        "interested": 15,
        "not_going": 2
      },
      "active_stream": {
        "session_id": "stream-uuid",
        "room_name": "event-room",
        "participant_count": 23,
        "started_at": "2024-12-25T20:00:00Z"
      }
    }
  ],
  "next_cursor": "0.8532|event-uuid"
}
```

**Cursor Pagination:**

- Results are ordered by composite score (descending), then by ID for stable ordering
- Cursor format: `score|eventID` (e.g., `0.8532|event-uuid`)
- Pass `cursor` query parameter to get next page
- `next_cursor` is empty when no more results

**Privacy Considerations:**

- Precise coordinates are jittered based on `allow_precise` flag
- Only public location data is returned
- Scene trust scores are aggregated and do not expose individual user data

**Performance:**

- Batch fetches RSVP counts to avoid N+1 queries
- Batch fetches active stream sessions
- Batch fetches trust scores when available
- Default limit of 50 ensures reasonable response times

**Example Requests:**

Basic search:
```
GET /search/events?bbox=-74.1,40.6,-73.9,40.8&from=2024-12-01T00:00:00Z&to=2024-12-31T23:59:59Z
```

Text search:
```
GET /search/events?bbox=-74.1,40.6,-73.9,40.8&from=2024-12-01T00:00:00Z&to=2024-12-31T23:59:59Z&q=electronic+music
```

With pagination:
```
GET /search/events?bbox=-74.1,40.6,-73.9,40.8&from=2024-12-01T00:00:00Z&to=2024-12-31T23:59:59Z&limit=20&cursor=0.8532|event-uuid
```

**Error Responses:**

- `400 Bad Request` with `validation` code: Invalid parameters
- `400 Bad Request` with `invalid_time_range` code: Invalid time window
- `500 Internal Server Error` with `internal` code: Server error

**Implementation Notes:**

- In-memory implementation uses simple substring matching and Euclidean distance
- Production PostgreSQL implementation should use:
  - Full-text search (FTS) with `tsvector` and `ts_rank` for text matching
  - PostGIS `ST_Distance` for proper geodesic distance calculations
  - Indexes on `starts_at`, `tsvector`, and geospatial columns for performance
