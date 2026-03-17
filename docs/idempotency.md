# Idempotency Key Middleware

## Overview

The idempotency key middleware prevents duplicate payment operations when clients retry requests due to network issues or timeouts. By requiring an `Idempotency-Key` header on critical POST endpoints, the system can safely return cached responses for duplicate requests without re-executing the business logic.

## Architecture

### Components

1. **Middleware** (`internal/middleware/idempotency.go`)
   - Validates idempotency key presence and format
   - Checks for existing keys in repository
   - Returns cached responses for duplicates
   - Stores successful responses for future lookups

2. **Repository** (`internal/idempotency/repository.go`)
   - In-memory implementation for testing
   - Interface for production Postgres implementation
   - Stores idempotency keys with response metadata

3. **Cleanup** (`internal/idempotency/cleanup.go`)
   - Periodic cleanup utilities
   - Removes keys older than 24 hours
   - Prevents unbounded storage growth

### Data Model

The `idempotency_keys` table stores:

```sql
CREATE TABLE idempotency_keys (
    key VARCHAR(64) PRIMARY KEY,
    method VARCHAR(10) NOT NULL,
    route VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payment_id UUID,
    response_hash VARCHAR(64) NOT NULL,
    status VARCHAR(50) NOT NULL,
    response_body TEXT NOT NULL,
    response_status_code INT NOT NULL
);
```

## Usage

### Client Implementation

Clients must include the `Idempotency-Key` header on POST requests to `/payments/checkout`:

```http
POST /payments/checkout HTTP/1.1
Host: api.subcults.com
Content-Type: application/json
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
Authorization: Bearer <token>

{
  "scene_id": "scene-123",
  "items": [{"price_id": "price_abc", "quantity": 2}],
  "success_url": "https://example.com/success",
  "cancel_url": "https://example.com/cancel"
}
```

**Key Requirements:**
- Must be unique per request
- Maximum length: 64 characters
- Recommended: Use UUIDv4
- Must be sent on every retry attempt
- Different keys = different requests

### Response Behavior

#### First Request (New Key)
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "session_url": "https://checkout.stripe.com/pay/cs_test123",
  "session_id": "cs_test123"
}
```

#### Duplicate Request (Same Key)
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "session_url": "https://checkout.stripe.com/pay/cs_test123",
  "session_id": "cs_test123"
}
```

The response is identical and the Stripe API is **not** called again.

#### Missing Header
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "missing_idempotency_key",
  "message": "Idempotency-Key header is required for this request"
}
```

#### Invalid Key (Too Long)
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "idempotency_key_too_long",
  "message": "Idempotency-Key exceeds maximum length of 64 characters"
}
```

## Configuration

### Protected Routes

Only routes explicitly configured in `main.go` require idempotency keys:

```go
idempotencyRoutes := map[string]bool{
    "/payments/checkout": true,
}
```

### Expiry

Keys are automatically cleaned up after 24 hours:

```go
const DefaultExpiry = 24 * time.Hour
```

## Testing

### Unit Tests

Run idempotency tests:

```bash
go test ./internal/idempotency/... -v
go test ./internal/middleware -run Idempotency -v
```

### Integration Tests

Test with the payment handler:

```bash
go test ./internal/api -run TestCreateCheckoutSession.*Idempotency -v
```

### Manual Testing

Using curl:

```bash
# First request
curl -X POST https://api.subcults.com/payments/checkout \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-key-123" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "scene_id": "scene-123",
    "items": [{"price_id": "price_abc", "quantity": 1}],
    "success_url": "https://example.com/success",
    "cancel_url": "https://example.com/cancel"
  }'

# Duplicate request (same key, same response)
curl -X POST https://api.subcults.com/payments/checkout \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-key-123" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "scene_id": "scene-123",
    "items": [{"price_id": "price_abc", "quantity": 1}],
    "success_url": "https://example.com/success",
    "cancel_url": "https://example.com/cancel"
  }'
```

## Monitoring

### Metrics to Track

- Total idempotency keys stored
- Cache hit rate (duplicate requests)
- Oldest key age
- Cleanup job execution

### SQL Queries

```sql
-- Check total keys
SELECT COUNT(*) FROM idempotency_keys;

-- Keys by age
SELECT 
    COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 hour') AS last_hour,
    COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours') AS last_day,
    COUNT(*) AS total
FROM idempotency_keys;

-- Keys by route
SELECT route, COUNT(*) AS count
FROM idempotency_keys
GROUP BY route
ORDER BY count DESC;
```

## Production Deployment

See [docs/idempotency-cleanup.md](./idempotency-cleanup.md) for detailed cleanup job setup instructions including:
- Cron job configuration
- Kubernetes CronJob manifests
- Built-in periodic cleanup
- SQL maintenance queries

## Security Considerations

1. **Key Privacy**: Idempotency keys are stored server-side only
2. **Length Validation**: Maximum 64 characters prevents abuse
3. **Response Isolation**: Cached responses are only returned to requests with matching keys
4. **Expiry**: 24-hour cleanup prevents long-term storage of sensitive data
5. **Error Responses**: Not cached to prevent masking issues

## Performance

- **Latency**: ~1ms overhead for key lookup
- **Storage**: ~500 bytes per key
- **Cleanup**: O(n) deletion with indexed timestamp
- **Cache Hit**: Skips entire handler execution including Stripe API calls

## Limitations

1. **In-Memory Repository**: Current implementation uses in-memory storage; production should use Postgres
2. **Single-Instance**: In-memory repository doesn't work with multiple API instances (requires shared database)
3. **Success Only**: Only 2xx responses are cached
4. **No Retry Logic**: Clients must implement their own retry with exponential backoff

## Future Enhancements

- [ ] Postgres repository implementation
- [ ] Prometheus metrics for cache hit/miss rates
- [ ] Redis cache layer for hot keys
- [ ] Configurable expiry per route
- [ ] Admin API for key inspection/deletion
