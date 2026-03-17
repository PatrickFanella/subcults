# Membership Request Workflow API

## Overview

The membership request workflow allows users to request membership in scenes and enables scene owners to approve or reject those requests. This implements a controlled access mechanism for scene participation.

## Endpoints

### 1. Request Membership

**Endpoint:** `POST /scenes/{sceneId}/membership/request`

**Description:** Creates a pending membership request for the authenticated user in the specified scene.

**Authentication:** Required (JWT token with user DID)

**Path Parameters:**
- `sceneId` (string, required): The UUID of the scene to request membership in

**Request Body:** None

**Success Response:**
- **Status Code:** 201 Created
- **Body:** Membership object with status "pending"

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "scene_id": "123e4567-e89b-12d3-a456-426614174000",
  "user_did": "did:plc:abc123xyz",
  "role": "member",
  "status": "pending",
  "trust_weight": 0.5,
  "since": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**

- **401 Unauthorized:** Missing or invalid authentication
  ```json
  {
    "error": {
      "code": "auth_failed",
      "message": "Authentication required"
    }
  }
  ```

- **404 Not Found:** Scene does not exist
  ```json
  {
    "error": {
      "code": "not_found",
      "message": "Scene not found"
    }
  }
  ```

- **409 Conflict:** One of:
  - User is the scene owner
  - Pending request already exists
  - User is already an active member
  
  ```json
  {
    "error": {
      "code": "conflict",
      "message": "Pending membership request already exists"
    }
  }
  ```

**Behavior:**
- Scene owners cannot request membership in their own scenes
- If a previous request was rejected, a new request can be created (updates the existing record)
- If a pending request already exists, returns 409 Conflict
- If user is already an active member, returns 409 Conflict
- Default role is "member" with trust_weight 0.5

**Audit Logging:** Creates audit log entry with action "membership_request"

---

### 2. Approve Membership

**Endpoint:** `POST /scenes/{sceneId}/membership/{userDid}/approve`

**Description:** Approves a pending membership request. Only the scene owner can approve requests.

**Authentication:** Required (must be scene owner)

**Path Parameters:**
- `sceneId` (string, required): The UUID of the scene
- `userDid` (string, required): The DID of the user whose membership to approve (URL-encoded)

**Request Body:** None

**Success Response:**
- **Status Code:** 200 OK
- **Body:** Updated membership object with status "active"

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "scene_id": "123e4567-e89b-12d3-a456-426614174000",
  "user_did": "did:plc:abc123xyz",
  "role": "member",
  "status": "active",
  "trust_weight": 0.5,
  "since": "2024-01-15T10:35:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

**Error Responses:**

- **401 Unauthorized:** Missing or invalid authentication
- **403 Forbidden:** Authenticated user is not the scene owner
  ```json
  {
    "error": {
      "code": "forbidden",
      "message": "Only scene owner can approve memberships"
    }
  }
  ```

- **404 Not Found:** Scene or membership request not found (uniform error message to prevent user enumeration)
  ```json
  {
    "error": {
      "code": "not_found",
      "message": "Membership request not found"
    }
  }
  ```

- **409 Conflict:** Membership is not in pending status
  ```json
  {
    "error": {
      "code": "conflict",
      "message": "Only pending membership requests can be approved"
    }
  }
  ```

**Behavior:**
- Only scene owners can approve memberships
- Only pending memberships can be approved
- Sets status to "active" and updates the "since" timestamp to current time
- Uses uniform error messages to prevent user enumeration attacks

**Audit Logging:** Creates audit log entry with action "membership_approve"

**Security:**
- Implements timing attack prevention with uniform error messages
- Authorization check ensures only scene owner can approve

---

### 3. Reject Membership

**Endpoint:** `POST /scenes/{sceneId}/membership/{userDid}/reject`

**Description:** Rejects a pending membership request. Only the scene owner can reject requests.

**Authentication:** Required (must be scene owner)

**Path Parameters:**
- `sceneId` (string, required): The UUID of the scene
- `userDid` (string, required): The DID of the user whose membership to reject (URL-encoded)

**Request Body:** None

**Success Response:**
- **Status Code:** 200 OK
- **Body:** Updated membership object with status "rejected"

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "scene_id": "123e4567-e89b-12d3-a456-426614174000",
  "user_did": "did:plc:abc123xyz",
  "role": "member",
  "status": "rejected",
  "trust_weight": 0.5,
  "since": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

**Error Responses:**

- **401 Unauthorized:** Missing or invalid authentication
- **403 Forbidden:** Authenticated user is not the scene owner
  ```json
  {
    "error": {
      "code": "forbidden",
      "message": "Only scene owner can reject memberships"
    }
  }
  ```

- **404 Not Found:** Scene or membership request not found (uniform error message to prevent user enumeration)
  ```json
  {
    "error": {
      "code": "not_found",
      "message": "Membership request not found"
    }
  }
  ```

- **409 Conflict:** Membership is not in pending status
  ```json
  {
    "error": {
      "code": "conflict",
      "message": "Only pending membership requests can be rejected"
    }
  }
  ```

**Behavior:**
- Only scene owners can reject memberships
- Only pending memberships can be rejected
- Sets status to "rejected" without changing the "since" timestamp
- Rejected members can submit a new request later
- Uses uniform error messages to prevent user enumeration attacks

**Audit Logging:** Creates audit log entry with action "membership_reject"

**Security:**
- Implements timing attack prevention with uniform error messages
- Authorization check ensures only scene owner can reject

---

## Membership Status Flow

```
┌─────────┐
│ pending │ ──approve──> │ active │
└─────────┘               └────────┘
     │
     │
     └──reject──> │ rejected │ ──new request──> │ pending │
                  └──────────┘                    └─────────┘
```

**Status Transitions:**
- `pending` → `active` (via approve)
- `pending` → `rejected` (via reject)
- `rejected` → `pending` (via new request)
- `active` cannot transition (permanent)

---

## Security Considerations

### 1. Enumeration Prevention

All error messages are carefully crafted to prevent user enumeration:
- Non-existent scenes return the same error as forbidden access
- Non-existent membership requests return the same error as forbidden access
- Timing attacks are mitigated by using uniform error handling

### 2. Authorization

- Only authenticated users can request membership
- Only scene owners can approve/reject memberships
- Scene owners cannot request membership in their own scenes

### 3. Audit Logging

All membership operations are logged with:
- User DID (authenticated user)
- Entity ID (membership ID)
- Action (membership_request, membership_approve, membership_reject)
- Request ID for tracing
- IP address and user agent

### 4. Idempotency

- Duplicate pending requests return 409 Conflict
- Rejected users can reapply (creates new pending request)
- Active members cannot request again

---

## Database Schema

The memberships table includes:

```sql
CREATE TABLE memberships (
    id UUID PRIMARY KEY,
    scene_id UUID NOT NULL REFERENCES scenes(id),
    user_did VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    trust_weight FLOAT DEFAULT 0.5,
    since TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE(scene_id, user_did)
);
```

**Key Constraints:**
- Unique index on (scene_id, user_did) prevents duplicate memberships
- Foreign key to scenes table with CASCADE delete
- CHECK constraint on trust_weight (0.0-1.0)

---

## Role Configuration

Default role assignments:
- New requests: `role = "member"`, `trust_weight = 0.5`
- Role can be updated by scene owner after approval
- Valid roles: `member`, `curator`, `admin`

Trust weight multipliers (from trust graph):
- `member`: 1.0x
- `curator`: 1.2x
- `admin`: 1.5x

---

## Example Usage

### Request membership in a scene

```bash
curl -X POST https://api.subcults.app/scenes/123e4567-e89b-12d3-a456-426614174000/membership/request \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json"
```

### Approve a membership request

```bash
curl -X POST https://api.subcults.app/scenes/123e4567-e89b-12d3-a456-426614174000/membership/did:plc:abc123xyz/approve \
  -H "Authorization: Bearer OWNER_JWT_TOKEN" \
  -H "Content-Type: application/json"
```

### Reject a membership request

```bash
curl -X POST https://api.subcults.app/scenes/123e4567-e89b-12d3-a456-426614174000/membership/did:plc:abc123xyz/reject \
  -H "Authorization: Bearer OWNER_JWT_TOKEN" \
  -H "Content-Type: application/json"
```

---

## Testing

Run tests with:

```bash
# All membership tests
go test -v ./internal/api/... -run Membership

# Repository tests
go test -v ./internal/membership/...

# Full test suite
make test
```

Tests cover:
- ✅ Successful membership request
- ✅ Duplicate pending request (409)
- ✅ Scene owner cannot request (409)
- ✅ Rejected user can reapply
- ✅ Successful approval (status → active)
- ✅ Unauthorized approval (403)
- ✅ Non-pending approval (409)
- ✅ Successful rejection (status → rejected)
- ✅ Unauthorized rejection (403)
- ✅ Enumeration attack prevention
