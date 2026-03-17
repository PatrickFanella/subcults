/**
 * ThemeProvider Component
 * Initializes theme on mount and syncs with system preference changes
 */

import { useEffect } from 'react';
import { useThemeActions } from '../stores/themeStore';

interface ThemeProviderProps {
  children: React.ReactNode;
}

/**
 * ThemeProvider wraps the app and manages theme initialization
 */
export function ThemeProvider({ children }: ThemeProviderProps) {
  const { initializeTheme, setTheme } = useThemeActions();

  useEffect(() => {
    // Initialize theme on mount
    initializeTheme();

    // Listen for system theme changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    
    const handleChange = (e: MediaQueryListEvent) => {
      // Only auto-switch if user hasn't manually set preference
      const storedTheme = localStorage.getItem('subcults-theme');
      if (!storedTheme) {
        setTheme(e.matches ? 'dark' : 'light');
      }
    };

    // Modern browsers
    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', handleChange);
      return () => mediaQuery.removeEventListener('change', handleChange);
    }
    // Legacy browsers
    else if (mediaQuery.addListener) {
      mediaQuery.addListener(handleChange);
      return () => mediaQuery.removeListener(handleChange);
    }
  }, [initializeTheme, setTheme]);

  return <>{children}</>;
}
