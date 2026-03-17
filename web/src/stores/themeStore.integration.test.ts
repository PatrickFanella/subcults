/**
 * Theme Store Integration Tests
 * End-to-end tests for dark mode functionality across the app
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useThemeStore } from './themeStore';

describe('Dark Mode Integration', () => {
  beforeEach(() => {
    localStorage.clear();
    useThemeStore.setState({ theme: 'light' });
    document.documentElement.classList.remove('dark');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Persistence across sessions', () => {
    it('persists dark mode preference and restores it', () => {
      const { result } = renderHook(() => useThemeStore());

      // User sets dark mode
      act(() => {
        result.current.setTheme('dark');
      });

      expect(localStorage.getItem('subcults-theme')).toBe('dark');
      expect(document.documentElement.classList.contains('dark')).toBe(true);

      // Simulate page reload by reinitializing
      act(() => {
        result.current.initializeTheme();
      });

      expect(result.current.theme).toBe('dark');
      expect(document.documentElement.classList.contains('dark')).toBe(true);
    });

    it('persists light mode preference and restores it', () => {
      const { result } = renderHook(() => useThemeStore());

      // User explicitly sets light mode
      act(() => {
        result.current.setTheme('light');
      });

      expect(localStorage.getItem('subcults-theme')).toBe('light');

      // Simulate page reload
      act(() => {
        result.current.initializeTheme();
      });

      expect(result.current.theme).toBe('light');
      expect(document.documentElement.classList.contains('dark')).toBe(false);
    });
  });

  describe('System preference detection', () => {
    it('respects system dark mode when no user preference', () => {
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
      expect(document.documentElement.classList.contains('dark')).toBe(true);
      // Should NOT persist system preference
      expect(localStorage.getItem('subcults-theme')).toBeNull();
    });

    it('user preference overrides system preference', () => {
      // System prefers dark
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

      // But user has explicitly chosen light
      localStorage.setItem('subcults-theme', 'light');

      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.initializeTheme();
      });

      // User preference should win
      expect(result.current.theme).toBe('light');
      expect(document.documentElement.classList.contains('dark')).toBe(false);
    });
  });

  describe('No flicker on page load', () => {
    it('applies theme immediately without flicker', () => {
      localStorage.setItem('subcults-theme', 'dark');
      
      const { result } = renderHook(() => useThemeStore());

      // Initialize should happen synchronously
      act(() => {
        result.current.initializeTheme();
      });

      // Theme should be applied immediately
      expect(result.current.theme).toBe('dark');
      expect(document.documentElement.classList.contains('dark')).toBe(true);
    });
  });

  describe('Toggle behavior', () => {
    it('toggles between light and dark multiple times', () => {
      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.setTheme('light');
      });
      expect(result.current.theme).toBe('light');

      act(() => {
        result.current.toggleTheme();
      });
      expect(result.current.theme).toBe('dark');
      expect(localStorage.getItem('subcults-theme')).toBe('dark');

      act(() => {
        result.current.toggleTheme();
      });
      expect(result.current.theme).toBe('light');
      expect(localStorage.getItem('subcults-theme')).toBe('light');

      act(() => {
        result.current.toggleTheme();
      });
      expect(result.current.theme).toBe('dark');
      expect(localStorage.getItem('subcults-theme')).toBe('dark');
    });
  });

  describe('Document class management', () => {
    it('maintains single source of truth for dark class', () => {
      const { result } = renderHook(() => useThemeStore());

      // Start with light
      act(() => {
        result.current.setTheme('light');
      });
      expect(document.documentElement.classList.contains('dark')).toBe(false);

      // Switch to dark
      act(() => {
        result.current.setTheme('dark');
      });
      expect(document.documentElement.classList.contains('dark')).toBe(true);

      // Toggle back to light
      act(() => {
        result.current.toggleTheme();
      });
      expect(document.documentElement.classList.contains('dark')).toBe(false);
    });

    it('does not create duplicate dark classes', () => {
      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.setTheme('dark');
      });
      
      const classList = document.documentElement.classList;
      const darkClassCount = Array.from(classList).filter(c => c === 'dark').length;
      
      expect(darkClassCount).toBe(1);
    });
  });

  describe('Error handling', () => {
    it('handles corrupted localStorage gracefully', () => {
      localStorage.setItem('subcults-theme', 'invalid-theme');

      const { result } = renderHook(() => useThemeStore());

      act(() => {
        result.current.initializeTheme();
      });

      // Should fall back to system preference or default
      expect(['light', 'dark']).toContain(result.current.theme);
    });

    it('works when localStorage is unavailable', () => {
      const originalGetItem = localStorage.getItem;
      const originalSetItem = localStorage.setItem;
      
      // Mock localStorage failure
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
        throw new Error('localStorage unavailable');
      });
      vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
        throw new Error('localStorage unavailable');
      });

      const { result } = renderHook(() => useThemeStore());

      // Should not throw
      expect(() => {
        act(() => {
          result.current.setTheme('dark');
        });
      }).not.toThrow();

      // Restore
      localStorage.getItem = originalGetItem;
      localStorage.setItem = originalSetItem;
    });
  });
});
