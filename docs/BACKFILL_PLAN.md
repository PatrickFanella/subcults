# Backfill Plan: Historical AT Protocol Data Ingestion

## Overview

This document details the process for ingesting historical AT Protocol data into the Subcults database. Historical data arrives via two sources: **Jetstream backfill endpoints** (replaying recent events) and **CAR file archives** (complete repository snapshots). The backfill process reuses the existing `internal/indexer` pipeline to maintain consistency with live ingestion.

## Scope

### Entity Types (Priority Order)

1. **Scenes** (`app.subcult.scene`) — core community containers; required first for FK integrity
2. **Events** (`app.subcult.event`) — depend on scenes
3. **Posts** (`app.subcult.post`) — depend on scenes and optionally events
4. **Alliances** (`app.subcult.alliance`) — scene-to-scene trust links

### Time Range Strategy

- **Recent-first**: Prioritize the most recent 90 days, then backfill older data in reverse chronological order.
- **Rationale**: Recent data is most relevant to active users; older data can be ingested during off-peak hours.
- **Trust graph dependencies**: Alliance records must follow scene records. Process all scenes for a time window before alliances.

## Data Sources

### 1. Jetstream Backfill Endpoints

- Replay recent commits by specifying a `cursor` (timestamp in microseconds).
- Reuse existing `internal/indexer.Client` with a custom start cursor.
- Supports incremental backfill: resume from last checkpoint if interrupted.
- Subject to the same CBOR parsing, filtering, and validation pipeline.

### 2. CAR File Archives

- CAR v1 files containing IPLD-encoded repository blocks.
- Obtained from AT Protocol PDS exports or relay archives.
- Parsed via streaming reader to avoid loading entire files into memory.
- Each block validated against expected CID hash before processing.

## Processing Pipeline

```
Source (Jetstream / CAR)
    │
    ▼
┌──────────────────────┐
│  RecordFilter         │  Existing filter: lexicon check, field validation
│  (internal/indexer)   │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  Idempotency Check   │  CheckIdempotencyKey() — skip already-ingested records
│  (RecordRepository)  │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  Consent Enforcement  │  EnforceLocationConsent() — clear PrecisePoint if needed
│  (internal/scene)    │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  UpsertRecord         │  Transactional insert/update with FK enforcement
│  (RecordRepository)  │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  SequenceTracker      │  Update checkpoint cursor for resume
└──────────────────────┘
```

## Resource Throttling

| Parameter          | Default     | Notes                                     |
| ------------------ | ----------- | ----------------------------------------- |
| Batch size         | 500 records | Records per transaction commit            |
| QPS limit          | 100 req/s   | Against Jetstream endpoints               |
| Concurrent workers | 1           | Single-threaded to preserve ordering      |
| Memory ceiling     | 256 MB      | Streaming parser; no full-file buffering  |
| DB connection pool | 5           | Dedicated pool separate from live indexer |

### Backoff Strategy

- On transient DB errors: exponential backoff starting at 100ms, max 30s, with 50% jitter.
- On Jetstream 429: respect `Retry-After` header, minimum 5s delay.
- On CAR block read errors: skip corrupt block, log to audit, continue.

## Retry Policies

1. **Batch-level retry**: If a batch fails, retry the entire batch up to 3 times before skipping.
2. **Record-level skip**: Individual corrupt or invalid records are skipped and logged (never block the batch).
3. **Source-level resume**: The backfill command persists a checkpoint after each successful batch. On restart, it resumes from the last checkpoint.

## Idempotency

- Every record is keyed by `(DID, collection, rkey)` — the natural AT Protocol composite key.
- The existing `CheckIdempotencyKey()` and `UpsertRecord()` in `RecordRepository` handle deduplication.
- CAR imports generate an idempotency key from `sha256(DID + collection + rkey + CID)`.
- Re-running a backfill is safe: duplicate records are detected and skipped.

## Integrity Checks

### During Ingestion

- **CID validation**: Every CAR block's CID is recomputed and compared to the declared CID.
- **Schema validation**: Records pass through the same `RecordFilter` as live data.
- **FK integrity**: Scenes must exist before events/posts referencing them. Use `ON CONFLICT DO NOTHING` for out-of-order records, then re-process orphans in a second pass.

### Post-Ingestion Verification

- Run the consistency verification tool (`./bin/api consistency-check --sample-size=1000`).
- Compare sampled records against AT Protocol source data.
- Log any mismatches for manual review or automated re-indexing.

## Privacy Requirements

- **Location consent**: All ingested scene/event records must pass through `EnforceLocationConsent()` before persistence. The backfill pipeline calls this identically to the live path.
- **Redacted records**: If a record's DID appears in a takedown list, skip ingestion entirely.
- **Audit logging**: Every backfill batch logs record counts and error counts to the audit log. No PII (DID values, content text) appears in logs — only aggregate counts and record IDs.
- **Jitter application**: Public coordinates receive geohash-based jitter on read, not on write. Backfill writes raw consented data; jitter is applied at API response time.

## Rollback Plan

### Before Starting

1. Record current database state: `SELECT count(*) FROM scenes; SELECT count(*) FROM events; ...`
2. Create a database snapshot (Neon branch or `pg_dump` of affected tables).

### During Backfill

- Each batch is atomic (single transaction). Failed batches leave no partial state.
- The checkpoint cursor tracks exactly which records have been committed.

### Reverting a Backfill

1. Identify the backfill time window from checkpoint metadata.
2. Delete records where `created_at >= backfill_start AND source = 'backfill'`.
3. Reset the backfill checkpoint to the pre-backfill cursor value.
4. Verify record counts match pre-backfill snapshot.

## Monitoring

- **Prometheus metrics**: `backfill_records_processed_total`, `backfill_records_skipped_total`, `backfill_errors_total`, `backfill_batch_duration_seconds`.
- **Alerts**: Stall detection (no progress for 30+ minutes), error rate > 5%, queue depth > 100K.
- **Structured logs**: Per-batch summary with `batch_id`, `records_processed`, `records_skipped`, `duration_ms`.

## CLI Usage

```bash
# Dry run — show batch segmentation without writing
./bin/backfill --source=jetstream --start-ts=2025-01-01T00:00:00Z --end-ts=2025-06-01T00:00:00Z --batch=500 --dry-run

# Live Jetstream backfill
./bin/backfill --source=jetstream --start-ts=2025-01-01T00:00:00Z --end-ts=2025-06-01T00:00:00Z --batch=500

# CAR file import
./bin/backfill --source=car --car-path=/data/exports/repo.car --batch=500

# Resume interrupted backfill
./bin/backfill --source=jetstream --resume
```

## Scheduling

| Phase | Window                     | Entities     | Expected Volume   |
| ----- | -------------------------- | ------------ | ----------------- |
| 1     | Off-peak (02:00–06:00 UTC) | Scenes       | ~10K records      |
| 2     | Off-peak                   | Events       | ~50K records      |
| 3     | Off-peak                   | Posts        | ~200K records     |
| 4     | Any time                   | Alliances    | ~5K records       |
| 5     | Post-import                | Verification | Sample 1K records |

## Related Documents

- [Jetstream Reconnection](jetstream-reconnection.md) — resume and cursor handling
- [Backpressure](BACKPRESSURE.md) — queue management during high load
- [CBOR Parsing](CBOR_PARSING.md) — record parsing pipeline
- [Data Retention Policy](legal/DATA_RETENTION_POLICY.md) — retention periods
- [Privacy](PRIVACY.md) — location consent enforcement
