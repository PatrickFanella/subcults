/**
 * ThemeProvider Tests
 * Validates theme initialization and system preference sync
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render } from '@testing-library/react';
import { ThemeProvider } from './ThemeProvider';
import { useThemeStore } from '../stores/themeStore';

describe('ThemeProvider', () => {
  let mockMediaQuery: {
    matches: boolean;
    media: string;
    addEventListener: ReturnType<typeof vi.fn>;
    removeEventListener: ReturnType<typeof vi.fn>;
    addListener: ReturnType<typeof vi.fn>;
    removeListener: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    localStorage.clear();
    useThemeStore.setState({ theme: 'light' });
    document.documentElement.classList.remove('dark');

    // Setup matchMedia mock
    mockMediaQuery = {
      matches: false,
      media: '(prefers-color-scheme: dark)',
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
    };

    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockReturnValue(mockMediaQuery),
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders children', () => {
    const { getByText } = render(
      <ThemeProvider>
        <div>Test Content</div>
      </ThemeProvider>
    );

    expect(getByText('Test Content')).toBeInTheDocument();
  });

  it('initializes theme on mount', () => {
    localStorage.setItem('subcults-theme', 'dark');

    render(
      <ThemeProvider>
        <div>Test</div>
      </ThemeProvider>
    );

    expect(useThemeStore.getState().theme).toBe('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('listens for system theme changes (modern browsers)', () => {
    render(
      <ThemeProvider>
        <div>Test</div>
      </ThemeProvider>
    );

    expect(mockMediaQuery.addEventListener).toHaveBeenCalledWith(
      'change',
      expect.any(Function)
    );
  });

  it('cleans up event listener on unmount (modern browsers)', () => {
    const { unmount } = render(
      <ThemeProvider>
        <div>Test</div>
      </ThemeProvider>
    );

    unmount();

    expect(mockMediaQuery.removeEventListener).toHaveBeenCalledWith(
      'change',
      expect.any(Function)
    );
  });

  it('handles legacy browser API', () => {
    // Remove modern addEventListener
    const mockMediaQueryLegacy = {
      ...mockMediaQuery,
      addEventListener: undefined,
    };
    
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockReturnValue(mockMediaQueryLegacy),
    });

    render(
      <ThemeProvider>
        <div>Test</div>
      </ThemeProvider>
    );

    expect(mockMediaQueryLegacy.addListener).toHaveBeenCalledWith(expect.any(Function));
  });

  it('cleans up legacy listener on unmount', () => {
    const mockMediaQueryLegacy = {
      ...mockMediaQuery,
      addEventListener: undefined,
    };
    
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockReturnValue(mockMediaQueryLegacy),
    });
    
    const { unmount } = render(
      <ThemeProvider>
        <div>Test</div>
      </ThemeProvider>
    );

    unmount();

    expect(mockMediaQueryLegacy.removeListener).toHaveBeenCalledWith(expect.any(Function));
  });

  it('auto-switches theme based on system preference when no manual preference', () => {
    // Start with no stored preference and mock system to prefer light
    localStorage.clear();
    mockMediaQuery.matches = false;
    
    render(
      <ThemeProvider>
        <div>Test</div>
      </ThemeProvider>
    );

    // Clear the localStorage that was set during initialization
    // to simulate the scenario where user hasn't manually chosen
    localStorage.removeItem('subcults-theme');

    // Now simulate system theme change event
    const changeHandler = mockMediaQuery.addEventListener.mock.calls[0][1] as (
      e: MediaQueryListEvent
    ) => void;
    
    changeHandler({ matches: true } as MediaQueryListEvent);

    expect(useThemeStore.getState().theme).toBe('dark');
  });

  it('does not auto-switch when user has manual preference', () => {
    localStorage.setItem('subcults-theme', 'light');

    render(
      <ThemeProvider>
        <div>Test</div>
      </ThemeProvider>
    );

    // Simulate system theme change event
    const changeHandler = mockMediaQuery.addEventListener.mock.calls[0][1] as (
      e: MediaQueryListEvent
    ) => void;
    
    changeHandler({ matches: true } as MediaQueryListEvent);

    // Should stay light because user manually chose it
    expect(useThemeStore.getState().theme).toBe('light');
  });
});
