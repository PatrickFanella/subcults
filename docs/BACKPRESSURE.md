# Backpressure Handling in Jetstream Consumer

## Overview

The Jetstream consumer implements backpressure handling to gracefully manage slow database writes and prevent queue explosion. This ensures the system remains stable under high load conditions while avoiding message loss.

## Implementation

### Architecture

The backpressure system consists of:

1. **Message Queue**: A buffered channel with capacity of 2000 messages (2x pause threshold)
2. **Separate Goroutines**: 
   - Reader goroutine: Reads from WebSocket and enqueues messages
   - Processor goroutine: Dequeues and processes messages
3. **Backpressure Control**: Monitors queue depth and pauses/resumes consumption

### Thresholds

| Threshold | Value | Description |
|-----------|-------|-------------|
| Pause Threshold | 1000 messages | Pause WebSocket reads when pending > 1000 |
| Resume Threshold | 100 messages | Resume reads when pending < 100 |
| Max Pause Duration | 30 seconds | Emit warning if paused longer than this |
| Queue Buffer Size | 2000 messages | Total capacity = 2x pause threshold |
| Queue Timeout | 5 seconds | Timeout for queuing a single message; triggers reconnect if exceeded |

### Behavior

#### Normal Operation
- Messages flow from WebSocket → Queue → Handler
- Queue depth tracked continuously
- Metrics updated in real-time

#### High Load (Queue Building Up)
1. Queue depth exceeds 1000 messages
2. **Pause** is triggered:
   - WebSocket reads pause (with 100ms polling interval)
   - Pause timestamp recorded
   - `backpressure_paused_total` counter incremented
   - Warning logged with queue depth
3. Processor continues draining queue
4. Queue depth monitored every 100ms

#### Recovery
1. Queue depth drops below 100 messages
2. **Resume** is triggered:
   - WebSocket reads resume
   - Pause duration calculated and recorded
   - `backpressure_resumed_total` counter incremented
   - `backpressure_pause_duration_seconds` histogram updated
   - Info message logged with pause duration

#### Excessive Pause (>30 seconds)
- Warning emitted every 30 seconds during prolonged pause
- Indicates sustained overload condition
- Administrative intervention may be required

### Metrics

All backpressure metrics are exposed via the Prometheus `/metrics` HTTP endpoint on the indexer metrics server:

```prometheus
# Number of times consumption was paused
indexer_backpressure_paused_total

# Number of times consumption resumed
indexer_backpressure_resumed_total

# Histogram of pause durations (seconds)
# Buckets: 0.1, 0.5, 1, 5, 10, 30, 60
indexer_backpressure_pause_duration_seconds

# Current number of pending messages
indexer_pending_messages
```

### Usage

```go
// Create metrics instance
metrics := indexer.NewMetrics()
reg := prometheus.NewRegistry()
metrics.Register(reg)

// Create Jetstream client with backpressure support
config := indexer.DefaultConfig("wss://jetstream.example.com")
handler := func(messageType int, payload []byte) error {
    // Process message (database writes, etc.)
    return nil
}

client, err := indexer.NewClientWithMetrics(config, handler, logger, metrics)
if err != nil {
    log.Fatal(err)
}

// Run client (handles backpressure automatically)
ctx := context.Background()
client.Run(ctx)
```

### Graceful Shutdown

On context cancellation:
1. Reader goroutine stops accepting new messages
2. Processor goroutine drains remaining messages (5s timeout)
3. Messages successfully processed during drain are saved
4. Any unprocessed messages after timeout are lost and logged
5. Connection closed cleanly

**Note**: Message loss during shutdown only occurs if:
- Drain timeout (5s) is exceeded
- Process is forcefully terminated (SIGKILL)
- Handler returns error during drain

For maximum reliability, allow sufficient time for graceful shutdown.

## Message Loss Scenarios

The implementation is designed to minimize message loss, but certain extreme conditions can result in dropped messages:

### 1. Queue Timeout (5 seconds)
**When**: Queue is completely full and cannot accept a message for >5 seconds
**Action**: Connection is closed and reconnection attempted
**Impact**: Messages received during reconnection window may be lost
**Prevention**: 
- Optimize handler processing speed
- Scale database resources
- Monitor `indexer_pending_messages` metric

### 2. Graceful Shutdown Timeout
**When**: Process shutdown initiated with messages still in queue
**Action**: 5-second drain timeout attempted
**Impact**: Messages not processed within timeout are lost
**Prevention**: 
- Allow sufficient shutdown time (>5s)
- Monitor queue depth before shutdown
- Avoid forced termination (SIGKILL)

### 3. Handler Errors
**When**: Message handler returns error during processing
**Action**: Error logged, processing continues
**Impact**: Single message may not be persisted
**Prevention**:
- Implement robust error handling in handler
- Add retry logic for transient failures
- Monitor `indexer_messages_error_total` metric

### Under Normal Operation
- **No message loss**: Queue depth stays below pause threshold
- **Backpressure triggered**: Pause/resume mechanism prevents queue overflow
- **Connection issues**: Automatic reconnection with exponential backoff

## Monitoring

### Key Metrics to Watch

1. **`indexer_pending_messages`**: Should remain low (<100) under normal load
   - Sustained high values (>500) indicate backpressure risk
   - Values near 1000 trigger pause mechanism

2. **`indexer_backpressure_paused_total`**: Should be zero or very low
   - Frequent pauses indicate database/processing bottleneck
   - Consider scaling database or optimizing queries

3. **`indexer_backpressure_pause_duration_seconds`**: Should be short (<1s)
   - Long pauses (>10s) indicate severe overload
   - Multiple pauses >30s require immediate investigation

4. **`indexer_messages_processed_total`**: Message throughput
   - Compare with arrival rate to assess processing lag

### Alerting Recommendations

```yaml
# Alert on sustained backpressure
- alert: IndexerBackpressure
  expr: indexer_pending_messages > 500
  for: 5m
  annotations:
    summary: "Jetstream indexer experiencing backpressure"
    
# Alert on frequent pauses
- alert: IndexerFrequentPauses
  expr: rate(indexer_backpressure_paused_total[5m]) > 0.1
  for: 10m
  annotations:
    summary: "Jetstream indexer pausing frequently"

# Alert on long pause durations
- alert: IndexerLongPause
  expr: histogram_quantile(0.95, rate(indexer_backpressure_pause_duration_seconds_bucket[5m])) > 10
  for: 5m
  annotations:
    summary: "Jetstream indexer experiencing long pauses"
```

## Performance Characteristics

### No Backpressure
- **Latency**: ~1-5ms per message (queue → handler)
- **Throughput**: Limited by handler processing speed
- **Memory**: Minimal (queue near-empty)

### Under Backpressure
- **Latency**: Increases as queue fills
- **Throughput**: Matches processor capacity
- **Memory**: ~4MB for full queue (2000 messages × ~2KB/msg)
- **Pause overhead**: 100ms polling interval

### Recovery Time
- Depends on:
  - Handler processing speed
  - Message arrival rate
  - Queue depth at pause

Typical recovery: 5-30 seconds for 1000 messages at 50 msg/s processing rate

## Testing

Comprehensive test suite covers:

- ✅ Pause triggers when queue > 1000
- ✅ Resume triggers when queue < 100
- ✅ No message loss during backpressure
- ✅ Metrics properly incremented
- ✅ Max pause duration warnings

See `client_test.go` for full test coverage.

## Troubleshooting

### Problem: Frequent backpressure pauses

**Cause**: Message processing slower than arrival rate

**Solutions**:
1. Optimize database queries (indexes, batch inserts)
2. Scale database (more IOPS, connections)
3. Reduce handler processing time
4. Increase worker pool size (if applicable)

### Problem: Long pause durations (>30s)

**Cause**: Sustained overload or database issues

**Solutions**:
1. Check database health (CPU, disk I/O, connections)
2. Review slow query logs
3. Consider rate limiting at ingestion
4. Temporarily increase thresholds (if safe)

### Problem: Messages timing out during queue

**Cause**: Queue completely full for >5 seconds (critical overload)

**Immediate Impact**: Connection closed, automatic reconnection triggered

**Solutions**:
1. **Critical situation** - immediate intervention required
2. Check database health (CPU, disk I/O, connections)
3. Scale database resources vertically or horizontally
4. Review handler processing logic for bottlenecks
5. Consider temporary rate limiting at source
6. Monitor reconnection attempts and success rate

**Note**: Connection reset helps prevent sustained queue overflow. Messages may be lost during reconnection window.

## Future Enhancements

Potential improvements:

1. **Dynamic Thresholds**: Adjust based on processing rate
2. **Batch Processing**: Group messages for bulk database inserts
3. **Priority Queue**: Prioritize certain message types
4. **Disk Spillover**: Persist queue to disk when full
5. **Rate Limiting**: Upstream throttling at WebSocket level

## References

- [Jetstream Indexer Epic #305](https://github.com/subculture-collective/subcults/issues/305) (canonical)
- [Backpressure Logic Issue #371](https://github.com/subculture-collective/subcults/issues/371)
- [Client Implementation](./client.go)
- [Backpressure Tests](./client_test.go)
- [Canonical Roadmap #416](https://github.com/subculture-collective/subcults/issues/416)
