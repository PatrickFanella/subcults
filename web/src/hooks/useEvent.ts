/**
 * useEvent Hook
 * Hook for fetching and subscribing to an event with automatic stale revalidation
 */

import { useEffect, useMemo } from 'react';
import { useEntityStore } from '../stores/entityStore';
import { isStale, TTL_CONFIG } from '../stores/entityStore';
import type { Event } from '../types/scene';

export interface UseEventResult {
  event: Event | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook to fetch and subscribe to an event
 * Automatically fetches if missing or stale (stale-while-revalidate)
 */
export function useEvent(id: string | undefined): UseEventResult {
  const cached = useEntityStore((state) => (id ? state.event.events[id] : undefined));
  const fetchEvent = useEntityStore((state) => state.fetchEvent);
  const markEventStale = useEntityStore((state) => state.markEventStale);

  // Fetch event if missing or stale
  useEffect(() => {
    if (!id) return;

    const shouldFetch = !cached || isStale(cached.metadata, TTL_CONFIG.DEFAULT);

    if (shouldFetch && !cached?.metadata.loading) {
      // Background fetch if stale (stale-while-revalidate)
      // Errors are stored in metadata and exposed via the error property
      fetchEvent(id).catch(() => {
        // Error is captured in store metadata
      });
    }
  }, [id, cached, fetchEvent]);

  // Memoized refetch function
  const refetch = useMemo(
    () => async () => {
      if (!id) return;
      // Mark as stale to force refetch
      markEventStale(id);
      await fetchEvent(id);
    },
    [id, fetchEvent, markEventStale]
  );

  return {
    event: cached?.data?.id ? cached.data : null,
    loading: cached?.metadata.loading || false,
    error: cached?.metadata.error || null,
    refetch,
  };
}
