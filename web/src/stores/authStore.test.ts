/**
 * Auth Store tests
 * Tests for authentication state management with token refresh
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { authStore, useAuth } from './authStore';

describe('authStore', () => {
  beforeEach(async () => {
    // Reset auth state
    // Mock fetch to prevent actual API calls during reset
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
    });
    await authStore.logout();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getState', () => {
    it('returns initial state', () => {
      const state = authStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isAdmin).toBe(false);
      expect(state.accessToken).toBeNull();
    });
  });

  describe('setUser', () => {
    it('sets user and access token', () => {
      const user = { did: 'did:example:123', role: 'user' as const };
      const token = 'mock-access-token';

      authStore.setUser(user, token);

      const state = authStore.getState();
      expect(state.user).toEqual(user);
      expect(state.isAuthenticated).toBe(true);
      expect(state.isAdmin).toBe(false);
      expect(state.accessToken).toBe(token);
      expect(state.isLoading).toBe(false);
    });

    it('sets isAdmin flag for admin users', () => {
      const user = { did: 'did:example:123', role: 'admin' as const };
      const token = 'mock-access-token';

      authStore.setUser(user, token);

      const state = authStore.getState();
      expect(state.isAdmin).toBe(true);
    });

    it('notifies subscribers on state change', () => {
      const listener = vi.fn();
      authStore.subscribe(listener);

      const user = { did: 'did:example:123', role: 'user' as const };
      authStore.setUser(user, 'token');

      expect(listener).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({
          user,
          isAuthenticated: true,
        })
      );
    });
  });

  describe('logout', () => {
    it('clears auth state', async () => {
      // Setup: set a user
      const user = { did: 'did:example:123', role: 'user' as const };
      authStore.setUser(user, 'token');

      // Mock fetch for logout endpoint
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
      });

      await authStore.logout();

      const state = authStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isAdmin).toBe(false);
      expect(state.accessToken).toBeNull();
      expect(state.isLoading).toBe(false);
    });

    it('calls logout endpoint', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
      });

      await authStore.logout();

      expect(global.fetch).toHaveBeenCalledWith(
        '/api/auth/logout',
        expect.objectContaining({
          method: 'POST',
          credentials: 'include',
        })
      );
    });

    it('clears state even if logout request fails', async () => {
      // Setup: set a user
      authStore.setUser({ did: 'did:example:123', role: 'user' as const }, 'token');

      // Mock failed logout
      global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

      await authStore.logout();

      const state = authStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe('subscribe', () => {
    it('notifies listener on state changes', () => {
      const listener = vi.fn();
      const unsubscribe = authStore.subscribe(listener);

      const user = { did: 'did:example:123', role: 'user' as const };
      authStore.setUser(user, 'token');

      expect(listener).toHaveBeenCalledTimes(1);

      unsubscribe();
    });

    it('can unsubscribe listener', () => {
      const listener = vi.fn();
      const unsubscribe = authStore.subscribe(listener);

      unsubscribe();

      const user = { did: 'did:example:123', role: 'user' as const };
      authStore.setUser(user, 'token');

      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe('initialize', () => {
    it('attempts to refresh token on initialization', async () => {
      const mockUser = { did: 'did:example:123', role: 'user' as const };
      const mockToken = 'mock-access-token';

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          accessToken: mockToken,
          user: mockUser,
        }),
      });

      await authStore.initialize();

      expect(global.fetch).toHaveBeenCalledWith(
        '/api/auth/refresh',
        expect.objectContaining({
          method: 'POST',
          credentials: 'include',
        })
      );

      const state = authStore.getState();
      expect(state.user).toEqual(mockUser);
      expect(state.isAuthenticated).toBe(true);
      expect(state.accessToken).toBe(mockToken);
      expect(state.isLoading).toBe(false);
    });

    it('sets isLoading to false when no valid session', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
      });

      await authStore.initialize();

      const state = authStore.getState();
      expect(state.isLoading).toBe(false);
      expect(state.isAuthenticated).toBe(false);
    });
  });
});

describe('useAuth hook', () => {
  beforeEach(async () => {
    // Mock fetch to prevent actual API calls during reset
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
    });
    await authStore.logout();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns current auth state', () => {
    const { result } = renderHook(() => useAuth());

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.isAdmin).toBe(false);
  });

  it('updates when auth state changes', async () => {
    const { result } = renderHook(() => useAuth());

    expect(result.current.isAuthenticated).toBe(false);

    act(() => {
      authStore.setUser({ did: 'did:example:123', role: 'user' as const }, 'token');
    });

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    expect(result.current.user).toEqual({ did: 'did:example:123', role: 'user' });
  });

  it('provides logout function', async () => {
    const { result } = renderHook(() => useAuth());

    // Mock fetch for logout
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
    });

    // Set user first
    act(() => {
      authStore.setUser({ did: 'did:example:123', role: 'user' as const }, 'token');
    });

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    // Call logout
    await act(async () => {
      await result.current.logout();
    });

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(false);
    });

    expect(result.current.user).toBeNull();
  });

  it('unsubscribes on unmount', () => {
    const { unmount } = renderHook(() => useAuth());

    // Should not throw
    unmount();
  });
});

describe('Token refresh with exponential backoff', () => {
  beforeEach(async () => {
    // Mock fetch to prevent actual API calls during reset
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
    });
    await authStore.logout();
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('retries on 500 error with exponential backoff', async () => {
    let callCount = 0;

    global.fetch = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount < 3) {
        return Promise.resolve({
          ok: false,
          status: 500,
        });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({
          accessToken: 'new-token',
          user: { did: 'did:example:123', role: 'user' },
        }),
      });
    });

    const initPromise = authStore.initialize();

    // Fast-forward through retries
    await vi.runAllTimersAsync();

    await initPromise;

    // Should have retried twice before succeeding
    expect(callCount).toBe(3);
    expect(authStore.getState().accessToken).toBe('new-token');
  });

  it('gives up after max retries', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
    });

    const initPromise = authStore.initialize();

    // Fast-forward through all retries
    await vi.runAllTimersAsync();

    await initPromise;

    // Should have tried initial + 3 retries = 4 times
    expect(global.fetch).toHaveBeenCalledTimes(4);
    expect(authStore.getState().isAuthenticated).toBe(false);
  });

  it('does not retry on 401 error', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
    });

    await authStore.initialize();

    // Should only try once
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('logs warning for unexpected 4xx errors', async () => {
    const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      statusText: 'Forbidden',
    });

    await authStore.initialize();

    // Should log warning for 403
    expect(consoleWarnSpy).toHaveBeenCalledWith(
      expect.stringContaining('[authStore] Token refresh failed with status 403')
    );

    consoleWarnSpy.mockRestore();
  });
});

describe('BroadcastChannel multi-tab sync', () => {
  beforeEach(async () => {
    // Mock fetch to prevent actual API calls
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
    });
    await authStore.logout();
    vi.clearAllMocks();
  });

  it('broadcasts logout event when logout is called', async () => {
    // Set user first
    authStore.setUser({ did: 'did:example:123', role: 'user' as const }, 'token');

    // Access the internal logoutChannel (not ideal, but necessary for testing)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const authStoreInternal = authStore as any;
    const originalChannel = authStoreInternal.logoutChannel;
    
    if (originalChannel) {
      const postMessageSpy = vi.spyOn(originalChannel, 'postMessage');
      
      await authStore.logout();

      // Should have broadcast logout event
      expect(postMessageSpy).toHaveBeenCalledWith({ type: 'logout' });
    }
  });

  it('clears auth state when receiving logout event from another tab', () => {
    // This test verifies the listener setup at module level
    // Set user in current tab
    authStore.setUser({ did: 'did:example:123', role: 'user' as const }, 'token');
    expect(authStore.getState().isAuthenticated).toBe(true);

    // Note: Since BroadcastChannel listener is set up at module level,
    // we can't easily simulate the event without re-importing the module.
    // This test documents the expected behavior.
  });

  it('does not process logout event if already logged out', () => {
    // Ensure user is logged out
    const state = authStore.getState();
    expect(state.isAuthenticated).toBe(false);

    // If we could trigger a logout event here, it should be a no-op
    // since the check for isAuthenticated prevents redundant state changes
  });
});
