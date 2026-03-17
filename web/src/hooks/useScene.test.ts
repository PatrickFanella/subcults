/**
 * useScene Hook Tests
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { useScene } from './useScene';
import { useEntityStore, TTL_CONFIG } from '../stores/entityStore';
import { Scene } from '../types/scene';
import * as apiClientModule from '../lib/api-client';

// Mock API client
vi.mock('../lib/api-client', () => ({
  apiClient: {
    get: vi.fn(),
  },
}));

describe('useScene', () => {
  const mockScene: Scene = {
    id: 'scene-1',
    name: 'Test Scene',
    description: 'A test scene',
    allow_precise: false,
    coarse_geohash: 'abc123',
    visibility: 'public',
  };

  beforeEach(() => {
    // Reset store
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: { events: {} },
      user: { users: {} },
    });

    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns null when id is undefined', () => {
    const { result } = renderHook(() => useScene(undefined));

    expect(result.current.scene).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it('fetches scene if not in cache', async () => {
    vi.spyOn(apiClientModule.apiClient, 'get').mockResolvedValue(mockScene);

    const { result } = renderHook(() => useScene('scene-1'));

    // Initially loading
    expect(result.current.loading).toBe(true);

    // Wait for fetch to complete
    await waitFor(() => {
      expect(result.current.scene).toEqual(mockScene);
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
    expect(apiClientModule.apiClient.get).toHaveBeenCalledWith('/scenes/scene-1');
  });

  it('returns cached scene without refetching', async () => {
    // Pre-populate cache
    useEntityStore.getState().setScene(mockScene);

    const { result } = renderHook(() => useScene('scene-1'));

    expect(result.current.scene).toEqual(mockScene);
    expect(result.current.loading).toBe(false);
    expect(apiClientModule.apiClient.get).not.toHaveBeenCalled();
  });

  it('refetches stale scene in background', async () => {
    // Pre-populate with stale data
    const staleTimestamp = Date.now() - TTL_CONFIG.DEFAULT - 1000;
    useEntityStore.setState({
      scene: {
        scenes: {
          'scene-1': {
            data: mockScene,
            metadata: {
              timestamp: staleTimestamp,
              loading: false,
              error: null,
              stale: false,
            },
          },
        },
        optimisticUpdates: {},
      },
      event: { events: {} },
      user: { users: {} },
    });

    vi.spyOn(apiClientModule.apiClient, 'get').mockResolvedValue(mockScene);

    const { result } = renderHook(() => useScene('scene-1'));

    // Returns cached data immediately (stale-while-revalidate)
    expect(result.current.scene).toEqual(mockScene);

    // Triggers background fetch
    await waitFor(() => {
      expect(apiClientModule.apiClient.get).toHaveBeenCalledWith('/scenes/scene-1');
    });
  });

  it('provides refetch function', async () => {
    useEntityStore.getState().setScene(mockScene);
    vi.spyOn(apiClientModule.apiClient, 'get').mockResolvedValue(mockScene);

    const { result } = renderHook(() => useScene('scene-1'));

    await act(async () => {
      await result.current.refetch();
    });

    expect(apiClientModule.apiClient.get).toHaveBeenCalledWith('/scenes/scene-1');
  });

  it('exposes error state', async () => {
    const error = new Error('Network error');
    vi.spyOn(apiClientModule.apiClient, 'get').mockRejectedValue(error);

    const { result } = renderHook(() => useScene('scene-1'));

    await waitFor(() => {
      expect(result.current.error).toBe('Network error');
    });

    expect(result.current.scene).toBeNull();
    expect(result.current.loading).toBe(false);
  });
});
