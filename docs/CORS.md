# CORS Security Configuration

This document describes the CORS (Cross-Origin Resource Sharing) security implementation in Subcults.

## Overview

CORS is implemented as HTTP middleware that enforces strict origin validation. It protects the API from unauthorized cross-origin requests while allowing legitimate frontend applications to access the API.

## Security Features

- **No Wildcard Origins**: Only explicitly listed origins are allowed. Wildcard (`*`) origins are not supported for security.
- **Strict Origin Validation**: Every cross-origin request is validated against the allowlist.
- **Preflight Support**: OPTIONS preflight requests are handled automatically.
- **Configurable per Environment**: Different origins can be configured for development and production.
- **Same-Origin Bypass**: Requests without an `Origin` header (same-origin requests) are allowed through.

## Configuration

CORS is configured via environment variables. If no origins are configured, CORS is disabled. The server will not add CORS headers to responses, which means browsers will only allow same-origin requests.

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `CORS_ALLOWED_ORIGINS` | Comma-separated list of allowed origins (no wildcards) | `""` (disabled) | `http://localhost:3000,https://subcults.com` |
| `CORS_ALLOWED_METHODS` | Comma-separated list of allowed HTTP methods | `GET,POST,PUT,PATCH,DELETE,OPTIONS` | `GET,POST,OPTIONS` |
| `CORS_ALLOWED_HEADERS` | Comma-separated list of allowed request headers | `Content-Type,Authorization,X-Request-ID` | `Content-Type,Authorization` |
| `CORS_ALLOW_CREDENTIALS` | Allow credentials (cookies, auth headers) | `true` | `true` or `false` |
| `CORS_MAX_AGE` | Preflight cache duration in seconds | `3600` | `3600` |

### Development Configuration

For local development with frontend running on different ports:

```bash
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Request-ID
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600
```

### Production Configuration

For production deployment:

```bash
CORS_ALLOWED_ORIGINS=https://subcults.com,https://app.subcults.com
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Request-ID
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600
```

## How It Works

### Request Flow

1. **Same-Origin Requests**: Requests without an `Origin` header pass through without CORS checks.

2. **Cross-Origin Requests**:
   - The middleware checks if the `Origin` header matches any allowed origin.
   - If allowed, CORS headers are set and the request proceeds.
   - If not allowed, the request is rejected with `403 Forbidden`.

3. **Preflight Requests** (OPTIONS):
   - Browsers send OPTIONS requests before actual cross-origin requests.
   - The middleware responds with appropriate CORS headers and `204 No Content`.
   - The actual request is then allowed to proceed if the preflight succeeds.

### Middleware Position

CORS middleware is positioned early in the middleware chain:

```
Request → Tracing → Profiling → CORS → Canary → Rate Limiting → Metrics → RequestID → Logging → Handler
```

Note: Canary routing is only present if enabled via configuration.

This ensures:
- Unauthorized origins are rejected before consuming resources.
- CORS headers are present even when later middleware rejects the request.
- Request IDs and tracing work for all requests, including rejected ones.

## Response Headers

### Allowed Origin

For allowed cross-origin requests:

```http
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Credentials: true
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID
```

### Preflight Response

For OPTIONS preflight requests:

```http
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 3600
```

### Unauthorized Origin

For requests from unauthorized origins:

```http
HTTP/1.1 403 Forbidden
(No CORS headers)
```

## Testing

### Unit Tests

Run the CORS middleware tests:

```bash
go test -v ./internal/middleware/cors_test.go ./internal/middleware/cors.go
```

### Integration Tests

Run the CORS integration tests with other middleware:

```bash
go test -v ./internal/middleware/cors_integration_test.go ./internal/middleware/cors.go ./internal/middleware/requestid.go ./internal/middleware/logging.go
```

### Manual Testing

Use the provided script to test CORS with a running server:

```bash
# Start the API server with CORS configured
export CORS_ALLOWED_ORIGINS=http://localhost:3000
./bin/api

# In another terminal, run the CORS test script
./scripts/test-cors.sh
```

Or test manually with `curl`:

```bash
# Test preflight request
curl -i -X OPTIONS http://localhost:8080/health/live \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET"

# Test actual request
curl -i -X GET http://localhost:8080/health/live \
  -H "Origin: http://localhost:3000"

# Test unauthorized origin
curl -i -X GET http://localhost:8080/health/live \
  -H "Origin: http://malicious.com"
```

## Common Issues

### CORS Headers Not Appearing

**Symptom**: No `Access-Control-Allow-Origin` header in response.

**Causes**:
1. CORS is disabled (no `CORS_ALLOWED_ORIGINS` configured)
2. Request is same-origin (no `Origin` header in request)
3. Origin is not in the allowed list

**Solution**:
- Check that `CORS_ALLOWED_ORIGINS` is set
- Verify the frontend is sending the `Origin` header
- Ensure the origin is in the allowlist (exact match required)

### Preflight Requests Failing

**Symptom**: Browser shows preflight error, actual request never sent.

**Causes**:
1. Origin not allowed
2. Method not in `CORS_ALLOWED_METHODS`
3. Header not in `CORS_ALLOWED_HEADERS`

**Solution**:
- Add the origin to `CORS_ALLOWED_ORIGINS`
- Add required methods to `CORS_ALLOWED_METHODS`
- Add required headers to `CORS_ALLOWED_HEADERS`

### Credentials Not Sent

**Symptom**: Cookies or Authorization header not sent by browser.

**Cause**: Frontend not configured to send credentials, or `CORS_ALLOW_CREDENTIALS` is false.

**Solution**:
1. Set `CORS_ALLOW_CREDENTIALS=true` on backend
2. Configure frontend to send credentials:
   ```javascript
   // fetch API
   fetch('https://api.subcults.com/health', {
     credentials: 'include'
   })

   // axios
   axios.get('https://api.subcults.com/health', {
     withCredentials: true
   })
   ```

## Security Considerations

### Do Not Use Wildcards

Never use `*` as an allowed origin. This would allow any website to make requests to your API, potentially exposing user data.

❌ **Don't do this**:
```bash
CORS_ALLOWED_ORIGINS=*  # INSECURE!
```

✅ **Do this**:
```bash
CORS_ALLOWED_ORIGINS=https://subcults.com,https://app.subcults.com
```

### Exact Origin Matching

Origins must match exactly, including protocol and port:

- `http://localhost:3000` ≠ `http://localhost:3001`
- `http://localhost:3000` ≠ `https://localhost:3000`
- `http://example.com` ≠ `http://www.example.com`

### Credentials and Origins

When `CORS_ALLOW_CREDENTIALS=true`, you cannot use wildcard origins. This is a browser security requirement to prevent credential leaking.

### Production Checklist

Before deploying to production:

- [ ] Only list production origins in `CORS_ALLOWED_ORIGINS`
- [ ] Remove development origins (localhost, etc.)
- [ ] Verify origins use HTTPS (except localhost)
- [ ] Test preflight requests work correctly
- [ ] Verify credentials are sent/received if needed
- [ ] Check logs for rejected CORS requests

## Related Files

- Middleware implementation: `internal/middleware/cors.go`
- Unit tests: `internal/middleware/cors_test.go`
- Integration tests: `internal/middleware/cors_integration_test.go`
- Configuration: `internal/config/config.go`
- Example configuration: `configs/dev.env.example`
- Test script: `scripts/test-cors.sh`

## References

- [MDN: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [W3C CORS Specification](https://www.w3.org/TR/cors/)
- [OWASP: CORS Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/CORS_Cheat_Sheet.html)
