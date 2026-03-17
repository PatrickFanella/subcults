/**
 * useSearch Hook
 * Hook for debounced global search with request cancellation
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { apiClient } from '../lib/api-client';
import type { SearchResults } from '../types/search';

const DEBOUNCE_MS = 300;
const DEFAULT_LIMIT = 5; // Maximum results per category (scenes, events, posts)

export interface UseSearchOptions {
  /**
   * Debounce delay in milliseconds (default: 300)
   */
  debounceMs?: number;
  /**
   * Maximum results per category (default: 5)
   */
  limit?: number;
}

export interface UseSearchResult {
  results: SearchResults;
  loading: boolean;
  error: string | null;
  search: (query: string) => void;
  clear: () => void;
}

/**
 * Hook for performing debounced global search across scenes, events, and posts
 * Automatically cancels in-flight requests when a new search is triggered
 */
export function useSearch(options: UseSearchOptions = {}): UseSearchResult {
  const {
    debounceMs = DEBOUNCE_MS,
    limit = DEFAULT_LIMIT,
  } = options;

  const [results, setResults] = useState<SearchResults>({
    scenes: [],
    events: [],
    posts: [],
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Track debounce timeout
  const debounceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Track abort controller for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null);

  /**
   * Execute search with parallel requests
   */
  const executeSearch = useCallback(
    async (searchQuery: string) => {
      if (!searchQuery.trim()) {
        setResults({ scenes: [], events: [], posts: [] });
        setLoading(false);
        setError(null);
        return;
      }

      // Cancel any in-flight request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      // Create new abort controller
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      setLoading(true);
      setError(null);

      try {
        // Execute all searches in parallel with concurrency limit (Promise.all handles this)
        const [scenes, events, posts] = await Promise.all([
          apiClient.searchScenes(searchQuery, limit, abortController.signal).catch(() => []),
          apiClient.searchEvents(searchQuery, limit, abortController.signal).catch(() => []),
          apiClient.searchPosts(searchQuery, limit, abortController.signal).catch(() => []),
        ]);

        // Only update state if this request wasn't cancelled
        if (!abortController.signal.aborted) {
          setResults({ scenes, events, posts });
          setLoading(false);
        }
      } catch (err) {
        // Only handle error if request wasn't cancelled
        if (!abortController.signal.aborted) {
          setError(err instanceof Error ? err.message : 'Search failed');
          setLoading(false);
        }
      }
    },
    [limit]
  );

  /**
   * Debounced search function
   */
  const search = useCallback(
    (newQuery: string) => {
      // Clear existing timeout
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }

      // Set new timeout for debounced execution
      debounceTimeoutRef.current = setTimeout(() => {
        executeSearch(newQuery);
      }, debounceMs);
    },
    [debounceMs, executeSearch]
  );

  /**
   * Clear search results and cancel any in-flight requests
   */
  const clear = useCallback(() => {
    // Cancel in-flight request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    // Clear debounce timeout
    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current);
    }

    setResults({ scenes: [], events: [], posts: [] });
    setLoading(false);
    setError(null);
  }, []);

  /**
   * Cleanup on unmount
   */
  useEffect(() => {
    return () => {
      // Cancel in-flight request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      // Clear debounce timeout
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }
    };
  }, []);

  return {
    results,
    loading,
    error,
    search,
    clear,
  };
}
