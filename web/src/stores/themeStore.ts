/**
 * Theme Store
 * Manages dark mode state with localStorage persistence
 */

import { create } from 'zustand';

export type Theme = 'light' | 'dark';

interface ThemeState {
  theme: Theme;
}

interface ThemeActions {
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
  initializeTheme: () => void;
}

export type ThemeStore = ThemeState & ThemeActions;

/**
 * Local storage key for theme preference
 */
const THEME_STORAGE_KEY = 'subcults-theme';

/**
 * Get initial theme from localStorage or system preference
 * Returns theme and whether it was manually set by user
 */
function getInitialTheme(): { theme: Theme; isManuallySet: boolean } {
  // Check localStorage first
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    if (stored === 'light' || stored === 'dark') {
      return { theme: stored, isManuallySet: true };
    }
  } catch (error) {
    // localStorage unavailable or error - continue to system preference
    console.warn('[themeStore] Failed to read from localStorage:', error);
  }

  // Fall back to system preference
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return { theme: 'dark', isManuallySet: false };
  }

  return { theme: 'light', isManuallySet: false };
}

/**
 * Apply theme to document (DOM only, no persistence)
 */
function applyThemeToDOM(theme: Theme): void {
  const root = document.documentElement;
  
  if (theme === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
}

/**
 * Persist theme to localStorage
 */
function persistTheme(theme: Theme): void {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
  } catch (error) {
    // localStorage unavailable or quota exceeded - silently fail
    console.warn('[themeStore] Failed to persist theme to localStorage:', error);
  }
}

/**
 * Theme store with dark mode management
 */
export const useThemeStore = create<ThemeStore>((set, get) => ({
  theme: getInitialTheme().theme,

  setTheme: (theme: Theme) => {
    set({ theme });
    applyThemeToDOM(theme);
    persistTheme(theme); // Only persist when user explicitly sets theme
  },

  toggleTheme: () => {
    const currentTheme = get().theme;
    const newTheme: Theme = currentTheme === 'light' ? 'dark' : 'light';
    get().setTheme(newTheme);
  },

  initializeTheme: () => {
    const { theme, isManuallySet } = getInitialTheme();
    set({ theme });
    applyThemeToDOM(theme);
    // Only persist if it was already manually set
    if (isManuallySet) {
      persistTheme(theme);
    }
  },
}));

/**
 * Hook for theme value only (optimized for re-renders)
 */
export function useTheme(): Theme {
  return useThemeStore((state) => state.theme);
}

/**
 * Hook for theme actions only (stable reference)
 */
export function useThemeActions() {
  const setTheme = useThemeStore((state) => state.setTheme);
  const toggleTheme = useThemeStore((state) => state.toggleTheme);
  const initializeTheme = useThemeStore((state) => state.initializeTheme);
  
  return {
    setTheme,
    toggleTheme,
    initializeTheme,
  };
}
