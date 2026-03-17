/**
 * Entity Store
 * Base types and utilities for entity state management with caching and TTL
 */

import { create } from 'zustand';
import type { Scene, Event } from '../types/scene';
import type { User } from './authStore';

/**
 * Cache entry metadata
 */
export interface CacheMetadata {
  timestamp: number; // When entity was cached
  loading: boolean; // Currently fetching
  error: string | null; // Last error message
  stale: boolean; // Marked as stale (needs revalidation)
}

/**
 * Cached entity with metadata
 */
export interface CachedEntity<T> {
  data: T;
  metadata: CacheMetadata;
}

/**
 * TTL configuration (in milliseconds)
 */
export const TTL_CONFIG = {
  DEFAULT: 60000, // 60 seconds
  SHORT: 30000, // 30 seconds
  LONG: 300000, // 5 minutes
};

/**
 * Check if cache entry is stale based on TTL
 */
export function isStale(metadata: CacheMetadata, ttl: number = TTL_CONFIG.DEFAULT): boolean {
  if (metadata.stale) return true;
  return Date.now() - metadata.timestamp > ttl;
}

/**
 * Create fresh cache metadata
 */
export function createFreshMetadata(): CacheMetadata {
  return {
    timestamp: Date.now(),
    loading: false,
    error: null,
    stale: false,
  };
}

/**
 * Update metadata to loading state
 */
export function setLoadingMetadata(metadata: CacheMetadata): CacheMetadata {
  return {
    ...metadata,
    loading: true,
    error: null,
  };
}

/**
 * Update metadata after successful fetch
 */
export function setSuccessMetadata(): CacheMetadata {
  return {
    timestamp: Date.now(),
    loading: false,
    error: null,
    stale: false,
  };
}

/**
 * Update metadata after failed fetch
 */
export function setErrorMetadata(metadata: CacheMetadata, error: string): CacheMetadata {
  return {
    ...metadata,
    loading: false,
    error,
  };
}

/**
 * Scene state slice
 */
export interface SceneState {
  scenes: Record<string, CachedEntity<Scene>>;
  optimisticUpdates: Record<string, Scene>; // For rollback
}

/**
 * Event state slice
 */
export interface EventState {
  events: Record<string, CachedEntity<Event>>;
}

/**
 * User state slice (minimal PII)
 */
export interface UserState {
  users: Record<string, CachedEntity<User>>;
}

/**
 * Combined entity store state
 */
export interface EntityStoreState {
  scene: SceneState;
  event: EventState;
  user: UserState;
}

/**
 * Scene actions
 */
export interface SceneActions {
  // Fetch scene by ID
  fetchScene: (id: string) => Promise<Scene>;
  
  // Set scene in cache
  setScene: (scene: Scene) => void;
  
  // Mark scene as stale
  markSceneStale: (id: string) => void;
  
  // Optimistic membership join
  optimisticJoinScene: (sceneId: string, userId: string) => void;
  
  // Rollback optimistic update
  rollbackSceneUpdate: (sceneId: string) => void;
  
  // Commit optimistic update
  commitSceneUpdate: (sceneId: string) => void;
  
  // Clear scene error
  clearSceneError: (id: string) => void;
  
  // Remove scene from cache
  removeScene: (id: string) => void;
  
  // Update scene
  updateScene: (id: string, updates: Partial<Scene>) => Promise<Scene>;
}

/**
 * Event actions
 */
export interface EventActions {
  // Fetch event by ID
  fetchEvent: (id: string) => Promise<Event>;
  
  // Set event in cache
  setEvent: (event: Event) => void;
  
  // Mark event as stale
  markEventStale: (id: string) => void;
  
  // Clear event error
  clearEventError: (id: string) => void;
  
  // Remove event from cache
  removeEvent: (id: string) => void;
}

/**
 * User actions
 */
export interface UserActions {
  // Fetch user by DID
  fetchUser: (did: string) => Promise<User>;
  
  // Set user in cache
  setUser: (user: User) => void;
  
  // Mark user as stale
  markUserStale: (did: string) => void;
  
  // Clear user error
  clearUserError: (did: string) => void;
  
  // Remove user from cache
  removeUser: (did: string) => void;
}

/**
 * Combined store actions
 */
export type EntityStoreActions = SceneActions & EventActions & UserActions;

/**
 * Full entity store type
 */
export type EntityStore = EntityStoreState & EntityStoreActions;

/**
 * In-flight requests tracker to prevent duplicate fetches
 * Internal implementation detail, not exported
 */
const inFlightRequests = new Map<string, Promise<unknown>>();

// Default timeout for in-flight requests (ms)
const IN_FLIGHT_REQUEST_TIMEOUT = 60000; // 60 seconds

/**
 * Get or create in-flight request
 * Deduplicates concurrent requests for the same entity
 * Cleans up after timeout to prevent memory leaks
 */
export function getOrCreateRequest<T>(
  key: string,
  requestFn: () => Promise<T>,
  timeoutMs: number = IN_FLIGHT_REQUEST_TIMEOUT
): Promise<T> {
  const existing = inFlightRequests.get(key);
  if (existing) {
    return existing as Promise<T>;
  }

  let timeoutId: ReturnType<typeof setTimeout> | null = null;
  const promise = requestFn().finally(() => {
    inFlightRequests.delete(key);
    if (timeoutId) clearTimeout(timeoutId);
  });

  inFlightRequests.set(key, promise);
  
  // Set timeout to clean up if promise never resolves
  timeoutId = setTimeout(() => {
    inFlightRequests.delete(key);
  }, timeoutMs);

  return promise;
}

/**
 * For testing: reset in-flight requests map
 */
export function resetInFlightRequests(): void {
  inFlightRequests.clear();
}

/**
 * Entity Store
 * Combined store using slice pattern
 */
import { createSceneSlice } from './slices/sceneSlice';
import { createEventSlice } from './slices/eventSlice';
import { createUserSlice } from './slices/userSlice';

export const useEntityStore = create<EntityStore>()((...a) => ({
  ...createSceneSlice(...a),
  ...createEventSlice(...a),
  ...createUserSlice(...a),
}));
