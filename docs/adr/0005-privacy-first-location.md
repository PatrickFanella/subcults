# ADR-005: Privacy-First Location Consent Model

**Status:** Accepted
**Date:** 2025-12-01

## Context

Subcults maps underground music communities. Location data is essential for discovery but is also sensitive — publishing exact venue coordinates could endanger underground/DIY spaces that operate in legal gray areas or rely on discretion. We need a location model that enables geographic discovery while protecting scene operators who don't want their exact address published.

## Decision

All location data follows a two-tier consent model:

1. **Coarse (default):** Scenes and events always store a `coarse_geohash` (6 characters, ~±610m accuracy). Public map display uses jittered coordinates (~250m deterministic offset).

2. **Precise (opt-in):** Only when `allow_precise = true` is the exact `precise_point` (PostGIS `GEOGRAPHY(Point, 4326)`) persisted and available for proximity queries.

Enforcement is layered:

| Layer      | Mechanism                                                                    |
| ---------- | ---------------------------------------------------------------------------- |
| Model      | `EnforceLocationConsent()` clears `PrecisePoint` when `AllowPrecise = false` |
| Repository | All persist methods call `EnforceLocationConsent()` before write             |
| Database   | CHECK constraint: `allow_precise = FALSE → precise_point IS NULL`            |
| API        | Jitter applied before returning coordinates in public responses              |

## Consequences

### Positive

- Scene operators control their own location exposure — consent is explicit, not assumed.
- Database constraints make it impossible to accidentally store precise data without consent.
- Deterministic jitter means map markers are stable (no visual jitter on refresh) while still protecting exact location.
- Default-safe: new scenes are imprecise unless the creator opts in.

### Negative

- Proximity-based search is degraded for non-precise scenes (can only match on geohash prefix).
- Additional code complexity in every location-touching code path (model, repo, API).
- Jitter function must be deterministic — changing the algorithm shifts all existing markers.

### Neutral

- Geohash precision (6 chars) is a tunable parameter if more/less accuracy is needed later.

## Alternatives Considered

### Alternative 1: Always Precise, Access-Controlled

Store exact locations for all scenes but restrict who can see them (e.g., only members). Rejected because data at rest is still a liability — a database breach exposes all locations regardless of access controls.

### Alternative 2: User-Specified Radius

Let scene creators define a custom "fuzzing radius." Rejected because it shifts the privacy burden to users who may not understand the implications. A consistent system-wide policy is safer and more predictable.
