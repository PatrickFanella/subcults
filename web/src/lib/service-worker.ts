/**
 * Service Worker Registration Utility
 * Handles service worker lifecycle and updates
 */

export interface ServiceWorkerUpdateHandler {
  onUpdateAvailable?: () => void;
  onUpdateInstalled?: () => void;
}

/**
 * Register service worker with update detection
 * Returns a cleanup function to stop update checks
 */
export async function registerServiceWorker(
  handlers: ServiceWorkerUpdateHandler = {}
): Promise<ServiceWorkerRegistration | null> {
  if (!('serviceWorker' in navigator)) {
    console.warn('[ServiceWorker] Not supported in this browser');
    return null;
  }

  try {
    const registration = await navigator.serviceWorker.register('/sw.js');
    console.log('[ServiceWorker] Registered:', registration);

    // Check for updates periodically (every hour)
    const updateInterval = setInterval(() => {
      registration.update();
    }, 60 * 60 * 1000);

    // Listen for updates
    registration.addEventListener('updatefound', () => {
      const newWorker = registration.installing;

      if (!newWorker) {
        return;
      }

      if (handlers.onUpdateAvailable) {
        handlers.onUpdateAvailable();
      }

      newWorker.addEventListener('statechange', () => {
        if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
          // New service worker available
          console.log('[ServiceWorker] New version available');

          if (handlers.onUpdateInstalled) {
            handlers.onUpdateInstalled();
          }
        }
      });
    });

    // Clean up interval on page unload
    const cleanup = () => {
      clearInterval(updateInterval);
    };
    
    window.addEventListener('beforeunload', cleanup);

    return registration;
  } catch (error) {
    console.error('[ServiceWorker] Registration failed:', error);
    return null;
  }
}

/**
 * Unregister service worker
 */
export async function unregisterServiceWorker(): Promise<boolean> {
  if (!('serviceWorker' in navigator)) {
    return false;
  }

  try {
    const registration = await navigator.serviceWorker.getRegistration();
    if (registration) {
      const success = await registration.unregister();
      console.log('[ServiceWorker] Unregistered:', success);
      return success;
    }
    return false;
  } catch (error) {
    console.error('[ServiceWorker] Unregistration failed:', error);
    return false;
  }
}

/**
 * Check if service worker is registered
 */
export async function isServiceWorkerRegistered(): Promise<boolean> {
  if (!('serviceWorker' in navigator)) {
    return false;
  }

  try {
    const registration = await navigator.serviceWorker.getRegistration();
    return !!registration;
  } catch (error) {
    return false;
  }
}

/**
 * Get service worker registration
 */
export async function getServiceWorkerRegistration(): Promise<ServiceWorkerRegistration | null> {
  if (!('serviceWorker' in navigator)) {
    return null;
  }

  try {
    return await navigator.serviceWorker.getRegistration() || null;
  } catch (error) {
    console.error('[ServiceWorker] Failed to get registration:', error);
    return null;
  }
}
