/**
 * useEvents Hook
 * Hook for accessing multiple events with derived selectors
 */

import { useMemo } from 'react';
import { useEntityStore } from '../stores/entityStore';
import type { Event } from '../types/scene';

export interface UseEventsOptions {
  filterByScene?: string;
  sortBy?: 'name';
  includeLoading?: boolean;
}

export interface UseEventsResult {
  events: Event[];
  totalCount: number;
  loading: boolean;
}

/**
 * Hook to access and filter events from cache
 * Provides derived data like upcoming events count
 */
export function useEvents(options: UseEventsOptions = {}): UseEventsResult {
  const { filterByScene, includeLoading = false } = options;

  const cachedEvents = useEntityStore((state) => state.event.events);

  // Memoized filtered and sorted events
  const events = useMemo(() => {
    const eventList = Object.values(cachedEvents)
      .filter((cached) => {
        // Skip loading entries unless explicitly requested
        if (!includeLoading && cached.metadata.loading) return false;

        // Skip entries with errors or no data
        if (cached.metadata.error || !cached.data.id) return false;

        return true;
      })
      .map((cached) => cached.data);

    // Apply filters
    let filtered = eventList;

    if (filterByScene) {
      filtered = filtered.filter((event) => event.scene_id === filterByScene);
    }

    // Sort events by name
    const sorted = [...filtered].sort((a, b) => a.name.localeCompare(b.name));

    return sorted;
  }, [cachedEvents, filterByScene, includeLoading]);

  // Calculate total count
  const totalCount = useMemo(() => {
    return events.length;
  }, [events]);

  // Check if any event is loading
  const loading = useMemo(() => {
    return Object.values(cachedEvents).some((cached) => cached.metadata.loading);
  }, [cachedEvents]);

  return {
    events,
    totalCount,
    loading,
  };
}

/**
 * Hook to get events for a specific scene
 */
export function useSceneEvents(sceneId: string | undefined): UseEventsResult {
  const result = useEvents({ filterByScene: sceneId });

  // Return empty results if sceneId is undefined
  if (!sceneId) {
    return {
      events: [],
      totalCount: 0,
      loading: false,
    };
  }

  return result;
}

/**
 * Hook to get all events sorted by name
 * TODO: Add date-based filtering when Event type includes date fields
 */
export function useUpcomingEvents(): UseEventsResult {
  return useEvents({ sortBy: 'name' });
}
