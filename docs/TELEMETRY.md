# Client Telemetry System

Lightweight event bus for collecting structured analytics events with privacy-first design.

## Overview

The telemetry system provides:
- **Event batching**: Automatically batches events and sends them in intervals (5s) or when reaching size threshold (20 events)
- **Retry logic**: Retries failed network requests once with exponential backoff
- **Privacy opt-out**: Users can disable telemetry collection via settings
- **Session tracking**: Each browser tab gets a unique session ID (persists until tab closes)
- **Automatic auth integration**: Includes user DID when authenticated

## Architecture

### Components

1. **TelemetryEvent** (`types/telemetry.ts`): Type definitions for events
2. **TelemetryService** (`lib/telemetry-service.ts`): Core event queue and batching logic
3. **useTelemetry** (`hooks/useTelemetry.ts`): React hook for emitting events
4. **SettingsStore** (`stores/settingsStore.ts`): Privacy opt-out flag

### Data Flow

```
Component → useTelemetry() → TelemetryService → Queue → Batch POST /api/telemetry
                                                              ↓ (on error)
                                                           Retry (1x)
```

## Usage

### Basic Event Emission

```tsx
import { useTelemetry } from '../hooks';

function SearchComponent() {
  const emit = useTelemetry();

  const handleSearch = (query: string) => {
    // ... perform search ...
    
    emit('search.scene', {
      query_length: query.length,
      results_count: results.length,
    });
  };

  return <SearchBar onSearch={handleSearch} />;
}
```

### Integration Example: Search with Telemetry

```tsx
import { useState } from 'react';
import { useTelemetry } from '../hooks';
import { apiClient } from '../lib/api-client';

function SceneSearch() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const emit = useTelemetry();

  const handleSearch = async () => {
    const startTime = Date.now();
    
    try {
      const scenes = await apiClient.searchScenes(query);
      setResults(scenes);
      
      // Emit successful search event
      emit('search.scene', {
        query_length: query.length,
        results_count: scenes.length,
        duration_ms: Date.now() - startTime,
      });
    } catch (error) {
      // Emit failed search event
      emit('search.error', {
        query_length: query.length,
        error_type: 'network',
      });
    }
  };

  return (
    <div>
      <input value={query} onChange={(e) => setQuery(e.target.value)} />
      <button onClick={handleSearch}>Search</button>
    </div>
  );
}
```

### Integration Example: Stream Join with Telemetry

```tsx
import { useTelemetry } from '../hooks';
import { useStreamingActions } from '../stores';

function StreamControls({ roomId }: { roomId: string }) {
  const emit = useTelemetry();
  const { connect, disconnect } = useStreamingActions();

  const handleJoin = async () => {
    const joinTime = Date.now();
    
    try {
      await connect(roomId);
      
      emit('stream.join', {
        room_id: roomId,
      });
      
      // Track duration on leave
      return () => {
        emit('stream.leave', {
          room_id: roomId,
          duration_ms: Date.now() - joinTime,
        });
      };
    } catch (error) {
      emit('stream.join_error', {
        room_id: roomId,
        error_type: 'connection_failed',
      });
    }
  };

  return <button onClick={handleJoin}>Join Stream</button>;
}
```

### Event Naming Conventions

Use **dot-notation** to organize events hierarchically:

| Category | Event Name | Purpose |
|----------|------------|---------|
| **Search** | `search.scene` | Scene search performed |
| | `search.event` | Event search performed |
| | `search.post` | Post search performed |
| **Streaming** | `stream.join` | Joined audio stream |
| | `stream.leave` | Left audio stream |
| **Scene/Event** | `scene.view` | Viewed scene details |
| | `event.view` | Viewed event details |
| **Map** | `map.zoom` | Map zoom level changed |
| | `map.move` | Map viewport moved |

### Payload Guidelines

**DO:**
- Keep payloads minimal (only essential metadata)
- Use aggregated metrics (e.g., `query_length` not `query`)
- Include non-sensitive IDs (e.g., `scene_id`, `room_id`)

**DON'T:**
- Include sensitive content (PII, precise locations, full text)
- Send large objects or arrays
- Log user-generated content verbatim

**Examples:**

✅ Good:
```typescript
emit('search.scene', { 
  query_length: 5, 
  results_count: 10 
});

emit('stream.join', { 
  room_id: 'xyz', 
  duration_ms: 1234 
});
```

❌ Bad:
```typescript
emit('search.scene', { 
  query: 'underground techno berlin', // Contains user input
  user_location: [52.5200, 13.4050] // Precise location
});
```

## Privacy

### Opt-Out

Users can disable telemetry via settings:

```tsx
import { useSettingsActions } from '../stores';

function PrivacySettings() {
  const { setTelemetryOptOut } = useSettingsActions();

  return (
    <button onClick={() => setTelemetryOptOut(true)}>
      Disable Analytics
    </button>
  );
}
```

When opted out:
- No events are queued
- No network requests are made
- Setting persists in localStorage

### Session ID

- **Scope**: Single browser tab
- **Lifetime**: Until tab closes (persists across page reloads within the tab)
- **Format**: UUID v4
- **Purpose**: Track user journey within a session (not across tabs/sessions)

### User ID

- **Included**: Only if user is authenticated
- **Format**: DID (Decentralized Identifier)
- **Automatic**: useTelemetry() hook handles this

## Setup & Initialization

The settings store must be initialized on app startup to load persisted preferences from localStorage:

```typescript
// In App.tsx or root component:
import { useEffect } from 'react';
import { useSettingsStore } from './stores';

function App() {
  useEffect(() => {
    // Initialize settings from localStorage
    useSettingsStore.getState().initializeSettings();
  }, []);
  
  // ... rest of app
}
```

**Important:** Without this initialization, user opt-out preferences will not be respected until they manually change settings in the current session.

## Configuration

Default configuration (can be overridden for testing):

```typescript
{
  flushInterval: 5000,    // 5 seconds
  maxBatchSize: 20,       // events
  maxRetries: 1,          // attempts
  retryDelay: 1000,       // 1 second
}
```

## API Endpoint

Events are sent to:

```
POST /api/telemetry
Content-Type: application/json

{
  "events": [
    {
      "name": "search.scene",
      "ts": 1234567890123,
      "sessionId": "uuid-v4",
      "userId": "did:plc:...", // optional
      "payload": { ... }        // optional
    }
  ]
}
```

## Testing

### Unit Tests

- **settingsStore.test.ts**: Opt-out persistence and state management
- **telemetry-service.test.ts**: Queue, batching, retry logic, opt-out bypass
- **useTelemetry.test.ts**: Hook behavior, auth integration, cleanup

Run tests:

```bash
npm test -- settingsStore.test
npm test -- telemetry-service.test
npm test -- useTelemetry.test
```

### Manual Testing

```tsx
// Enable telemetry
useSettingsActions().setTelemetryOptOut(false);

// Emit test event
const emit = useTelemetry();
emit('test.event', { foo: 'bar' });

// Check network tab for POST /api/telemetry after 5s
```

## Future Enhancements

- [ ] Add telemetry dashboard (admin-only)
- [ ] Implement server-side event processing
- [ ] Add event schema validation
- [ ] Support custom flush intervals per event type
- [ ] Integrate with product analytics platform
- [ ] Add A/B testing support via event metadata
