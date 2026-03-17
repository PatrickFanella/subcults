# Health Check Endpoints

The Subcults API provides health check endpoints for Kubernetes liveness and readiness probes, as well as general monitoring.

## Endpoints

### Liveness Probe: `GET /health/live`

Returns a lightweight health check indicating whether the service is running.

**Purpose**: Kubernetes uses this to determine if the container should be restarted.

**Checks Performed**:
- None (lightweight runtime check only)

**Response Format**:
```json
{
  "status": "up",
  "uptime_s": 3600
}
```

**Response Codes**:
- `200 OK`: Service is running

**Kubernetes Configuration**:
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 3
```

### Readiness Probe: `GET /health/ready`

Returns a comprehensive health check including external dependencies.

**Purpose**: Kubernetes uses this to determine if the pod should receive traffic.

**Checks Performed**:
- Database connectivity (Postgres/PostGIS) - if configured
- Redis connectivity - if configured
- LiveKit availability - if configured

**Response Format** (all healthy):
```json
{
  "status": "up",
  "checks": {
    "db": "ok",
    "redis": "ok",
    "livekit": "ok"
  },
  "uptime_s": 3600
}
```

**Response Format** (unhealthy):
```json
{
  "status": "unhealthy",
  "checks": {
    "db": "error",
    "redis": "ok",
    "livekit": "ok"
  },
  "uptime_s": 3600
}
```

**Response Codes**:
- `200 OK`: Service is ready to handle traffic
- `503 Service Unavailable`: One or more dependencies are unhealthy

**Kubernetes Configuration**:
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 1
```

## Implementation Details

### Health Checkers

Health checkers are implemented in the `internal/health` package and conform to the `HealthChecker` interface:

```go
type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}
```

Available health checkers:
- **DBChecker**: Checks database connectivity using `PingContext()`
- **RedisChecker**: Checks Redis connectivity using `PING` command
- **LiveKitChecker**: Checks LiveKit server reachability via HTTP

### Configuration

Health checkers are configured in `cmd/api/main.go`:

```go
healthHandlers := api.NewHealthHandlers(api.HealthHandlersConfig{
    DBChecker:      health.NewDBChecker(db),           // If database configured
    RedisChecker:   health.NewRedisChecker(redisClient), // If Redis configured
    LiveKitChecker: health.NewLiveKitChecker(livekitURL), // If LiveKit configured
    MetricsEnabled: true,
})
```

If a dependency is not configured (checker is `nil`), it won't be checked and won't appear in the response.

## Docker Compose

The Docker Compose healthcheck uses the liveness endpoint:

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health/live"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 10s
```

## Monitoring

### Response Time Tracking

Health check response times should be monitored to detect degradation:
- Liveness checks should complete in <10ms
- Readiness checks should complete in <100ms

### Failure Alerting

Alert on:
- Repeated liveness probe failures (indicates service crash loop)
- Readiness probe failures (indicates dependency issues)
- Slow health check responses (>500ms)

### Example Prometheus Queries

```promql
# Health check failures
rate(http_request_duration_seconds_count{endpoint="/health/ready",status="503"}[5m])

# Health check latency
histogram_quantile(0.95, 
  rate(http_request_duration_seconds_bucket{endpoint=~"/health/(live|ready)"}[5m])
)
```

## Testing

### Manual Testing

```bash
# Test liveness
curl http://localhost:8080/health/live

# Test readiness
curl http://localhost:8080/health/ready
```

### Unit Tests

Health handlers have comprehensive unit tests in `internal/api/health_handlers_test.go`:
- Liveness endpoint behavior
- Readiness with all dependencies healthy
- Readiness with individual dependencies unhealthy
- Readiness with multiple dependencies unhealthy
- Method validation (only GET allowed)
- Response format validation

### Integration Tests

Integration tests should verify:
- Health checks against real database
- Health checks against real Redis
- Health checks against real LiveKit (if available)
- Proper status codes and response formats

## Troubleshooting

### Pod keeps restarting

Check liveness probe logs:
```bash
kubectl describe pod <pod-name> -n subcults
kubectl logs <pod-name> -n subcults --previous
```

Common causes:
- Application crash during startup
- Deadlock or infinite loop
- Resource exhaustion (OOM)

### Pod not receiving traffic

Check readiness probe status:
```bash
kubectl get pod <pod-name> -n subcults -o jsonpath='{.status.conditions[?(@.type=="Ready")]}'
```

Common causes:
- Database connection failure
- Redis connection failure
- LiveKit server unreachable
- Network policy blocking dependencies

### Slow health checks

If readiness checks are timing out:
1. Check database query performance
2. Check Redis latency
3. Check network connectivity to dependencies
4. Consider increasing timeout in probe configuration

## Best Practices

1. **Keep liveness checks lightweight**: Don't check external dependencies in liveness probes
2. **Set appropriate timeouts**: Allow enough time for dependency checks in readiness probes
3. **Use startup probes for slow-starting applications**: Prevent premature liveness failures
4. **Monitor health check metrics**: Track response times and failure rates
5. **Test failure scenarios**: Verify health checks correctly detect dependency failures

## Security Considerations

Health check endpoints do not leak sensitive information:
- No database connection strings
- No API keys or credentials
- No internal service details
- Only high-level status information

Health checks are intentionally unauthenticated to allow Kubernetes and monitoring systems easy access.
