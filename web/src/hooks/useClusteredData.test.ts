import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { useClusteredData, boundsToBox } from './useClusteredData';
import type { Scene, Event } from '../types/scene';
import type { LngLatBounds } from 'maplibre-gl';

// Mock fetch
const mockFetch = vi.fn();
globalThis.fetch = mockFetch as never;

describe('boundsToBox', () => {
  it('converts MapLibre bounds to BBox', () => {
    const mockBounds = {
      getNorth: () => 37.8,
      getSouth: () => 37.7,
      getEast: () => -122.4,
      getWest: () => -122.5,
    } as LngLatBounds;

    const result = boundsToBox(mockBounds);

    expect(result).toEqual({
      north: 37.8,
      south: 37.7,
      east: -122.4,
      west: -122.5,
    });
  });
});

describe('useClusteredData', () => {
  const mockScenes: Scene[] = [
    {
      id: 'scene1',
      name: 'Test Scene',
      allow_precise: true,
      precise_point: { lat: 37.7749, lng: -122.4194 },
      coarse_geohash: '9q8yy',
    },
  ];

  const mockEvents: Event[] = [
    {
      id: 'event1',
      scene_id: 'scene1',
      name: 'Test Event',
      allow_precise: true,
      precise_point: { lat: 37.7849, lng: -122.4094 },
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    mockFetch.mockClear();
  });

  it('initializes with empty data', () => {
    const { result } = renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    expect(result.current.data).toEqual({
      type: 'FeatureCollection',
      features: [],
    });
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it('does not fetch without bbox by default', async () => {
    renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 100));
    });

    expect(mockFetch).not.toHaveBeenCalled();
  });

  it('fetches data when updateBBox is called', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockScenes,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockEvents,
      });

    const { result } = renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    const bbox = {
      north: 37.8,
      south: 37.7,
      east: -122.4,
      west: -122.5,
    };

    act(() => {
      result.current.updateBBox(bbox);
    });

    await waitFor(
      () => {
        expect(mockFetch).toHaveBeenCalledTimes(2);
      },
      { timeout: 3000 }
    );

    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/scenes?'),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    );
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/events?'),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    );

    await waitFor(
      () => {
        expect(result.current.loading).toBe(false);
      },
      { timeout: 3000 }
    );

    expect(result.current.data.features).toHaveLength(2);
    expect(result.current.error).toBeNull();
  });

  it('builds GeoJSON from fetched scenes and events', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockScenes,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockEvents,
      });

    const { result } = renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    act(() => {
      result.current.updateBBox({
        north: 37.8,
        south: 37.7,
        east: -122.4,
        west: -122.5,
      });
    });

    await waitFor(
      () => {
        expect(result.current.data.features).toHaveLength(2);
      },
      { timeout: 3000 }
    );

    const features = result.current.data.features;
    const sceneFeature = features.find((f) => f.properties.type === 'scene');
    const eventFeature = features.find((f) => f.properties.type === 'event');

    expect(sceneFeature?.properties.id).toBe('scene1');
    expect(eventFeature?.properties.id).toBe('event1');
  });

  it('handles fetch errors gracefully', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    act(() => {
      result.current.updateBBox({
        north: 37.8,
        south: 37.7,
        east: -122.4,
        west: -122.5,
      });
    });

    await waitFor(
      () => {
        expect(result.current.error).toBe('Network error');
      },
      { timeout: 3000 }
    );

    expect(result.current.data.features).toHaveLength(0);
  });

  it('handles non-ok response status', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: 'Not Found',
    });

    const { result } = renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    act(() => {
      result.current.updateBBox({
        north: 37.8,
        south: 37.7,
        east: -122.4,
        west: -122.5,
      });
    });

    await waitFor(
      () => {
        expect(result.current.error).toContain('Failed to fetch scenes');
      },
      { timeout: 3000 }
    );
  });

  it('uses custom API URL from options', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      });

    const { result } = renderHook(() =>
      useClusteredData(null, { apiUrl: 'https://custom-api.com', debounceMs: 50 })
    );

    act(() => {
      result.current.updateBBox({
        north: 37.8,
        south: 37.7,
        east: -122.4,
        west: -122.5,
      });
    });

    await waitFor(
      () => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('https://custom-api.com/scenes'),
          expect.any(Object)
        );
      },
      { timeout: 3000 }
    );
  });

  it('supports manual refetch', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockScenes,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      });

    const { result } = renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    act(() => {
      result.current.updateBBox({
        north: 37.8,
        south: 37.7,
        east: -122.4,
        west: -122.5,
      });
    });

    await waitFor(
      () => {
        expect(mockFetch).toHaveBeenCalledTimes(2);
      },
      { timeout: 3000 }
    );

    expect(result.current.data.features).toHaveLength(0);

    // Refetch should use same bbox
    act(() => {
      result.current.refetch();
    });

    await waitFor(
      () => {
        expect(result.current.data.features).toHaveLength(1);
      },
      { timeout: 3000 }
    );
  });

  it('clears data when bbox is set to null', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockScenes,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      });

    const { result } = renderHook(() => useClusteredData(null, { debounceMs: 50 }));

    act(() => {
      result.current.updateBBox({
        north: 37.8,
        south: 37.7,
        east: -122.4,
        west: -122.5,
      });
    });

    await waitFor(
      () => {
        expect(result.current.data.features).toHaveLength(1);
      },
      { timeout: 3000 }
    );

    act(() => {
      result.current.updateBBox(null);
    });

    // Wait for debounce and data to clear
    await waitFor(
      () => {
        expect(result.current.data.features).toHaveLength(0);
      },
      { timeout: 3000 }
    );
  });
});
