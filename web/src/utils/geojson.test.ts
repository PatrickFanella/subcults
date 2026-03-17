import { describe, it, expect } from 'vitest';
import { buildGeoJSON, decodeGeohash, getDisplayCoordinates } from './geojson';
import type { Scene, Event } from '../types/scene';

describe('decodeGeohash', () => {
  it('decodes a 5-character geohash to approximate coordinates', () => {
    // '9q8yy' is approximately San Francisco (37.77, -122.42)
    const result = decodeGeohash('9q8yy');
    expect(result.lat).toBeCloseTo(37.77, 1);
    expect(result.lng).toBeCloseTo(-122.42, 1);
  });

  it('decodes a 7-character geohash to more precise coordinates', () => {
    // '9q8yyk8' is more precise location in San Francisco
    const result = decodeGeohash('9q8yyk8');
    expect(result.lat).toBeCloseTo(37.77, 2);
    expect(result.lng).toBeCloseTo(-122.42, 2);
  });

  it('throws error for invalid geohash characters', () => {
    expect(() => decodeGeohash('invalid!')).toThrow('Invalid geohash character');
  });

  it('throws error for empty geohash string', () => {
    expect(() => decodeGeohash('')).toThrow('Invalid geohash: empty string');
  });
});

describe('getDisplayCoordinates', () => {
  it('returns precise point when allow_precise is true for scene', () => {
    const scene: Scene = {
      id: '1',
      name: 'Test Scene',
      allow_precise: true,
      precise_point: { lat: 37.7749, lng: -122.4194 },
      coarse_geohash: '9q8yy',
    };

    const result = getDisplayCoordinates(scene);
    expect(result).toEqual({ lat: 37.7749, lng: -122.4194 });
  });

  it('returns coarse geohash coordinates when allow_precise is false for scene', () => {
    const scene: Scene = {
      id: '1',
      name: 'Test Scene',
      allow_precise: false,
      precise_point: { lat: 37.7749, lng: -122.4194 },
      coarse_geohash: '9q8yy',
    };

    const result = getDisplayCoordinates(scene);
    // Should use geohash, not precise point
    expect(result.lat).toBeCloseTo(37.77, 1);
    expect(result.lng).toBeCloseTo(-122.42, 1);
    expect(result).not.toEqual(scene.precise_point);
  });

  it('returns coarse geohash coordinates when precise_point is missing for scene', () => {
    const scene: Scene = {
      id: '1',
      name: 'Test Scene',
      allow_precise: true,
      coarse_geohash: '9q8yy',
    };

    const result = getDisplayCoordinates(scene);
    expect(result.lat).toBeCloseTo(37.77, 1);
    expect(result.lng).toBeCloseTo(-122.42, 1);
  });

  it('returns precise point when allow_precise is true for event', () => {
    const event: Event = {
      id: '1',
      scene_id: 'scene1',
      name: 'Test Event',
      allow_precise: true,
      precise_point: { lat: 37.7749, lng: -122.4194 },
    };

    const result = getDisplayCoordinates(event);
    expect(result).toEqual({ lat: 37.7749, lng: -122.4194 });
  });

  it('returns coarse geohash coordinates when event has no precise point but has coarse_geohash', () => {
    const event: Event = {
      id: '1',
      scene_id: 'scene1',
      name: 'Test Event',
      allow_precise: false,
      coarse_geohash: '9q8yy',
    };

    const result = getDisplayCoordinates(event);
    expect(result.lat).toBeCloseTo(37.77, 1);
    expect(result.lng).toBeCloseTo(-122.42, 1);
  });

  it('throws error when event has no precise point and no coarse_geohash', () => {
    const event: Event = {
      id: '1',
      scene_id: 'scene1',
      name: 'Test Event',
      allow_precise: false,
    };

    expect(() => getDisplayCoordinates(event)).toThrow(
      'Event 1 missing location data - events must have precise_point or coarse_geohash'
    );
  });

  it('throws error when scene has no coarse_geohash', () => {
    const scene: Scene = {
      id: '1',
      name: 'Test Scene',
      allow_precise: false,
      coarse_geohash: '',
    };

    expect(() => getDisplayCoordinates(scene)).toThrow(
      'Scene 1 missing required coarse_geohash for privacy enforcement'
    );
  });
});

describe('buildGeoJSON', () => {
  it('builds empty FeatureCollection from empty arrays', () => {
    const result = buildGeoJSON([], []);
    expect(result).toEqual({
      type: 'FeatureCollection',
      features: [],
    });
  });

  it('builds GeoJSON from scenes with precise locations', () => {
    const scenes: Scene[] = [
      {
        id: 'scene1',
        name: 'Underground Venue',
        description: 'Secret spot',
        allow_precise: true,
        precise_point: { lat: 37.7749, lng: -122.4194 },
        coarse_geohash: '9q8yy',
        tags: ['techno', 'warehouse'],
        visibility: 'public',
      },
    ];

    const result = buildGeoJSON(scenes, []);
    
    expect(result.type).toBe('FeatureCollection');
    expect(result.features).toHaveLength(1);
    
    const feature = result.features[0];
    expect(feature.geometry.type).toBe('Point');
    expect(feature.geometry.coordinates).toEqual([-122.4194, 37.7749]);
    expect(feature.properties.id).toBe('scene1');
    expect(feature.properties.type).toBe('scene');
    expect(feature.properties.name).toBe('Underground Venue');
    expect(feature.properties.coarse_geohash).toBe('9q8yy');
    expect(feature.properties.tags).toEqual(['techno', 'warehouse']);
  });

  it('builds GeoJSON from scenes with coarse locations only', () => {
    const scenes: Scene[] = [
      {
        id: 'scene1',
        name: 'Private Scene',
        allow_precise: false,
        precise_point: { lat: 37.7749, lng: -122.4194 },
        coarse_geohash: '9q8yy',
      },
    ];

    const result = buildGeoJSON(scenes, []);
    
    const feature = result.features[0];
    // Coordinates should be from geohash, not precise point
    expect(feature.geometry.coordinates[0]).toBeCloseTo(-122.42, 1);
    expect(feature.geometry.coordinates[1]).toBeCloseTo(37.77, 1);
    expect(feature.geometry.coordinates).not.toEqual([-122.4194, 37.7749]);
  });

  it('builds GeoJSON from events', () => {
    const events: Event[] = [
      {
        id: 'event1',
        scene_id: 'scene1',
        name: 'Weekend Show',
        description: 'Live performance',
        allow_precise: true,
        precise_point: { lat: 37.7849, lng: -122.4094 },
      },
    ];

    const result = buildGeoJSON([], events);
    
    expect(result.features).toHaveLength(1);
    
    const feature = result.features[0];
    expect(feature.geometry.coordinates).toEqual([-122.4094, 37.7849]);
    expect(feature.properties.id).toBe('event1');
    expect(feature.properties.type).toBe('event');
    expect(feature.properties.scene_id).toBe('scene1');
  });

  it('builds GeoJSON combining scenes and events', () => {
    const scenes: Scene[] = [
      {
        id: 'scene1',
        name: 'Venue A',
        allow_precise: true,
        precise_point: { lat: 37.7749, lng: -122.4194 },
        coarse_geohash: '9q8yy',
      },
      {
        id: 'scene2',
        name: 'Venue B',
        allow_precise: false,
        coarse_geohash: '9q9p1',
      },
    ];

    const events: Event[] = [
      {
        id: 'event1',
        scene_id: 'scene1',
        name: 'Show 1',
        allow_precise: true,
        precise_point: { lat: 37.7849, lng: -122.4094 },
      },
    ];

    const result = buildGeoJSON(scenes, events);
    
    expect(result.features).toHaveLength(3);
    
    // Verify types
    const sceneFeatures = result.features.filter(f => f.properties.type === 'scene');
    const eventFeatures = result.features.filter(f => f.properties.type === 'event');
    expect(sceneFeatures).toHaveLength(2);
    expect(eventFeatures).toHaveLength(1);
  });

  it('handles scenes with all optional properties', () => {
    const scenes: Scene[] = [
      {
        id: 'scene1',
        name: 'Minimal Scene',
        allow_precise: true,
        precise_point: { lat: 37.7749, lng: -122.4194 },
        coarse_geohash: '9q8yy',
        tags: ['minimal'],
        visibility: 'unlisted',
        palette: { primary: '#ff0000', secondary: '#00ff00' },
      },
    ];

    const result = buildGeoJSON(scenes, []);
    
    const feature = result.features[0];
    expect(feature.properties.tags).toEqual(['minimal']);
    expect(feature.properties.visibility).toBe('unlisted');
    expect(feature.properties.palette).toEqual({ primary: '#ff0000', secondary: '#00ff00' });
  });

  it('applies jitter to coarse coordinates by default', () => {
    const scenes: Scene[] = [
      {
        id: 'scene-jitter',
        name: 'Scene with Coarse Location',
        allow_precise: false,
        precise_point: { lat: 37.7749, lng: -122.4194 },
        coarse_geohash: '9q8yy',
      },
    ];

    const result = buildGeoJSON(scenes, []);
    const feature = result.features[0];
    
    // Should have jitter applied
    expect(feature.properties.is_jittered).toBe(true);
    
    // Get coordinates with and without jitter to verify difference
    const resultNoJitter = buildGeoJSON(scenes, [], false);
    const coordsWithJitter = feature.geometry.coordinates;
    const coordsNoJitter = resultNoJitter.features[0].geometry.coordinates;
    
    // Coordinates should be different when jitter is applied
    expect(coordsWithJitter).not.toEqual(coordsNoJitter);
  });

  it('does not apply jitter to precise coordinates', () => {
    const scenes: Scene[] = [
      {
        id: 'scene-precise',
        name: 'Scene with Precise Location',
        allow_precise: true,
        precise_point: { lat: 37.7749, lng: -122.4194 },
        coarse_geohash: '9q8yy',
      },
    ];

    const result = buildGeoJSON(scenes, []);
    const feature = result.features[0];
    
    // Should NOT have jitter applied
    expect(feature.properties.is_jittered).toBe(false);
    
    // Coordinates should match precise point exactly
    expect(feature.geometry.coordinates).toEqual([-122.4194, 37.7749]);
  });

  it('applies consistent jitter for same entity ID across calls', () => {
    const scene: Scene = {
      id: 'scene-consistent',
      name: 'Consistent Scene',
      allow_precise: false,
      coarse_geohash: '9q8yy',
    };

    const result1 = buildGeoJSON([scene], []);
    const result2 = buildGeoJSON([scene], []);
    
    const coords1 = result1.features[0].geometry.coordinates;
    const coords2 = result2.features[0].geometry.coordinates;
    
    // Same scene should produce identical jittered coordinates
    expect(coords1).toEqual(coords2);
  });

  it('respects enableJitter parameter', () => {
    const scene: Scene = {
      id: 'scene-no-jitter',
      name: 'Scene Without Jitter',
      allow_precise: false,
      coarse_geohash: '9q8yy',
    };

    const resultWithJitter = buildGeoJSON([scene], [], true);
    const resultWithoutJitter = buildGeoJSON([scene], [], false);
    
    // With jitter enabled
    expect(resultWithJitter.features[0].properties.is_jittered).toBe(true);
    
    // With jitter disabled
    expect(resultWithoutJitter.features[0].properties.is_jittered).toBe(false);
    
    // Coordinates should differ when jitter is toggled
    const coordsWithJitter = resultWithJitter.features[0].geometry.coordinates;
    const coordsWithoutJitter = resultWithoutJitter.features[0].geometry.coordinates;
    expect(coordsWithJitter).not.toEqual(coordsWithoutJitter);
  });

  it('applies jitter to events with coarse locations', () => {
    const event: Event = {
      id: 'event-jitter',
      scene_id: 'scene1',
      name: 'Event with Coarse Location',
      allow_precise: false,
      coarse_geohash: '9q8yy',
    };

    const result = buildGeoJSON([], [event]);
    const feature = result.features[0];
    
    expect(feature.properties.is_jittered).toBe(true);
    expect(feature.properties.type).toBe('event');
  });
});
