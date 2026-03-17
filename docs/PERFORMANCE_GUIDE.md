# Performance Guide & Budgets

This document defines quantitative performance budgets, measurement sources, enforcement mechanisms, and the escalation workflow for regressions.

## Performance Budgets

### API Latency

| Endpoint Class             | p50 Target | p95 Target | p99 Target | Measurement                                |
| -------------------------- | ---------- | ---------- | ---------- | ------------------------------------------ |
| Health/readiness           | <10ms      | <50ms      | <100ms     | Prometheus `http_request_duration_seconds` |
| Scene CRUD                 | <50ms      | <200ms     | <500ms     | Prometheus `http_request_duration_seconds` |
| Search (FTS)               | <100ms     | <300ms     | <800ms     | Prometheus `http_request_duration_seconds` |
| Ranking query              | <50ms      | <150ms     | <400ms     | Prometheus `http_request_duration_seconds` |
| Feed aggregation           | <80ms      | <250ms     | <600ms     | Prometheus `http_request_duration_seconds` |
| Auth (token issue/refresh) | <30ms      | <100ms     | <200ms     | Prometheus `http_request_duration_seconds` |
| File upload (R2)           | <500ms     | <2s        | <5s        | Prometheus `http_request_duration_seconds` |

### Database Query Latency

| Query Class | Threshold | Action                                                                  |
| ----------- | --------- | ----------------------------------------------------------------------- |
| Normal      | <100ms    | No action                                                               |
| Slow        | 100ms–5s  | Logged at WARN level, `db_slow_queries_total` counter incremented       |
| Very slow   | >5s       | Logged at ERROR level, `db_very_slow_queries_total` counter incremented |

Configured via `SLOW_QUERY_THRESHOLD_MS` (default: 100) and `VERY_SLOW_QUERY_THRESHOLD_MS` (default: 5000). See `internal/db/slowquery.go`.

### Streaming

| Metric                  | Target | Measurement                         |
| ----------------------- | ------ | ----------------------------------- |
| Stream join time        | <2s    | `stream_join_time_ms`               |
| Publish-to-first-packet | <500ms | `stream_publish_to_first_packet_ms` |
| Audio codec latency     | <100ms | LiveKit dashboard                   |
| Reconnection time       | <3s    | `stream_reconnect_duration_ms`      |

### Frontend (Core Web Vitals)

| Metric                         | Target | Measurement                       |
| ------------------------------ | ------ | --------------------------------- |
| First Contentful Paint (FCP)   | <1.0s  | Lighthouse CI (`lighthouserc.js`) |
| Largest Contentful Paint (LCP) | <2.5s  | Lighthouse CI                     |
| Cumulative Layout Shift (CLS)  | <0.1   | Lighthouse CI                     |
| Time to First Byte (TTFB)      | <600ms | Lighthouse CI                     |
| Total Blocking Time (TBT)      | <200ms | Lighthouse CI                     |
| Speed Index                    | <3.0s  | Lighthouse CI                     |
| Time to Interactive (TTI)      | <3.8s  | Lighthouse CI                     |
| Map initial render             | <1.2s  | Playwright performance marks      |

### Bundle Size

| Resource           | Budget      | Enforcement                                      |
| ------------------ | ----------- | ------------------------------------------------ |
| JavaScript (total) | <300KB gzip | Lighthouse CI `resource-summary:script:size`     |
| CSS (total)        | <50KB gzip  | Lighthouse CI `resource-summary:stylesheet:size` |
| HTML document      | <20KB       | Lighthouse CI `resource-summary:document:size`   |
| Images (total)     | <500KB      | Lighthouse CI `resource-summary:image:size`      |
| Total payload      | <1MB        | Lighthouse CI `resource-summary:total:size`      |

### Background Jobs

| Job                         | Target Duration | Frequency                   |
| --------------------------- | --------------- | --------------------------- |
| Trust recompute             | <5 min          | Adaptive (based on DB load) |
| Jetstream cursor checkpoint | <10ms           | Every 1000 commits          |
| Feed cache refresh          | <500ms          | On-demand with 15s TTL      |

## Measurement Sources

### Prometheus Metrics

Backend metrics are instrumented via `internal/middleware/metrics.go`:

- `http_request_duration_seconds` — histogram with method, path, status labels
- `http_requests_total` — counter with method, path, status labels
- `db_query_duration_seconds` — histogram with 12 buckets (1ms to 10s)
- `db_slow_queries_total` — counter for queries exceeding slow threshold
- `db_very_slow_queries_total` — counter for queries exceeding very-slow threshold

### Lighthouse CI

Frontend budgets are enforced via `lighthouserc.js` at the repo root. Run locally:

```bash
npx @lhci/cli@0.14.x autorun
```

### k6 Load Tests

Located in `perf/k6/`. Run scenarios:

```bash
k6 run perf/k6/stream-load-test.js
```

### EXPLAIN Baselines

Critical query plans stored in `perf/baselines/` (see #167). Capture with:

```bash
go run scripts/capture-explain.go
```

## Threshold Configuration

Budgets map to environment variables with sensible defaults:

| Variable                       | Default | Description                  |
| ------------------------------ | ------- | ---------------------------- |
| `SLOW_QUERY_THRESHOLD_MS`      | 100     | Slow query logging threshold |
| `VERY_SLOW_QUERY_THRESHOLD_MS` | 5000    | Very slow query threshold    |
| `API_READ_TIMEOUT`             | 30s     | HTTP read timeout            |
| `API_WRITE_TIMEOUT`            | 60s     | HTTP write timeout           |
| `STREAM_JOIN_BUDGET_MS`        | 2000    | Stream join latency budget   |

## Ranking Query Performance

The composite ranking formula uses configurable weights (see `configs/ranking.calibration.json`):

```
composite_score = (text_relevance * 0.4) + (proximity_score * 0.3) + (recency * 0.2) + (trust_weight * 0.1)
```

Trust weight is feature-flagged (`RANK_TRUST_ENABLED`); falls back to 0.0 when disabled. The ranking query must complete within **150ms p95** to stay within the search endpoint budget.

## Escalation & Regression Workflow

### Detection

1. **CI detection**: Lighthouse CI and coverage gates fail the build on budget violations
2. **Prometheus alerts**: Configured thresholds trigger when p95 exceeds budgets
3. **Manual detection**: Developers run load tests and compare against baselines

### Response Process

1. **Identify**: Which budget was violated? What changed?
2. **Capture**: Run `EXPLAIN ANALYZE` on suspect queries; collect Prometheus snapshots
3. **Triage**: Is this a regression (was passing, now failing) or a known gap?
4. **File issue**: Create an issue with:
   - Metric name and current value vs. budget
   - Baseline diff (before/after)
   - Suspect commit range
   - EXPLAIN plans if database-related
5. **Fix or defer**: Apply fix within the sprint, or document rationale for deferral
6. **Verify**: Confirm fix brings metric back within budget

### Rollback Criteria

Trigger an immediate rollback if any of these occur after deployment:

- API error rate increases >1% above baseline
- p95 latency degrades >50% for any endpoint class
- Stream join failures exceed 5% of attempts
- Health check failures for >30 seconds

## Optimization Strategies

### Database

- Use `EXPLAIN ANALYZE` on all new queries before merging
- Add composite indexes for common filter combinations
- Prefer cursor-based pagination over OFFSET for large datasets
- Use partial indexes for hot paths (e.g., active scenes only)
- Monitor `db_slow_queries_total` for regressions

### API

- Keep handler logic thin; push computation to repository layer
- Use context timeouts to prevent runaway queries
- Return only necessary fields (avoid `SELECT *` patterns)
- Apply compression for responses >1KB (Caddy handles this at the edge)

### Frontend

- All routes are lazy-loaded via `React.lazy()` (see `web/src/routes/index.tsx`)
- Heavy components (MapLibre, audio player) loaded on demand
- Images use responsive srcset with WebP format
- Static assets served with `Cache-Control: public, max-age=31536000, immutable`

### Streaming

- Opus codec at 48kHz for optimal quality/latency tradeoff
- Server region affinity to minimize round-trip time
- Adaptive bitrate based on participant bandwidth estimates

## Review Cadence

- **Monthly**: Review Prometheus dashboards for trend analysis
- **Per-sprint**: Check Lighthouse CI results for frontend regressions
- **Per-release**: Run k6 load test suite against staging environment
- **Quarterly**: Re-evaluate budget targets based on user growth and infrastructure changes
