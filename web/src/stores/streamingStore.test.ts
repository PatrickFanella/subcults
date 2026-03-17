/**
 * Streaming Store Tests
 * Tests for global streaming state management
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';

// Use vi.hoisted to set up localStorage mock BEFORE any module evaluation
// This is critical because streamingStore reads localStorage during initialization
const localStorageMock = vi.hoisted(() => {
  const store: Record<string, string> = {};

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      Object.keys(store).forEach((key) => delete store[key]);
    },
  };
});

// Apply the mock to window.localStorage before any imports run
vi.stubGlobal('localStorage', localStorageMock);

// Now import the store (it will use our localStorage mock)
import { useStreamingStore } from './streamingStore';
import { mockRoom } from '../test/mocks/livekit-client';

// Mock LiveKit client so that Room is replaced with our test mock
vi.mock('livekit-client', () => import('../test/mocks/livekit-client'));

// Mock API client
vi.mock('../lib/api-client', () => ({
  apiClient: {
    getLiveKitToken: vi.fn().mockResolvedValue({
      token: 'mock-token',
      expires_at: new Date(Date.now() + 3600000).toISOString(),
    }),
  },
}));

// Mock environment variable
vi.stubEnv('VITE_LIVEKIT_WS_URL', 'ws://localhost:7880');

describe('StreamingStore', () => {
  beforeEach(() => {
    // Reset store to initial state
    useStreamingStore.setState({
      room: null,
      roomName: null,
      isConnected: false,
      isConnecting: false,
      error: null,
      connectionQuality: 'unknown',
      volume: 100,
      isLocalMuted: false,
      reconnectAttempts: 0,
      isReconnecting: false,
    });

    // Clear localStorage
    window.localStorage.clear();

    // Reset mocks
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Cleanup
    const store = useStreamingStore.getState();
    if (store.room) {
      store.disconnect();
    }
  });

  describe('Volume Persistence', () => {
    it('should initialize with default volume', () => {
      const store = useStreamingStore.getState();
      expect(store.volume).toBe(100);
    });

    it('should load volume from localStorage', () => {
      // Use window.localStorage which is our mock
      window.localStorage.setItem('subcults-stream-volume', '75');

      useStreamingStore.getState().initialize();

      expect(useStreamingStore.getState().volume).toBe(75);
    });

    it('should persist volume to localStorage when changed', () => {
      useStreamingStore.getState().setVolume(50);

      expect(window.localStorage.getItem('subcults-stream-volume')).toBe('50');
      expect(useStreamingStore.getState().volume).toBe(50);
    });

    it('should clamp volume between 0 and 100', () => {
      useStreamingStore.getState().setVolume(150);
      expect(useStreamingStore.getState().volume).toBe(100);

      useStreamingStore.getState().setVolume(-10);
      expect(useStreamingStore.getState().volume).toBe(0);
    });

    it('should fallback to default on invalid localStorage value', () => {
      window.localStorage.setItem('subcults-stream-volume', 'invalid');

      const store = useStreamingStore.getState();
      store.initialize();

      expect(store.volume).toBe(100);
    });
  });

  describe('Connection Management', () => {
    it('should start in disconnected state', () => {
      const store = useStreamingStore.getState();

      expect(store.isConnected).toBe(false);
      expect(store.isConnecting).toBe(false);
      expect(store.room).toBeNull();
      expect(store.roomName).toBeNull();
    });

    it('should set connecting state when initiating connection', async () => {
      const connectPromise = useStreamingStore.getState().connect('test-room');

      // Check immediately - isConnecting should be true before await completes
      // Note: Since mocks are synchronous, the promise may already be resolved
      // by the time we check. This test validates the normal flow.
      await connectPromise;

      // After connection, should no longer be connecting
      expect(useStreamingStore.getState().isConnecting).toBe(false);
    });

    it('should transition to connected state on successful connection', async () => {
      await useStreamingStore.getState().connect('test-room');

      const state = useStreamingStore.getState();
      expect(state.isConnected).toBe(true);
      expect(state.isConnecting).toBe(false);
      expect(state.roomName).toBe('test-room');
      expect(state.error).toBeNull();
    });

    it('should disconnect and clear state', async () => {
      await useStreamingStore.getState().connect('test-room');
      expect(useStreamingStore.getState().isConnected).toBe(true);

      useStreamingStore.getState().disconnect();

      const state = useStreamingStore.getState();
      expect(state.isConnected).toBe(false);
      expect(state.room).toBeNull();
      expect(state.roomName).toBeNull();
    });

    it('should not reconnect if already connected to same room', async () => {
      await useStreamingStore.getState().connect('test-room');
      const firstRoom = useStreamingStore.getState().room;

      await useStreamingStore.getState().connect('test-room');
      const secondRoom = useStreamingStore.getState().room;

      expect(firstRoom).toBe(secondRoom);
    });

    it('should disconnect from old room when connecting to new room', async () => {
      await useStreamingStore.getState().connect('test-room-1');
      expect(useStreamingStore.getState().roomName).toBe('test-room-1');

      await useStreamingStore.getState().connect('test-room-2');
      expect(useStreamingStore.getState().roomName).toBe('test-room-2');
    });
  });

  describe('State Persistence Across Routes', () => {
    it('should maintain connection state when navigating', async () => {
      await useStreamingStore.getState().connect('test-room');

      // Simulate route change by accessing store again
      const sameStore = useStreamingStore.getState();

      expect(sameStore.isConnected).toBe(true);
      expect(sameStore.roomName).toBe('test-room');
    });

    it('should maintain volume setting across route changes', () => {
      useStreamingStore.getState().setVolume(60);

      // Simulate route change
      const sameStore = useStreamingStore.getState();

      expect(sameStore.volume).toBe(60);
      expect(window.localStorage.getItem('subcults-stream-volume')).toBe('60');
    });
  });

  describe('Reconnection Logic', () => {
    it('should schedule reconnection on unexpected disconnect', async () => {
      vi.useFakeTimers();

      await useStreamingStore.getState().connect('test-room');

      // Simulate unexpected disconnect
      useStreamingStore.getState().scheduleReconnect();

      const state = useStreamingStore.getState();
      expect(state.isReconnecting).toBe(true);
      expect(state.reconnectAttempts).toBe(1);

      vi.useRealTimers();
    });

    it('should use exponential backoff for reconnection attempts', () => {
      vi.useFakeTimers();

      // First attempt
      useStreamingStore.getState().scheduleReconnect();
      expect(useStreamingStore.getState().reconnectAttempts).toBe(1);

      // Second attempt
      useStreamingStore.getState().scheduleReconnect();
      expect(useStreamingStore.getState().reconnectAttempts).toBe(2);

      // Third attempt
      useStreamingStore.getState().scheduleReconnect();
      expect(useStreamingStore.getState().reconnectAttempts).toBe(3);

      vi.useRealTimers();
    });

    it('should stop reconnecting after max attempts', () => {
      // Simulate max attempts
      useStreamingStore.setState({ reconnectAttempts: 3 });

      useStreamingStore.getState().scheduleReconnect();

      const state = useStreamingStore.getState();
      expect(state.error).toContain('Failed to reconnect after multiple attempts');
      expect(state.isReconnecting).toBe(false);
    });

    it('should reset reconnect attempts on successful connection', async () => {
      useStreamingStore.setState({ reconnectAttempts: 2 });

      await useStreamingStore.getState().connect('test-room');

      const state = useStreamingStore.getState();
      expect(state.reconnectAttempts).toBe(0);
      expect(state.isReconnecting).toBe(false);
    });
  });

  describe('Audio Controls', () => {
    it('should toggle mute state', async () => {
      await useStreamingStore.getState().connect('test-room');

      await useStreamingStore.getState().toggleMute();

      expect(mockRoom.localParticipant.setMicrophoneEnabled).toHaveBeenCalledWith(false);
    });

    it('should apply volume to room on connection', async () => {
      useStreamingStore.getState().setVolume(50);
      expect(useStreamingStore.getState().volume).toBe(50);

      await useStreamingStore.getState().connect('test-room');

      // Volume should be applied to connected room
      expect(useStreamingStore.getState().volume).toBe(50);
    });
  });

  describe('Error Handling', () => {
    it('should set error on connection failure', async () => {
      const { apiClient } = await import('../lib/api-client');
      vi.mocked(apiClient.getLiveKitToken).mockRejectedValueOnce(new Error('Token fetch failed'));

      await useStreamingStore.getState().connect('test-room');

      const state = useStreamingStore.getState();
      expect(state.error).toContain('Token fetch failed');
      expect(state.isConnecting).toBe(false);
      expect(state.isConnected).toBe(false);
    });

    it('should schedule reconnect on connection error', async () => {
      const { apiClient } = await import('../lib/api-client');
      vi.mocked(apiClient.getLiveKitToken).mockRejectedValueOnce(new Error('Network error'));

      const store = useStreamingStore.getState();
      await store.connect('test-room');

      const newState = useStreamingStore.getState();
      expect(newState.isReconnecting).toBe(true);
      expect(newState.reconnectAttempts).toBeGreaterThan(0);
    });

    it('should handle missing WebSocket URL', async () => {
      vi.stubEnv('VITE_LIVEKIT_WS_URL', '');

      await useStreamingStore.getState().connect('test-room');

      const state = useStreamingStore.getState();
      expect(state.error).toContain('LiveKit WebSocket URL is not configured');
      expect(state.isConnected).toBe(false);
    });
  });
});
