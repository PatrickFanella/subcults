/**
 * HomePage Component
 * Main map view showing scenes and events
 */

import React, { Suspense, lazy, useRef } from 'react';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import type { MapViewHandle } from '../components/MapView';

const MapView = lazy(() =>
  import('../components/MapView').then((module) => ({ default: module.MapView }))
);

export const HomePage: React.FC = () => {
  // MapViewHandle ref - reserved for future map interactions (flyTo, etc.)
  const mapRef = useRef<MapViewHandle>(null);

  return (
    <div style={{ height: '100%', width: '100%' }}>
      <Suspense fallback={<LoadingSkeleton />}>
        <MapView
          ref={mapRef}
          enableGeolocation={false}
          onLoad={() => {
            // Map loaded successfully
          }}
        />
      </Suspense>
    </div>
  );
};
