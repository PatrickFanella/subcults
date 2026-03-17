/**
 * Theme Store Tests
 * Validates dark mode state management
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useThemeStore, useTheme, useThemeActions } from './themeStore';

describe('themeStore', () => {
  beforeEach(() => {
    // Clear localStorage
    localStorage.clear();
    // Reset store to initial state
    useThemeStore.setState({ theme: 'light' });
    // Remove dark class from document
    document.documentElement.classList.remove('dark');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getInitialTheme', () => {
    it('returns stored theme from localStorage', () => {
      localStorage.setItem('subcults-theme', 'dark');
      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.initializeTheme();
      });

      expect(result.current.theme).toBe('dark');
      // Should still be in localStorage since it was manually set
      expect(localStorage.getItem('subcults-theme')).toBe('dark');
    });

    it('falls back to system preference when no stored theme', () => {
      // Mock prefers-color-scheme: dark
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation((query: string) => ({
          matches: query === '(prefers-color-scheme: dark)',
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })),
      });

      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.initializeTheme();
      });

      expect(result.current.theme).toBe('dark');
      // Should NOT persist to localStorage when derived from system preference
      expect(localStorage.getItem('subcults-theme')).toBeNull();
    });

    it('defaults to light mode when no preference available', () => {
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation((query: string) => ({
          matches: false,
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })),
      });

      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.initializeTheme();
      });

      expect(result.current.theme).toBe('light');
      // Should NOT persist to localStorage when derived from system preference
      expect(localStorage.getItem('subcults-theme')).toBeNull();
    });
  });

  describe('setTheme', () => {
    it('updates theme state', () => {
      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.setTheme('dark');
      });

      expect(result.current.theme).toBe('dark');
    });

    it('adds dark class to document when theme is dark', () => {
      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.setTheme('dark');
      });

      expect(document.documentElement.classList.contains('dark')).toBe(true);
    });

    it('removes dark class from document when theme is light', () => {
      document.documentElement.classList.add('dark');
      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.setTheme('light');
      });

      expect(document.documentElement.classList.contains('dark')).toBe(false);
    });

    it('persists theme to localStorage', () => {
      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.setTheme('dark');
      });

      expect(localStorage.getItem('subcults-theme')).toBe('dark');
    });
  });

  describe('toggleTheme', () => {
    it('toggles from light to dark', () => {
      const { result } = renderHook(() => useThemeStore());
      
      act(() => {
        result.current.setTheme('light');
      });

      act(() => {
        result.current.toggleTheme();
      });

      expect(result.current.theme).toBe('dark');
    });

    it('toggles from dark to light', () => {
      const { result } = renderHook(() => useThemeStore());
      
      act(() => {
        result.current.setTheme('dark');
      });

      act(() => {
        result.current.toggleTheme();
      });

      expect(result.current.theme).toBe('light');
    });

    it('updates document and localStorage when toggling', () => {
      const { result } = renderHook(() => useThemeStore());
      
      act(() => {
        result.current.setTheme('light');
      });

      act(() => {
        result.current.toggleTheme();
      });

      expect(document.documentElement.classList.contains('dark')).toBe(true);
      expect(localStorage.getItem('subcults-theme')).toBe('dark');
    });
  });

  describe('useTheme hook', () => {
    it('returns current theme', () => {
      useThemeStore.setState({ theme: 'dark' });
      const { result } = renderHook(() => useTheme());

      expect(result.current).toBe('dark');
    });
  });

  describe('useThemeActions hook', () => {
    it('returns stable action references', () => {
      const { result, rerender } = renderHook(() => useThemeActions());
      const firstRender = result.current;

      rerender();
      const secondRender = result.current;

      // Actions should maintain referential equality
      expect(firstRender.setTheme).toBe(secondRender.setTheme);
      expect(firstRender.toggleTheme).toBe(secondRender.toggleTheme);
      expect(firstRender.initializeTheme).toBe(secondRender.initializeTheme);
    });
  });
});
