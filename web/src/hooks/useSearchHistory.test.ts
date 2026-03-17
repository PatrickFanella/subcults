/**
 * useSearchHistory Tests
 * Validates search history management and localStorage persistence
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSearchHistory } from './useSearchHistory';

describe('useSearchHistory', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear();
    vi.clearAllMocks();
  });

  describe('Initial State', () => {
    it('starts with empty history when localStorage is empty', () => {
      const { result } = renderHook(() => useSearchHistory());
      expect(result.current.history).toEqual([]);
    });

    it('loads history from localStorage on mount', () => {
      const mockHistory = [
        { query: 'test query', timestamp: Date.now() },
        { query: 'another search', timestamp: Date.now() - 1000 },
      ];
      localStorage.setItem('subcults-search-history', JSON.stringify(mockHistory));

      const { result } = renderHook(() => useSearchHistory());
      expect(result.current.history).toEqual(mockHistory);
    });

    it('handles corrupted localStorage data gracefully', () => {
      localStorage.setItem('subcults-search-history', 'invalid json');

      const { result } = renderHook(() => useSearchHistory());
      expect(result.current.history).toEqual([]);
    });

    it('handles non-array localStorage data gracefully', () => {
      localStorage.setItem('subcults-search-history', JSON.stringify({ invalid: 'data' }));

      const { result } = renderHook(() => useSearchHistory());
      expect(result.current.history).toEqual([]);
    });
  });

  describe('addToHistory', () => {
    it('adds a new search query to history', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('test query');
      });

      expect(result.current.history).toHaveLength(1);
      expect(result.current.history[0].query).toBe('test query');
      expect(result.current.history[0].timestamp).toBeGreaterThan(0);
    });

    it('adds new items to the beginning of the list', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('first query');
      });

      act(() => {
        result.current.addToHistory('second query');
      });

      expect(result.current.history[0].query).toBe('second query');
      expect(result.current.history[1].query).toBe('first query');
    });

    it('trims whitespace from queries', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('  test query  ');
      });

      expect(result.current.history[0].query).toBe('test query');
    });

    it('ignores empty or whitespace-only queries', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('');
      });

      act(() => {
        result.current.addToHistory('   ');
      });

      expect(result.current.history).toHaveLength(0);
    });

    it('deduplicates queries and moves them to the top', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('first query');
      });

      act(() => {
        result.current.addToHistory('second query');
      });

      act(() => {
        result.current.addToHistory('first query');
      });

      expect(result.current.history).toHaveLength(2);
      expect(result.current.history[0].query).toBe('first query');
      expect(result.current.history[1].query).toBe('second query');
    });

    it('limits history to 5 items', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('query 1');
        result.current.addToHistory('query 2');
        result.current.addToHistory('query 3');
        result.current.addToHistory('query 4');
        result.current.addToHistory('query 5');
        result.current.addToHistory('query 6');
      });

      expect(result.current.history).toHaveLength(5);
      expect(result.current.history[0].query).toBe('query 6');
      expect(result.current.history[4].query).toBe('query 2');
    });

    it('persists to localStorage', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('test query');
      });

      const stored = localStorage.getItem('subcults-search-history');
      expect(stored).toBeTruthy();

      const parsed = JSON.parse(stored!);
      expect(parsed).toHaveLength(1);
      expect(parsed[0].query).toBe('test query');
    });
  });

  describe('removeFromHistory', () => {
    it('removes a specific query from history', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('first query');
        result.current.addToHistory('second query');
        result.current.addToHistory('third query');
      });

      act(() => {
        result.current.removeFromHistory('second query');
      });

      expect(result.current.history).toHaveLength(2);
      expect(result.current.history.find((item) => item.query === 'second query')).toBeUndefined();
    });

    it('does nothing if query is not in history', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('test query');
      });

      act(() => {
        result.current.removeFromHistory('nonexistent query');
      });

      expect(result.current.history).toHaveLength(1);
    });

    it('persists removal to localStorage', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('first query');
        result.current.addToHistory('second query');
      });

      act(() => {
        result.current.removeFromHistory('first query');
      });

      const stored = localStorage.getItem('subcults-search-history');
      const parsed = JSON.parse(stored!);
      expect(parsed).toHaveLength(1);
      expect(parsed[0].query).toBe('second query');
    });
  });

  describe('clearHistory', () => {
    it('clears all history', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('first query');
        result.current.addToHistory('second query');
      });

      act(() => {
        result.current.clearHistory();
      });

      expect(result.current.history).toEqual([]);
    });

    it('clears localStorage', () => {
      const { result } = renderHook(() => useSearchHistory());

      act(() => {
        result.current.addToHistory('test query');
      });

      act(() => {
        result.current.clearHistory();
      });

      const stored = localStorage.getItem('subcults-search-history');
      const parsed = JSON.parse(stored!);
      expect(parsed).toEqual([]);
    });
  });
});
