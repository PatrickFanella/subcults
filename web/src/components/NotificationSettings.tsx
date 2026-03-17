/**
 * NotificationSettings Component
 * User interface for managing Web Push notification preferences
 */

import React, { useState, useEffect } from 'react';
import { useNotificationStore } from '../stores/notificationStore';
import {
  isBrowserSupported,
  requestNotificationPermission,
  subscribeToPushNotifications,
  unsubscribeFromPushNotifications,
  sendSubscriptionToBackend,
  deleteSubscriptionFromBackend,
  getCurrentSubscription,
} from '../lib/notification-service';

export const NotificationSettings: React.FC = () => {
  const {
    permission,
    isSubscribed,
    subscription,
    isLoading,
    error,
    setPermission,
    setSubscription,
    setLoading,
    setError,
  } = useNotificationStore();

  const [browserSupported] = useState(isBrowserSupported());

  // Sync permission state on mount
  useEffect(() => {
    if (browserSupported && typeof Notification !== 'undefined') {
      setPermission(Notification.permission);
    }
  }, [browserSupported, setPermission]);

  // Check for existing subscription on mount
  useEffect(() => {
    const checkExistingSubscription = async () => {
      if (!browserSupported) return;

      try {
        const currentSub = await getCurrentSubscription();
        // Always sync store state to the current browser subscription.
        // If currentSub is null, clear the store subscription to avoid stale UI state.
        setSubscription(currentSub || null);
      } catch (error) {
        console.error('[NotificationSettings] Failed to check existing subscription:', error);
      }
    };

    checkExistingSubscription();
  }, [browserSupported, setSubscription]);

  const handleEnableNotifications = async () => {
    setLoading(true);
    setError(null);

    try {
      // Step 1: Request permission
      const newPermission = await requestNotificationPermission();
      setPermission(newPermission);

      if (newPermission !== 'granted') {
        setError(
          'Notification permission was denied. Please enable notifications in your browser settings.'
        );
        return;
      }

      // Step 2: Subscribe to push notifications
      const subscriptionData = await subscribeToPushNotifications();

      // Step 3: Immediately update local state so UI matches browser subscription
      setSubscription(subscriptionData);

      // Step 4: Send subscription to backend
      try {
        await sendSubscriptionToBackend(subscriptionData);
      } catch (backendError) {
        console.error(
          '[NotificationSettings] Failed to sync subscription with backend, rolling back subscription:',
          backendError
        );
        try {
          await unsubscribeFromPushNotifications();
        } catch (unsubscribeError) {
          console.error(
            '[NotificationSettings] Failed to roll back push subscription after backend failure:',
            unsubscribeError
          );
        }
        setSubscription(null);
        throw backendError;
      }

      // TODO: Increment metrics for successful subscription
      console.log('[NotificationSettings] Successfully subscribed to notifications');
    } catch (error) {
      console.error('[NotificationSettings] Failed to enable notifications:', error);
      setError(error instanceof Error ? error.message : 'Failed to enable notifications');
    } finally {
      setLoading(false);
    }
  };

  const handleDisableNotifications = async () => {
    setLoading(true);
    setError(null);

    try {
      // Step 1: Delete subscription from backend
      if (subscription) {
        await deleteSubscriptionFromBackend(subscription);
      }

      // Step 2: Unsubscribe from push notifications
      await unsubscribeFromPushNotifications();

      // Step 3: Update state
      setSubscription(null);

      console.log('[NotificationSettings] Successfully unsubscribed from notifications');
    } catch (error) {
      console.error('[NotificationSettings] Failed to disable notifications:', error);
      setError(error instanceof Error ? error.message : 'Failed to disable notifications');
    } finally {
      setLoading(false);
    }
  };

  if (!browserSupported) {
    return (
      <div className="bg-background-secondary border border-border rounded-none p-6 theme-transition">
        <h2 className="text-2xl font-semibold mb-4 text-foreground">Notifications</h2>
        <div className="bg-background border border-status-warning rounded-none p-4">
          <p className="text-foreground-secondary">
            <strong>Browser not supported:</strong> Your browser does not support Web Push
            notifications. Please use a modern browser like Chrome, Firefox, Edge, or Safari to
            enable notifications.
          </p>
        </div>
        <div className="mt-4 text-sm text-foreground-muted">
          <p className="mb-2">
            <strong>Privacy Note:</strong> Notifications are completely optional and require your
            explicit consent.
          </p>
          <p>
            We respect your privacy and will only send notifications about events that matter to
            you, such as new events near your location, live stream starts, or membership approvals.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-background-secondary border border-border rounded-none p-6 theme-transition">
      <h2 className="text-2xl font-semibold mb-4 text-foreground">Notifications</h2>

      <div className="space-y-4">
        {/* Status Display */}
        <div className="flex items-center justify-between py-4 border-b border-border">
          <div>
            <h3 className="text-lg font-medium text-foreground mb-1">Push Notifications</h3>
            <p className="text-sm text-foreground-secondary">
              {isSubscribed
                ? 'You will receive notifications about new events, streams, and updates'
                : 'Enable notifications to stay updated on events and streams'}
            </p>
          </div>

          <div className="flex items-center gap-4">
            {/* Status Badge */}
            <span
              className={`px-3 py-1 rounded-full text-sm font-medium ${
                isSubscribed
                  ? 'bg-background border border-status-success text-foreground'
                  : 'bg-background border border-border text-foreground-secondary'
              }`}
            >
              {isSubscribed ? 'Enabled' : 'Disabled'}
            </span>

            {/* Toggle Button */}
            <button
              onClick={isSubscribed ? handleDisableNotifications : handleEnableNotifications}
              disabled={isLoading || (!isSubscribed && permission === 'denied')}
              className={`px-4 py-2 rounded-none font-bold uppercase tracking-[0.05em] transition-none disabled:opacity-50 disabled:cursor-not-allowed ${
                isSubscribed
                  ? 'bg-status-error hover:opacity-90 text-white'
                  : 'bg-brand-primary hover:bg-brand-primary-dark text-white'
              }`}
            >
              {isLoading ? 'Processing...' : isSubscribed ? 'Disable' : 'Enable'}
            </button>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div className="bg-background border border-status-error rounded-none p-4">
            <p className="text-status-error text-sm">
              <strong>Error:</strong> {error}
            </p>
          </div>
        )}

        {/* Permission Denied Message */}
        {permission === 'denied' && (
          <div className="bg-background border border-status-warning rounded-none p-4">
            <p className="text-foreground-secondary text-sm">
              <strong>Permission denied:</strong> You have blocked notifications for this site. To
              enable notifications, please update your browser settings and reload the page.
            </p>
          </div>
        )}

        {/* Subscription Details (development only) */}
        {import.meta.env.DEV && subscription && (
          <details className="text-sm">
            <summary className="cursor-pointer text-foreground-secondary hover:text-foreground">
              Subscription Details (Dev Only)
            </summary>
            <pre className="mt-2 p-3 bg-background rounded-none border border-border overflow-x-auto text-xs text-foreground-muted">
              {JSON.stringify({ endpoint: subscription.endpoint }, null, 2)}
            </pre>
          </details>
        )}

        {/* Privacy Information */}
        <div className="mt-6 pt-6 border-t border-border">
          <h3 className="text-lg font-medium text-foreground mb-3">Privacy &amp; Consent</h3>
          <div className="text-sm text-foreground-secondary space-y-2">
            <p>
              <strong>Explicit Opt-In:</strong> Notifications are completely optional and will only
              be enabled if you explicitly grant permission.
            </p>
            <p>
              <strong>What We'll Notify You About:</strong> New events near you, live stream starts,
              scene membership approvals, and other updates you've subscribed to.
            </p>
            <p>
              <strong>Your Control:</strong> You can disable notifications at any time by clicking
              the "Disable" button above. You can also manage notification permissions in your
              browser settings.
            </p>
            <p>
              <strong>Privacy First:</strong> We respect your privacy. Notification subscriptions
              are stored securely and are only used to send you relevant updates. We never sell or
              share your subscription data with third parties.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
