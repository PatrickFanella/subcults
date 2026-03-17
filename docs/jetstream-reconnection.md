# Jetstream Indexer - Reconnection and Resume

## Overview

The Jetstream indexer implements resilient reconnection logic with automatic resume capability to handle network interruptions gracefully. This ensures no message loss and maintains processing continuity across disconnections.

## Features

### 1. Exponential Backoff Reconnection

The client automatically reconnects with exponential backoff and jitter:
- **Base delay**: 100ms (configurable)
- **Max delay**: 30s (configurable)
- **Jitter**: 50% (configurable, prevents thundering herd)
- **Backoff formula**: `baseDelay * 2^attempts`, capped at maxDelay

Example backoff sequence:
```
Attempt 1: 100ms + jitter
Attempt 2: 200ms + jitter
Attempt 3: 400ms + jitter
Attempt 4: 800ms + jitter
Attempt 5: 1.6s + jitter
...
Attempt N: 30s + jitter (max)
```

### 2. Sequence Tracking and Resume

The indexer tracks the last successfully processed message using the `time_us` field from Jetstream messages as a cursor:

- **Storage**: Persisted in `indexer_state.cursor` column
- **Resume URL**: Automatically appends `?cursor={sequence}` parameter on reconnect
- **Idempotency**: Updates sequence after successful processing to prevent reprocessing
- **Monotonic**: Only updates if new sequence > old sequence

#### Sequence Update Points

The sequence is updated after:
- ✅ Successful record upserts
- ✅ Successful record deletes
- ✅ Non-matched records (to skip on resume)
- ✅ Invalid records (to skip on resume)

This ensures no messages are lost or reprocessed on reconnect.

### 3. Max Retry Attempts with Alerting

Configurable maximum retry attempts with alert logging:
- **Default**: 5 consecutive attempts
- **Behavior**: Logs at ERROR level when limit exceeded for monitoring/alerting
- **Continues**: Does not give up after max attempts, but alerts are triggered

Example log when limit is reached:
```
ERROR: max reconnection attempts reached - alerting required
  max_attempts=5 current_attempt=6 error="connection refused"
```

### 4. Metrics

The indexer exposes Prometheus metrics for monitoring:

| Metric | Type | Description |
|--------|------|-------------|
| `indexer_reconnection_attempts_total` | Counter | Total reconnection attempts |
| `indexer_reconnection_success_total` | Counter | Successful reconnections |
| `indexer_pending_messages` | Gauge | Messages queued for processing |
| `indexer_processing_lag_seconds` | Gauge | Time lag from message creation |

## Configuration

### Environment Variables

```bash
# Jetstream WebSocket URL (required)
JETSTREAM_URL="wss://jetstream1.us-east.bsky.network/subscribe"

# Database URL for sequence persistence (required for production)
DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=require"

# Metrics and monitoring
METRICS_PORT="9090"
INTERNAL_AUTH_TOKEN="secret-token-for-metrics"

# Application environment
SUBCULT_ENV="production"  # or "development"
```

### Code Configuration

```go
config := indexer.Config{
    URL:              jetstreamURL,
    BaseDelay:        100 * time.Millisecond,
    MaxDelay:         30 * time.Second,
    JitterFactor:     0.5,
    MaxRetryAttempts: 5,
}
```

## Usage

### Starting the Indexer

```bash
# With database (production)
export DATABASE_URL="postgres://..."
go run ./cmd/indexer

# Without database (testing, in-memory)
go run ./cmd/indexer
```

### Startup Logs

The indexer logs resume status on startup:

```
INFO: will resume from last processed sequence
  cursor=1234567890 last_message_time=2026-01-30T10:15:30Z
```

Or if starting fresh:
```
INFO: starting from beginning (no previous sequence found)
```

### Monitoring Reconnections

Watch metrics endpoint:
```bash
curl http://localhost:9090/internal/indexer/metrics | grep reconnection
```

Expected output:
```
indexer_reconnection_attempts_total 15
indexer_reconnection_success_total 14
```

## Architecture

### Sequence Tracking Flow

```
┌──────────────┐
│   Jetstream  │
│   WebSocket  │
└──────┬───────┘
       │ CBOR Message (time_us: 123456)
       ▼
┌──────────────┐
│    Client    │
│  Read Loop   │
└──────┬───────┘
       │ Queue Message
       ▼
┌──────────────┐
│   Message    │
│  Processor   │
└──────┬───────┘
       │ Call Handler
       ▼
┌──────────────────┐
│  Message Handler │
│  - Filter        │
│  - Validate      │
│  - Persist       │
│  - Update Seq    │
└──────┬───────────┘
       │ Success
       ▼
┌──────────────────┐
│ indexer_state    │
│ cursor=123456    │
└──────────────────┘
```

### Reconnection Flow

```
Connection Lost
       │
       ▼
┌──────────────────┐
│ Compute Backoff  │
│ 100ms * 2^N      │
└──────┬───────────┘
       │ Wait
       ▼
┌──────────────────┐
│ Load Last Cursor │
│ from DB          │
└──────┬───────────┘
       │ cursor=123456
       ▼
┌──────────────────┐
│ Connect with     │
│ ?cursor=123456   │
└──────┬───────────┘
       │
       ├─ Success ──► Reset retry count
       │              Track success metric
       │              Resume processing
       │
       └─ Failure ──► Increment retry count
                      Check max attempts
                      Loop back
```

## Testing

### Unit Tests

```bash
# Test sequence tracking
go test -v ./internal/indexer -run TestInMemorySequenceTracker

# Test reconnection logic
go test -v ./internal/indexer -run TestClient_SequenceTracking

# Test max retry behavior
go test -v ./internal/indexer -run TestClient_MaxRetryAttempts
```

### Integration Tests

```bash
# Full end-to-end tests
go test -v ./internal/indexer -run TestIntegration
```

## Troubleshooting

### Issue: Messages being reprocessed after reconnect

**Cause**: Sequence not being updated after processing

**Solution**: Check handler logs for sequence update errors:
```
WARN: failed to update sequence after upsert error="..."
```

### Issue: Too many reconnection attempts

**Cause**: Network instability or Jetstream unavailable

**Check**:
1. Network connectivity: `ping jetstream1.us-east.bsky.network`
2. Metrics: `curl localhost:9090/internal/indexer/metrics | grep reconnection`
3. Logs for ERROR level messages about max attempts

**Solution**: 
- Verify Jetstream URL is correct
- Check firewall/network policies
- Consider adjusting `MaxRetryAttempts` or `MaxDelay`

### Issue: Sequence not resuming on restart

**Cause**: Database connection issue or migration not applied

**Check**:
1. Database connection: `psql $DATABASE_URL -c "SELECT cursor FROM indexer_state;"`
2. Table exists: Migration 000000 should have created `indexer_state`

**Solution**:
```bash
# Apply migrations
make migrate-up

# Verify table
psql $DATABASE_URL -c "\d indexer_state"
```

## Performance Considerations

### Memory Usage

- Message queue buffer: 2000 messages (configurable via `QueueBufferSize`)
- Each message: ~1-10KB depending on content
- Total queue memory: ~2-20MB typical

### Database Load

- Sequence updates: 1 UPDATE per message processed
- Indexed on `id` for fast updates
- Minimal overhead (~1ms per update)

### Network Reconnection

- Backoff prevents overwhelming Jetstream
- Jitter prevents synchronized reconnection storms
- Max delay caps worst-case retry time

## Related Documentation

- [Idempotency and Cleanup](./idempotency-cleanup.md)
- [Architecture Overview](./ARCHITECTURE.md)
- [GitHub Issue #372](https://github.com/subculture-collective/subcults/issues/372)
