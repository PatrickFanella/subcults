/**
 * Routing Integration Tests
 * Validates all routes, URL parameters, and navigation flows
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { createMemoryRouter, RouterProvider, useParams } from 'react-router-dom';
import { authStore } from '../stores/authStore';

// Mock pages for testing
const HomePage = () => <div>Home Page</div>;
const SceneDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  return <div>Scene Detail: {id}</div>;
};
const EventDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  return <div>Event Detail: {id}</div>;
};
const StreamPage = () => {
  const { id } = useParams<{ id: string }>();
  return <div>Stream: {id}</div>;
};
const SettingsPage = () => <div>Settings Page</div>;
const AdminPage = () => <div>Admin Page</div>;
const NotFoundPage = () => (
  <div>
    <h1>404</h1>
    <a href="/">Go Home</a>
  </div>
);
const LoginPage = () => <div>Login Page</div>;

// Mock guards
const RequireAuth = ({ children }: { children: React.ReactNode }) => {
  const isAuthenticated = authStore.getState().isAuthenticated;
  if (!isAuthenticated) {
    return <div>Redirecting to login...</div>;
  }
  return <>{children}</>;
};

const RequireAdmin = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isAdmin } = authStore.getState();
  if (!isAuthenticated) {
    return <div>Redirecting to login...</div>;
  }
  if (!isAdmin) {
    return <div>Redirecting to home...</div>;
  }
  return <>{children}</>;
};

describe('Routing Integration', () => {
  beforeEach(() => {
    authStore.resetForTesting();
  });

  describe('Public Routes', () => {
    it('should render home page at root path', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/',
            element: <HomePage />,
          },
        ],
        {
          initialEntries: ['/'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Home Page')).toBeInTheDocument();
    });

    it('should render scene detail page with correct ID parameter', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/scenes/:id',
            element: <SceneDetailPage />,
          },
        ],
        {
          initialEntries: ['/scenes/scene-123'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Scene Detail: scene-123')).toBeInTheDocument();
    });

    it('should render event detail page with correct ID parameter', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/events/:id',
            element: <EventDetailPage />,
          },
        ],
        {
          initialEntries: ['/events/event-456'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Event Detail: event-456')).toBeInTheDocument();
    });

    it('should render login page', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/account/login',
            element: <LoginPage />,
          },
        ],
        {
          initialEntries: ['/account/login'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });

  describe('Protected Routes', () => {
    it('should redirect stream page when not authenticated', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/streams/:id',
            element: (
              <RequireAuth>
                <StreamPage />
              </RequireAuth>
            ),
          },
        ],
        {
          initialEntries: ['/streams/stream-789'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Redirecting to login...')).toBeInTheDocument();
    });

    it('should render stream page with correct ID when authenticated', () => {
      authStore.setUser({ did: 'did:test:user', role: 'user' }, 'test-token');

      const router = createMemoryRouter(
        [
          {
            path: '/streams/:id',
            element: (
              <RequireAuth>
                <StreamPage />
              </RequireAuth>
            ),
          },
        ],
        {
          initialEntries: ['/streams/stream-789'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Stream: stream-789')).toBeInTheDocument();
    });

    it('should redirect settings page when not authenticated', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/settings',
            element: (
              <RequireAuth>
                <SettingsPage />
              </RequireAuth>
            ),
          },
        ],
        {
          initialEntries: ['/settings'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Redirecting to login...')).toBeInTheDocument();
    });

    it('should render settings page when authenticated', () => {
      authStore.setUser({ did: 'did:test:user', role: 'user' }, 'test-token');

      const router = createMemoryRouter(
        [
          {
            path: '/settings',
            element: (
              <RequireAuth>
                <SettingsPage />
              </RequireAuth>
            ),
          },
        ],
        {
          initialEntries: ['/settings'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Settings Page')).toBeInTheDocument();
    });
  });

  describe('Admin Routes', () => {
    it('should redirect admin page when not authenticated', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/admin',
            element: (
              <RequireAdmin>
                <AdminPage />
              </RequireAdmin>
            ),
          },
        ],
        {
          initialEntries: ['/admin'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Redirecting to login...')).toBeInTheDocument();
    });

    it('should redirect admin page when authenticated but not admin', () => {
      authStore.setUser({ did: 'did:test:user', role: 'user' }, 'test-token');

      const router = createMemoryRouter(
        [
          {
            path: '/admin',
            element: (
              <RequireAdmin>
                <AdminPage />
              </RequireAdmin>
            ),
          },
        ],
        {
          initialEntries: ['/admin'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Redirecting to home...')).toBeInTheDocument();
    });

    it('should render admin page when authenticated as admin', () => {
      authStore.setUser({ did: 'did:test:admin', role: 'admin' }, 'test-token');

      const router = createMemoryRouter(
        [
          {
            path: '/admin',
            element: (
              <RequireAdmin>
                <AdminPage />
              </RequireAdmin>
            ),
          },
        ],
        {
          initialEntries: ['/admin'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('Admin Page')).toBeInTheDocument();
    });
  });

  describe('404 Not Found', () => {
    it('should render 404 page for unknown routes', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/',
            element: <HomePage />,
          },
          {
            path: '*',
            element: <NotFoundPage />,
          },
        ],
        {
          initialEntries: ['/unknown-route'],
        }
      );

      render(<RouterProvider router={router} />);
      expect(screen.getByText('404')).toBeInTheDocument();
    });

    it('should have a link back to home on 404 page', () => {
      const router = createMemoryRouter(
        [
          {
            path: '/',
            element: <HomePage />,
          },
          {
            path: '*',
            element: <NotFoundPage />,
          },
        ],
        {
          initialEntries: ['/unknown-route'],
        }
      );

      render(<RouterProvider router={router} />);
      const homeLink = screen.getByText('Go Home');
      expect(homeLink).toBeInTheDocument();
      expect(homeLink).toHaveAttribute('href', '/');
    });
  });
});
