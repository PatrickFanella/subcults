import { useState, useEffect, useCallback, useRef } from 'react';
import type { LngLatBounds } from 'maplibre-gl';
import type { Scene, Event } from '../types/scene';
import type { FeatureCollection, Point as GeoJSONPoint } from 'geojson';
import { buildGeoJSON, type FeatureProperties } from '../utils/geojson';

/**
 * Bounding box for geographic queries
 */
export interface BBox {
  north: number;
  south: number;
  east: number;
  west: number;
}

/**
 * Convert MapLibre LngLatBounds to BBox
 */
export function boundsToBox(bounds: LngLatBounds): BBox {
  return {
    north: bounds.getNorth(),
    south: bounds.getSouth(),
    east: bounds.getEast(),
    west: bounds.getWest(),
  };
}

/**
 * Options for useClusteredData hook
 */
export interface UseClusteredDataOptions {
  /**
   * API endpoint base URL
   * Default: uses VITE_API_URL env var or '/api'
   */
  apiUrl?: string;
  
  /**
   * Whether to automatically fetch on mount
   * Default: false
   */
  autoFetch?: boolean;
  
  /**
   * Debounce delay in milliseconds for bbox changes
   * Default: 300
   */
  debounceMs?: number;
}

/**
 * Result from useClusteredData hook
 */
export interface UseClusteredDataResult {
  /**
   * GeoJSON FeatureCollection for rendering
   */
  data: FeatureCollection<GeoJSONPoint, FeatureProperties>;
  
  /**
   * Whether data is currently being fetched
   */
  loading: boolean;
  
  /**
   * Error message if fetch failed
   */
  error: string | null;
  
  /**
   * Manually trigger a data fetch
   */
  refetch: () => void;
  
  /**
   * Update the bounding box and trigger fetch
   */
  updateBBox: (bbox: BBox | null) => void;
}

/**
 * Hook for fetching and clustering scene/event data based on map bounds
 * 
 * @param bbox - Initial bounding box (optional)
 * @param options - Configuration options
 * @returns Object with data, loading state, error, and refetch function
 * 
 * @example
 * const { data, loading, error, updateBBox } = useClusteredData();
 * 
 * // Update when map moves
 * map.on('moveend', () => {
 *   const bounds = map.getBounds();
 *   updateBBox(boundsToBox(bounds));
 * });
 */
export function useClusteredData(
  initialBBox: BBox | null = null,
  options: UseClusteredDataOptions = {}
): UseClusteredDataResult {
  const {
    apiUrl = import.meta.env.VITE_API_URL || '/api',
    debounceMs = 300,
  } = options;

  const [bbox, setBBox] = useState<BBox | null>(initialBBox);
  const [data, setData] = useState<FeatureCollection<GeoJSONPoint, FeatureProperties>>({
    type: 'FeatureCollection',
    features: [],
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Use ref to track the latest fetch controller for cancellation
  const abortControllerRef = useRef<AbortController | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Counter for unique performance mark IDs
  const fetchCounterRef = useRef(0);

  const fetchData = useCallback(async (currentBBox: BBox | null) => {
    // Cancel any pending fetch
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    if (!currentBBox) {
      setData({ type: 'FeatureCollection', features: [] });
      return;
    }

    setLoading(true);
    setError(null);

    const controller = new AbortController();
    abortControllerRef.current = controller;

    // Performance mark: Start data fetch
    const fetchId = `data-fetch-${++fetchCounterRef.current}`;
    performance.mark(`${fetchId}-start`);

    try {
      // Build query parameters
      const params = new URLSearchParams({
        north: currentBBox.north.toString(),
        south: currentBBox.south.toString(),
        east: currentBBox.east.toString(),
        west: currentBBox.west.toString(),
      });

      // Fetch scenes and events in parallel
      const [scenesRes, eventsRes] = await Promise.all([
        fetch(`${apiUrl}/scenes?${params}`, { signal: controller.signal }),
        fetch(`${apiUrl}/events?${params}`, { signal: controller.signal }),
      ]);

      if (!scenesRes.ok) {
        throw new Error(`Failed to fetch scenes: ${scenesRes.status} ${scenesRes.statusText}`);
      }
      if (!eventsRes.ok) {
        throw new Error(`Failed to fetch events: ${eventsRes.status} ${eventsRes.statusText}`);
      }

      const scenes: Scene[] = await scenesRes.json();
      const events: Event[] = await eventsRes.json();

      // Performance mark: Start GeoJSON build
      performance.mark(`${fetchId}-geojson-start`);

      // Build GeoJSON from entities
      const geojson = buildGeoJSON(scenes, events);
      
      // Performance mark: Complete GeoJSON build
      performance.mark(`${fetchId}-geojson-end`);
      performance.measure(`${fetchId}-geojson-build`, `${fetchId}-geojson-start`, `${fetchId}-geojson-end`);
      
      setData(geojson);
      setError(null);

      // Performance mark: Complete data fetch
      performance.mark(`${fetchId}-end`);
      performance.measure(`${fetchId}-total`, `${fetchId}-start`, `${fetchId}-end`);

      // Log performance metrics
      const totalMeasure = performance.getEntriesByName(`${fetchId}-total`)[0] as PerformanceMeasure;
      const geojsonMeasure = performance.getEntriesByName(`${fetchId}-geojson-build`)[0] as PerformanceMeasure;
      
      if (totalMeasure && geojsonMeasure) {
        console.log(`[Performance] Data fetch: ${totalMeasure.duration.toFixed(2)}ms (GeoJSON build: ${geojsonMeasure.duration.toFixed(2)}ms)`);
      }
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        // Fetch was cancelled, ignore
        return;
      }
      
      console.error('Failed to fetch clustered data:', err);
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
      abortControllerRef.current = null;
    }
  }, [apiUrl]);

  const updateBBox = useCallback((newBBox: BBox | null) => {
    setBBox(newBBox);
  }, []);

  const refetch = useCallback(() => {
    fetchData(bbox);
  }, [bbox, fetchData]);

  // Debounced fetch when bbox changes
  useEffect(() => {
    // Clear existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // Debounce the fetch
    debounceTimerRef.current = setTimeout(() => {
      fetchData(bbox);
    }, debounceMs);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [bbox, fetchData, debounceMs]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  return {
    data,
    loading,
    error,
    refetch,
    updateBBox,
  };
}
