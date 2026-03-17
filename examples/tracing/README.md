# OpenTelemetry Tracing Example

This example demonstrates how to use OpenTelemetry tracing in the Subcults API.

## Prerequisites

- Jaeger running on localhost (ports 4318 for OTLP HTTP, 16686 for UI)
- Go 1.22+ installed

## Running Jaeger

```bash
# From the project root
docker compose up -d jaeger
```

Verify Jaeger is running by visiting http://localhost:16686

## Running the Example

```bash
# From the project root
go run examples/tracing/main.go
```

You should see output like:
```
INFO starting example server on :8081
INFO try these endpoints:
INFO   curl http://localhost:8081/hello
INFO   curl http://localhost:8081/process
INFO   curl http://localhost:8081/error
INFO view traces at http://localhost:16686
```

## Try the Endpoints

### 1. Simple traced request
```bash
curl http://localhost:8081/hello
```

This creates a basic trace with HTTP instrumentation.

### 2. Complex traced request with custom spans
```bash
curl http://localhost:8081/process
```

This creates a trace with:
- HTTP request span (automatic)
- Custom business logic span
- Custom attributes (user.id, operation.type)
- An event marker
- Database query span

### 3. Error tracing
```bash
curl http://localhost:8081/error
```

This demonstrates error tracking with:
- Failed operation span
- Error recorded in span
- Error propagation

## Viewing Traces in Jaeger

1. Open http://localhost:16686
2. Select "tracing-example" from the Service dropdown
3. Click "Find Traces"
4. Click on any trace to see:
   - Span hierarchy and timeline
   - HTTP attributes (method, status, url)
   - Custom attributes
   - Events
   - Error information

## What to Look For

### In the `/hello` trace:
- Single span: `GET /hello`
- HTTP method, status code, route attributes
- Request duration

### In the `/process` trace:
- Parent span: `GET /process`
- Child span: `process_business_logic`
  - Custom attributes: user.id, operation.type
  - Event: validation_complete
- Child span: `query users` (database operation)
  - Database-specific attributes

### In the `/error` trace:
- Parent span: `GET /error`
- Child span: `failing_operation` (with error status)
- Child span: `database_transaction` (also with error)
- Error information in span details

## Code Patterns

### Creating a custom span
```go
ctx, endSpan := tracing.StartSpan(ctx, "operation_name")
defer endSpan(err)  // Pass error or nil
```

### Adding attributes
```go
tracing.SetAttributes(ctx,
    attribute.String("key", "value"),
    attribute.Int("count", 42),
)
```

### Adding events
```go
tracing.AddEvent(ctx, "event_name",
    attribute.Bool("success", true),
)
```

### Database spans
```go
ctx, endSpan := tracing.StartDBSpan(ctx, "table_name", tracing.DBOperationQuery)
defer endSpan(err)
```

## Stopping the Example

Press Ctrl+C to gracefully shutdown the server. This will:
1. Stop accepting new requests
2. Flush any pending spans to Jaeger
3. Clean up resources

## Next Steps

- Read the [full tracing documentation](../../docs/tracing.md)
- Read the [quick start guide](../../docs/tracing-quickstart.md)
- Instrument your own handlers and functions
- Experiment with different sampling rates
- Try connecting to other observability platforms
