import type { Scene, Event, Point } from '../types/scene';
import type { FeatureCollection, Feature, Point as GeoJSONPoint } from 'geojson';
import { applyJitter, shouldApplyJitter } from './jitter';

/**
 * Properties for a scene or event feature in GeoJSON
 */
export interface FeatureProperties {
  id: string;
  type: 'scene' | 'event';
  name: string;
  description?: string;
  coarse_geohash?: string;
  scene_id?: string;
  tags?: string[];
  visibility?: string;
  palette?: {
    primary: string;
    secondary: string;
  };
  is_jittered?: boolean; // Indicates if coordinates have privacy jitter applied
}

/**
 * Decode a geohash to approximate lat/lng coordinates
 * This is a simplified implementation for coarse geohashes (5-7 chars)
 * 
 * @param geohash - The geohash string to decode
 * @returns Point with lat/lng coordinates at the center of the geohash cell
 */
export function decodeGeohash(geohash: string): Point {
  if (!geohash || geohash.length === 0) {
    throw new Error('Invalid geohash: empty string');
  }
  
  const BASE32 = '0123456789bcdefghjkmnpqrstuvwxyz';
  let evenBit = true;
  let latMin = -90, latMax = 90;
  let lngMin = -180, lngMax = 180;

  for (let i = 0; i < geohash.length; i++) {
    const char = geohash[i];
    const idx = BASE32.indexOf(char);
    
    if (idx === -1) {
      throw new Error(`Invalid geohash character: ${char}`);
    }

    for (let j = 4; j >= 0; j--) {
      const bit = (idx >> j) & 1;
      
      if (evenBit) {
        // longitude
        const lngMid = (lngMin + lngMax) / 2;
        if (bit === 1) {
          lngMin = lngMid;
        } else {
          lngMax = lngMid;
        }
      } else {
        // latitude
        const latMid = (latMin + latMax) / 2;
        if (bit === 1) {
          latMin = latMid;
        } else {
          latMax = latMid;
        }
      }
      
      evenBit = !evenBit;
    }
  }

  return {
    lat: (latMin + latMax) / 2,
    lng: (lngMin + lngMax) / 2,
  };
}

/**
 * Get display coordinates for an entity, respecting location consent
 * Returns precise point if allowed, otherwise returns coarse geohash coordinates
 * 
 * @param entity - Scene or Event entity
 * @returns Point coordinates for display
 */
export function getDisplayCoordinates(entity: Scene | Event): Point {
  // For events, check for precise point or coarse geohash
  if ('scene_id' in entity) {
    // Event: use precise point if available and allowed
    if (entity.allow_precise && entity.precise_point) {
      return entity.precise_point;
    }
    // Use event's coarse geohash if available
    if (entity.coarse_geohash) {
      return decodeGeohash(entity.coarse_geohash);
    }
    // Data integrity error - events must have location data
    throw new Error(`Event ${entity.id} missing location data - events must have precise_point or coarse_geohash`);
  }
  
  // Scene: use precise point if allowed, otherwise decode coarse geohash
  if (entity.allow_precise && entity.precise_point) {
    return entity.precise_point;
  }
  
  // Use coarse geohash for privacy
  if (!entity.coarse_geohash) {
    throw new Error(`Scene ${entity.id} missing required coarse_geohash for privacy enforcement`);
  }
  return decodeGeohash(entity.coarse_geohash);
}

/**
 * Build a GeoJSON FeatureCollection from scenes and events
 * Respects location privacy by using coarse geohash when precise location is not allowed
 * Applies deterministic jitter to coarse coordinates for privacy visualization
 * 
 * @param scenes - Array of Scene entities
 * @param events - Array of Event entities
 * @param enableJitter - Whether to apply jitter to coarse coordinates (default: true)
 * @returns GeoJSON FeatureCollection with features for all entities
 */
export function buildGeoJSON(
  scenes: Scene[] = [],
  events: Event[] = [],
  enableJitter: boolean = true
): FeatureCollection<GeoJSONPoint, FeatureProperties> {
  const features: Feature<GeoJSONPoint, FeatureProperties>[] = [];

  // Add scene features
  for (const scene of scenes) {
    let coords = getDisplayCoordinates(scene);
    const applyJitterToCoords = enableJitter && shouldApplyJitter(scene.allow_precise);
    
    // Apply jitter to coarse coordinates for privacy visualization
    if (applyJitterToCoords) {
      coords = applyJitter(coords.lat, coords.lng, scene.id);
    }
    
    features.push({
      type: 'Feature',
      geometry: {
        type: 'Point',
        coordinates: [coords.lng, coords.lat],
      },
      properties: {
        id: scene.id,
        type: 'scene',
        name: scene.name,
        description: scene.description,
        coarse_geohash: scene.coarse_geohash,
        tags: scene.tags,
        visibility: scene.visibility,
        palette: scene.palette,
        is_jittered: applyJitterToCoords,
      },
    });
  }

  // Add event features
  for (const event of events) {
    let coords = getDisplayCoordinates(event);
    const applyJitterToCoords = enableJitter && shouldApplyJitter(event.allow_precise);
    
    // Apply jitter to coarse coordinates for privacy visualization
    if (applyJitterToCoords) {
      coords = applyJitter(coords.lat, coords.lng, event.id);
    }
    
    features.push({
      type: 'Feature',
      geometry: {
        type: 'Point',
        coordinates: [coords.lng, coords.lat],
      },
      properties: {
        id: event.id,
        type: 'event',
        name: event.name,
        description: event.description,
        scene_id: event.scene_id,
        is_jittered: applyJitterToCoords,
      },
    });
  }

  return {
    type: 'FeatureCollection',
    features,
  };
}
