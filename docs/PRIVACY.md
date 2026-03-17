# Privacy Technical Overview

Subcult implements a **privacy-first** architecture designed to protect underground music communities. This document describes the technical privacy measures currently implemented and planned.

> **Note:** This is a technical summary for contributors and community review—not legal terms of service.

## Location Handling

Location data is handled with explicit consent controls and default privacy protections.

### Consent Model

All location-aware entities (Scenes, Events) include an `allow_precise` consent flag:

- **Default:** `allow_precise = FALSE` — precise coordinates are never stored without explicit opt-in
- **Database Enforcement:** Schema constraints ensure `precise_point IS NULL` when `allow_precise = FALSE`
- **Application Enforcement:** The `EnforceLocationConsent()` method clears precise coordinates before any persistence operation

```sql
-- Database constraint (from migrations/000000_initial_schema.up.sql)
CONSTRAINT chk_precise_consent CHECK (
    allow_precise = TRUE OR precise_point IS NULL
)
```

### Coarse Geohash

When precise location consent is not granted, only coarse location is stored:

- **Precision:** 6-character geohash (~±0.61 km accuracy)
- **Purpose:** Enables regional discovery without pinpointing exact venues
- **Validation:** Only valid base32 geohash characters are accepted; invalid input is rejected

The `RoundGeohash()` function truncates geohashes to the configured precision, preventing accidental leakage of higher-precision data.

### Location Jitter (Planned)

Future enhancement: random offset applied to map display coordinates to prevent deanonymization through coordinate triangulation.

## Media Sanitization

### EXIF Stripping

All uploaded media has EXIF metadata stripped before storage to prevent location/device leakage:

- **GPS coordinates** embedded in photos
- **Device identifiers** (camera make, model, serial numbers)
- **Timestamps** (original capture time, modification time)
- **Camera metadata** (exposure, ISO, aperture, software)

**Implementation**: The `internal/image` package uses [bimg](https://github.com/h2non/bimg) (libvips binding) to:
1. Strip all EXIF metadata with `StripMetadata: true`
2. Re-encode images to JPEG, WebP, or PNG formats
3. Apply EXIF orientation correction before stripping (ensures correct display)
4. Maintain image quality with configurable settings (default: 85%)

**Usage**:
```go
import "github.com/onnwee/subcults/internal/image"

// Process with defaults (JPEG, quality 85, strip metadata)
sanitizedBytes, err := image.Process(fileReader)

// Verify EXIF was removed
noEXIF, err := image.VerifyNoEXIF(sanitizedBytes)
```

**Status**: ✅ Implemented (service layer). Integration with media upload API endpoints is in progress.

See [`internal/image/README.md`](../internal/image/README.md) for detailed documentation and configuration options.

### Storage Security

Media assets are stored in Cloudflare R2 with:

- No public listing of buckets
- Signed URLs for time-limited access (planned)
- Content-type validation to prevent MIME-type attacks (planned)

## Access Logging

Structured request logging captures security-relevant events without excessive personal data.

### Logged Fields

| Field | Description |
|-------|-------------|
| `request_id` | UUID for request correlation (from `X-Request-ID` header or auto-generated) |
| `method` | HTTP method |
| `path` | Request path (no query strings logged) |
| `status` | HTTP response status code |
| `latency_ms` | Request duration in milliseconds |
| `size` | Response body size |
| `user_did` | Authenticated user's DID (if present) |
| `error_code` | Application error code (for 4xx/5xx responses) |

### What Is NOT Logged

- Request bodies or form data
- Full URLs with query parameters
- IP addresses (except for rate limiting decisions)
- Authentication credentials

### Log Levels

- **5xx errors:** `ERROR` level
- **4xx errors:** `WARN` level
- **Success:** `INFO` level

### Privacy Enforcement Logging

Scene visibility enforcement events are logged at **DEBUG level only** to prevent information leakage:

- Access denials for members-only scenes
- Access denials for hidden scenes
- Visibility mode of protected scenes

Production logs (INFO and above) do not reveal:
- Whether a forbidden scene exists
- The visibility mode of inaccessible scenes
- Who attempted to access protected scenes

This prevents attackers from discovering scenes through log analysis.

## Scene Visibility Controls

Scenes support three visibility modes that control access and discoverability:

### Public Scenes

- Accessible to all users (authenticated and unauthenticated)
- Appear in search results and map views
- Use case: Open community scenes, public events

### Members-Only Scenes

- Accessible only to the scene owner and active members
- Appear in search results only for authorized users
- Membership status must be `"active"` (pending/rejected members denied)
- Use case: Curated communities, invite-only scenes, trust-based groups

### Hidden Scenes

- Accessible only to the scene owner
- **Exempt from search results** (even for members)
- Accessible via direct URL only if user is owner
- Use case: Personal archives, draft scenes, private collections

### User Enumeration Prevention

All unauthorized access attempts return the same `404 Not Found` error as non-existent scenes. This prevents:

- Discovering which scenes exist by trying different IDs
- Determining scene visibility modes through error messages
- Building a database of scenes through brute force

**Example**: Attempting to access a members-only scene as a non-member returns the same error as requesting a non-existent scene.

See [Scene Visibility Documentation](./api/SCENE_VISIBILITY.md) for complete API details.

## User Controls

### Telemetry & Diagnostics

Users have granular control over diagnostic data collection:

#### Error Logging (Always Active)

Client-side errors are automatically logged to help improve application stability:

- **Data Collected:** Error messages, stack traces, component traces, URL, user agent, session ID
- **PII Redaction:** All sensitive data is automatically removed before transmission:
  - JWT tokens (eyJ... format)
  - Email addresses
  - DID identifiers (did:plc:..., did:web:...)
  - Authorization headers
  - API keys (32+ character alphanumeric strings)
- **Rate Limiting:** Maximum 10 errors per minute to prevent abuse
- **Endpoint:** `POST /api/log/client-error`

Error logging is essential for identifying and fixing bugs that impact user experience. All data is redacted to protect privacy.

#### Session Replay (Opt-In Only)

Session replay captures user interactions to diagnose complex UX issues:

- **Default:** **OFF** (user must explicitly enable)
- **Events Recorded:** DOM changes (sampled 50%), clicks (100%), navigation (100%), scroll (10%)
- **PII Protection:**
  - No text content captured
  - No form values recorded
  - Element IDs/classes only (no sensitive attributes)
  - Only pathname recorded (no query parameters)
- **Buffer Limit:** 100 events maximum (ring buffer)
- **Performance Threshold:** Recording stops when buffer reaches 90% capacity
- **Transmission:** Only sent when an error occurs AND user has opted in
- **Control:** Toggle in Settings → Privacy → Session Replay

**How to Enable:**
```typescript
// In Settings UI
setSessionReplayOptIn(true);
```

When enabled, session replay provides context for error reports by showing the sequence of interactions leading to the problem. All recorded data is sanitized to prevent PII exposure.

#### Telemetry Opt-Out

Users can disable general analytics collection:

- **Default:** ON (telemetry enabled by default)
- **Scope:** Usage analytics, performance metrics, feature engagement
- **Does not affect:** Error logging (required for stability)
- **Control:** Toggle in Settings → Privacy → Analytics

**How to Disable:**
```typescript
// In Settings UI
setTelemetryOptOut(true);
```

See [Telemetry Documentation](../web/src/lib/TELEMETRY.md) for complete details on what data is collected.

### Authentication

JWT-based authentication with Decentralized Identifiers (DIDs):

- **Access tokens:** 15-minute expiry, includes `did` claim
- **Refresh tokens:** 7-day expiry, no DID (reduces exposure)
- **Algorithm:** HS256 with validated signing method
- **Leeway:** 30-second clock skew tolerance

### Rate Limiting

Tiered rate limits protect against abuse:

| Scope | Limit | Window |
|-------|-------|--------|
| Global | 100 requests | 1 minute |
| Auth endpoints | 10 requests | 1 minute |
| Search endpoints | 30 requests | 1 minute |

Rate limit keys use:
1. Authenticated user DID (preferred)
2. IP address from `X-Forwarded-For`, `X-Real-IP`, or connection (fallback)

Standard headers returned on rate limit:
- `Retry-After`: Seconds until limit resets
- `X-RateLimit-Reset`: Unix timestamp when limit resets

### Request ID Validation

Incoming `X-Request-ID` headers are validated:
- Maximum 128 characters
- Alphanumeric, hyphens, and underscores only
- Invalid IDs are replaced with generated UUIDs

This prevents log injection attacks while supporting request correlation.

## Data Retention

| Data Type | Retention Period | Notes |
|-----------|------------------|-------|
| Access logs | 90 days | Security audit trail |
| Client error logs | 30 days | Debugging and stability analysis |
| Session replay data | 7 days | Only when opted-in; auto-deleted after 7 days |
| Soft-deleted content | 30 days | Grace period before hard delete |
| Session tokens | Until expiry | Access: 15min, Refresh: 7 days |
| Uploaded media | Until user deletion | Subject to storage quotas |
| Telemetry events | 90 days | Aggregated for analytics; not tied to individual users |

**User Rights:**
- Request deletion of error logs via support contact
- Disable session replay at any time (no future data collected)
- Opt out of telemetry without affecting core functionality

## Decentralized Identity

Subcult uses [AT Protocol](https://atproto.com/) for decentralized identity:

- User identity is tied to DIDs, not platform accounts
- Data portability through Jetstream ingestion
- No centralized identity provider lock-in

## Future Enhancements

Privacy improvements tracked in the [Privacy & Safety Epic](https://github.com/subculture-collective/subcults/issues/6):

- [x] EXIF/metadata stripping for uploaded media (service layer implemented)
- [x] Scene visibility controls (public, members-only, hidden)
- [x] Client-side error logging with PII redaction
- [x] Opt-in session replay for UX debugging
- [ ] EXIF stripping integration with media upload API endpoints
- [ ] Search endpoint integration for hidden scene exclusion
- [ ] Location jitter for map display
- [ ] Signed URLs for media access
- [ ] Configurable data export (GDPR-style)
- [ ] Trust graph privacy controls (alliance visibility)
- [ ] Content encryption for private scenes

## Reporting Issues

Found a privacy concern? Please report responsibly:

1. **Security vulnerabilities:** See `SECURITY.md` (when available) or contact maintainers directly
2. **General privacy feedback:** Open a GitHub issue with the `privacy` label

---

*Last updated: January 2025*
