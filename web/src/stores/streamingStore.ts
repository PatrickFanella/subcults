/**
 * Streaming Store
 * Manages global LiveKit connection state for persistent audio across routes
 */

import { create } from 'zustand';
import {
  Room,
  RoomEvent,
  ConnectionQuality as LKConnectionQuality,
  DisconnectReason,
  Participant as LKParticipant,
  Track,
} from 'livekit-client';
import { apiClient } from '../lib/api-client';
import { useParticipantStore, normalizeIdentity } from '../stores/participantStore';
import { useLatencyStore } from '../stores/latencyStore';
import type { ConnectionQuality, Participant } from '../types/streaming';

/**
 * Reconnection configuration
 */
const MAX_RECONNECT_ATTEMPTS = 3;
const INITIAL_RECONNECT_DELAY = 1000; // 1 second
const MAX_RECONNECT_DELAY = 10000; // 10 seconds

/**
 * Local storage keys
 */
const VOLUME_STORAGE_KEY = 'subcults-stream-volume';
const VOLUME_DEFAULT = 100;

/**
 * Streaming state
 */
interface StreamingState {
  // Connection state
  room: Room | null;
  roomName: string | null;
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
  connectionQuality: ConnectionQuality;

  // Audio state
  volume: number;
  isLocalMuted: boolean; // Local microphone mute state (whether others can hear you)

  // Reconnection state
  reconnectAttempts: number;
  isReconnecting: boolean;
  reconnectTimeoutId: ReturnType<typeof setTimeout> | null;

  // Room metadata
  sceneId?: string;
  eventId?: string;
}

/**
 * Streaming actions
 */
interface StreamingActions {
  // Connection management
  connect: (roomName: string, sceneId?: string, eventId?: string) => Promise<void>;
  disconnect: () => void;

  // Audio controls
  setVolume: (volume: number) => void;
  toggleMute: () => Promise<void>;

  // Internal state management
  setConnectionQuality: (quality: ConnectionQuality) => void;
  setError: (error: string | null) => void;

  // Reconnection
  scheduleReconnect: () => void;

  // Initialization
  initialize: () => void;
}

export type StreamingStore = StreamingState & StreamingActions;

/**
 * Get initial volume from localStorage
 */
function getInitialVolume(): number {
  try {
    const stored = localStorage.getItem(VOLUME_STORAGE_KEY);
    if (stored) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= 0 && parsed <= 100) {
        return parsed;
      }
    }
  } catch (error) {
    console.warn('Failed to read volume from localStorage:', error);
  }
  return VOLUME_DEFAULT;
}

/**
 * Persist volume to localStorage
 */
function persistVolume(volume: number): void {
  try {
    localStorage.setItem(VOLUME_STORAGE_KEY, volume.toString());
  } catch (error) {
    console.warn('Failed to persist volume to localStorage:', error);
  }
}

/**
 * Map LiveKit connection quality to our quality type
 */
function mapConnectionQuality(lkQuality: LKConnectionQuality): ConnectionQuality {
  switch (lkQuality) {
    case LKConnectionQuality.Excellent:
      return 'excellent';
    case LKConnectionQuality.Good:
      return 'good';
    case LKConnectionQuality.Poor:
      return 'poor';
    default:
      return 'unknown';
  }
}

/**
 * Convert LiveKit participant to our Participant type
 */
function convertParticipant(participant: LKParticipant, isLocal: boolean): Participant {
  const audioTrack = participant.getTrackPublication(Track.Source.Microphone);

  return {
    identity: participant.identity,
    name: participant.name || participant.identity,
    isLocal,
    isMuted: audioTrack?.isMuted ?? true,
    isSpeaking: participant.isSpeaking,
  };
}

/**
 * Update participants list from room
 */
function updateParticipants(room: Room) {
  const store = useParticipantStore.getState();

  // Update remote participants in store
  room.remoteParticipants.forEach((participant) => {
    const converted = convertParticipant(participant, false);
    store.addParticipant(converted);
  });

  // Update local participant in store
  if (room.localParticipant) {
    const localPart = convertParticipant(room.localParticipant, true);
    store.addParticipant(localPart);
    store.setLocalIdentity(localPart.identity);
  }
}

/**
 * Track rooms that already have volume event handlers attached
 * to avoid registering duplicate listeners.
 */
const roomsWithVolumeHandlers = new WeakSet<Room>();

/**
 * Apply volume to all audio tracks for a given participant.
 */
function applyVolumeToParticipant(participant: LKParticipant, normalizedVolume: number) {
  participant.audioTrackPublications.forEach((publication) => {
    if (publication.audioTrack) {
      try {
        // Type guard for setVolume method
        if (typeof (publication.audioTrack as { setVolume?: unknown }).setVolume === 'function') {
          (publication.audioTrack as { setVolume: (volume: number) => void }).setVolume(
            normalizedVolume
          );
        }
      } catch (error) {
        console.warn('Volume control not supported:', error);
      }
    }
  });
}

/**
 * Apply volume to all remote audio tracks in a room
 * and ensure newly-joined participants also get the correct volume.
 */
function applyVolumeToRoom(room: Room, volume: number): void {
  const normalizedVolume = volume / 100;

  // Register event listeners once per room so that
  // participants who join later also receive the preferred volume.
  if (!roomsWithVolumeHandlers.has(room)) {
    roomsWithVolumeHandlers.add(room);

    room.on(RoomEvent.ParticipantConnected, (participant: LKParticipant) => {
      const state = useStreamingStore.getState();
      const effectiveVolume = state.volume / 100;
      applyVolumeToParticipant(participant, effectiveVolume);
    });

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    room.on(RoomEvent.TrackSubscribed, (_track, publication, _participant: LKParticipant) => {
      if (publication.audioTrack) {
        const state = useStreamingStore.getState();
        const effectiveVolume = state.volume / 100;
        try {
          if (typeof (publication.audioTrack as { setVolume?: unknown }).setVolume === 'function') {
            (publication.audioTrack as { setVolume: (volume: number) => void }).setVolume(
              effectiveVolume
            );
          }
        } catch (error) {
          console.warn('Volume control not supported:', error);
        }
      }
    });
  }

  // Initial application for all current remote participants
  room.remoteParticipants.forEach((participant) => {
    applyVolumeToParticipant(participant, normalizedVolume);
  });
}

/**
 * Streaming store with global connection management
 */
export const useStreamingStore = create<StreamingStore>((set, get) => ({
  // Initial state
  room: null,
  roomName: null,
  isConnected: false,
  isConnecting: false,
  error: null,
  connectionQuality: 'unknown',
  volume: getInitialVolume(),
  isLocalMuted: false,
  reconnectAttempts: 0,
  isReconnecting: false,
  reconnectTimeoutId: null,

  /**
   * Initialize the store (call on app startup)
   * Re-reads volume from localStorage in case it was set after store creation
   */
  initialize: () => {
    const storedVolume = getInitialVolume();
    set({ volume: storedVolume });
  },

  /**
   * Connect to a LiveKit room
   */
  connect: async (roomName: string, sceneId?: string, eventId?: string) => {
    const state = get();

    // Don't reconnect if already connected to same room
    if (state.isConnected && state.roomName === roomName) {
      return;
    }

    // Disconnect from any existing connection
    if (state.room) {
      state.disconnect();
    }

    // Reset latency tracking for new join attempt
    const latencyStore = useLatencyStore.getState();
    latencyStore.resetLatency();
    useLatencyStore.setState((prev) => ({
      ...prev,
      lastLatency: null,
    }));

    set({
      isConnecting: true,
      error: null,
      roomName,
      sceneId,
      eventId,
      reconnectAttempts: 0,
    });

    try {
      // Fetch token (t1: token received)
      const { token } = await apiClient.getLiveKitToken(roomName, sceneId, eventId);

      // Record token received timestamp
      latencyStore.recordTokenReceived();

      // Create room
      const room = new Room();

      // Set up connection quality monitoring
      room.on(RoomEvent.ConnectionQualityChanged, (quality: LKConnectionQuality) => {
        get().setConnectionQuality(mapConnectionQuality(quality));
      });

      // Set up disconnect handler
      room.on(RoomEvent.Disconnected, (reason?: DisconnectReason) => {
        const state = get();
        const isClientInitiated = reason === DisconnectReason.CLIENT_INITIATED;

        if (!isClientInitiated) {
          // Unexpected disconnect - attempt reconnection
          console.warn('Unexpected disconnect:', reason);
          set({
            isConnected: false,
            error: 'Connection lost. Attempting to reconnect...',
          });
          state.scheduleReconnect();
        } else {
          // Clean disconnect
          set({
            isConnected: false,
            isConnecting: false,
            room: null,
            error: null,
          });
        }
      });

      // Set up participant tracking event listeners
      room.on(RoomEvent.ParticipantConnected, (participant: LKParticipant) => {
        const participantStore = useParticipantStore.getState();
        const converted = convertParticipant(participant, false);
        participantStore.addParticipant(converted);
        updateParticipants(room);
      });

      room.on(RoomEvent.ParticipantDisconnected, (participant: LKParticipant) => {
        const participantStore = useParticipantStore.getState();
        participantStore.removeParticipant(participant.identity);
        updateParticipants(room);
      });

      room.on(RoomEvent.LocalTrackPublished, () => {
        updateParticipants(room);
      });

      room.on(RoomEvent.LocalTrackUnpublished, () => {
        updateParticipants(room);
      });

      room.on(RoomEvent.TrackMuted, (publication, participant: LKParticipant) => {
        if (publication.source === Track.Source.Microphone) {
          const participantStore = useParticipantStore.getState();
          participantStore.updateParticipantMute(participant.identity, true);
          updateParticipants(room);
        }
      });

      room.on(RoomEvent.TrackUnmuted, (publication, participant: LKParticipant) => {
        if (publication.source === Track.Source.Microphone) {
          const participantStore = useParticipantStore.getState();
          participantStore.updateParticipantMute(participant.identity, false);
          updateParticipants(room);
        }
      });

      room.on(RoomEvent.ActiveSpeakersChanged, (speakers: LKParticipant[]) => {
        const participantStore = useParticipantStore.getState();
        // Get current speaking state
        const allParticipants = participantStore.getParticipantsArray();

        // Normalize LiveKit identities to match store participants
        const speakerIdentities = new Set(speakers.map((s) => normalizeIdentity(s.identity)));

        // Only update participants whose speaking status changed
        allParticipants.forEach((p) => {
          const shouldBeSpeaking = speakerIdentities.has(p.identity);
          if (p.isSpeaking !== shouldBeSpeaking) {
            participantStore.updateParticipantSpeaking(p.identity, shouldBeSpeaking);
          }
        });

        updateParticipants(room);
      });

      // Track first audio subscription for latency measurement (t3)
      let firstAudioTracked = false;
      room.on(RoomEvent.TrackSubscribed, (track) => {
        if (!firstAudioTracked && track.kind === 'audio') {
          const latencyStore = useLatencyStore.getState();
          latencyStore.recordFirstAudioSubscribed();
          latencyStore.finalizeLatency();
          firstAudioTracked = true;
        }
      });

      // Connect to room
      const wsUrl = import.meta.env.VITE_LIVEKIT_WS_URL;
      if (!wsUrl || typeof wsUrl !== 'string' || wsUrl.trim() === '') {
        throw new Error('LiveKit WebSocket URL is not configured');
      }

      await room.connect(wsUrl, token);

      // Record room connected timestamp (t2)
      latencyStore.recordRoomConnected();

      // Enable local microphone
      await room.localParticipant.setMicrophoneEnabled(true);

      // Update participants
      updateParticipants(room);

      // Apply current volume setting to room
      const { volume } = get();
      applyVolumeToRoom(room, volume);

      set({
        room,
        isConnected: true,
        isConnecting: false,
        error: null,
        reconnectAttempts: 0,
        isReconnecting: false,
      });

      console.info('Connected to room:', roomName);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to connect to room';
      console.error('Connection error:', errorMessage);

      set({
        isConnecting: false,
        error: errorMessage,
      });

      // Schedule reconnection on failure
      get().scheduleReconnect();
    }
  },

  /**
   * Disconnect from current room
   */
  disconnect: () => {
    const { room, reconnectTimeoutId } = get();

    if (room) {
      room.removeAllListeners();
      room.disconnect();
    }

    // Clear any pending reconnect timeout
    if (reconnectTimeoutId) {
      clearTimeout(reconnectTimeoutId);
    }

    // Clear participant store
    const participantStore = useParticipantStore.getState();
    participantStore.clearParticipants();

    set({
      room: null,
      roomName: null,
      isConnected: false,
      isConnecting: false,
      error: null,
      connectionQuality: 'unknown',
      reconnectAttempts: 0,
      isReconnecting: false,
      reconnectTimeoutId: null,
    });

    console.info('Disconnected from room');
  },

  /**
   * Set playback volume (0-100)
   */
  setVolume: (volume: number) => {
    // Clamp volume
    const clampedVolume = Math.max(0, Math.min(100, volume));

    // Persist to localStorage
    persistVolume(clampedVolume);

    // Update state
    set({ volume: clampedVolume });

    // Apply to current room
    const { room } = get();
    if (room) {
      applyVolumeToRoom(room, clampedVolume);
    }
  },

  /**
   * Toggle local microphone mute
   */
  toggleMute: async () => {
    const { room } = get();
    if (!room) return;

    const targetEnabled = !room.localParticipant.isMicrophoneEnabled;

    try {
      await room.localParticipant.setMicrophoneEnabled(targetEnabled);

      // Re-read the room and microphone state after the operation completes
      const latestRoom = get().room;
      if (!latestRoom) {
        return;
      }

      const isEnabled = latestRoom.localParticipant.isMicrophoneEnabled;
      set({ isLocalMuted: !isEnabled });
    } catch (error) {
      console.error('Failed to toggle microphone mute', error);
      set({ error: 'Unable to toggle microphone' });
    }
  },

  /**
   * Set connection quality
   */
  setConnectionQuality: (quality: ConnectionQuality) => {
    set({ connectionQuality: quality });
  },

  /**
   * Set error message
   */
  setError: (error: string | null) => {
    set({ error });
  },

  /**
   * Schedule reconnection with exponential backoff
   */
  scheduleReconnect: () => {
    const state = get();

    // Check if we've exceeded max attempts
    if (state.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      set({
        error: 'Failed to reconnect after multiple attempts. Please try again manually.',
        isReconnecting: false,
      });
      console.error('Max reconnection attempts reached');
      return;
    }

    // Calculate backoff delay with exponential increase
    const delay = Math.min(
      INITIAL_RECONNECT_DELAY * Math.pow(2, state.reconnectAttempts),
      MAX_RECONNECT_DELAY
    );

    set({
      isReconnecting: true,
      reconnectAttempts: state.reconnectAttempts + 1,
    });

    console.info(`Scheduling reconnection attempt ${state.reconnectAttempts + 1} in ${delay}ms`);

    const timeoutId = setTimeout(() => {
      set({ reconnectTimeoutId: null });
      const currentState = get();

      // Only reconnect if still disconnected and have room info
      if (!currentState.isConnected && currentState.roomName) {
        console.info(`Attempting reconnection (attempt ${currentState.reconnectAttempts})`);
        currentState.connect(currentState.roomName, currentState.sceneId, currentState.eventId);
      }
    }, delay);

    set({ reconnectTimeoutId: timeoutId });
  },
}));

/**
 * Individual streaming state selectors - use primitives to avoid infinite loops
 */
export const useStreamingIsConnected = () => useStreamingStore((state) => state.isConnected);
export const useStreamingIsConnecting = () => useStreamingStore((state) => state.isConnecting);
export const useStreamingRoomName = () => useStreamingStore((state) => state.roomName);
export const useStreamingError = () => useStreamingStore((state) => state.error);
export const useStreamingConnectionQuality = () =>
  useStreamingStore((state) => state.connectionQuality);
export const useStreamingVolume = () => useStreamingStore((state) => state.volume);
export const useStreamingIsLocalMuted = () => useStreamingStore((state) => state.isLocalMuted);
export const useStreamingSetVolume = () => useStreamingStore((state) => state.setVolume);
export const useStreamingToggleMute = () => useStreamingStore((state) => state.toggleMute);
export const useStreamingConnect = () => useStreamingStore((state) => state.connect);
export const useStreamingDisconnect = () => useStreamingStore((state) => state.disconnect);

/**
 * @deprecated Use individual selectors instead (useStreamingIsConnected, etc.)
 * Kept for backwards compatibility - will cause re-renders on any state change
 */
export function useStreamingConnection() {
  const isConnected = useStreamingIsConnected();
  const isConnecting = useStreamingIsConnecting();
  const roomName = useStreamingRoomName();
  const error = useStreamingError();
  const connectionQuality = useStreamingConnectionQuality();
  return { isConnected, isConnecting, roomName, error, connectionQuality };
}

/**
 * @deprecated Use individual selectors instead (useStreamingVolume, etc.)
 */
export function useStreamingAudio() {
  const volume = useStreamingVolume();
  const isLocalMuted = useStreamingIsLocalMuted();
  const setVolume = useStreamingSetVolume();
  const toggleMute = useStreamingToggleMute();
  return { volume, isLocalMuted, setVolume, toggleMute };
}

/**
 * @deprecated Use individual selectors instead (useStreamingConnect, etc.)
 */
export function useStreamingActions() {
  const connect = useStreamingConnect();
  const disconnect = useStreamingDisconnect();
  return { connect, disconnect };
}
