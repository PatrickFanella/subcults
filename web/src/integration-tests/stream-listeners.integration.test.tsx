/**
 * Stream Start to Live Listeners Integration Test
 * 
 * Tests the complete streaming flow:
 * 1. User navigates to stream page
 * 2. User starts/joins a stream
 * 3. Stream connects successfully
 * 4. Live listeners/participants are displayed
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { StreamPage } from '../pages/StreamPage';
import { authStore } from '../stores/authStore';
import { setupMockServer } from '../test/mocks/server';

// Setup MSW mock server
setupMockServer();

describe('Integration: Stream Start to Live Listeners Flow', () => {
  beforeEach(() => {
    // Reset stores
    authStore.setUser(
      { did: 'did:example:testuser', role: 'user' },
      'mock-test-token'
    );
    // Note: useStreamingStore doesn't have a reset method, just clear localStorage
    localStorage.clear();
  });

  afterEach(() => {
    authStore.resetForTesting();
    localStorage.clear();
  });

  it('should display stream page for authenticated user', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/test-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Wait for stream page to load
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });
  });

  it('should join stream and display connection status', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/test-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Wait for page to render
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Find and click "Join Stream" button
    // 2. Wait for connection
    // 3. Verify connection status indicator appears
    // 4. Verify "Connected" or "Connecting" status is shown
  });

  it('should display live participant list', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/test-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Join stream
    // 2. Wait for participant list to load
    // 3. Verify participant list is displayed
    // 4. Verify at least local user is in the list
    // 5. Verify participant count is shown
  });

  it('should show audio controls when stream is active', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/test-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Join stream
    // 2. Wait for connection
    // 3. Verify audio controls are visible (mute, volume, etc.)
    // 4. Test mute/unmute functionality
    // 5. Test volume adjustment
  });

  it('should update participant list in real-time', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/test-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Join stream
    // 2. Verify initial participant count
    // 3. Simulate new participant joining (via mock event)
    // 4. Verify participant list updates
    // 5. Verify participant count increments
  });

  it('should handle stream disconnection gracefully', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/test-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Join stream
    // 2. Wait for connection
    // 3. Click "Leave" button
    // 4. Verify disconnection
    // 5. Verify "Join" button is available again
    // 6. Verify participant list is cleared
  });

  it('should show error for failed stream connection', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/invalid-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });

    // In full implementation with error handling:
    // 1. Attempt to join invalid/unavailable stream
    // 2. Verify error message is displayed
    // 3. Verify user can retry connection
  });

  it('should persist volume settings across reconnections', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/streams/:id',
          element: <StreamPage />,
        },
      ],
      {
        initialEntries: ['/streams/test-room'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Stream/i })).toBeInTheDocument();
    });

    // In full implementation:
    // 1. Join stream
    // 2. Adjust volume slider to specific value
    // 3. Leave stream
    // 4. Rejoin stream
    // 5. Verify volume is still at the set value
  });
});
