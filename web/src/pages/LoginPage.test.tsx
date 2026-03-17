/**
 * LoginPage Tests
 * Validates login form rendering, user interactions, and authentication flow
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { LoginPage } from './LoginPage';
import { authStore } from '../stores/authStore';
import * as authService from '../lib/auth-service';

// Mock auth service
vi.mock('../lib/auth-service', () => ({
  login: vi.fn(),
}));

const renderLoginPage = (initialPath = '/account/login') => {
  const router = createMemoryRouter(
    [
      {
        path: '/',
        element: <div>Home Page</div>,
      },
      {
        path: '/account/login',
        element: <LoginPage />,
      },
      {
        path: '/account',
        element: <div>Account Page</div>,
      },
    ],
    {
      initialEntries: [initialPath],
      future: {
        v7_startTransition: true,
        v7_relativeSplatPath: true,
      },
    }
  );

  return render(<RouterProvider router={router} />);
};

describe('LoginPage', () => {
  beforeEach(() => {
    authStore.resetForTesting();
    vi.clearAllMocks();
    // Clear localStorage
    localStorage.clear();
  });

  afterEach(() => {
    authStore.resetForTesting();
    localStorage.clear();
  });

  describe('Rendering', () => {
    it('renders login form with all required fields', () => {
      renderLoginPage();

      expect(screen.getByRole('heading', { name: /welcome to subcults/i })).toBeInTheDocument();
      expect(screen.getByLabelText(/username or email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument();
      expect(screen.getByRole('checkbox', { name: /remember me/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
    });

    it('renders forgot password link', () => {
      renderLoginPage();

      const forgotPasswordLink = screen.getByRole('link', { name: /forgot password/i });
      expect(forgotPasswordLink).toBeInTheDocument();
      expect(forgotPasswordLink).toHaveAttribute('href', '#');
      expect(forgotPasswordLink).toHaveAttribute('title', 'Coming soon');
    });

    it('renders sign up link', () => {
      renderLoginPage();

      const signUpLink = screen.getByRole('link', { name: /sign up/i });
      expect(signUpLink).toBeInTheDocument();
      expect(signUpLink).toHaveAttribute('href', '#');
      expect(signUpLink).toHaveAttribute('title', 'Coming soon');
    });

    it('renders password visibility toggle button', () => {
      renderLoginPage();

      const toggleButton = screen.getByLabelText(/show password/i);
      expect(toggleButton).toBeInTheDocument();
    });
  });

  describe('Form Interactions', () => {
    it('allows user to type in username field', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      await user.type(usernameInput, 'testuser');

      expect(usernameInput).toHaveValue('testuser');
    });

    it('allows user to type in password field', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const passwordInput = screen.getByLabelText(/^password$/i);
      await user.type(passwordInput, 'password123');

      expect(passwordInput).toHaveValue('password123');
    });

    it('toggles password visibility', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const passwordInput = screen.getByLabelText(/^password$/i) as HTMLInputElement;
      const toggleButton = screen.getByLabelText(/show password/i);

      // Initially hidden
      expect(passwordInput.type).toBe('password');

      // Click to show
      await user.click(toggleButton);
      expect(passwordInput.type).toBe('text');
      expect(screen.getByLabelText(/hide password/i)).toBeInTheDocument();

      // Click to hide again
      await user.click(toggleButton);
      expect(passwordInput.type).toBe('password');
    });

    it('toggles remember me checkbox', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const checkbox = screen.getByRole('checkbox', { name: /remember me/i });

      expect(checkbox).not.toBeChecked();

      await user.click(checkbox);
      expect(checkbox).toBeChecked();

      await user.click(checkbox);
      expect(checkbox).not.toBeChecked();
    });

    it('disables submit button when form is empty', () => {
      renderLoginPage();

      const submitButton = screen.getByRole('button', { name: /sign in/i });
      expect(submitButton).toBeDisabled();
    });

    it('enables submit button when form is filled', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'password123');

      expect(submitButton).toBeEnabled();
    });
  });

  describe('Form Submission', () => {
    it('calls login service with correct credentials', async () => {
      const user = userEvent.setup();
      const mockLogin = vi.mocked(authService.login);
      mockLogin.mockResolvedValue({
        did: 'did:example:testuser',
        role: 'user',
      });

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      expect(mockLogin).toHaveBeenCalledWith({
        username: 'testuser',
        password: 'password123',
      });
    });

    it('shows loading state during login', async () => {
      const user = userEvent.setup();
      const mockLogin = vi.mocked(authService.login);
      
      // Create a promise that we can control
      let resolveLogin: ((value: { did: string; role: string }) => void) | undefined;
      const loginPromise = new Promise<{ did: string; role: string }>((resolve) => {
        resolveLogin = resolve;
      });
      mockLogin.mockReturnValue(loginPromise);

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      // Should show loading state
      expect(screen.getByRole('button', { name: /signing in/i })).toBeInTheDocument();
      expect(submitButton).toBeDisabled();

      // Resolve the login
      resolveLogin!({ did: 'did:example:testuser', role: 'user' });
      
      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /signing in/i })).not.toBeInTheDocument();
      });
    });

    it('displays error message on login failure', async () => {
      const user = userEvent.setup();
      const mockLogin = vi.mocked(authService.login);
      mockLogin.mockRejectedValue(new Error('Invalid credentials'));

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'wrongpassword');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
      });
    });

    it('displays generic error message for non-Error objects', async () => {
      const user = userEvent.setup();
      const mockLogin = vi.mocked(authService.login);
      mockLogin.mockRejectedValue('Some string error');

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'password');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/login failed/i)).toBeInTheDocument();
      });
    });
  });

  describe('Remember Me Functionality', () => {
    it('saves username to localStorage when remember me is checked', async () => {
      const user = userEvent.setup();
      const mockLogin = vi.mocked(authService.login);
      mockLogin.mockResolvedValue({
        did: 'did:example:testuser',
        role: 'user',
      });

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const rememberMeCheckbox = screen.getByRole('checkbox', { name: /remember me/i });
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'password123');
      await user.click(rememberMeCheckbox);
      await user.click(submitButton);

      await waitFor(() => {
        expect(localStorage.getItem('subcults-remembered-username')).toBe('testuser');
      });
    });

    it('does not save username when remember me is unchecked', async () => {
      const user = userEvent.setup();
      const mockLogin = vi.mocked(authService.login);
      mockLogin.mockResolvedValue({
        did: 'did:example:testuser',
        role: 'user',
      });

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalled();
      });

      expect(localStorage.getItem('subcults-remembered-username')).toBeNull();
    });

    it('loads saved username on mount', () => {
      localStorage.setItem('subcults-remembered-username', 'saveduser');

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i) as HTMLInputElement;
      const rememberMeCheckbox = screen.getByRole('checkbox', { name: /remember me/i });

      expect(usernameInput.value).toBe('saveduser');
      expect(rememberMeCheckbox).toBeChecked();
    });
  });

  describe('Accessibility', () => {
    it('has proper form labels', () => {
      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);

      expect(usernameInput).toHaveAttribute('id', 'username');
      expect(passwordInput).toHaveAttribute('id', 'password');
    });

    it('marks required fields', () => {
      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);

      expect(usernameInput).toBeRequired();
      expect(passwordInput).toBeRequired();
    });

    it('uses proper autocomplete attributes', () => {
      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);

      expect(usernameInput).toHaveAttribute('autocomplete', 'username');
      expect(passwordInput).toHaveAttribute('autocomplete', 'current-password');
    });

    it('displays error with role="alert"', async () => {
      const user = userEvent.setup();
      const mockLogin = vi.mocked(authService.login);
      mockLogin.mockRejectedValue(new Error('Test error'));

      renderLoginPage();

      const usernameInput = screen.getByLabelText(/username or email/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(usernameInput, 'testuser');
      await user.type(passwordInput, 'password');
      await user.click(submitButton);

      await waitFor(() => {
        const alert = screen.getByRole('alert');
        expect(alert).toBeInTheDocument();
      });
    });
  });
});
