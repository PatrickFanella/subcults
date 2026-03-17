# Database Migrations

This directory contains database schema migrations for the Subcults application.

## Migration Tool

Migrations are managed using [golang-migrate](https://github.com/golang-migrate/migrate).

## Prerequisites

### PostgreSQL with PostGIS

The Subcults application requires PostgreSQL with the **PostGIS** extension for geographic queries. PostGIS enables:

- Storage of geographic points (`GEOGRAPHY(Point, 4326)`)
- Spatial indexing with GIST indexes
- Proximity queries for scene/event discovery
- Coarse location privacy with geohash support

**PostGIS must be available on your PostgreSQL instance.** 

For local development, use a PostGIS-enabled Docker image:

```bash
docker run -d \
  --name subcults-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgis/postgis:16-3.4
```

For production, use a managed PostgreSQL service with PostGIS support:
- **Neon** (recommended): PostGIS is available as an extension
- **AWS RDS**: Enable PostGIS extension
- **Google Cloud SQL**: Enable PostGIS extension
- **Azure Database for PostgreSQL**: Enable PostGIS extension

### Verifying PostGIS Installation

After connecting to your database, verify PostGIS is available:

```sql
SELECT PostGIS_Version();
```

Expected output: A version string like `3.4 USE_GEOS=1 USE_PROJ=1 USE_STATS=1`

## Running Migrations

Set the `DATABASE_URL` environment variable:

```bash
export DATABASE_URL='postgres://user:pass@localhost:5432/subcults?sslmode=disable'
```

### Using Make targets (recommended)

```bash
# Apply all pending migrations
make migrate-up

# Rollback the last migration
make migrate-down
```

### Using the migration script directly

```bash
# Apply all pending migrations
./scripts/migrate.sh up

# Apply a specific number of migrations
./scripts/migrate.sh up 2

# Rollback the last migration
./scripts/migrate.sh down 1

# Check current migration version
./scripts/migrate.sh version

# Force a specific version (use with caution)
./scripts/migrate.sh force 1
```

## Migration Files

| Version | Name | Description |
|---------|------|-------------|
| 000000 | initial_schema | Core tables: scenes, events, posts, memberships, alliances, stream_sessions, indexer_state. Enables PostGIS and uuid-ossp extensions. |
| 000001 | add_allow_precise | No-op migration kept for version continuity. The allow_precise column was originally added by this migration but is now included in the initial schema. Removing it would break existing deployments that have already run this version. |
| 000002 | enable_postgis | Explicit PostGIS extension verification |
| 000003 | posts_table | Enhanced posts table: JSONB attachments, moderation labels, full-text search (FTS), scene_id/event_id association constraint. Supports feed rendering and content moderation. |
| 000004 | users_table | Users table for core identity and ATProto DID linking. Foundation for ownership and membership relations. |
| 000005 | events_table | Enhanced events table: title (renamed from name), tags array, status with CHECK constraint, stream_session_id FK, full-text search (FTS) on title+tags. Supports schedule-based discovery. |
| 000006 | trust_graph_columns | Adds trust_weight (0-1) and since columns to memberships. Adds reason, status, and since columns to alliances. Indexes on weight and status for filtering. Enables trust graph computation. |
| 000007 | audit_logs | Audit logs table for privacy-compliant access logging. Records scene/event/post access with retention policies. |
| 000008 | at_protocol_record_keys | Adds record_did and record_rkey columns to all entity tables (scenes, events, posts, memberships, alliances, stream_sessions) with unique constraints for idempotent ingestion. Maps (did + rkey) to internal UUID for upsert operations. |
| 000009 | enhance_scenes_table | Enhances scenes table with tags array, visibility CHECK constraint (public/private/unlisted), palette JSONB (replaces primary_color/secondary_color), owner_user_id FK, and FTS generated column. Makes coarse_geohash NOT NULL for privacy-conscious discovery. |
| 000010-000025 | Various | Additional migrations for scene names, event RSVPs, stream analytics, payment records, webhook events, idempotency keys, stream participants, and quality metrics. |
| 000026 | add_fts_immutable_wrapper | Creates IMMUTABLE wrapper function for to_tsvector to enable GIN indexes on full-text search expressions. Adds FTS GIN indexes on scenes (name+description+tags), events (title+tags), and posts (text). Resolves PostgreSQL immutability constraints that prevented direct FTS indexing. |

## Writing New Migrations

1. Create up and down migration files with the next sequential version number:
   ```
   migrations/000003_your_change.up.sql
   migrations/000003_your_change.down.sql
   ```

2. Use `IF NOT EXISTS` / `IF EXISTS` for idempotent operations

3. Include appropriate indexes for query performance

4. Add comments explaining the migration's purpose

5. Test both up and down migrations locally before committing

## Schema Overview

### Core Tables

- **users**: Core identity with optional ATProto DID linking
- **scenes**: Underground music scenes with privacy-controlled location data
- **events**: Temporal happenings within scenes
- **posts**: Content within scenes/events
- **memberships**: Scene participation (member, curator, admin roles) with trust_weight (0-1) for trust scoring
- **alliances**: Trust relationships between scenes with weight (0-1), reason, and status
- **stream_sessions**: LiveKit audio rooms
- **indexer_state**: Cursor tracking for Jetstream ingestion
- **audit_logs**: Privacy-compliant access logging with retention policies

### Location Privacy

All location-aware tables enforce a consent model:

```sql
CONSTRAINT chk_precise_consent CHECK (
    allow_precise = TRUE OR precise_point IS NULL
)
```

- `allow_precise = FALSE`: Only coarse location (geohash) is stored
- `allow_precise = TRUE`: Precise coordinates may be stored

See `internal/scene/model.go` for the Go-side enforcement via `EnforceLocationConsent()`.

## Full-Text Search (FTS)

The database includes GIN indexes for full-text search on key entity fields:

### Searchable Fields

- **scenes**: `name + description + tags`
- **events**: `title + tags`
- **posts**: `text`

### Using FTS in Queries

Full-text search uses the `to_tsvector_immutable()` wrapper function and PostgreSQL's `@@` operator:

```sql
-- Search scenes by keyword
SELECT * FROM scenes
WHERE to_tsvector_immutable(
    COALESCE(name, '') || ' ' ||
    COALESCE(description, '') || ' ' ||
    COALESCE(array_to_string(tags, ' '), '')
) @@ to_tsquery('english', 'techno')
AND deleted_at IS NULL;

-- Search events with multiple terms
SELECT * FROM events
WHERE to_tsvector_immutable(
    COALESCE(title, '') || ' ' ||
    COALESCE(array_to_string(tags, ' '), '')
) @@ to_tsquery('english', 'underground & electronic')
AND deleted_at IS NULL AND cancelled_at IS NULL;

-- Search posts
SELECT * FROM posts
WHERE to_tsvector_immutable(COALESCE(text, ''))
@@ to_tsquery('english', 'warehouse')
AND deleted_at IS NULL;
```

### Query Operators

- `&` - AND (both terms must match)
- `|` - OR (either term can match)
- `!` - NOT (term must not match)
- `<->` - FOLLOWED BY (terms must be adjacent)

Example: `'techno & (warehouse | underground) & !mainstream'`

### Stemming

PostgreSQL's FTS automatically handles word stemming:
- "electronic" matches "electronics", "electronically"
- "warehouse" matches "warehouses", "warehousing"

The `to_tsvector_immutable()` wrapper ensures GIN indexes can be created on FTS expressions by marking the function as IMMUTABLE with a hard-coded 'english' configuration.



## Troubleshooting

### "extension postgis is not available"

Your PostgreSQL instance does not have PostGIS installed. Either:
- Use a PostGIS-enabled Docker image
- Install PostGIS on your server
- Enable PostGIS extension in your managed database service

### Migration dirty state

If a migration fails partway through:

```bash
./scripts/migrate.sh force <version>
```

Then fix the issue and re-run the migration.

### Stream Analytics Tables (Migration 000020)

**stream_participant_events**: Granular tracking of individual join/leave events
- Records every participant join and leave event with timestamps
- Optional geohash_prefix (4 chars) for privacy-safe geographic distribution
- Used to compute detailed analytics on stream end

**stream_analytics**: Computed aggregate metrics for ended streams
- Peak concurrent listeners, unique participants, join attempts
- Stream duration, engagement lag (time to first join)
- Retention metrics: average and median listen duration
- Geographic distribution (JSONB map of 4-char geohash prefix -> count)
- Privacy-first: No PII, only aggregates

Analytics are automatically computed when a stream ends and can be viewed by the stream host via `GET /streams/{id}/analytics`.

See `docs/STREAM_ANALYTICS.md` for full API documentation.
