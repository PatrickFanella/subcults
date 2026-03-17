# Participant State Synchronization in Streams

## Overview

This implementation provides real-time participant state tracking for LiveKit audio streaming sessions. It tracks when participants join and leave streams, handles reconnections, and broadcasts events via WebSocket for live UI updates.

## Architecture

### Database Schema

#### `stream_participants` Table

Tracks individual participant sessions within a stream:

```sql
CREATE TABLE stream_participants (
    id UUID PRIMARY KEY,
    stream_session_id UUID NOT NULL REFERENCES stream_sessions(id),
    participant_id VARCHAR(255) NOT NULL,  -- LiveKit identity (e.g., "user-abc123")
    user_did VARCHAR(255) NOT NULL,        -- Decentralized Identifier
    
    joined_at TIMESTAMPTZ NOT NULL,
    left_at TIMESTAMPTZ,                   -- NULL while active
    
    reconnection_count INT NOT NULL DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    
    CONSTRAINT unique_active_participant UNIQUE (stream_session_id, participant_id, left_at)
);
```

**Key Design Decisions:**

1. **Unique Constraint**: The `unique_active_participant` constraint prevents duplicate active participants. The constraint includes `left_at` to allow the same participant to appear multiple times in history (for reconnections).

2. **Reconnection Count**: Tracks how many times a participant has rejoined after leaving. Useful for analytics and detecting connection stability issues.

3. **Soft Delete**: Participants are never deleted. Setting `left_at` marks them as inactive while preserving history.

#### `stream_sessions.active_participant_count`

Denormalized field added to `stream_sessions` table for efficient queries:

```sql
ALTER TABLE stream_sessions
ADD COLUMN active_participant_count INT NOT NULL DEFAULT 0;
```

This avoids expensive `COUNT(*)` queries on every request. Updated atomically when participants join/leave.

### Core Components

#### 1. Participant Model (`internal/stream/participant.go`)

```go
type Participant struct {
    ID                string
    StreamSessionID   string
    ParticipantID     string     // LiveKit identity
    UserDID           string     // AT Protocol DID
    JoinedAt          time.Time
    LeftAt            *time.Time // NULL = active
    ReconnectionCount int
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

#### 2. Participant Repository (`internal/stream/participant_repository.go`)

**Interface:**
```go
type ParticipantRepository interface {
    RecordJoin(streamSessionID, participantID, userDID string) (*Participant, bool, error)
    RecordLeave(streamSessionID, participantID string) error
    GetActiveParticipants(streamSessionID string) ([]*Participant, error)
    GetParticipantHistory(streamSessionID string) ([]*Participant, error)
    GetActiveCount(streamSessionID string) (int, error)
    UpdateSessionParticipantCount(streamSessionID string, count int) error
}
```

**Key Methods:**

- **`RecordJoin`**: Creates a new participant record. Returns `(participant, isReconnection, error)`. 
  - Checks if participant is already active → returns `ErrParticipantAlreadyActive`
  - Checks history for previous joins → sets `reconnection_count` and `isReconnection` flag
  - Updates denormalized count on session

- **`RecordLeave`**: Marks participant as left by setting `left_at` timestamp.
  - Removes from active index
  - Updates denormalized count

- **`GetActiveCount`**: Returns count of active participants (efficient, uses index)

#### 3. Event Broadcaster (`internal/stream/event_broadcaster.go`)

WebSocket event broadcasting for real-time updates:

```go
type EventBroadcaster struct {
    connections map[string]map[*websocket.Conn]bool  // streamID -> connections
}

func (b *EventBroadcaster) Broadcast(streamSessionID string, event *ParticipantStateEvent)
```

**Event Format:**
```json
{
  "type": "participant_joined",  // or "participant_left"
  "stream_session_id": "uuid",
  "participant_id": "user-abc123",
  "user_did": "did:plc:abc123",
  "timestamp": "2024-01-28T18:00:00Z",
  "is_reconnection": false,
  "active_count": 5
}
```

### API Endpoints

#### Join Stream: `POST /streams/{id}/join`

Records a participant joining and broadcasts event:

1. Generate participant ID from user DID (deterministic)
2. Call `participantRepo.RecordJoin()`
3. Broadcast `participant_joined` event
4. Increment join count (analytics)
5. Record metrics

**Request:**
```json
{
  "token_issued_at": "2024-01-28T18:00:00Z",  // Optional, for latency tracking
  "geohash_prefix": "abcd"                     // Optional, for geo analytics
}
```

**Response:**
```json
{
  "stream_id": "uuid",
  "room_name": "scene-123-1234567890",
  "join_count": 10,
  "status": "joined"
}
```

#### Leave Stream: `POST /streams/{id}/leave`

Records a participant leaving and broadcasts event:

1. Generate participant ID from user DID
2. Call `participantRepo.RecordLeave()`
3. Broadcast `participant_left` event
4. Increment leave count (analytics)

**Response:**
```json
{
  "stream_id": "uuid",
  "room_name": "scene-123-1234567890",
  "leave_count": 5,
  "status": "left"
}
```

#### Get Active Participants: `GET /streams/{id}/participants`

Returns current participant count (no PII):

**Response:**
```json
{
  "stream_id": "uuid",
  "active_count": 5,
  "room_name": "scene-123-1234567890"
}
```

**Privacy Note**: Individual participant identities are NOT exposed. Only aggregate count is returned.

#### WebSocket Subscription: `GET /streams/{id}/participants/ws`

Upgrades to WebSocket for real-time participant events:

1. Verifies stream exists
2. Upgrades HTTP connection to WebSocket
3. Subscribes to event broadcaster
4. Streams `participant_joined` and `participant_left` events

**Example Events:**
```json
// Join event
{
  "type": "participant_joined",
  "stream_session_id": "uuid",
  "participant_id": "user-abc123",
  "user_did": "did:plc:abc123",
  "timestamp": "2024-01-28T18:00:00Z",
  "is_reconnection": false,
  "active_count": 6
}

// Leave event
{
  "type": "participant_left",
  "stream_session_id": "uuid",
  "participant_id": "user-abc123",
  "user_did": "did:plc:abc123",
  "timestamp": "2024-01-28T18:05:00Z",
  "is_reconnection": false,
  "active_count": 5
}
```

## Participant Identity Generation

**Function**: `stream.GenerateParticipantID(did string) string`

Generates deterministic participant IDs from user DIDs:

**Format**: `user-{identifier}`

**Algorithm:**
1. Parse DID (format: `did:method:identifier`)
2. Extract identifier part (last segment after `:`)
3. Truncate to 48 chars if needed
4. Prefix with `user-`

**Example:**
- Input: `did:plc:abc123xyz`
- Output: `user-abc123xyz`

**Why Deterministic?**

- Same user always gets same participant ID
- Enables LiveKit's automatic reconnection handling
- Proper cleanup of previous sessions when rejoining
- Consistent participant tracking across multiple join attempts

## Reconnection Handling

### Scenario: User Temporarily Disconnects

1. **Initial Join:**
   - User joins stream → `participant_id: user-abc123`, `reconnection_count: 0`
   - Active participants: 1

2. **Network Drop:**
   - User loses connection
   - UI detects disconnect, calls `/streams/{id}/leave`
   - `left_at` set to current timestamp
   - Active participants: 0

3. **Reconnection:**
   - User regains connection, calls `/streams/{id}/join`
   - System finds previous participant record
   - Creates new record with `reconnection_count: 1`
   - Broadcast event with `is_reconnection: true`
   - Active participants: 1

### Handling Multiple Active Sessions

The `unique_active_participant` constraint prevents duplicate active participants:

- If user tries to join twice (e.g., different tabs), second join returns `ErrParticipantAlreadyActive`
- Frontend should handle this gracefully (show already connected message)

### Edge Cases

1. **Stale Active Sessions**: If a user disconnects without calling `/leave`:
   - Session remains "active" in database
   - Consider implementing background cleanup job for sessions older than token expiry (15 min)
   - Or: LiveKit webhook integration to detect actual disconnections

2. **Rapid Rejoin**: User leaves and immediately rejoins:
   - Works correctly due to unique constraint logic
   - Reconnection count increments properly

## Testing

### Unit Tests

**Participant Repository:**
- ✅ Initial join creates new record
- ✅ Duplicate join returns error
- ✅ Leave marks participant as inactive
- ✅ Rejoin after leave increments reconnection count
- ✅ Active count updates correctly
- ✅ Participant history includes all sessions

**Participant ID Generation:**
- ✅ Standard DID format
- ✅ Long identifier truncation
- ✅ Malformed DID handling
- ✅ Deterministic (same input = same output)
- ✅ Uniqueness (different inputs = different outputs)

### Integration Testing Checklist

- [ ] Join stream via API → verify participant created
- [ ] Join count appears in response
- [ ] Active count increases
- [ ] WebSocket receives join event
- [ ] Leave stream → verify participant marked inactive
- [ ] Leave count appears in response
- [ ] Active count decreases
- [ ] WebSocket receives leave event
- [ ] Rejoin after leave → reconnection count increments
- [ ] WebSocket event has `is_reconnection: true`
- [ ] Duplicate join returns error
- [ ] Stream end → all active participants still tracked
- [ ] GET /participants returns correct count
- [ ] Denormalized count matches actual count

## Privacy & Security

### Privacy Considerations

1. **No PII in Public Endpoints**: The `/participants` endpoint returns only aggregate count, not individual identities.

2. **WebSocket Events Include DIDs**: Events broadcast via WebSocket include `user_did` for authenticated listeners. This is intentional - participants should know who else is in the stream.

3. **Audit Logging**: All join/leave actions are logged via audit repository with request ID for traceability.

4. **Geographic Privacy**: Optional `geohash_prefix` is truncated to 4 characters (~20km precision) for privacy-safe geographic distribution analytics.

### Security

1. **Authentication Required**: All endpoints require valid JWT authentication.

2. **Stream Validation**: All endpoints verify stream exists before processing.

3. **WebSocket Origin Check**: TODO - Implement proper CORS checking based on configuration.

4. **Rate Limiting**: Should be implemented at reverse proxy level (Caddy).

## Performance Considerations

### Denormalized Count

The `active_participant_count` field on `stream_sessions` provides O(1) lookup for participant counts without expensive `COUNT(*)` queries.

**Trade-off**: Extra write operations on join/leave vs. significantly faster reads.

### WebSocket Scalability

Current implementation broadcasts to all WebSocket connections for a stream. For large streams (>1000 active connections):

1. Consider Redis pub/sub for multi-instance deployments
2. Implement connection pooling/batching
3. Add WebSocket connection limits per stream

### Database Indexes

```sql
-- Efficient lookup of active participants
CREATE INDEX idx_stream_participants_session 
ON stream_participants(stream_session_id) WHERE left_at IS NULL;

-- User history queries
CREATE INDEX idx_stream_participants_user 
ON stream_participants(user_did);

-- Time-based queries
CREATE INDEX idx_stream_participants_joined 
ON stream_participants(joined_at);
```

## Future Enhancements

### LiveKit Webhook Integration

Instead of relying on client-side calls to `/join` and `/leave`, integrate LiveKit's webhook events:

- `participant_joined` webhook → call `RecordJoin`
- `participant_left` webhook → call `RecordLeave`

**Benefits:**
- More reliable (doesn't depend on client)
- Handles unexpected disconnections
- No stale "active" sessions

### Participant Metadata

Extend `stream_participants` to store:
- Connection quality metrics
- Geographic location (coarse)
- Device type
- Join source (web, mobile, embed)

### Presence Indicators

Add "typing" or "speaking" indicators:
- Extend `ParticipantStateEvent` with activity type
- Track last activity timestamp
- Broadcast activity events via WebSocket

### Analytics Dashboard

Leverage participant history for:
- Peak concurrent listeners over time
- Average session duration per participant
- Reconnection rate (quality indicator)
- Geographic distribution heatmap

## Migration Notes

### Running the Migration

```bash
# Apply migration
make migrate-up

# Verify
psql $DATABASE_URL -c "SELECT COUNT(*) FROM stream_participants;"

# Rollback if needed
make migrate-down
```

### Zero-Downtime Deployment

1. Deploy new code (includes migration)
2. Run migration during low-traffic period
3. Old code continues to work (ignores new table/column)
4. New code starts using participant tracking
5. Monitor for errors

**Backward Compatibility:**
- New fields have defaults
- Repository checks for nil before using `participantRepo`
- Graceful degradation if participant tracking fails

## Troubleshooting

### Common Issues

**Issue**: Active count doesn't match actual participants
- **Cause**: Denormalized count out of sync
- **Fix**: Run repair query:
  ```sql
  UPDATE stream_sessions
  SET active_participant_count = (
    SELECT COUNT(*) FROM stream_participants
    WHERE stream_session_id = stream_sessions.id AND left_at IS NULL
  );
  ```

**Issue**: Duplicate active participant error
- **Cause**: User trying to join from multiple devices/tabs
- **Fix**: Handle gracefully in UI - show "Already connected" message

**Issue**: WebSocket connection drops
- **Cause**: Network interruption, timeout
- **Fix**: Implement reconnection logic in frontend with exponential backoff

**Issue**: Participant stuck as "active" after disconnect
- **Cause**: Client didn't call `/leave` before disconnecting
- **Fix**: Implement background cleanup job or LiveKit webhook integration

## References

- Migration: `migrations/000021_stream_participants.up.sql`
- Models: `internal/stream/participant.go`
- Repository: `internal/stream/participant_repository.go`
- Handlers: `internal/api/stream_handlers.go`, `internal/api/participant_ws_handlers.go`
- Tests: `internal/stream/participant_repository_test.go`, `internal/stream/participant_test.go`
