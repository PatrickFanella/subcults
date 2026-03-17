/**
 * useScenes Hook Tests
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useScenes, usePublicScenes, useUserScenes } from './useScenes';
import { useEntityStore } from '../stores/entityStore';
import { Scene } from '../types/scene';

describe('useScenes', () => {
  const mockScenes: Scene[] = [
    {
      id: 'scene-1',
      name: 'Public Scene',
      description: 'A public scene',
      allow_precise: false,
      coarse_geohash: 'abc123',
      visibility: 'public',
      owner_user_id: 'user-1',
    },
    {
      id: 'scene-2',
      name: 'Private Scene',
      description: 'A private scene',
      allow_precise: false,
      coarse_geohash: 'abc456',
      visibility: 'private',
      owner_user_id: 'user-1',
    },
    {
      id: 'scene-3',
      name: 'Another Public Scene',
      description: 'Another public scene',
      allow_precise: false,
      coarse_geohash: 'def789',
      visibility: 'public',
      owner_user_id: 'user-2',
    },
  ];

  beforeEach(() => {
    // Reset store and populate with test data
    useEntityStore.setState({
      scene: {
        scenes: {
          'scene-1': {
            data: mockScenes[0],
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
          'scene-2': {
            data: mockScenes[1],
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
          'scene-3': {
            data: mockScenes[2],
            metadata: {
              timestamp: Date.now(),
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
  });

  it('returns all cached scenes', () => {
    const { result } = renderHook(() => useScenes());

    expect(result.current.scenes).toHaveLength(3);
    expect(result.current.loading).toBe(false);
  });

  it('filters scenes by owner', () => {
    const { result } = renderHook(() => useScenes({ filterByOwner: 'user-1' }));

    expect(result.current.scenes).toHaveLength(2);
    expect(result.current.scenes.every((s) => s.owner_user_id === 'user-1')).toBe(true);
  });

  it('filters scenes by visibility', () => {
    const { result } = renderHook(() => useScenes({ filterByVisibility: 'public' }));

    expect(result.current.scenes).toHaveLength(2);
    expect(result.current.scenes.every((s) => s.visibility === 'public')).toBe(true);
  });

  it('calculates active count correctly', () => {
    const { result } = renderHook(() => useScenes());

    // Active count should exclude private scenes
    expect(result.current.activeCount).toBe(2);
  });

  it('excludes loading scenes by default', () => {
    // Add a loading scene
    useEntityStore.setState({
      scene: {
        scenes: {
          ...useEntityStore.getState().scene.scenes,
          'scene-4': {
            data: {} as Scene,
            metadata: {
              timestamp: Date.now(),
              loading: true,
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

    const { result } = renderHook(() => useScenes());

    expect(result.current.scenes).toHaveLength(3);
  });

  it('includes loading scenes when requested', () => {
    // Add a loading scene with valid data
    const loadingScene: Scene = {
      id: 'scene-4',
      name: 'Loading Scene',
      description: 'A loading scene',
      allow_precise: false,
      coarse_geohash: 'xyz123',
      visibility: 'public',
    };

    useEntityStore.setState({
      scene: {
        scenes: {
          ...useEntityStore.getState().scene.scenes,
          'scene-4': {
            data: loadingScene,
            metadata: {
              timestamp: Date.now(),
              loading: true,
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

    const { result } = renderHook(() => useScenes({ includeLoading: true }));

    expect(result.current.scenes).toHaveLength(4);
    expect(result.current.loading).toBe(true);
  });

  it('excludes scenes with errors', () => {
    // Add a scene with error
    useEntityStore.setState({
      scene: {
        scenes: {
          ...useEntityStore.getState().scene.scenes,
          'scene-4': {
            data: {} as Scene,
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: 'Failed to load',
              stale: false,
            },
          },
        },
        optimisticUpdates: {},
      },
      event: { events: {} },
      user: { users: {} },
    });

    const { result } = renderHook(() => useScenes());

    expect(result.current.scenes).toHaveLength(3);
  });
});

describe('usePublicScenes', () => {
  beforeEach(() => {
    useEntityStore.setState({
      scene: {
        scenes: {
          'scene-1': {
            data: {
              id: 'scene-1',
              name: 'Public Scene',
              description: 'A public scene',
              allow_precise: false,
              coarse_geohash: 'abc123',
              visibility: 'public',
            },
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
          'scene-2': {
            data: {
              id: 'scene-2',
              name: 'Private Scene',
              description: 'A private scene',
              allow_precise: false,
              coarse_geohash: 'abc456',
              visibility: 'private',
            },
            metadata: {
              timestamp: Date.now(),
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
  });

  it('returns only public scenes', () => {
    const { result } = renderHook(() => usePublicScenes());

    expect(result.current.scenes).toHaveLength(1);
    expect(result.current.scenes[0].visibility).toBe('public');
  });
});

describe('useUserScenes', () => {
  beforeEach(() => {
    useEntityStore.setState({
      scene: {
        scenes: {
          'scene-1': {
            data: {
              id: 'scene-1',
              name: 'User 1 Scene',
              description: 'Scene owned by user 1',
              allow_precise: false,
              coarse_geohash: 'abc123',
              visibility: 'public',
              owner_user_id: 'user-1',
            },
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
          'scene-2': {
            data: {
              id: 'scene-2',
              name: 'User 2 Scene',
              description: 'Scene owned by user 2',
              allow_precise: false,
              coarse_geohash: 'abc456',
              visibility: 'public',
              owner_user_id: 'user-2',
            },
            metadata: {
              timestamp: Date.now(),
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
  });

  it('returns scenes for specified user', () => {
    const { result } = renderHook(() => useUserScenes('user-1'));

    expect(result.current.scenes).toHaveLength(1);
    expect(result.current.scenes[0].owner_user_id).toBe('user-1');
  });

  it('returns empty array when user has no scenes', () => {
    const { result } = renderHook(() => useUserScenes('user-3'));

    expect(result.current.scenes).toHaveLength(0);
  });

  it('returns empty array when userId is undefined', () => {
    const { result } = renderHook(() => useUserScenes(undefined));

    expect(result.current.scenes).toHaveLength(0);
  });
});
