# Tracing Quick Start Guide

## Overview

This guide will help you get started with distributed tracing in the Subcults API using Jaeger and OpenTelemetry.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.22+ installed
- Make installed

## Step 1: Start Jaeger

Start Jaeger using Docker Compose:

```bash
docker compose up -d jaeger
```

Verify Jaeger is running:
- Open http://localhost:16686 in your browser
- You should see the Jaeger UI

## Step 2: Configure the API

Create or update your `configs/dev.env` file:

```bash
# Copy the example file if you haven't already
cp configs/dev.env.example configs/dev.env

# Add tracing configuration
cat >> configs/dev.env << EOF

# Tracing Configuration
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=localhost:4318
TRACING_SAMPLE_RATE=1.0
TRACING_INSECURE=true
EOF
```

## Step 3: Start the API Server

```bash
# Build the API
make build-api

# Run the API with environment variables
set -a && . ./configs/dev.env && set +a && ./bin/api
```

You should see a log message:
```
INFO tracing initialized exporter=otlp-http endpoint=localhost:4318 sample_rate=1
```

## Step 4: Generate Traces

Make some requests to the API:

```bash
# Health check (creates a trace)
curl http://localhost:8080/health

# Example event search (if you have data)
curl http://localhost:8080/search/events?q=music

# Create multiple requests to see more traces
for i in {1..10}; do
  curl http://localhost:8080/health
  sleep 0.5
done
```

## Step 5: View Traces in Jaeger

1. Open http://localhost:16686
2. Select "subcults-api" from the Service dropdown
3. Click "Find Traces"
4. Click on any trace to see detailed span information

### What to Look For

- **Trace Timeline**: Shows request duration and span hierarchy
- **HTTP Attributes**: Method, URL, status code
- **Request ID**: Correlation with logs
- **Error Information**: If requests failed

## Step 6: Test Database Tracing (Optional)

To see database query spans, you'll need to:

1. Instrument repository methods with tracing helpers
2. Make requests that trigger database queries

Example instrumentation:

```go
import "github.com/onnwee/subcults/internal/tracing"

func (r *SceneRepository) GetByID(ctx context.Context, id string) (*Scene, error) {
    ctx, endSpan := tracing.StartDBSpan(ctx, "scenes", tracing.DBOperationQuery)
    defer endSpan(nil)
    
    // Your database query here
    var scene Scene
    err := r.db.QueryRowContext(ctx, "SELECT * FROM scenes WHERE id = $1", id).Scan(&scene)
    if err != nil {
        endSpan(err)
        return nil, err
    }
    
    return &scene, nil
}
```

## Troubleshooting

### Traces Not Appearing

1. **Check API logs:**
   ```bash
   grep "tracing" /path/to/api/logs
   ```

2. **Verify Jaeger is running:**
   ```bash
   docker compose ps jaeger
   curl http://localhost:4318/v1/traces
   ```

3. **Check configuration:**
   ```bash
   env | grep TRACING
   ```

### Performance Issues

If you notice high CPU/memory usage:

1. **Reduce sampling:**
   ```bash
   TRACING_SAMPLE_RATE=0.1  # Sample only 10%
   ```

2. **Check Jaeger resource usage:**
   ```bash
   docker stats subcults-jaeger
   ```

## Advanced Usage

### Using OTLP gRPC Instead of HTTP

```bash
TRACING_EXPORTER_TYPE=otlp-grpc
TRACING_OTLP_ENDPOINT=localhost:4317
```

### Sending Traces to External Platform

For production or testing with cloud platforms:

```bash
# Example for Grafana Cloud Tempo
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=tempo-prod-04-prod-us-east-0.grafana.net:443
TRACING_SAMPLE_RATE=0.1
TRACING_INSECURE=false
```

### Running with Docker Compose

When the API service is uncommented in docker-compose.yml:

```bash
# Start all services including API
docker compose up -d

# View API logs
docker compose logs -f api

# Access Jaeger UI
# http://localhost:16686
```

## Next Steps

1. **Read the full documentation**: `docs/tracing.md`
2. **Instrument more code**: Add spans to repository methods, external API calls
3. **Analyze traces**: Look for slow queries, bottlenecks, errors
4. **Set up alerts**: Configure your observability platform for SLO alerts

## Stopping Services

```bash
# Stop Jaeger
docker compose stop jaeger

# Remove Jaeger
docker compose down jaeger

# Stop all services
docker compose down
```

## References

- [Full Tracing Documentation](./tracing.md)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [Jaeger Getting Started](https://www.jaegertracing.io/docs/1.6/getting-started/)
