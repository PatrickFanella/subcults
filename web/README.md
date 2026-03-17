# Subcults Web Frontend

React + TypeScript + Vite frontend for the Subcults platform.

## Features

- **MapLibre GL** integration with MapTiler tiles
- **Scene & Event Clustering** for scalable map visualization
- **Privacy-first** geolocation (opt-in, coarse accuracy)
- **Privacy-enforced** location data (respects `allow_precise` flag)
- **Internationalization (i18n)** with React i18next, namespace separation, and lazy loading
- Responsive map with ResizeObserver
- TypeScript for type safety
- Vitest for testing

## Setup

1. Install dependencies:
   ```bash
   npm install
   ```

2. Configure environment variables:
   ```bash
   cp .env.example .env
   # Edit .env and add your VITE_MAPTILER_API_KEY
   # Optional: VITE_API_URL (default: /api)
   ```

3. Run development server:
   ```bash
   npm run dev
   ```

## Scripts

- `npm run dev` - Start development server with HMR
- `npm run build` - Build for production
- `npm run preview` - Preview production build locally
- `npm run test` - Run unit tests
- `npm run test:ui` - Run tests with Vitest UI
- `npm run test:coverage` - Generate coverage report
- `npm run lint` - Run ESLint
- `npm run check:i18n` - Validate translation keys (CI check)

## Accessibility

Subcults is designed to meet **WCAG 2.1 Level AA** accessibility standards.

- **Automated Testing**: Comprehensive axe-core test suite (28 tests, 0 violations)
- **Keyboard Navigation**: Full keyboard accessibility across all components
- **Screen Reader Support**: Proper ARIA labels and semantic HTML
- **Mobile Accessibility**: Primary touch targets designed to meet 44x44px minimum
- **Focus Management**: Proper focus handling in modals and dialogs
- **Color Contrast**: WCAG AA compliant (4.5:1 for text, 3:1 for UI)

See [ACCESSIBILITY.md](ACCESSIBILITY.md) for comprehensive accessibility documentation and [A11Y_CHECKLIST.md](A11Y_CHECKLIST.md) for the component development checklist.

```bash
# Run accessibility tests
npm test -- src/a11y-audit.test.tsx
```

## Components

### ClusteredMapView (Recommended)

The `ClusteredMapView` component provides an enhanced map with automatic scene/event clustering:

```tsx
import { ClusteredMapView } from './components/ClusteredMapView';

function App() {
  return (
    <ClusteredMapView
      enableGeolocation={false}  // Privacy: opt-in only
      initialPosition={{
        center: [-122.4194, 37.7749],
        zoom: 12
      }}
      onLoad={(map) => console.log('Map with clustering loaded')}
    />
  );
}
```

**Features:**
- Automatic data fetching based on map bounds
- Privacy-enforced clustering (respects location consent)
- Click-to-expand cluster functionality
- Separate visual styling for scenes vs events
- Debounced updates on pan/zoom (300ms)
- High performance: <10ms for 10k entities

See [CLUSTERING.md](src/components/CLUSTERING.md) for detailed documentation.

### MapView (Base Component)

The `MapView` component is a privacy-conscious wrapper around MapLibre GL:

```tsx
import { useRef } from 'react';
import { MapView, type MapViewHandle } from './components/MapView';

function App() {
  const mapRef = useRef<MapViewHandle>(null);

  const flyToLocation = () => {
    mapRef.current?.flyTo([-122.4194, 37.7749], 14);
  };

  return (
    <MapView
      ref={mapRef}
      enableGeolocation={false}  // Privacy: opt-in only
      onLoad={(map) => console.log('Map loaded')}
    />
  );
}
```

### Props

#### ClusteredMapView Props
All MapView props plus:
- Standard MapView props (see below)

#### MapView Props
- `apiKey?: string` - MapTiler API key (or use VITE_MAPTILER_API_KEY env var)
- `initialPosition?: { bounds?, center?, zoom? }` - Initial map position
- `enableGeolocation?: boolean` - Enable geolocation fallback (default: false)
- `className?: string` - CSS class for map container
- `onLoad?: (map) => void` - Callback when map loads
- `onGeolocationSuccess?: (position) => void` - Geolocation success callback
- `onGeolocationError?: (error) => void` - Geolocation error callback

### Imperative Methods (via ref)

- `getMap()` - Get underlying MapLibre map instance
- `flyTo(center, zoom?)` - Animate to location
- `getBounds()` - Get current map bounds

## Hooks

### useClusteredData

React hook for fetching scene/event data based on map bounds:

```tsx
import { useClusteredData } from './hooks/useClusteredData';

function CustomMap() {
  const { data, loading, error, updateBBox } = useClusteredData(null, {
    apiUrl: '/api',
    debounceMs: 300
  });

  // data: GeoJSON FeatureCollection
  // updateBBox: (bbox) => void
  // loading: boolean
  // error: string | null
}
```

## Utilities

### GeoJSON Builder

Convert scenes/events to GeoJSON with privacy enforcement:

```tsx
import { buildGeoJSON } from './utils/geojson';
import type { Scene, Event } from './types/scene';

const scenes: Scene[] = [...];
const events: Event[] = [...];

const geojson = buildGeoJSON(scenes, events);
// Returns GeoJSON FeatureCollection
```

### Geohash Decoder

Decode geohash strings to approximate coordinates:

```tsx
import { decodeGeohash } from './utils/geojson';

const coords = decodeGeohash('9q8yy');
// Returns: { lat: 37.77..., lng: -122.42... }
```

## Privacy Considerations

- Geolocation is **opt-in** via `enableGeolocation` prop
- When enabled, uses `enableHighAccuracy: false` for coarse location
- Scene/event locations respect `allow_precise` flag:
  - `true`: Use exact `precise_point` coordinates
  - `false` (Scenes): Use approximate `coarse_geohash` coordinates (~1km precision)
  - `false` (Events): Use approximate `coarse_geohash` coordinates if available
- Events should include `coarse_geohash` field for privacy-compliant display
- MapTiler API key is client-side (acceptable for public tile access)
- Document key rotation procedure in production

## Testing

Tests use Vitest with React Testing Library:

```bash
npm test
```

Test suites:
- **MapView** - Base map component functionality
- **ClusteredMapView** - Clustering integration
- **useClusteredData** - Data fetching hook
- **geojson** - GeoJSON builder and privacy enforcement
- **geojson.perf** - Performance benchmarks (5k-10k entities)

Coverage target: >70% for frontend code

## Performance

The clustering system meets strict performance requirements:

| Operation                      | Target  | Actual |
|--------------------------------|---------|--------|
| Build GeoJSON (5k entities)    | <150ms  | ~6ms   |
| Build GeoJSON (10k entities)   | <300ms  | ~7ms   |
| Pan/zoom update (5k points)    | <150ms  | ✅     |

## Build Output

Production build creates optimized static assets in `dist/`:

```bash
npm run build
```

The build output is ready to be served by Caddy, nginx, or any static file server.

## Demo

See `src/ClusteredMapDemo.tsx` for a complete example application demonstrating:
- ClusteredMapView usage
- Map navigation controls
- Privacy-first configuration
- Cluster expansion interactions

## Architecture

```
web/src/
├── components/
│   ├── MapView.tsx              # Base map component
│   ├── ClusteredMapView.tsx     # Clustering-enabled map
│   ├── CLUSTERING.md            # Clustering documentation
│   └── *.test.tsx               # Component tests
├── hooks/
│   └── useClusteredData.ts      # Data fetching hook
├── utils/
│   ├── geojson.ts               # GeoJSON builder
│   └── *.test.ts                # Utility tests
├── types/
│   └── scene.ts                 # TypeScript types
└── clustering.ts                # Public API exports
```

## Internationalization (i18n)

Subcults supports multiple languages with automatic detection and lazy loading:

- **Supported Languages**: English (en), Spanish (es)
- **Namespace Organization**: common, scenes, events, streaming, auth
- **Lazy Loading**: Translations loaded on-demand via HTTP backend
- **Language Detection**: User preference → Browser language → Fallback to English

See [docs/I18N.md](docs/I18N.md) for complete documentation on:
- Using translations in components
- Adding new translations
- Adding new languages
- Translation validation

Quick Example:
```tsx
import { useT } from './hooks/useT';
import { useLanguageActions } from './stores/languageStore';

function MyComponent() {
  const { t } = useT('scenes');
  const { setLanguage } = useLanguageActions();
  
  return (
    <div>
      <h1>{t('title')}</h1>
      <button onClick={() => setLanguage('es')}>Español</button>
    </div>
  );
}
```

