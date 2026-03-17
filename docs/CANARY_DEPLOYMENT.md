# Canary Deployment Guide

## Overview

Subcults supports canary deployments to roll out changes incrementally and minimize risk. The canary system automatically routes a percentage of users to the new version while monitoring error rates and latency. If issues are detected, the system can automatically roll back to the stable version.

## Key Features

- **Deterministic User Assignment**: Users are consistently assigned to canary or stable cohorts based on a hash of their user ID or IP address
- **Traffic Split Control**: Configure the percentage of traffic routed to canary (default: 5%)
- **Automatic Rollback**: Monitors error rates and latency, automatically rolling back when thresholds are exceeded
- **Manual Control**: API endpoints for manual rollback and metrics monitoring
- **Prometheus Metrics**: Full observability with Prometheus metrics for both cohorts
- **Zero Configuration**: Works with existing deployment infrastructure (Docker Compose, Caddy)

## Configuration

### Environment Variables

Add these to your deployment environment (`.env` file or runtime environment):

```bash
# Enable canary deployment
CANARY_ENABLED=true

# Traffic split (0-100)
CANARY_TRAFFIC_PERCENT=5.0

# Auto-rollback thresholds
CANARY_ERROR_THRESHOLD=1.0          # Error rate % that triggers rollback
CANARY_LATENCY_THRESHOLD=2.0        # P95 latency in seconds
CANARY_AUTO_ROLLBACK=true           # Enable automatic rollback

# Monitoring window (seconds)
CANARY_MONITORING_WINDOW=300        # 5 minutes

# Version identifier
CANARY_VERSION=v1.2.0-canary        # Appears in metrics and headers
```

### Deployment Stages

Recommended rollout progression:

1. **Stage 1: 5% for 5 minutes**
   ```bash
   CANARY_TRAFFIC_PERCENT=5.0
   CANARY_MONITORING_WINDOW=300
   ```
   Monitor metrics closely. If stable, proceed to Stage 2.

2. **Stage 2: 25% for 10 minutes**
   ```bash
   CANARY_TRAFFIC_PERCENT=25.0
   CANARY_MONITORING_WINDOW=600
   ```
   Broader exposure to catch edge cases.

3. **Stage 3: 100% (full rollout)**
   ```bash
   CANARY_ENABLED=false  # Disable canary, all traffic to new version
   ```
   Canary becomes the new stable version.

## Monitoring

### API Endpoints

#### Get Current Metrics
```bash
curl http://localhost:8080/canary/metrics
```

Response:
```json
{
  "canary_requests": 1250,
  "canary_errors": 5,
  "canary_error_rate": 0.4,
  "canary_avg_latency": 0.123,
  "stable_requests": 23750,
  "stable_errors": 10,
  "stable_error_rate": 0.042,
  "stable_avg_latency": 0.115,
  "window_start": "2026-02-02T22:00:00Z",
  "window_duration": "5m0s",
  "canary_active": true,
  "canary_version": "v1.2.0-canary"
}
```

#### Manual Rollback
```bash
curl -X POST http://localhost:8080/canary/rollback \
  -H "Content-Type: application/json" \
  -d '{"reason": "manual_rollback_due_to_user_reports"}'
```

#### Reset Metrics Window
```bash
curl -X POST http://localhost:8080/canary/metrics/reset
```

### Prometheus Metrics

The following metrics are exposed at `/metrics`:

```prometheus
# Total requests by cohort
canary_requests_total{cohort="canary",version="v1.2.0-canary"}
canary_requests_total{cohort="stable",version="stable"}

# Total errors by cohort
canary_errors_total{cohort="canary",version="v1.2.0-canary"}
canary_errors_total{cohort="stable",version="stable"}

# Latency distribution by cohort
canary_latency_seconds{cohort="canary",version="v1.2.0-canary"}
canary_latency_seconds{cohort="stable",version="stable"}

# Canary deployment status (1 = active, 0 = inactive)
canary_active
```

### Grafana Dashboard Queries

**Error Rate Comparison**:
```promql
rate(canary_errors_total[5m]) / rate(canary_requests_total[5m]) * 100
```

**Latency P95 Comparison**:
```promql
histogram_quantile(0.95, rate(canary_latency_seconds_bucket[5m]))
```

**Traffic Split**:
```promql
rate(canary_requests_total[5m]) by (cohort)
```

## Auto-Rollback Conditions

The system automatically rolls back when any of these conditions are met:

1. **Error Rate Threshold**: Canary error rate exceeds `CANARY_ERROR_THRESHOLD` (default: 1%)
2. **Latency Threshold**: Canary P95 latency exceeds `CANARY_LATENCY_THRESHOLD` (default: 2s)
3. **Relative Error Rate**: Canary error rate is >2x the stable error rate

**Note**: Rollback only triggers after 100+ canary requests to ensure statistical significance.

## User Assignment

Users are deterministically assigned to cohorts using:
1. **Authenticated users**: SHA-256 hash of user DID (Decentralized Identifier)
2. **Anonymous users**: SHA-256 hash of IP address

This ensures:
- Consistent experience for each user
- No user sees version switching mid-session
- Reproducible cohort assignments for debugging

## Response Headers

All responses include these headers for observability:

```
X-Deployment-Cohort: canary
X-Deployment-Version: v1.2.0-canary
```

Use these headers to:
- Debug user-specific issues
- Verify canary assignment
- Correlate client-side errors with deployment cohort

## Deployment Workflow

### Initial Canary Deployment

1. **Deploy canary version** with `CANARY_ENABLED=true` and `CANARY_TRAFFIC_PERCENT=5.0`
2. **Monitor metrics** for 5 minutes:
   ```bash
   watch -n 5 'curl -s http://localhost:8080/canary/metrics | jq'
   ```
3. **Check for rollback**:
   - If `canary_active` becomes `false`, check logs for rollback reason
   - Fix issues and redeploy
4. **Increase traffic** to 25% if metrics are healthy
5. **Full rollout** by disabling canary (`CANARY_ENABLED=false`)

### Rollback Procedure

**Automatic Rollback**:
- System detects threshold breach
- Automatically routes all traffic to stable
- Logs rollback reason and metrics

**Manual Rollback**:
```bash
# Trigger immediate rollback
curl -X POST http://localhost:8080/canary/rollback \
  -H "Content-Type: application/json" \
  -d '{"reason": "manual_intervention"}'

# OR: Disable canary via environment variable
export CANARY_ENABLED=false
# Restart API server
```

### Post-Rollback

1. **Investigate logs** for error patterns:
   ```bash
   grep "canary rollback" /var/log/subcults-api.log
   ```
2. **Review metrics** to identify the issue
3. **Fix and redeploy** with same canary configuration
4. **Reset metrics** before starting new deployment:
   ```bash
   curl -X POST http://localhost:8080/canary/metrics/reset
   ```

## Best Practices

### Before Deployment

- [ ] Test canary build thoroughly in staging
- [ ] Set up Grafana alerts for canary metrics
- [ ] Verify rollback procedure works
- [ ] Document expected changes in metrics
- [ ] Plan monitoring window based on traffic patterns

### During Deployment

- [ ] Monitor error rates in real-time
- [ ] Check latency percentiles (P50, P95, P99)
- [ ] Watch for user-reported issues
- [ ] Compare canary vs stable metrics side-by-side
- [ ] Increase traffic gradually (5% → 25% → 100%)

### After Deployment

- [ ] Keep canary configuration for 24 hours post-rollout
- [ ] Archive metrics snapshots for postmortem
- [ ] Document any issues encountered
- [ ] Update thresholds based on observed metrics

## Troubleshooting

### Canary Not Activating

**Symptom**: All traffic goes to stable despite `CANARY_ENABLED=true`

**Checks**:
```bash
# Verify configuration
curl http://localhost:8080/canary/metrics | jq '.canary_active'

# Check environment variables
env | grep CANARY

# Review logs for rollback events
grep "rollback" /var/log/subcults-api.log
```

### Uneven Traffic Distribution

**Symptom**: Traffic split doesn't match `CANARY_TRAFFIC_PERCENT`

**Cause**: Small sample size or hash distribution

**Solution**: Wait for more requests (>1000) for accurate distribution

### Premature Rollback

**Symptom**: Canary rolls back before 100 requests

**Cause**: Safety threshold prevents rollback with insufficient data

**Solution**: This is intentional. Wait for 100+ canary requests before auto-rollback activates.

## Security Considerations

- **No PII in Cohort Assignment**: User hashing prevents cohort assignments from leaking user identities
- **Access Control**: Canary endpoints should be restricted to internal/admin access
- **Audit Logging**: All manual rollbacks are logged with reasons
- **Version Leakage**: Deployment versions are exposed in headers for debugging (consider removing in production)

## Performance Impact

- **Latency**: <1ms overhead per request for cohort assignment
- **Memory**: ~100KB for metrics tracking
- **CPU**: Negligible (hash computation amortized)

## Integration Examples

### Docker Compose

```yaml
services:
  api:
    environment:
      - CANARY_ENABLED=true
      - CANARY_TRAFFIC_PERCENT=5.0
      - CANARY_VERSION=v1.2.0-canary
      - CANARY_AUTO_ROLLBACK=true
```

### Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: canary-config
data:
  CANARY_ENABLED: "true"
  CANARY_TRAFFIC_PERCENT: "5.0"
  CANARY_VERSION: "v1.2.0-canary"
```

## Related Documentation

- [Prometheus Metrics](./PROMETHEUS_METRICS.md)
- [Deployment Infrastructure](./DEPLOYMENT.md)
- [API Reference](./API_REFERENCE.md)
