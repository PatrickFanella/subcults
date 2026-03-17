/**
 * Notification Service Tests
 * Validates Web Push API interactions and subscription management
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  initializeNotificationService,
  isBrowserSupported,
  getNotificationPermission,
  requestNotificationPermission,
  subscribeToPushNotifications,
  unsubscribeFromPushNotifications,
  sendSubscriptionToBackend,
  deleteSubscriptionFromBackend,
  getCurrentSubscription,
} from './notification-service';
import type { PushSubscriptionData } from '../stores/notificationStore';
import * as apiClientModule from './api-client';

// Mock the apiClient module
vi.mock('./api-client', () => ({
  apiClient: {
    post: vi.fn().mockResolvedValue(undefined),
    delete: vi.fn().mockResolvedValue(undefined),
  },
}));

describe('notification-service', () => {
  const mockConfig = {
    vapidPublicKey: 'BEl62iUYgUivxIkv69yViEuiBIa-Ib37J8xQmr8Db5s1234567890',
    apiEndpoint: '/api/notifications/subscribe',
  };

  const mockSubscriptionData: PushSubscriptionData = {
    endpoint: 'https://fcm.googleapis.com/fcm/send/test-endpoint',
    keys: {
      p256dh: 'test-p256dh-key',
      auth: 'test-auth-key',
    },
  };

  // Mock PushSubscription object
  const createMockPushSubscription = () => ({
    endpoint: mockSubscriptionData.endpoint,
    getKey: (name: string) => {
      if (name === 'p256dh') {
        return new TextEncoder().encode(mockSubscriptionData.keys.p256dh).buffer;
      }
      if (name === 'auth') {
        return new TextEncoder().encode(mockSubscriptionData.keys.auth).buffer;
      }
      return null;
    },
    unsubscribe: vi.fn().mockResolvedValue(true),
  });

  beforeEach(() => {
    // Clear all mocks
    vi.clearAllMocks();

    // Mock browser APIs
    global.Notification = {
      permission: 'default',
      requestPermission: vi.fn().mockResolvedValue('granted'),
    } as unknown as typeof Notification;

    // Mock PushManager
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (global as any).PushManager = class PushManager {};

    global.navigator.serviceWorker = {
      ready: Promise.resolve({
        pushManager: {
          subscribe: vi.fn().mockResolvedValue(createMockPushSubscription()),
          getSubscription: vi.fn().mockResolvedValue(null),
        },
      }),
    } as unknown as ServiceWorkerContainer;

    // Mock window.atob for base64 decoding
    global.atob = vi.fn((str: string) => str);

    // Initialize service
    initializeNotificationService(mockConfig);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initializeNotificationService', () => {
    it('initializes with configuration', () => {
      // Service is initialized in beforeEach
      // Just verify it doesn't throw
      expect(() => {
        initializeNotificationService(mockConfig);
      }).not.toThrow();
    });
  });

  describe('isBrowserSupported', () => {
    it('returns true when all required APIs are available', () => {
      expect(isBrowserSupported()).toBe(true);
    });

    it('returns false when Notification API is not available', () => {
      const originalNotification = global.Notification;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).Notification;

      expect(isBrowserSupported()).toBe(false);

      global.Notification = originalNotification;
    });

    it('returns false when serviceWorker is not available', () => {
      const originalServiceWorker = global.navigator.serviceWorker;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global.navigator as any).serviceWorker;

      expect(isBrowserSupported()).toBe(false);

      global.navigator.serviceWorker = originalServiceWorker;
    });

    it('returns false when PushManager is not available', () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const originalPushManager = (global as any).PushManager;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).PushManager;

      expect(isBrowserSupported()).toBe(false);

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (global as any).PushManager = originalPushManager;
    });
  });

  describe('getNotificationPermission', () => {
    it('returns current permission state', () => {
      global.Notification.permission = 'granted';
      expect(getNotificationPermission()).toBe('granted');
    });

    it('returns default when browser not supported', () => {
      const originalNotification = global.Notification;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).Notification;

      expect(getNotificationPermission()).toBe('default');

      global.Notification = originalNotification;
    });
  });

  describe('requestNotificationPermission', () => {
    it('requests permission and returns result', async () => {
      const permission = await requestNotificationPermission();

      expect(Notification.requestPermission).toHaveBeenCalled();
      expect(permission).toBe('granted');
    });

    it('throws error when browser not supported', async () => {
      const originalNotification = global.Notification;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).Notification;

      await expect(requestNotificationPermission()).rejects.toThrow(
        'Notifications not supported in this browser'
      );

      global.Notification = originalNotification;
    });

    it('throws error when permission request fails', async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (Notification.requestPermission as any) = vi
        .fn()
        .mockRejectedValue(new Error('User denied'));

      await expect(requestNotificationPermission()).rejects.toThrow(
        'Failed to request notification permission'
      );
    });
  });

  describe('subscribeToPushNotifications', () => {
    it('creates new subscription and returns data', async () => {
      global.Notification.permission = 'granted';

      const subscription = await subscribeToPushNotifications();

      expect(subscription).toBeDefined();
      expect(subscription.endpoint).toBe(mockSubscriptionData.endpoint);
      expect(subscription.keys).toBeDefined();
    });

    it('throws error when browser not supported', async () => {
      const originalNotification = global.Notification;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).Notification;

      await expect(subscribeToPushNotifications()).rejects.toThrow(
        'Notifications not supported in this browser'
      );

      global.Notification = originalNotification;
    });

    it('throws error when service not initialized', async () => {
      // Re-initialize with null config
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      initializeNotificationService(null as any);

      await expect(subscribeToPushNotifications()).rejects.toThrow(
        'Notification service not initialized'
      );

      // Restore config
      initializeNotificationService(mockConfig);
    });

    it('throws error when permission not granted', async () => {
      global.Notification.permission = 'denied';

      await expect(subscribeToPushNotifications()).rejects.toThrow(
        'Notification permission not granted'
      );
    });

    it('unsubscribes existing subscription before creating new one', async () => {
      global.Notification.permission = 'granted';
      const existingSubscription = createMockPushSubscription();

      global.navigator.serviceWorker = {
        ready: Promise.resolve({
          pushManager: {
            getSubscription: vi.fn().mockResolvedValue(existingSubscription),
            subscribe: vi.fn().mockResolvedValue(createMockPushSubscription()),
          },
        }),
      } as unknown as ServiceWorkerContainer;

      await subscribeToPushNotifications();

      expect(existingSubscription.unsubscribe).toHaveBeenCalled();
    });
  });

  describe('unsubscribeFromPushNotifications', () => {
    it('unsubscribes from existing subscription', async () => {
      const mockSubscription = createMockPushSubscription();

      global.navigator.serviceWorker = {
        ready: Promise.resolve({
          pushManager: {
            getSubscription: vi.fn().mockResolvedValue(mockSubscription),
          },
        }),
      } as unknown as ServiceWorkerContainer;

      await unsubscribeFromPushNotifications();

      expect(mockSubscription.unsubscribe).toHaveBeenCalled();
    });

    it('does nothing when no subscription exists', async () => {
      global.navigator.serviceWorker = {
        ready: Promise.resolve({
          pushManager: {
            getSubscription: vi.fn().mockResolvedValue(null),
          },
        }),
      } as unknown as ServiceWorkerContainer;

      await expect(unsubscribeFromPushNotifications()).resolves.not.toThrow();
    });

    it('throws error when browser not supported', async () => {
      const originalNotification = global.Notification;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).Notification;

      await expect(unsubscribeFromPushNotifications()).rejects.toThrow(
        'Notifications not supported in this browser'
      );

      global.Notification = originalNotification;
    });
  });

  describe('sendSubscriptionToBackend', () => {
    it('sends subscription to backend API', async () => {
      await sendSubscriptionToBackend(mockSubscriptionData);

      expect(apiClientModule.apiClient.post).toHaveBeenCalledWith(
        mockConfig.apiEndpoint,
        mockSubscriptionData
      );
    });

    it('throws error when service not initialized', async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      initializeNotificationService(null as any);

      await expect(sendSubscriptionToBackend(mockSubscriptionData)).rejects.toThrow(
        'Notification service not initialized'
      );

      initializeNotificationService(mockConfig);
    });

    it('throws error when API request fails', async () => {
      vi.mocked(apiClientModule.apiClient.post).mockRejectedValueOnce(new Error('API Error'));

      await expect(sendSubscriptionToBackend(mockSubscriptionData)).rejects.toThrow(
        'Failed to register subscription with server'
      );
    });
  });

  describe('deleteSubscriptionFromBackend', () => {
    it('deletes subscription from backend API', async () => {
      await deleteSubscriptionFromBackend(mockSubscriptionData);

      expect(apiClientModule.apiClient.delete).toHaveBeenCalledWith(
        mockConfig.apiEndpoint,
        {
          body: JSON.stringify({ endpoint: mockSubscriptionData.endpoint }),
        }
      );
    });

    it('throws error when service not initialized', async () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      initializeNotificationService(null as any);

      await expect(deleteSubscriptionFromBackend(mockSubscriptionData)).rejects.toThrow(
        'Notification service not initialized'
      );

      initializeNotificationService(mockConfig);
    });

    it('does not throw on 404 response', async () => {
      const error404 = { status: 404 };
      vi.mocked(apiClientModule.apiClient.delete).mockRejectedValueOnce(error404);

      await expect(deleteSubscriptionFromBackend(mockSubscriptionData)).resolves.not.toThrow();
    });

    it('does not throw on network error (best effort cleanup)', async () => {
      vi.mocked(apiClientModule.apiClient.delete).mockRejectedValueOnce(new Error('Network error'));

      await expect(deleteSubscriptionFromBackend(mockSubscriptionData)).resolves.not.toThrow();
    });
  });

  describe('getCurrentSubscription', () => {
    it('returns current subscription if exists', async () => {
      const mockSubscription = createMockPushSubscription();

      global.navigator.serviceWorker = {
        ready: Promise.resolve({
          pushManager: {
            getSubscription: vi.fn().mockResolvedValue(mockSubscription),
          },
        }),
      } as unknown as ServiceWorkerContainer;

      const subscription = await getCurrentSubscription();

      expect(subscription).toBeDefined();
      expect(subscription?.endpoint).toBe(mockSubscriptionData.endpoint);
    });

    it('returns null when no subscription exists', async () => {
      global.navigator.serviceWorker = {
        ready: Promise.resolve({
          pushManager: {
            getSubscription: vi.fn().mockResolvedValue(null),
          },
        }),
      } as unknown as ServiceWorkerContainer;

      const subscription = await getCurrentSubscription();

      expect(subscription).toBeNull();
    });

    it('returns null when browser not supported', async () => {
      const originalNotification = global.Notification;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (global as any).Notification;

      const subscription = await getCurrentSubscription();

      expect(subscription).toBeNull();

      global.Notification = originalNotification;
    });
  });
});
