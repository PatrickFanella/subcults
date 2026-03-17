# Trust Score API Documentation

## Overview

The Trust Score API endpoint provides read-only access to computed trust scores and their contributing factors for scenes. This enables frontend transparency and debugging of ranking behavior.

## Endpoint

```
GET /trust/{sceneId}
```

Returns the trust score and detailed breakdown for a specific scene.

## Parameters

- `sceneId` (path, required): UUID of the scene

## Response

### Success (200 OK)

```json
{
  "scene_id": "550e8400-e29b-41d4-a716-446655440000",
  "trust_score": 0.686,
  "breakdown": {
    "average_alliance_weight": 0.7,
    "average_membership_trust_weight": 0.7,
    "role_multiplier_aggregate": 1.5
  },
  "stale": false,
  "last_updated": "2026-01-21T01:45:00Z"
}
```

**Fields:**
- `scene_id`: UUID of the scene
- `trust_score`: Numeric trust score between 0.0 and 1.0
- `breakdown`: Detailed breakdown of trust score components. These values are informational summaries and do **not** represent the exact internal computation steps for `trust_score`. The breakdown field is omitted when there are no memberships.
  - `average_alliance_weight`: Average weight of alliances (defaults to 1.0 if no alliances)
  - `average_membership_trust_weight`: Informational average of raw membership trust weights across all memberships. The final `trust_score` instead uses each membership's trust weight together with its role multiplier.
  - `role_multiplier_aggregate`: Informational average role multiplier across all members. The effective contribution to `trust_score` is based on per-membership products `membership_trust_weight * role_multiplier`, which are then averaged internally.
- `stale`: Boolean indicating if the score needs recomputation (dirty flag)
- `last_updated`: ISO 8601 timestamp of when the score was last computed (omitted if no stored score)

### Error Responses

#### Scene Not Found (404)

```json
{
  "error": {
    "code": "scene_not_found",
    "message": "Scene not found"
  }
}
```

Returned when the scene does not exist or has been deleted.

#### Internal Error (500)

```json
{
  "error": {
    "code": "internal_error",
    "message": "Failed to retrieve trust score"
  }
}
```

Returned when there's an internal server error retrieving the trust score.

## Trust Score Computation

The trust score is computed using the following formula:

```
trust_score = avg(alliance_weights) * avg(membership_trust_weights * role_multipliers)
```

Where:
- Alliance weights range from 0.0 to 1.0
- Membership trust weights range from 0.0 to 1.0
- Role multipliers are:
  - `member`: 1.0
  - `curator`: 1.5
  - `admin`: 2.0

If there are no alliances, the alliance average defaults to 1.0.
If there are no memberships, the trust score is 0.0.

## Stale Flag

The `stale` flag indicates whether the trust score needs recomputation:
- `true`: The scene has pending changes (memberships or alliances) that haven't been reflected in the stored score
- `false`: The stored score is up-to-date

When `stale` is true, the system has marked the scene as dirty and it will be recomputed by the background job.

## Example Usage

### cURL

```bash
# Get trust score for a scene
curl -X GET http://localhost:8080/trust/550e8400-e29b-41d4-a716-446655440000

# With authentication (if required in future)
curl -X GET \
  -H "Authorization: Bearer ${TOKEN}" \
  http://localhost:8080/trust/550e8400-e29b-41d4-a716-446655440000
```

### JavaScript

```javascript
// Fetch trust score
async function getTrustScore(sceneId) {
  const response = await fetch(`/trust/${sceneId}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch trust score: ${response.statusText}`);
  }
  return await response.json();
}

// Usage
try {
  const trustData = await getTrustScore('550e8400-e29b-41d4-a716-446655440000');
  console.log(`Trust Score: ${trustData.trust_score}`);
  console.log(`Stale: ${trustData.stale ? 'Yes' : 'No'}`);
  console.log('Breakdown:', trustData.breakdown);
} catch (error) {
  console.error('Error:', error);
}
```

## Security & Privacy

- **No Sensitive Data**: Only aggregate trust scores are exposed; individual user trust weights are not returned
- **Public Access**: No authentication required (read-only endpoint)
- **Rate Limiting**: Standard API rate limits apply

## Related

- Trust Graph computation: `internal/trust/model.go`
- Recompute job: `internal/trust/job.go`
- Scene repository: `internal/scene/repository.go`

## Testing

Unit tests are available in `internal/api/trust_handlers_test.go` covering:
- Success case with memberships and alliances
- Scene not found (404)
- Stale flag when scene is dirty
- No stored score (computes on-the-fly)
- No memberships (returns 0.0 score)
