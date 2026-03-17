# Input Validation Package

Centralized input validation and sanitization utilities for the Subcults API. Provides protection against SQL injection, XSS, SSRF, and other common web vulnerabilities.

## Overview

The `validate` package provides a comprehensive set of validators for all user input:

- **String validation**: Length constraints, pattern matching, SQL keyword detection
- **HTML sanitization**: XSS prevention through entity escaping
- **Email validation**: RFC 5321 compliant format checking
- **URL validation**: SSRF protection with private IP blocking and domain allowlisting
- **File validation**: MIME type and size constraints for uploads

## Usage

### String Validation

```go
import "github.com/onnwee/subcults/internal/validate"

// Validate and sanitize a scene name
name, err := validate.SceneName("My Cool Scene")
// Returns: "My Cool Scene", nil

// Legitimate venue names are allowed (SQL keyword checking disabled)
name, err := validate.SceneName("Drop Zone Music Hall")
// Returns: "Drop Zone Music Hall", nil

// Custom string validation with SQL keyword checking
validated, err := validate.String("user input", validate.StringConstraints{
    MinLength: 5,
    MaxLength: 100,
    CheckSQLKeywords: true, // Enable for non-user-facing fields
    TrimSpace: true,
})
```

### Email Validation

```go
email, err := validate.Email("user@example.com")
// Returns: "user@example.com", nil (normalized to lowercase)

email, err := validate.Email("User@Example.COM  ")
// Returns: "user@example.com", nil (trimmed and lowercased)
```

### URL Validation

```go
// HTTPS only, blocks private IPs (default)
url, err := validate.AttachmentURL("https://cdn.example.com/image.jpg")
// Returns: valid URL, nil

url, err := validate.AttachmentURL("http://10.0.0.1/internal")
// Returns: "", error (private IP blocked)

// Custom URL constraints
url, err := validate.URL("https://api.example.com", validate.URLConstraints{
    AllowedSchemes: []string{"https"},
    AllowedDomains: []string{"example.com", "api.example.com"},
    BlockPrivate:   true,
    MaxLength:      2048,
})
```

### File Validation

```go
// Validate image upload
mimeType, err := validate.ImageFile("image/jpeg", 5*1024*1024) // 5MB
// Returns: "image/jpeg", nil

mimeType, err := validate.ImageFile("image/jpeg", 20*1024*1024) // 20MB
// Returns: "", error (exceeds 10MB limit)

// Custom file validation
mimeType, err := validate.File("audio/mpeg", sizeBytes, validate.FileConstraints{
    AllowedTypes: []string{"audio/mpeg", "audio/wav"},
    MaxSizeBytes: 50 * 1024 * 1024, // 50MB
})
```

### HTML Sanitization

```go
// Sanitize user-generated content
sanitized := validate.SanitizeHTML("<script>alert('xss')</script>")
// Returns: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"

// Combined validation and sanitization
content, err := validate.PostContent("User post with <b>HTML</b>")
// Returns: "User post with &lt;b&gt;HTML&lt;/b&gt;", nil
```

## Validation Rules

### Scene Names
- 1-100 characters
- Alphanumeric, spaces, dash, underscore, period only
- HTML sanitized
- **Note**: SQL keyword checking disabled to avoid false positives with legitimate venue names

### Event Titles
- 1-200 characters
- HTML sanitized
- **Note**: SQL keyword checking disabled to avoid false positives with legitimate event names

### Post Content
- 1-5000 characters
- HTML sanitized
- SQL keywords allowed (more freedom for user content)

### Email
- RFC 5321 compliant
- Max 254 characters total
- Max 64 characters for local part
- Max 255 characters for domain
- Normalized to lowercase

### URLs
- Configurable scheme allowlist (default: HTTPS only)
- Optional domain allowlist
- SSRF protection: blocks localhost, private IPs, link-local addresses
- Max 2048 characters (configurable)

### Files
- **Images**: JPEG, PNG, GIF, WebP (max 10MB)
- **Audio**: MP3, WAV, OGG (max 50MB)
- **Video**: MP4, WebM (max 500MB)
- Custom constraints supported

## Security Features

### SQL Injection Prevention
- Keyword detection for common SQL commands (with word boundary detection)
- **Disabled for user-facing fields** (scene names, event titles) to avoid false positives with legitimate names like "Drop Zone" or "The Executive Lounge"
- Available for other use cases via `CheckSQLKeywords: true` in `StringConstraints`
- **Primary defense**: Parameterized queries (already implemented in repository layer)
- Keywords checked: SELECT, INSERT, UPDATE, DELETE, DROP, CREATE, ALTER, TRUNCATE, EXEC, UNION
- Comment patterns blocked: --, /*, */, ;--
- Stored procedure prefixes blocked: xp_, sp_

### XSS Prevention
- HTML entity escaping for all user-generated text
- Converts `<`, `>`, `&`, `"`, `'` to safe HTML entities
- Applied automatically by high-level validators

### SSRF Prevention
- **Performance optimization**: Checks if hostname is an IP before DNS lookup
- **Timeout protection**: 2-second timeout on DNS resolution
- Blocks localhost and localhost.localdomain
- Blocks 0.0.0.0 (unspecified address)
- Blocks private IPv4 ranges: 10.x.x.x, 172.16-31.x.x, 192.168.x.x
- Blocks link-local addresses: 169.254.x.x (including cloud metadata service IPs)
- Blocks loopback addresses (127.0.0.0/8, ::1)
- Blocks private IPv6 ranges (fc00::/7)
- Optional domain allowlist for strict control
- **Note**: DNS rebinding protection - consumers should call validation immediately before making HTTP requests

### File Upload Security
- MIME type validation
- File size limits per type
- Prevents executable uploads
- Prevents oversized files

## Integration Examples

### HTTP Handler Integration

```go
func (h *SceneHandlers) CreateScene(w http.ResponseWriter, r *http.Request) {
    var req CreateSceneRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Validate and sanitize scene name
    name, err := validate.SceneName(req.Name)
    if err != nil {
        http.Error(w, fmt.Sprintf("Invalid scene name: %v", err), http.StatusBadRequest)
        return
    }

    // Validate description
    desc, err := validate.Description(req.Description)
    if err != nil {
        http.Error(w, fmt.Sprintf("Invalid description: %v", err), http.StatusBadRequest)
        return
    }

    // Create scene with validated data
    scene := &scene.Scene{
        Name:        name,
        Description: desc,
        // ... other fields
    }
    // ...
}
```

## Testing

Run tests:
```bash
go test ./internal/validate/...
```

Run tests with coverage:
```bash
go test -cover ./internal/validate/...
```

## Error Handling

All validators return descriptive errors:

```go
name, err := validate.SceneName(userInput)
if err != nil {
    switch {
    case errors.Is(err, validate.ErrStringTooShort):
        // Handle too short
    case errors.Is(err, validate.ErrStringTooLong):
        // Handle too long
    case errors.Is(err, validate.ErrSQLKeyword):
        // Handle SQL injection attempt
    default:
        // Handle other errors
    }
}
```

## Best Practices

1. **Always validate at the API boundary**: Validate all user input as soon as it enters your system
2. **Use parameterized queries**: The SQL keyword check is defense-in-depth, not a replacement for parameterized queries
3. **Sanitize for display**: Always use `SanitizeHTML` before displaying user content
4. **Apply SSRF protection**: Use `DefaultURLConstraints` or stricter for all external URLs
5. **Validate early, fail fast**: Return clear error messages to help users provide valid input
6. **Log security events**: Consider logging SQL keyword detections and SSRF attempts for monitoring

## Future Enhancements

- [ ] Struct tag support for declarative validation (e.g., `validate:"required,min=3,max=100"`)
- [ ] Rate limiting integration for validation failures
- [ ] Audit logging for security-related validation failures
- [ ] Additional validators (phone numbers, credit cards, etc.)
- [ ] Internationalization support (error messages in multiple languages)
