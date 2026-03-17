# ADR-002: Trust Ranking Behind Feature Flag

**Status:** Accepted
**Date:** 2025-12-01

## Context

The trust-based ranking system influences search result ordering based on alliance strength and membership weights. This is a novel ranking signal without established benchmarks — deploying it without a safety mechanism risks degrading search quality for all users simultaneously.

## Decision

Trust ranking is gated behind a feature flag (`trust_ranking`). When disabled, the trust weight component of the composite search score falls back to 0.0. The search formula becomes:

```
composite_score = (text_relevance × 0.4) + (proximity × 0.3) + (recency × 0.2) + (trust_weight × 0.1)
```

With `trust_ranking = false`: trust_weight is 0.0, so scoring uses only text, proximity, and recency.

## Consequences

### Positive

- Safe rollout: trust ranking can be enabled per-environment or per-cohort.
- Quick rollback: disable the flag if ranking quality degrades — no code deploy needed.
- A/B testing: can compare search quality with and without trust signal.

### Negative

- Feature flags add conditional branches to the search path.
- Trust data is computed regardless of flag state (compute cost without benefit when off).

### Neutral

- Flag check is a single boolean evaluation in the ranking function — negligible performance impact.

## Alternatives Considered

### Alternative 1: Ship Trust Ranking Without Flag

Rejected because the trust graph is sparse during early adoption, potentially producing misleading rankings before sufficient alliance data accumulates.

### Alternative 2: Separate Ranking Endpoint

Rejected because maintaining two search endpoints (trusted vs. untrusted) increases API surface area and client complexity.
