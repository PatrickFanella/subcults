# Operational Runbooks

## Overview

Step-by-step procedures for responding to alerts from the Subcults monitoring stack. Each runbook maps to one or more Prometheus alert rules defined in `subcults.yml`.

## Alert Response Priority

| Severity | Response Time     | Examples                                    |
| -------- | ----------------- | ------------------------------------------- |
| Critical | 15 min            | Service down, SLO breach, critical lag      |
| Warning  | 1 hour            | High error rate, slow queries, elevated lag |
| Info     | Next business day | Non-urgent operational notices              |

---

## API Alerts

### SubcultsAPIDown

**Severity**: Critical
**Condition**: API instance has been unreachable for 2 minutes.

**Steps**:

1. Check if the container is running: `docker ps | grep subcults-api`
2. Check container logs: `docker logs subcults-api --tail=100`
3. If container exited, check exit code and restart: `docker compose -f deploy/compose.yml up -d subcults-api`
4. Check resource usage: `docker stats subcults-api`
5. Verify the `/health/live` endpoint: `curl http://subcults-api:8080/health/live`
6. If OOM killed, increase memory limits in `deploy/compose.yml`

### SubcultsHighErrorRate / SubcultsErrorBudgetFastBurn

**Severity**: Warning / Critical
**Condition**: 5xx error rate exceeds 0.5% or error budget is burning too fast.

**Steps**:

1. Check Grafana API Overview dashboard for error spike timing
2. Check recent deployments — was anything just deployed?
3. Review API logs for error patterns: `docker logs subcults-api --since=10m 2>&1 | grep -i error`
4. Check if a specific endpoint is failing by reviewing the "Request Rate by Status Code" panel
5. Check database connectivity if errors are 500s
6. If caused by a deployment, roll back: `docker compose -f deploy/compose.yml up -d --force-recreate subcults-api`

### SubcultsHighRateLimiting

**Severity**: Warning
**Condition**: More than 100 requests/minute are being rate-limited.

**Steps**:

1. Check source IPs in logs for potential abuse
2. Review rate limit metrics in Grafana API Overview dashboard
3. If legitimate traffic, consider increasing rate limits in environment config
4. If abuse, consider IP-level blocking at Caddy

### SubcultsSlowSearch

**Severity**: Warning
**Condition**: Search endpoint p95 latency exceeds 1 second.

**Steps**:

1. Check database query performance (Grafana "Database Query Duration" panel)
2. Run `EXPLAIN ANALYZE` on slow search queries
3. Check if PostGIS indexes are in place
4. Check database connection pool saturation
5. Consider adding query result caching

---

## Indexer Alerts

### SubcultsIndexerDown

**Severity**: Critical
**Condition**: Indexer instance has been unreachable for 2 minutes.

**Steps**:

1. Check if the container is running: `docker ps | grep subcults-indexer`
2. Check container logs: `docker logs subcults-indexer --tail=100`
3. Verify Jetstream WebSocket connectivity
4. Check DATABASE_URL is correct and database is reachable
5. Restart: `docker compose -f deploy/compose.yml up -d subcults-indexer`

### SubcultsIndexerHighLag / SubcultsIndexerCriticalLag

**Severity**: Warning / Critical
**Condition**: Processing lag exceeds 60s (warning) or 300s (critical).

**Steps**:

1. Check Grafana Indexer dashboard — "Processing Lag Over Time" panel
2. Check if backpressure is active (Queue & Backpressure panel)
3. Check database write latency (Ingest Latency panel)
4. If database is slow, check for long-running queries or lock contention
5. Check if Jetstream is sending a burst of historical data
6. If lag is growing, check for memory/CPU resource constraints on the indexer container

### SubcultsIndexerHighErrorRate

**Severity**: Warning
**Condition**: Indexer error rate exceeds 5%.

**Steps**:

1. Check the "Error Rate" panel in Grafana Indexer dashboard
2. Review indexer logs for error patterns: `docker logs subcults-indexer --since=10m 2>&1 | grep -i error`
3. Check if errors are CBOR parsing failures (upstream data issues) or database write failures
4. If database errors, check database health and connectivity

### SubcultsIndexerDBWriteFailures

**Severity**: Warning
**Condition**: Database write failures occurring.

**Steps**:

1. Check the "Database Write Failures" panel in Grafana
2. Verify database connectivity: `docker exec subcults-postgres pg_isready`
3. Check for disk space issues on the database volume
4. Check for schema migration issues
5. Review error messages for constraint violations or connection timeouts

### SubcultsIndexerFrequentReconnections

**Severity**: Warning
**Condition**: More than 5 reconnection attempts in 15 minutes.

**Steps**:

1. Check network connectivity to Jetstream endpoint
2. Review indexer logs for WebSocket close codes
3. Check if the Jetstream service itself is having issues (AT Protocol status)
4. Verify DNS resolution for the Jetstream endpoint

### SubcultsIndexerBackpressure

**Severity**: Warning
**Condition**: Pending message queue exceeds 1000.

**Steps**:

1. Check the "Queue & Backpressure" panel in Grafana
2. This usually indicates the database can't keep up with ingestion rate
3. Check database query latency and connection pool usage
4. Consider scaling database resources or optimizing write queries

---

## Streaming Alerts

### SubcultsHighStreamJoinLatency

**Severity**: Warning
**Condition**: Stream join latency p95 exceeds 3 seconds.

**Steps**:

1. Check the "Join Latency Percentiles" panel in Grafana
2. Check LiveKit service health
3. Verify TURN/STUN connectivity
4. Check if the issue is regional (specific users/networks)

### SubcultsHighPacketLoss

**Severity**: Warning
**Condition**: Packet loss p95 exceeds 5%.

**Steps**:

1. Check the "Packet Loss" panel in Grafana
2. This is usually a network issue between participants and LiveKit
3. Check LiveKit server health and bandwidth
4. Review if specific sessions have disproportionate packet loss

### SubcultsHighJitter

**Severity**: Warning
**Condition**: Audio jitter p95 exceeds 30ms.

**Steps**:

1. Check the "Jitter & RTT" panel in Grafana
2. Check network quality metrics (RTT trends)
3. If systemic, check LiveKit server resources

---

## Trust & Jobs Alerts

### SubcultsTrustRecomputeStalled

**Severity**: Warning
**Condition**: Trust recompute duration exceeds 10 minutes.

**Steps**:

1. Check the "Trust Recompute Duration" panel in Grafana
2. Check if the trust graph has grown significantly
3. Review trust recompute logs for errors
4. Check database query performance for trust-related queries

### SubcultsTrustRecomputeErrors

**Severity**: Warning
**Condition**: Trust recompute errors detected.

**Steps**:

1. Check API logs for trust recompute error messages
2. Verify database connectivity
3. Check for data integrity issues in the alliance/trust tables

### SubcultsBackgroundJobFailures

**Severity**: Warning
**Condition**: Background job failure rate exceeds 10%.

**Steps**:

1. Check API logs for job failure messages: `docker logs subcults-api --since=30m 2>&1 | grep "job.*fail\|job.*error"`
2. Identify which job types are failing (job_type label)
3. Check dependencies for affected job types

---

## Database Alerts

### SubcultsSlowQueries

**Severity**: Warning
**Condition**: More than 0.1 queries/sec exceeding 100ms threshold.

**Steps**:

1. Check the "Slow Queries" panel in Grafana API Overview dashboard
2. Check the "Database Query Duration" panel for which operations are slow
3. Run `EXPLAIN ANALYZE` on slow query patterns
4. Check for missing indexes
5. Check database connection pool saturation
6. Review table statistics — run `ANALYZE` on affected tables

### SubcultsVerySlowQueries

**Severity**: Critical
**Condition**: Any query exceeding 5 second threshold.

**Steps**:

1. Check API/indexer logs for "very slow database query" error messages
2. Check for long-running transactions or lock contention: `SELECT * FROM pg_stat_activity WHERE state = 'active' AND query_start < now() - interval '5 seconds'`
3. Check for table bloat requiring VACUUM
4. Consider query optimization or adding database replicas

---

## General Troubleshooting

### Checking Service Health

```bash
# All services
docker compose -f deploy/compose.yml ps

# API health
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Indexer health
curl http://localhost:9090/health

# Metrics
curl http://localhost:8080/metrics
curl http://localhost:9090/internal/indexer/metrics
```

### Restarting Services

```bash
# Single service
docker compose -f deploy/compose.yml restart subcults-api

# All services
docker compose -f deploy/compose.yml up -d --force-recreate
```

### Checking Logs

```bash
# Recent logs
docker logs subcults-api --since=10m
docker logs subcults-indexer --since=10m

# Follow logs
docker logs -f subcults-api
```
