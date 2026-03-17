import {
  useEffect,
  useRef,
  useImperativeHandle,
  forwardRef,
  useState,
} from 'react';
import maplibregl, { type LngLatBoundsLike, type LngLatLike, Map } from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';

/**
 * Bounds or center/zoom for initial map position
 */
export interface InitialPosition {
  bounds?: LngLatBoundsLike;
  center?: LngLatLike;
  zoom?: number;
}

/**
 * Props for the MapView component
 */
export interface MapViewProps {
  /**
   * MapTiler API key for tile requests
   * Can be passed via prop or VITE_MAPTILER_API_KEY env var
   */
  apiKey?: string;
  
  /**
   * Initial map position (bounds or center/zoom)
   * If not provided and geolocation is enabled, will use user location
   */
  initialPosition?: InitialPosition;
  
  /**
   * Whether to attempt geolocation fallback when no initial position is provided
   * Default: false (privacy-conscious)
   */
  enableGeolocation?: boolean;
  
  /**
   * CSS class name for the map container
   */
  className?: string;
  
  /**
   * Callback when map is fully loaded
   */
  onLoad?: (map: Map) => void;
  
  /**
   * Callback when geolocation succeeds
   */
  onGeolocationSuccess?: (position: GeolocationPosition) => void;
  
  /**
   * Callback when geolocation fails
   */
  onGeolocationError?: (error: GeolocationPositionError) => void;
}

/**
 * Imperative handle exposed via ref
 */
export interface MapViewHandle {
  /**
   * Get the underlying MapLibre Map instance
   */
  getMap: () => Map | null;
  
  /**
   * Fly to a specific location
   */
  flyTo: (center: LngLatLike, zoom?: number) => void;
  
  /**
   * Get current map bounds
   */
  getBounds: () => maplibregl.LngLatBounds | null;
}

// Default fallback coordinates (San Francisco)
const DEFAULT_CENTER: [number, number] = [-122.4194, 37.7749];
const DEFAULT_ZOOM = 12;

/**
 * MapView component - Privacy-first map display using MapLibre GL with MapTiler tiles
 * 
 * Privacy considerations:
 * - Geolocation is opt-in via enableGeolocation prop
 * - No location requests without explicit user permission
 * - MapTiler API key is client-side (acceptable for public tile access)
 */
export const MapView = forwardRef<MapViewHandle, MapViewProps>(
  (
    {
      apiKey,
      initialPosition,
      enableGeolocation = false,
      className = '',
      onLoad,
      onGeolocationSuccess,
      onGeolocationError,
    },
    ref
  ) => {
    const mapContainerRef = useRef<HTMLDivElement>(null);
    const mapRef = useRef<Map | null>(null);
    const resizeObserverRef = useRef<ResizeObserver | null>(null);
    const [isMapLoaded, setIsMapLoaded] = useState(false);
    
    // Store callbacks in refs to avoid map recreation when they change
    const onLoadRef = useRef(onLoad);
    const onGeolocationSuccessRef = useRef(onGeolocationSuccess);
    const onGeolocationErrorRef = useRef(onGeolocationError);
    
    // Update refs when callbacks change
    useEffect(() => {
      onLoadRef.current = onLoad;
    }, [onLoad]);
    
    useEffect(() => {
      onGeolocationSuccessRef.current = onGeolocationSuccess;
    }, [onGeolocationSuccess]);
    
    useEffect(() => {
      onGeolocationErrorRef.current = onGeolocationError;
    }, [onGeolocationError]);

    // Get API key from prop or environment variable
    const maptilerApiKey = apiKey || import.meta.env.VITE_MAPTILER_API_KEY;

    // Expose imperative methods via ref
    useImperativeHandle(ref, () => ({
      getMap: () => mapRef.current,
      flyTo: (center: LngLatLike, zoom?: number) => {
        if (mapRef.current) {
          mapRef.current.flyTo({
            center,
            zoom: zoom ?? mapRef.current.getZoom(),
            essential: true,
          });
        }
      },
      getBounds: () => {
        return mapRef.current ? mapRef.current.getBounds() : null;
      },
    }));

    useEffect(() => {
      if (!maptilerApiKey || !mapContainerRef.current || mapRef.current) return;

      // Determine initial position
      let initialCenter: [number, number] = DEFAULT_CENTER as [number, number];
      let initialZoom = DEFAULT_ZOOM;
      let initialBounds: LngLatBoundsLike | undefined;

      if (initialPosition?.bounds) {
        initialBounds = initialPosition.bounds;
      } else if (initialPosition?.center) {
        // Convert LngLatLike to tuple
        const center = initialPosition.center;
        if (Array.isArray(center)) {
          initialCenter = center as [number, number];
        } else if ('lng' in center && 'lat' in center) {
          initialCenter = [center.lng, center.lat];
        }
        initialZoom = initialPosition.zoom ?? DEFAULT_ZOOM;
      }

      // Initialize map with MapTiler style
      const styleUrl = `https://api.maptiler.com/maps/streets-v2/style.json?key=${maptilerApiKey}`;

      const map = new maplibregl.Map({
        container: mapContainerRef.current,
        style: styleUrl,
        center: initialBounds ? undefined : initialCenter,
        zoom: initialBounds ? undefined : initialZoom,
        bounds: initialBounds,
      });

      mapRef.current = map;

      // Handle map load event
      map.on('load', () => {
        setIsMapLoaded(true);
        
        // Add placeholder source and layer for future cluster rendering
        map.addSource('scenes-placeholder', {
          type: 'geojson',
          data: {
            type: 'FeatureCollection',
            features: [],
          },
          cluster: true,
          clusterMaxZoom: 14,
          clusterRadius: 50,
        });

        map.addLayer({
          id: 'clusters-placeholder',
          type: 'circle',
          source: 'scenes-placeholder',
          filter: ['has', 'point_count'],
          paint: {
            'circle-color': '#11b4da',
            'circle-radius': 0, // Hidden until we have real data
          },
        });

        if (onLoadRef.current) {
          onLoadRef.current(map);
        }
      });

      // Attempt geolocation if enabled and no initial position provided
      if (
        enableGeolocation &&
        !initialPosition?.bounds &&
        !initialPosition?.center &&
        'geolocation' in navigator
      ) {
        navigator.geolocation.getCurrentPosition(
          (position) => {
            const { latitude, longitude } = position.coords;
            map.flyTo({
              center: [longitude, latitude],
              zoom: 13,
              essential: true,
            });
            if (onGeolocationSuccessRef.current) {
              onGeolocationSuccessRef.current(position);
            }
          },
          (error) => {
            console.warn('Geolocation failed:', error.message);
            if (onGeolocationErrorRef.current) {
              onGeolocationErrorRef.current(error);
            }
          },
          {
            enableHighAccuracy: false, // Use coarse location for privacy
            timeout: 5000,
            maximumAge: 60000,
          }
        );
      }

      // Setup resize observer to handle container size changes
      resizeObserverRef.current = new ResizeObserver(() => {
        if (mapRef.current) {
          mapRef.current.resize();
        }
      });

      if (mapContainerRef.current) {
        resizeObserverRef.current.observe(mapContainerRef.current);
      }

      // Cleanup function
      return () => {
        if (resizeObserverRef.current) {
          resizeObserverRef.current.disconnect();
        }
        if (mapRef.current) {
          mapRef.current.remove();
          mapRef.current = null;
        }
      };
    }, [
      maptilerApiKey,
      initialPosition,
      enableGeolocation,
    ]);

    // Show error if no API key
    if (!maptilerApiKey) {
      return (
        <div
          className={`map-error ${className}`}
          style={{ 
            width: '100%', 
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '2rem',
            backgroundColor: '#fee',
            color: '#c33',
            borderRadius: '0.5rem'
          }}
          data-testid="map-error"
        >
          <h2 style={{ fontSize: '1.25rem', fontWeight: 600, marginBottom: '0.5rem' }}>
            Map Unavailable
          </h2>
          <p style={{ textAlign: 'center' }}>
            MapTiler API key not provided.<br />
            Set <code style={{ backgroundColor: '#f5f5f5', padding: '0.25rem', borderRadius: '0.25rem' }}>VITE_MAPTILER_API_KEY</code> environment variable or pass <code style={{ backgroundColor: '#f5f5f5', padding: '0.25rem', borderRadius: '0.25rem' }}>apiKey</code> prop.
          </p>
        </div>
      );
    }

    return (
      <div
        ref={mapContainerRef}
        className={`map-container ${className}`}
        style={{ width: '100%', height: '100%' }}
        data-testid="map-container"
        data-map-loaded={isMapLoaded}
        role="application"
        aria-label="Interactive map showing scenes and events"
      />
    );
  }
);

MapView.displayName = 'MapView';
