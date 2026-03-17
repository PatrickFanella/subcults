/**
 * Auth Store
 * Authentication state management with token refresh and multi-tab synchronization
 */

import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '../lib/api-client';

export interface User {
  did: string;
  role: 'user' | 'admin';
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isLoading: boolean;
  accessToken: string | null;
}

// BroadcastChannel for multi-tab synchronization
const LOGOUT_CHANNEL = 'subcults_auth_logout';
let logoutChannel: BroadcastChannel | null = null;

// Initialize BroadcastChannel if supported
// NOTE: This module is intended for browser contexts only.
if (typeof BroadcastChannel !== 'undefined') {
  try {
    logoutChannel = new BroadcastChannel(LOGOUT_CHANNEL);
  } catch (e) {
    // Fallback: disable multi-tab sync if BroadcastChannel fails
    console.warn('[authStore] BroadcastChannel initialization failed, multi-tab sync disabled:', e);
    logoutChannel = null;
  }
}

// In-memory auth state (access token stored here, refresh token in httpOnly cookie)
let authState: AuthState = {
  user: null,
  isAuthenticated: false,
  isAdmin: false,
  isLoading: true, // Start as loading until we check for existing session
  accessToken: null,
};

const listeners = new Set<(state: AuthState) => void>();

// Exponential backoff configuration
const RETRY_CONFIG = {
  maxRetries: 3,
  initialDelay: 1000, // 1 second
  maxDelay: 10000, // 10 seconds
};

/**
 * Sleep utility for retry delays.
 * Assumes the timer will complete - safe because only used within try-catch blocks.
 */
const sleep = (ms: number): Promise<void> => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};

/**
 * Refresh access token using refresh token (stored in httpOnly cookie)
 * Implements exponential backoff for transient failures
 */
const refreshAccessToken = async (retryCount = 0): Promise<string | null> => {
  try {
    // Call refresh endpoint (refresh token sent automatically via httpOnly cookie)
    // In development, Vite's proxy forwards /api to localhost:8080
    const response = await fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include', // Include cookies
    });

    if (!response.ok) {
      // Don't retry on 401 (refresh token invalid/expired)
      if (response.status === 401) {
        return null;
      }

      // Log unexpected 4xx errors for debugging (except 401)
      if (response.status >= 400 && response.status < 500) {
        // Do not log PII or tokens
        console.warn(
          `[authStore] Token refresh failed with status ${response.status}: ${response.statusText}`
        );
      }

      // Retry on transient errors (5xx, network issues)
      if (response.status >= 500 && retryCount < RETRY_CONFIG.maxRetries) {
        const delay = Math.min(
          RETRY_CONFIG.initialDelay * Math.pow(2, retryCount),
          RETRY_CONFIG.maxDelay
        );
        await sleep(delay);
        return refreshAccessToken(retryCount + 1);
      }

      return null;
    }

    const data = await response.json();

    // Update access token in memory
    authState.accessToken = data.accessToken;
    authState.user = data.user;
    authState.isAuthenticated = true;
    authState.isAdmin = data.user.role === 'admin';
    authState.isLoading = false;

    notifyListeners();

    return data.accessToken;
  } catch (error) {
    // Retry on network errors
    if (retryCount < RETRY_CONFIG.maxRetries) {
      const delay = Math.min(
        RETRY_CONFIG.initialDelay * Math.pow(2, retryCount),
        RETRY_CONFIG.maxDelay
      );
      await sleep(delay);
      return refreshAccessToken(retryCount + 1);
    }

    console.error('Token refresh failed:', error);
    return null;
  }
};

/**
 * Handle unauthorized state (refresh failed)
 */
const handleUnauthorized = (): void => {
  // Clear auth state
  authState = {
    user: null,
    isAuthenticated: false,
    isAdmin: false,
    isLoading: false,
    accessToken: null,
  };

  notifyListeners();

  // Broadcast logout to other tabs
  if (logoutChannel) {
    logoutChannel.postMessage({ type: 'logout' });
  }

  // Redirect to login if not already there
  // Note: Uses window.location.href for forced logout (full page reload)
  // to ensure complete state cleanup, though this breaks SPA navigation
  if (window.location.pathname !== '/account/login') {
    window.location.href = '/account/login';
  }
};

/**
 * Notify all listeners of state changes
 */
const notifyListeners = (): void => {
  listeners.forEach((listener) => listener(authState));
};

/**
 * Initialize API client and event listeners.
 * Separated to avoid circular dependency issues with module-level initialization.
 */
const initializeApiClient = (): void => {
  // In development, point to the API server running on port 8080
  // In production, use relative path /api which will be proxied or served as a subpath
  const baseURL = import.meta.env.DEV ? 'http://localhost:8080/api' : '/api';

  apiClient.initialize({
    baseURL,
    getAccessToken: () => authState.accessToken,
    refreshToken: refreshAccessToken,
    onUnauthorized: handleUnauthorized,
  });
};

// Initialize API client with auth callbacks
initializeApiClient();

// Listen for logout events from other tabs
if (logoutChannel) {
  logoutChannel.addEventListener('message', (event) => {
    if (event.data.type === 'logout') {
      // Only process logout if currently authenticated to avoid redundant state changes
      if (authState.isAuthenticated) {
        // Another tab logged out, sync this tab
        authState = {
          user: null,
          isAuthenticated: false,
          isAdmin: false,
          isLoading: false,
          accessToken: null,
        };
        notifyListeners();

        // Redirect to login if not already there
        // Note: Uses window.location.href for forced logout (full page reload)
        // to ensure complete state cleanup, though this breaks SPA navigation
        if (window.location.pathname !== '/account/login') {
          window.location.href = '/account/login';
        }
      }
    }
  });
}

export const authStore = {
  getState: (): AuthState => authState,

  subscribe: (listener: (state: AuthState) => void): (() => void) => {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },

  /**
   * Set user and access token (called after successful login).
   *
   * Note: This replaces the previous setUser(user | null) signature.
   * To clear auth state, use logout() instead of setUser(null).
   *
   * @param user - The authenticated user object
   * @param accessToken - The access token received from login
   */
  setUser: (user: User, accessToken: string): void => {
    authState = {
      user,
      isAuthenticated: true,
      isAdmin: user.role === 'admin',
      isLoading: false,
      accessToken,
    };
    notifyListeners();
  },

  /**
   * Logout user and clear tokens
   * Refresh token is cleared by calling backend logout endpoint
   */
  logout: async (): Promise<void> => {
    try {
      // Call logout endpoint to clear refresh token cookie
      await fetch('/api/auth/logout', {
        method: 'POST',
        credentials: 'include',
      });
    } catch (error) {
      console.error('Logout request failed:', error);
      // Continue with local logout even if request fails
    }

    // Clear local state
    authState = {
      user: null,
      isAuthenticated: false,
      isAdmin: false,
      isLoading: false,
      accessToken: null,
    };
    notifyListeners();

    // Broadcast logout to other tabs
    if (logoutChannel) {
      logoutChannel.postMessage({ type: 'logout' });
    }
  },

  /**
   * Initialize auth state by checking for existing session
   * Should be called on app startup
   */
  initialize: async (): Promise<void> => {
    try {
      // Try to refresh token to check if we have a valid session
      const token = await refreshAccessToken();

      if (!token) {
        // No valid session
        authState.isLoading = false;
        notifyListeners();
      }
    } catch (error) {
      console.error('Auth initialization failed:', error);
      authState.isLoading = false;
      notifyListeners();
    }
  },

  /**
   * Reset auth state synchronously for testing purposes only.
   * Unlike logout(), this does NOT:
   * - Make network requests to /api/auth/logout
   * - Broadcast logout events to other tabs via BroadcastChannel
   * 
   * Multi-tab sync is intentionally skipped to maintain test isolation
   * and avoid side effects between concurrent test runs.
   * 
   * @internal This method is intended for test use only.
   */
  resetForTesting: (): void => {
    authState = {
      user: null,
      isAuthenticated: false,
      isAdmin: false,
      isLoading: false,
      accessToken: null,
    };
    notifyListeners();
  },
};

/**
 * React hook for accessing auth state
 */
export const useAuth = () => {
  const [state, setState] = useState(() => authStore.getState());

  useEffect(() => {
    const unsubscribe = authStore.subscribe(setState);
    return unsubscribe;
  }, []);

  // Provide logout function that components can call
  const logout = useCallback(async () => {
    await authStore.logout();
  }, []);

  return {
    ...state,
    logout,
  };
};
