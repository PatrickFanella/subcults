/* eslint-disable @typescript-eslint/no-explicit-any */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor } from '@testing-library/react';
import { ClusteredMapView } from './ClusteredMapView';

// Mock useClusteredData hook
const mockUpdateBBox = vi.fn();
const mockData = {
  type: 'FeatureCollection' as const,
  features: [
    {
      type: 'Feature' as const,
      geometry: { type: 'Point' as const, coordinates: [-122.4194, 37.7749] },
      properties: { id: 'scene1', type: 'scene' as const, name: 'Test Scene', coarse_geohash: '9q8yy' },
    },
  ],
};

vi.mock('../hooks/useClusteredData', () => ({
  useClusteredData: () => ({
    data: mockData,
    loading: false,
    error: null,
    updateBBox: mockUpdateBBox,
    refetch: vi.fn(),
  }),
  boundsToBox: (bounds: any) => ({
    north: bounds.getNorth(),
    south: bounds.getSouth(),
    east: bounds.getEast(),
    west: bounds.getWest(),
  }),
}));

// Mock MapView component
const mockMapInstance = {
  addSource: vi.fn(),
  addLayer: vi.fn(),
  removeSource: vi.fn(),
  removeLayer: vi.fn(),
  getSource: vi.fn(),
  getLayer: vi.fn(),
  on: vi.fn(),
  getBounds: vi.fn(() => ({
    getNorth: () => 37.8,
    getSouth: () => 37.7,
    getEast: () => -122.4,
    getWest: () => -122.5,
  })),
  getCanvas: vi.fn(() => ({
    style: { cursor: '' },
  })),
};

vi.mock('./MapView', () => ({
  MapView: vi.fn(({ onLoad }: any) => {
    // Simulate map loading
    if (onLoad) {
      setTimeout(() => onLoad(mockMapInstance), 0);
    }
    return <div data-testid="map-view">MapView Mock</div>;
  }),
}));

describe('ClusteredMapView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMapInstance.addSource.mockClear();
    mockMapInstance.addLayer.mockClear();
    mockMapInstance.removeSource.mockClear();
    mockMapInstance.removeLayer.mockClear();
    mockMapInstance.getSource.mockClear();
    mockMapInstance.getLayer.mockClear();
    mockMapInstance.on.mockClear();
    mockMapInstance.getBounds.mockClear();
    mockUpdateBBox.mockClear();
  });

  it('renders without errors', () => {
    const { getByTestId } = render(<ClusteredMapView />);
    expect(getByTestId('map-view')).toBeDefined();
  });

  it('removes placeholder source and layers on map load', async () => {
    mockMapInstance.getLayer.mockReturnValue({});
    mockMapInstance.getSource.mockReturnValue({});

    render(<ClusteredMapView />);

    await waitFor(() => {
      expect(mockMapInstance.removeLayer).toHaveBeenCalledWith('clusters-placeholder');
      expect(mockMapInstance.removeSource).toHaveBeenCalledWith('scenes-placeholder');
    });
  });

  it('adds cluster source with correct configuration', async () => {
    render(<ClusteredMapView />);

    await waitFor(() => {
      expect(mockMapInstance.addSource).toHaveBeenCalledWith(
        'scenes-events',
        expect.objectContaining({
          type: 'geojson',
          cluster: true,
          clusterMaxZoom: 14,
          clusterRadius: 50,
        })
      );
    });
  });

  it('adds cluster layers with style configuration', async () => {
    render(<ClusteredMapView />);

    await waitFor(() => {
      // Cluster circles layer
      expect(mockMapInstance.addLayer).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'clusters',
          type: 'circle',
          source: 'scenes-events',
        })
      );

      // Cluster count labels
      expect(mockMapInstance.addLayer).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'cluster-count',
          type: 'symbol',
          source: 'scenes-events',
        })
      );
    });
  });

  it('adds separate layers for scenes and events', async () => {
    render(<ClusteredMapView />);

    await waitFor(() => {
      // Scene layer
      expect(mockMapInstance.addLayer).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'unclustered-scene-point',
          type: 'circle',
          source: 'scenes-events',
        })
      );

      // Event layer
      expect(mockMapInstance.addLayer).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'unclustered-event-point',
          type: 'circle',
          source: 'scenes-events',
        })
      );
    });
  });

  it('sets up click handler for cluster expansion', async () => {
    render(<ClusteredMapView />);

    await waitFor(() => {
      const onCalls = mockMapInstance.on.mock.calls;
      const clusterClickHandler = onCalls.find((call: any) =>
        call[0] === 'click' && call[1] === 'clusters'
      );
      expect(clusterClickHandler).toBeDefined();
    });
  });

  it('sets up cursor change handlers for interactive elements', async () => {
    render(<ClusteredMapView />);

    await waitFor(() => {
      const onCalls = mockMapInstance.on.mock.calls;
      
      // Cluster hover
      expect(onCalls.some((call: any) =>
        call[0] === 'mouseenter' && call[1] === 'clusters'
      )).toBe(true);
      expect(onCalls.some((call: any) =>
        call[0] === 'mouseleave' && call[1] === 'clusters'
      )).toBe(true);

      // Scene point hover
      expect(onCalls.some((call: any) =>
        call[0] === 'mouseenter' && call[1] === 'unclustered-scene-point'
      )).toBe(true);
      expect(onCalls.some((call: any) =>
        call[0] === 'mouseleave' && call[1] === 'unclustered-scene-point'
      )).toBe(true);

      // Event point hover
      expect(onCalls.some((call: any) =>
        call[0] === 'mouseenter' && call[1] === 'unclustered-event-point'
      )).toBe(true);
      expect(onCalls.some((call: any) =>
        call[0] === 'mouseleave' && call[1] === 'unclustered-event-point'
      )).toBe(true);
    });
  });

  it('sets up moveend handler to update bbox', async () => {
    render(<ClusteredMapView />);

    await waitFor(() => {
      const onCalls = mockMapInstance.on.mock.calls;
      const moveendHandler = onCalls.find((call: any) => call[0] === 'moveend');
      expect(moveendHandler).toBeDefined();
    });
  });

  it('fetches initial data based on map bounds', async () => {
    render(<ClusteredMapView />);

    await waitFor(() => {
      expect(mockUpdateBBox).toHaveBeenCalledWith({
        north: 37.8,
        south: 37.7,
        east: -122.4,
        west: -122.5,
      });
    });
  });

  it('calls custom onLoad handler if provided', async () => {
    const onLoad = vi.fn();
    render(<ClusteredMapView onLoad={onLoad} />);

    await waitFor(() => {
      expect(onLoad).toHaveBeenCalledWith(mockMapInstance);
    });
  });
});
