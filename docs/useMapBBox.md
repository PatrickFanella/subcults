# useMapBBox Hook

A React hook for tracking map bounding box changes with debounced updates.

## Overview

The `useMapBBox` hook captures map movement events from a MapLibre GL map instance and provides debounced bounding box updates. This is essential for avoiding excessive network requests during rapid pan/zoom operations while keeping displayed data current.

## Features

- **Debounced Updates**: Configurable debounce delay (default 500ms) to prevent excessive API calls
- **Loading State**: Track when bbox changes are pending
- **Error Handling**: Graceful error handling for bbox computation failures
- **Standard Format**: Returns bbox in standard `[minLng, minLat, maxLng, maxLat]` format
- **Automatic Cleanup**: Event listeners and timers are properly cleaned up on unmount
- **Privacy-Aware**: Works seamlessly with privacy-first geolocation features

## Usage

### Basic Example

```tsx
import { useRef, useState } from 'react';
import type { Map } from 'maplibre-gl';
import { MapView, type MapViewHandle } from '../components/MapView';
import { useMapBBox } from '../hooks/useMapBBox';

function MapWithBBox() {
  const mapRef = useRef<MapViewHandle>(null);
  const [mapInstance, setMapInstance] = useState<Map | null>(null);
  
  const { bbox, loading } = useMapBBox(
    mapInstance,
    (newBBox) => {
      console.log('Bbox changed:', newBBox);
      // Fetch data for new bbox
      fetchScenes(newBBox);
    },
    { debounceMs: 500 }
  );
  
  return (
    <>
      <MapView
        ref={mapRef}
        onLoad={(map) => setMapInstance(map)}
      />
      {loading && <div>Loading new data...</div>}
      {bbox && <div>Current area: {bbox.join(', ')}</div>}
    </>
  );
}
```

### Integration with Data Fetching

```tsx
import { useRef, useState } from 'react';
import type { Map } from 'maplibre-gl';
import { MapView, type MapViewHandle } from '../components/MapView';
import { useMapBBox, type BBoxArray } from '../hooks/useMapBBox';
import { fetchScenesInBBox, fetchEventsInBBox } from '../services/api';

function MapWithDataFetch() {
  const mapRef = useRef<MapViewHandle>(null);
  const [mapInstance, setMapInstance] = useState<Map | null>(null);
  const [scenes, setScenes] = useState([]);
  const [events, setEvents] = useState([]);
  const abortControllerRef = useRef<AbortController | null>(null);
  
  const handleBBoxChange = async (bbox: BBoxArray) => {
    // Cancel previous fetch
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    
    const controller = new AbortController();
    abortControllerRef.current = controller;
    
    try {
      // Fetch scenes and events in parallel
      const [scenesData, eventsData] = await Promise.all([
        fetchScenesInBBox(bbox, { signal: controller.signal }),
        fetchEventsInBBox(bbox, { signal: controller.signal }),
      ]);
      
      setScenes(scenesData);
      setEvents(eventsData);
    } catch (error) {
      if (error instanceof Error && error.name !== 'AbortError') {
        console.error('Failed to fetch data:', error);
      }
    }
  };
  
  const { bbox, loading } = useMapBBox(
    mapInstance,
    handleBBoxChange,
    { debounceMs: 300 }
  );
  
  return (
    <div>
      <MapView
        ref={mapRef}
        onLoad={(map) => setMapInstance(map)}
      />
      {loading && <div className="loading-indicator">Updating...</div>}
      <div>
        Scenes: {scenes.length} | Events: {events.length}
      </div>
    </div>
  );
}
```

### Using with useClusteredData

```tsx
import { useRef, useState } from 'react';
import type { Map } from 'maplibre-gl';
import { MapView, type MapViewHandle } from '../components/MapView';
import { useMapBBox, type BBoxArray } from '../hooks/useMapBBox';
import { useClusteredData } from '../hooks/useClusteredData';

function MapWithClustering() {
  const mapRef = useRef<MapViewHandle>(null);
  const [mapInstance, setMapInstance] = useState<Map | null>(null);
  
  // Use useClusteredData for data fetching with its own debouncing
  const { data, loading: dataLoading, updateBBox } = useClusteredData(null, {
    debounceMs: 300,
  });
  
  // Track bbox changes and update data hook
  const { bbox, loading: bboxLoading } = useMapBBox(
    mapInstance,
    (newBBox: BBoxArray) => {
      // Convert from [minLng, minLat, maxLng, maxLat] to BBox object
      const bboxObj = {
        west: newBBox[0],
        south: newBBox[1],
        east: newBBox[2],
        north: newBBox[3],
      };
      updateBBox(bboxObj);
    },
    { debounceMs: 0 } // No debounce here, useClusteredData handles it
  );
  
  return (
    <div>
      <MapView
        ref={mapRef}
        onLoad={(map) => setMapInstance(map)}
      />
      {(bboxLoading || dataLoading) && (
        <div className="loading-indicator">Loading...</div>
      )}
      <div>Features: {data.features.length}</div>
    </div>
  );
}
```

## API Reference

### Hook Signature

```typescript
function useMapBBox(
  map: Map | null,
  onBBoxChange: (bbox: BBoxArray) => void,
  options?: UseMapBBoxOptions
): UseMapBBoxResult
```

### Parameters

#### `map`
- **Type**: `Map | null`
- **Description**: MapLibre GL Map instance. Can be `null` if not yet initialized.

#### `onBBoxChange`
- **Type**: `(bbox: BBoxArray) => void`
- **Description**: Callback function invoked after debounce with new bounding box.

#### `options`
- **Type**: `UseMapBBoxOptions` (optional)
- **Properties**:
  - `debounceMs` (number): Debounce delay in milliseconds. Default: `500`
  - `immediate` (boolean): Whether to call `onBBoxChange` immediately on mount if map has bounds. Default: `false`

### Return Value

```typescript
interface UseMapBBoxResult {
  bbox: BBoxArray | null;
  loading: boolean;
  error: string | null;
}
```

#### Properties

- **`bbox`**: Current bounding box in `[minLng, minLat, maxLng, maxLat]` format. `null` if map is not ready.
- **`loading`**: `true` when bbox change is pending (debouncing in progress)
- **`error`**: Error message if bbox computation failed. `null` otherwise.

### Types

```typescript
type BBoxArray = [number, number, number, number];

interface UseMapBBoxOptions {
  debounceMs?: number;
  immediate?: boolean;
}

interface UseMapBBoxResult {
  bbox: BBoxArray | null;
  loading: boolean;
  error: string | null;
}
```

## Behavior

### Event Handling

The hook listens to three MapLibre map events:

1. **`movestart`**: Fired when map movement begins. Sets `loading` to `true`.
2. **`move`**: Fired during map movement. Cancels any pending debounce timer.
3. **`moveend`**: Fired when map movement ends. Starts debounce timer.

### Debouncing Logic

```
User pans map → movestart (loading=true)
User continues panning → move (cancel timer)
User continues panning → move (cancel timer)
User stops panning → moveend (start timer)
[wait debounceMs milliseconds]
→ Compute bbox and call onBBoxChange
→ Set loading=false
```

If the user starts moving again before the debounce completes, the timer is cancelled and the process restarts.

### Acceptance Criteria

✅ **Rapid pans (≤300ms apart) result in single network call after settling**
- Multiple rapid movements cancel pending timers
- Only the final position triggers a callback

✅ **Hook returns consistent bbox format; no stale values after zoom**
- Always returns `[minLng, minLat, maxLng, maxLat]` format
- Bbox is updated synchronously with debounced callback

## Performance Considerations

- **Default debounce**: 500ms balances responsiveness with API efficiency
- **Recommended for high-frequency panning**: 300-500ms
- **Recommended for slower interactions**: 200-300ms
- **Loading state**: Use to show visual feedback during debounce

## Privacy & Security

The hook itself does not make network requests or handle sensitive data. However, when integrating with data fetching:

- **Bbox queries must not request precise coordinates** for non-consent entities (server responsibility)
- **Document expectations** in API integration code
- Use with privacy-aware utilities like `getDisplayCoordinates` from `utils/geojson`

## Testing

Comprehensive test coverage includes:

- Debounce behavior with rapid movements
- Bbox format consistency across pan/zoom
- Loading state tracking
- Event listener cleanup
- Error handling
- Timer cancellation on unmount

See `useMapBBox.test.ts` for test examples.

## Related

- **`useClusteredData`**: Hook for fetching and clustering scene/event data
- **`MapView`**: Privacy-first map component using MapLibre GL
- **`ClusteredMapView`**: Map component with built-in clustering
