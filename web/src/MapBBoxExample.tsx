import { useRef, useState } from 'react';
import { MapView, type MapViewHandle } from './components/MapView';
import { useMapBBox, type BBoxArray } from './hooks/useMapBBox';
import type { Map } from 'maplibre-gl';

/**
 * Example component demonstrating useMapBBox integration
 * 
 * This component shows how to use the useMapBBox hook to track
 * map bounding box changes with debouncing.
 */
function MapBBoxExample() {
  const mapRef = useRef<MapViewHandle>(null);
  const [mapInstance, setMapInstance] = useState<Map | null>(null);
  const [fetchedData, setFetchedData] = useState<string[]>([]);
  
  const handleBBoxChange = async (bbox: BBoxArray) => {
    console.log('Bbox changed:', bbox);
    
    // Simulate data fetch
    const timestamp = new Date().toISOString();
    // Keep last 5 entries (including the new one)
    setFetchedData(prev => {
      const newData = [...prev, `Fetched data for bbox: [${bbox.map(n => n.toFixed(4)).join(', ')}] at ${timestamp}`];
      return newData.slice(-5);
    });
  };
  
  const { bbox, loading, error } = useMapBBox(
    mapInstance,
    handleBBoxChange,
    { 
      debounceMs: 500,
      immediate: false,
    }
  );
  
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      {/* Info Panel */}
      <div style={{ 
        padding: '1rem', 
        backgroundColor: '#f5f5f5', 
        borderBottom: '1px solid #ddd' 
      }}>
        <h2 style={{ margin: '0 0 1rem 0' }}>useMapBBox Demo</h2>
        
        <div style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: '0.5rem 1rem' }}>
          <strong>Status:</strong>
          <span>{loading ? '⏳ Loading...' : '✓ Ready'}</span>
          
          <strong>Current BBox:</strong>
          <span style={{ fontFamily: 'monospace', fontSize: '0.9em' }}>
            {bbox ? `[${bbox.map(n => n.toFixed(4)).join(', ')}]` : 'null'}
          </span>
          
          <strong>Error:</strong>
          <span style={{ color: error ? 'red' : 'green' }}>
            {error || 'None'}
          </span>
        </div>
      </div>
      
      {/* Map Container */}
      <div style={{ flex: 1, position: 'relative' }}>
        <MapView
          ref={mapRef}
          onLoad={(map) => {
            console.log('Map loaded');
            setMapInstance(map);
          }}
          initialPosition={{
            center: [-122.4194, 37.7749], // San Francisco
            zoom: 12,
          }}
        />
        
        {/* Loading Overlay */}
        {loading && (
          <div style={{
            position: 'absolute',
            top: '1rem',
            right: '1rem',
            backgroundColor: 'rgba(0, 0, 0, 0.8)',
            color: 'white',
            padding: '0.5rem 1rem',
            borderRadius: '0.25rem',
            zIndex: 1000,
          }}>
            Loading new data...
          </div>
        )}
      </div>
      
      {/* Fetch Log */}
      <div style={{ 
        padding: '1rem', 
        backgroundColor: '#fff', 
        borderTop: '1px solid #ddd',
        maxHeight: '200px',
        overflow: 'auto',
      }}>
        <h3 style={{ margin: '0 0 0.5rem 0', fontSize: '1rem' }}>Fetch Log</h3>
        {fetchedData.length === 0 ? (
          <p style={{ margin: 0, color: '#999' }}>
            Pan or zoom the map to trigger bbox changes...
          </p>
        ) : (
          <ul style={{ margin: 0, padding: '0 0 0 1.5rem', fontSize: '0.85em' }}>
            {fetchedData.map((entry, idx) => (
              <li key={idx} style={{ marginBottom: '0.25rem' }}>{entry}</li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

export default MapBBoxExample;
