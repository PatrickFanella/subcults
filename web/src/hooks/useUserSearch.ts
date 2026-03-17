/**
 * useUserSearch Hook
 * Hook for searching users with trust-based filtering and ranking
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { apiClient } from '../lib/api-client';

/**
 * User search result with trust information
 */
export interface UserSearchResult {
  /** User DID (Decentralized Identifier) */
  id: string;
  
  /** Display name */
  name?: string;
  
  /** User avatar URL */
  avatar?: string;
  
  /** User bio or description */
  bio?: string;
  
  /** Trust score (0-1) based on alliances */
  trustScore?: number;
  
  /** Number of followers */
  followers?: number;
  
  /** Number of scenes hosted */
  scenesHosted?: number;
  
  /** Verification status */
  verified?: boolean;
}

/**
 * User search filters
 */
export interface UserSearchFilters {
  /** Minimum trust score (0-1) */
  minTrustScore?: number;
  
  /** Filter by user role */
  role?: 'organizer' | 'artist' | 'promoter' | 'venue';
  
  /** Filter by verification status */
  verified?: boolean;
  
  /** Minimum followers */
  minFollowers?: number;
}

/**
 * Result from useUserSearch hook
 */
export interface UseUserSearchResult {
  /** Search results */
  results: UserSearchResult[];
  
  /** Whether search is loading */
  loading: boolean;
  
  /** Error message if search failed */
  error: string | null;
  
  /** Function to perform search */
  search: (query: string, filters?: UserSearchFilters) => void;
  
  /** Function to clear results */
  clear: () => void;
}

const DEBOUNCE_MS = 300;
const DEFAULT_LIMIT = 10;

/**
 * Hook for searching users with trust-based filtering
 * 
 * @param options - Configuration options
 * @returns Object with results, loading state, and search function
 * 
 * @example
 * ```tsx
 * const { results, search, loading } = useUserSearch();
 * 
 * const handleSearch = (query: string) => {
 *   search(query, {
 *     minTrustScore: 0.5,
 *     role: 'artist',
 *   });
 * };
 * ```
 */
export function useUserSearch(options: {
  /** Debounce delay in ms */
  debounceMs?: number;
  /** Max results to return */
  limit?: number;
} = {}): UseUserSearchResult {
  const {
    debounceMs = DEBOUNCE_MS,
    limit = DEFAULT_LIMIT,
  } = options;

  const [results, setResults] = useState<UserSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const lastQueryRef = useRef<{ query: string; filters?: UserSearchFilters } | null>(null);

  /**
   * Execute user search
   */
  const executeSearch = useCallback(
    async (query: string, filters?: UserSearchFilters) => {
      if (!query.trim()) {
        setResults([]);
        setLoading(false);
        setError(null);
        return;
      }

      // Cancel any in-flight request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      setLoading(true);
      setError(null);

      try {
        const params = new URLSearchParams({
          q: query,
          limit: limit.toString(),
        });

        if (filters?.minTrustScore !== undefined) {
          params.append('minTrustScore', filters.minTrustScore.toString());
        }
        if (filters?.role) {
          params.append('role', filters.role);
        }
        if (filters?.verified !== undefined) {
          params.append('verified', filters.verified.toString());
        }
        if (filters?.minFollowers !== undefined) {
          params.append('minFollowers', filters.minFollowers.toString());
        }

        const data = await apiClient.get<{ results?: UserSearchResult[] }>(
          `/users/search?${params}`,
          { signal: abortController.signal, skipAutoRetry: true },
        );

        // Only update if this request wasn't cancelled
        if (!abortController.signal.aborted) {
          const resultsData = data.results ?? [];
          setResults(resultsData);
          setError(null);
        }
      } catch (err) {
        if (!abortController.signal.aborted) {
          setError(err instanceof Error ? err.message : 'Search failed');
          setLoading(false);
        }
      } finally {
        setLoading(false);
      }
    },
    [limit]
  );

  /**
   * Public search function with debounce
   */
  const search = useCallback(
    (query: string, filters?: UserSearchFilters) => {
      lastQueryRef.current = { query, filters };

      // Clear existing timer
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }

      // Debounce the search
      debounceTimerRef.current = setTimeout(() => {
        executeSearch(query, filters);
      }, debounceMs);
    },
    [debounceMs, executeSearch]
  );

  /**
   * Clear results
   */
  const clear = useCallback(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    setResults([]);
    setLoading(false);
    setError(null);
    lastQueryRef.current = null;
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
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
