/**
 * useSearch Hook Tests
 * Validates debounced search behavior, cancellation, and state management
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react';
import { useSearch } from './useSearch';
import { apiClient } from '../lib/api-client';

// Mock the API client
vi.mock('../lib/api-client', () => ({
  apiClient: {
    searchScenes: vi.fn(),
    searchEvents: vi.fn(),
    searchPosts: vi.fn(),
  },
}));

describe('useSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Default mock responses
    vi.mocked(apiClient.searchScenes).mockResolvedValue([]);
    vi.mocked(apiClient.searchEvents).mockResolvedValue([]);
    vi.mocked(apiClient.searchPosts).mockResolvedValue([]);
  });

  afterEach(() => {
    vi.clearAllTimers();
  });

  describe('Initial State', () => {
    it('returns empty results initially', () => {
      const { result } = renderHook(() => useSearch());

      expect(result.current.results).toEqual({
        scenes: [],
        events: [],
        posts: [],
      });
      expect(result.current.loading).toBe(false);
      expect(result.current.error).toBe(null);
    });
  });

  describe('Debounce Behavior', () => {
    it('debounces search calls with default delay', async () => {
      const { result } = renderHook(() => useSearch());

      // Start search
      await act(async () => {
        result.current.search('test');
      });

      // Should not call API immediately
      expect(apiClient.searchScenes).not.toHaveBeenCalled();

      // Wait for debounce
      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalledWith('test', 5, expect.any(AbortSignal));
        },
        { timeout: 500 }
      );
    });

    it('uses custom debounce delay', async () => {
      const { result } = renderHook(() => useSearch({ debounceMs: 100 }));

      await act(async () => {
        result.current.search('test');
      });

      // Should call after custom delay
      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalled();
        },
        { timeout: 300 }
      );
    });

    it('cancels previous search when new query is typed', async () => {
      const { result } = renderHook(() => useSearch());

      // Start first search
      await act(async () => {
        result.current.search('first');
      });

      // Start second search before first completes
      await act(async () => {
        result.current.search('second');
      });

      // Wait for debounce
      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalledTimes(1);
          expect(apiClient.searchScenes).toHaveBeenCalledWith('second', 5, expect.any(AbortSignal));
        },
        { timeout: 500 }
      );
    });
  });

  describe('Parallel Search Execution', () => {
    it('executes all searches in parallel', async () => {
      const { result } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalledWith('test', 5, expect.any(AbortSignal));
          expect(apiClient.searchEvents).toHaveBeenCalledWith('test', 5, expect.any(AbortSignal));
          expect(apiClient.searchPosts).toHaveBeenCalledWith('test', 5, expect.any(AbortSignal));
        },
        { timeout: 500 }
      );
    });

    it('uses custom limit', async () => {
      const { result } = renderHook(() => useSearch({ limit: 10 }));

      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalledWith('test', 10, expect.any(AbortSignal));
          expect(apiClient.searchEvents).toHaveBeenCalledWith('test', 10, expect.any(AbortSignal));
          expect(apiClient.searchPosts).toHaveBeenCalledWith('test', 10, expect.any(AbortSignal));
        },
        { timeout: 500 }
      );
    });
  });

  describe('Results Handling', () => {
    it('updates results when searches complete', async () => {
      const mockScenes = [{ id: '1', name: 'Scene 1', allow_precise: true, coarse_geohash: 'abc' }];
      const mockEvents = [{ id: '2', scene_id: 's1', name: 'Event 1', allow_precise: true }];
      const mockPosts = [{ id: '3', content: 'Post 1' }];

      vi.mocked(apiClient.searchScenes).mockResolvedValue(mockScenes);
      vi.mocked(apiClient.searchEvents).mockResolvedValue(mockEvents);
      vi.mocked(apiClient.searchPosts).mockResolvedValue(mockPosts);

      const { result } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(result.current.results).toEqual({
            scenes: mockScenes,
            events: mockEvents,
            posts: mockPosts,
          });
        },
        { timeout: 500 }
      );
    });

    it('sets loading state during search', async () => {
      let resolveSearch: (value: unknown) => void;
      const searchPromise = new Promise<void>((resolve) => {
        resolveSearch = resolve;
      });

      vi.mocked(apiClient.searchScenes).mockReturnValue(searchPromise as Promise<unknown>);
      vi.mocked(apiClient.searchEvents).mockReturnValue(searchPromise as Promise<unknown>);
      vi.mocked(apiClient.searchPosts).mockReturnValue(searchPromise as Promise<unknown>);

      const { result } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(result.current.loading).toBe(true);
        },
        { timeout: 500 }
      );

      // Resolve search
      await act(async () => {
        resolveSearch!([]);
      });

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
    });

    it('handles individual endpoint failures gracefully', async () => {
      const mockScenes = [{ id: '1', name: 'Scene 1', allow_precise: true, coarse_geohash: 'abc' }];

      vi.mocked(apiClient.searchScenes).mockResolvedValue(mockScenes);
      vi.mocked(apiClient.searchEvents).mockRejectedValue(new Error('API error'));
      vi.mocked(apiClient.searchPosts).mockResolvedValue([]);

      const { result } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('test');
      });

      // Should still return successful results
      await waitFor(
        () => {
          expect(result.current.results.scenes).toEqual(mockScenes);
          expect(result.current.results.events).toEqual([]);
          expect(result.current.results.posts).toEqual([]);
          expect(result.current.loading).toBe(false);
        },
        { timeout: 500 }
      );
    });
  });

  describe('Request Cancellation', () => {
    it('aborts in-flight request when new search starts', async () => {
      let abortSignal: AbortSignal | undefined;

      vi.mocked(apiClient.searchScenes).mockImplementation(async (query, limit, signal) => {
        abortSignal = signal;
        // Never resolve
        return new Promise(() => {});
      });

      const { result } = renderHook(() => useSearch());

      // Start first search
      await act(async () => {
        result.current.search('first');
      });

      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalled();
        },
        { timeout: 500 }
      );

      const firstSignal = abortSignal;

      // Start second search
      await act(async () => {
        result.current.search('second');
      });

      // First signal should be aborted
      await waitFor(
        () => {
          expect(firstSignal?.aborted).toBe(true);
        },
        { timeout: 500 }
      );
    });

    it('does not update state if request was cancelled', async () => {
      let resolveFirst: (value: unknown) => void;
      const firstPromise = new Promise<void>((resolve) => {
        resolveFirst = resolve;
      });

      vi.mocked(apiClient.searchScenes).mockReturnValueOnce(firstPromise as Promise<unknown>);
      vi.mocked(apiClient.searchEvents).mockReturnValueOnce(firstPromise as Promise<unknown>);
      vi.mocked(apiClient.searchPosts).mockReturnValueOnce(firstPromise as Promise<unknown>);

      const mockScenes = [{ id: '2', name: 'Scene 2', allow_precise: true, coarse_geohash: 'def' }];
      vi.mocked(apiClient.searchScenes).mockResolvedValue(mockScenes);
      vi.mocked(apiClient.searchEvents).mockResolvedValue([]);
      vi.mocked(apiClient.searchPosts).mockResolvedValue([]);

      const { result } = renderHook(() => useSearch());

      // Start first search
      await act(async () => {
        result.current.search('first');
      });

      await waitFor(
        () => {
          expect(result.current.loading).toBe(true);
        },
        { timeout: 500 }
      );

      // Start second search (cancels first)
      await act(async () => {
        result.current.search('second');
      });

      // Resolve first search (should be ignored)
      await act(async () => {
        resolveFirst!([{ id: '1', name: 'Scene 1', allow_precise: true, coarse_geohash: 'abc' }]);
      });

      // Wait for second search
      await waitFor(
        () => {
          expect(result.current.results.scenes).toEqual(mockScenes);
        },
        { timeout: 1000 }
      );
    });
  });

  describe('Clear Functionality', () => {
    it('clears results and state', async () => {
      const mockScenes = [{ id: '1', name: 'Scene 1', allow_precise: true, coarse_geohash: 'abc' }];
      vi.mocked(apiClient.searchScenes).mockResolvedValue(mockScenes);
      vi.mocked(apiClient.searchEvents).mockResolvedValue([]);
      vi.mocked(apiClient.searchPosts).mockResolvedValue([]);

      const { result } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(result.current.results.scenes.length).toBeGreaterThan(0);
        },
        { timeout: 500 }
      );

      act(() => {
        result.current.clear();
      });

      expect(result.current.results).toEqual({
        scenes: [],
        events: [],
        posts: [],
      });
      expect(result.current.loading).toBe(false);
      expect(result.current.error).toBe(null);
    });

    it('cancels in-flight request when clearing', async () => {
      let abortSignal: AbortSignal | undefined;

      vi.mocked(apiClient.searchScenes).mockImplementation(async (query, limit, signal) => {
        abortSignal = signal;
        return new Promise(() => {}); // Never resolve
      });

      const { result } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalled();
        },
        { timeout: 500 }
      );

      act(() => {
        result.current.clear();
      });

      expect(abortSignal?.aborted).toBe(true);
    });

    it('clears pending debounced search', async () => {
      const { result } = renderHook(() => useSearch());

      act(() => {
        result.current.search('test');
      });

      // Clear before debounce completes
      act(() => {
        result.current.clear();
      });

      // Wait for what would have been the debounce delay
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Should not have called API
      expect(apiClient.searchScenes).not.toHaveBeenCalled();
    });
  });

  describe('Empty Query Handling', () => {
    it('clears results when query is empty', async () => {
      const mockScenes = [{ id: '1', name: 'Scene 1', allow_precise: true, coarse_geohash: 'abc' }];
      vi.mocked(apiClient.searchScenes).mockResolvedValue(mockScenes);
      vi.mocked(apiClient.searchEvents).mockResolvedValue([]);
      vi.mocked(apiClient.searchPosts).mockResolvedValue([]);

      const { result } = renderHook(() => useSearch());

      // Search with query
      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(result.current.results.scenes.length).toBeGreaterThan(0);
        },
        { timeout: 500 }
      );

      // Search with empty query
      await act(async () => {
        result.current.search('');
      });

      await waitFor(() => {
        expect(result.current.results).toEqual({
          scenes: [],
          events: [],
          posts: [],
        });
      });
    });

    it('does not call API for whitespace-only query', async () => {
      const { result } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('   ');
      });

      // Wait for what would have been the debounce delay
      await act(async () => {
        await new Promise((resolve) => setTimeout(resolve, 500));
      });

      expect(apiClient.searchScenes).not.toHaveBeenCalled();
    });
  });

  describe('Cleanup', () => {
    it('cancels in-flight requests on unmount', async () => {
      let abortSignal: AbortSignal | undefined;

      vi.mocked(apiClient.searchScenes).mockImplementation(async (query, limit, signal) => {
        abortSignal = signal;
        return new Promise(() => {}); // Never resolve
      });

      const { result, unmount } = renderHook(() => useSearch());

      await act(async () => {
        result.current.search('test');
      });

      await waitFor(
        () => {
          expect(apiClient.searchScenes).toHaveBeenCalled();
        },
        { timeout: 500 }
      );

      unmount();

      expect(abortSignal?.aborted).toBe(true);
    });

    it('clears debounce timeout on unmount', async () => {
      const { result, unmount } = renderHook(() => useSearch());

      act(() => {
        result.current.search('test');
      });

      unmount();

      // Wait for what would have been the debounce delay
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Should not have called API after unmount
      expect(apiClient.searchScenes).not.toHaveBeenCalled();
    });
  });
});
