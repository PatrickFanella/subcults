# MapView Component

## Overview

The `MapView` component is a React wrapper around MapLibre GL JS that provides a privacy-first map display using MapTiler tiles. It's designed to be the foundation for displaying scenes, events, and clusters on the Subcults platform.

## Architecture

### Component Structure

```
MapView.tsx
├── Props Interface (MapViewProps)
├── Handle Interface (MapViewHandle) - Imperative API via ref
├── Component Implementation
│   ├── Map Initialization (useEffect)
│   ├── Resize Observer Setup
│   ├── Geolocation Handler (opt-in)
│   └── Cleanup on Unmount
└── Placeholder Layers (future cluster support)
```

### Key Design Decisions

1. **Privacy First**
   - Geolocation is disabled by default (`enableGeolocation={false}`)
   - When enabled, uses `enableHighAccuracy: false` for coarse location
   - No location tracking without explicit user consent

2. **Responsive Design**
   - Uses `ResizeObserver` to detect container size changes
   - Automatically calls `map.resize()` to prevent stale tiles
   - Works with any CSS layout (flexbox, grid, absolute positioning)

3. **Imperative API**
   - Exposes `getMap()` for direct MapLibre access
   - Provides `flyTo()` for programmatic navigation
   - Includes `getBounds()` for viewport queries
   - Uses `forwardRef` pattern for React best practices

4. **Placeholder Infrastructure**
   - Creates `scenes-placeholder` GeoJSON source with clustering enabled
   - Adds `clusters-placeholder` layer (circle-radius: 0 until real data)
   - Prepares for future scene/event rendering

## Usage Examples

### Basic Usage

```tsx
import { MapView } from './components/MapView';

function SceneMap() {
  return (
    <div style={{ height: '100vh' }}>
      <MapView />
    </div>
  );
}
```

### With Geolocation

```tsx
<MapView
  enableGeolocation={true}
  onGeolocationSuccess={(position) => {
    console.log('User location:', position.coords);
  }}
  onGeolocationError={(error) => {
    console.warn('Location access denied:', error.message);
  }}
/>
```

### With Initial Position

```tsx
// Option 1: Center + Zoom
<MapView
  initialPosition={{
    center: [-118.2437, 34.0522], // Los Angeles
    zoom: 12
  }}
/>

// Option 2: Bounds (fits viewport to area)
<MapView
  initialPosition={{
    bounds: [
      [-122.5, 37.7], // Southwest corner
      [-122.3, 37.8]  // Northeast corner
    ]
  }}
/>
```

### With Imperative Controls

```tsx
function InteractiveMap() {
  const mapRef = useRef<MapViewHandle>(null);

  const handleSearch = async (location: string) => {
    const coords = await geocode(location);
    mapRef.current?.flyTo(coords, 14);
  };

  const handleExport = () => {
    const bounds = mapRef.current?.getBounds();
    console.log('Current viewport:', bounds);
  };

  return (
    <>
      <MapView ref={mapRef} />
      <button onClick={() => handleSearch('San Francisco')}>
        Go to SF
      </button>
      <button onClick={handleExport}>
        Export Bounds
      </button>
    </>
  );
}
```

## Integration Points

### Future Cluster Rendering

The component includes a placeholder layer that can be updated with real scene data:

```tsx
function SceneMapWithData() {
  const mapRef = useRef<MapViewHandle>(null);

  useEffect(() => {
    const map = mapRef.current?.getMap();
    if (!map) return;

    // Wait for map to load
    map.on('load', () => {
      // Update placeholder source with real data
      const source = map.getSource('scenes-placeholder');
      if (source && source.type === 'geojson') {
        source.setData({
          type: 'FeatureCollection',
          features: scenes.map(scene => ({
            type: 'Feature',
            geometry: {
              type: 'Point',
              coordinates: [scene.lng, scene.lat]
            },
            properties: {
              id: scene.id,
              name: scene.name,
              // ... other scene metadata
            }
          }))
        });
      }

      // Update cluster layer visibility
      map.setPaintProperty('clusters-placeholder', 'circle-radius', 20);
    });
  }, [scenes]);

  return <MapView ref={mapRef} />;
}
```

### Custom Map Interactions

```tsx
const map = mapRef.current?.getMap();

// Add click handler
map?.on('click', 'clusters-placeholder', (e) => {
  const clusterId = e.features[0].properties.cluster_id;
  const source = map.getSource('scenes-placeholder');
  
  source.getClusterExpansionZoom(clusterId, (err, zoom) => {
    if (err) return;
    
    map.easeTo({
      center: e.lngLat,
      zoom: zoom
    });
  });
});

// Change cursor on hover
map?.on('mouseenter', 'clusters-placeholder', () => {
  map.getCanvas().style.cursor = 'pointer';
});

map?.on('mouseleave', 'clusters-placeholder', () => {
  map.getCanvas().style.cursor = '';
});
```

## Testing

The component includes comprehensive test coverage:

- Map initialization with MapTiler style
- Resize observer setup and cleanup
- Imperative ref methods (getMap, flyTo, getBounds)
- Geolocation privacy controls
- Component lifecycle (mount/unmount)
- Placeholder layer rendering

Run tests with:
```bash
cd web && npm test
```

## Environment Variables

The component requires `VITE_MAPTILER_API_KEY` to be set:

1. Copy `.env.example` to `.env` in the `web/` directory
2. Get an API key from https://cloud.maptiler.com/account/keys/
3. Add: `VITE_MAPTILER_API_KEY=your_key_here`

Note: MapTiler keys are client-side and will be included in the browser bundle. This is acceptable for tile access. See [MapTiler Security Best Practices](https://docs.maptiler.com/cloud/api/authentication-key/) for production recommendations.

## Performance Considerations

- Map tiles are cached by MapLibre GL
- Cluster calculations happen in MapLibre's native code (WebGL)
- ResizeObserver only triggers on actual size changes
- Component avoids unnecessary re-renders via ref pattern

## Privacy & Security

1. **No Automatic Location Requests**
   - Geolocation must be explicitly enabled via prop
   - Default behavior is static map at fallback coordinates

2. **Coarse Location Only**
   - `enableHighAccuracy: false` prevents precise GPS coordinates
   - Matches platform privacy principles (coarse > precise)

3. **MapTiler API Key**
   - Client-side key is acceptable for public tile access
   - Implement key rotation in production
   - Consider domain restrictions in MapTiler dashboard

## Next Steps

1. **Scene Data Integration** - Connect to `/api/scenes` endpoint
2. **Cluster Customization** - Style clusters based on scene count/type
3. **Popup Cards** - Show scene details on marker click
4. **User Location Marker** - Add blue dot for user position (when geolocation enabled)
5. **Offline Support** - Cache tiles for offline map viewing
6. **Dark Mode** - Switch between light/dark MapTiler styles

## Related Documentation

- [MapLibre GL JS API](https://maplibre.org/maplibre-gl-js/docs/API/)
- [MapTiler Cloud](https://docs.maptiler.com/cloud/)
- [Privacy Design Principles](../../docs/PRIVACY.md)
- [Frontend Architecture](../README.md)
