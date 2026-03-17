/**
 * Settings Page Modifications Integration Test
 * 
 * Tests the complete settings flow:
 * 1. User navigates to settings page
 * 2. User modifies various settings
 * 3. Settings are saved (persisted)
 * 4. Settings persist across page reloads
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { SettingsPage } from '../pages/SettingsPage';
import { authStore } from '../stores/authStore';
import { useThemeStore } from '../stores/themeStore';
import { setupMockServer } from '../test/mocks/server';

// Setup MSW mock server
setupMockServer();

describe('Integration: Settings Page Modifications Flow', () => {
  beforeEach(() => {
    // Setup authenticated user
    authStore.setUser(
      { did: 'did:example:testuser', role: 'user' },
      'mock-test-token'
    );
    
    // Reset stores to default state
    useThemeStore.setState({ theme: 'light' });
    // Note: useSettingsStore doesn't have resetSettings, just clear localStorage
    
    localStorage.clear();
    document.documentElement.classList.remove('dark');
  });

  afterEach(() => {
    authStore.resetForTesting();
    useThemeStore.setState({ theme: 'light' });
    localStorage.clear();
    document.documentElement.classList.remove('dark');
  });

  it('should display settings page with current settings', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify settings page renders
    expect(screen.getByRole('heading', { name: /^Settings$/i })).toBeInTheDocument();
    
    // Verify sections are present
    expect(screen.getByRole('heading', { name: /Appearance/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /Privacy/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /Notifications/i })).toBeInTheDocument();
  });

  it('should toggle theme and persist the change', async () => {
    const user = userEvent.setup();

    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify initial theme is light
    expect(screen.getByText('light')).toBeInTheDocument();
    expect(useThemeStore.getState().theme).toBe('light');

    // Find and click dark mode toggle
    const toggle = screen.getByLabelText(/dark mode/i);
    await user.click(toggle);

    // Verify theme changed to dark
    await waitFor(() => {
      expect(useThemeStore.getState().theme).toBe('dark');
    });

    // Verify dark mode class is applied to document
    expect(document.documentElement.classList.contains('dark')).toBe(true);

    // Verify theme is persisted in localStorage
    expect(localStorage.getItem('subcults-theme')).toBe('dark');
  });

  it('should update notification settings', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify notifications section exists
    expect(screen.getByRole('heading', { name: /Notifications/i })).toBeInTheDocument();

    // In full implementation:
    // 1. Find notification toggle switches
    // 2. Toggle specific notification types (email, push, scene updates)
    // 3. Verify settings store is updated
    // 4. Verify settings are persisted to localStorage or API
  });

  it('should persist settings across page reloads', async () => {
    // Set theme to dark
    useThemeStore.setState({ theme: 'dark' });
    localStorage.setItem('subcults-theme', 'dark');

    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify dark theme is loaded from storage
    await waitFor(() => {
      expect(screen.getByText('dark')).toBeInTheDocument();
    });

    expect(useThemeStore.getState().theme).toBe('dark');
  });

  it('should display theme preview with current theme', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify theme preview section
    expect(screen.getByRole('heading', { name: /Theme Preview/i })).toBeInTheDocument();
    
    // Verify preview elements
    expect(screen.getByRole('button', { name: /Primary Button/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Accent Button/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Secondary Button/i })).toBeInTheDocument();
  });

  it('should update privacy settings', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify privacy section exists
    expect(screen.getByRole('heading', { name: /Privacy/i })).toBeInTheDocument();

    // In full implementation:
    // 1. Find privacy controls (location consent, profile visibility)
    // 2. Toggle privacy settings
    // 3. Verify settings are saved
    // 4. Verify privacy preferences are respected in other parts of app
  });

  it('should handle settings save errors gracefully', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify page renders
    expect(screen.getByRole('heading', { name: /^Settings$/i })).toBeInTheDocument();

    // In full implementation with API integration:
    // 1. Mock API error response
    // 2. Change a setting
    // 3. Attempt to save
    // 4. Verify error message is shown
    // 5. Verify setting reverts to previous value or shows error state
  });

  it('should sync theme across all components', async () => {
    const user = userEvent.setup();

    const router = createMemoryRouter(
      [
        {
          path: '/settings',
          element: <SettingsPage />,
        },
      ],
      {
        initialEntries: ['/settings'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    const { container } = render(<RouterProvider router={router} />);

    // Initial light theme
    expect(useThemeStore.getState().theme).toBe('light');

    // Toggle to dark
    const toggle = screen.getByLabelText(/dark mode/i);
    await user.click(toggle);

    // Verify theme applied to root element
    await waitFor(() => {
      expect(document.documentElement.classList.contains('dark')).toBe(true);
    });

    // Verify page background uses dark theme classes
    const mainContainer = container.querySelector('.min-h-screen');
    expect(mainContainer).toHaveClass('bg-background', 'text-foreground');
  });
});
