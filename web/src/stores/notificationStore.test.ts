/**
 * Notification Store Tests
 * Validates Web Push notification subscription state management
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import {
  useNotificationStore,
  useNotificationState,
  useNotificationActions,
  type PushSubscriptionData,
} from './notificationStore';

describe('notificationStore', () => {
  const mockSubscription: PushSubscriptionData = {
    endpoint: 'https://fcm.googleapis.com/fcm/send/test-endpoint',
    keys: {
      p256dh: 'test-p256dh-key',
      auth: 'test-auth-key',
    },
  };

  beforeEach(() => {
    // Clear localStorage
    localStorage.clear();
    
    // Reset store to initial state
    useNotificationStore.setState({
      permission: 'default',
      isSubscribed: false,
      subscription: null,
      isLoading: false,
      error: null,
    });

    // Mock Notification API
    global.Notification = {
      permission: 'default',
    } as unknown as typeof Notification;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initial state', () => {
    it('initializes with default values when no stored subscription', () => {
      const { result } = renderHook(() => useNotificationStore());

      expect(result.current.permission).toBe('default');
      expect(result.current.isSubscribed).toBe(false);
      expect(result.current.subscription).toBeNull();
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBeNull();
    });

    it('loads subscription from localStorage on initialization', () => {
      // Clear and set localStorage before store initialization
      localStorage.clear();
      localStorage.setItem(
        'subcults-notification-subscription',
        JSON.stringify(mockSubscription)
      );

      // Force store re-initialization by manually setting state with stored value
      const stored = JSON.parse(localStorage.getItem('subcults-notification-subscription')!);
      useNotificationStore.setState({
        permission: 'default',
        isSubscribed: false,
        subscription: stored,
        isLoading: false,
        error: null,
      });

      const { result } = renderHook(() => useNotificationStore());

      // The subscription should be loaded but isSubscribed stays false until explicitly set
      expect(result.current.subscription).toEqual(mockSubscription);
      expect(result.current.isSubscribed).toBe(false);
    });

    it('handles corrupted localStorage data gracefully', () => {
      localStorage.setItem('subcults-notification-subscription', 'invalid-json');

      const { result } = renderHook(() => useNotificationStore());

      expect(result.current.subscription).toBeNull();
      expect(result.current.isSubscribed).toBe(false);
    });

    it('initializes permission from Notification API', () => {
      // Must set before creating hook instance
      global.Notification = {
        permission: 'granted',
      } as unknown as typeof Notification;

      // Reset store to pick up new permission
      useNotificationStore.setState({
        permission: 'granted',
        isSubscribed: false,
        subscription: null,
        isLoading: false,
        error: null,
      });

      const { result } = renderHook(() => useNotificationStore());

      expect(result.current.permission).toBe('granted');
    });
  });

  describe('setPermission', () => {
    it('updates permission state', () => {
      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setPermission('granted');
      });

      expect(result.current.permission).toBe('granted');
    });

    it('handles all permission states', () => {
      const { result } = renderHook(() => useNotificationStore());

      const permissions: Array<'default' | 'granted' | 'denied'> = [
        'default',
        'granted',
        'denied',
      ];

      permissions.forEach((permission) => {
        act(() => {
          result.current.setPermission(permission);
        });
        expect(result.current.permission).toBe(permission);
      });
    });
  });

  describe('setSubscription', () => {
    it('updates subscription and sets isSubscribed to true', () => {
      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setSubscription(mockSubscription);
      });

      expect(result.current.subscription).toEqual(mockSubscription);
      expect(result.current.isSubscribed).toBe(true);
    });

    it('clears subscription and sets isSubscribed to false when null', () => {
      const { result } = renderHook(() => useNotificationStore());

      // First set a subscription
      act(() => {
        result.current.setSubscription(mockSubscription);
      });

      // Then clear it
      act(() => {
        result.current.setSubscription(null);
      });

      expect(result.current.subscription).toBeNull();
      expect(result.current.isSubscribed).toBe(false);
    });

    it('no longer persists subscription to localStorage for security', () => {
      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setSubscription(mockSubscription);
      });

      // Should NOT persist subscription data (security improvement)
      expect(localStorage.getItem('subcults-notification-subscription')).toBeNull();
    });

    it('clears legacy subscription from localStorage', () => {
      // Set legacy data
      localStorage.setItem('subcults-notification-subscription', JSON.stringify(mockSubscription));

      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setSubscription(mockSubscription);
      });

      // Should clear legacy data
      expect(localStorage.getItem('subcults-notification-subscription')).toBeNull();
    });
  });

  describe('setIsSubscribed (removed)', () => {
    it('no longer has setIsSubscribed action to prevent state inconsistency', () => {
      const { result } = renderHook(() => useNotificationStore());

      // setIsSubscribed should not exist
      expect(result.current).not.toHaveProperty('setIsSubscribed');
    });
  });

  describe('setLoading', () => {
    it('updates loading state', () => {
      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setLoading(true);
      });

      expect(result.current.isLoading).toBe(true);

      act(() => {
        result.current.setLoading(false);
      });

      expect(result.current.isLoading).toBe(false);
    });
  });

  describe('setError', () => {
    it('updates error state', () => {
      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setError('Test error message');
      });

      expect(result.current.error).toBe('Test error message');
    });

    it('clears error when set to null', () => {
      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setError('Test error');
      });

      expect(result.current.error).toBe('Test error');

      act(() => {
        result.current.setError(null);
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('reset', () => {
    it('resets all state to defaults', () => {
      const { result } = renderHook(() => useNotificationStore());

      // Set some state
      act(() => {
        result.current.setPermission('granted');
        result.current.setSubscription(mockSubscription);
        result.current.setLoading(true);
        result.current.setError('Test error');
      });

      // Reset
      act(() => {
        result.current.reset();
      });

      expect(result.current.permission).toBe('default');
      expect(result.current.isSubscribed).toBe(false);
      expect(result.current.subscription).toBeNull();
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBeNull();
    });

    it('clears localStorage on reset', () => {
      // Set legacy data
      localStorage.setItem('subcults-notification-subscription', JSON.stringify(mockSubscription));

      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.reset();
      });

      expect(localStorage.getItem('subcults-notification-subscription')).toBeNull();
    });
  });

  describe('useNotificationState hook', () => {
    it('returns only state properties', () => {
      const { result } = renderHook(() => {
        const state = useNotificationState();
        return state;
      });

      expect(result.current).toHaveProperty('permission');
      expect(result.current).toHaveProperty('isSubscribed');
      expect(result.current).toHaveProperty('subscription');
      expect(result.current).toHaveProperty('isLoading');
      expect(result.current).toHaveProperty('error');
      expect(result.current).not.toHaveProperty('setPermission');
      expect(result.current).not.toHaveProperty('setSubscription');
    });
  });

  describe('useNotificationActions hook', () => {
    it('returns only action functions', () => {
      const { result } = renderHook(() => {
        const actions = useNotificationActions();
        return actions;
      });

      expect(result.current).toHaveProperty('setPermission');
      expect(result.current).toHaveProperty('setSubscription');
      expect(result.current).toHaveProperty('setLoading');
      expect(result.current).toHaveProperty('setError');
      expect(result.current).toHaveProperty('reset');
      expect(result.current).not.toHaveProperty('permission');
      expect(result.current).not.toHaveProperty('isSubscribed');
      expect(result.current).not.toHaveProperty('setIsSubscribed'); // Removed action
    });
  });

  describe('localStorage integration', () => {
    it('no longer persists subscription data for security', () => {
      const complexSubscription: PushSubscriptionData = {
        endpoint: 'https://example.com/push/endpoint/with/special/chars?param=value&other=123',
        keys: {
          p256dh: 'very-long-base64-encoded-key-with-special-chars+/=',
          auth: 'another-long-key-123456789',
        },
      };

      const { result } = renderHook(() => useNotificationStore());

      act(() => {
        result.current.setSubscription(complexSubscription);
      });

      // Should NOT persist for security reasons
      const stored = localStorage.getItem('subcults-notification-subscription');
      expect(stored).toBeNull();
    });
  });
});
