# Alliance CRUD Handlers

## Overview

The alliance handlers provide HTTP endpoints for managing trust relationships between scenes. Alliances influence trust score computation and scene ranking. All operations require authentication and ownership verification.

## Endpoints

### POST /alliances

Creates a new alliance between two scenes.

**Authentication:** Required (JWT token)

**Authorization:** User must own the `from_scene_id`

**Request Body:**
```json
{
  "from_scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "to_scene_id": "660e8400-e29b-41d4-a716-446655440001",
  "weight": 0.8,
  "reason": "Long-standing collaboration on underground music events"
}
```

**Response:** `201 Created`
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "from_scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "to_scene_id": "660e8400-e29b-41d4-a716-446655440001",
  "weight": 0.8,
  "status": "active",
  "reason": "Long-standing collaboration on underground music events",
  "since": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Validation:**
- `from_scene_id`: Required, must be a valid scene UUID
- `to_scene_id`: Required, must be a valid scene UUID, must be different from `from_scene_id`
- `weight`: Required, must be between 0.0 and 1.0 (inclusive)
- `reason`: Optional, maximum 256 characters

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `invalid_weight` | Weight is not between 0.0 and 1.0 |
| 400 | `self_alliance` | Attempt to create alliance with same scene (from == to) |
| 400 | `validation_error` | Reason exceeds 256 characters |
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the from_scene |
| 404 | `not_found` | Either from_scene or to_scene not found |

**Example Error:**
```json
{
  "error": {
    "code": "invalid_weight",
    "message": "weight must be between 0.0 and 1.0"
  }
}
```

---

### GET /alliances/{id}

Retrieves an alliance by its ID.

**Authentication:** Optional

**Request Parameters:**
- `id`: Alliance UUID (path parameter)

**Response:** `200 OK`
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "from_scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "to_scene_id": "660e8400-e29b-41d4-a716-446655440001",
  "weight": 0.8,
  "status": "active",
  "reason": "Long-standing collaboration on underground music events",
  "since": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 404 | `not_found` | Alliance not found |
| 404 | `alliance_deleted` | Alliance has been deleted |

---

### PATCH /alliances/{id}

Updates an existing alliance's weight and/or reason.

**Authentication:** Required (JWT token)

**Authorization:** User must own the alliance's `from_scene_id`

**Request Parameters:**
- `id`: Alliance UUID (path parameter)

**Request Body:**
```json
{
  "weight": 0.9,
  "reason": "Updated: Expanding our collaborative reach"
}
```

**Note:** All fields are optional. Only provided fields will be updated.

**Response:** `200 OK`
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "from_scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "to_scene_id": "660e8400-e29b-41d4-a716-446655440001",
  "weight": 0.9,
  "status": "active",
  "reason": "Updated: Expanding our collaborative reach",
  "since": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:45:00Z"
}
```

**Validation:**
- `weight`: If provided, must be between 0.0 and 1.0 (inclusive)
- `reason`: If provided, maximum 256 characters

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `invalid_weight` | Weight is not between 0.0 and 1.0 |
| 400 | `validation_error` | Reason exceeds 256 characters |
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the from_scene |
| 404 | `not_found` | Alliance not found |
| 404 | `alliance_deleted` | Alliance has been deleted |

---

### DELETE /alliances/{id}

Soft-deletes an alliance. The alliance will be excluded from trust score computation and no longer retrievable.

**Authentication:** Required (JWT token)

**Authorization:** User must own the alliance's `from_scene_id`

**Request Parameters:**
- `id`: Alliance UUID (path parameter)

**Response:** `204 No Content`

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the from_scene |
| 404 | `not_found` | Alliance not found |
| 404 | `alliance_deleted` | Alliance already deleted (idempotent) |

---

## Security & Privacy

### Authorization Model
- **Create:** Only the owner of `from_scene_id` can create an alliance
- **Update:** Only the owner of `from_scene_id` can update the alliance
- **Delete:** Only the owner of `from_scene_id` can delete the alliance
- **Read:** Public (no authorization required)

### Soft Delete Behavior
When an alliance is deleted:
1. The `deleted_at` timestamp is set
2. The alliance is excluded from `GetByID` queries (returns 404)
3. The alliance is excluded from trust score computation
4. The database record is preserved for audit purposes

### Validation & Sanitization
- **Weight validation:** Strict enforcement of 0.0-1.0 range
- **HTML escaping:** Reason text is escaped to prevent XSS
- **Self-alliance prevention:** Cannot create alliance from scene to itself
- **Scene existence:** Both scenes must exist and be non-deleted

## Trust Score Integration

Alliances affect trust scores as follows:
1. **Active alliances only:** Deleted alliances are excluded
2. **Weight influence:** Higher weight = stronger trust signal
3. **Status filtering:** Only alliances with `status='active'` contribute
4. **Directional:** Trust flows from `from_scene_id` → `to_scene_id`

## Implementation Notes

### Repository Layer
- **In-memory implementation:** Provided for development/testing
- **Soft delete enforcement:** `GetByID()` returns `ErrAllianceDeleted` for deleted records
- **Idempotent operations:** Multiple deletes return appropriate error

### Testing Coverage
All endpoints have comprehensive test coverage including:
- Success cases
- Validation failures (invalid weight, reason length, self-alliance)
- Authorization failures (unauthorized user, missing authentication)
- Not found cases (missing scenes, missing alliance)
- Soft delete verification
- Idempotency checks

### Error Code Reference

| Error Code | HTTP Status | Usage |
|------------|-------------|-------|
| `invalid_weight` | 400 | Weight not in range [0.0, 1.0] |
| `self_alliance` | 400 | Attempt to ally scene with itself |
| `validation_error` | 400 | General validation failure (reason length) |
| `bad_request` | 400 | Malformed JSON or missing required fields |
| `auth_failed` | 401 | Authentication missing or invalid |
| `forbidden` | 403 | User lacks permission (not scene owner) |
| `not_found` | 404 | Alliance or scene not found |
| `alliance_deleted` | 404 | Alliance has been soft-deleted |
| `internal_error` | 500 | Server error during operation |

## Example Usage

### Creating an Alliance
```bash
curl -X POST http://localhost:8080/alliances \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "from_scene_id": "550e8400-e29b-41d4-a716-446655440000",
    "to_scene_id": "660e8400-e29b-41d4-a716-446655440001",
    "weight": 0.75,
    "reason": "Shared community values"
  }'
```

### Updating Alliance Weight
```bash
curl -X PATCH http://localhost:8080/alliances/770e8400-e29b-41d4-a716-446655440002 \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "weight": 0.9
  }'
```

### Deleting an Alliance
```bash
curl -X DELETE http://localhost:8080/alliances/770e8400-e29b-41d4-a716-446655440002 \
  -H "Authorization: Bearer $JWT_TOKEN"
```

## Future Enhancements

- **List alliances:** GET /scenes/{id}/alliances (inbound + outbound)
- **Alliance status transitions:** Support for pending/rejected states
- **Bulk operations:** Create multiple alliances atomically
- **Alliance analytics:** Metrics on alliance strength distribution
- **Reciprocal alliances:** Automatically suggest bidirectional alliances
