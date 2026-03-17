/**
 * Privacy-preserving location jitter utilities
 * 
 * Applies deterministic random offsets to coordinates to prevent exact location tracking
 * while maintaining consistent visualization per entity across sessions.
 */

/**
 * Maximum jitter offset in meters (default: 250m visual offset)
 */
export const DEFAULT_JITTER_RADIUS_METERS = 250;

/**
 * Meters per degree latitude (approximate, varies slightly with latitude)
 */
const METERS_PER_DEGREE_LAT = 111320;

/**
 * Simple hash function to generate deterministic pseudo-random values from a string
 * Uses FNV-1a hash algorithm for simplicity and speed
 * 
 * @param str - String to hash (typically entity ID)
 * @returns 32-bit unsigned integer hash
 */
function simpleHash(str: string): number {
  let hash = 2166136261; // FNV offset basis
  
  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i);
    hash = Math.imul(hash, 16777619); // FNV prime
  }
  
  return hash >>> 0; // Convert to unsigned 32-bit integer
}

/**
 * Generate deterministic jitter offset for an entity
 * Uses entity ID to produce consistent offsets across re-renders
 * 
 * @param entityId - Unique entity identifier
 * @param radiusMeters - Maximum jitter radius in meters (default: 250m)
 * @returns Object with latOffset and lngOffset in degrees
 */
export function calculateJitterOffset(
  entityId: string,
  radiusMeters: number = DEFAULT_JITTER_RADIUS_METERS
): { latOffset: number; lngOffset: number } {
  // Generate two independent hash values by using different suffixes
  const hash1 = simpleHash(entityId + '-lat');
  const hash2 = simpleHash(entityId + '-lng');
  
  // Convert hashes to normalized values [0, 1)
  // Use Math.pow(2, 32) to ensure proper normalization of 32-bit unsigned integers
  const norm1 = hash1 / Math.pow(2, 32);
  const norm2 = hash2 / Math.pow(2, 32);
  
  // Use polar coordinates to ensure uniform distribution within circle
  // Convert to [0, 2π) for angle and [0, 1) for radius
  const angle = norm1 * 2 * Math.PI;
  const radius = Math.sqrt(norm2) * radiusMeters; // sqrt for uniform area distribution
  
  // Convert polar to Cartesian offsets in meters
  const offsetXMeters = radius * Math.cos(angle);
  const offsetYMeters = radius * Math.sin(angle);
  
  // Convert meters to degrees
  // NOTE: Latitude conversion is straightforward (constant ~111,320 m/deg)
  const latOffset = offsetYMeters / METERS_PER_DEGREE_LAT;
  
  // Longitude: Use equatorial conversion as base (will be scaled by cos(lat) in applyJitter)
  // This allows calculateJitterOffset to be latitude-independent while still deterministic
  const lngOffsetBase = offsetXMeters / METERS_PER_DEGREE_LAT;
  
  return { latOffset, lngOffset: lngOffsetBase };
}

/**
 * Apply jitter offset to coordinates
 * Scales base longitude offset by latitude to maintain consistent distance
 * 
 * The longitude offset from calculateJitterOffset uses equatorial meters/degree
 * as a base value. This function scales it by 1/cos(latitude) to maintain
 * the correct distance in meters at the given latitude.
 * 
 * For extreme latitudes (near poles), the latitude is clamped to ±85° to prevent
 * division by values too close to zero, which would cause excessive longitude offsets.
 * 
 * @param lat - Original latitude
 * @param lng - Original longitude
 * @param entityId - Unique entity identifier for deterministic offset
 * @param radiusMeters - Maximum jitter radius in meters (default: 250m)
 * @returns Jittered coordinates
 */
export function applyJitter(
  lat: number,
  lng: number,
  entityId: string,
  radiusMeters: number = DEFAULT_JITTER_RADIUS_METERS
): { lat: number; lng: number } {
  const { latOffset, lngOffset } = calculateJitterOffset(entityId, radiusMeters);
  
  // Clamp latitude to ±85° to avoid division by values too close to zero near poles
  // This is the same limit used by Web Mercator projection (e.g., Google Maps, OpenStreetMap)
  const clampedLat = Math.max(-85, Math.min(85, lat));
  
  // Scale longitude offset to account for latitude
  // At equator: cos(0°) = 1.0, no scaling needed
  // At 60° lat: cos(60°) = 0.5, double the offset to maintain distance
  const scaledLngOffset = lngOffset / Math.cos((clampedLat * Math.PI) / 180);
  
  return {
    lat: lat + latOffset,
    lng: lng + scaledLngOffset,
  };
}

/**
 * Check if coordinates should be jittered based on privacy flags
 * Jitter is only applied to coarse (non-precise) locations
 * 
 * @param allowPrecise - Whether precise location is allowed
 * @returns True if jitter should be applied
 */
export function shouldApplyJitter(allowPrecise: boolean): boolean {
  return !allowPrecise;
}

/**
 * Calculate the distance in meters between two points using Haversine formula
 * Useful for validating jitter stays within bounds
 * 
 * @param lat1 - First point latitude
 * @param lng1 - First point longitude
 * @param lat2 - Second point latitude
 * @param lng2 - Second point longitude
 * @returns Distance in meters
 */
export function haversineDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number
): number {
  const R = 6371000; // Earth radius in meters
  const φ1 = (lat1 * Math.PI) / 180;
  const φ2 = (lat2 * Math.PI) / 180;
  const Δφ = ((lat2 - lat1) * Math.PI) / 180;
  const Δλ = ((lng2 - lng1) * Math.PI) / 180;

  const a =
    Math.sin(Δφ / 2) * Math.sin(Δφ / 2) +
    Math.cos(φ1) * Math.cos(φ2) * Math.sin(Δλ / 2) * Math.sin(Δλ / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));

  return R * c;
}
