/**
 * Notification Service
 * Handles Web Push notification subscription and API communication
 */

import type { PushSubscriptionData } from '../stores/notificationStore';
import { apiClient } from './api-client';

/**
 * Configuration for notification service
 */
interface NotificationServiceConfig {
  vapidPublicKey: string;
  apiEndpoint: string;
}

let serviceConfig: NotificationServiceConfig | null = null;

/**
 * Initialize the notification service with configuration
 */
export function initializeNotificationService(config: NotificationServiceConfig): void {
  serviceConfig = config;
}

/**
 * Check if browser supports Web Push notifications
 */
export function isBrowserSupported(): boolean {
  return (
    'Notification' in window &&
    'serviceWorker' in navigator &&
    'PushManager' in window
  );
}

/**
 * Get current notification permission state
 */
export function getNotificationPermission(): NotificationPermission {
  if (!isBrowserSupported()) {
    return 'default';
  }
  return Notification.permission;
}

/**
 * Request notification permission from user
 * Returns the permission state after request
 */
export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (!isBrowserSupported()) {
    throw new Error('Notifications not supported in this browser');
  }

  try {
    const permission = await Notification.requestPermission();
    return permission;
  } catch (error) {
    console.error('[notificationService] Permission request failed:', error);
    throw new Error('Failed to request notification permission');
  }
}

/**
 * Convert base64 VAPID key to Uint8Array for PushManager
 */
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, '+')
    .replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

/**
 * Subscribe to push notifications
 * Registers service worker and creates push subscription
 */
export async function subscribeToPushNotifications(): Promise<PushSubscriptionData> {
  if (!isBrowserSupported()) {
    throw new Error('Notifications not supported in this browser');
  }

  if (!serviceConfig) {
    throw new Error('Notification service not initialized. Call initializeNotificationService first.');
  }

  try {
    // Check permission first
    if (Notification.permission !== 'granted') {
      throw new Error('Notification permission not granted');
    }

    // Get service worker registration
    const registration = await navigator.serviceWorker.ready;

    // Check for existing subscription
    const existingSubscription = await registration.pushManager.getSubscription();
    
    // If subscription exists, unsubscribe first to ensure clean state
    if (existingSubscription) {
      await existingSubscription.unsubscribe();
    }

    // Subscribe to push notifications
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(serviceConfig.vapidPublicKey) as BufferSource,
    });

    // Convert subscription to serializable format
    const subscriptionData: PushSubscriptionData = {
      endpoint: subscription.endpoint,
      keys: {
        p256dh: arrayBufferToBase64(subscription.getKey('p256dh')),
        auth: arrayBufferToBase64(subscription.getKey('auth')),
      },
    };

    return subscriptionData;
  } catch (error) {
    console.error('[notificationService] Subscription failed:', error);
    // Re-throw known errors with their original message
    if (error instanceof Error && error.message === 'Notification permission not granted') {
      throw error;
    }
    throw new Error('Failed to subscribe to push notifications');
  }
}

/**
 * Unsubscribe from push notifications
 */
export async function unsubscribeFromPushNotifications(): Promise<void> {
  if (!isBrowserSupported()) {
    throw new Error('Notifications not supported in this browser');
  }

  try {
    const registration = await navigator.serviceWorker.ready;
    const subscription = await registration.pushManager.getSubscription();

    if (subscription) {
      await subscription.unsubscribe();
    }
  } catch (error) {
    console.error('[notificationService] Unsubscribe failed:', error);
    throw new Error('Failed to unsubscribe from push notifications');
  }
}

/**
 * Send subscription to backend API
 */
export async function sendSubscriptionToBackend(
  subscription: PushSubscriptionData
): Promise<void> {
  if (!serviceConfig) {
    throw new Error('Notification service not initialized');
  }

  try {
    await apiClient.post(serviceConfig.apiEndpoint, subscription);
  } catch (error) {
    console.error('[notificationService] Failed to send subscription to backend:', error);
    throw new Error('Failed to register subscription with server');
  }
}

/**
 * Delete subscription from backend API
 */
export async function deleteSubscriptionFromBackend(
  subscription: PushSubscriptionData
): Promise<void> {
  if (!serviceConfig) {
    throw new Error('Notification service not initialized');
  }

  try {
    await apiClient.delete(serviceConfig.apiEndpoint, {
      body: JSON.stringify({ endpoint: subscription.endpoint }),
    });
  } catch (error) {
    // Don't throw on 404 - subscription might already be deleted
    if (error && typeof error === 'object' && 'status' in error && error.status === 404) {
      return;
    }
    console.error('[notificationService] Failed to delete subscription from backend:', error);
    // Don't throw - best effort cleanup
  }
}

/**
 * Helper: Convert ArrayBuffer to base64 string
 */
function arrayBufferToBase64(buffer: ArrayBuffer | null): string {
  if (!buffer) {
    return '';
  }
  
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return window.btoa(binary);
}

/**
 * Get current push subscription if it exists
 */
export async function getCurrentSubscription(): Promise<PushSubscriptionData | null> {
  if (!isBrowserSupported()) {
    return null;
  }

  try {
    const registration = await navigator.serviceWorker.ready;
    const subscription = await registration.pushManager.getSubscription();

    if (!subscription) {
      return null;
    }

    return {
      endpoint: subscription.endpoint,
      keys: {
        p256dh: arrayBufferToBase64(subscription.getKey('p256dh')),
        auth: arrayBufferToBase64(subscription.getKey('auth')),
      },
    };
  } catch (error) {
    console.error('[notificationService] Failed to get current subscription:', error);
    return null;
  }
}
