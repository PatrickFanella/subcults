# Organizer Stream Controls - Implementation Summary

## Issue
**Issue**: Organizer stream controls (mute, kick, featured participant, lock)
**Epic**: LiveKit Streaming - Complete WebRTC Audio Implementation (#306)

## Objective
Implement organizer controls for stream management, allowing scene/event owners to moderate their live audio streams.

## Implementation

### 1. New API Endpoints

Four new endpoints provide comprehensive stream moderation:

- `POST /streams/{stream_id}/participants/{participant_id}/mute` - Mute/unmute participant audio
- `POST /streams/{stream_id}/participants/{participant_id}/kick` - Remove participant from stream
- `PATCH /streams/{stream_id}/featured_participant` - Set/clear featured speaker
- `PATCH /streams/{stream_id}/lock` - Lock/unlock stream to prevent new joins

### 2. LiveKit Room Service

Created `RoomService` wrapper (`internal/livekit/room_service.go`) providing:
- Connection to LiveKit server via server SDK v2
- Methods for muting tracks, removing participants, updating metadata
- Graceful handling of missing configuration

### 3. Data Model Updates

Extended `Session` model with:
- `IsLocked` - boolean flag to prevent new joins
- `FeaturedParticipant` - optional participant ID for spotlighting

Added repository methods:
- `SetLockStatus(id string, locked bool) error`
- `SetFeaturedParticipant(id string, participantID *string) error`

### 4. Security & Authorization

All endpoints implement:
- JWT authentication
- Host verification (session.HostDID == userDID)
- 403 Forbidden for unauthorized attempts

### 5. Audit Logging

All organizer actions are logged with appropriate action types (muted, unmuted, kicked, locked, unlocked, featured_participant_set, featured_participant_cleared).

### 6. Testing

Unit tests cover:
- Authorization checks
- Lock/unlock functionality
- Featured participant operations
- Non-host access denial

## Files Modified/Created

**Created:**
- `internal/livekit/room_service.go`
- `internal/api/stream_organizer_test.go`
- `docs/api/organizer-stream-controls.md`

**Modified:**
- `internal/api/stream_handlers.go`
- `internal/stream/repository.go`
- `cmd/api/main.go`
- `go.mod` / `go.sum`

## Configuration Required

```bash
LIVEKIT_URL=wss://livekit.example.com
LIVEKIT_API_KEY=your-api-key
LIVEKIT_API_SECRET=your-api-secret
```

## Acceptance Criteria ✅

- [x] Controls work correctly
- [x] Only stream organizer can execute
- [x] Actions logged for audit
- [x] Authorization checks in place
- [x] Documentation provided
