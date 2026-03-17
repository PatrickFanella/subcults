/**
 * AppLayout tests
 * Tests for layout structure and accessibility features
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { AppLayout } from './AppLayout';
import { authStore } from '../stores/authStore';

describe('AppLayout', () => {
  beforeEach(() => {
    authStore.resetForTesting();
  });

  it('renders header with logo', () => {
    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    expect(screen.getByText('Subcults')).toBeInTheDocument();
  });

  it('renders skip-to-content link', () => {
    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    const skipLink = screen.getByText('Skip to content');
    expect(skipLink).toBeInTheDocument();
    expect(skipLink.getAttribute('href')).toBe('#main-content');
  });

  it('renders main content area with proper landmarks', () => {
    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    const { container } = render(<RouterProvider router={router} />);

    // Check for semantic HTML elements
    expect(container.querySelector('header[role="banner"]')).toBeInTheDocument();
    expect(container.querySelector('main[role="main"]')).toBeInTheDocument();
    // Sidebar nav is present
    expect(container.querySelector('aside[aria-label="Sidebar navigation"]')).toBeInTheDocument();
  });

  it('shows login button when not authenticated', () => {
    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    expect(screen.getByText('Login')).toBeInTheDocument();
  });

  it('shows user info and logout when authenticated', async () => {
    authStore.setUser({ did: 'did:example:test-user-12345', role: 'user' }, 'test-token');

    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    const { container } = render(<RouterProvider router={router} />);
    const user = userEvent.setup();

    // ProfileDropdown shows user avatar (find it in the header with initials)
    const avatar = container.querySelector('[aria-hidden="true"].bg-brand-primary.text-white');
    expect(avatar).toBeInTheDocument();
    expect(avatar?.textContent).toBe('EX'); // First 2 chars after "did:" prefix
    
    // Find the profile button (parent of avatar)
    const profileButton = avatar?.parentElement as HTMLButtonElement;
    expect(profileButton).toBeTruthy();
    
    // Click to open dropdown
    await user.click(profileButton);
    
    // DID is shown in dropdown
    expect(screen.getByText(/did:example:test-user-12345/)).toBeInTheDocument();
    expect(screen.getByText('Sign out')).toBeInTheDocument();
  });

  it('shows admin link when user is admin', async () => {
    authStore.setUser({ did: 'did:example:admin', role: 'admin' }, 'test-token');

    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    const { container } = render(<RouterProvider router={router} />);
    const user = userEvent.setup();

    // Admin link appears in sidebar (which is visible on desktop, hidden on mobile)
    // On desktop: sidebar shows admin link
    // On mobile: need to open sidebar first
    const adminLinks = screen.getAllByText('Admin');
    expect(adminLinks.length).toBeGreaterThan(0);
    
    // Also check profile dropdown has admin link
    const avatar = container.querySelector('[aria-hidden="true"].bg-brand-primary.text-white');
    const profileButton = avatar?.parentElement as HTMLButtonElement;
    await user.click(profileButton);
    expect(screen.getByText('Admin Panel')).toBeInTheDocument();
  });

  it('does not show admin link for regular users', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');

    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    const { container } = render(<RouterProvider router={router} />);
    const user = userEvent.setup();

    // Admin should not appear in sidebar
    expect(screen.queryByText('Admin')).not.toBeInTheDocument();
    
    // Also check profile dropdown doesn't have admin link
    const avatar = container.querySelector('[aria-hidden="true"].bg-brand-primary.text-white');
    const profileButton = avatar?.parentElement as HTMLButtonElement;
    await user.click(profileButton);
    expect(screen.queryByText('Admin Panel')).not.toBeInTheDocument();
  });

  it('renders navigation with proper aria labels', () => {
    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <AppLayout />,
        },
      ],
      {
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    const { container } = render(<RouterProvider router={router} />);

    // Sidebar has proper aria-label
    expect(container.querySelector('aside[aria-label="Sidebar navigation"]')).toBeInTheDocument();
  });
});
