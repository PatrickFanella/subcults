# HTTP Request Metrics Middleware

## Overview

The HTTP request metrics middleware automatically instruments all HTTP endpoints with Prometheus metrics for observability.

## Metrics Exposed

### http_request_duration_seconds (histogram)
- **Description**: HTTP request duration in seconds
- **Labels**: method, path, status
- **Buckets**: [0.01, 0.1, 0.5, 1.0, 2.0]
- **Example**: 
  ```
  http_request_duration_seconds{method="GET",path="/events",status="200"} 0.123
  ```

### http_requests_total (counter)
- **Description**: Total number of HTTP requests
- **Labels**: method, path, status
- **Example**: 
  ```
  http_requests_total{method="POST",path="/events",status="201"} 42
  ```

### http_request_size_bytes (histogram)
- **Description**: HTTP request size in bytes (from Content-Length header)
- **Labels**: method, path, status
- **Buckets**: Exponential [100, 1000, 10000, ...] up to ~100 MB
- **Example**: 
  ```
  http_request_size_bytes{method="POST",path="/events",status="201"} 1024
  ```

### http_response_size_bytes (histogram)
- **Description**: HTTP response size in bytes
- **Labels**: method, path, status
- **Buckets**: Exponential [100, 1000, 10000, ...] up to ~100 MB
- **Example**: 
  ```
  http_response_size_bytes{method="GET",path="/events",status="200"} 2048
  ```

## Health Check Exclusion

Health check endpoints (`/health` and `/ready`) are automatically excluded from metrics to prevent cardinality explosion and metric pollution.

## Path Normalization

To prevent cardinality explosion, dynamic path segments (IDs) are normalized to route patterns:

- `/events/123` → `/events/{id}`
- `/events/abc-def/cancel` → `/events/{id}/cancel`
- `/streams/stream-456/join` → `/streams/{id}/join`
- `/posts/post-789` → `/posts/{id}`
- `/trust/did:plc:abc123` → `/trust/{id}`

This ensures metrics remain aggregatable across different resource IDs while maintaining useful endpoint-level visibility. Static routes (e.g., `/search/events`, `/payments/checkout`) are not normalized.

## Middleware Chain

The HTTP metrics middleware is integrated into the middleware chain as follows:

```
RateLimiter → HTTPMetrics → RequestID → Logging → Handler
```

This ordering ensures:
1. Rate limiting happens first to block excessive requests early
2. HTTP metrics capture all requests (including rate-limited ones)
3. Request IDs are available for correlation
4. Logging happens with all context

## Performance Impact

The middleware has minimal performance overhead:
- No blocking operations
- Metrics are recorded asynchronously by Prometheus client
- Response writer wrapping adds negligible latency
- Measured impact: <5% (typically <1ms per request)

## Accessing Metrics

Metrics are exposed on the `/metrics` endpoint in Prometheus text format:

```bash
curl http://localhost:8080/metrics
```

If `METRICS_AUTH_TOKEN` is configured, include it:

```bash
curl -H "Authorization: Bearer $METRICS_AUTH_TOKEN" http://localhost:8080/metrics
```

## Example Queries

### Request rate by endpoint
```promql
rate(http_requests_total[5m])
```

### P95 latency by endpoint
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Error rate (4xx and 5xx)
```promql
rate(http_requests_total{status=~"4..|5.."}[5m])
```

### Average request/response sizes
```promql
rate(http_request_size_bytes_sum[5m]) / rate(http_request_size_bytes_count[5m])
rate(http_response_size_bytes_sum[5m]) / rate(http_response_size_bytes_count[5m])
```

## Testing

The middleware includes comprehensive tests:
- Unit tests for individual components
- Integration tests for middleware composition
- Label verification tests
- Health check exclusion tests
- Performance tests

Run tests:
```bash
make test
# or
go test ./internal/middleware/...
```

## Configuration

No additional configuration is required. The middleware is automatically applied to all routes except health checks.

The metrics share the same Prometheus registry as other application metrics (stream metrics, rate limiting metrics, etc.).
