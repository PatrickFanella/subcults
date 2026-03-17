import { describe, it, expect } from 'vitest';
import {
  calculateJitterOffset,
  applyJitter,
  shouldApplyJitter,
  haversineDistance,
  DEFAULT_JITTER_RADIUS_METERS,
} from './jitter';

describe('calculateJitterOffset', () => {
  it('generates consistent offsets for the same entity ID', () => {
    const entityId = 'scene-123';
    const offset1 = calculateJitterOffset(entityId);
    const offset2 = calculateJitterOffset(entityId);

    expect(offset1.latOffset).toBe(offset2.latOffset);
    expect(offset1.lngOffset).toBe(offset2.lngOffset);
  });

  it('generates different offsets for different entity IDs', () => {
    const offset1 = calculateJitterOffset('scene-123');
    const offset2 = calculateJitterOffset('scene-456');

    expect(offset1.latOffset).not.toBe(offset2.latOffset);
    expect(offset1.lngOffset).not.toBe(offset2.lngOffset);
  });

  it('generates offsets within reasonable bounds for default radius', () => {
    const entityId = 'scene-test';
    const { latOffset, lngOffset } = calculateJitterOffset(entityId);

    // With 250m radius, max offset should be ~0.0022 degrees latitude
    // (250m / 111320 m/degree ≈ 0.00224)
    const maxDegrees = (DEFAULT_JITTER_RADIUS_METERS / 111320) * 1.5; // Add margin for longitude

    expect(Math.abs(latOffset)).toBeLessThan(maxDegrees);
    expect(Math.abs(lngOffset)).toBeLessThan(maxDegrees);
  });

  it('respects custom radius parameter', () => {
    const entityId = 'scene-test';
    const smallRadius = 50; // 50 meters
    const { latOffset, lngOffset } = calculateJitterOffset(entityId, smallRadius);

    // With 50m radius, max offset should be ~0.00045 degrees
    const maxDegrees = (smallRadius / 111320) * 1.5;

    expect(Math.abs(latOffset)).toBeLessThan(maxDegrees);
    expect(Math.abs(lngOffset)).toBeLessThan(maxDegrees);
  });

  it('produces deterministic results across multiple calls', () => {
    const entityId = 'deterministic-test';
    const calls = Array.from({ length: 100 }, () => calculateJitterOffset(entityId));

    // All calls should produce identical results
    const firstCall = calls[0];
    calls.forEach((call) => {
      expect(call.latOffset).toBe(firstCall.latOffset);
      expect(call.lngOffset).toBe(firstCall.lngOffset);
    });
  });

  it('distributes offsets across all quadrants', () => {
    // Test that different IDs produce offsets in all four quadrants
    const ids = Array.from({ length: 100 }, (_, i) => `scene-${i}`);
    const offsets = ids.map((id) => calculateJitterOffset(id));

    const quadrants = {
      posPos: 0, // lat > 0, lng > 0
      posNeg: 0, // lat > 0, lng < 0
      negPos: 0, // lat < 0, lng > 0
      negNeg: 0, // lat < 0, lng < 0
    };

    offsets.forEach(({ latOffset, lngOffset }) => {
      if (latOffset > 0 && lngOffset > 0) quadrants.posPos++;
      if (latOffset > 0 && lngOffset < 0) quadrants.posNeg++;
      if (latOffset < 0 && lngOffset > 0) quadrants.negPos++;
      if (latOffset < 0 && lngOffset < 0) quadrants.negNeg++;
    });

    // Each quadrant should have at least 10% of samples (approximately uniform)
    Object.values(quadrants).forEach((count) => {
      expect(count).toBeGreaterThan(10);
    });
  });
});

describe('applyJitter', () => {
  it('applies jitter offset to coordinates', () => {
    const lat = 37.7749; // San Francisco
    const lng = -122.4194;
    const entityId = 'scene-sf';

    const jittered = applyJitter(lat, lng, entityId);

    expect(jittered.lat).not.toBe(lat);
    expect(jittered.lng).not.toBe(lng);
  });

  it('produces consistent jittered coordinates for the same entity', () => {
    const lat = 37.7749;
    const lng = -122.4194;
    const entityId = 'scene-consistent';

    const jittered1 = applyJitter(lat, lng, entityId);
    const jittered2 = applyJitter(lat, lng, entityId);

    expect(jittered1.lat).toBe(jittered2.lat);
    expect(jittered1.lng).toBe(jittered2.lng);
  });

  it('stays within configured radius', () => {
    const lat = 37.7749;
    const lng = -122.4194;
    const entityId = 'scene-bounds';
    const radiusMeters = 250;

    const jittered = applyJitter(lat, lng, entityId, radiusMeters);
    const distance = haversineDistance(lat, lng, jittered.lat, jittered.lng);

    // Distance should be within radius (with small floating-point tolerance)
    expect(distance).toBeLessThanOrEqual(radiusMeters + 1);
  });

  it('scales longitude offset based on latitude', () => {
    const entityId = 'scene-scaling';

    // Test at equator (lat = 0) where lng scaling is minimal
    const equator = applyJitter(0, 0, entityId);
    const equatorLngDiff = Math.abs(equator.lng);

    // Test at high latitude (lat = 60) where lng scaling should be significant
    const highLat = applyJitter(60, 0, entityId);
    const highLatLngDiff = Math.abs(highLat.lng);

    // At 60° latitude, longitude degrees are half as wide, so offset should be ~2x larger
    expect(highLatLngDiff).toBeGreaterThan(equatorLngDiff);
  });

  it('preserves determinism with different radius values', () => {
    const lat = 37.7749;
    const lng = -122.4194;
    const entityId = 'scene-radius-test';

    const jitter100 = applyJitter(lat, lng, entityId, 100);
    const jitter200 = applyJitter(lat, lng, entityId, 200);

    // Same entity ID should produce same direction, different magnitude
    const angle1 = Math.atan2(
      jitter100.lat - lat,
      jitter100.lng - lng
    );
    const angle2 = Math.atan2(
      jitter200.lat - lat,
      jitter200.lng - lng
    );

    expect(angle1).toBeCloseTo(angle2, 5);
  });

  it('handles extreme latitudes near poles without excessive longitude offsets', () => {
    const entityId = 'scene-polar';
    const radiusMeters = 250;

    // Test at 85° latitude (clamping threshold)
    const lat85 = applyJitter(85, 0, entityId, radiusMeters);
    const distance85 = haversineDistance(85, 0, lat85.lat, lat85.lng);
    
    // Distance should still be within reasonable bounds (allowing some variance)
    expect(distance85).toBeLessThan(radiusMeters * 3); // 3x is generous for high latitude
    
    // Test at 89° latitude (near pole, should be clamped to 85)
    const lat89 = applyJitter(89, 0, entityId, radiusMeters);
    const distance89 = haversineDistance(89, 0, lat89.lat, lat89.lng);
    
    // Should not produce infinite or extremely large offsets
    expect(distance89).toBeLessThan(radiusMeters * 3);
    expect(isFinite(lat89.lng)).toBe(true);
    
    // Test at -87° latitude (southern hemisphere, near pole)
    const latNeg87 = applyJitter(-87, 0, entityId, radiusMeters);
    const distanceNeg87 = haversineDistance(-87, 0, latNeg87.lat, latNeg87.lng);
    
    expect(distanceNeg87).toBeLessThan(radiusMeters * 3);
    expect(isFinite(latNeg87.lng)).toBe(true);
  });

  it('clamps latitude to ±85° for longitude scaling', () => {
    const entityId = 'scene-clamp-test';
    
    // For the same entity ID, coordinates at 85° and 90° should produce
    // similar longitude offsets (because 90° is clamped to 85°)
    const jitter85 = applyJitter(85, 0, entityId);
    const jitter90 = applyJitter(90, 0, entityId);
    
    // Longitude offsets should be similar (both use 85° for scaling)
    expect(Math.abs(jitter85.lng - jitter90.lng)).toBeLessThan(0.0001);
    
    // Same for negative latitudes
    const jitterNeg85 = applyJitter(-85, 0, entityId);
    const jitterNeg90 = applyJitter(-90, 0, entityId);
    
    expect(Math.abs(jitterNeg85.lng - jitterNeg90.lng)).toBeLessThan(0.0001);
  });
});

describe('shouldApplyJitter', () => {
  it('returns true when precise location is not allowed', () => {
    expect(shouldApplyJitter(false)).toBe(true);
  });

  it('returns false when precise location is allowed', () => {
    expect(shouldApplyJitter(true)).toBe(false);
  });
});

describe('haversineDistance', () => {
  it('calculates zero distance for same point', () => {
    const distance = haversineDistance(37.7749, -122.4194, 37.7749, -122.4194);
    expect(distance).toBe(0);
  });

  it('calculates correct distance for known points', () => {
    // San Francisco to Los Angeles (approximately 559 km)
    const sf = { lat: 37.7749, lng: -122.4194 };
    const la = { lat: 34.0522, lng: -118.2437 };
    const distance = haversineDistance(sf.lat, sf.lng, la.lat, la.lng);

    // Should be approximately 559,000 meters (±5%)
    expect(distance).toBeGreaterThan(531000);
    expect(distance).toBeLessThan(587000);
  });

  it('calculates small distances accurately', () => {
    // Two points approximately 100 meters apart
    const lat1 = 37.7749;
    const lng1 = -122.4194;
    const lat2 = 37.7749 + (100 / 111320); // Move ~100m north
    const lng2 = lng1;

    const distance = haversineDistance(lat1, lng1, lat2, lng2);

    expect(distance).toBeGreaterThan(99);
    expect(distance).toBeLessThan(101);
  });
});

describe('jitter distribution', () => {
  it('produces roughly uniform distribution within radius', () => {
    const lat = 37.7749;
    const lng = -122.4194;
    const radiusMeters = 250;
    const samples = 1000;

    // Generate many jittered points
    const distances = Array.from({ length: samples }, (_, i) => {
      const jittered = applyJitter(lat, lng, `scene-${i}`, radiusMeters);
      return haversineDistance(lat, lng, jittered.lat, jittered.lng);
    });

    // All distances should be within radius
    distances.forEach((d) => {
      expect(d).toBeLessThanOrEqual(radiusMeters + 1);
    });

    // Check for approximately uniform distribution
    // Divide into 5 concentric rings
    const ringSize = radiusMeters / 5;
    const rings = Array(5).fill(0);

    distances.forEach((d) => {
      const ringIndex = Math.min(Math.floor(d / ringSize), 4);
      rings[ringIndex]++;
    });

    // Outer rings should have more points (area increases with radius)
    // Ring 0 (0-50m) should have fewer points than Ring 4 (200-250m)
    expect(rings[0]).toBeLessThan(rings[4]);
  });
});
