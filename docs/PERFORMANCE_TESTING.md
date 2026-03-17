# Map Performance Testing Guide

This guide explains how to profile and test map rendering performance in the Subcults frontend.

## Overview

The map performance testing infrastructure includes:

1. **Performance Marks:** Instrumentation in production code that tracks timing
2. **Automated Tests:** Vitest tests that verify performance budgets
3. **Manual Profiling:** Chrome DevTools workflow for detailed analysis
4. **Mocking Utilities:** Functions to generate realistic datasets

## Performance Instrumentation

### Where Marks Are Added

Performance marks have been added to track the complete map render cycle:

#### `useClusteredData` Hook (Data Fetching)
- `data-fetch-{id}-start` - Beginning of data fetch
- `data-fetch-{id}-geojson-start` - Start GeoJSON conversion
- `data-fetch-{id}-geojson-end` - Complete GeoJSON conversion
- `data-fetch-{id}-end` - Complete data fetch cycle

**Console Output:**
```
[Performance] Data fetch: 215.30ms (GeoJSON build: 6.25ms)
```

#### `ClusteredMapView` Component (Map Operations)
- `map-init-{id}-start` - Start map initialization
- `map-init-{id}-end` - Complete source/layer setup
- `source-update-{id}-start` - Start source data update
- `source-update-{id}-end` - Complete source update
- `render-complete-{id}` - Layer render complete

**Console Output:**
```
[Performance] Map initialization: 45.20ms
[Performance] Map source update: 8.15ms (5000 features)
[Performance] Layer render complete
```

### Accessing Performance Data

#### In Browser Console

```javascript
// Get all performance marks
performance.getEntriesByType('mark')

// Get all performance measures
performance.getEntriesByType('measure')

// Get specific measurement
performance.getEntriesByName('data-fetch-123-total')

// Clear all marks/measures
performance.clearMarks()
performance.clearMeasures()
```

#### In Chrome DevTools

1. Open Performance panel
2. Record a session with map interactions
3. In the timeline, look for "User Timing" track
4. Expand to see custom performance marks and measures

## Automated Testing

### Running Performance Tests

```bash
# Run all performance tests
npm test -- --run geojson.perf.test.ts
npm test -- --run ClusteredMapView.perf.test.tsx

# Run with console output (see timing logs)
npm test -- --run geojson.perf.test.ts --reporter=verbose

# Run in watch mode (during development)
npm test -- geojson.perf.test.ts
```

### Test Structure

#### GeoJSON Performance Tests (`geojson.perf.test.ts`)

Tests the pure data transformation performance:

```typescript
// 5k entities should convert in <150ms
it('builds GeoJSON from 5000 scenes and events in <150ms', () => {
  const scenes = generateMockScenes(2500);
  const events = generateMockEvents(2500);
  
  const startTime = performance.now();
  const geojson = buildGeoJSON(scenes, events);
  const elapsed = performance.now() - startTime;
  
  expect(elapsed).toBeLessThan(150);
});
```

#### Map Performance Tests (`ClusteredMapView.perf.test.tsx`)

Tests the complete map render cycle with mocked MapLibre:

```typescript
// Generates 5k points with privacy compliance
it('generates 5000+ points with privacy compliance', () => {
  const scenes = generateMockScenes(2500, 50); // 50% precise
  const events = generateMockEvents(2500, 33); // 33% precise
  
  // Verify privacy distribution
  expect(scenes.filter(s => s.allow_precise).length).toBeGreaterThan(1000);
  
  // Verify all have coarse fallback
  expect(scenes.every(s => s.coarse_geohash)).toBe(true);
});
```

### Mock Data Generators

#### `generateMockScenes(count, precisePct)`

Generates scenes with configurable privacy settings:

```typescript
const scenes = generateMockScenes(1000, 50);
// Returns 1000 scenes, 50% with allow_precise=true

// Privacy-compliant structure:
// - 50% have precise_point defined
// - 100% have coarse_geohash fallback
// - Coordinates clustered around SF Bay Area
```

#### `generateMockEvents(count, precisePct)`

Generates events with configurable privacy settings:

```typescript
const events = generateMockEvents(1000, 33);
// Returns 1000 events, 33% with allow_precise=true

// Same privacy structure as scenes
// - Linked to mock scenes via scene_id
```

## Manual Profiling

### Chrome Performance Panel

**Best for:** FPS tracking, identifying jank, measuring total latency

**Steps:**
1. Open DevTools → Performance tab
2. Enable Screenshots and Memory
3. Click Record (or Cmd+E / Ctrl+E)
4. Interact with map:
   - Pan across map (test FPS during movement)
   - Zoom in/out (test cluster expansion)
   - Click markers (test detail panel)
5. Stop recording after 5-10 seconds
6. Analyze results:
   - FPS chart should stay ≥50fps
   - Main thread should have minimal long tasks
   - User Timing shows custom performance marks

**What to Look For:**
- ❌ FPS drops below 50 during panning
- ❌ Long tasks (>50ms) in scripting
- ❌ Layout thrashing (forced reflows)
- ✅ Smooth rendering at 60fps
- ✅ Quick source updates (<50ms)

### React DevTools Profiler

**Best for:** Identifying unnecessary React re-renders

**Steps:**
1. Install React DevTools extension
2. Open DevTools → Profiler tab
3. Click "Start profiling" (or Cmd+Option+P)
4. Interact with map (pan, click markers)
5. Stop profiling
6. Review Flamegraph:
   - ClusteredMapView should render once per data change
   - MapView should NOT re-render (uses imperative API)
   - DetailPanel should render only on open/close

**What to Look For:**
- ❌ MapView re-rendering on every data update
- ❌ Multiple consecutive renders of same component
- ❌ Expensive computations in render (should be memoized)
- ✅ Minimal render time (<30ms per component)
- ✅ Callbacks properly memoized with useCallback

### Performance Monitor (Real-time FPS)

**Best for:** Continuous FPS monitoring during interaction

**Steps:**
1. Open DevTools → More tools → Performance monitor
2. Watch FPS, CPU usage, and memory in real-time
3. Interact with map and observe metrics

**Targets:**
- FPS: 50-60fps during panning
- CPU: <30% on modern hardware
- Memory: No continuous growth (leak detection)

## Performance Budgets

### Current Targets

| Operation | Target | Critical |
|-----------|--------|----------|
| Data Fetch (5k bbox) | <400ms | <600ms |
| GeoJSON Build (5k) | <150ms | <200ms |
| GeoJSON Build (10k) | <300ms | <400ms |
| Source Update | <50ms | <100ms |
| Cluster Render | <200ms | <400ms |
| FPS (panning) | ≥50fps | ≥30fps |

### Tested Scenarios

**5k Point Dataset (Baseline):**
- 2,500 scenes (50% precise, 50% coarse)
- 2,500 events (33% precise, 67% coarse)
- Mixed distribution across SF Bay Area
- Current performance: ✅ All targets met

**10k Point Dataset (Stress Test):**
- 5,000 scenes
- 5,000 events
- Same privacy distribution
- Current performance: ✅ All targets met

## Optimization Checklist

Use this checklist when performance issues are detected:

### Data Fetching
- [ ] Verify debounce is working (300ms delay)
- [ ] Check AbortController cancels previous requests
- [ ] Ensure parallel fetching (not sequential)
- [ ] Monitor network waterfall in DevTools

### GeoJSON Conversion
- [ ] Profile buildGeoJSON with large datasets
- [ ] Verify jitter calculation is deterministic (cached)
- [ ] Check privacy enforcement isn't duplicated
- [ ] Consider WebWorker for >10k points

### React Re-renders
- [ ] Use React Profiler to identify unnecessary renders
- [ ] Verify useCallback wraps all event handlers
- [ ] Check dependencies array in useCallback/useMemo
- [ ] Consider React.memo for expensive components
- [ ] Use refs for values that don't need re-render

### MapLibre Rendering
- [ ] Verify cluster configuration (radius: 50, maxZoom: 14)
- [ ] Check layer filters are expression-based (not JS)
- [ ] Ensure imperative setData (not prop-based)
- [ ] Monitor memory usage for map tiles
- [ ] Test on lower-end hardware/mobile

## Common Issues

### Issue: Data fetch exceeds 400ms

**Likely Causes:**
- Network latency (especially on mobile)
- Backend query performance (missing indexes)
- Large bbox returning >10k points

**Solutions:**
- Add loading skeleton/progressive loading
- Implement result pagination on backend
- Cache recent queries in localStorage/IndexedDB

### Issue: FPS drops below 50 during panning

**Likely Causes:**
- Too many points rendered (>15k)
- Inefficient layer styles (JS expressions)
- React re-rendering on every frame

**Solutions:**
- Increase cluster radius (reduces render load)
- Use MapLibre expressions for styles (WebGL-based)
- Memoize event handlers to prevent re-creates
- Profile with Performance panel to find bottleneck

### Issue: Memory usage grows continuously

**Likely Causes:**
- Event listeners not cleaned up
- Cached entities never evicted
- MapLibre tiles not garbage collected

**Solutions:**
- Verify useEffect cleanup functions
- Implement LRU cache for entities (max 1000 items)
- Clear performance marks/measures periodically

## References

- [Performance API (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Performance_API)
- [Chrome DevTools Performance](https://developer.chrome.com/docs/devtools/performance/)
- [React DevTools Profiler](https://react.dev/reference/react/Profiler)
- [MapLibre Performance](https://maplibre.org/maplibre-gl-js/docs/API/)
- [Web Performance Best Practices](https://web.dev/fast/)

---

**Last Updated:** 2024-12-08
**Maintained By:** Frontend Team
