# Location Jitter Utility

## Overview

The jitter utility provides privacy-preserving location visualization by applying deterministic random offsets to coarse (non-precise) coordinates. This prevents exact location tracking while maintaining consistent visualization across sessions.

## Privacy Principles

1. **Deterministic per entity**: Same entity ID always produces the same offset
2. **Session-consistent**: No flickering on re-renders
3. **Bounded radius**: Offsets stay within configured maximum (default 250m)
4. **Privacy-aware**: Only applied to coarse locations (when `allow_precise=false`)
5. **Reversible decision**: Jitter can be toggled on/off via `enableJitter` parameter

## Key Functions

### `calculateJitterOffset(entityId, radiusMeters)`

Generates deterministic offset values for an entity.

**Parameters:**
- `entityId` (string): Unique entity identifier
- `radiusMeters` (number, optional): Maximum jitter radius (default: 250m)

**Returns:**
```typescript
{
  latOffset: number;  // Offset in degrees latitude
  lngOffset: number;  // Offset in degrees longitude (before latitude scaling)
}
```

**Example:**
```typescript
const offset = calculateJitterOffset('scene-123', 250);
// Always returns same offset for 'scene-123'
```

### `applyJitter(lat, lng, entityId, radiusMeters)`

Applies jitter offset to coordinates with latitude scaling.

**Parameters:**
- `lat` (number): Original latitude
- `lng` (number): Original longitude  
- `entityId` (string): Unique entity identifier
- `radiusMeters` (number, optional): Maximum jitter radius (default: 250m)

**Returns:**
```typescript
{
  lat: number;  // Jittered latitude
  lng: number;  // Jittered longitude
}
```

**Example:**
```typescript
const jittered = applyJitter(37.7749, -122.4194, 'scene-sf', 250);
// Returns consistently jittered coordinates
```

### `shouldApplyJitter(allowPrecise)`

Determines if jitter should be applied based on privacy consent.

**Parameters:**
- `allowPrecise` (boolean): Whether precise location is allowed

**Returns:** `boolean` - True if jitter should be applied (when precise is NOT allowed)

**Example:**
```typescript
if (shouldApplyJitter(scene.allow_precise)) {
  coords = applyJitter(coords.lat, coords.lng, scene.id);
}
```

### `haversineDistance(lat1, lng1, lat2, lng2)`

Calculates great-circle distance between two points in meters.

**Parameters:**
- `lat1`, `lng1`: First point coordinates
- `lat2`, `lng2`: Second point coordinates

**Returns:** Distance in meters

**Example:**
```typescript
const distance = haversineDistance(37.7749, -122.4194, 37.7849, -122.4094);
// Returns ~1540 meters
```

## Algorithm Details

### Hash Function

Uses FNV-1a hash algorithm for fast, deterministic pseudo-random number generation:

1. Start with FNV offset basis (2166136261)
2. For each character: XOR with character code, multiply by FNV prime (16777619)
3. Convert to unsigned 32-bit integer

Two independent hashes are generated per entity (using suffixes `-lat` and `-lng`) to ensure independent x/y offsets.

### Offset Distribution

Polar coordinate approach ensures uniform distribution within circular area:

1. **Angle**: Hash value normalized to [0, 2π)
2. **Radius**: Square root of normalized hash (uniform area distribution) * max radius
3. **Cartesian**: Convert polar to x/y offsets in meters
4. **Latitude scaling**: Longitude offset scaled by `cos(latitude)` to maintain consistent distance

### Why This Approach?

- **Deterministic**: Same input → same output (no random seed needed)
- **Uniform**: Points evenly distributed within circle (not clustered at center)
- **Fast**: Simple hash computation, no complex crypto
- **Privacy-preserving**: Cannot reverse engineer original coordinates
- **Bounded**: Guaranteed within specified radius

## Integration with GeoJSON

The `buildGeoJSON` function automatically applies jitter:

```typescript
const geojson = buildGeoJSON(scenes, events, enableJitter = true);
```

Features with jittered coordinates include `is_jittered: true` in properties.

## Visual Indicators

### Map Markers

Jittered markers have subtle visual differences:
- **Opacity**: 0.8 (vs 1.0 for precise locations)
- **Tooltip**: Shows "📍 Approximate location (privacy preserved)" on hover
- **Consistent**: Same marker always appears at same offset

### Example

```typescript
// Scene with coarse location
const scene = {
  id: 'scene-123',
  name: 'Underground Venue',
  allow_precise: false,
  coarse_geohash: '9q8yy'
};

// buildGeoJSON applies jitter automatically
const geojson = buildGeoJSON([scene], []);

// Feature properties include:
// is_jittered: true
// Tooltip shows privacy notice on hover
```

## Testing

### Unit Tests

17 comprehensive tests covering:
- Deterministic offset generation
- Distribution uniformity
- Radius bounds enforcement
- Latitude scaling
- Distance calculations

### Integration Tests

22 tests in geojson.test.ts covering:
- Jitter application to coarse locations
- No jitter for precise locations
- Consistency across calls
- enableJitter toggle

### Performance

- 10k entities processed in <20ms (including jitter)
- Hash computation: ~0.001ms per entity
- No performance impact on map rendering

## Configuration

### Default Values

```typescript
export const DEFAULT_JITTER_RADIUS_METERS = 250;
```

### Customization

```typescript
// Custom radius
const jittered = applyJitter(lat, lng, id, 100); // 100m radius

// Disable jitter
const geojson = buildGeoJSON(scenes, events, false);
```

## Privacy Considerations

### What Jitter Does

✅ Prevents exact location tracking  
✅ Maintains regional discovery  
✅ Consistent visualization per session  
✅ Visual privacy indicator  

### What Jitter Doesn't Do

❌ Does not replace consent enforcement (that's in backend)  
❌ Does not modify stored coordinates (display-only)  
❌ Does not prevent clustering analysis (coarse geohash still available)  

### Best Practices

1. **Always enforce consent**: Jitter is defense-in-depth, not primary protection
2. **Backend validation**: Server must enforce `allow_precise` flag
3. **Clear communication**: Tooltips explain approximate locations
4. **Audit logging**: Track when precise locations are accessed
5. **Debug mode**: Provide QA toggle to view raw coordinates

## Future Enhancements

### Debug Toggle

Planned: Developer mode to show both raw and jittered coordinates for QA.

```typescript
<ClusteredMapView debugMode={true} />
```

This would:
- Show raw coordinates in debug panel
- Toggle between raw/jittered views
- Highlight privacy boundaries
- Export jitter offsets for testing

### Geohash Boundary Enforcement

Ensure jittered coordinates stay within original coarse geohash cell:

```typescript
function constrainToGeohashBounds(coords, geohash) {
  const bounds = decodeGeohashBounds(geohash);
  return {
    lat: clamp(coords.lat, bounds.latMin, bounds.latMax),
    lng: clamp(coords.lng, bounds.lngMin, bounds.lngMax),
  };
}
```

## References

- [FNV-1a Hash Algorithm](http://www.isthe.com/chongo/tech/comp/fnv/)
- [Haversine Distance Formula](https://en.wikipedia.org/wiki/Haversine_formula)
- [Geohash Precision](https://en.wikipedia.org/wiki/Geohash#Digits_and_precision_in_km)
- [MapLibre Popup API](https://maplibre.org/maplibre-gl-js/docs/API/classes/Popup/)

## Support

For issues or questions:
- Check test files: `jitter.test.ts`, `geojson.test.ts`
- Review privacy tests: `internal/scene/privacy_test.go`
- See clustering docs: `web/src/components/CLUSTERING.md`
