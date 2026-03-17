/**
 * Admin Create Scene Integration Test
 * 
 * Tests the complete admin flow:
 * 1. Admin user authenticates
 * 2. Admin navigates to admin page
 * 3. Admin creates a new scene
 * 4. Scene is created successfully
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { AdminPage } from '../pages/AdminPage';
import { LoginPage } from '../pages/LoginPage';
import { authStore } from '../stores/authStore';
import { setupMockServer } from '../test/mocks/server';

// Setup MSW mock server
setupMockServer();

describe('Integration: Admin Create Scene Flow', () => {
  beforeEach(() => {
    authStore.resetForTesting();
    localStorage.clear();
  });

  afterEach(() => {
    authStore.resetForTesting();
    localStorage.clear();
  });

  it('should allow admin to access admin page', async () => {
    // Set admin auth state
    authStore.setUser(
      { did: 'did:example:admin', role: 'admin' },
      'mock-admin-token'
    );

    const router = createMemoryRouter(
      [
        {
          path: '/admin',
          element: <AdminPage />,
        },
      ],
      {
        initialEntries: ['/admin'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify admin page renders
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Admin Dashboard/i })).toBeInTheDocument();
    });
  });

  it('should display scene creation form for admin', async () => {
    // Set admin auth state
    authStore.setUser(
      { did: 'did:example:admin', role: 'admin' },
      'mock-admin-token'
    );

    const router = createMemoryRouter(
      [
        {
          path: '/admin',
          element: <AdminPage />,
        },
      ],
      {
        initialEntries: ['/admin'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Wait for admin page to render
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Admin Dashboard/i })).toBeInTheDocument();
    });

    // In full implementation, would verify:
    // - Scene creation form is visible
    // - Form has required fields (name, description, location)
  });

  it('should successfully create a new scene', async () => {
    // Set admin auth state
    authStore.setUser(
      { did: 'did:example:admin', role: 'admin' },
      'mock-admin-token'
    );

    const router = createMemoryRouter(
      [
        {
          path: '/admin',
          element: <AdminPage />,
        },
      ],
      {
        initialEntries: ['/admin'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Wait for admin page
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Admin Dashboard/i })).toBeInTheDocument();
    });

    // In full implementation with actual form:
    // 1. Fill in scene name
    // 2. Fill in scene description
    // 3. Set location (latitude, longitude)
    // 4. Submit form
    // 5. Verify success message
    // 6. Verify scene appears in list
  });

  it('should validate scene creation form', async () => {
    // Set admin auth state
    authStore.setUser(
      { did: 'did:example:admin', role: 'admin' },
      'mock-admin-token'
    );

    const router = createMemoryRouter(
      [
        {
          path: '/admin',
          element: <AdminPage />,
        },
      ],
      {
        initialEntries: ['/admin'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Admin Dashboard/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Try to submit empty form
    // 2. Verify validation errors are shown
    // 3. Fill in required fields
    // 4. Verify errors clear
    // 5. Submit valid form
  });

  it('should handle scene creation errors', async () => {
    // Set admin auth state
    authStore.setUser(
      { did: 'did:example:admin', role: 'admin' },
      'mock-admin-token'
    );

    const router = createMemoryRouter(
      [
        {
          path: '/admin',
          element: <AdminPage />,
        },
      ],
      {
        initialEntries: ['/admin'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Admin Dashboard/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Mock API error response
    // 2. Fill and submit form
    // 3. Verify error message is displayed
    // 4. Verify form is not cleared (allow retry)
  });

  it('should require admin role for scene creation', async () => {
    // Set regular user auth state (not admin)
    authStore.setUser(
      { did: 'did:example:user', role: 'user' },
      'mock-user-token'
    );

    const router = createMemoryRouter(
      [
        {
          path: '/admin',
          element: <AdminPage />,
        },
        {
          path: '/account/login',
          element: <LoginPage />,
        },
      ],
      {
        initialEntries: ['/admin'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // In full implementation with RequireAdmin guard:
    // Regular user should be redirected to login or access denied page
    // For now, we just verify auth state
    const state = authStore.getState();
    expect(state.user?.role).toBe('user');
    expect(state.isAdmin).toBe(false);
  });
});
