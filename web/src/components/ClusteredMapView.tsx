import { useEffect, useRef, useCallback, useState, forwardRef } from 'react';
import { useClusteredData, boundsToBox } from '../hooks/useClusteredData';
import { MapView, type MapViewHandle, type MapViewProps } from './MapView';
import type { Map as MapLibreMap, GeoJSONSource } from 'maplibre-gl';
import maplibregl from 'maplibre-gl';
import { DetailPanel } from './DetailPanel';
import type { Scene, Event } from '../types/scene';

/**
 * Props for ClusteredMapView component
 */
export interface ClusteredMapViewProps extends Omit<MapViewProps, 'onLoad'> {
  /**
   * Custom handler when map loads (receives map instance)
   */
  onLoad?: (map: MapLibreMap) => void;
}

/**
 * ClusteredMapView - Enhanced MapView with scene/event clustering
 * 
 * This component wraps MapView and adds:
 * - Automatic data fetching based on map bounds
 * - Real-time cluster rendering for scenes and events
 * - Click handlers for cluster expansion and detail panel
 * - Separate icon styling for scenes vs events
 * - Privacy jitter visualization with tooltips
 * - Detail panel for marker interaction
 * 
 * Privacy considerations:
 * - Respects location consent flags from backend
 * - Uses coarse geohash coordinates when precise location is not allowed
 * - Applies deterministic jitter to coarse coordinates
 * - Shows privacy notice in tooltips for jittered locations (HTML-escaped)
 * - Detail panel enforces privacy for coordinate display
 */
export const ClusteredMapView = forwardRef<MapViewHandle, ClusteredMapViewProps>(function ClusteredMapView(props, ref) {
  const mapRef = useRef<MapViewHandle>(null);
  const { data, updateBBox, loading, error } = useClusteredData(null, { debounceMs: 300 });
  const mapInstanceRef = useRef<MapLibreMap | null>(null);
  const popupRef = useRef<maplibregl.Popup | null>(null);
  
  // Counters for unique performance mark IDs
  const initCounterRef = useRef(0);
  const updateCounterRef = useRef(0);
  const renderCounterRef = useRef(0);
  
  // Detail panel state
  const [selectedEntity, setSelectedEntity] = useState<Scene | Event | null>(null);
  const [panelLoading, setPanelLoading] = useState(false);
  const entityCacheRef = useRef(new Map<string, Scene | Event>());

  // Fetch entity details
  const fetchEntityDetails = useCallback(async (id: string, type: 'scene' | 'event') => {
    // Check cache first
    const entityCache = entityCacheRef.current;
    const cacheKey = `${type}-${id}`;
    if (entityCache.has(cacheKey)) {
      setSelectedEntity(entityCache.get(cacheKey)!);
      return;
    }

    setPanelLoading(true);
    try {
      const apiUrl = import.meta.env.VITE_API_URL || '/api';
      const endpoint = type === 'scene' ? 'scenes' : 'events';
      const response = await fetch(`${apiUrl}/${endpoint}/${id}`);
      
      if (!response.ok) {
        throw new Error(`Failed to fetch ${type}: ${response.status}`);
      }
      
      const entity: Scene | Event = await response.json();
      
      // Cache the entity
      entityCache.set(cacheKey, entity);
      setSelectedEntity(entity);
    } catch (err) {
      console.error(`Failed to fetch ${type} details:`, err);
      // Keep basic info from GeoJSON properties displayed
    } finally {
      setPanelLoading(false);
    }
  }, []);

  // Handle marker click
  const handleMarkerClick = useCallback((feature: GeoJSON.Feature) => {
    if (!feature.properties) return;
    
    const { id, type, name, description, tags, allow_precise, coarse_geohash } = feature.properties;
    
    // Create a basic entity from GeoJSON properties
    const basicEntity: Scene | Event = type === 'scene'
      ? {
          id,
          name,
          description,
          allow_precise: allow_precise || false,
          coarse_geohash: coarse_geohash || '',
          tags: tags ? tags.split(',') : undefined,
          visibility: feature.properties.visibility || 'public',
        }
      : {
          id,
          scene_id: feature.properties.scene_id || '',
          name,
          description,
          allow_precise: allow_precise || false,
          coarse_geohash: coarse_geohash || undefined,
        };
    
    // Show basic info immediately
    setSelectedEntity(basicEntity);
    
    // Fetch full details in background
    fetchEntityDetails(id, type);
  }, [fetchEntityDetails]);

  // Helper function to show privacy tooltip for jittered markers
  // Memoized to avoid recreating on every render
  const showPrivacyTooltip = useCallback((
    map: MapLibreMap,
    coordinates: [number, number],
    name: string
  ) => {
    // Create popup if it doesn't exist
    if (!popupRef.current) {
      popupRef.current = new maplibregl.Popup({
        closeButton: false,
        closeOnClick: false,
        offset: 15,
      });
    }
    
    // Escape HTML to prevent XSS
    const escapedName = name
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
    
    popupRef.current
      .setLngLat(coordinates)
      .setHTML(`
        <div style="padding: 8px; font-size: 12px; line-height: 1.4;">
          <strong>${escapedName}</strong><br/>
          <em style="color: #666;">📍 Approximate location (privacy preserved)</em>
        </div>
      `)
      .addTo(map);
  }, []); // Empty deps array since popup ref is stable and function doesn't depend on props/state

  // Handle map load event
  const handleMapLoad = (map: MapLibreMap) => {
    // Performance mark: Start map initialization
    const initId = `map-init-${++initCounterRef.current}`;
    performance.mark(`${initId}-start`);

    mapInstanceRef.current = map;

    // Remove placeholder source and layers
    if (map.getLayer('clusters-placeholder')) {
      map.removeLayer('clusters-placeholder');
    }
    if (map.getSource('scenes-placeholder')) {
      map.removeSource('scenes-placeholder');
    }

    // Add real cluster source
    map.addSource('scenes-events', {
      type: 'geojson',
      data,
      cluster: true,
      clusterMaxZoom: 14,
      clusterRadius: 50,
    });

    // Add cluster circle layer with size buckets
    map.addLayer({
      id: 'clusters',
      type: 'circle',
      source: 'scenes-events',
      filter: ['has', 'point_count'],
      paint: {
        // Size based on point_count
        'circle-color': [
          'step',
          ['get', 'point_count'],
          '#51bbd6', // < 10 points
          10,
          '#f1f075', // 10-100 points
          100,
          '#f28cb1', // 100+ points
        ],
        'circle-radius': [
          'step',
          ['get', 'point_count'],
          20, // < 10 points
          10,
          30, // 10-100 points
          100,
          40, // 100+ points
        ],
      },
    });

    // Add cluster count labels
    map.addLayer({
      id: 'cluster-count',
      type: 'symbol',
      source: 'scenes-events',
      filter: ['has', 'point_count'],
      layout: {
        'text-field': '{point_count_abbreviated}',
        'text-font': ['DIN Offc Pro Medium', 'Arial Unicode MS Bold'],
        'text-size': 12,
      },
    });

    // Add unclustered scene points layer
    map.addLayer({
      id: 'unclustered-scene-point',
      type: 'circle',
      source: 'scenes-events',
      filter: ['all', ['!', ['has', 'point_count']], ['==', ['get', 'type'], 'scene']],
      paint: {
        'circle-color': '#11b4da',
        'circle-radius': 8,
        'circle-stroke-width': 2,
        'circle-stroke-color': '#fff',
        // Add subtle opacity for jittered markers
        'circle-opacity': [
          'case',
          ['get', 'is_jittered'],
          0.8, // Slightly transparent for jittered
          1.0, // Fully opaque for precise
        ],
      },
    });

    // Add unclustered event points layer
    map.addLayer({
      id: 'unclustered-event-point',
      type: 'circle',
      source: 'scenes-events',
      filter: ['all', ['!', ['has', 'point_count']], ['==', ['get', 'type'], 'event']],
      paint: {
        'circle-color': '#f28cb1',
        'circle-radius': 6,
        'circle-stroke-width': 2,
        'circle-stroke-color': '#fff',
        // Add subtle opacity for jittered markers
        'circle-opacity': [
          'case',
          ['get', 'is_jittered'],
          0.8, // Slightly transparent for jittered
          1.0, // Fully opaque for precise
        ],
      },
    });

    // Add click handler for clusters to expand/zoom
    map.on('click', 'clusters', (e) => {
      const features = map.queryRenderedFeatures(e.point, {
        layers: ['clusters'],
      });
      
      if (!features.length) return;

      const clusterId = features[0].properties?.cluster_id;
      const source = map.getSource('scenes-events');
      
      if (source && 'getClusterExpansionZoom' in source && clusterId !== undefined) {
        // Cast to any for callback-based getClusterExpansionZoom (MapLibre runtime supports both Promise and callback)
        (source as GeoJSONSource & { getClusterExpansionZoom(clusterId: number, callback: (err: Error | null, zoom: number | null) => void): void }).getClusterExpansionZoom(clusterId, (err, zoom) => {
          if (err) {
            console.error('Failed to expand cluster:', err);
            return;
          }
          if (!features[0].geometry || features[0].geometry.type !== 'Point') return;

          map.easeTo({
            center: features[0].geometry.coordinates as [number, number],
            zoom: zoom !== null ? zoom : map.getZoom() + 2,
          });
        });
      }
    });

    // Change cursor on hover over clusters
    map.on('mouseenter', 'clusters', () => {
      map.getCanvas().style.cursor = 'pointer';
    });
    map.on('mouseleave', 'clusters', () => {
      map.getCanvas().style.cursor = '';
    });

    // Add cursor and tooltip for unclustered scene points
    map.on('mouseenter', 'unclustered-scene-point', (e) => {
      map.getCanvas().style.cursor = 'pointer';
      
      if (e.features && e.features.length > 0) {
        const feature = e.features[0];
        const isJittered = feature.properties?.is_jittered;
        const name = feature.properties?.name || 'Scene';
        
        // Always remove existing popup first to prevent stale tooltips
        if (popupRef.current) {
          popupRef.current.remove();
        }
        
        if (isJittered) {
          const coordinates = (feature.geometry as GeoJSON.Point).coordinates.slice() as [number, number];
          showPrivacyTooltip(map, coordinates, name);
        }
      }
    });
    
    map.on('mouseleave', 'unclustered-scene-point', () => {
      map.getCanvas().style.cursor = '';
      if (popupRef.current) {
        popupRef.current.remove();
      }
    });
    
    // Add cursor and tooltip for unclustered event points
    map.on('mouseenter', 'unclustered-event-point', (e) => {
      map.getCanvas().style.cursor = 'pointer';
      
      if (e.features && e.features.length > 0) {
        const feature = e.features[0];
        const isJittered = feature.properties?.is_jittered;
        const name = feature.properties?.name || 'Event';
        
        // Always remove existing popup first to prevent stale tooltips
        if (popupRef.current) {
          popupRef.current.remove();
        }
        
        if (isJittered) {
          const coordinates = (feature.geometry as GeoJSON.Point).coordinates.slice() as [number, number];
          showPrivacyTooltip(map, coordinates, name);
        }
      }
    });
    
    map.on('mouseleave', 'unclustered-event-point', () => {
      map.getCanvas().style.cursor = '';
      if (popupRef.current) {
        popupRef.current.remove();
      }
    });

    // Add click handler for unclustered scene points
    map.on('click', 'unclustered-scene-point', (e) => {
      if (e.features && e.features.length > 0) {
        const feature = e.features[0];
        handleMarkerClick(feature as GeoJSON.Feature);
        
        // Remove tooltip if showing
        if (popupRef.current) {
          popupRef.current.remove();
        }
      }
    });

    // Add click handler for unclustered event points
    map.on('click', 'unclustered-event-point', (e) => {
      if (e.features && e.features.length > 0) {
        const feature = e.features[0];
        handleMarkerClick(feature as GeoJSON.Feature);
        
        // Remove tooltip if showing
        if (popupRef.current) {
          popupRef.current.remove();
        }
      }
    });

    // Update bbox on map move
    map.on('moveend', () => {
      const bounds = map.getBounds();
      updateBBox(boundsToBox(bounds));
    });

    // Initial bbox fetch
    const bounds = map.getBounds();
    updateBBox(boundsToBox(bounds));

    // Performance mark: Complete map initialization
    performance.mark(`${initId}-end`);
    performance.measure(`${initId}-duration`, `${initId}-start`, `${initId}-end`);

    const measure = performance.getEntriesByName(`${initId}-duration`)[0] as PerformanceMeasure;
    if (measure) {
      console.log(`[Performance] Map initialization: ${measure.duration.toFixed(2)}ms`);
    }

    // Call custom onLoad handler if provided
    if (props.onLoad) {
      props.onLoad(map);
    }
  };

  // Update source data when clustered data changes
  useEffect(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    const source = map.getSource('scenes-events');
    if (source && 'setData' in source) {
      // Performance mark: Start source update
      const updateId = `source-update-${++updateCounterRef.current}`;
      performance.mark(`${updateId}-start`);

      // Cast to GeoJSONSource - setData is defined on the type
      (source as GeoJSONSource).setData(data);

      // Performance mark: Complete source update
      performance.mark(`${updateId}-end`);
      performance.measure(`${updateId}-duration`, `${updateId}-start`, `${updateId}-end`);

      const measure = performance.getEntriesByName(`${updateId}-duration`)[0] as PerformanceMeasure;
      if (measure) {
        console.log(`[Performance] Map source update: ${measure.duration.toFixed(2)}ms (${data.features.length} features)`);
      }

      // Schedule render complete detection
      requestAnimationFrame(() => {
        const renderId = `render-complete-${++renderCounterRef.current}`;
        performance.mark(renderId);
        console.log(`[Performance] Layer render complete`);
      });
    }
  }, [data]);

  // Cleanup popup on unmount
  useEffect(() => {
    return () => {
      if (popupRef.current) {
        popupRef.current.remove();
        popupRef.current = null;
      }
    };
  }, []);

  // Display loading/error states
  if (error) {
    console.error('Clustering error:', error);
  }
  if (loading) {
    console.log('Loading clustered data...');
  }

  // Handle panel close
  const handlePanelClose = useCallback(() => {
    setSelectedEntity(null);
  }, []);

  // Analytics callback (placeholder for future implementation)
  const handleAnalyticsEvent = useCallback((eventName: string, data?: Record<string, unknown>) => {
    console.log('Analytics event:', eventName, data);
    // TODO: Integrate with analytics service
  }, []);

  return (
    <>
      <MapView {...props} ref={ref || mapRef} onLoad={handleMapLoad} />
      <DetailPanel
        isOpen={selectedEntity !== null}
        onClose={handlePanelClose}
        entity={selectedEntity}
        loading={panelLoading}
        onAnalyticsEvent={handleAnalyticsEvent}
      />
    </>
  );
});
