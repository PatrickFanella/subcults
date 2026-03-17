import { describe, it, expect } from 'vitest';
import { buildGeoJSON } from './geojson';
import type { Scene, Event } from '../types/scene';

/**
 * Generate mock scene data for performance testing
 */
function generateMockScenes(count: number): Scene[] {
  const scenes: Scene[] = [];
  
  // Generate scenes clustered around SF Bay Area
  const baseLat = 37.7749;
  const baseLng = -122.4194;
  const spread = 1.0; // degrees

  for (let i = 0; i < count; i++) {
    const lat = baseLat + (Math.random() - 0.5) * spread;
    const lng = baseLng + (Math.random() - 0.5) * spread;
    
    scenes.push({
      id: `scene-${i}`,
      name: `Scene ${i}`,
      description: `Test scene ${i}`,
      allow_precise: i % 2 === 0, // 50% allow precise
      precise_point: { lat, lng },
      coarse_geohash: '9q8yy', // Simplified for test
      tags: ['test', `tag-${i % 10}`],
      visibility: 'public',
    });
  }

  return scenes;
}

/**
 * Generate mock event data for performance testing
 */
function generateMockEvents(count: number): Event[] {
  const events: Event[] = [];
  
  const baseLat = 37.7749;
  const baseLng = -122.4194;
  const spread = 1.0;

  for (let i = 0; i < count; i++) {
    const lat = baseLat + (Math.random() - 0.5) * spread;
    const lng = baseLng + (Math.random() - 0.5) * spread;
    
    events.push({
      id: `event-${i}`,
      scene_id: `scene-${i % 100}`,
      name: `Event ${i}`,
      description: `Test event ${i}`,
      allow_precise: i % 3 === 0, // 33% allow precise
      precise_point: { lat, lng },
      coarse_geohash: '9q8yy', // Add coarse geohash for events without precise point
    });
  }

  return events;
}

describe('GeoJSON Performance', () => {
  it('builds GeoJSON from 5000 scenes and events in <150ms', () => {
    const scenes = generateMockScenes(2500);
    const events = generateMockEvents(2500);

    const startTime = performance.now();
    const geojson = buildGeoJSON(scenes, events);
    const elapsed = performance.now() - startTime;

    expect(geojson.features).toHaveLength(5000);
    expect(elapsed).toBeLessThan(150);
    
    console.log(`Built GeoJSON from 5000 entities in ${elapsed.toFixed(2)}ms`);
  });

  it('builds GeoJSON from 10000 scenes and events in <300ms', () => {
    const scenes = generateMockScenes(5000);
    const events = generateMockEvents(5000);

    const startTime = performance.now();
    const geojson = buildGeoJSON(scenes, events);
    const elapsed = performance.now() - startTime;

    expect(geojson.features).toHaveLength(10000);
    expect(elapsed).toBeLessThan(300);
    
    console.log(`Built GeoJSON from 10000 entities in ${elapsed.toFixed(2)}ms`);
  });

  it('respects privacy settings for all mock scenes', () => {
    const scenes = generateMockScenes(1000);
    const geojson = buildGeoJSON(scenes, []);

    // Verify that scenes without allow_precise use geohash coordinates
    const features = geojson.features;
    let preciseCoordsCount = 0;
    let geohashCoordsCount = 0;

    for (let i = 0; i < features.length; i++) {
      const feature = features[i];
      const scene = scenes[i];
      
      if (scene.allow_precise && scene.precise_point) {
        // Should use exact coordinates
        expect(feature.geometry.coordinates).toEqual([
          scene.precise_point.lng,
          scene.precise_point.lat,
        ]);
        preciseCoordsCount++;
      } else {
        // Should use geohash (approximation)
        const coords = feature.geometry.coordinates;
        expect(coords[0]).toBeCloseTo(-122.42, 1);
        expect(coords[1]).toBeCloseTo(37.77, 1);
        geohashCoordsCount++;
      }
    }

    // Verify distribution
    expect(preciseCoordsCount).toBeGreaterThan(400);
    expect(geohashCoordsCount).toBeGreaterThan(400);
    
    console.log(`Privacy: ${preciseCoordsCount} precise, ${geohashCoordsCount} coarse`);
  });
});
