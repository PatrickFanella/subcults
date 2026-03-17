import { useEffect, useRef, useCallback, useState } from 'react';
import type { Map } from 'maplibre-gl';

/**
 * Bounding box in standard [minLng, minLat, maxLng, maxLat] format
 * Compatible with most geospatial APIs
 */
export type BBoxArray = [number, number, number, number];

/**
 * Options for useMapBBox hook
 */
export interface UseMapBBoxOptions {
  /**
   * Debounce delay in milliseconds
   * Default: 500ms
   */
  debounceMs?: number;
  
  /**
   * Whether to call onBBoxChange immediately on mount if map has bounds
   * Default: false
   */
  immediate?: boolean;
}

/**
 * Result from useMapBBox hook
 */
export interface UseMapBBoxResult {
  /**
   * Current bounding box in [minLng, minLat, maxLng, maxLat] format
   * Null if map is not ready or has no bounds
   */
  bbox: BBoxArray | null;
  
  /**
   * Whether bbox change is pending (debouncing)
   */
  loading: boolean;
  
  /**
   * Error message if bbox computation failed
   */
  error: string | null;
}

/**
 * Hook for tracking map bounding box with debounced updates
 * 
 * Captures map movement events and provides debounced bbox updates to avoid
 * excessive network requests during rapid pan/zoom operations.
 * 
 * @param map - MapLibre Map instance (can be null if not yet initialized)
 * @param onBBoxChange - Callback invoked after debounce with new bbox
 * @param options - Configuration options
 * @returns Object with current bbox, loading state, and error
 * 
 * @example
 * ```tsx
 * const mapRef = useRef<Map>(null);
 * const { bbox, loading } = useMapBBox(
 *   mapRef.current,
 *   (bbox) => {
 *     // Fetch data for new bbox
 *     fetchScenes(bbox);
 *   },
 *   { debounceMs: 300 }
 * );
 * ```
 */
export function useMapBBox(
  map: Map | null,
  onBBoxChange: (bbox: BBoxArray) => void,
  options: UseMapBBoxOptions = {}
): UseMapBBoxResult {
  const { debounceMs = 500, immediate = false } = options;
  
  const [bbox, setBBox] = useState<BBoxArray | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Store callback in ref to avoid recreating event listeners
  const onBBoxChangeRef = useRef(onBBoxChange);
  
  // Track debounce timer
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  
  // Track if we're currently moving (for loading state)
  const isMovingRef = useRef(false);
  
  // Update callback ref when it changes
  useEffect(() => {
    onBBoxChangeRef.current = onBBoxChange;
  }, [onBBoxChange]);
  
  /**
   * Compute bbox from map bounds and update state
   */
  const computeAndUpdateBBox = useCallback(() => {
    if (!map) {
      setBBox(null);
      setError(null);
      return;
    }
    
    try {
      const bounds = map.getBounds();
      if (!bounds) {
        setBBox(null);
        setError('Map bounds not available');
        return;
      }
      
      // Convert to standard bbox format: [minLng, minLat, maxLng, maxLat]
      const newBBox: BBoxArray = [
        bounds.getWest(),
        bounds.getSouth(),
        bounds.getEast(),
        bounds.getNorth(),
      ];
      
      setBBox(newBBox);
      setError(null);
      setLoading(false);
      
      // Call the callback with new bbox
      if (onBBoxChangeRef.current) {
        onBBoxChangeRef.current(newBBox);
      }
    } catch (err) {
      console.error('Failed to compute bbox:', err);
      setError(err instanceof Error ? err.message : 'Unknown error');
      setLoading(false);
    }
  }, [map]);
  
  /**
   * Handle movestart event - track that movement has begun
   */
  const handleMoveStart = useCallback(() => {
    isMovingRef.current = true;
    setLoading(true);
  }, []);
  
  /**
   * Handle move event - cancel pending timer
   */
  const handleMove = useCallback(() => {
    // Cancel any pending debounce timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }
  }, []);
  
  /**
   * Handle moveend event - start debounce timer
   */
  const handleMoveEnd = useCallback(() => {
    isMovingRef.current = false;
    
    // Clear any existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    
    // Start new debounce timer
    debounceTimerRef.current = setTimeout(() => {
      computeAndUpdateBBox();
      debounceTimerRef.current = null;
    }, debounceMs);
  }, [debounceMs, computeAndUpdateBBox]);
  
  // Setup and teardown map event listeners
  useEffect(() => {
    if (!map) {
      return;
    }
    
    // Add event listeners
    map.on('movestart', handleMoveStart);
    map.on('move', handleMove);
    map.on('moveend', handleMoveEnd);
    
    // Call immediately if requested and map has bounds
    if (immediate) {
      // Use setTimeout to avoid calling setState synchronously in effect
      const timer = setTimeout(() => {
        computeAndUpdateBBox();
      }, 0);
      
      // Cleanup function
      return () => {
        clearTimeout(timer);
        map.off('movestart', handleMoveStart);
        map.off('move', handleMove);
        map.off('moveend', handleMoveEnd);
        
        // Clear pending timer
        if (debounceTimerRef.current) {
          clearTimeout(debounceTimerRef.current);
          debounceTimerRef.current = null;
        }
      };
    }
    
    // Cleanup function
    return () => {
      map.off('movestart', handleMoveStart);
      map.off('move', handleMove);
      map.off('moveend', handleMoveEnd);
      
      // Clear pending timer
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
        debounceTimerRef.current = null;
      }
    };
    // computeAndUpdateBBox is stable (only changes when map changes, which is in deps)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [map, immediate, handleMoveStart, handleMove, handleMoveEnd]);
  
  return {
    bbox,
    loading,
    error,
  };
}
