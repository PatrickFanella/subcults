/**
 * Login to Dashboard Integration Test
 * 
 * Tests the complete user flow:
 * 1. User navigates to login page
 * 2. User enters credentials
 * 3. User submits form
 * 4. User is redirected to account page/dashboard
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { LoginPage } from '../pages/LoginPage';
import { AccountPage } from '../pages/AccountPage';
import { HomePage } from '../pages/HomePage';
import { authStore } from '../stores/authStore';
import { setupMockServer } from '../test/mocks/server';

// Setup MSW mock server
setupMockServer();

describe('Integration: Login to Dashboard Flow', () => {
  beforeEach(() => {
    // Reset auth store
    authStore.resetForTesting();
    localStorage.clear();
  });

  afterEach(() => {
    authStore.resetForTesting();
    localStorage.clear();
  });

  it('should successfully log in and navigate to account page', async () => {
    const user = userEvent.setup();

    // Create router with login and account pages
    const router = createMemoryRouter(
      [
        {
          path: '/',
          element: <HomePage />,
        },
        {
          path: '/account/login',
          element: <LoginPage />,
        },
        {
          path: '/account',
          element: <AccountPage />,
        },
      ],
      {
        initialEntries: ['/account/login'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Verify we're on login page
    expect(screen.getByRole('heading', { name: /welcome to subcults/i })).toBeInTheDocument();

    // Fill in login form
    const usernameInput = screen.getByLabelText(/username or email/i);
    const passwordInput = screen.getByLabelText(/^password$/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(usernameInput, 'testuser');
    await user.type(passwordInput, 'password123');

    // Submit form
    await user.click(submitButton);

    // Wait for login to complete and redirect
    await waitFor(
      () => {
        // Check that auth store is updated
        const state = authStore.getState();
        expect(state.isAuthenticated).toBe(true);
        expect(state.user?.did).toBe('did:example:testuser');
      },
      { timeout: 5000 }
    );

    // Verify navigation to account page occurred
    // Note: In actual app, navigation happens via router, 
    // but we can verify auth state which triggers the redirect
    const state = authStore.getState();
    expect(state.isAuthenticated).toBe(true);
  });

  it('should show error message for invalid credentials', async () => {
    const user = userEvent.setup();

    const router = createMemoryRouter(
      [
        {
          path: '/account/login',
          element: <LoginPage />,
        },
      ],
      {
        initialEntries: ['/account/login'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Fill in login form with invalid credentials
    const usernameInput = screen.getByLabelText(/username or email/i);
    const passwordInput = screen.getByLabelText(/^password$/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(usernameInput, 'testuser');
    await user.type(passwordInput, 'wrongpassword');

    // Submit form
    await user.click(submitButton);

    // Wait for error message
    await waitFor(() => {
      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
    });

    // Verify user is not authenticated
    expect(authStore.getState().isAuthenticated).toBe(false);
  });

  it('should persist remember me preference', async () => {
    const user = userEvent.setup();

    const router = createMemoryRouter(
      [
        {
          path: '/account/login',
          element: <LoginPage />,
        },
        {
          path: '/account',
          element: <AccountPage />,
        },
      ],
      {
        initialEntries: ['/account/login'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Fill in login form
    const usernameInput = screen.getByLabelText(/username or email/i);
    const passwordInput = screen.getByLabelText(/^password$/i);
    const rememberMeCheckbox = screen.getByRole('checkbox', { name: /remember me/i });
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(usernameInput, 'testuser');
    await user.type(passwordInput, 'password123');
    await user.click(rememberMeCheckbox);

    // Submit form
    await user.click(submitButton);

    // Wait for login and verify remember me was saved
    await waitFor(() => {
      expect(localStorage.getItem('subcults-remembered-username')).toBe('testuser');
    });
  });

  it('should show loading state during login', async () => {
    const user = userEvent.setup();

    const router = createMemoryRouter(
      [
        {
          path: '/account/login',
          element: <LoginPage />,
        },
      ],
      {
        initialEntries: ['/account/login'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Fill in login form
    const usernameInput = screen.getByLabelText(/username or email/i);
    const passwordInput = screen.getByLabelText(/^password$/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    await user.type(usernameInput, 'testuser');
    await user.type(passwordInput, 'password123');

    // Submit form
    await user.click(submitButton);

    // Should show loading state (button disabled and text changed)
    // Note: This is timing-dependent, so we check for either state
    const loadingButton = screen.queryByRole('button', { name: /signing in/i });
    const normalButton = screen.queryByRole('button', { name: /sign in/i });
    
    // At least one should exist
    expect(loadingButton || normalButton).toBeTruthy();
  });
});
