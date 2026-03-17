# API Package

The `internal/api` package provides HTTP API utilities for the Subcults API server, including standardized error handling.

## Error Handling

### Overview

All API errors return a standardized JSON format:

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable error message"
  }
}
```

This ensures consistent error handling across all API endpoints and simplifies client-side error processing.

### Usage

#### Basic Error Response

```go
package main

import (
    "net/http"
    "github.com/onnwee/subcults/internal/api"
    "github.com/onnwee/subcults/internal/middleware"
)

func handler(w http.ResponseWriter, r *http.Request) {
    // Set error code in context for logging middleware
    ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeNotFound)
    
    // Write error response
    api.WriteError(w, ctx, http.StatusNotFound, api.ErrCodeNotFound, "Scene not found")
}
```

#### Common Error Codes

The package provides the following predefined error codes:

| Constant | Code | HTTP Status | Description |
|----------|------|-------------|-------------|
| `ErrCodeValidation` | `validation_error` | 400 | Input validation failure |
| `ErrCodeBadRequest` | `bad_request` | 400 | Malformed request |
| `ErrCodeInvalidTimeRange` | `invalid_time_range` | 400 | Event start time not before end time |
| `ErrCodeAuthFailed` | `auth_failed` | 401 | Authentication failure |
| `ErrCodeForbidden` | `forbidden` | 403 | Request is forbidden |
| `ErrCodeNotFound` | `not_found` | 404 | Resource not found |
| `ErrCodeConflict` | `conflict` | 409 | Conflict with current state |
| `ErrCodeRateLimited` | `rate_limited` | 429 | Rate limit exceeded |
| `ErrCodeInternal` | `internal_error` | 500 | Internal server error |

#### Status Code Mapping

Use `StatusCodeMapping()` to get the recommended HTTP status code for a given error code:

```go
status := api.StatusCodeMapping(api.ErrCodeValidation)
// Returns http.StatusBadRequest (400)
```

### Integration with Logging Middleware

The error handling system integrates seamlessly with the logging middleware:

1. **Set error code in context** using `middleware.SetErrorCode()`
2. **Write error response** using `api.WriteError()`
3. **Logging middleware automatically captures** the error code for 4xx and 5xx responses

Example:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Error code is set in context
    ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeValidation)
    
    // WriteError sends the response
    api.WriteError(w, ctx, http.StatusBadRequest, api.ErrCodeValidation, "Invalid email format")
    
    // Logging middleware will automatically log:
    // - status: 400
    // - error_code: validation_error
    // - request_id, user_did (if present)
}
```

### Security Considerations

- **Never expose internal stack traces** in error messages
- **Avoid leaking sensitive information** in error details
- **Use generic messages for internal errors** (e.g., "Internal server error")
- **Provide specific messages for client errors** (e.g., "Invalid email format")

### Examples

#### Validation Error

```go
ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeValidation)
api.WriteError(w, ctx, http.StatusBadRequest, api.ErrCodeValidation, "Email field is required")
```

Response:
```json
{
  "error": {
    "code": "validation_error",
    "message": "Email field is required"
  }
}
```

#### Authentication Error

```go
ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeAuthFailed)
api.WriteError(w, ctx, http.StatusUnauthorized, api.ErrCodeAuthFailed, "Invalid or expired token")
```

Response:
```json
{
  "error": {
    "code": "auth_failed",
    "message": "Invalid or expired token"
  }
}
```

#### Not Found Error

```go
ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeNotFound)
api.WriteError(w, ctx, http.StatusNotFound, api.ErrCodeNotFound, "Scene not found")
```

Response:
```json
{
  "error": {
    "code": "not_found",
    "message": "Scene not found"
  }
}
```

#### Internal Error

```go
// Log detailed error internally
slog.Error("database query failed", "error", err, "query", query)

// Return generic error to client
ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeInternal)
api.WriteError(w, ctx, http.StatusInternalServerError, api.ErrCodeInternal, "Internal server error")
```

Response:
```json
{
  "error": {
    "code": "internal_error",
    "message": "Internal server error"
  }
}
```

## Testing

The package includes comprehensive unit tests covering:

- Basic error response formatting
- All error code constants
- Content-Type headers
- JSON structure validation
- Integration with logging middleware
- Request ID propagation
- Special characters in error messages
- Empty messages
- Full end-to-end integration tests

Run tests:

```bash
go test -v -race -cover ./internal/api/...
```

## Future Enhancements

- Error code validation in CI (grep check or lint rule)
- Additional error codes as needed
- Error response localization (i18n)
- Structured error details (e.g., field-level validation errors)
