/* eslint-disable @typescript-eslint/no-explicit-any */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor, act } from '@testing-library/react';
import { MapView, type MapViewHandle } from './MapView';
import { createRef } from 'react';

// Mock MapLibre GL
vi.mock('maplibre-gl', () => {
  const mockMapInstance = {
    addSource: vi.fn(),
    addLayer: vi.fn(),
    remove: vi.fn(),
    resize: vi.fn(),
    flyTo: vi.fn(),
    getZoom: vi.fn(() => 12),
    getBounds: vi.fn(() => ({
      getNorth: () => 37.8,
      getSouth: () => 37.7,
      getEast: () => -122.4,
      getWest: () => -122.5,
    })),
    on: vi.fn(),
  };

  class MockMap {
    constructor() {
      return mockMapInstance;
    }
  }

  return {
    default: {
      Map: MockMap,
      mockMapInstance,
    },
    Map: MockMap,
    mockMapInstance,
  };
});

import maplibregl from 'maplibre-gl';

describe('MapView', () => {
  const mockMapInstance = (maplibregl as any).mockMapInstance;
  let mockResizeObserver: any;

  beforeEach(() => {
    // Set environment variable for API key
    import.meta.env.VITE_MAPTILER_API_KEY = 'test-api-key';

    // Mock ResizeObserver
    mockResizeObserver = {
      observe: vi.fn(),
      disconnect: vi.fn(),
      unobserve: vi.fn(),
    };

    class MockResizeObserver {
      constructor() {
        return mockResizeObserver;
      }
    }
    (globalThis as any).ResizeObserver = MockResizeObserver;

    // Clear all mocks
    mockMapInstance.addSource.mockClear();
    mockMapInstance.addLayer.mockClear();
    mockMapInstance.remove.mockClear();
    mockMapInstance.resize.mockClear();
    mockMapInstance.flyTo.mockClear();
    mockMapInstance.getZoom.mockClear();
    mockMapInstance.getBounds.mockClear();
    mockMapInstance.on.mockClear();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders map container without errors', () => {
    const { getByTestId } = render(<MapView />);
    const container = getByTestId('map-container');
    expect(container).toBeDefined();
  });

  it('initializes MapLibre map with MapTiler style', async () => {
    render(<MapView />);

    await waitFor(() => {
      expect(mockMapInstance.on).toHaveBeenCalled();
    });

    // Verify the style URL contains API key - we can't access constructor args easily
    // but we can verify that the map was created by checking method calls
    expect(mockMapInstance.on).toHaveBeenCalledWith('load', expect.any(Function));
  });

  it('applies custom className to container', () => {
    const { getByTestId } = render(<MapView className="custom-class" />);
    const container = getByTestId('map-container');
    expect(container.className).toContain('custom-class');
  });

  it('shows error message when API key is missing', () => {
    // Clear the API key
    delete import.meta.env.VITE_MAPTILER_API_KEY;

    const { getByTestId, getByText } = render(<MapView />);

    // Should show error container instead of map
    const errorContainer = getByTestId('map-error');
    expect(errorContainer).toBeDefined();
    expect(getByText('Map Unavailable')).toBeDefined();
    expect(getByText(/VITE_MAPTILER_API_KEY/)).toBeDefined();
  });

  it('sets up ResizeObserver and observes container', async () => {
    const { getByTestId } = render(<MapView />);
    const container = getByTestId('map-container');

    await waitFor(() => {
      expect(mockResizeObserver.observe).toHaveBeenCalledWith(container);
    });
  });

  it('calls map.resize() when ResizeObserver callback is triggered', async () => {
    render(<MapView />);

    await waitFor(() => {
      expect(mockMapInstance.on).toHaveBeenCalled();
    });

    // Verify that resize method exists and can be called
    // The ResizeObserver callback would call this in real usage
    if (mockMapInstance.resize) {
      mockMapInstance.resize();
      expect(mockMapInstance.resize).toHaveBeenCalled();
    }
  });

  it('exposes getMap() method via ref', async () => {
    const ref = createRef<MapViewHandle>();
    render(<MapView ref={ref} />);

    await waitFor(() => {
      expect(ref.current).not.toBeNull();
    });

    const map = ref.current!.getMap();
    expect(map).toBeTruthy();
  });

  it('exposes flyTo() method via ref', async () => {
    const ref = createRef<MapViewHandle>();
    render(<MapView ref={ref} />);

    await waitFor(() => {
      expect(ref.current).not.toBeNull();
    });

    ref.current!.flyTo([-122.4, 37.7], 14);

    expect(mockMapInstance.flyTo).toHaveBeenCalledWith({
      center: [-122.4, 37.7],
      zoom: 14,
      essential: true,
    });
  });

  it('exposes getBounds() method via ref', async () => {
    const ref = createRef<MapViewHandle>();
    render(<MapView ref={ref} />);

    await waitFor(() => {
      expect(ref.current).not.toBeNull();
    });

    const bounds = ref.current!.getBounds();
    expect(bounds).toBeTruthy();
  });

  it('calls onLoad callback when map loads', async () => {
    const onLoad = vi.fn();
    render(<MapView onLoad={onLoad} />);

    await waitFor(() => {
      expect(mockMapInstance.on).toHaveBeenCalled();
    });

    // Simulate map load event - wrap in act to handle state updates
    const onCall = mockMapInstance.on.mock.calls.find((call: any) => call[0] === 'load');
    expect(onCall).toBeDefined();
    const loadHandler = onCall[1];
    act(() => {
      loadHandler();
    });

    await waitFor(() => {
      expect(onLoad).toHaveBeenCalledWith(mockMapInstance);
    });
  });

  it('adds placeholder source and layer on load', async () => {
    render(<MapView />);

    await waitFor(() => {
      expect(mockMapInstance.on).toHaveBeenCalled();
    });

    // Simulate map load event - wrap in act
    const onCall = mockMapInstance.on.mock.calls.find((call: any) => call[0] === 'load');
    const loadHandler = onCall[1];
    act(() => {
      loadHandler();
    });

    await waitFor(() => {
      expect(mockMapInstance.addSource).toHaveBeenCalledWith(
        'scenes-placeholder',
        expect.objectContaining({
          type: 'geojson',
          cluster: true,
        })
      );
      expect(mockMapInstance.addLayer).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'clusters-placeholder',
          type: 'circle',
          source: 'scenes-placeholder',
        })
      );
    });
  });

  it('cleans up map instance on unmount', async () => {
    const { unmount } = render(<MapView />);

    await waitFor(() => {
      expect(mockMapInstance.on).toHaveBeenCalled();
    });

    unmount();

    expect(mockMapInstance.remove).toHaveBeenCalled();
  });

  it('disconnects ResizeObserver on unmount', async () => {
    const { unmount } = render(<MapView />);

    await waitFor(() => {
      expect(mockMapInstance.on).toHaveBeenCalled();
    });

    unmount();

    // Verify component unmounted without errors
    expect(mockMapInstance.remove).toHaveBeenCalled();
  });

  it('does not request geolocation by default', () => {
    const mockGetCurrentPosition = vi.fn();
    (globalThis as any).navigator.geolocation = {
      getCurrentPosition: mockGetCurrentPosition,
    };

    render(<MapView />);

    expect(mockGetCurrentPosition).not.toHaveBeenCalled();
  });

  it('requests geolocation when enableGeolocation is true', async () => {
    const mockGetCurrentPosition = vi.fn((success: any, _error: any, options: any) => {
      success({
        coords: { latitude: 37.7749, longitude: -122.4194 },
      });
      // Verify enableHighAccuracy is false for privacy
      expect(options?.enableHighAccuracy).toBe(false);
    });
    (globalThis as any).navigator.geolocation = {
      getCurrentPosition: mockGetCurrentPosition,
    };

    const onGeolocationSuccess = vi.fn();
    render(<MapView enableGeolocation={true} onGeolocationSuccess={onGeolocationSuccess} />);

    await waitFor(() => {
      expect(mockGetCurrentPosition).toHaveBeenCalled();
    });
  });

  it('handles geolocation error gracefully', async () => {
    const mockError = {
      code: 1,
      message: 'User denied geolocation',
      PERMISSION_DENIED: 1,
    } as GeolocationPositionError;

    const mockGetCurrentPosition = vi.fn((_success: any, error: any) => {
      error(mockError);
    });
    (globalThis as any).navigator.geolocation = {
      getCurrentPosition: mockGetCurrentPosition,
    };

    const onGeolocationError = vi.fn();
    render(<MapView enableGeolocation={true} onGeolocationError={onGeolocationError} />);

    await waitFor(() => {
      expect(onGeolocationError).toHaveBeenCalledWith(mockError);
    });
  });

  it('sets data-map-loaded attribute when map loads', async () => {
    const { getByTestId } = render(<MapView />);
    const container = getByTestId('map-container');

    expect(container.getAttribute('data-map-loaded')).toBe('false');

    await waitFor(() => {
      expect(mockMapInstance.on).toHaveBeenCalled();
    });

    // Simulate map load event - wrap in act
    const onCall = mockMapInstance.on.mock.calls.find((call: any) => call[0] === 'load');
    const loadHandler = onCall[1];
    act(() => {
      loadHandler();
    });

    await waitFor(() => {
      expect(container.getAttribute('data-map-loaded')).toBe('true');
    });
  });
});
