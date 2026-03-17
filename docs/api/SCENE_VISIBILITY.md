# Scene Visibility & Privacy Enforcement

## Overview

Scene visibility controls determine who can view and discover scenes on the Subcults platform. This document describes the visibility modes, access rules, and privacy enforcement mechanisms.

## Visibility Modes

Scenes support three visibility modes that control access and discoverability:

### 1. Public (`public`)

- **Access**: Visible to all users (authenticated and unauthenticated)
- **Search**: Appears in search results and map views
- **Use Case**: Open community scenes, public events, general discovery

**Example**: A public techno scene in Berlin that welcomes all music lovers.

### 2. Members Only (`private`)

- **Access**: Visible only to:
  - The scene owner
  - Users with **active** membership status
- **Search**: Appears in search results only for authorized users
- **Use Case**: Curated communities, invite-only scenes, trust-based groups

**Example**: A private underground scene requiring membership approval.

**Note**: Database uses `private` for this mode, mapped to `VisibilityMembersOnly` in code.

### 3. Hidden (`unlisted`)

- **Access**: Visible only to the scene owner
- **Search**: **Exempt** from search results (even for members)
- **Direct Access**: Accessible via direct URL if user is the owner
- **Use Case**: Personal archives, draft scenes, private collections

**Example**: A scene being prepared for launch or a personal music collection.

**Note**: Database uses `unlisted` for this mode, mapped to `VisibilityHidden` in code.

## API Endpoints

### Get Scene

Retrieve a single scene with visibility enforcement.

```
GET /scenes/{id}
```

#### Authentication

Optional. Including a valid JWT token allows access to members-only scenes if the user is an active member.

#### Path Parameters

- `id` (string, required): Scene UUID

#### Response

##### Success (200 OK)

Returns the scene object if the requester has access:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Underground Techno Scene",
  "description": "Late-night techno sessions",
  "owner_did": "did:plc:abc123",
  "visibility": "public",
  "allow_precise": true,
  "precise_point": {
    "lat": 52.5200,
    "lng": 13.4050
  },
  "coarse_geohash": "u33dc1",
  "tags": ["techno", "electronic", "berlin"],
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

##### Not Found (404)

Returns when:
- Scene does not exist
- Scene is deleted
- User lacks permission to view the scene

**Important**: The error message is intentionally uniform to prevent user enumeration attacks.

```json
{
  "error": {
    "code": "not_found",
    "message": "Scene not found"
  }
}
```

The same error is returned for:
- Non-existent scenes
- Members-only scenes accessed by non-members
- Hidden scenes accessed by non-owners

This prevents attackers from discovering which scenes exist.

## Access Control Rules

### Public Scenes

```
if visibility == "public":
    allow access to everyone
```

### Members-Only Scenes

```
if visibility == "private":
    if requester is owner:
        allow access
    else if requester is active member:
        allow access
    else:
        deny access (return 404)
```

**Membership Requirements**:
- Membership must exist in the `memberships` table
- Membership status must be `"active"`
- Pending (`"pending"`) or rejected (`"rejected"`) memberships do NOT grant access

### Hidden Scenes

```
if visibility == "unlisted":
    if requester is owner:
        allow access
    else:
        deny access (return 404)
```

## Privacy Enforcement

### Location Privacy

All scenes automatically enforce location privacy through the repository layer:

- If `allow_precise` is `false`, the `precise_point` field is set to `NULL` before storage
- This is enforced by the `EnforceLocationConsent()` method called by the repository
- The `coarse_geohash` field is always present for privacy-conscious discovery

**Example**:

```go
scene := &Scene{
    AllowPrecise: false,
    PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060}, // Will be cleared
}

// After repository Insert/Update
// scene.PrecisePoint == nil (privacy enforced)
```

### Logging

Privacy enforcement events are logged at **debug level only** to avoid leaking information:

```go
slog.DebugContext(ctx, "scene access denied", 
    "scene_id", sceneID, 
    "visibility", foundScene.Visibility,
    "requester_did", requesterDID)
```

Production logs (info and above) do not reveal:
- Whether a scene exists
- The visibility mode of inaccessible scenes
- Who attempted to access which scenes

## Security Considerations

### User Enumeration Prevention

All forbidden access returns the same `404 Not Found` response as non-existent scenes. This prevents:

- Discovering which scenes exist by trying different IDs
- Determining scene visibility modes through error messages
- Building a database of scenes by brute force

### Timing Attack Prevention

The implementation uses consistent error paths:

1. Retrieve scene from database
2. Check access permissions
3. Return uniform `404` error if unauthorized

This prevents timing analysis from revealing whether a scene exists.

### Membership Verification

Members-only scenes verify:
- Membership record exists
- Membership status is exactly `"active"`
- No other statuses grant access

This ensures only explicitly approved members can access restricted scenes.

## Database Schema

The visibility mode is enforced by a database CHECK constraint:

```sql
ALTER TABLE scenes ADD CONSTRAINT chk_scene_visibility
    CHECK (visibility IN ('public', 'private', 'unlisted'));
```

Default value is `'public'` if not specified.

## Code Examples

### Creating a Members-Only Scene

```go
scene := &scene.Scene{
    Name:          "Secret Underground Scene",
    OwnerDID:      "did:plc:owner123",
    Visibility:    scene.VisibilityMembersOnly, // Maps to "private"
    CoarseGeohash: "u33dc1",
}

err := repo.Insert(scene)
```

### Checking Scene Access

```go
// In handler
requesterDID := middleware.GetUserDID(r.Context())

canAccess, err := handlers.canAccessScene(ctx, foundScene, requesterDID)
if !canAccess {
    // Return uniform 404 error
    WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
    return
}
```

### Granting Membership

```go
membership := &membership.Membership{
    SceneID: sceneID,
    UserDID: userDID,
    Status:  "active", // Required for access
}

_, err := membershipRepo.Upsert(membership)
```

## Testing

Comprehensive test coverage includes:

1. **Public scenes**: Accessible by all users
2. **Members-only scenes**:
   - Denied for non-members
   - Denied for pending members
   - Allowed for active members
   - Always allowed for owner
3. **Hidden scenes**:
   - Allowed only for owner
   - Denied for all other users
4. **Privacy enforcement**: `precise_point` cleared when `allow_precise=false`
5. **Uniform errors**: Same error for forbidden and non-existent scenes

Run tests:

```bash
go test ./internal/api/... -run TestGetScene
```

## Future Enhancements

### Search Integration

Hidden scenes will be **exempt from search endpoints**:

```go
// In search handler
WHERE deleted_at IS NULL 
  AND visibility != 'unlisted'  // Exclude hidden scenes
```

### Role-Based Access

Future enhancement may include role-based permissions within members-only scenes:

- `admin`: Full management permissions
- `curator`: Content moderation
- `member`: Basic access

### Visibility History

Track visibility changes for audit trails:

```sql
CREATE TABLE scene_visibility_history (
    id UUID PRIMARY KEY,
    scene_id UUID REFERENCES scenes(id),
    old_visibility TEXT,
    new_visibility TEXT,
    changed_by VARCHAR(255),
    changed_at TIMESTAMPTZ
);
```

## Related Documentation

- [Privacy Principles](/docs/PRIVACY.md)
- [Scene Palette Endpoint](/docs/api/PALETTE_ENDPOINT.md)
- [Membership API](/internal/api/MEMBERSHIP_API.md)
