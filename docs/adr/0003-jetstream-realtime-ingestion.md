# ADR-003: Jetstream Real-Time Ingestion vs Periodic Polling

**Status:** Accepted
**Date:** 2025-12-01

## Context

Subcults needs to ingest scene, event, and post records from the AT Protocol network. Two approaches were considered: subscribing to a real-time WebSocket feed (Jetstream) or periodically polling repositories for changes.

## Decision

We use Jetstream's real-time WebSocket feed to consume AT Protocol commits as they happen. The Indexer service (`cmd/indexer`) maintains a persistent WebSocket connection with:

- Exponential backoff + jitter for reconnection
- Sequence-based cursor for crash recovery (resume from last processed event)
- Backpressure handling (pause/resume at configurable queue thresholds)

## Consequences

### Positive

- Content appears on the map within seconds of posting — core to the "real-time discovery" value proposition.
- Sequence cursors eliminate data gaps after restarts or network interruptions.
- Backpressure prevents memory exhaustion during traffic spikes.

### Negative

- WebSocket connections require health monitoring and reconnection logic.
- Requires handling out-of-order or duplicate messages (idempotency via record DID + rkey).
- Jetstream availability becomes a dependency for data freshness.

### Neutral

- The Indexer is a separate binary (`cmd/indexer`), independently scalable from the API.

## Alternatives Considered

### Alternative 1: Periodic Polling

Rejected because polling introduces latency proportional to the poll interval (minutes). For a real-time discovery platform, stale data degrades the core experience. Polling also wastes bandwidth re-fetching unchanged data.

### Alternative 2: AT Protocol Firehose Direct

Rejected because the raw firehose includes all record types across all users. Jetstream provides filtered, typed events scoped to relevant collections, reducing processing overhead.
