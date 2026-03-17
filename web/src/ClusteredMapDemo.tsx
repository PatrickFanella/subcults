import { useRef } from 'react';
import { ClusteredMapView } from './components/ClusteredMapView';
import type { MapViewHandle } from './components/MapView';

/**
 * Demo application showing ClusteredMapView usage
 *
 * To use this demo:
 * 1. Replace App.tsx content with this file
 * 2. Set VITE_MAPTILER_API_KEY in your .env file
 * 3. Ensure backend API is running with /api/scenes and /api/events endpoints
 * 4. Run `npm run dev`
 */
function ClusteredMapDemo() {
  const mapRef = useRef<MapViewHandle>(null);

  const handleFlyToSF = () => {
    if (mapRef.current) {
      mapRef.current.flyTo([-122.4194, 37.7749], 14);
    }
  };

  const handleFlyToNYC = () => {
    if (mapRef.current) {
      mapRef.current.flyTo([-74.006, 40.7128], 14);
    }
  };

  const handleFlyToLA = () => {
    if (mapRef.current) {
      mapRef.current.flyTo([-118.2437, 34.0522], 14);
    }
  };

  return (
    <div className="flex flex-col h-screen">
      <div className="p-4 bg-brand-underground text-foreground">
        <h1 className="text-2xl font-bold">Subcults - Clustered Scene &amp; Event Map</h1>
        <p className="mt-2 text-sm text-foreground-muted">
          Interactive map with privacy-first clustering of underground music scenes and events
        </p>
        <div className="flex gap-2 mt-2">
          <button
            onClick={handleFlyToSF}
            className="px-3 py-1 text-sm transition-none border rounded-none bg-background border-border text-foreground hover:bg-background-hover"
          >
            San Francisco
          </button>
          <button
            onClick={handleFlyToNYC}
            className="px-3 py-1 text-sm transition-none border rounded-none bg-background border-border text-foreground hover:bg-background-hover"
          >
            New York
          </button>
          <button
            onClick={handleFlyToLA}
            className="px-3 py-1 text-sm transition-none border rounded-none bg-background border-border text-foreground hover:bg-background-hover"
          >
            Los Angeles
          </button>
        </div>
        <div className="mt-2 text-xs text-foreground-muted">
          Tip: Click clusters to expand &bull; Click markers for details &bull; Blue = Scenes &bull;
          Pink = Events
        </div>
      </div>
      <div className="relative flex-1">
        <ClusteredMapView
          ref={mapRef}
          enableGeolocation={false}
          onLoad={(map) => {
            console.log('Clustered map loaded:', map);
            console.log('Clustering configuration:', {
              clusterRadius: 50,
              clusterMaxZoom: 14,
              privacyEnforced: true,
            });
          }}
          initialPosition={{
            center: [-122.4194, 37.7749],
            zoom: 12,
          }}
        />
      </div>
    </div>
  );
}

export default ClusteredMapDemo;
