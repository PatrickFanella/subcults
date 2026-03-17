/**
 * useScenes Hook
 * Hook for accessing multiple scenes with derived selectors
 */

import { useMemo } from 'react';
import { useEntityStore } from '../stores/entityStore';
import type { Scene } from '../types/scene';

export interface UseScenesOptions {
  filterByOwner?: string;
  filterByVisibility?: 'public' | 'private' | 'unlisted';
  includeLoading?: boolean;
}

export interface UseScenesResult {
  scenes: Scene[];
  activeCount: number;
  loading: boolean;
}

/**
 * Hook to access and filter scenes from cache
 * Provides derived data like active count
 */
export function useScenes(options: UseScenesOptions = {}): UseScenesResult {
  const { filterByOwner, filterByVisibility, includeLoading = false } = options;

  const cachedScenes = useEntityStore((state) => state.scene.scenes);

  // Memoized filtered and sorted scenes
  const scenes = useMemo(() => {
    const sceneList = Object.values(cachedScenes)
      .filter((cached) => {
        // Skip loading entries unless explicitly requested
        if (!includeLoading && cached.metadata.loading) return false;
        
        // Skip entries with errors or no data
        if (cached.metadata.error || !cached.data.id) return false;

        return true;
      })
      .map((cached) => cached.data);

    // Apply filters
    let filtered = sceneList;

    if (filterByOwner) {
      filtered = filtered.filter((scene) => scene.owner_user_id === filterByOwner);
    }

    if (filterByVisibility) {
      filtered = filtered.filter((scene) => scene.visibility === filterByVisibility);
    }

    return filtered;
  }, [cachedScenes, filterByOwner, filterByVisibility, includeLoading]);

  // Calculate active scenes count (public or unlisted)
  const activeCount = useMemo(() => {
    return scenes.filter((scene) => scene.visibility !== 'private').length;
  }, [scenes]);

  // Check if any scene is loading
  const loading = useMemo(() => {
    return Object.values(cachedScenes).some((cached) => cached.metadata.loading);
  }, [cachedScenes]);

  return {
    scenes,
    activeCount,
    loading,
  };
}

/**
 * Hook to get a specific user's scenes
 */
export function useUserScenes(userId: string | undefined): UseScenesResult {
  const result = useScenes({ filterByOwner: userId });
  
  // Return empty results if userId is undefined
  if (!userId) {
    return {
      scenes: [],
      activeCount: 0,
      loading: false,
    };
  }
  
  return result;
}

/**
 * Hook to get public scenes
 */
export function usePublicScenes(): UseScenesResult {
  return useScenes({ filterByVisibility: 'public' });
}
