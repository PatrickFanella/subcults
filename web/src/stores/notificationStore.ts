/**
 * Notification Store
 * Manages Web Push notification subscription state with localStorage persistence
 */

import { create } from 'zustand';

export type NotificationPermission = 'default' | 'granted' | 'denied';

export interface PushSubscriptionData {
  endpoint: string;
  keys: {
    p256dh: string;
    auth: string;
  };
}

interface NotificationState {
  permission: NotificationPermission;
  isSubscribed: boolean;
  subscription: PushSubscriptionData | null;
  isLoading: boolean;
  error: string | null;
}

interface NotificationActions {
  setPermission: (permission: NotificationPermission) => void;
  setSubscription: (subscription: PushSubscriptionData | null) => void;
  setLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

export type NotificationStore = NotificationState & NotificationActions;

/**
 * Local storage key for subscription state
 */
const NOTIFICATION_STORAGE_KEY = 'subcults-notification-subscription';

/**
 * Get initial subscription state from localStorage
 */
function getStoredSubscription(): PushSubscriptionData | null {
  try {
    // Clear any legacy stored subscription data — subscriptions should be
    // obtained from PushManager.getSubscription(), not from localStorage.
    localStorage.removeItem(NOTIFICATION_STORAGE_KEY);
  } catch (error) {
    console.warn('[notificationStore] Failed to clear legacy subscription:', error);
  }
  return null;
}

/**
 * Handle subscription persistence.
 *
 * For privacy and XSS resilience, we do NOT persist the full push subscription
 * (endpoint + keys) in localStorage. The active subscription should be derived
 * from `PushManager.getSubscription()` instead.
 *
 * This function is kept to preserve the existing API surface and to clear any
 * legacy stored subscription data that may still exist under the old key.
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function persistSubscription(_subscription: PushSubscriptionData | null): void {
  try {
    // Clear any legacy persisted subscription data; do not persist new data.
    localStorage.removeItem(NOTIFICATION_STORAGE_KEY);
  } catch (error) {
    console.warn('[notificationStore] Failed to update subscription persistence:', error);
  }
}

/**
 * Get initial notification permission
 */
function getInitialPermission(): NotificationPermission {
  if (typeof Notification === 'undefined') {
    return 'default';
  }
  return Notification.permission as NotificationPermission;
}

const initialSubscription = getStoredSubscription();

/**
 * Notification store for managing Web Push subscriptions
 */
export const useNotificationStore = create<NotificationStore>((set) => ({
  permission: getInitialPermission(),
  isSubscribed: initialSubscription !== null,
  subscription: initialSubscription,
  isLoading: false,
  error: null,

  setPermission: (permission: NotificationPermission) => {
    set({ permission });
  },

  setSubscription: (subscription: PushSubscriptionData | null) => {
    set({ subscription, isSubscribed: subscription !== null });
    persistSubscription(subscription);
  },

  setLoading: (isLoading: boolean) => {
    set({ isLoading });
  },

  setError: (error: string | null) => {
    set({ error });
  },

  reset: () => {
    set({
      permission: getInitialPermission(),
      isSubscribed: false,
      subscription: null,
      isLoading: false,
      error: null,
    });
    persistSubscription(null);
  },
}));

/**
 * Hook for notification state only (optimized for re-renders)
 */
export function useNotificationState() {
  const permission = useNotificationStore((state) => state.permission);
  const isSubscribed = useNotificationStore((state) => state.isSubscribed);
  const subscription = useNotificationStore((state) => state.subscription);
  const isLoading = useNotificationStore((state) => state.isLoading);
  const error = useNotificationStore((state) => state.error);
  
  return {
    permission,
    isSubscribed,
    subscription,
    isLoading,
    error,
  };
}

/**
 * Hook for notification actions only (stable reference)
 */
export function useNotificationActions() {
  const setPermission = useNotificationStore((state) => state.setPermission);
  const setSubscription = useNotificationStore((state) => state.setSubscription);
  const setLoading = useNotificationStore((state) => state.setLoading);
  const setError = useNotificationStore((state) => state.setError);
  const reset = useNotificationStore((state) => state.reset);
  
  return {
    setPermission,
    setSubscription,
    setLoading,
    setError,
    reset,
  };
}
