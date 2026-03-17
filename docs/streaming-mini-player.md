# Streaming Mini-Player Integration Guide

## Overview

The streaming mini-player provides persistent audio playback across route changes, allowing users to navigate the application while maintaining their connection to live audio streams.

## Architecture

### Global State Management

The streaming functionality is managed by a global Zustand store (`streamingStore`) that maintains:

- **Connection state**: Room connection, connection quality, error messages
- **Audio state**: Volume level (persisted to localStorage), mute status
- **Reconnection state**: Automatic reconnection with exponential backoff

### Components

1. **MiniPlayer** (`web/src/components/MiniPlayer.tsx`)
   - Bottom-docked persistent player
   - Shows when user is connected to a stream
   - Provides audio controls (mute, volume, disconnect)
   - Accessible via keyboard shortcuts

2. **StreamPage** (`web/src/pages/StreamPage.tsx`)
   - Full streaming room interface
   - Uses global streaming store for connection
   - Shows participant list and connection quality

## Usage

### Connecting to a Stream

```typescript
import { useStreamingActions } from '../stores/streamingStore';

function MyComponent() {
  const { connect } = useStreamingActions();
  
  const handleJoin = async () => {
    await connect('room-name', 'scene-id', 'event-id');
  };
  
  return <button onClick={handleJoin}>Join Stream</button>;
}
```

### Monitoring Connection State

```typescript
import { useStreamingConnection } from '../stores/streamingStore';

function ConnectionStatus() {
  const { isConnected, isConnecting, error, connectionQuality } = useStreamingConnection();
  
  return (
    <div>
      {isConnecting && <p>Connecting...</p>}
      {isConnected && <p>Connected - Quality: {connectionQuality}</p>}
      {error && <p>Error: {error}</p>}
    </div>
  );
}
```

### Audio Controls

```typescript
import { useStreamingAudio } from '../stores/streamingStore';

function VolumeControl() {
  const { volume, isMuted, setVolume, toggleMute } = useStreamingAudio();
  
  return (
    <div>
      <button onClick={toggleMute}>
        {isMuted ? 'Unmute' : 'Mute'}
      </button>
      <input
        type="range"
        min="0"
        max="100"
        value={volume}
        onChange={(e) => setVolume(parseInt(e.target.value))}
      />
    </div>
  );
}
```

## Features

### 1. Persistent Audio Across Routes

The MiniPlayer is rendered in `AppLayout` and remains visible across all route changes as long as a stream is active. The LiveKit room connection is managed globally, independent of component lifecycle.

### 2. Volume Persistence

Volume settings are automatically persisted to `localStorage` under the key `subcults-stream-volume`:

- Default volume: 100
- Range: 0-100
- Restored on app initialization

### 3. Auto-Reconnection

The store implements automatic reconnection with exponential backoff:

- Maximum attempts: 3
- Initial delay: 1 second
- Maximum delay: 10 seconds
- Exponential backoff: delay = min(1000 * 2^attempt, 10000)

Reconnection is triggered on:
- Network disconnection (unexpected)
- Connection failures

### 4. Keyboard Shortcuts

The MiniPlayer supports keyboard shortcuts:

- **Space**: Toggle mute
- **Escape**: Close volume slider (when open)

All controls are keyboard-accessible and follow ARIA best practices.

### 5. Connection Quality Indicator

Visual indicator shows connection quality:

- 🟢 **Excellent**: Green
- 🟡 **Good**: Yellow/Orange
- 🔴 **Poor**: Red
- ⚪ **Unknown**: Gray

## Accessibility

The MiniPlayer follows WCAG 2.1 AA standards:

- All controls have proper ARIA labels
- Volume slider includes aria-valuemin, aria-valuemax, aria-valuenow, aria-valuetext
- Keyboard navigation fully supported
- Focus indicators visible on all interactive elements
- Semantic HTML with proper roles and regions

## State Flow

### Connection Flow

```
User clicks "Join Stream"
  ↓
StreamingStore.connect(roomName, sceneId, eventId)
  ↓
Fetch LiveKit token from API
  ↓
Create Room instance
  ↓
Set up event listeners
  ↓
Connect to LiveKit WebSocket
  ↓
Enable local microphone
  ↓
Update state: isConnected = true
  ↓
MiniPlayer becomes visible
```

### Reconnection Flow

```
Unexpected disconnect detected
  ↓
Check reconnectAttempts < MAX_RECONNECT_ATTEMPTS
  ↓
Set isReconnecting = true
  ↓
Calculate backoff delay
  ↓
Wait delay milliseconds
  ↓
Attempt reconnection with same room/credentials
  ↓
Success: Reset reconnectAttempts
Failure: Increment reconnectAttempts and retry
```

### Disconnection Flow

```
User clicks "Leave"
  ↓
StreamingStore.disconnect()
  ↓
Remove all room event listeners
  ↓
Call room.disconnect()
  ↓
Clear participant store
  ↓
Reset state to initial values
  ↓
MiniPlayer disappears
```

## Performance Optimizations

1. **Memoization**: MiniPlayer component is wrapped in `React.memo` to prevent unnecessary re-renders

2. **Selector Optimization**: Store hooks use selective subscriptions to minimize re-renders:
   ```typescript
   // Only re-renders when volume changes
   const volume = useStreamingStore((state) => state.volume);
   ```

3. **Event Handler Stability**: Volume and mute handlers are stable references from the store

4. **Conditional Rendering**: MiniPlayer only renders when `isConnected && roomName` are truthy

## Testing

### Running Tests

```bash
cd web
npm test -- streamingStore.test.ts
npm test -- MiniPlayer.test.tsx
```

### Test Coverage

- **Volume persistence**: localStorage read/write, clamping
- **Connection management**: connect, disconnect, state transitions
- **Reconnection logic**: exponential backoff, max attempts
- **Route persistence**: state maintained across navigation
- **Audio controls**: mute toggle, volume change
- **Error handling**: connection failures, missing config
- **Accessibility**: ARIA attributes, keyboard navigation

## Configuration

### Environment Variables

Required environment variables in `.env`:

```bash
# LiveKit WebSocket URL
VITE_LIVEKIT_WS_URL=wss://your-livekit-server.com
```

### Customization

To customize reconnection behavior, edit constants in `streamingStore.ts`:

```typescript
const MAX_RECONNECT_ATTEMPTS = 3; // Maximum retry attempts
const INITIAL_RECONNECT_DELAY = 1000; // 1 second
const MAX_RECONNECT_DELAY = 10000; // 10 seconds
```

To customize volume storage key:

```typescript
const VOLUME_STORAGE_KEY = 'subcults-stream-volume';
```

## Troubleshooting

### MiniPlayer Not Appearing

1. Check connection state: `useStreamingConnection()`
2. Verify `isConnected === true` and `roomName !== null`
3. Check browser console for errors

### Volume Not Persisting

1. Check localStorage is enabled
2. Verify localStorage key: `localStorage.getItem('subcults-stream-volume')`
3. Check for localStorage quota errors in console

### Reconnection Not Working

1. Check network connectivity
2. Verify `reconnectAttempts < MAX_RECONNECT_ATTEMPTS`
3. Check console for reconnection logs
4. Verify LiveKit WebSocket URL is configured

### Audio Not Playing

1. Check browser audio permissions
2. Verify microphone access granted
3. Check volume slider is not at 0
4. Verify not muted
5. Check LiveKit server is running and accessible

## Security Considerations

### Token Management

LiveKit tokens are fetched from the backend API and should:

- Have appropriate expiration times (e.g., 1 hour)
- Include minimal required permissions
- Be refreshed before expiry (currently requires manual rejoin)

### Privacy

The mini-player does not log or persist:

- User DIDs beyond necessary debug info
- Room contents or audio data
- Participant information beyond what's needed for display

## Future Enhancements

Potential improvements for future iterations:

1. **Token Refresh**: Seamless token refresh without manual rejoin
2. **Notification Permissions**: Browser notifications for connection issues
3. **Bandwidth Adaptation**: Automatic quality adjustment based on network
4. **Picture-in-Picture**: Minimize to floating window
5. **Audio Visualizer**: Waveform display for active speakers
6. **Recording**: Save stream audio locally
7. **Screen Sharing**: Add video/screen sharing support

## API Reference

See the following files for detailed API documentation:

- `web/src/stores/streamingStore.ts` - Global streaming store
- `web/src/components/MiniPlayer.tsx` - Mini-player component
- `web/src/pages/StreamPage.tsx` - Full streaming page
- `web/src/types/streaming.ts` - TypeScript type definitions
