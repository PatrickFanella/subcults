# Jetstream Indexer - Transaction Consistency and Atomicity

This package implements a robust, transactional indexer for AT Protocol records from Jetstream with full atomicity guarantees and idempotency support.

## Overview

The indexer consumes real-time AT Protocol commit streams from Jetstream, validates records, and persists them to PostgreSQL with the following guarantees:

- **Atomic transactions**: All-or-nothing commits per record
- **Idempotency**: Duplicate records are safely ignored on replay/reconnection
- **Graceful failure handling**: Individual failures don't crash the stream
- **Privacy compliance**: Automatic cleanup of processing metadata

**AT Protocol Schema**: See [docs/AT_PROTOCOL_SCHEMA.md](../../docs/AT_PROTOCOL_SCHEMA.md) for complete record schema definitions and validation rules.

## Architecture

```
Jetstream WebSocket → Client → Filter → Repository → PostgreSQL
                         ↓         ↓          ↓
                     Metrics   Validation  Transaction
```

### Components

#### 1. RecordRepository

Interface for transactional database operations:

```go
type RecordRepository interface {
    UpsertRecord(ctx context.Context, record *FilterResult) (string, bool, error)
    DeleteRecord(ctx context.Context, did, collection, rkey string) error
    CheckIdempotencyKey(ctx context.Context, key string) (bool, error)
}
```

**Implementations:**
- `PostgresRecordRepository`: Full transaction support with BEGIN/COMMIT/ROLLBACK
- `InMemoryRecordRepository`: Thread-safe in-memory storage for testing

#### 2. RecordFilter

Validates AT Protocol records against lexicon schemas:

```go
filter := NewRecordFilter(metrics)
result := filter.FilterCBOR(payload)

if result.Valid && result.Matched {
    // Process the record
}
```

Supported collections:
- `app.subcult.scene` - Music scenes
- `app.subcult.event` - Events at scenes
- `app.subcult.post` - Scene-related posts
- `app.subcult.alliance` - Trust relationships between scenes

**Schema Details**: See [AT Protocol Schema documentation](../../docs/AT_PROTOCOL_SCHEMA.md) for complete field requirements and validation rules.

#### 3. CleanupService

Manages idempotency key retention:

```go
cleanup := NewCleanupService(db, logger, CleanupConfig{
    RetentionPeriod: 24 * time.Hour,
    CleanupInterval: 1 * time.Hour,
})
cleanup.Start(ctx)
defer cleanup.Stop()
```

## Database Schema

### Idempotency Table

```sql
CREATE TABLE ingestion_idempotency (
    idempotency_key VARCHAR(64) PRIMARY KEY,  -- SHA256(did:collection:rkey:rev)
    did VARCHAR(255) NOT NULL,
    collection VARCHAR(255) NOT NULL,
    rkey VARCHAR(255) NOT NULL,
    rev VARCHAR(255) NOT NULL,
    record_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Record Tables

Each collection has `record_did` and `record_rkey` columns with unique constraints:

```sql
CREATE UNIQUE INDEX idx_scenes_record_key 
    ON scenes(record_did, record_rkey) 
    WHERE record_did IS NOT NULL AND record_rkey IS NOT NULL;
```

## Usage

### Basic Setup

```go
// Initialize database
db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}

// Create repository with transaction support
repo := indexer.NewPostgresRecordRepository(db, logger)

// Create filter
filter := indexer.NewRecordFilter(indexer.NewFilterMetrics())

// Message handler
handler := func(messageType int, payload []byte) error {
    result := filter.FilterCBOR(payload)
    
    if !result.Valid || !result.Matched {
        return nil
    }
    
    if result.Operation == "delete" {
        return repo.DeleteRecord(ctx, result.DID, result.Collection, result.RKey)
    }
    
    _, _, err := repo.UpsertRecord(ctx, &result)
    return err
}

// Start client
config := indexer.DefaultConfig("wss://jetstream.example.com/subscribe")
client, err := indexer.NewClientWithMetrics(config, handler, logger, metrics)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
if err := client.Run(ctx); err != nil {
    log.Fatal(err)
}
```

### Idempotency Key Generation

Keys are deterministic SHA256 hashes:

```go
func generateIdempotencyKey(did, collection, rkey, rev string) string {
    data := fmt.Sprintf("%s:%s:%s:%s", did, collection, rkey, rev)
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

This ensures:
- Same record revision → same key
- Different revision → different key (reprocessed)
- Collision probability: ~0 (SHA256 is cryptographically secure)

## Transaction Flow

### Upsert Operation

```
1. BEGIN TRANSACTION
2. Check idempotency key in database
3. If exists → COMMIT (skip, already processed)
4. If not exists:
   a. Check if record exists (by did + rkey)
   b. INSERT or UPDATE record
   c. INSERT idempotency key
   d. COMMIT
5. On error → ROLLBACK
```

### Delete Operation

```
1. BEGIN TRANSACTION
2. DELETE FROM <table> WHERE record_did = $1 AND record_rkey = $2
3. COMMIT
4. On error → ROLLBACK
```

## Error Handling

### Database Failures

All database errors trigger automatic rollback:

```go
defer func() {
    if err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            logger.Error("rollback failed", 
                "error", rbErr,
                "original_error", err)
        }
    }
}()
```

### Validation Failures

Invalid records are logged but don't crash the stream:

```go
if !result.Valid {
    metrics.IncMessagesError()
    logger.Warn("validation failed",
        "collection", result.Collection,
        "error", result.Error)
    return nil  // Continue processing
}
```

## Metrics

Track indexer health with Prometheus metrics:

```go
// Messages processed
indexer_messages_processed_total

// Successful upserts
indexer_database_upserts_total

// Database write failures
indexer_database_writes_failed_total

// Processing lag
indexer_processing_lag_seconds

// Pending messages (backpressure)
indexer_pending_messages
```

## Testing

Run all tests:

```bash
go test ./internal/indexer/...
```

Run specific test suites:

```bash
# Repository tests
go test -v ./internal/indexer/... -run TestInMemoryRepository

# Transaction atomicity
go test -v ./internal/indexer/... -run TestTransactionAtomicity

# Cleanup service
go test -v ./internal/indexer/... -run Cleanup

# Integration tests (requires Postgres)
go test -v ./internal/indexer/... -run Integration
```

### Test Coverage

- **106 tests** covering all components
- Repository operations (upsert, delete, idempotency)
- Concurrent access and thread safety
- Transaction rollback scenarios
- Cleanup service lifecycle
- Backpressure handling
- CBOR encoding/decoding
- Record filtering and validation

## Performance

### Throughput

- Target: 1000+ commits/second
- Backpressure kicks in at >1000 pending messages
- Auto-resume when queue clears to <100 messages

### Latency

- p50: <50ms per record
- p95: <300ms per record
- Lag calculation: message timestamp → processing complete

### Database Optimization

- Unique indexes on (record_did, record_rkey) for fast upserts
- Index on ingestion_idempotency.created_at for efficient cleanup
- Connection pooling recommended (10-50 connections)

## Privacy & Data Retention

### Idempotency Key Cleanup

Default retention: **24 hours**

```go
// Automatic cleanup every hour
cleanup := NewCleanupService(db, logger, CleanupConfig{
    RetentionPeriod: 24 * time.Hour,
    CleanupInterval: 1 * time.Hour,
})
```

Cleanup deletes keys older than retention period:

```sql
DELETE FROM ingestion_idempotency
WHERE created_at < NOW() - INTERVAL '24 hours'
```

### Data Minimization

Only essential fields are stored:
- Record key components (did, collection, rkey, rev)
- Record ID (for debugging)
- Timestamp (for cleanup)

**No sensitive data** (record payload) is persisted in idempotency table.

## Deployment

### Environment Variables

```bash
# Required
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=require
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# Optional
METRICS_PORT=9090                    # Prometheus metrics endpoint
INTERNAL_AUTH_TOKEN=secret           # Protect metrics endpoint
SUBCULT_ENV=production               # Logging format (json vs text)
```

### Database Migration

Run migrations before deployment:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

Required migrations:
- `000008_at_protocol_record_keys.up.sql` - Adds record_did/record_rkey columns
- `000025_ingestion_idempotency.up.sql` - Creates idempotency table

### Monitoring

Key metrics to alert on:

1. **Lag**: `indexer_processing_lag_seconds > 60` (falling behind)
2. **Failures**: `rate(indexer_database_writes_failed_total[5m]) > 10` (persistent errors)
3. **Backpressure**: `indexer_pending_messages > 1000` (queue building up)
4. **Cleanup failures**: Check logs for `"cleanup failed"`

## Troubleshooting

### Issue: Duplicate records despite idempotency

**Cause**: Database transaction isolation level too low

**Solution**: Ensure `READ COMMITTED` or higher:

```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelReadCommitted,
})
```

### Issue: Idempotency table growing unbounded

**Cause**: Cleanup service not running or failing

**Solution**: 
1. Check cleanup service logs
2. Verify database permissions for DELETE
3. Manually clean if needed:
   ```sql
   DELETE FROM ingestion_idempotency WHERE created_at < NOW() - INTERVAL '24 hours';
   ```

### Issue: High database CPU usage

**Cause**: Too many concurrent transactions

**Solution**:
1. Reduce backpressure threshold
2. Increase database connection pool
3. Add database read replicas for queries

### Issue: Processing lag increasing

**Cause**: Backpressure or slow database writes

**Solution**:
1. Check `indexer_pending_messages` metric
2. Optimize database indexes
3. Scale database vertically
4. Consider batching (future enhancement)

## Future Enhancements

1. **Batch processing**: Group multiple records into single transaction
2. **Partition idempotency table**: By date for faster cleanup
3. **Postgres-specific repository**: Switch from in-memory to Postgres when DATABASE_URL is set
4. **Dead letter queue**: Persist failed records for manual review
5. **Metrics dashboard**: Pre-built Grafana dashboard

## References

- **AT Protocol Schema**: [docs/AT_PROTOCOL_SCHEMA.md](../../docs/AT_PROTOCOL_SCHEMA.md) - Subcults record schema definitions
- AT Protocol Spec: https://atproto.com/specs/repository
- Jetstream Docs: https://github.com/ericvolp12/jetstream
- Migration 000025: `migrations/000025_ingestion_idempotency.up.sql`
- Related Issues: #370 (Transaction consistency), #305 (Jetstream epic)
