/**
 * Geographic point with latitude and longitude
 */
export interface Point {
  lat: number;
  lng: number;
}

/**
 * Color palette for scene visual identity
 */
export interface Palette {
  primary: string;
  secondary: string;
  accent: string;
  background: string;
  text: string;
}

/**
 * Scene entity representing a subcultural scene
 */
export interface Scene {
  id: string;
  name: string;
  description?: string;
  owner_did?: string; // Decentralized Identifier of scene owner
  allow_precise: boolean;
  precise_point?: Point;
  coarse_geohash: string;
  tags?: string[];
  visibility?: 'public' | 'private' | 'unlisted';
  palette?: Palette;
  owner_user_id?: string;
  record_did?: string;
  record_rkey?: string;
}

/**
 * Event entity representing an event within a scene
 */
export interface Event {
  id: string;
  scene_id: string;
  name: string;
  description?: string;
  allow_precise: boolean;
  precise_point?: Point;
  coarse_geohash?: string;  // Optional coarse location for privacy when precise is not allowed
  record_did?: string;
  record_rkey?: string;
}

/**
 * Union type for scene or event
 */
export type SceneOrEvent = Scene | Event;

/**
 * Type guard to check if entity is a Scene
 */
export function isScene(entity: SceneOrEvent): entity is Scene {
  return 'coarse_geohash' in entity;
}

/**
 * Type guard to check if entity is an Event
 */
export function isEvent(entity: SceneOrEvent): entity is Event {
  return 'scene_id' in entity;
}
