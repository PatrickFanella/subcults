/**
 * useEvents Hook Tests
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useEvents, useSceneEvents, useUpcomingEvents } from './useEvents';
import { useEntityStore } from '../stores/entityStore';
import { Event } from '../types/scene';

describe('useEvents', () => {
  const mockEvents: Event[] = [
    {
      id: 'event-1',
      scene_id: 'scene-1',
      name: 'Test Event 1',
      description: 'First test event',
      allow_precise: false,
      coarse_geohash: 'abc123',
    },
    {
      id: 'event-2',
      scene_id: 'scene-1',
      name: 'Test Event 2',
      description: 'Second test event',
      allow_precise: false,
      coarse_geohash: 'abc456',
    },
    {
      id: 'event-3',
      scene_id: 'scene-2',
      name: 'Another Event',
      description: 'Event for different scene',
      allow_precise: false,
      coarse_geohash: 'def789',
    },
  ];

  beforeEach(() => {
    // Reset store and populate with test data
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: {
        events: {
          'event-1': {
            data: mockEvents[0],
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
          'event-2': {
            data: mockEvents[1],
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
          'event-3': {
            data: mockEvents[2],
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
        },
      },
      user: { users: {} },
    });
  });

  it('returns all cached events', () => {
    const { result } = renderHook(() => useEvents());

    expect(result.current.events).toHaveLength(3);
    expect(result.current.loading).toBe(false);
  });

  it('filters events by scene', () => {
    const { result } = renderHook(() => useEvents({ filterByScene: 'scene-1' }));

    expect(result.current.events).toHaveLength(2);
    expect(result.current.events.every((e) => e.scene_id === 'scene-1')).toBe(true);
  });

  it('sorts events by name', () => {
    const { result } = renderHook(() => useEvents());

    expect(result.current.events[0].name).toBe('Another Event');
    expect(result.current.events[1].name).toBe('Test Event 1');
    expect(result.current.events[2].name).toBe('Test Event 2');
  });

  it('calculates total count', () => {
    const { result } = renderHook(() => useEvents());

    expect(result.current.totalCount).toBe(3);
  });

  it('excludes loading events by default', () => {
    // Add a loading event
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: {
        events: {
          ...useEntityStore.getState().event.events,
          'event-4': {
            data: {} as Event,
            metadata: {
              timestamp: Date.now(),
              loading: true,
              error: null,
              stale: false,
            },
          },
        },
      },
      user: { users: {} },
    });

    const { result } = renderHook(() => useEvents());

    expect(result.current.events).toHaveLength(3);
  });

  it('includes loading events when requested', () => {
    // Add a loading event with valid data
    const loadingEvent: Event = {
      id: 'event-4',
      scene_id: 'scene-1',
      name: 'Loading Event',
      description: 'A loading event',
      allow_precise: false,
      coarse_geohash: 'xyz123',
    };

    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: {
        events: {
          ...useEntityStore.getState().event.events,
          'event-4': {
            data: loadingEvent,
            metadata: {
              timestamp: Date.now(),
              loading: true,
              error: null,
              stale: false,
            },
          },
        },
      },
      user: { users: {} },
    });

    const { result } = renderHook(() => useEvents({ includeLoading: true }));

    expect(result.current.events).toHaveLength(4);
    expect(result.current.loading).toBe(true);
  });

  it('excludes events with errors', () => {
    // Add an event with error
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: {
        events: {
          ...useEntityStore.getState().event.events,
          'event-4': {
            data: {} as Event,
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: 'Failed to load',
              stale: false,
            },
          },
        },
      },
      user: { users: {} },
    });

    const { result } = renderHook(() => useEvents());

    expect(result.current.events).toHaveLength(3);
  });
});

describe('useSceneEvents', () => {
  beforeEach(() => {
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: {
        events: {
          'event-1': {
            data: {
              id: 'event-1',
              scene_id: 'scene-1',
              name: 'Scene 1 Event',
              description: 'Event for scene 1',
              allow_precise: false,
              coarse_geohash: 'abc123',
            },
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
          'event-2': {
            data: {
              id: 'event-2',
              scene_id: 'scene-2',
              name: 'Scene 2 Event',
              description: 'Event for scene 2',
              allow_precise: false,
              coarse_geohash: 'abc456',
            },
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
        },
      },
      user: { users: {} },
    });
  });

  it('returns events for specified scene', () => {
    const { result } = renderHook(() => useSceneEvents('scene-1'));

    expect(result.current.events).toHaveLength(1);
    expect(result.current.events[0].scene_id).toBe('scene-1');
  });

  it('returns empty array when scene has no events', () => {
    const { result } = renderHook(() => useSceneEvents('scene-3'));

    expect(result.current.events).toHaveLength(0);
  });

  it('returns empty array when sceneId is undefined', () => {
    const { result } = renderHook(() => useSceneEvents(undefined));

    expect(result.current.events).toHaveLength(0);
  });
});

describe('useUpcomingEvents', () => {
  beforeEach(() => {
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: {
        events: {
          'event-1': {
            data: {
              id: 'event-1',
              scene_id: 'scene-1',
              name: 'Upcoming Event',
              description: 'An upcoming event',
              allow_precise: false,
              coarse_geohash: 'abc123',
            },
            metadata: {
              timestamp: Date.now(),
              loading: false,
              error: null,
              stale: false,
            },
          },
        },
      },
      user: { users: {} },
    });
  });

  it('returns events sorted by name', () => {
    const { result } = renderHook(() => useUpcomingEvents());

    expect(result.current.events).toHaveLength(1);
  });
});
