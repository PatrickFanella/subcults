/**
 * Entity Store Tests
 * Tests for global entity state management with caching and TTL
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  useEntityStore,
  isStale,
  createFreshMetadata,
  setLoadingMetadata,
  setSuccessMetadata,
  setErrorMetadata,
  TTL_CONFIG,
  resetInFlightRequests,
} from './entityStore';
import type { User } from './authStore';
import { Scene, Event } from '../types/scene';
import * as apiClientModule from '../lib/api-client';

// Mock API client
vi.mock('../lib/api-client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('entityStore', () => {
  beforeEach(() => {
    // Reset store state
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: { events: {} },
      user: { users: {} },
    });

    // Clear in-flight requests
    resetInFlightRequests();

    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Metadata helpers', () => {
    it('creates fresh metadata', () => {
      const metadata = createFreshMetadata();
      expect(metadata.loading).toBe(false);
      expect(metadata.error).toBeNull();
      expect(metadata.stale).toBe(false);
      expect(metadata.timestamp).toBeGreaterThan(0);
    });

    it('sets loading state', () => {
      const metadata = createFreshMetadata();
      const loading = setLoadingMetadata(metadata);
      expect(loading.loading).toBe(true);
      expect(loading.error).toBeNull();
    });

    it('sets success state', () => {
      const metadata = setSuccessMetadata();
      expect(metadata.loading).toBe(false);
      expect(metadata.error).toBeNull();
      expect(metadata.stale).toBe(false);
    });

    it('sets error state', () => {
      const metadata = createFreshMetadata();
      const error = setErrorMetadata(metadata, 'Test error');
      expect(error.loading).toBe(false);
      expect(error.error).toBe('Test error');
    });

    it('detects stale entries based on TTL', () => {
      const fresh = createFreshMetadata();
      expect(isStale(fresh, TTL_CONFIG.DEFAULT)).toBe(false);

      const old = { ...fresh, timestamp: Date.now() - TTL_CONFIG.DEFAULT - 1000 };
      expect(isStale(old, TTL_CONFIG.DEFAULT)).toBe(true);
    });

    it('respects manual stale marking', () => {
      const metadata = { ...createFreshMetadata(), stale: true };
      expect(isStale(metadata, TTL_CONFIG.DEFAULT)).toBe(true);
    });
  });

  describe('Scene slice', () => {
    const mockScene: Scene = {
      id: 'scene-1',
      name: 'Test Scene',
      description: 'A test scene',
      allow_precise: false,
      coarse_geohash: 'abc123',
      visibility: 'public',
    };

    it('fetches and caches a scene', async () => {
      vi.spyOn(apiClientModule.apiClient, 'get').mockResolvedValue(mockScene);

      const { fetchScene } = useEntityStore.getState();
      const result = await fetchScene('scene-1');

      expect(result).toEqual(mockScene);
      expect(apiClientModule.apiClient.get).toHaveBeenCalledWith('/scenes/scene-1');

      const state = useEntityStore.getState();
      expect(state.scene.scenes['scene-1']).toBeDefined();
      expect(state.scene.scenes['scene-1'].data).toEqual(mockScene);
      expect(state.scene.scenes['scene-1'].metadata.loading).toBe(false);
      expect(state.scene.scenes['scene-1'].metadata.error).toBeNull();
    });

    it('deduplicates concurrent requests', async () => {
      vi.spyOn(apiClientModule.apiClient, 'get').mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(() => resolve(mockScene), 100)
          )
      );

      const { fetchScene } = useEntityStore.getState();
      const [result1, result2, result3] = await Promise.all([
        fetchScene('scene-1'),
        fetchScene('scene-1'),
        fetchScene('scene-1'),
      ]);

      expect(result1).toEqual(mockScene);
      expect(result2).toEqual(mockScene);
      expect(result3).toEqual(mockScene);
      // Should only call API once
      expect(apiClientModule.apiClient.get).toHaveBeenCalledTimes(1);
    });

    it('handles fetch errors', async () => {
      const error = new Error('Network error');
      vi.spyOn(apiClientModule.apiClient, 'get').mockRejectedValue(error);

      const { fetchScene } = useEntityStore.getState();
      await expect(fetchScene('scene-1')).rejects.toThrow('Network error');

      const state = useEntityStore.getState();
      expect(state.scene.scenes['scene-1'].metadata.error).toBe('Network error');
      expect(state.scene.scenes['scene-1'].metadata.loading).toBe(false);
    });

    it('sets scene directly', () => {
      const { setScene } = useEntityStore.getState();
      setScene(mockScene);

      const state = useEntityStore.getState();
      expect(state.scene.scenes['scene-1'].data).toEqual(mockScene);
      expect(state.scene.scenes['scene-1'].metadata.error).toBeNull();
    });

    it('marks scene as stale', () => {
      const { setScene, markSceneStale } = useEntityStore.getState();
      setScene(mockScene);

      markSceneStale('scene-1');

      const state = useEntityStore.getState();
      expect(state.scene.scenes['scene-1'].metadata.stale).toBe(true);
    });

    it('clears scene error', () => {
      useEntityStore.setState({
        scene: {
          scenes: {
            'scene-1': {
              data: mockScene,
              metadata: { ...createFreshMetadata(), error: 'Test error' },
            },
          },
          optimisticUpdates: {},
        },
        event: { events: {} },
        user: { users: {} },
      });

      const { clearSceneError } = useEntityStore.getState();
      clearSceneError('scene-1');

      const state = useEntityStore.getState();
      expect(state.scene.scenes['scene-1'].metadata.error).toBeNull();
    });

    it('removes scene from cache', () => {
      const { setScene, removeScene } = useEntityStore.getState();
      setScene(mockScene);

      removeScene('scene-1');

      const state = useEntityStore.getState();
      expect(state.scene.scenes['scene-1']).toBeUndefined();
    });
  });

  describe('Optimistic updates', () => {
    const mockScene: Scene = {
      id: 'scene-1',
      name: 'Test Scene',
      description: 'A test scene',
      allow_precise: false,
      coarse_geohash: 'abc123',
      visibility: 'public',
    };

    it('saves backup for optimistic update', () => {
      const { setScene, optimisticJoinScene } = useEntityStore.getState();
      setScene(mockScene);

      optimisticJoinScene('scene-1', 'user-1');

      const state = useEntityStore.getState();
      expect(state.scene.optimisticUpdates['scene-1']).toEqual(mockScene);
    });

    it('rolls back optimistic update', () => {
      const { setScene, optimisticJoinScene, rollbackSceneUpdate } = useEntityStore.getState();
      setScene(mockScene);
      optimisticJoinScene('scene-1', 'user-1');

      // Modify scene data (simulating optimistic change)
      const modifiedScene = { ...mockScene, name: 'Modified' };
      useEntityStore.setState({
        scene: {
          scenes: {
            'scene-1': {
              data: modifiedScene,
              metadata: createFreshMetadata(),
            },
          },
          optimisticUpdates: { 'scene-1': mockScene },
        },
        event: { events: {} },
        user: { users: {} },
      });

      rollbackSceneUpdate('scene-1');

      const state = useEntityStore.getState();
      expect(state.scene.scenes['scene-1'].data).toEqual(mockScene);
      expect(state.scene.optimisticUpdates['scene-1']).toBeUndefined();
    });

    it('commits optimistic update', () => {
      const { setScene, optimisticJoinScene, commitSceneUpdate } = useEntityStore.getState();
      setScene(mockScene);
      optimisticJoinScene('scene-1', 'user-1');

      commitSceneUpdate('scene-1');

      const state = useEntityStore.getState();
      expect(state.scene.optimisticUpdates['scene-1']).toBeUndefined();
    });
  });

  describe('Event slice', () => {
    const mockEvent: Event = {
      id: 'event-1',
      scene_id: 'scene-1',
      name: 'Test Event',
      description: 'A test event',
      allow_precise: false,
      coarse_geohash: 'abc123',
    };

    it('fetches and caches an event', async () => {
      vi.spyOn(apiClientModule.apiClient, 'get').mockResolvedValue(mockEvent);

      const { fetchEvent } = useEntityStore.getState();
      const result = await fetchEvent('event-1');

      expect(result).toEqual(mockEvent);
      expect(apiClientModule.apiClient.get).toHaveBeenCalledWith('/events/event-1');

      const state = useEntityStore.getState();
      expect(state.event.events['event-1']).toBeDefined();
      expect(state.event.events['event-1'].data).toEqual(mockEvent);
    });

    it('sets event directly', () => {
      const { setEvent } = useEntityStore.getState();
      setEvent(mockEvent);

      const state = useEntityStore.getState();
      expect(state.event.events['event-1'].data).toEqual(mockEvent);
    });

    it('marks event as stale', () => {
      const { setEvent, markEventStale } = useEntityStore.getState();
      setEvent(mockEvent);

      markEventStale('event-1');

      const state = useEntityStore.getState();
      expect(state.event.events['event-1'].metadata.stale).toBe(true);
    });

    it('removes event from cache', () => {
      const { setEvent, removeEvent } = useEntityStore.getState();
      setEvent(mockEvent);

      removeEvent('event-1');

      const state = useEntityStore.getState();
      expect(state.event.events['event-1']).toBeUndefined();
    });
  });

  describe('User slice', () => {
    const mockUser: User = {
      did: 'did:example:123',
      role: 'user',
    };

    it('fetches and caches a user', async () => {
      vi.spyOn(apiClientModule.apiClient, 'get').mockResolvedValue(mockUser);

      const { fetchUser } = useEntityStore.getState();
      const result = await fetchUser('did:example:123');

      expect(result).toEqual(mockUser);
      expect(apiClientModule.apiClient.get).toHaveBeenCalledWith('/users/did:example:123');

      const state = useEntityStore.getState();
      expect(state.user.users['did:example:123']).toBeDefined();
      expect(state.user.users['did:example:123'].data).toEqual(mockUser);
    });

    it('sets user directly', () => {
      const { setUser } = useEntityStore.getState();
      setUser(mockUser);

      const state = useEntityStore.getState();
      expect(state.user.users['did:example:123'].data).toEqual(mockUser);
    });

    it('marks user as stale', () => {
      const { setUser, markUserStale } = useEntityStore.getState();
      setUser(mockUser);

      markUserStale('did:example:123');

      const state = useEntityStore.getState();
      expect(state.user.users['did:example:123'].metadata.stale).toBe(true);
    });

    it('removes user from cache', () => {
      const { setUser, removeUser } = useEntityStore.getState();
      setUser(mockUser);

      removeUser('did:example:123');

      const state = useEntityStore.getState();
      expect(state.user.users['did:example:123']).toBeUndefined();
    });
  });
});
