# API Client Reference

## Overview

The API client provides a centralized HTTP client with automatic token refresh, retry logic, timeout support, and telemetry hooks. It handles common concerns like authentication, error normalization, and resilience in a consistent way across the application.

## Features

- **Automatic Authentication**: Injects access tokens automatically
- **Token Refresh**: Handles 401 responses with automatic token refresh and retry
- **Retry Logic**: Automatic retry for idempotent methods on network/server errors
- **Timeout Support**: Configurable request timeouts with AbortController
- **Error Normalization**: Structured error objects with status, code, message, and retry count
- **Telemetry**: Event emission for monitoring and debugging
- **Type Safety**: Full TypeScript support

## Installation & Initialization

The API client is initialized in `authStore.ts` and exported as a singleton:

```typescript
import { apiClient } from '@/lib/api-client';

// Initialize with configuration (done once in authStore)
apiClient.initialize({
  baseURL: '/api',
  getAccessToken: () => authState.accessToken,
  refreshToken: refreshAccessToken,
  onUnauthorized: handleUnauthorized,
  onTelemetry: (event) => {
    // Optional: send to analytics/monitoring service
    console.log('API Event:', event);
  },
});
```

## Basic Usage

### GET Request

```typescript
import { apiClient } from '@/lib/api-client';

// Simple GET request
const user = await apiClient.get<User>('/users/me');

// GET with query parameters (construct URL manually)
const scenes = await apiClient.get<Scene[]>('/scenes?lat=37.7749&lng=-122.4194');
```

### POST Request

```typescript
// POST with JSON body
const newScene = await apiClient.post<Scene>('/scenes', {
  name: 'My Scene',
  coarse_geohash: 'dr5r',
  allow_precise: false,
});

// POST without body
await apiClient.post('/scenes/123/join');
```

### PUT Request

```typescript
// Update a resource
const updatedScene = await apiClient.put<Scene>('/scenes/123', {
  name: 'Updated Name',
  description: 'New description',
});
```

### PATCH Request

```typescript
// Partial update
const scene = await apiClient.patch<Scene>('/scenes/123', {
  description: 'Updated description only',
});
```

### DELETE Request

```typescript
// Delete a resource
await apiClient.delete('/scenes/123');

// DELETE returns empty object for 204 No Content
const result = await apiClient.delete('/scenes/123');
// result === {}
```

## Advanced Configuration

### Skip Authentication

For public endpoints that don't require authentication:

```typescript
// Login endpoint doesn't need auth token
const response = await apiClient.post<LoginResponse>(
  '/auth/login',
  { username, password },
  { skipAuth: true }
);
```

### Skip Token Refresh

Prevent automatic retry on 401 (useful for logout or when you want to handle 401 yourself):

```typescript
try {
  await apiClient.get('/protected-resource', { skipRetry: true });
} catch (error) {
  if (error instanceof ApiClientError && error.status === 401) {
    // Handle 401 without automatic refresh attempt
  }
}
```

### Custom Timeout

Set a custom timeout for slow operations (default is 10 seconds):

```typescript
// Long-running operation with 30 second timeout
const report = await apiClient.get<Report>('/reports/generate', {
  timeout: 30000, // 30 seconds
});

// Quick health check with 2 second timeout
const health = await apiClient.get('/health', {
  timeout: 2000,
});
```

### Skip Automatic Retry

Disable automatic retry for specific requests:

```typescript
// Don't retry even if it's an idempotent method
const data = await apiClient.get('/data', {
  skipAutoRetry: true,
});
```

## Retry Behavior

### Idempotent Methods (Automatic Retry)

The following HTTP methods are automatically retried on failure:
- **GET**: Safe to retry, no side effects
- **PUT**: Idempotent by design
- **DELETE**: Idempotent by design
- **HEAD**: Safe to retry
- **OPTIONS**: Safe to retry

### Non-Idempotent Methods (No Retry)

These methods are **never** automatically retried:
- **POST**: May create duplicate resources
- **PATCH**: Partial updates may not be idempotent

### Retry Conditions

Requests are retried when:
1. Network error occurs (status 0)
2. Server error (5xx status codes)
3. Request timeout occurs

Requests are **not** retried for:
- Client errors (4xx except 401)
- Successful responses (2xx, 3xx)
- Non-idempotent methods (POST)
- When `skipAutoRetry: true` is set

### Retry Configuration

- **Max Attempts**: 3 (including the initial request)
- **Initial Delay**: 100ms
- **Max Delay**: 2000ms (2 seconds)
- **Backoff Strategy**: Exponential with 30% random jitter
- **Jitter Purpose**: Prevents thundering herd problem

Example retry sequence:
```
Attempt 1: Immediate
Attempt 2: ~100ms delay (with jitter: 70-130ms)
Attempt 3: ~200ms delay (with jitter: 140-260ms)
```

## Error Handling

All errors are thrown as `ApiClientError` instances with structured information:

```typescript
import { apiClient, ApiClientError } from '@/lib/api-client';

try {
  const scene = await apiClient.get<Scene>('/scenes/123');
} catch (error) {
  if (error instanceof ApiClientError) {
    console.error('API Error:', {
      status: error.status,      // HTTP status code (e.g., 404)
      code: error.code,          // Error code from API (e.g., 'not_found')
      message: error.message,    // Human-readable message
      retryCount: error.retryCount, // Number of retry attempts made
    });

    // Handle specific errors
    if (error.status === 404) {
      // Resource not found
    } else if (error.status === 403) {
      // Forbidden
    } else if (error.code === 'timeout') {
      // Request timed out
    } else if (error.code === 'network_error') {
      // Network failure
    }
  }
}
```

### Common Error Codes

| Code | Status | Description |
|------|--------|-------------|
| `unauthorized` | 401 | Authentication required or token expired |
| `forbidden` | 403 | Insufficient permissions |
| `not_found` | 404 | Resource not found |
| `validation_error` | 400 | Invalid request data |
| `timeout` | 0 | Request timeout |
| `network_error` | 0 | Network failure |
| `internal_error` | 500 | Server error |
| `service_unavailable` | 503 | Service temporarily unavailable |

## Telemetry Events

The API client emits telemetry events for monitoring and debugging:

### Request Event

Emitted when a request starts:

```typescript
{
  type: 'api_request',
  method: 'GET',
  endpoint: '/scenes',
  timestamp: 1701234567890
}
```

### Response Event

Emitted when a request completes (success or failure):

```typescript
{
  type: 'api_response',
  method: 'GET',
  endpoint: '/scenes',
  status: 200,
  duration: 342,        // milliseconds
  retryCount: 0,        // number of retry attempts
  timestamp: 1701234568232
}
```

### Telemetry Callback

Configure the telemetry callback during initialization:

```typescript
apiClient.initialize({
  baseURL: '/api',
  getAccessToken: () => authState.accessToken,
  refreshToken: refreshAccessToken,
  onUnauthorized: handleUnauthorized,
  onTelemetry: (event) => {
    if (event.type === 'api_response') {
      // Track response metrics
      metrics.apiLatency.observe(event.duration);
      metrics.apiStatus.inc({ status: event.status, method: event.method });
      
      if (event.retryCount > 0) {
        metrics.apiRetries.inc({ endpoint: event.endpoint });
      }
    }
  },
});
```

**Note**: Telemetry callbacks should never throw errors, as they are wrapped in try-catch to prevent breaking requests.

## Type Definitions

### RequestConfig

Extended `RequestInit` with additional options:

```typescript
interface RequestConfig extends RequestInit {
  skipAuth?: boolean;       // Skip adding Authorization header
  skipRetry?: boolean;      // Skip retry on 401
  timeout?: number;         // Request timeout in milliseconds (default: 10000)
  skipAutoRetry?: boolean;  // Skip automatic retry on network/5xx errors
}
```

### ApiError

Structured error information:

```typescript
interface ApiError {
  status: number;      // HTTP status code
  code: string;        // Error code from API
  message: string;     // Human-readable message
  retryCount: number;  // Number of retry attempts made
}
```

### TelemetryEvent

```typescript
type ApiRequestEvent = {
  type: 'api_request';
  method: string;
  endpoint: string;
  timestamp: number;
};

type ApiResponseEvent = {
  type: 'api_response';
  method: string;
  endpoint: string;
  status: number;
  duration: number;
  retryCount: number;
  timestamp: number;
};

type TelemetryEvent = ApiRequestEvent | ApiResponseEvent;
```

## Best Practices

### 1. Use Type Parameters

Always specify the response type for better type safety:

```typescript
// Good
const scene = await apiClient.get<Scene>('/scenes/123');

// Avoid
const scene = await apiClient.get('/scenes/123'); // Type is unknown
```

### 2. Handle Errors Appropriately

Don't swallow errors - handle them or let them propagate:

```typescript
// Good
try {
  const scene = await apiClient.get<Scene>('/scenes/123');
  setScene(scene);
} catch (error) {
  if (error instanceof ApiClientError && error.status === 404) {
    setError('Scene not found');
  } else {
    throw error; // Propagate unexpected errors
  }
}

// Avoid silent failures
try {
  await apiClient.post('/scenes', data);
} catch {
  // Don't ignore errors without handling them
}
```

### 3. Use Appropriate Timeouts

Set timeouts based on expected operation duration:

```typescript
// Quick operations: 2-5 seconds
const health = await apiClient.get('/health', { timeout: 2000 });

// Normal operations: 10 seconds (default)
const scenes = await apiClient.get<Scene[]>('/scenes');

// Long operations: 30+ seconds
const export = await apiClient.get('/export/full', { timeout: 30000 });
```

### 4. Don't Manually Add Authorization Headers

The client handles this automatically:

```typescript
// Good
const user = await apiClient.get<User>('/users/me');

// Avoid
const user = await apiClient.get<User>('/users/me', {
  headers: { Authorization: `Bearer ${token}` }, // Unnecessary
});
```

### 5. Use skipAuth for Public Endpoints

Explicitly mark public endpoints:

```typescript
// Good
const response = await apiClient.post('/auth/login', credentials, {
  skipAuth: true,
});

// Avoid relying on client checking for null token
const response = await apiClient.post('/auth/login', credentials);
```

## Security Considerations

### Token Storage

- **Access tokens** are stored in memory only (via `authStore`)
- **Refresh tokens** are stored in httpOnly secure cookies
- **Never** log or send tokens to third-party services

### HTTPS Only

The API client should only be used over HTTPS in production. HTTP connections leak tokens in transit.

### Token Leakage Prevention

- Tokens are **only** sent in `Authorization` headers, never in URLs or query parameters
- Tokens are **never** included in telemetry events or error messages

### CORS

Requests include credentials by default (`credentials: 'include'`) to send httpOnly cookies. Ensure backend CORS configuration is strict (no wildcard origins).

## Troubleshooting

### Request Keeps Timing Out

1. Check network connectivity
2. Increase timeout: `{ timeout: 30000 }`
3. Check backend logs for slow queries
4. Verify backend is running and accessible

### Automatic Retries Not Working

1. Verify method is idempotent (GET, PUT, DELETE)
2. Check if `skipAutoRetry: true` is set
3. Verify error is 5xx or network error (0)

### Token Refresh Failing

1. Check refresh token cookie is being sent
2. Verify backend `/auth/refresh` endpoint is working
3. Check browser console for CORS errors
4. Ensure refresh token hasn't expired

### Getting 401 Errors

1. Verify user is logged in (`authStore.isAuthenticated`)
2. Check access token is present (`authStore.accessToken`)
3. Verify token hasn't expired (check backend logs)
4. Check if endpoint requires authentication

## Pagination

The API uses cursor-based pagination for all search and feed endpoints, ensuring deterministic ordering, no duplicates, and complete coverage of results across pages.

### Pagination Guarantees

All paginated endpoints guarantee:

1. **No Duplicates**: Each item appears exactly once across all pages
2. **Complete Coverage**: All matching items are returned when paginating to exhaustion
3. **Deterministic Ordering**: Results are ordered consistently using composite scores with ID-based tie-breaking
4. **Stable Cursors**: Cursors maintain full precision and round-trip without data loss
5. **Insertion Order Independence**: Results are ordered by score and ID, not insertion time

### Cursor Formats

Different endpoints use different cursor encoding formats:

- **Scene Search**: Base64-encoded JSON `{"score": 0.85, "id": "scene-123"}`
- **Event Search**: Pipe-delimited `score|eventID` (e.g., `0.8532|event-uuid`)
- **Post Search**: Colon-delimited `score:id` (e.g., `0.750000:post-uuid`)
- **Feeds**: Timestamp-based `{unix_nano}:{post_id}`

### Ordering Rules

Results are ordered using the following rules for stable pagination:

1. **Primary**: Composite score (DESC) - weighted combination of relevance factors
2. **Secondary**: ID (ASC) - guarantees deterministic ordering when scores are identical

This two-tier ordering ensures:
- Items with higher scores appear first
- Items with identical scores are ordered alphabetically by ID
- Order remains consistent across multiple requests

### Example: Scene Search with Pagination

```typescript
// First page
const page1 = await apiClient.get<SearchScenesResponse>(
  '/search/scenes?bbox=-74.1,40.6,-73.9,40.8&q=music&limit=20'
);

// Second page using cursor from first page
const page2 = await apiClient.get<SearchScenesResponse>(
  `/search/scenes?bbox=-74.1,40.6,-73.9,40.8&q=music&limit=20&cursor=${page1.next_cursor}`
);

// Continue until next_cursor is empty
let cursor = page1.next_cursor;
const allResults = [...page1.scenes];

while (cursor) {
  const page = await apiClient.get<SearchScenesResponse>(
    `/search/scenes?bbox=-74.1,40.6,-73.9,40.8&q=music&limit=20&cursor=${cursor}`
  );
  allResults.push(...page.scenes);
  cursor = page.next_cursor;
}
```

### Edge Cases

- **Empty Results**: Returns empty array with `next_cursor: ""`
- **Last Page**: Returns results with `next_cursor: ""` (no more pages)
- **Single Item**: Returns one item with `next_cursor: ""`
- **Invalid Cursor**: Returns 400 Bad Request with error code `invalid_cursor`

### Score Ties

When multiple items have identical composite scores (e.g., same location, same text match), they are ordered deterministically by ID (ascending). This ensures:

- Consistent ordering across requests
- No items are skipped or duplicated
- Reproducible pagination sequences

### Performance

- **Cursor Decode**: O(1) complexity, typically <1ms per operation
- **No Offset-Based Pagination**: Cursors provide efficient deep pagination without performance degradation
- **Recommended Page Size**: 20-50 items for optimal balance between requests and payload size

### Testing

Comprehensive pagination tests verify:
- No duplicates across pages (`TestScenePagination_NoDuplicates`, `TestEventPagination_NoDuplicates`, `TestPostPagination_NoDuplicates`)
- Ordering consistency (`TestScenePagination_OrderingConsistency`, `TestPostPagination_OrderingConsistency`)
- Deterministic tie-breaking (`TestScenePagination_ScoreTies`, `TestEventPagination_ScoreTies`, `TestPostPagination_ScoreTies`)
- Cursor round-trip stability (`TestSceneCursor_RoundTrip`, `TestPostCursor_RoundTrip`)
- Edge cases (empty results, last page, single item)

See `internal/scene/pagination_test.go` and `internal/post/pagination_test.go` for full test coverage.

## Payment Endpoints

The payment system uses Stripe Connect for direct scene monetization with transparent platform fees.

### POST /payments/onboard

Creates a Stripe Connect Express onboarding link for scene owners to enable payment processing.

**Authentication**: Required (JWT)

**Request Body**:
```json
{
  "scene_id": "uuid"
}
```

**Response** (200 OK):
```json
{
  "url": "https://connect.stripe.com/setup/s/...",
  "expires_at": "2026-01-27T15:30:00Z"
}
```

**Error Codes**:
- `unauthorized` (401) - Authentication required
- `bad_request` (400) - Missing or invalid scene_id
- `not_found` (404) - Scene not found
- `forbidden` (403) - User is not the scene owner
- `already_onboarded` (400) - Scene already has connected account
- `internal_error` (500) - Stripe API error

See [Payment Handlers](./PAYMENT_HANDLERS.md) for detailed documentation.

### POST /payments/checkout

Creates a Stripe Checkout Session for event tickets or merchandise with platform fee.

**Authentication**: Required (JWT)

**Headers**:
- `Idempotency-Key` (required) - Unique key to prevent duplicate charges (max 64 chars, recommend UUIDv4)

**Request Body**:
```json
{
  "scene_id": "uuid",
  "event_id": "uuid (optional)",
  "items": [
    {
      "price_id": "price_xxx",
      "quantity": 2
    }
  ],
  "success_url": "https://example.com/success",
  "cancel_url": "https://example.com/cancel"
}
```

**Response** (200 OK):
```json
{
  "session_url": "https://checkout.stripe.com/pay/cs_test_...",
  "session_id": "cs_test_..."
}
```

**Error Codes**:
- `unauthorized` (401) - Authentication required
- `missing_idempotency_key` (400) - Idempotency-Key header required
- `idempotency_key_too_long` (400) - Key exceeds 64 characters
- `bad_request` (400) - Missing required fields or invalid data
- `not_onboarded` (400) - Scene must be onboarded before accepting payments
- `not_found` (404) - Scene not found
- `internal_error` (500) - Stripe API error

**Idempotency**: Duplicate requests with the same `Idempotency-Key` return the cached response without creating a new checkout session. Keys expire after 24 hours.

### GET /payments/status

Retrieves the current status of a payment by checkout session ID.

**Authentication**: Required (JWT)

**Query Parameters**:
- `sessionId` (required) - Stripe Checkout Session ID

**Response** (200 OK):
```json
{
  "status": "succeeded",
  "amount_cents": 10000,
  "fee_cents": 500,
  "currency": "usd",
  "updated_at": "2026-01-27T15:30:00Z"
}
```

**Status Values**:
- `pending` - Payment not yet completed
- `succeeded` - Payment successfully processed
- `failed` - Payment failed
- `canceled` - Payment canceled by user
- `refunded` - Payment refunded

**Error Codes**:
- `unauthorized` (401) - Authentication required
- `bad_request` (400) - Missing sessionId parameter
- `payment_not_found` (404) - Payment record not found
- `forbidden` (403) - Only payment creator or scene owner can access status
- `internal_error` (500) - Database error

**Authorization**: Only the user who created the payment or the scene owner can retrieve payment status.

**Caching**: Terminal statuses (succeeded, failed, canceled, refunded) include `Cache-Control: private, max-age=5` to reduce polling overhead.

### POST /internal/stripe (Webhook)

Processes Stripe webhook events for payment status updates. This endpoint is for Stripe's internal use only.

**Authentication**: Stripe signature verification (via `Stripe-Signature` header)

**Supported Events**:
- `checkout.session.completed` - Checkout session completed
- `payment_intent.succeeded` - Payment successfully processed
- `payment_intent.payment_failed` - Payment failed
- `account.updated` - Connect account status changed

**Response**: Always returns 200 OK to acknowledge receipt

See [Stripe Webhooks](./STRIPE_WEBHOOKS.md) for detailed documentation.

### Payment Status Transitions

The payment system enforces a strict state machine to prevent invalid transitions:

```
pending -> succeeded, failed, canceled
succeeded -> refunded
```

All other transitions are rejected with `invalid_status_transition` error.

See [Payment Status Transitions](./PAYMENT_STATUS_TRANSITIONS.md) for detailed state machine documentation.

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md) - System architecture and design decisions
- [Privacy Guidelines](./PRIVACY.md) - Privacy-first design principles
- [Auth Flow](./ARCHITECTURE.md#authentication) - Authentication and authorization flow
- [Payment Handlers](./PAYMENT_HANDLERS.md) - Detailed payment endpoint documentation
- [Stripe Webhooks](./STRIPE_WEBHOOKS.md) - Webhook processing and security
- [Idempotency](./idempotency.md) - Idempotency key middleware documentation
