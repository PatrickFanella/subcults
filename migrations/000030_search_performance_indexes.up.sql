-- Migration: Search performance indexes for 10k+ scale
-- Adds targeted composite and partial indexes to meet search performance targets:
--   p95 latency < 300ms for first page (top 20 results)
--   p95 latency < 500ms for second page
--
-- Design rationale:
--   Existing indexes cover individual columns (visibility, precise_point, starts_at).
--   These new indexes either (a) combine filters for more selective scans or
--   (b) add columns used in ORDER BY to avoid sort steps.
--
-- To verify index usage after applying:
--   EXPLAIN (ANALYZE, BUFFERS) SELECT ...  -- actual execution plan with buffer stats
--   EXPLAIN (FORMAT JSON) SELECT ...       -- machine-readable plan for regression tests

-- ============================================
-- SCENE SEARCH INDEXES
-- ============================================

-- More selective spatial index restricted to searchable visibility states.
-- The existing idx_scenes_location covers all visibility values including 'unlisted'.
-- Excluding 'unlisted' scenes at index creation reduces scan range for SearchScenes.
--
-- Expected query plan for SearchScenes with bbox:
--   -> Index Scan using idx_scenes_search_visible on scenes
--        Index Cond: (precise_point && ST_MakeEnvelope($1,$2,$3,$4,4326)::geography)
--        Filter: (visibility IN ('public','private'))
--   Estimated improvement: ~20-40% fewer index rows vs idx_scenes_location
--   when 'unlisted' scenes represent a non-trivial fraction of the dataset.
CREATE INDEX IF NOT EXISTS idx_scenes_search_visible
    ON scenes USING GIST(precise_point)
    WHERE deleted_at IS NULL
      AND allow_precise = TRUE
      AND visibility IN ('public', 'private');

COMMENT ON INDEX idx_scenes_search_visible IS
    'Spatial index for SearchScenes, restricted to publicly searchable visibility states. '
    'More selective than idx_scenes_location; excludes unlisted scenes from the spatial scan. '
    'Expected plan: Index Scan on idx_scenes_search_visible for bbox-filtered SearchScenes.';

-- Partial index for browsing public scenes sorted by creation time.
-- Enables efficient "newest public scenes" browse without a text query or bbox.
-- Supports ORDER BY created_at DESC LIMIT N without a full-table sort.
--
-- Expected query plan for recency-sorted browse:
--   -> Index Scan Backward using idx_scenes_public_created on scenes  (cost≈0.29..8.30)
--        Filter: (deleted_at IS NULL AND visibility = 'public')
--   Rows returned in index order; no Sort node required.
CREATE INDEX IF NOT EXISTS idx_scenes_public_created
    ON scenes(created_at DESC)
    WHERE deleted_at IS NULL AND visibility = 'public';

COMMENT ON INDEX idx_scenes_public_created IS
    'Supports recency-sorted browse of public scenes without a bbox or text filter. '
    'Expected plan: Index Scan Backward on idx_scenes_public_created; no Sort required.';

-- ============================================
-- EVENT SEARCH INDEXES
-- ============================================

-- Composite index for event discovery by coarse geohash prefix and time window.
-- Covers the common SearchByBboxAndTime pattern: filter by geohash prefix first,
-- then apply the time range before the precise GIST spatial filter.
-- This reduces the candidate set fed to the GIST index scan.
--
-- Column order rationale: placing coarse_geohash first allows the index to use an
-- equality/prefix condition on that column before the starts_at range scan.
-- A btree multicolumn index cannot use trailing columns after a leading range
-- condition, so (starts_at, coarse_geohash) would leave coarse_geohash unused.
--
-- Expected query plan for geohash-prefix + time-range event search:
--   -> Index Scan using idx_events_upcoming_geohash on events
--        Index Cond: ((coarse_geohash >= $1) AND (coarse_geohash < $2)
--                     AND (starts_at >= $3) AND (starts_at <= $4))
--        Filter: (deleted_at IS NULL AND cancelled_at IS NULL)
--   Estimated improvement: avoids full GIST scan when geohash prefix and time window are selective.
CREATE INDEX IF NOT EXISTS idx_events_upcoming_geohash
    ON events(coarse_geohash, starts_at)
    WHERE deleted_at IS NULL AND cancelled_at IS NULL;

COMMENT ON INDEX idx_events_upcoming_geohash IS
    'Composite index for event discovery by coarse geohash prefix and time window. '
    'Reduces spatial scan candidates for SearchByBboxAndTime and SearchEvents queries. '
    'Expected plan: Index Scan on (coarse_geohash, starts_at) with prefix + range conditions.';

-- Composite index for scene-specific upcoming event queries.
-- Supports "upcoming events for scene X" API endpoint and scene event feed.
-- Avoids a sort step when returning events ordered by start time.
--
-- Expected query plan for scene event listing:
--   -> Index Scan using idx_events_scene_upcoming on events
--        Index Cond: ((scene_id = $1) AND (starts_at > NOW()))
--        Filter: (deleted_at IS NULL AND cancelled_at IS NULL)
--   Rows returned in starts_at order; no Sort node required.
CREATE INDEX IF NOT EXISTS idx_events_scene_upcoming
    ON events(scene_id, starts_at)
    WHERE deleted_at IS NULL AND cancelled_at IS NULL;

COMMENT ON INDEX idx_events_scene_upcoming IS
    'Composite index for scene-specific upcoming event listing with time ordering. '
    'Expected plan: Index Scan on (scene_id, starts_at); no Sort required for time-ordered results.';

-- ============================================
-- SLOW QUERY MONITORING
-- ============================================
-- Configure in postgresql.conf (or via ALTER SYSTEM) to enable slow query detection:
--
--   log_min_duration_statement = 300   -- log queries exceeding the p95 target (ms)
--   pg_stat_statements.track = 'all'   -- track all query fingerprints
--
-- Detect regressions with pg_stat_statements:
--   SELECT left(query, 120) AS query_excerpt,
--          calls,
--          round(mean_exec_time::numeric, 2) AS avg_ms,
--          round(stddev_exec_time::numeric, 2) AS stddev_ms
--   FROM pg_stat_statements
--   WHERE mean_exec_time > 100      -- surface queries approaching the 300ms target
--     AND query ILIKE '%scenes%'
--   ORDER BY mean_exec_time DESC
--   LIMIT 20;
--
-- Check index utilization for the new search indexes:
--   SELECT indexrelname, idx_scan, idx_tup_read, idx_tup_fetch
--   FROM pg_stat_user_indexes
--   WHERE indexrelname IN (
--     'idx_scenes_search_visible',
--     'idx_scenes_public_created',
--     'idx_events_upcoming_geohash',
--     'idx_events_scene_upcoming'
--   );

-- Update schema version
INSERT INTO schema_version (version, description)
VALUES (30, 'search performance indexes for scene and event discovery');
