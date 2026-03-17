# Organizer Stream Controls API

This document describes the API endpoints for organizer controls over LiveKit streams.

## Overview

Stream organizers (scene/event owners) can manage their live streams using the following control operations:
- Mute/unmute individual participants
- Kick (remove) participants from the stream
- Set a featured/spotlighted participant
- Lock the stream to prevent new participants from joining

All endpoints require authentication and verify that the requester is the stream host.

## Authentication

All endpoints require a valid JWT token with the user's DID. The token should be included in the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

## Endpoints

### Mute Participant

Mute or unmute a participant's audio in the stream.

**Endpoint:** `POST /streams/{stream_id}/participants/{participant_id}/mute`

**Authorization:** Stream host only

**Request Body:**
```json
{
  "muted": true
}
```

**Parameters:**
- `muted` (boolean, required): `true` to mute, `false` to unmute

**Response (200 OK):**
```json
{
  "stream_id": "abc123",
  "participant_id": "user-xyz789",
  "muted": true,
  "tracks_muted": 1
}
```

**Error Responses:**
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Only the stream host can mute participants
- `404 Not Found`: Stream session or participant not found
- `503 Service Unavailable`: LiveKit room service not configured

**Example:**
```bash
curl -X POST "https://api.subcults.app/streams/abc123/participants/user-xyz789/mute" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"muted": true}'
```

---

### Kick Participant

Remove a participant from the stream.

**Endpoint:** `POST /streams/{stream_id}/participants/{participant_id}/kick`

**Authorization:** Stream host only

**Request Body:** None

**Response (200 OK):**
```json
{
  "stream_id": "abc123",
  "participant_id": "user-xyz789",
  "status": "kicked"
}
```

**Error Responses:**
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Only the stream host can kick participants
- `404 Not Found`: Stream session or participant not found
- `503 Service Unavailable`: LiveKit room service not configured

**Example:**
```bash
curl -X POST "https://api.subcults.app/streams/abc123/participants/user-xyz789/kick" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### Set Featured Participant

Set or clear the featured (spotlighted) participant in the stream. Featured participants are highlighted in the UI.

**Endpoint:** `PATCH /streams/{stream_id}/featured_participant`

**Authorization:** Stream host only

**Request Body:**
```json
{
  "participant_id": "user-xyz789"
}
```

To clear the featured participant, send `null`:
```json
{
  "participant_id": null
}
```

**Parameters:**
- `participant_id` (string, nullable): Participant ID to feature, or `null` to clear

**Response (200 OK):**
```json
{
  "stream_id": "abc123",
  "featured_participant": "user-xyz789"
}
```

**Error Responses:**
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Only the stream host can set featured participant
- `404 Not Found`: Stream session not found

**Example:**
```bash
# Set featured participant
curl -X PATCH "https://api.subcults.app/streams/abc123/featured_participant" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"participant_id": "user-xyz789"}'

# Clear featured participant
curl -X PATCH "https://api.subcults.app/streams/abc123/featured_participant" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"participant_id": null}'
```

---

### Lock Stream

Lock or unlock the stream to prevent new participants from joining.

**Endpoint:** `PATCH /streams/{stream_id}/lock`

**Authorization:** Stream host only

**Request Body:**
```json
{
  "locked": true
}
```

**Parameters:**
- `locked` (boolean, required): `true` to lock, `false` to unlock

**Response (200 OK):**
```json
{
  "stream_id": "abc123",
  "locked": true
}
```

**Error Responses:**
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Only the stream host can lock the stream
- `404 Not Found`: Stream session not found

**Example:**
```bash
# Lock stream
curl -X PATCH "https://api.subcults.app/streams/abc123/lock" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"locked": true}'

# Unlock stream
curl -X PATCH "https://api.subcults.app/streams/abc123/lock" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"locked": false}'
```

---

## Audit Logging

All organizer control actions are logged to the audit system with the following action types:
- `muted` - Participant audio was muted
- `unmuted` - Participant audio was unmuted
- `kicked` - Participant was removed from stream
- `featured_participant_set` - Featured participant was set
- `featured_participant_cleared` - Featured participant was cleared
- `locked` - Stream was locked
- `unlocked` - Stream was unlocked

Audit entries include:
- User DID (who performed the action)
- Entity type and ID
- Action type
- Request ID for tracing
- Timestamp

---

## Configuration

The organizer controls require LiveKit server to be configured with the following environment variables:

```bash
# LiveKit server URL
LIVEKIT_URL=wss://livekit.example.com

# LiveKit API credentials
LIVEKIT_API_KEY=your-api-key
LIVEKIT_API_SECRET=your-api-secret
```

If LiveKit is not configured, control endpoints will return `503 Service Unavailable`.

---

## Privacy Considerations

1. **No PII Exposure**: The API does not expose personally identifiable information about participants
2. **Host-Only Access**: Only the stream host (scene/event owner) can use these controls
3. **Audit Trail**: All actions are logged for accountability and debugging
4. **Participant IDs**: Use deterministic, privacy-preserving identifiers derived from DIDs

---

## Integration Example

Here's a complete example of managing a stream as an organizer:

```javascript
const API_BASE = 'https://api.subcults.app';
const token = 'your-jwt-token';

// Mute a disruptive participant
async function muteParticipant(streamId, participantId) {
  const response = await fetch(
    `${API_BASE}/streams/${streamId}/participants/${participantId}/mute`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ muted: true }),
    }
  );
  return response.json();
}

// Set featured speaker
async function setFeaturedSpeaker(streamId, participantId) {
  const response = await fetch(
    `${API_BASE}/streams/${streamId}/featured_participant`,
    {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ participant_id: participantId }),
    }
  );
  return response.json();
}

// Lock stream after main talk begins
async function lockStream(streamId) {
  const response = await fetch(
    `${API_BASE}/streams/${streamId}/lock`,
    {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ locked: true }),
    }
  );
  return response.json();
}

// Example usage
const streamId = 'abc123';
await muteParticipant(streamId, 'user-disruptive');
await setFeaturedSpeaker(streamId, 'user-speaker');
await lockStream(streamId);
```

---

## Error Handling

All endpoints use standard HTTP status codes and return JSON error responses:

```json
{
  "error": {
    "code": "forbidden",
    "message": "Only the stream host can mute participants"
  }
}
```

Common error codes:
- `auth_failed` - Authentication required or invalid
- `forbidden` - Insufficient permissions
- `not_found` - Resource not found
- `bad_request` - Invalid request format
- `validation` - Invalid parameters
- `internal` - Server error

---

## Rate Limiting

Organizer control endpoints are subject to rate limiting to prevent abuse:
- Per-endpoint limits: TBD
- Global limits per user: TBD

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1609459200
```
