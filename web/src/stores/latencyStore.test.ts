/**
 * Latency Store Tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useLatencyStore } from './latencyStore';

describe('latencyStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useLatencyStore.setState({
      currentLatency: {
        joinClicked: null,
        tokenReceived: null,
        roomConnected: null,
        firstAudioSubscribed: null,
      },
      lastLatency: null,
    });
    
    // Mock performance.now() for consistent testing
    vi.spyOn(performance, 'now').mockReturnValue(1000);
  });

  describe('recordJoinClicked', () => {
    it('should record join click timestamp', () => {
      const { recordJoinClicked } = useLatencyStore.getState();
      recordJoinClicked();

      const { currentLatency } = useLatencyStore.getState();
      expect(currentLatency.joinClicked).toBe(1000);
      expect(currentLatency.tokenReceived).toBeNull();
      expect(currentLatency.roomConnected).toBeNull();
      expect(currentLatency.firstAudioSubscribed).toBeNull();
    });

    it('should reset all other timestamps when recording join click', () => {
      // Set some existing values
      useLatencyStore.setState({
        currentLatency: {
          joinClicked: 500,
          tokenReceived: 600,
          roomConnected: 700,
          firstAudioSubscribed: 800,
        },
        lastLatency: null,
      });

      const { recordJoinClicked } = useLatencyStore.getState();
      recordJoinClicked();

      const { currentLatency } = useLatencyStore.getState();
      expect(currentLatency.joinClicked).toBe(1000);
      expect(currentLatency.tokenReceived).toBeNull();
      expect(currentLatency.roomConnected).toBeNull();
      expect(currentLatency.firstAudioSubscribed).toBeNull();
    });
  });

  describe('recordTokenReceived', () => {
    it('should record token received timestamp', () => {
      const store = useLatencyStore.getState();
      store.recordJoinClicked();

      vi.spyOn(performance, 'now').mockReturnValue(1200);
      store.recordTokenReceived();

      const { currentLatency } = useLatencyStore.getState();
      expect(currentLatency.tokenReceived).toBe(1200);
    });
  });

  describe('recordRoomConnected', () => {
    it('should record room connected timestamp', () => {
      const store = useLatencyStore.getState();
      store.recordJoinClicked();
      
      vi.spyOn(performance, 'now').mockReturnValue(1200);
      store.recordTokenReceived();

      vi.spyOn(performance, 'now').mockReturnValue(1500);
      store.recordRoomConnected();

      const { currentLatency } = useLatencyStore.getState();
      expect(currentLatency.roomConnected).toBe(1500);
    });
  });

  describe('recordFirstAudioSubscribed', () => {
    it('should record first audio subscribed timestamp', () => {
      const store = useLatencyStore.getState();
      store.recordJoinClicked();
      
      vi.spyOn(performance, 'now').mockReturnValue(1200);
      store.recordTokenReceived();

      vi.spyOn(performance, 'now').mockReturnValue(1500);
      store.recordRoomConnected();

      vi.spyOn(performance, 'now').mockReturnValue(1800);
      store.recordFirstAudioSubscribed();

      const { currentLatency } = useLatencyStore.getState();
      expect(currentLatency.firstAudioSubscribed).toBe(1800);
    });
  });

  describe('computeSegments', () => {
    it('should compute latency segments correctly', () => {
      const store = useLatencyStore.getState();
      
      // Simulate a complete join sequence
      vi.spyOn(performance, 'now').mockReturnValue(1000);
      store.recordJoinClicked();

      vi.spyOn(performance, 'now').mockReturnValue(1200);
      store.recordTokenReceived();

      vi.spyOn(performance, 'now').mockReturnValue(1500);
      store.recordRoomConnected();

      vi.spyOn(performance, 'now').mockReturnValue(1800);
      store.recordFirstAudioSubscribed();

      const segments = store.computeSegments();
      
      expect(segments.tokenFetch).toBe(200); // 1200 - 1000
      expect(segments.roomConnection).toBe(300); // 1500 - 1200
      expect(segments.audioSubscription).toBe(300); // 1800 - 1500
      expect(segments.total).toBe(800); // 1800 - 1000
    });

    it('should return null for missing segments', () => {
      const store = useLatencyStore.getState();
      
      // Only record join click
      store.recordJoinClicked();

      const segments = store.computeSegments();
      
      expect(segments.tokenFetch).toBeNull();
      expect(segments.roomConnection).toBeNull();
      expect(segments.audioSubscription).toBeNull();
      expect(segments.total).toBeNull();
    });

    it('should compute partial segments when some timestamps are available', () => {
      const store = useLatencyStore.getState();
      
      vi.spyOn(performance, 'now').mockReturnValue(1000);
      store.recordJoinClicked();

      vi.spyOn(performance, 'now').mockReturnValue(1200);
      store.recordTokenReceived();

      const segments = store.computeSegments();
      
      expect(segments.tokenFetch).toBe(200);
      expect(segments.roomConnection).toBeNull();
      expect(segments.audioSubscription).toBeNull();
      expect(segments.total).toBeNull();
    });
  });

  describe('resetLatency', () => {
    it('should reset current latency to empty state', () => {
      const store = useLatencyStore.getState();
      
      // Set some values
      store.recordJoinClicked();
      store.recordTokenReceived();
      
      store.resetLatency();

      const { currentLatency } = useLatencyStore.getState();
      expect(currentLatency.joinClicked).toBeNull();
      expect(currentLatency.tokenReceived).toBeNull();
      expect(currentLatency.roomConnected).toBeNull();
      expect(currentLatency.firstAudioSubscribed).toBeNull();
    });
  });

  describe('finalizeLatency', () => {
    it('should copy current latency to last latency', () => {
      const store = useLatencyStore.getState();
      
      vi.spyOn(performance, 'now').mockReturnValue(1000);
      store.recordJoinClicked();

      vi.spyOn(performance, 'now').mockReturnValue(1200);
      store.recordTokenReceived();

      store.finalizeLatency();

      const { lastLatency } = useLatencyStore.getState();
      expect(lastLatency).not.toBeNull();
      expect(lastLatency!.joinClicked).toBe(1000);
      expect(lastLatency!.tokenReceived).toBe(1200);
    });
  });
});
