/**
 * usePaginatedFeed Hook
 * React hook for handling cursor-based pagination with infinite scroll support
 * 
 * Features:
 * - Cursor-based pagination for efficient data retrieval
 * - Automatic duplicate detection and filtering
 * - Memory-efficient incremental loading
 * - Abort controller for request cancellation
 * - Proper error handling and recovery
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import type { Scene, Event } from '../types/scene';

/**
 * Pagination cursor for resuming data fetches
 */
export interface PaginationCursor {
  /** Cursor token for next page (from API response) */
  next?: string;
  /** Whether more data is available */
  hasMore: boolean;
}

/**
 * Result from usePaginatedFeed hook
 */
export interface UsePaginatedFeedResult<T> {
  /** Current loaded items */
  items: T[];
  
  /** Whether initial fetch is loading */
  loading: boolean;
  
  /** Whether appending more items */
  loadingMore: boolean;
  
  /** Error message if fetch failed */
  error: string | null;
  
  /** Function to load next page */
  loadMore: () => Promise<void>;
  
  /** Function to reset and reload from start */
  reset: () => Promise<void>;
  
  /** Whether more data is available */
  hasMore: boolean;
}

/**
 * Hook for paginated feed with incremental loading
 * 
 * @param fetchFn - Function to fetch items (should return { items, cursor })
 * @param pageSize - Items per page (default: 20)
 * 
 * @example
 * ```tsx
 * const { items, loading, loadMore, hasMore } = usePaginatedFeed(
 *   async (cursor, signal) => {
 *     const response = await fetch(`/api/scenes?limit=20&cursor=${cursor}`, { signal });
 *     return response.json();
 *   }
 * );
 * 
 * return (
 *   <InfiniteScroll dataLength={items.length} next={loadMore} hasMore={hasMore}>
 *     {items.map(item => <SceneCard key={item.id} scene={item} />)}
 *   </InfiniteScroll>
 * );
 * ```
 */
export function usePaginatedFeed<T extends { id: string }>(
  fetchFn: (cursor?: string, signal?: AbortSignal) => Promise<{ items: T[]; cursor: PaginationCursor }>,
): UsePaginatedFeedResult<T> {
  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cursor, setCursor] = useState<PaginationCursor>({ hasMore: true });
  
  // Use ref to track seen IDs for deduplication
  const seenIdsRef = useRef(new Set<string>());
  const fetchCounterRef = useRef(0);
  const abortControllerRef = useRef<AbortController | null>(null);

  /**
   * Fetch initial items
   */
  const reset = useCallback(async () => {
    // Cancel any pending requests
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    const abortController = new AbortController();
    abortControllerRef.current = abortController;
    
    // Clear state
    setItems([]);
    seenIdsRef.current.clear();
    setCursor({ hasMore: true });
    setLoading(true);
    setError(null);

    try {
      const fetchId = `feed-fetch-${++fetchCounterRef.current}`;
      performance.mark(`${fetchId}-start`);
      
      const result = await fetchFn(undefined, abortController.signal);
      
      performance.mark(`${fetchId}-end`);
      performance.measure(`${fetchId}-total`, `${fetchId}-start`, `${fetchId}-end`);
      
      // Deduplicate and store items
      const newItems = result.items.filter((item) => {
        if (seenIdsRef.current.has(item.id)) {
          return false;
        }
        seenIdsRef.current.add(item.id);
        return true;
      });

      setItems(newItems);
      setCursor(result.cursor);
      setError(null);
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError(err.message);
      }
    } finally {
      setLoading(false);
    }
  }, [fetchFn]);

  /**
   * Load next page of items
   */
  const loadMore = useCallback(async () => {
    if (!cursor.hasMore || loadingMore) {
      return;
    }

    setLoadingMore(true);
    setError(null);

    // Create a new abort controller for this loadMore request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    try {
      const fetchId = `feed-more-${++fetchCounterRef.current}`;
      performance.mark(`${fetchId}-start`);
      
      const result = await fetchFn(cursor.next, abortController.signal);
      
      performance.mark(`${fetchId}-end`);
      performance.measure(`${fetchId}-total`, `${fetchId}-start`, `${fetchId}-end`);
      
      // Deduplicate new items against existing ones
      const newItems = result.items.filter((item) => {
        if (seenIdsRef.current.has(item.id)) {
          return false;
        }
        seenIdsRef.current.add(item.id);
        return true;
      });

      // Append to existing items
      setItems((prevItems) => [...prevItems, ...newItems]);
      setCursor(result.cursor);
      setError(null);
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError(err.message);
      }
    } finally {
      setLoadingMore(false);
    }
  }, [cursor, fetchFn, loadingMore]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  // Initial fetch
  useEffect(() => {
    reset();
  }, [reset]);

  return {
    items,
    loading,
    loadingMore,
    error,
    loadMore,
    reset,
    hasMore: cursor.hasMore,
  };
}

/**
 * Hook for paginated scenes with search/filtering
 */
export function usePaginatedScenes(
  filters?: {
    tags?: string[];
    visibility?: 'public' | 'private' | 'unlisted';
    ownerId?: string;
  }
): UsePaginatedFeedResult<Scene> {
  const apiUrl = import.meta.env.VITE_API_URL || '/api';
  
  const fetchFn = useCallback(
    async (cursor?: string, signal?: AbortSignal) => {
      const params = new URLSearchParams();
      
      if (cursor) {
        params.append('cursor', cursor);
      }
      if (filters?.tags?.length) {
        params.append('tags', filters.tags.join(','));
      }
      if (filters?.visibility) {
        params.append('visibility', filters.visibility);
      }
      if (filters?.ownerId) {
        params.append('owner_id', filters.ownerId);
      }
      params.append('limit', '20');
      
      const response = await fetch(`${apiUrl}/scenes?${params}`, { signal });
      if (!response.ok) {
        throw new Error(`Failed to fetch scenes: ${response.statusText}`);
      }
      
      const data = await response.json();
      return {
        items: Array.isArray(data) ? data : data.items || [],
        cursor: {
          next: data.next_cursor,
          hasMore: !!data.next_cursor,
        },
      };
    },
    [apiUrl, filters]
  );

  return usePaginatedFeed(fetchFn);
}

/**
 * Hook for paginated events with search/filtering
 */
export function usePaginatedEvents(
  filters?: {
    sceneId?: string;
    status?: 'scheduled' | 'live' | 'ended';
    visibility?: 'public' | 'private' | 'unlisted';
  }
): UsePaginatedFeedResult<Event> {
  const apiUrl = import.meta.env.VITE_API_URL || '/api';
  
  const fetchFn = useCallback(
    async (cursor?: string, signal?: AbortSignal) => {
      const params = new URLSearchParams();
      
      if (cursor) {
        params.append('cursor', cursor);
      }
      if (filters?.sceneId) {
        params.append('scene_id', filters.sceneId);
      }
      if (filters?.status) {
        params.append('status', filters.status);
      }
      if (filters?.visibility) {
        params.append('visibility', filters.visibility);
      }
      params.append('limit', '20');
      
      const response = await fetch(`${apiUrl}/events?${params}`, { signal });
      if (!response.ok) {
        throw new Error(`Failed to fetch events: ${response.statusText}`);
      }
      
      const data = await response.json();
      return {
        items: Array.isArray(data) ? data : data.items || [],
        cursor: {
          next: data.next_cursor,
          hasMore: !!data.next_cursor,
        },
      };
    },
    [apiUrl, filters]
  );

  return usePaginatedFeed(fetchFn);
}
