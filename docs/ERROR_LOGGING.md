# Client-Side Error Logging & Session Replay

Privacy-conscious error tracking and diagnostic tools for debugging production issues.

## Overview

The error logging system provides:

- **Automatic error capture**: React errors, unhandled exceptions, promise rejections
- **PII redaction**: Removes tokens, emails, DIDs from all error data
- **Rate limiting**: Max 10 errors per minute per session
- **Session replay (opt-in)**: Captures user interactions for debugging complex UX issues
- **Privacy-first design**: Session replay defaults to OFF, requires explicit user consent

## Components

### 1. Error Logger (`error-logger.ts`)

Captures and redacts errors before sending to backend.

**Features:**
- Automatic redaction of sensitive patterns (JWT tokens, emails, DIDs, API keys)
- Rate limiting to prevent error storms
- Session ID for error grouping
- Integration with session replay

**Usage:**
```typescript
import { errorLogger } from './lib/error-logger';

// Log an error manually
try {
  riskyOperation();
} catch (error) {
  errorLogger.logError(error);
}

// Automatically integrated with:
// - ErrorBoundary component (React errors)
// - window.addEventListener('error') (global errors)
// - window.addEventListener('unhandledrejection') (promise rejections)
```

**API Endpoint:**
```
POST /api/log/client-error
Content-Type: application/json

{
  "message": "Error message (redacted)",
  "stack": "Stack trace (redacted)",
  "type": "ErrorType",
  "timestamp": 1234567890,
  "url": "https://app.subcults.com/scenes",
  "userAgent": "Mozilla/5.0...",
  "componentStack": "Component trace (redacted, React errors only)",
  "sessionId": "uuid",
  "replayEvents": [ /* only if session replay is enabled */ ]
}
```

### 2. Session Replay (`session-replay.ts`)

Records user interactions in a privacy-conscious manner.

**Features:**
- Opt-in only (default: OFF)
- Sanitizes all DOM data (no text content, no form values)
- Ring buffer (max 100 events)
- Performance-aware (stops recording at 90% buffer capacity)
- Smart sampling (50% DOM mutations, 100% clicks, 10% scrolls)

**Event Types:**
- **Click**: Element type, ID, class, coordinates
- **Navigation**: Pathname only (no query params)
- **Scroll**: Scroll coordinates
- **DOM Change**: Mutation type, target element (sanitized)

**Usage:**
```typescript
import { sessionReplay } from './lib/session-replay';

// Start recording (checks opt-in status internally)
sessionReplay.start();

// Stop recording
sessionReplay.stop();

// Get events (automatically called by error logger)
const events = sessionReplay.getAndClearBuffer();
```

**Privacy Guarantees:**
- No text content captured
- No input values recorded
- No sensitive attributes (data-*, aria-label with PII)
- Only pathname recorded (no query strings)
- Automatic redaction of any captured data

### 3. Settings Store Integration

User controls for telemetry and session replay.

**Settings:**
```typescript
interface SettingsState {
  telemetryOptOut: boolean;      // Default: false (telemetry enabled)
  sessionReplayOptIn: boolean;   // Default: false (replay disabled)
}
```

**Hooks:**
```typescript
import { 
  useSessionReplayOptIn, 
  useSettingsActions 
} from './stores/settingsStore';

function PrivacySettings() {
  const sessionReplayEnabled = useSessionReplayOptIn();
  const { setSessionReplayOptIn } = useSettingsActions();

  return (
    <label>
      <input
        type="checkbox"
        checked={sessionReplayEnabled}
        onChange={(e) => setSessionReplayOptIn(e.target.checked)}
      />
      Enable session replay (helps us diagnose UX issues)
    </label>
  );
}
```

## Privacy Architecture

### Redaction Patterns

All error data is automatically redacted before transmission:

| Pattern | Example | Redacted To |
|---------|---------|-------------|
| JWT tokens | `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...` | `[REDACTED]` |
| Email addresses | `user@example.com` | `[REDACTED]` |
| DID identifiers | `did:plc:abc123xyz` | `[REDACTED]` |
| Authorization headers | `Authorization: Bearer token123` | `[REDACTED]` |
| API keys | `sk_live_1234567890abcdef...` | `[REDACTED]` |

### Session Replay Privacy

**What IS captured:**
- Element types (button, div, input)
- Element IDs and classes (non-sensitive only)
- Click coordinates
- Scroll positions
- Navigation paths (pathname only)

**What is NOT captured:**
- Text content
- Input values
- Form data
- Query parameters
- Sensitive attributes (data-user-id, aria-label with PII)

### Data Flow

```
User Action → Event → Privacy Check → Sanitize → Buffer → (On Error) → Redact → Send to Backend
                          ↓                                                          ↓
                   Opt-in status?                                            Remove PII patterns
                          ↓
                    If NO: drop event
                    If YES: continue
```

## Rate Limiting

Error logging is rate-limited to prevent abuse and reduce backend load.

**Limits:**
- **Maximum:** 10 errors per minute per session
- **Behavior:** Errors beyond limit are dropped (not queued)
- **Reset:** Counter resets every 60 seconds
- **Logging:** Dropped errors are logged to console in development

**Example:**
```typescript
// First 10 errors are sent
for (let i = 0; i < 10; i++) {
  errorLogger.logError(new Error(`Error ${i}`)); // Sent
}

// 11th error is dropped
errorLogger.logError(new Error('Error 11')); // Dropped (rate limit)

// After 60 seconds, counter resets
setTimeout(() => {
  errorLogger.logError(new Error('Error 12')); // Sent (new window)
}, 60000);
```

## Performance Considerations

### Session Replay Buffer Management

The session replay service uses a ring buffer with performance thresholds:

1. **Buffer Size:** 100 events maximum
2. **Ring Buffer:** When full, oldest event is dropped to make room
3. **Performance Threshold:** Recording stops when buffer reaches 90% capacity (90 events)
4. **Sampling:** Reduces event volume
   - DOM mutations: 50% sample rate
   - Clicks: 100% sample rate
   - Scrolls: 10% sample rate (throttled)

**Early Exit Example:**
```typescript
// Buffer at 90% capacity (90/100 events)
sessionReplay.getBufferSize(); // 90

// New DOM mutation event arrives
// Check: 90 >= 90 (threshold reached)
// Action: Drop event, don't process mutation
```

### Error Logger Overhead

- **Redaction:** ~1ms per error (regex matching)
- **Fetch:** Asynchronous, non-blocking
- **Rate limit check:** O(1) counter check
- **Session ID:** Cached in sessionStorage

## Testing

### Unit Tests

**Error Logger:**
```bash
npm test -- error-logger.test.ts
```

Coverage:
- Redaction patterns (JWT, email, DID, API keys)
- Rate limiting
- Error payload structure
- Fetch error handling

**Session Replay:**
```bash
npm test -- session-replay.test.ts
```

Coverage:
- Opt-in gating
- Event recording (click, navigation, scroll, DOM mutations)
- Buffer management (ring buffer, performance threshold)
- Privacy sanitization

**Settings Store:**
```bash
npm test -- settingsStore.test.ts
```

Coverage:
- sessionReplayOptIn flag
- localStorage persistence
- Default values (privacy-first)

### Manual Testing

**Error Logging:**
```typescript
// Trigger a test error
throw new Error('Test error with email user@test.com and DID did:plc:123');

// Check network tab for POST to /api/log/client-error
// Verify message is redacted: "Test error with email [REDACTED] and DID [REDACTED]"
```

**Session Replay:**
```typescript
// Enable session replay
useSettingsStore.getState().setSessionReplayOptIn(true);

// Reload page to start recording
sessionReplay.start();

// Perform actions (clicks, navigation, scrolling)
// Trigger an error
throw new Error('Test error');

// Check network tab - error payload should include replayEvents array
```

## Backend Integration

### Error Logging Endpoint

**Schema (Go):**
```go
type ClientErrorLog struct {
    Message        string                   `json:"message"`
    Stack          *string                  `json:"stack,omitempty"`
    Type           string                   `json:"type"`
    Timestamp      int64                    `json:"timestamp"`
    URL            string                   `json:"url"`
    UserAgent      string                   `json:"user_agent"`
    ComponentStack *string                  `json:"componentStack,omitempty"`
    SessionID      string                   `json:"sessionId"`
    ReplayEvents   []SessionReplayEvent     `json:"replayEvents,omitempty"`
}

type SessionReplayEvent struct {
    Type      string                 `json:"type"`
    Timestamp int64                  `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
}
```

**Handler (example):**
```go
func handleClientError(w http.ResponseWriter, r *http.Request) {
    var payload ClientErrorLog
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "Invalid payload", http.StatusBadRequest)
        return
    }

    // Validate
    if payload.Message == "" || payload.SessionID == "" {
        http.Error(w, "Missing required fields", http.StatusBadRequest)
        return
    }

    // Store in database or logging service
    logger.Error("Client error",
        "message", payload.Message,
        "type", payload.Type,
        "session_id", payload.SessionID,
        "has_replay", len(payload.ReplayEvents) > 0,
    )

    w.WriteHeader(http.StatusOK)
}
```

## Configuration

### Error Logger Config

```typescript
const errorLogger = new ErrorLogger({
  maxErrorsPerMinute: 10,              // Rate limit threshold
  endpoint: '/api/log/client-error',   // Backend endpoint
  consoleLogging: true,                // Log to console in dev mode
});
```

### Session Replay Config

```typescript
const sessionReplay = new SessionReplay({
  maxBufferSize: 100,          // Maximum events before ring buffer overflow
  domMutationSampleRate: 0.5,  // 50% of DOM mutations
  clickSampleRate: 1.0,        // 100% of clicks
  performanceMonitoring: true, // Enable 90% threshold early exit
});
```

## Troubleshooting

### Error logs not appearing in backend

1. Check rate limit: `errorLogger.getErrorCount()`
2. Verify endpoint URL in network tab
3. Check browser console for fetch errors
4. Verify CORS configuration on backend

### Session replay not capturing events

1. Check opt-in status: `useSessionReplayOptIn()`
2. Verify recording started: `sessionReplay.isActive()`
3. Check buffer size: `sessionReplay.getBufferSize()`
4. Ensure settings initialized: `useSettingsStore.getState().initializeSettings()`

### Too many events captured

1. Reduce sample rates in config
2. Increase buffer size to avoid drops
3. Check for event loops triggering DOM mutations

## Security Considerations

### Redaction Bypass Prevention

The error logger applies redaction at the **payload construction** level, not just display:

```typescript
// Redaction is applied before sending
const payload = {
  message: redactSensitiveData(error.message),  // ✅ Redacted
  stack: redactSensitiveData(error.stack),      // ✅ Redacted
  // ... other fields
};

// NOT just:
console.log(redactSensitiveData(payload)); // ❌ Would not protect backend
```

### Session Replay Opt-Out Enforcement

Session replay checks opt-in status at multiple layers:

1. **Start:** `sessionReplay.start()` exits early if opted out
2. **Event capture:** Each event handler checks `isOptedIn()` before recording
3. **Buffer retrieval:** `getAndClearBuffer()` returns `[]` if opted out

This defense-in-depth approach prevents accidental data collection.

## References

- [Telemetry Documentation](./TELEMETRY.md)
- [Privacy Policy](../../docs/PRIVACY.md)
- [Settings Store](../stores/settingsStore.ts)
