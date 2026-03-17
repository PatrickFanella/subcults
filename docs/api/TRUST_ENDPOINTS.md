# Trust Score API Endpoints

## Overview

The Trust Score API provides read-only access to computed trust scores and their breakdowns for scenes. Trust scores influence search ranking and discovery (when the feature flag is enabled).

## Base URL

```
/trust
```

## Endpoints

### GET /trust/{sceneId}

Retrieves the trust score and detailed breakdown for a specific scene.

**Authentication:** Optional (public read access)

**Path Parameters:**
- `sceneId`: Scene UUID

**Success Response:** `200 OK`

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

**Response Fields:**

- `scene_id` (string): UUID of the scene
- `trust_score` (number): Computed trust score between 0.0 and 1.0
- `breakdown` (object, nullable): Detailed breakdown of trust score components
  - **Note**: This field is omitted when there are no memberships
  - Values are informational summaries and do **not** represent exact internal computation steps
  - `average_alliance_weight`: Average weight of alliances (defaults to 1.0 if no alliances)
  - `average_membership_trust_weight`: Informational average of raw membership trust weights
  - `role_multiplier_aggregate`: Informational average role multiplier across all members
- `stale` (boolean): Indicates if the score needs recomputation (dirty flag set)
- `last_updated` (string, nullable): ISO 8601 timestamp of last computation (omitted if no stored score)

**Error Responses:**

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `bad_request` | Scene ID missing or invalid format |
| 404 | `scene_not_found` | Scene does not exist or has been deleted |
| 500 | `internal_error` | Server error retrieving trust score |

**Example Error:**
```json
{
  "error": {
    "code": "scene_not_found",
    "message": "Scene not found"
  }
}
```

---

## Trust Score Computation

The trust score is computed using the following formula:

```
trust_score = avg(alliance_weights) * avg(membership_trust_weights * role_multipliers)
```

### Components

#### Alliance Weights
- Range: 0.0 to 1.0
- Default: 1.0 (when scene has no alliances)
- Only active alliances contribute to the average

#### Membership Trust Weights
- Range: 0.0 to 1.0
- Represents base trust level for each member

#### Role Multipliers
- `owner`: 1.0 (highest authority)
- `curator`: 0.8 (elevated privileges)
- `member`: 0.5 (baseline)
- `guest`: 0.3 (lowest weight)

### Edge Cases

- **No memberships**: `trust_score = 0.0`, `breakdown` field is omitted
- **No alliances**: Alliance average defaults to 1.0
- **Deleted alliances**: Excluded from computation
- **Inactive alliances**: Only `status='active'` alliances contribute

---

## Stale Flag

The `stale` flag indicates whether the trust score needs recomputation:

- `true`: Scene has pending changes (memberships or alliances modified)
- `false`: Stored score is up-to-date

When `stale` is true:
1. The scene is marked as "dirty" in the recompute tracker
2. The background recompute job will process it in the next cycle
3. The returned score is from the last computation (may be outdated)

**Recompute Job Configuration:**
- **Interval**: `TRUST_RECOMPUTE_INTERVAL` environment variable (default: `30s`)
- **Timeout**: `TRUST_RECOMPUTE_TIMEOUT` environment variable (default: `30s`)
- **Trigger events**: Alliance create/update/delete, membership changes

---

## Example Usage

### Basic Request

```bash
curl https://api.subcults.com/trust/550e8400-e29b-41d4-a716-446655440000
```

### With Authentication (optional)

```bash
curl https://api.subcults.com/trust/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### JavaScript Example

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
  if (trustData.breakdown) {
    console.log('Breakdown:', trustData.breakdown);
  }
} catch (error) {
  console.error('Error:', error);
}
```

---

## Trust Ranking Integration

Trust scores influence search results when the trust ranking feature flag is enabled:

**Feature Flag:** Set via `TRUST_RANKING_ENABLED` environment variable (values: `true`, `1`, `yes`, `on`)

**Search Ranking Formula:**
```
composite_score = (text_relevance * 0.4) + (proximity_score * 0.3) + 
                  (recency * 0.2) + (trust_weight * 0.1)
```

Where:
- `trust_weight = 0.0` when feature flag is disabled
- `trust_weight = trust_score * 0.1` when feature flag is enabled

### Behavior

- **Flag enabled**: Trust scores influence search ranking
- **Flag disabled**: Trust scores are computed but don't affect ranking
- **Fallback**: If trust score unavailable, defaults to 0.0 (no influence)

---

## Security & Privacy

### No Sensitive Data
- Only aggregate trust scores are exposed
- Individual user trust weights are not returned
- Membership details are not included in responses

### Public Access
- No authentication required for read operations
- Trust scores are public information
- Rate limiting applies (standard API limits)

### Performance
- Trust scores are pre-computed and cached
- Background job recomputes dirty scores every ~30 seconds
- Queries are fast (served from in-memory store)

---

## Monitoring & Observability

### Prometheus Metrics

The trust system exposes several metrics for monitoring:

```
# Trust recompute job metrics
background_jobs_total{job_type="trust_recompute", status="success|failure"}
background_jobs_duration_seconds{job_type="trust_recompute"}
background_job_errors_total{job_type="trust_recompute", error_type="..."}

# Trust-specific metrics
trust_recompute_total
trust_recompute_duration_seconds
trust_recompute_errors_total
trust_last_recompute_timestamp
trust_last_recompute_scene_count
```

### Logs

The recompute job logs completion events with structured fields:

```json
{
  "level": "info",
  "msg": "trust recompute completed",
  "duration_seconds": 2.5,
  "scenes_processed": 150,
  "scenes_failed": 0,
  "avg_weight_variance": 0.02
}
```

---

## Related Documentation

- **Trust Handlers**: `/docs/TRUST_HANDLERS.md` - Handler implementation details
- **Alliance Endpoints**: `/docs/api/ALLIANCE_ENDPOINTS.md` - Alliance management
- **Trust Graph**: `/internal/trust/model.go` - Trust computation formula
- **Trust Job**: `/internal/trust/job.go` - Recompute job implementation
- **Search Handlers**: `/docs/SEARCH_HANDLERS.md` - Trust-weighted ranking

---

## Troubleshooting

### Trust score is always 0.0
- Check if scene has any memberships (requirement for non-zero score)
- Verify alliances exist and have `status='active'`
- Check membership trust weights are > 0.0

### Stale flag is always true
- Verify trust recompute job is running (check logs)
- Check `TRUST_RECOMPUTE_INTERVAL` is not too long
- Inspect job metrics for errors

### Trust scores not affecting search ranking
- Verify `TRUST_RANKING_ENABLED` is set to `true`
- Check feature flag initialization in logs
- Confirm trust scores are being fetched in search handler

### Scores not updating after alliance changes
- Normal: Recompute job runs every 30 seconds
- Check `stale` flag becomes `true` after changes
- Wait for next recompute cycle
- Inspect job logs for processing confirmation
