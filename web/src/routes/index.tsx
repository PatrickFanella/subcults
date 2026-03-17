/**
 * Route Configuration
 * Defines all application routes with lazy loading and guards
 */

import React, { Suspense, lazy } from 'react';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';
import { AppLayout } from '../layouts/AppLayout';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { ErrorBoundary } from '../components/ErrorBoundary';
import { RequireAuth } from '../guards/RequireAuth';
import { RequireAdmin } from '../guards/RequireAdmin';

// Lazy loaded pages
const HomePage = lazy(() =>
  import('../pages/HomePage').then((module) => ({ default: module.HomePage }))
);
const SceneDetailPage = lazy(() =>
  import('../pages/SceneDetailPage').then((module) => ({ default: module.SceneDetailPage }))
);
const SceneSettingsPage = lazy(() =>
  import('../pages/SceneSettingsPage').then((module) => ({ default: module.SceneSettingsPage }))
);
const EventDetailPage = lazy(() =>
  import('../pages/EventDetailPage').then((module) => ({ default: module.EventDetailPage }))
);
const AccountPage = lazy(() =>
  import('../pages/AccountPage').then((module) => ({ default: module.AccountPage }))
);
const LoginPage = lazy(() =>
  import('../pages/LoginPage').then((module) => ({ default: module.LoginPage }))
);
const SettingsPage = lazy(() =>
  import('../pages/SettingsPage').then((module) => ({ default: module.SettingsPage }))
);
const NotFoundPage = lazy(() =>
  import('../pages/NotFoundPage').then((module) => ({ default: module.NotFoundPage }))
);
const StreamPage = lazy(() =>
  import('../pages/StreamPage').then((module) => ({ default: module.StreamPage }))
);
const AdminPage = lazy(() =>
  import('../pages/AdminPage').then((module) => ({ default: module.AdminPage }))
);
const SearchResultsPage = lazy(() =>
  import('../pages/SearchResultsPage').then((module) => ({ default: module.SearchResultsPage }))
);
const ScenesPage = lazy(() =>
  import('../pages/ScenesPage').then((module) => ({ default: module.ScenesPage }))
);
const EventsPage = lazy(() =>
  import('../pages/EventsPage').then((module) => ({ default: module.EventsPage }))
);
const StreamingDemo = lazy(() =>
  import('../StreamingDemo').then((module) => ({ default: module.StreamingDemo }))
);
const StreamUIComponentsDemo = lazy(() =>
  import('../StreamUIComponentsDemo').then((module) => ({ default: module.StreamUIComponentsDemo }))
);

/**
 * Router configuration
 * Routes are organized by access level and loading strategy
 */
const router = createBrowserRouter(
  [
    {
      path: '/',
      element: <AppLayout />,
      errorElement: (
        <ErrorBoundary>
          <Suspense fallback={<LoadingSkeleton />}>
            <NotFoundPage />
          </Suspense>
        </ErrorBoundary>
      ),
      children: [
        // Public routes
        {
          index: true,
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <HomePage />
            </Suspense>
          ),
        },
        {
          path: 'scenes',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <ScenesPage />
            </Suspense>
          ),
        },
        {
          path: 'scenes/:id',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <SceneDetailPage />
            </Suspense>
          ),
        },
        {
          path: 'scenes/:id/settings',
          element: (
            <RequireAuth>
              <Suspense fallback={<LoadingSkeleton />}>
                <SceneSettingsPage />
              </Suspense>
            </RequireAuth>
          ),
        },
        {
          path: 'events',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <EventsPage />
            </Suspense>
          ),
        },
        {
          path: 'events/:id',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <EventDetailPage />
            </Suspense>
          ),
        },
        {
          path: 'search',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <SearchResultsPage />
            </Suspense>
          ),
        },
        {
          path: 'account/login',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <LoginPage />
            </Suspense>
          ),
        },
        {
          path: 'demo/streaming',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <StreamingDemo />
            </Suspense>
          ),
        },
        {
          path: 'demo/stream-ui',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <StreamUIComponentsDemo />
            </Suspense>
          ),
        },

        // Protected routes (require authentication)
        {
          path: 'account',
          element: (
            <RequireAuth>
              <Suspense fallback={<LoadingSkeleton />}>
                <AccountPage />
              </Suspense>
            </RequireAuth>
          ),
        },
        {
          path: 'settings',
          element: (
            <RequireAuth>
              <Suspense fallback={<LoadingSkeleton />}>
                <SettingsPage />
              </Suspense>
            </RequireAuth>
          ),
        },
        {
          path: 'streams/:id',
          element: (
            <RequireAuth>
              <Suspense fallback={<LoadingSkeleton />}>
                <StreamPage />
              </Suspense>
            </RequireAuth>
          ),
        },

        // Admin routes (require admin role)
        {
          path: 'admin',
          element: (
            <RequireAdmin>
              <Suspense fallback={<LoadingSkeleton />}>
                <AdminPage />
              </Suspense>
            </RequireAdmin>
          ),
        },

        // 404 catch-all
        {
          path: '*',
          element: (
            <Suspense fallback={<LoadingSkeleton />}>
              <NotFoundPage />
            </Suspense>
          ),
        },
      ],
    },
  ],
  {
    future: {
      v7_relativeSplatPath: true,
    },
  }
);

/**
 * AppRouter Component
 * Provides routing context to the application
 */
export const AppRouter: React.FC = () => {
  return <RouterProvider router={router} />;
};
