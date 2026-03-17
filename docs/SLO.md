# Service Level Objectives (SLOs)

## Overview

This document defines the Service Level Objectives for Subcults services. SLOs guide operational decisions, alert thresholds, and capacity planning.

**Error Budget Model**: Each SLO has a 30-day rolling window. When the error budget is exhausted, new feature work pauses in favor of reliability improvements.

## API Service SLOs

### Availability

| SLO              | Target | Window  | Measurement                             |
| ---------------- | ------ | ------- | --------------------------------------- |
| API Availability | 99.5%  | 30 days | `1 - (5xx responses / total responses)` |

**Error Budget**: 3.6 hours of downtime per 30-day window.

**PromQL**:

```promql
1 - (
  sum(rate(http_requests_total{status=~"5.."}[30d]))
  /
  sum(rate(http_requests_total[30d]))
)
```

### Latency

| SLO                  | Target  | Window     | Measurement                                                                      |
| -------------------- | ------- | ---------- | -------------------------------------------------------------------------------- |
| API Latency (p95)    | < 300ms | Rolling 5m | `histogram_quantile(0.95, http_request_duration_seconds)`                        |
| API Latency (p99)    | < 1s    | Rolling 5m | `histogram_quantile(0.99, http_request_duration_seconds)`                        |
| Search Latency (p95) | < 500ms | Rolling 5m | `histogram_quantile(0.95, http_request_duration_seconds{path="/search/scenes"})` |

### Error Rate

| SLO              | Target | Window     | Measurement                                                                    |
| ---------------- | ------ | ---------- | ------------------------------------------------------------------------------ |
| Error Rate (5xx) | < 0.5% | Rolling 5m | `rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])` |

## Indexer Service SLOs

### Processing Lag

| SLO            | Target | Window     | Measurement                      |
| -------------- | ------ | ---------- | -------------------------------- |
| Processing Lag | < 60s  | Continuous | `indexer_processing_lag_seconds` |
| Critical Lag   | < 300s | Continuous | `indexer_processing_lag_seconds` |

### Throughput

| SLO               | Target | Window     | Measurement                                                                           |
| ----------------- | ------ | ---------- | ------------------------------------------------------------------------------------- |
| Error Rate        | < 5%   | Rolling 5m | `rate(indexer_messages_error_total[5m]) / rate(indexer_messages_processed_total[5m])` |
| DB Write Failures | < 1%   | Rolling 5m | `rate(indexer_database_writes_failed_total[5m])`                                      |

## Streaming SLOs

### Join Performance

| SLO                       | Target | Window     | Measurement                                             |
| ------------------------- | ------ | ---------- | ------------------------------------------------------- |
| Stream Join Latency (p95) | < 2s   | Rolling 5m | `histogram_quantile(0.95, stream_join_latency_seconds)` |

### Audio Quality

| SLO               | Target | Window     | Measurement                                                  |
| ----------------- | ------ | ---------- | ------------------------------------------------------------ |
| Packet Loss (p95) | < 5%   | Rolling 5m | `histogram_quantile(0.95, stream_audio_packet_loss_percent)` |
| Jitter (p95)      | < 30ms | Rolling 5m | `histogram_quantile(0.95, stream_audio_jitter_ms)`           |

## Database SLOs

### Query Performance

| SLO                     | Target    | Window     | Measurement                            |
| ----------------------- | --------- | ---------- | -------------------------------------- |
| Slow Queries (>100ms)   | < 0.1/sec | Rolling 5m | `rate(db_slow_queries_total[5m])`      |
| Very Slow Queries (>5s) | 0         | Rolling 5m | `rate(db_very_slow_queries_total[5m])` |

## Trust System SLOs

### Recompute Performance

| SLO             | Target  | Window         | Measurement                        |
| --------------- | ------- | -------------- | ---------------------------------- |
| Trust Recompute | < 5 min | Per invocation | `trust_recompute_duration_seconds` |

## Performance Budgets

These budgets apply to frontend and end-to-end measurements:

| Metric          | Budget      | Tool       |
| --------------- | ----------- | ---------- |
| API Latency     | p95 < 300ms | Prometheus |
| Stream Join     | < 2s        | Prometheus |
| Map Render      | < 1.2s      | Lighthouse |
| FCP             | < 1.0s      | Lighthouse |
| Trust Recompute | < 5m        | Prometheus |

## Alert Configuration

Alert rules are defined in the monitoring stack at:

```
~/projects/monitoring/prometheus/alerts/subcults.yml
```

Alert groups:

- **subcults_slo_alerts** — SLO burn-rate alerts (availability, error rate, latency)
- **subcults_api_alerts** — API health (down, rate limiting, slow search)
- **subcults_indexer_alerts** — Indexer health (down, lag, errors, reconnections)
- **subcults_streaming_alerts** — Audio quality (join latency, packet loss, jitter)
- **subcults_trust_job_alerts** — Trust recompute (stalled, errors, slow)
- **subcults_database_alerts** — Database (slow queries, very slow queries)

## Dashboards

Grafana dashboards are provisioned at:

```
~/projects/monitoring/grafana/provisioning/dashboards/subcults/
```

- **API Overview** — Request rates, latency percentiles, error rates, rate limiting, background jobs
- **Indexer** — Processing lag, throughput, queue depth, reconnections, DB write failures
- **Streaming & Trust** — Join/leave rates, audio quality, trust recompute duration
