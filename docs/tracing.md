# OpenTelemetry Tracing Documentation

## Overview

The Subcults API implements distributed tracing using OpenTelemetry, providing visibility into request flows across the application. Traces help identify performance bottlenecks, debug issues, and understand system behavior in production.

## Architecture

### Components

1. **Tracer Provider** (`internal/tracing/tracing.go`)
   - Manages OpenTelemetry SDK initialization
   - Configures exporters (OTLP HTTP/gRPC)
   - Controls sampling rates
   - Handles graceful shutdown

2. **Tracing Middleware** (`internal/middleware/tracing.go`)
   - Instruments HTTP handlers
   - Creates spans for each request
   - Propagates trace context using W3C headers
   - Integrates with existing RequestID middleware

3. **Database Helpers** (`internal/tracing/helpers.go`)
   - Span creation for database operations
   - Automatic error recording
   - PostgreSQL-specific attributes

### Trace Propagation

The implementation uses **W3C Trace Context** standard for trace propagation:
- `traceparent` header: Contains trace-id, parent-id, trace-flags
- `tracestate` header: Vendor-specific trace information

This ensures compatibility with other OpenTelemetry-instrumented services and observability platforms.

## Configuration

### Environment Variables

```bash
# Enable/disable tracing
TRACING_ENABLED=true          # Default: false

# Exporter type
TRACING_EXPORTER_TYPE=otlp-http  # Options: otlp-http, otlp-grpc
                                  # Default: otlp-http

# OTLP endpoint
TRACING_OTLP_ENDPOINT=localhost:4318  # HTTP endpoint
# OR
TRACING_OTLP_ENDPOINT=localhost:4317  # gRPC endpoint

# Sampling rate (0.0 to 1.0)
TRACING_SAMPLE_RATE=0.1      # 10% sampling (production)
TRACING_SAMPLE_RATE=1.0      # 100% sampling (development)

# TLS configuration
TRACING_INSECURE=false       # Set to true for local development without TLS
```

### Configuration in Code

The config package (`internal/config/config.go`) provides structured configuration:

```go
type Config struct {
    TracingEnabled      bool    // Enable distributed tracing
    TracingExporterType string  // otlp-http, otlp-grpc
    TracingOTLPEndpoint string  // OTLP endpoint URL
    TracingSampleRate   float64 // 0.0 to 1.0
    TracingInsecure     bool    // Disable TLS
}
```

## Development Setup

### Using Jaeger for Local Development

1. **Start Jaeger with Docker:**

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

2. **Configure the API:**

```bash
# In configs/dev.env
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=localhost:4318
TRACING_SAMPLE_RATE=1.0
TRACING_INSECURE=true
```

3. **Access Jaeger UI:**
   - Open http://localhost:16686 in your browser
   - Select "subcults-api" from the service dropdown
   - View traces and analyze request flows

### Using OTLP gRPC (Alternative)

```bash
TRACING_EXPORTER_TYPE=otlp-grpc
TRACING_OTLP_ENDPOINT=localhost:4317
```

## Production Deployment

### Recommended Setup

1. **Use a dedicated OTLP collector:**
   - Deploy OpenTelemetry Collector as a sidecar or separate service
   - Configure collector to buffer, batch, and export to your backend

2. **Configure sampling:**
   ```bash
   TRACING_SAMPLE_RATE=0.1  # Sample 10% of traces
   ```

3. **Enable TLS:**
   ```bash
   TRACING_INSECURE=false
   TRACING_OTLP_ENDPOINT=collector.example.com:4318
   ```

### Observability Platforms

The OTLP exporter is compatible with many observability platforms:

- **Jaeger**: Native OTLP support
- **Grafana Tempo**: Via OTLP
- **Datadog**: Via OTLP
- **New Relic**: Via OTLP
- **Honeycomb**: Via OTLP
- **Lightstep**: Via OTLP

Example for Grafana Cloud:

```bash
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=tempo-prod-04-prod-us-east-0.grafana.net:443
TRACING_SAMPLE_RATE=0.1
TRACING_INSECURE=false
```

### Performance Considerations

#### Sampling Strategies

- **Development**: `TRACING_SAMPLE_RATE=1.0` (100%) - See all traces
- **Staging**: `TRACING_SAMPLE_RATE=0.5` (50%) - Good coverage for testing
- **Production**: `TRACING_SAMPLE_RATE=0.1` (10%) - Balance cost and visibility

#### Performance Overhead

Based on OpenTelemetry benchmarks:
- **10% sampling**: <1% CPU overhead, <2% memory overhead
- **100% sampling**: ~3-5% CPU overhead, ~5-8% memory overhead

The implementation uses batch span processing to minimize overhead:
- Batch timeout: 5 seconds
- Max batch size: 512 spans

## Instrumentation Points

### HTTP Requests

All HTTP requests are automatically instrumented:

```
Span: GET /events
├─ Attributes:
│  ├─ http.method: GET
│  ├─ http.url: /events
│  ├─ http.status_code: 200
│  └─ http.route: /events
└─ Context: trace-id, span-id propagated via headers
```

### Database Queries

Use the provided helpers to instrument database operations:

```go
import "github.com/onnwee/subcults/internal/tracing"

func (r *Repository) GetScene(ctx context.Context, id string) (*Scene, error) {
    ctx, endSpan := tracing.StartDBSpan(ctx, "scenes", tracing.DBOperationQuery)
    defer endSpan(nil) // Pass error if query fails
    
    // Perform database query
    var scene Scene
    err := r.db.QueryRowContext(ctx, "SELECT * FROM scenes WHERE id = $1", id).Scan(&scene)
    if err != nil {
        endSpan(err) // Records error in span
        return nil, err
    }
    
    return &scene, nil
}
```

### External API Calls

Instrument calls to external services (LiveKit, Stripe, etc.):

```go
import "github.com/onnwee/subcults/internal/tracing"

func (c *StripeClient) CreatePaymentIntent(ctx context.Context, amount int64) error {
    ctx, endSpan := tracing.StartSpan(ctx, "stripe.create_payment_intent")
    defer endSpan(nil)
    
    // Add attributes
    tracing.SetAttributes(ctx,
        attribute.Int64("payment.amount", amount),
        attribute.String("payment.currency", "usd"),
    )
    
    // Make API call
    intent, err := c.client.PaymentIntents.New(params)
    if err != nil {
        endSpan(err)
        return err
    }
    
    return nil
}
```

### Background Jobs

For background processing, create a root span:

```go
func (w *Worker) ProcessJob(ctx context.Context, job *Job) error {
    ctx, endSpan := tracing.StartSpan(ctx, "job.process")
    defer endSpan(nil)
    
    tracing.SetAttributes(ctx,
        attribute.String("job.id", job.ID),
        attribute.String("job.type", job.Type),
    )
    
    // Process job...
    
    return nil
}
```

## Querying Traces

### Finding Request Flows

1. **In Jaeger UI:**
   - Select "subcults-api" service
   - Use filters: status_code, duration, operation
   - View full request trace with all spans

2. **Common Queries:**
   - Slow requests: `duration > 1s`
   - Failed requests: `error=true`
   - Database queries: `operation=query*`
   - Specific endpoints: `http.route=/events`

### Analyzing Performance

1. **Identify bottlenecks:**
   - Sort spans by duration
   - Look for long-running database queries
   - Check external API call latency

2. **Debug errors:**
   - Filter by error status
   - Examine error messages and stack traces
   - Trace error propagation through services

## Troubleshooting

### Traces not appearing

1. **Check configuration:**
   ```bash
   # Verify environment variables
   env | grep TRACING
   ```

2. **Check logs:**
   ```bash
   # Look for tracing initialization
   grep "tracing initialized" /var/log/subcults-api.log
   ```

3. **Test connectivity:**
   ```bash
   # Verify OTLP endpoint is reachable
   curl http://localhost:4318/v1/traces
   ```

### High overhead

1. **Reduce sampling rate:**
   ```bash
   TRACING_SAMPLE_RATE=0.05  # Reduce to 5%
   ```

2. **Check batch settings:**
   - Current: 5s timeout, 512 max batch size
   - Increase timeout for lower frequency exports

### Missing spans

1. **Check sampling:**
   - Not all requests are sampled based on `TRACING_SAMPLE_RATE`
   - Increase sampling rate in development

2. **Verify context propagation:**
   - Ensure context is passed through all function calls
   - Use `ctx` parameter consistently

## Security Considerations

### Data Privacy

- Traces may contain sensitive information (user IDs, request paths)
- Configure your observability platform's data retention policies
- Consider PII scrubbing in OTLP collector

### Authentication

For production OTLP endpoints:

1. **Use TLS:**
   ```bash
   TRACING_INSECURE=false
   ```

2. **Add authentication headers:**
   - Modify `createOTLPHTTPExporter` to add auth headers
   - Use API keys or bearer tokens

3. **Network isolation:**
   - Use private networks for collector communication
   - Firewall rules to restrict access

## Metrics

OpenTelemetry also supports metrics collection. Future enhancements could include:

- Request rate metrics
- Error rate metrics
- Latency histograms
- Custom business metrics

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)

## Implemented Instrumentation

### Repository Layer

The following repositories have been instrumented with tracing:

#### Indexer Repository (`internal/indexer/repository.go`)

All database operations in the PostgreSQL record repository include tracing:

- **UpsertRecord**: Traces the full upsert transaction including:
  - Idempotency checks
  - Table-specific upsert operations
  - Transaction commit
  - Success/failure events

- **DeleteRecord**: Traces soft-delete operations with:
  - Transaction management
  - Row count tracking
  - Error recording

- **CheckIdempotencyKey**: Traces idempotency validation queries

Example trace flow for record ingestion:
```
exec ingestion_idempotency (120ms)
├─ Check idempotency (15ms)
├─ Upsert scene/event/post (80ms)
├─ Store idempotency key (20ms)
└─ Event: record_upserted
```

#### Scene Handlers (`internal/api/scene_handlers.go`)

API handlers include tracing for business logic:

- **CreateScene**:
  - `check_duplicate_scene_name`: Duplicate validation
  - `insert_scene`: Scene creation with privacy enforcement
  - `get_created_scene`: Retrieve created scene

Example trace for scene creation:
```
POST /scenes (250ms)
├─ check_duplicate_scene_name (50ms)
│  └─ query scenes (45ms)
├─ insert_scene (150ms)
│  └─ insert scenes (140ms)
└─ get_created_scene (30ms)
   └─ query scenes (25ms)
```

### Testing

Comprehensive integration tests are available in `internal/tracing/integration_test.go`:

- **TestEndToEndTracing**: Validates complete trace creation and propagation
- **TestTracingDisabled**: Ensures graceful degradation when tracing is off
- **TestTraceContextPropagation**: Verifies W3C Trace Context headers

Run tests:
```bash
go test -v ./internal/tracing/...
```

### Example Application

A complete working example is available in `examples/tracing/main.go`:

- HTTP server with tracing middleware
- Custom span creation
- Database operation simulation
- Error handling
- Event tracking

Run the example:
```bash
# Start Jaeger
docker run -d -p 16686:16686 -p 4318:4318 \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/all-in-one:latest

# Run example
cd examples/tracing
go run main.go

# Make requests
curl http://localhost:8081/hello
curl http://localhost:8081/process
curl http://localhost:8081/error

# View traces
open http://localhost:16686
```

### Trace Attributes

Standard attributes used across the codebase:

**HTTP Operations:**
- `http.method`: HTTP method (GET, POST, etc.)
- `http.url`: Request path
- `http.status_code`: Response status
- `http.route`: Matched route pattern

**Database Operations:**
- `db.system`: Always "postgresql"
- `db.operation`: query, insert, update, delete, exec
- `db.sql.table`: Target table name

**Business Logic:**
- `collection`: AT Protocol collection type
- `did`: Decentralized Identifier (when not PII)
- `rkey`: Record key
- `scene_id`: Scene identifier
- `visibility`: Scene visibility setting
- `allow_precise`: Location precision flag

**Events:**
- `duplicate_record_skipped`: Idempotent record detected
- `record_upserted`: Record successfully created/updated
- `record_deleted`: Record soft-deleted
- `scene_created`: Scene created successfully

### Privacy Compliance

All instrumentation follows privacy-first principles:

- ✅ **Opaque IDs only**: Scene IDs, event IDs, user DIDs
- ✅ **No PII in span names**: Generic operation names
- ✅ **No PII in attributes**: No names, emails, locations
- ✅ **Sanitized data**: All user input sanitized before tracing
- ❌ **Never trace**: Passwords, tokens, precise coordinates, PII

