# Performance Benchmarks & Optimization Guide

This document tracks performance baselines, budgets, and optimization strategies for the Subcults platform.

## Table of Contents
- [Performance Budgets](#performance-budgets)
- [Map Performance](#map-performance)
- [Backend Performance](#backend-performance)
- [Profiling Tools](#profiling-tools)
- [Optimization Strategies](#optimization-strategies)

## Performance Budgets

High-level performance targets for the application:

| Metric | Target | Critical Threshold |
|--------|--------|-------------------|
| API Latency (p95) | <300ms | <500ms |
| Stream Join | <2s | <5s |
| Map Render | <1.2s | <2s |
| First Contentful Paint (FCP) | <1.0s | <1.5s |
| Trust Recompute | <5m | <10m |

## Map Performance

### Overview

The map rendering system is critical for user experience, handling clustering, real-time data updates, and privacy-aware location display. Performance monitoring tracks data fetch, GeoJSON conversion, and render cycles.

### Baseline Metrics

**Test Environment:**
- Browser: Chrome 120+
- Dataset: 5,000-10,000 points (scenes + events)
- Privacy Mix: 50% precise, 50% coarse (with jitter)

**Measured Performance (as of 2024-12-08):**

| Operation | Dataset Size | Baseline | Target | Status |
|-----------|--------------|----------|--------|---------|
| Data Fetch | 5k bbox | ~150-250ms | <400ms | ✅ Pass |
| GeoJSON Build | 5k entities | ~4-7ms | <150ms | ✅ Pass |
| GeoJSON Build | 10k entities | ~8-15ms | <300ms | ✅ Pass |
| Source Update | 5k features | ~5-10ms | <50ms | ✅ Pass |
| Cluster Render | 5k features | ~50-100ms | <200ms | ✅ Pass |
| FPS (panning) | 5k features | 50-60fps | ≥50fps | ✅ Pass |

### Cluster Configuration

Current cluster settings optimized for balance between performance and UX:

```javascript
{
  cluster: true,
  clusterRadius: 50,    // Pixels - affects grouping density
  clusterMaxZoom: 14,   // Zoom level where clustering stops
}
```

**Rationale:**
- `clusterRadius: 50` - Provides good visual grouping without excessive overlap
- `clusterMaxZoom: 14` - Shows individual points at neighborhood zoom level
- Trade-off: Larger radius = fewer clusters = better performance, but less granular control

### Performance Marks

The map implementation uses Performance API marks to track critical operations:

#### Data Fetch Lifecycle
```
data-fetch-{id}-start
  → API request to /scenes and /events
data-fetch-{id}-geojson-start
  → Start GeoJSON conversion
data-fetch-{id}-geojson-end
  → Complete GeoJSON conversion
data-fetch-{id}-end
  → Complete data fetch cycle
```

#### Map Operations
```
map-init-{id}-start
  → Start map initialization
map-init-{id}-end
  → Complete source/layer setup

source-update-{id}-start
  → Start MapLibre source.setData()
source-update-{id}-end
  → Complete source update

render-complete-{id}
  → First frame after layer update
```

### Chrome DevTools Profiling

**How to Profile Map Performance:**

1. **Open Performance Panel:**
   - Open Chrome DevTools (F12)
   - Navigate to "Performance" tab
   - Enable "Screenshots" and "Memory" options

2. **Record a Session:**
   - Click record button
   - Perform map interactions:
     - Pan across large dataset (5k+ points)
     - Zoom in/out through cluster levels
     - Click markers to open detail panel
   - Stop recording after 5-10 seconds

3. **Analyze Results:**
   - **FPS Chart:** Should maintain ≥50fps during panning
   - **Main Thread:** Look for long tasks (>50ms)
   - **Network:** Verify data fetch <400ms
   - **Memory:** Check for memory leaks on repeated interactions

4. **Key Metrics to Check:**
   - Scripting time per frame (<10ms ideal)
   - Paint/composite time (<16ms for 60fps)
   - React component render time
   - MapLibre layer update time

**Expected Profile (5k points):**
```
Data Fetch:     150-250ms  ✓
GeoJSON Build:    4-7ms    ✓
Source Update:    5-10ms   ✓
Layer Render:    50-100ms  ✓
Total Latency:  ~200-370ms ✓ (<400ms target)
```

### React DevTools Profiler

**Identifying Re-render Issues:**

1. **Enable Profiler:**
   - Install React DevTools extension
   - Open DevTools → Profiler tab
   - Click "Start profiling"

2. **Record Map Interactions:**
   - Pan map (triggers bbox updates)
   - Hover markers (triggers tooltip rendering)
   - Click markers (opens detail panel)
   - Stop profiling

3. **Analyze Flamegraph:**
   - **Expected behavior:**
     - ClusteredMapView: ~20-30ms on data update
     - DetailPanel: ~10-15ms on open/close
     - MapView: No re-render on data changes (uses imperative API)
   
   - **Red flags (unnecessary re-renders):**
     - MapView re-rendering on every data fetch
     - Multiple DetailPanel renders on single interaction
     - Callbacks recreating on every render

4. **Memoization Status:**
   - ✅ `useCallback` for event handlers (fetchEntityDetails, handleMarkerClick, etc.)
   - ✅ `useCallback` for tooltip display (showPrivacyTooltip)
   - ✅ `useRef` for stable map instance and popup references
   - ✅ `forwardRef` for parent component access
   - ⚠️ Consider `React.memo` for DetailPanel if props are stable

### Optimization Strategies

#### Data Fetching
- **Debouncing:** 300ms debounce on bbox changes (prevents rapid-fire requests)
- **Request Cancellation:** AbortController cancels pending fetches on new bbox
- **Parallel Fetching:** Scenes and events fetched concurrently with Promise.all()

#### GeoJSON Conversion
- **Jitter Caching:** Deterministic jitter uses entity ID as seed (same coords per entity)
- **Minimal Properties:** Only include necessary properties in GeoJSON features
- **Privacy Enforcement:** Coordinate resolution happens once during conversion

#### React Re-renders
- **Imperative Map API:** MapLibre instance updated directly, not via props
- **Callback Memoization:** Event handlers wrapped in useCallback with stable dependencies
- **Ref-based State:** Map instance and popups stored in refs, not state

#### MapLibre Rendering
- **Clustering:** Reduces render load by grouping nearby points
- **Layer Filters:** Separate layers for scenes/events avoid dynamic style switching
- **Expression-based Styles:** MapLibre evaluates styles in WebGL, not JavaScript

### Regression Testing

**Automated Performance Tests:**

Run with: `npm run test -- ClusteredMapView.perf`

- ✅ Generates 5k+ points with privacy compliance
- ✅ Converts large datasets to GeoJSON <150ms (5k) / <300ms (10k)
- ✅ Tracks performance marks for operations
- ✅ Validates cluster configuration (radius: 50, maxZoom: 14)

**Manual Profiling Checklist:**

Before major releases, perform manual profiling with:
- [ ] 5k point dataset (mixed precise/coarse)
- [ ] 10k point dataset (stress test)
- [ ] Chrome Performance panel recording
- [ ] React DevTools Profiler recording
- [ ] FPS verification during pan/zoom
- [ ] Memory leak check (repeat interactions)

### Known Limitations

1. **Large Bbox Queries:** Fetching entire world bbox (>100k points) not optimized
   - Mitigation: Backend should limit query results or use pagination
   
2. **Mobile Performance:** Lower-end devices may struggle with 10k+ points
   - Mitigation: Consider adaptive clustering (larger radius on mobile)
   
3. **Network Latency:** Slow connections exceed 400ms target
   - Mitigation: Add loading skeleton, cache recent queries

4. **Initial Load:** First render includes map tile loading (not measured)
   - Mitigation: Prefetch tiles for common areas, use service worker

### Future Optimizations

**Potential improvements (not yet implemented):**

1. **WebWorker for GeoJSON:** Offload conversion to background thread
2. **Virtual Clustering:** Render only visible clusters, paginate rest
3. **Tile-based Caching:** Cache GeoJSON by map tile for instant pan
4. **Adaptive Clustering:** Adjust radius based on zoom level and point density
5. **GPU Acceleration:** Investigate MapLibre native layer performance

---

## Backend Performance

*Coming soon: API latency baselines, database query optimization, indexing strategies*

## Profiling Tools

### Browser Tools
- **Chrome DevTools Performance Panel:** CPU profiling, FPS tracking
- **React DevTools Profiler:** Component render timing
- **Performance API:** Programmatic performance marks/measures

### Backend Tools
- **Go pprof:** CPU and memory profiling
- **Prometheus:** Metrics collection and visualization
- **OpenTelemetry:** Distributed tracing

---

## Optimization Strategies

### General Principles
1. **Measure first:** Always profile before optimizing
2. **Target bottlenecks:** Focus on highest-impact areas (80/20 rule)
3. **Test regressions:** Automated performance tests prevent degradation
4. **Document decisions:** Record why optimizations were made

### Frontend
- Minimize re-renders with React.memo, useMemo, useCallback
- Lazy load non-critical components
- Use imperative APIs for heavy libraries (MapLibre, etc.)
- Debounce expensive operations (network, rendering)
- Leverage browser caching and service workers

### Backend
- Index database columns used in WHERE/JOIN/ORDER BY
- Use connection pooling for database access
- Cache frequently accessed data (Redis)
- Stream large responses instead of buffering
- Profile slow queries with EXPLAIN ANALYZE

---

**Last Updated:** 2024-12-08  
**Next Review:** 2025-01-08 (monthly cadence)
