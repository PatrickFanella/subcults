# Stream Analytics API

## Overview

The Stream Analytics API provides comprehensive engagement metrics for LiveKit audio streaming sessions. Analytics are automatically computed when a stream ends and can be accessed by the stream host (scene or event organizer).

## Privacy Principles

All analytics data is **privacy-first** and **PII-free**:
- No individual participant identities are exposed
- Geographic distribution uses coarse 4-character geohash prefixes (~20km resolution)
- Only aggregate statistics are provided
- Participant DIDs are used internally for deduplication but never exposed

## Metrics

### Core Engagement Metrics

- **Peak Concurrent Listeners**: Maximum number of simultaneous participants during the stream
- **Total Unique Participants**: Count of distinct participants who joined
- **Total Join Attempts**: Total number of join events (includes re-joins)

### Timing Metrics

- **Stream Duration**: Total duration of the stream in seconds (from start to end)
- **Engagement Lag**: Time in seconds from stream start to the first participant join (NULL if no joins)

### Retention Metrics

- **Average Listen Duration**: Mean time participants stayed in the stream (in seconds)
- **Median Listen Duration**: Median time participants stayed in the stream (in seconds)

### Geographic Distribution

- **Geographic Distribution**: Privacy-safe aggregate of participant locations
  - Format: `{"geohash_prefix": unique_participant_count}`
  - Example: `{"dr5r": 5, "9q8y": 3}`
  - Each prefix is 4 characters (~20km resolution)
  - Counts represent the number of distinct participants (deduplicated by DID) whose location resolves to that geohash prefix; re-joins are not double-counted
  - Only participants who provided location data are included

## API Endpoints

### GET /streams/{id}/analytics

Retrieves computed analytics for a completed stream session.

**Authorization**: Required. Must be the stream host (scene owner for scene streams, event host for event streams).

**Path Parameters**:
- `id` (string, required): Stream session ID

**Response**: 200 OK

```json
{
  "id": "analytics-uuid",
  "stream_session_id": "stream-uuid",
  "peak_concurrent_listeners": 15,
  "total_unique_participants": 23,
  "total_join_attempts": 28,
  "stream_duration_seconds": 3600,
  "engagement_lag_seconds": 45,
  "avg_listen_duration_seconds": 1200.5,
  "median_listen_duration_seconds": 950.0,
  "geographic_distribution": {
    "dr5r": 10,
    "9q8y": 8,
    "u4pr": 5
  },
  "computed_at": "2026-01-28T18:00:00Z"
}
```

**Error Responses**:

- `400 Bad Request`: Stream has not ended yet
  ```json
  {
    "error": "validation_error",
    "message": "Analytics not available until stream ends"
  }
  ```

- `403 Forbidden`: User is not the stream host
  ```json
  {
    "error": "forbidden",
    "message": "You must be the stream host to view analytics"
  }
  ```

- `404 Not Found`: Stream not found or analytics not computed
  ```json
  {
    "error": "not_found",
    "message": "Analytics not yet computed for this stream"
  }
  ```

## Participant Event Recording

Analytics are computed from granular participant events recorded during the stream lifecycle.

### POST /streams/{id}/join

Records a participant join event. Optionally accepts geographic data for distribution analytics.

**Request Body**:
```json
{
  "token_issued_at": "2026-01-28T17:30:00Z",
  "geohash_prefix": "dr5regw3"
}
```

**Fields**:
- `token_issued_at` (string, optional): RFC3339 timestamp for latency tracking
- `geohash_prefix` (string, optional): Geohash for location tracking (truncated to 4 chars for privacy)

### POST /streams/{id}/leave

Records a participant leave event. Used to calculate retention metrics.

**Request Body**: Empty

## Analytics Computation

Analytics are automatically computed when a stream ends:

1. Stream host calls `POST /streams/{id}/end`
2. Backend marks stream as ended
3. Backend automatically computes analytics from recorded participant events
4. Analytics become available via `GET /streams/{id}/analytics`

### Computation Algorithm

**Peak Concurrent Listeners**:
- Track concurrent participant count by processing join/leave events chronologically
- Record the maximum concurrent count observed

**Engagement Lag**:
- Calculate time difference between stream start and first join event
- NULL if no participants joined

**Retention Metrics**:
- For each participant who left, calculate duration = leave_time - join_time
- Average: Mean of all durations
- Median: Middle value of sorted durations
- Only includes participants who explicitly left (excludes still-listening participants)

**Geographic Distribution**:
- Group unique participants by 4-character geohash prefix
- Count distinct participants per prefix (deduplicated by participant DID, ignoring re-joins)
- Excludes participants without geographic data

## Database Schema

### stream_participant_events

Records individual join/leave events for detailed analytics.

```sql
CREATE TABLE stream_participant_events (
    id UUID PRIMARY KEY,
    stream_session_id UUID NOT NULL REFERENCES stream_sessions(id) ON DELETE CASCADE,
    participant_did VARCHAR(255) NOT NULL,
    event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('join', 'leave')),
    geohash_prefix VARCHAR(4), -- Privacy-safe 4-char prefix
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### stream_analytics

Stores computed aggregate metrics for ended streams.

```sql
CREATE TABLE stream_analytics (
    id UUID PRIMARY KEY,
    stream_session_id UUID NOT NULL UNIQUE REFERENCES stream_sessions(id) ON DELETE CASCADE,
    peak_concurrent_listeners INTEGER NOT NULL DEFAULT 0,
    total_unique_participants INTEGER NOT NULL DEFAULT 0,
    total_join_attempts INTEGER NOT NULL DEFAULT 0,
    stream_duration_seconds INTEGER NOT NULL DEFAULT 0,
    engagement_lag_seconds INTEGER, -- NULL if no joins
    avg_listen_duration_seconds FLOAT,
    median_listen_duration_seconds FLOAT,
    geographic_distribution JSONB DEFAULT '{}'::jsonb,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Example Usage

### Frontend Integration

```typescript
// Record join with optional location
async function joinStream(streamId: string, location?: GeolocationPosition) {
  const geohash = location ? encodeGeohash(location) : null;
  
  await fetch(`/api/streams/${streamId}/join`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      token_issued_at: new Date().toISOString(),
      geohash_prefix: geohash?.substring(0, 8) // Client sends 8 chars, server truncates to 4
    })
  });
}

// View analytics after stream ends
async function fetchAnalytics(streamId: string) {
  const response = await fetch(`/api/streams/${streamId}/analytics`);
  if (response.ok) {
    const analytics = await response.json();
    displayAnalyticsDashboard(analytics);
  }
}
```

## Testing

Comprehensive test coverage includes:
- Unit tests for analytics computation logic
- Integration tests for API endpoints
- Privacy constraint verification (geohash truncation, authorization)
- Edge cases (no participants, active streams, re-joins)

Run tests:
```bash
go test -v ./internal/stream/...
```

## Performance Considerations

- Analytics computation is O(n) where n = number of participant events
- Retention calculation requires sorting: O(n log n)
- Database indexes optimize queries:
  - `stream_session_id` for event lookups
  - `event_type, occurred_at` for chronological processing

## Future Enhancements

Potential additions (not in current scope):
- Real-time analytics dashboard (WebSocket updates)
- Historical trend analysis across multiple streams
- Comparative metrics (vs. previous streams)
- Export to CSV/PDF for reporting
