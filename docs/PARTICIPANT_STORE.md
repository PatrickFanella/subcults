# Participant Store Documentation

The participant store manages LiveKit participant state with real-time synchronization, providing centralized state management for audio room participants.

## Overview

The participant store is built using Zustand and provides:
- Centralized participant state management
- Debounced mute updates (50ms) to prevent UI flicker
- Identity normalization
- Real-time LiveKit event synchronization
- Performance-optimized selectors

## Features

### Debounced Mute Updates

Mute state changes are debounced by 50ms to handle rapid toggle events without causing UI flicker. This ensures smooth UX even when users rapidly toggle their microphone.

### Identity Normalization

The store automatically normalizes participant identities by stripping common prefixes (`user:`, `participant:`). This ensures consistent identity handling across the application.

### Metadata Tracking

Each participant has associated metadata:
- `timestamp`: When the participant was added to the store
- `lastMuteUpdate`: When the mute state was last changed

## API Reference

### Store State

```typescript
interface ParticipantState {
  participants: Record<string, CachedParticipant>;
  localIdentity: string | null;
  pendingMuteUpdates: Record<string, NodeJS.Timeout>;
}
```

### Store Actions

#### `addParticipant(participant: Participant): void`

Adds or updates a participant in the store. Identity is automatically normalized.

```typescript
participantStore.addParticipant({
  identity: 'user:alice',
  name: 'Alice',
  isLocal: false,
  isMuted: false,
  isSpeaking: false,
});
```

#### `removeParticipant(identity: string): void`

Removes a participant from the store. Automatically cancels any pending mute updates.

```typescript
participantStore.removeParticipant('alice');
```

#### `updateParticipantMute(identity: string, isMuted: boolean): void`

Updates a participant's mute status with 50ms debouncing. This prevents UI flicker on rapid mute toggles.

```typescript
participantStore.updateParticipantMute('alice', true);
```

#### `updateParticipantSpeaking(identity: string, isSpeaking: boolean): void`

Updates a participant's speaking status immediately (no debounce). Used for active speaker indicators.

```typescript
participantStore.updateParticipantSpeaking('alice', true);
```

#### `setLocalIdentity(identity: string | null): void`

Sets the local participant's identity for quick lookup.

```typescript
participantStore.setLocalIdentity('alice');
```

#### `clearParticipants(): void`

Removes all participants and clears pending updates. Used when disconnecting from a room.

```typescript
participantStore.clearParticipants();
```

#### `getParticipantsArray(): Participant[]`

Returns all participants as an array.

```typescript
const participants = participantStore.getParticipantsArray();
```

#### `getParticipant(identity: string): Participant | null`

Returns a specific participant by identity.

```typescript
const participant = participantStore.getParticipant('alice');
```

#### `getLocalParticipant(): Participant | null`

Returns the local participant.

```typescript
const local = participantStore.getLocalParticipant();
```

### React Hooks

#### `useParticipants(): Participant[]`

Hook to get all remote participants (excludes local participant).

```typescript
import { useParticipants } from '../stores/participantStore';

function MyComponent() {
  const participants = useParticipants();
  return <div>Remote participants: {participants.length}</div>;
}
```

**Note**: This hook uses shallow comparison to minimize re-renders.

#### `useParticipant(identity: string): Participant | null`

Hook to get a specific participant by identity.

```typescript
import { useParticipant } from '../stores/participantStore';

function ParticipantDetail({ id }: { id: string }) {
  const participant = useParticipant(id);
  if (!participant) return <div>Not found</div>;
  return <div>{participant.name}</div>;
}
```

#### `useLocalParticipant(): Participant | null`

Hook to get the local participant.

```typescript
import { useLocalParticipant } from '../stores/participantStore';

function LocalControls() {
  const local = useLocalParticipant();
  return <div>{local?.name} (You)</div>;
}
```

## Integration with useLiveAudio

The participant store is automatically integrated with the `useLiveAudio` hook. LiveKit events are synchronized to the store:

- `ParticipantConnected` → `addParticipant()`
- `ParticipantDisconnected` → `removeParticipant()`
- `TrackMuted` → `updateParticipantMute(..., true)`
- `TrackUnmuted` → `updateParticipantMute(..., false)`
- `ActiveSpeakersChanged` → `updateParticipantSpeaking()`

### Example Integration

```typescript
import { useLiveAudio } from '../hooks/useLiveAudio';
import { useParticipants, useLocalParticipant } from '../stores/participantStore';

function StreamRoom() {
  const { connect, disconnect, toggleMute } = useLiveAudio('room-123');
  const participants = useParticipants();
  const local = useLocalParticipant();

  return (
    <div>
      <button onClick={connect}>Join</button>
      <button onClick={disconnect}>Leave</button>
      <button onClick={toggleMute}>
        {local?.isMuted ? 'Unmute' : 'Mute'}
      </button>
      <ul>
        {participants.map(p => (
          <li key={p.identity}>
            {p.name} {p.isSpeaking && '🎤'} {p.isMuted && '🔇'}
          </li>
        ))}
      </ul>
    </div>
  );
}
```

## Performance Characteristics

### Debouncing

- **Mute updates**: 50ms debounce
- **Speaking updates**: No debounce (immediate)
- **Participant add/remove**: Immediate

### Timing Guarantees (per acceptance criteria)

- **Participant removal**: ≤1s from disconnect event
- **Mute toggle reflection**: ≤250ms (actual: 50ms debounce)

### Memory Management

- Pending timeouts are automatically cleared on participant removal
- All timeouts are cleared on `clearParticipants()`
- No memory leaks from orphaned timers

## Testing

The store includes comprehensive test coverage:

```bash
npm test -- participantStore.test.ts
```

Test coverage includes:
- Identity normalization
- Add/update/remove operations
- Mute debouncing
- Speaking status updates
- Event sequences
- Hook selectors
- Edge cases (rapid toggles, non-existent participants, etc.)

## Best Practices

1. **Always use normalized identities**: The store handles this automatically
2. **Don't bypass debouncing**: Use `updateParticipantMute()` for mute changes
3. **Clean up on disconnect**: Call `clearParticipants()` when leaving a room
4. **Use hooks in components**: Prefer `useParticipants()` over direct store access
5. **Trust the integration**: The `useLiveAudio` hook manages synchronization automatically

## Security & Privacy

- No PII beyond display names is stored
- Identities are normalized to prevent leaking internal ID formats
- No participant data persists after disconnect
- All operations are client-side only

## Migration from Local State

If migrating from component-local participant state:

**Before:**
```typescript
const [participants, setParticipants] = useState<Participant[]>([]);
```

**After:**
```typescript
import { useParticipants } from '../stores/participantStore';
const participants = useParticipants();
```

No manual synchronization needed—the store handles LiveKit events automatically via `useLiveAudio`.
