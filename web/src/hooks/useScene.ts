/**
 * useScene Hook
 * Hook for fetching and subscribing to a scene with automatic stale revalidation
 */

import { useEffect, useMemo } from 'react';
import { useEntityStore } from '../stores/entityStore';
import { isStale, TTL_CONFIG } from '../stores/entityStore';
import type { Scene } from '../types/scene';

export interface UseSceneResult {
  scene: Scene | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook to fetch and subscribe to a scene
 * Automatically fetches if missing or stale (stale-while-revalidate)
 */
export function useScene(id: string | undefined): UseSceneResult {
  const cached = useEntityStore((state) => (id ? state.scene.scenes[id] : undefined));
  const fetchScene = useEntityStore((state) => state.fetchScene);
  const markSceneStale = useEntityStore((state) => state.markSceneStale);

  // Fetch scene if missing or stale
  useEffect(() => {
    if (!id) return;

    const shouldFetch = !cached || isStale(cached.metadata, TTL_CONFIG.DEFAULT);

    if (shouldFetch && !cached?.metadata.loading) {
      // Background fetch if stale (stale-while-revalidate)
      // Errors are stored in metadata and exposed via the error property
      fetchScene(id).catch(() => {
        // Error is captured in store metadata
      });
    }
  }, [id, cached, fetchScene]);

  // Memoized refetch function
  const refetch = useMemo(
    () => async () => {
      if (!id) return;
      // Mark as stale to force refetch
      markSceneStale(id);
      await fetchScene(id);
    },
    [id, fetchScene, markSceneStale]
  );

  return {
    scene: cached?.data?.id ? cached.data : null,
    loading: cached?.metadata.loading || false,
    error: cached?.metadata.error || null,
    refetch,
  };
}
