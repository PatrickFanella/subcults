/**
 * useSearchHistory Hook
 * Hook for managing search history with localStorage persistence
 */

import { useState, useCallback, useEffect } from 'react';

const SEARCH_HISTORY_KEY = 'subcults-search-history';
const MAX_HISTORY_ITEMS = 5;

export interface SearchHistoryItem {
  query: string;
  timestamp: number;
}

export interface UseSearchHistoryResult {
  history: SearchHistoryItem[];
  addToHistory: (query: string) => void;
  removeFromHistory: (query: string) => void;
  clearHistory: () => void;
}

/**
 * Load search history from localStorage
 */
function loadHistory(): SearchHistoryItem[] {
  try {
    const stored = localStorage.getItem(SEARCH_HISTORY_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      if (Array.isArray(parsed)) {
        return parsed;
      }
    }
  } catch (error) {
    console.warn('[useSearchHistory] Failed to load history from localStorage:', error);
  }
  return [];
}

/**
 * Save search history to localStorage
 */
function saveHistory(history: SearchHistoryItem[]): void {
  try {
    localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(history));
  } catch (error) {
    console.warn('[useSearchHistory] Failed to save history to localStorage:', error);
  }
}

/**
 * Hook for managing search history with localStorage persistence
 * Automatically loads history on mount and persists changes
 */
export function useSearchHistory(): UseSearchHistoryResult {
  const [history, setHistory] = useState<SearchHistoryItem[]>([]);

  // Load history on mount
  useEffect(() => {
    const loaded = loadHistory();
    setHistory(loaded);
  }, []);

  /**
   * Add a search query to history
   * Deduplicates entries and limits to MAX_HISTORY_ITEMS
   */
  const addToHistory = useCallback((query: string) => {
    const trimmed = query.trim();
    if (!trimmed) return;

    setHistory((prev) => {
      // Remove any existing entry with the same query
      const filtered = prev.filter((item) => item.query !== trimmed);

      // Add new entry at the beginning
      const updated = [
        { query: trimmed, timestamp: Date.now() },
        ...filtered,
      ].slice(0, MAX_HISTORY_ITEMS);

      // Persist to localStorage
      saveHistory(updated);

      return updated;
    });
  }, []);

  /**
   * Remove a specific query from history
   */
  const removeFromHistory = useCallback((query: string) => {
    setHistory((prev) => {
      const updated = prev.filter((item) => item.query !== query);
      saveHistory(updated);
      return updated;
    });
  }, []);

  /**
   * Clear all search history
   */
  const clearHistory = useCallback(() => {
    setHistory([]);
    saveHistory([]);
  }, []);

  return {
    history,
    addToHistory,
    removeFromHistory,
    clearHistory,
  };
}
