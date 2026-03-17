# Observability Guide

## Overview

Subcults implements comprehensive observability for operational insight, performance monitoring, and troubleshooting. This guide covers the available metrics, their usage, and privacy considerations.

## Metrics Endpoint

All metrics are exposed via the Prometheus exposition format at:

```
GET /metrics
```

The endpoint returns metrics in text format compatible with Prometheus scraping.

## Available Metrics

### Stream Session Metrics

#### `stream_joins_total`

**Type**: Counter  
**Description**: Total number of stream join events across all sessions.

**Usage**: Track overall streaming participation and popularity trends.

**Example Queries**:
```promql
# Total joins in the last hour
rate(stream_joins_total[1h]) * 3600

# Join rate per minute
rate(stream_joins_total[1m]) * 60
```

#### `stream_leaves_total`

**Type**: Counter  
**Description**: Total number of stream leave events across all sessions.

**Usage**: Track user engagement duration and churn patterns.

**Example Queries**:
```promql
# Total leaves in the last hour
rate(stream_leaves_total[1h]) * 3600

# Leave rate per minute
rate(stream_leaves_total[1m]) * 60

# Join to leave ratio (retention indicator)
stream_joins_total / stream_leaves_total
```

#### `stream_join_latency_seconds`

**Type**: Histogram  
**Description**: Time from token issuance to successful stream join completion (first audio track subscription).

**Buckets**: 0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0 seconds

**Usage**: Monitor stream join performance and identify latency issues.

**Example Queries**:
```promql
# 95th percentile join latency
histogram_quantile(0.95, rate(stream_join_latency_seconds_bucket[5m]))

# 50th percentile (median) join latency
histogram_quantile(0.50, rate(stream_join_latency_seconds_bucket[5m]))

# Average join latency
rate(stream_join_latency_seconds_sum[5m]) / rate(stream_join_latency_seconds_count[5m])

# Percentage of joins completing within 2 seconds
sum(rate(stream_join_latency_seconds_bucket{le="2.0"}[5m])) / sum(rate(stream_join_latency_seconds_count[5m])) * 100
```

### Indexer Metrics

The Jetstream indexer exposes metrics at `/internal/indexer/metrics`.

Authentication for this endpoint is optional and controlled by the `INTERNAL_AUTH_TOKEN` configuration:

- If `INTERNAL_AUTH_TOKEN` is **not** set, `/internal/indexer/metrics` is exposed without authentication (intended for trusted internal networks only).
- If `INTERNAL_AUTH_TOKEN` **is** set, all requests must include the following HTTP header:

  ```http
  X-Internal-Token: <INTERNAL_AUTH_TOKEN>
  ```

This header-based internal token is what `InternalAuthMiddleware` validates before allowing access to the indexer metrics.

#### `indexer_messages_processed_total`

**Type**: Counter  
**Description**: Total number of messages successfully processed from Jetstream.

**Usage**: Track overall indexer throughput and message volume.

**Example Queries**:
```promql
# Messages processed per second
rate(indexer_messages_processed_total[1m])

# Total messages processed in the last hour
increase(indexer_messages_processed_total[1h])
```

#### `indexer_messages_error_total`

**Type**: Counter  
**Description**: Total number of messages that resulted in processing errors.

**Usage**: Monitor indexer reliability and identify issues.

**Example Queries**:
```promql
# Error rate per second
rate(indexer_messages_error_total[1m])

# Error percentage
rate(indexer_messages_error_total[5m]) / rate(indexer_messages_processed_total[5m]) * 100
```

#### `indexer_upserts_total`

**Type**: Counter  
**Description**: Total number of database upsert operations performed.

**Usage**: Track database write activity.

#### `indexer_trust_recompute_total`

**Type**: Counter  
**Description**: Total number of trust score recomputation operations triggered.

**Usage**: Monitor trust graph computation frequency.

#### `indexer_ingest_latency_seconds`

**Type**: Histogram  
**Description**: Time taken to process individual messages (from receipt to completion).

**Buckets**: Default Prometheus buckets

**Usage**: Monitor message processing performance.

**Example Queries**:
```promql
# 95th percentile processing latency
histogram_quantile(0.95, rate(indexer_ingest_latency_seconds_bucket[5m]))

# Average processing latency
rate(indexer_ingest_latency_seconds_sum[5m]) / rate(indexer_ingest_latency_seconds_count[5m])
```

#### `indexer_processing_lag_seconds`

**Type**: Gauge  
**Description**: Time difference between message timestamp and processing time (indicates how far behind real-time the indexer is).

**Usage**: Monitor indexer lag and detect backlog issues.

**Example Queries**:
```promql
# Current processing lag
indexer_processing_lag_seconds

# Maximum lag in the last 5 minutes
max_over_time(indexer_processing_lag_seconds[5m])
```

#### `indexer_reconnection_attempts_total`

**Type**: Counter  
**Description**: Total number of reconnection attempts to Jetstream.

**Usage**: Monitor connection stability and network issues.

**Example Queries**:
```promql
# Reconnection rate per minute
rate(indexer_reconnection_attempts_total[1m]) * 60

# Total reconnections in the last hour
increase(indexer_reconnection_attempts_total[1h])
```

#### `indexer_database_writes_failed_total`

**Type**: Counter  
**Description**: Total number of failed database write operations.

**Usage**: Monitor database health and identify persistence issues.

**Example Queries**:
```promql
# Database failure rate
rate(indexer_database_writes_failed_total[5m])

# Failure percentage
rate(indexer_database_writes_failed_total[5m]) / rate(indexer_upserts_total[5m]) * 100
```

#### `indexer_backpressure_paused_total`

**Type**: Counter  
**Description**: Total number of times message consumption was paused due to backpressure.

**Usage**: Monitor queue saturation and processing capacity.

#### `indexer_backpressure_resumed_total`

**Type**: Counter  
**Description**: Total number of times message consumption resumed after backpressure.

**Usage**: Track backpressure recovery cycles.

#### `indexer_backpressure_pause_duration_seconds`

**Type**: Histogram  
**Description**: Duration of backpressure pause events.

**Buckets**: 0.1, 0.5, 1, 5, 10, 30, 60 seconds

**Usage**: Analyze backpressure severity and duration.

**Example Queries**:
```promql
# 95th percentile pause duration
histogram_quantile(0.95, rate(indexer_backpressure_pause_duration_seconds_bucket[5m]))

# Average pause duration
rate(indexer_backpressure_pause_duration_seconds_sum[5m]) / rate(indexer_backpressure_pause_duration_seconds_count[5m])
```

#### `indexer_pending_messages`

**Type**: Gauge  
**Description**: Current number of messages waiting in the processing queue.

**Usage**: Monitor queue depth and detect processing bottlenecks.

**Example Queries**:
```promql
# Current queue depth
indexer_pending_messages

# Maximum queue depth in the last hour
max_over_time(indexer_pending_messages[1h])
```

### Indexer Alert Conditions

Recommended Prometheus alert rules for the indexer:

```yaml
groups:
  - name: indexer
    rules:
      # High processing lag
      - alert: HighIndexerProcessingLag
        expr: indexer_processing_lag_seconds > 60
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Indexer is lagging behind real-time"
          description: "Processing lag is {{ $value }}s (threshold: 60s)"

      # Frequent reconnections
      - alert: FrequentIndexerReconnections
        expr: rate(indexer_reconnection_attempts_total[5m]) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Indexer experiencing frequent reconnections"
          description: "Reconnection rate is {{ $value }}/sec"

      # High error rate
      - alert: HighIndexerErrorRate
        expr: rate(indexer_messages_error_total[5m]) / rate(indexer_messages_processed_total[5m]) > 0.05
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "High indexer error rate"
          description: "Error rate is {{ $value | humanizePercentage }}"

      # Database write failures
      - alert: IndexerDatabaseWriteFailures
        expr: rate(indexer_database_writes_failed_total[5m]) > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Indexer database writes failing"
          description: "Database write failure rate is {{ $value }}/sec"

      # Prolonged backpressure
      - alert: IndexerBackpressure
        expr: indexer_pending_messages > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Indexer experiencing backpressure"
          description: "Pending messages: {{ $value }} (threshold: 1000)"
```

### Background Job Metrics

Background jobs (trust recomputation, payment processing, stream cleanup, etc.) expose centralized metrics for monitoring health and performance.

#### `background_jobs_total`

**Type**: Counter  
**Labels**: `job_type`, `status`  
**Description**: Total number of background job executions by type and status.

**Job Types**:
- `trust_recompute` - Trust score recomputation jobs
- `index_backfill` - Index backfill operations
- `index_processing` - Continuous index message processing
- `payment_processing` - Payment webhook processing
- `stream_cleanup` - Stream session cleanup
- `cache_invalidation` - Cache invalidation jobs
- `report_generation` - Report generation jobs

**Status Values**:
- `success` - Job completed successfully
- `failure` - Job completed with errors

**Usage**: Track job execution frequency, success rates, and failure patterns.

**Example Queries**:
```promql
# Success rate by job type (last 5 minutes)
sum(rate(background_jobs_total{status="success"}[5m])) by (job_type) /
sum(rate(background_jobs_total[5m])) by (job_type)

# Total successful jobs in the last hour
sum(increase(background_jobs_total{status="success"}[1h])) by (job_type)

# Failed trust recompute jobs
sum(increase(background_jobs_total{job_type="trust_recompute", status="failure"}[1h]))

# Overall job failure rate
sum(rate(background_jobs_total{status="failure"}[5m])) /
sum(rate(background_jobs_total[5m]))
```

#### `background_jobs_duration_seconds`

**Type**: Histogram  
**Labels**: `job_type`  
**Description**: Histogram of background job execution duration in seconds.

**Buckets**: 0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0 seconds

**Usage**: Monitor job performance, detect slowdowns, and set SLOs.

**Example Queries**:
```promql
# 95th percentile job duration by type
histogram_quantile(0.95, sum by (job_type, le) (rate(background_jobs_duration_seconds_bucket[5m])))

# 50th percentile (median) duration
histogram_quantile(0.50, sum by (job_type, le) (rate(background_jobs_duration_seconds_bucket[5m])))

# Average job duration
sum by (job_type) (rate(background_jobs_duration_seconds_sum[5m])) /
sum by (job_type) (rate(background_jobs_duration_seconds_count[5m]))

# Jobs with p95 duration exceeding 30 seconds
histogram_quantile(0.95, sum by (job_type, le) (rate(background_jobs_duration_seconds_bucket[5m]))) > 30
```

#### `background_job_errors_total`

**Type**: Counter  
**Labels**: `job_type`, `error_type`  
**Description**: Total number of background job errors by type and error category.

**Common Error Types**:
- `timeout` - Job exceeded execution timeout
- `database_error` - Database operation failure
- `network_error` - Network connectivity issue
- `validation_error` - Input validation failure
- `permission_denied` - Authorization failure
- `recompute_error` - Trust score computation error
- `not_found` - Resource not found

**Usage**: Track specific error patterns and identify failure root causes.

**Example Queries**:
```promql
# Error rate by type for trust recompute
rate(background_job_errors_total{job_type="trust_recompute"}[5m]) by (error_type)

# Most common error types across all jobs
topk(5, sum(rate(background_job_errors_total[1h])) by (error_type))

# Timeout errors in the last hour
sum(increase(background_job_errors_total{error_type="timeout"}[1h])) by (job_type)

# Jobs with high database error rates
sum(rate(background_job_errors_total{error_type="database_error"}[5m])) by (job_type) > 0.1
```

### Background Job Alert Conditions

Recommended Prometheus alert rules for background jobs:

```yaml
groups:
  - name: background_jobs
    rules:
      # High job failure rate
      - alert: HighBackgroundJobFailureRate
        expr: |
          sum(rate(background_jobs_total{status="failure"}[5m])) by (job_type) /
          sum(rate(background_jobs_total[5m])) by (job_type) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High failure rate for {{ $labels.job_type }}"
          description: "Failure rate is {{ $value | humanizePercentage }}"

      # Job duration exceeding SLO
      - alert: BackgroundJobSlow
        expr: |
          histogram_quantile(0.95,
            sum by (job_type, le) (rate(background_jobs_duration_seconds_bucket[5m]))
          ) > 60
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "{{ $labels.job_type }} p95 duration exceeding 60s"
          description: "p95 duration is {{ $value }}s"

      # Job not running (no executions in expected interval)
      - alert: BackgroundJobStalled
        expr: |
          sum by (job_type) (rate(background_jobs_total[10m])) == 0
        labels:
          severity: critical
        annotations:
          summary: "{{ $labels.job_type }} has not run in 10 minutes"
          description: "Check job scheduler and service health"

      # High timeout error rate
      - alert: BackgroundJobTimeouts
        expr: |
          sum(rate(background_job_errors_total{error_type="timeout"}[5m])) by (job_type) > 0.05
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High timeout rate for {{ $labels.job_type }}"
          description: "Timeout error rate is {{ $value }}/sec"

      # Database errors in jobs
      - alert: BackgroundJobDatabaseErrors
        expr: |
          sum(rate(background_job_errors_total{error_type="database_error"}[5m])) by (job_type) > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Database errors in {{ $labels.job_type }}"
          description: "Database error rate is {{ $value }}/sec"
```

## Recording Join/Leave Events

### Join Event

**Endpoint**: `POST /streams/{id}/join`

**Request Body** (optional):
```json
{
  "token_issued_at": "2025-12-24T00:44:43Z"
}
```

**Behavior**:
- Increments `stream_joins_total` counter
- Increments per-session `join_count` in database
- If `token_issued_at` provided, records join latency histogram sample
- Creates audit log entry (action: "joined")

**Response**:
```json
{
  "stream_id": "uuid-here",
  "room_name": "scene-123-1703376283",
  "join_count": 5,
  "status": "joined"
}
```

### Leave Event

**Endpoint**: `POST /streams/{id}/leave`

**Request Body**: None required

**Behavior**:
- Increments `stream_leaves_total` counter
- Increments per-session `leave_count` in database
- Creates audit log entry (action: "left")

**Response**:
```json
{
  "stream_id": "uuid-here",
  "room_name": "scene-123-1703376283",
  "leave_count": 3,
  "status": "left"
}
```

## Privacy Considerations

### No PII in Metrics Labels

All streaming metrics follow strict privacy guidelines:

- **DO NOT** include user DIDs in metric labels
- **DO NOT** include participant identities in metric labels
- **DO NOT** include geographic coordinates in metric labels

Only use:
- Session IDs (UUIDs)
- Room names (anonymous identifiers)
- Aggregate counts

### Audit Logging

Join/leave events create audit log entries that include:
- User DID (hashed for tamper detection)
- Entity type: `"stream_session"`
- Entity ID: stream session UUID
- Action: `"joined"` or `"left"`
- Request ID for correlation

**Access to audit logs is restricted and requires appropriate authorization.**

## Performance Budgets

### Stream Join Latency Targets

- **p50** (median): < 1.0 second
- **p95**: < 2.0 seconds
- **p99**: < 5.0 seconds

### Alert Conditions

Recommended Prometheus alert rules:

```yaml
groups:
  - name: streaming
    rules:
      # High join latency
      - alert: HighStreamJoinLatency
        expr: histogram_quantile(0.95, rate(stream_join_latency_seconds_bucket[5m])) > 3
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Stream join latency above 3s (p95)"
          description: "95th percentile join latency is {{ $value }}s"

      # Join failures (inferred from high leave rate shortly after joins)
      - alert: HighStreamChurnRate
        expr: rate(stream_leaves_total[5m]) / rate(stream_joins_total[5m]) > 0.8
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "High stream churn rate detected"
          description: "{{ $value | humanizePercentage }} of users leaving shortly after joining"
```

## Historical Analytics

Per-session aggregate counts (`join_count`, `leave_count`) are stored in the `stream_sessions` table for historical queries:

```sql
-- Average joins per session
SELECT AVG(join_count) FROM stream_sessions WHERE ended_at IS NOT NULL;

-- Sessions with high churn (more leaves than joins)
SELECT id, room_name, join_count, leave_count
FROM stream_sessions
WHERE leave_count > join_count
  AND ended_at IS NOT NULL;

-- Total participation over time
SELECT DATE_TRUNC('day', started_at) as day,
       SUM(join_count) as total_joins,
       SUM(leave_count) as total_leaves
FROM stream_sessions
WHERE started_at >= NOW() - INTERVAL '30 days'
GROUP BY day
ORDER BY day;
```

## Integration with Monitoring Stack

### Prometheus Configuration

Add scrape config to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'subcults-api'
    static_configs:
      - targets: ['api:8080']
    scrape_interval: 15s
    metrics_path: /metrics

  - job_name: 'subcults-indexer'
    static_configs:
      - targets: ['indexer:9090']
    scrape_interval: 15s
    metrics_path: /internal/indexer/metrics
    # Optional: Add authentication if configured
    # bearer_token: '<internal-auth-token>'
```

### Grafana Dashboards

#### Streaming Dashboard

Recommended panels:

1. **Join Rate Timeline**: `rate(stream_joins_total[5m])`
2. **Leave Rate Timeline**: `rate(stream_leaves_total[5m])`
3. **Join Latency Heatmap**: Histogram visualization of `stream_join_latency_seconds`
4. **P95 Latency**: `histogram_quantile(0.95, rate(stream_join_latency_seconds_bucket[5m]))`
5. **Active Sessions**: Count from database query

#### Indexer Dashboard

Recommended panels:

1. **Message Processing Rate**: `rate(indexer_messages_processed_total[1m])`
2. **Error Rate**: `rate(indexer_messages_error_total[1m])`
3. **Processing Lag**: `indexer_processing_lag_seconds`
4. **Queue Depth**: `indexer_pending_messages`
5. **Reconnection Rate**: `rate(indexer_reconnection_attempts_total[5m])`
6. **P95 Ingest Latency**: `histogram_quantile(0.95, rate(indexer_ingest_latency_seconds_bucket[5m]))`
7. **Backpressure Events**: `rate(indexer_backpressure_paused_total[5m])`
8. **Database Write Failures**: `rate(indexer_database_writes_failed_total[5m])`

## Troubleshooting

### High Join Latency

Possible causes:
1. Network congestion between client and LiveKit SFU
2. TURN server overload (check relay usage)
3. Client device performance issues
4. WebRTC negotiation failures

**Investigation**:
- Check LiveKit server metrics
- Review client-side logs for ICE connection failures
- Verify TURN server availability

### Metric Discrepancies

If `stream_joins_total` and database `SUM(join_count)` don't match:
- Counter may have been reset (server restart)
- Some joins may have failed before database update
- Check audit logs for errors

**Resolution**: Use database counts as source of truth for historical data.

## Future Enhancements

Planned additions:
- Per-scene join/leave metrics (with scene ID label)
- Stream duration histogram
- Participant count gauge per active session
- Client-side error rate tracking

## Audio Quality Metrics

### Overview

Audio quality metrics track real-time network conditions and audio performance for stream participants. These metrics enable proactive detection of quality degradation and support adaptive streaming strategies.

### Available Audio Quality Metrics

#### `stream_audio_bitrate_kbps`

**Type**: Histogram  
**Description**: Audio bitrate in kilobits per second for active participants.

**Buckets**: 16, 32, 64, 96, 128, 160, 192, 256, 320 kbps

**Usage**: Monitor audio quality and detect bandwidth constraints.

**Example Queries**:
```promql
# 95th percentile audio bitrate
histogram_quantile(0.95, rate(stream_audio_bitrate_kbps_bucket[5m]))

# Average audio bitrate
rate(stream_audio_bitrate_kbps_sum[5m]) / rate(stream_audio_bitrate_kbps_count[5m])
```

#### `stream_audio_jitter_ms`

**Type**: Histogram  
**Description**: Audio jitter (packet delay variation) in milliseconds.

**Buckets**: 1, 5, 10, 20, 30, 50, 100, 200 ms

**Usage**: Detect network instability and buffer issues.

**Alert Threshold**: Jitter > 30ms indicates poor network quality.

**Example Queries**:
```promql
# 95th percentile jitter
histogram_quantile(0.95, rate(stream_audio_jitter_ms_bucket[5m]))

# Participants experiencing high jitter (>30ms)
1 - (sum(rate(stream_audio_jitter_ms_bucket{le="30"}[5m])) / sum(rate(stream_audio_jitter_ms_count[5m])))
```

#### `stream_audio_packet_loss_percent`

**Type**: Histogram  
**Description**: Audio packet loss percentage (0-100) for active participants.

**Buckets**: 0.1, 0.5, 1, 2, 5, 10, 20, 50 percent

**Usage**: Identify network reliability issues and quality degradation.

**Alert Threshold**: Packet loss > 5% triggers high packet loss counter.

**Example Queries**:
```promql
# 95th percentile packet loss
histogram_quantile(0.95, rate(stream_audio_packet_loss_percent_bucket[5m]))

# Participants experiencing high packet loss (>5%)
(sum(rate(stream_audio_packet_loss_percent_bucket{le="+Inf"}[5m])) - sum(rate(stream_audio_packet_loss_percent_bucket{le="5"}[5m]))) / sum(rate(stream_audio_packet_loss_percent_count[5m]))
```

#### `stream_audio_level`

**Type**: Histogram  
**Description**: Audio level from 0.0 (silent) to 1.0 (loudest).

**Buckets**: 0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0

**Usage**: Monitor audio activity and detect silence/dropout issues.

**Example Queries**:
```promql
# 95th percentile audio level
histogram_quantile(0.95, rate(stream_audio_level_bucket[5m]))

# Percentage of time with active audio (>0.1)
(sum(rate(stream_audio_level_bucket{le="+Inf"}[5m])) - sum(rate(stream_audio_level_bucket{le="0.1"}[5m]))) / sum(rate(stream_audio_level_count[5m]))
```

#### `stream_network_rtt_ms`

**Type**: Histogram  
**Description**: Network round-trip time in milliseconds between client and server.

**Buckets**: 10, 25, 50, 100, 150, 200, 300, 500, 1000 ms

**Usage**: Monitor network latency and connection quality.

**Alert Threshold**: RTT > 300ms indicates poor network quality.

**Example Queries**:
```promql
# 95th percentile RTT
histogram_quantile(0.95, rate(stream_network_rtt_ms_bucket[5m]))

# Average RTT
rate(stream_network_rtt_ms_sum[5m]) / rate(stream_network_rtt_ms_count[5m])
```

#### `stream_quality_alerts_total`

**Type**: Counter  
**Description**: Total number of audio quality alerts triggered due to poor network conditions.

**Usage**: Track frequency of quality degradation events.

**Trigger Conditions**:
- Packet loss > 5%
- Jitter > 30ms
- RTT > 300ms

**Example Queries**:
```promql
# Quality alert rate per minute
rate(stream_quality_alerts_total[1m]) * 60

# Total alerts in last hour
increase(stream_quality_alerts_total[1h])
```

#### `stream_high_packet_loss_total`

**Type**: Counter  
**Description**: Total number of high packet loss events (>5%).

**Usage**: Track packet loss occurrences specifically for quick alerting.

**Example Queries**:
```promql
# High packet loss event rate
rate(stream_high_packet_loss_total[5m])

# Percentage of observations with high packet loss
rate(stream_high_packet_loss_total[5m]) / rate(stream_audio_packet_loss_percent_count[5m]) * 100
```

### Audio Quality API Endpoints

#### Get Stream Quality Metrics

**Endpoint**: `GET /streams/{id}/quality-metrics`

**Query Parameters**:
- `limit` (optional): Number of recent metrics to return (default: 100, max: 1000)

**Response**:
```json
{
  "stream_id": "uuid-here",
  "metrics": [
    {
      "id": "metric-uuid",
      "stream_session_id": "stream-uuid",
      "participant_id": "user-abc123",
      "bitrate_kbps": 128.5,
      "jitter_ms": 15.2,
      "packet_loss_percent": 2.3,
      "audio_level": 0.72,
      "rtt_ms": 45.8,
      "measured_at": "2026-01-29T02:30:00Z"
    }
  ],
  "count": 1
}
```

#### Get Participant Quality Metrics

**Endpoint**: `GET /streams/{id}/participants/{participant_id}/quality-metrics`

**Response**: Returns the most recent quality metrics for the specified participant.

#### Collect Quality Metrics

**Endpoint**: `POST /streams/{id}/quality-metrics/collect`

**Description**: Manually trigger collection of quality metrics from LiveKit for all participants. Typically called periodically by a background job.

**Response**:
```json
{
  "stream_id": "uuid-here",
  "participants": 5,
  "metrics_recorded": 5,
  "alerts_triggered": 1,
  "measured_at": "2026-01-29T02:30:00Z"
}
```

#### Get High Packet Loss Participants

**Endpoint**: `GET /streams/{id}/quality-metrics/high-packet-loss`

**Query Parameters**:
- `since_minutes` (optional): Time window in minutes (default: 5, max: 60)

**Response**:
```json
{
  "stream_id": "uuid-here",
  "since_minutes": 5,
  "participants": ["user-abc123", "user-def456"],
  "count": 2
}
```

### Quality Alert Conditions

Recommended Prometheus alert rules for audio quality:

```yaml
groups:
  - name: audio_quality
    rules:
      # High packet loss across stream
      - alert: HighStreamPacketLoss
        expr: histogram_quantile(0.95, rate(stream_audio_packet_loss_percent_bucket[5m])) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High packet loss detected in stream"
          description: "95th percentile packet loss is {{ $value }}% (threshold: 5%)"

      # High jitter across stream
      - alert: HighStreamJitter
        expr: histogram_quantile(0.95, rate(stream_audio_jitter_ms_bucket[5m])) > 30
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High jitter detected in stream"
          description: "95th percentile jitter is {{ $value }}ms (threshold: 30ms)"

      # High network latency
      - alert: HighStreamLatency
        expr: histogram_quantile(0.95, rate(stream_network_rtt_ms_bucket[5m])) > 300
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High network latency detected"
          description: "95th percentile RTT is {{ $value }}ms (threshold: 300ms)"

      # Frequent quality alerts
      - alert: FrequentQualityAlerts
        expr: rate(stream_quality_alerts_total[5m]) > 0.1
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Frequent quality alerts detected"
          description: "Quality alert rate is {{ $value }}/sec"

      # Low audio bitrate indicating quality degradation
      - alert: LowAudioBitrate
        expr: histogram_quantile(0.50, rate(stream_audio_bitrate_kbps_bucket[5m])) < 64
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Low audio bitrate detected"
          description: "Median bitrate is {{ $value }} kbps (threshold: 64 kbps)"
```

### Quality Degradation Strategy

When poor network quality is detected:

1. **Detection**: Metrics collection identifies participants with:
   - Packet loss > 5%
   - Jitter > 30ms
   - RTT > 300ms

2. **Alerting**: 
   - Prometheus alerts trigger
   - Logs capture quality degradation events
   - `stream_quality_alerts_total` counter increments

3. **Client Response** (recommended):
   - Poll `/streams/{id}/quality-metrics/high-packet-loss` endpoint
   - Reduce audio codec bitrate
   - Suggest network troubleshooting to user
   - Consider graceful quality reduction

4. **Monitoring**:
   - Track alert frequency
   - Analyze time series data for patterns
   - Identify systematic network issues

### Performance Budgets

Audio quality targets:
- **Packet Loss**: p95 < 2%, p99 < 5%
- **Jitter**: p95 < 20ms, p99 < 30ms
- **RTT**: p95 < 150ms, p99 < 300ms
- **Bitrate**: p50 > 96 kbps, p95 > 128 kbps

### Integration with Monitoring Stack

Add audio quality panels to Grafana dashboards:

```yaml
# Grafana Dashboard - Audio Quality Panel
- title: "Audio Quality Metrics"
  panels:
    - title: "Packet Loss (p95)"
      query: histogram_quantile(0.95, rate(stream_audio_packet_loss_percent_bucket[5m]))
      threshold: 5
      
    - title: "Jitter (p95)"
      query: histogram_quantile(0.95, rate(stream_audio_jitter_ms_bucket[5m]))
      threshold: 30
      
    - title: "Network RTT (p95)"
      query: histogram_quantile(0.95, rate(stream_network_rtt_ms_bucket[5m]))
      threshold: 300
      
    - title: "Quality Alerts Rate"
      query: rate(stream_quality_alerts_total[5m])
      
    - title: "High Packet Loss Events"
      query: rate(stream_high_packet_loss_total[5m])
```

## See Also

- [API Reference](./API_REFERENCE.md) - Full API documentation
- [Privacy Guide](./PRIVACY.md) - Privacy policies and data handling
- [Architecture](./ARCHITECTURE.md) - System architecture overview
