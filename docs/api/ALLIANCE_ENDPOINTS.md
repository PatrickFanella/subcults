# Alliance API Endpoints

## Overview

Alliance endpoints manage trust relationships between scenes. Alliances influence trust score computation and search ranking. All write operations require authentication and scene ownership verification.

## Base URL

```
/alliances
```

## Endpoints

### POST /alliances

Creates a new alliance between two scenes.

**Authentication:** Required (JWT token in `Authorization: Bearer <token>` header)

**Authorization:** User must own the `from_scene_id` scene

**Request Body:**
```json
{
  "from_scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "to_scene_id": "660e8400-e29b-41d4-a716-446655440001",
  "weight": 0.8,
  "reason": "Long-standing collaboration on underground music events"
}
```

**Field Requirements:**
- `from_scene_id` (required): UUID of the source scene
- `to_scene_id` (required): UUID of the target scene (must differ from `from_scene_id`)
- `weight` (required): Trust weight between 0.0 and 1.0 (inclusive)
- `reason` (optional): Description of alliance rationale (max 256 characters)

**Success Response:** `201 Created`
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
| 400 | `invalid_weight` | Weight not in range [0.0, 1.0] |
| 400 | `self_alliance` | Attempt to create alliance with same scene |
| 400 | `validation_error` | Reason exceeds 256 characters or other validation failure |
| 400 | `bad_request` | Malformed JSON or missing required fields |
| 401 | `auth_failed` | Authentication missing or invalid |
| 403 | `forbidden` | User does not own the from_scene |
| 404 | `not_found` | Either from_scene or to_scene not found |
| 500 | `internal_error` | Server error during operation |

---

### GET /alliances/{id}

Retrieves an alliance by its ID.

**Authentication:** Optional (public read access)

**Path Parameters:**
- `id`: Alliance UUID

**Success Response:** `200 OK`
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
| 500 | `internal_error` | Server error during retrieval |

---

### PATCH /alliances/{id}

Updates an existing alliance's weight and/or reason.

**Authentication:** Required (JWT token)

**Authorization:** User must own the alliance's `from_scene_id`

**Path Parameters:**
- `id`: Alliance UUID

**Request Body:**
```json
{
  "weight": 0.9,
  "reason": "Updated: Expanding our collaborative reach"
}
```

**Note:** All fields are optional. Only provided fields will be updated.

**Success Response:** `200 OK`
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

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `invalid_weight` | Weight not in range [0.0, 1.0] |
| 400 | `validation_error` | Reason exceeds 256 characters |
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the from_scene |
| 404 | `not_found` | Alliance not found |
| 404 | `alliance_deleted` | Alliance has been deleted |
| 500 | `internal_error` | Server error during update |

---

### DELETE /alliances/{id}

Soft-deletes an alliance. The alliance is excluded from trust score computation and becomes non-retrievable.

**Authentication:** Required (JWT token)

**Authorization:** User must own the alliance's `from_scene_id`

**Path Parameters:**
- `id`: Alliance UUID

**Success Response:** `204 No Content`

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 401 | `auth_failed` | Authentication required |
| 403 | `forbidden` | User does not own the from_scene |
| 404 | `not_found` | Alliance not found |
| 404 | `alliance_deleted` | Alliance already deleted (idempotent) |
| 500 | `internal_error` | Server error during deletion |

---

## Trust Score Integration

Alliances influence trust scores according to the following formula:

```
trust_score = avg(alliance_weights) * avg(membership_trust_weights * role_multipliers)
```

### Alliance Weight Influence

- **Active alliances only**: Deleted alliances are excluded from computation
- **Weight range**: 0.0 to 1.0 (higher = stronger trust signal)
- **Status filtering**: Only alliances with `status='active'` contribute
- **Directional**: Trust flows from `from_scene_id` → `to_scene_id`
- **Default behavior**: If a scene has no alliances, the average defaults to 1.0

### Recomputation Triggers

The trust recompute job runs periodically (default: 30 seconds) and processes scenes marked as "dirty" after alliance changes:
- Creating a new alliance marks the `from_scene_id` as dirty
- Updating an alliance marks the `from_scene_id` as dirty
- Deleting an alliance marks the `from_scene_id` as dirty

**Configuration Environment Variables:**
- `TRUST_RECOMPUTE_INTERVAL`: Duration between recompute cycles (e.g., `30s`, `1m`)
- `TRUST_RECOMPUTE_TIMEOUT`: Timeout for each recompute cycle (e.g., `30s`)

---

## Example Usage

### Creating an Alliance

```bash
curl -X POST https://api.subcults.com/alliances \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "from_scene_id": "550e8400-e29b-41d4-a716-446655440000",
    "to_scene_id": "660e8400-e29b-41d4-a716-446655440001",
    "weight": 0.75,
    "reason": "Shared community values"
  }'
```

### Retrieving an Alliance

```bash
curl https://api.subcults.com/alliances/770e8400-e29b-41d4-a716-446655440002
```

### Updating Alliance Weight

```bash
curl -X PATCH https://api.subcults.com/alliances/770e8400-e29b-41d4-a716-446655440002 \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "weight": 0.9
  }'
```

### Deleting an Alliance

```bash
curl -X DELETE https://api.subcults.com/alliances/770e8400-e29b-41d4-a716-446655440002 \
  -H "Authorization: Bearer $JWT_TOKEN"
```

---

## Security Considerations

### Authorization Model
- **Create**: Only the owner of `from_scene_id` can create alliances
- **Update**: Only the owner of `from_scene_id` can update alliances
- **Delete**: Only the owner of `from_scene_id` can delete alliances
- **Read**: Public access (no authorization required)

### Input Validation
- **Weight bounds**: Strictly enforced [0.0, 1.0] range
- **HTML escaping**: Reason text is sanitized to prevent XSS attacks
- **Self-alliance prevention**: Cannot create alliance from scene to itself
- **Scene existence**: Both scenes must exist and be non-deleted
- **Reason length**: Maximum 256 characters

### Soft Delete Behavior
When an alliance is deleted:
1. The `deleted_at` timestamp is set
2. The alliance becomes non-retrievable via `GET /alliances/{id}`
3. The alliance is excluded from trust score computation
4. The database record is preserved for audit purposes
5. Idempotent: Deleting an already-deleted alliance returns 404

---

## Related Documentation

- **Alliance Handlers**: `/docs/ALLIANCE_HANDLERS.md` - Handler implementation details
- **Trust Handlers**: `/docs/TRUST_HANDLERS.md` - Trust score API documentation
- **Trust Graph**: `/internal/trust/model.go` - Trust computation formula
- **Trust Job**: `/internal/trust/job.go` - Recompute job implementation
