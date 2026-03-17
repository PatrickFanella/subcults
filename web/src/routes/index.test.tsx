/**
 * Route tests
 * Tests for routing behavior, guards, and navigation
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { RouterProvider, useLocation } from 'react-router-dom';
import { createMemoryRouter } from 'react-router-dom';
import { RequireAuth } from '../guards/RequireAuth';
import { RequireAdmin } from '../guards/RequireAdmin';
import { authStore } from '../stores/authStore';
import { ScenesPage } from '../pages/ScenesPage';
import { EventsPage } from '../pages/EventsPage';

// Mock pages for testing
const MockPublicPage = () => <div>Public Page</div>;
const MockProtectedPage = () => <div>Protected Page</div>;
const MockAdminPage = () => <div>Admin Page</div>;
const MockSearchPage = () => {
  const location = useLocation();
  return <div data-testid="search-page">Search Page{location.search}</div>;
};
interface LocationState {
  from?: { pathname: string };
}

const MockLoginPage = () => {
  const location = useLocation();
  const state = location.state as LocationState | null;
  const from = state?.from?.pathname;
  return (
    <div>
      Login Page
      {from && <span data-testid="return-url">{from}</span>}
    </div>
  );
};

describe('Route Guards', () => {
  beforeEach(() => {
    // Reset auth state before each test
    authStore.resetForTesting();
  });

  describe('RequireAuth', () => {
    it('redirects to login when not authenticated', async () => {
      const router = createMemoryRouter(
        [
          {
            path: '/account/login',
            element: <MockLoginPage />,
          },
          {
            path: '/protected',
            element: (
              <RequireAuth>
                <MockProtectedPage />
              </RequireAuth>
            ),
          },
        ],
        {
          initialEntries: ['/protected'],
          future: {
            v7_startTransition: true,
            v7_relativeSplatPath: true,
          },
        }
      );

      render(<RouterProvider router={router} />);

      await waitFor(() => {
        expect(screen.getByText('Login Page')).toBeInTheDocument();
      });
    });

    it('preserves return URL when redirecting to login', async () => {
      const router = createMemoryRouter(
        [
          {
            path: '/account/login',
            element: <MockLoginPage />,
          },
          {
            path: '/protected',
            element: (
              <RequireAuth>
                <MockProtectedPage />
              </RequireAuth>
            ),
          },
        ],
        {
          initialEntries: ['/protected'],
          future: {
            v7_startTransition: true,
            v7_relativeSplatPath: true,
          },
        }
      );

      render(<RouterProvider router={router} />);

      await waitFor(() => {
        expect(screen.getByText('Login Page')).toBeInTheDocument();
        expect(screen.getByTestId('return-url')).toHaveTextContent('/protected');
      });
    });

    it('renders protected content when authenticated', async () => {
      authStore.setUser({ did: 'did:test:user', role: 'user' }, 'test-token');

      const router = createMemoryRouter(
        [
          {
            path: '/account/login',
            element: <MockLoginPage />,
          },
          {
            path: '/protected',
            element: (
              <RequireAuth>
                <MockProtectedPage />
              </RequireAuth>
            ),
          },
        ],
        {
          initialEntries: ['/protected'],
          future: {
            v7_startTransition: true,
            v7_relativeSplatPath: true,
          },
        }
      );

      render(<RouterProvider router={router} />);

      await waitFor(() => {
        expect(screen.getByText('Protected Page')).toBeInTheDocument();
      });
    });
  });

  describe('RequireAdmin', () => {
    it('redirects to login when not authenticated', async () => {
      const router = createMemoryRouter(
        [
          {
            path: '/account/login',
            element: <MockLoginPage />,
          },
          {
            path: '/',
            element: <MockPublicPage />,
          },
          {
            path: '/admin',
            element: (
              <RequireAdmin>
                <MockAdminPage />
              </RequireAdmin>
            ),
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
        expect(screen.getByText('Login Page')).toBeInTheDocument();
      });
    });

    it('redirects to home when authenticated but not admin', async () => {
      authStore.setUser({ did: 'did:test:user', role: 'user' }, 'test-token');

      const router = createMemoryRouter(
        [
          {
            path: '/account/login',
            element: <MockLoginPage />,
          },
          {
            path: '/',
            element: <MockPublicPage />,
          },
          {
            path: '/admin',
            element: (
              <RequireAdmin>
                <MockAdminPage />
              </RequireAdmin>
            ),
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
        expect(screen.getByText('Public Page')).toBeInTheDocument();
      });
    });

    it('renders admin content when authenticated as admin', async () => {
      authStore.setUser({ did: 'did:test:admin', role: 'admin' }, 'test-token');

      const router = createMemoryRouter(
        [
          {
            path: '/account/login',
            element: <MockLoginPage />,
          },
          {
            path: '/',
            element: <MockPublicPage />,
          },
          {
            path: '/admin',
            element: (
              <RequireAdmin>
                <MockAdminPage />
              </RequireAdmin>
            ),
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
        expect(screen.getByText('Admin Page')).toBeInTheDocument();
      });
    });
  });
});

describe('Navigation Redirects', () => {
  it('redirects /scenes to /search?type=scenes', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/scenes',
          element: <ScenesPage />,
        },
        {
          path: '/search',
          element: <MockSearchPage />,
        },
      ],
      {
        initialEntries: ['/scenes'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByTestId('search-page')).toBeInTheDocument();
      expect(screen.getByTestId('search-page').textContent).toContain('?type=scenes');
    });
  });

  it('redirects /events to /search?type=events', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/events',
          element: <EventsPage />,
        },
        {
          path: '/search',
          element: <MockSearchPage />,
        },
      ],
      {
        initialEntries: ['/events'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByTestId('search-page')).toBeInTheDocument();
      expect(screen.getByTestId('search-page').textContent).toContain('?type=events');
    });
  });
});
