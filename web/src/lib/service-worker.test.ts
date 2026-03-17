/**
 * Service Worker Registration Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  registerServiceWorker,
  unregisterServiceWorker,
  isServiceWorkerRegistered,
  getServiceWorkerRegistration,
} from './service-worker';

describe('service-worker', () => {
  let mockRegistration: Partial<ServiceWorkerRegistration>;
  let mockServiceWorker: Partial<ServiceWorker>;

  beforeEach(() => {
    // Mock ServiceWorker API
    mockServiceWorker = {
      state: 'activated',
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    };

    mockRegistration = {
      installing: null,
      waiting: null,
      active: mockServiceWorker as ServiceWorker,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      update: vi.fn().mockResolvedValue(undefined),
      unregister: vi.fn().mockResolvedValue(true),
    };

    // Mock navigator.serviceWorker
    Object.defineProperty(navigator, 'serviceWorker', {
      value: {
        register: vi.fn().mockResolvedValue(mockRegistration),
        getRegistration: vi.fn().mockResolvedValue(mockRegistration),
        controller: null,
      },
      writable: true,
      configurable: true,
    });

    // Mock setInterval/clearInterval
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  describe('registerServiceWorker', () => {
    it('should register service worker successfully', async () => {
      const registration = await registerServiceWorker();

      expect(navigator.serviceWorker.register).toHaveBeenCalledWith('/sw.js');
      expect(registration).toBe(mockRegistration);
    });

    it('should return null if service worker is not supported', async () => {
      // Remove service worker support
      const originalServiceWorker = navigator.serviceWorker;
      // @ts-expect-error - Testing unsupported browser
      delete navigator.serviceWorker;

      const registration = await registerServiceWorker();

      expect(registration).toBeNull();

      // Restore
      Object.defineProperty(navigator, 'serviceWorker', {
        value: originalServiceWorker,
        writable: true,
        configurable: true,
      });
    });

    it('should call onUpdateAvailable when update is found', async () => {
      const onUpdateAvailable = vi.fn();
      const installingWorker = { ...mockServiceWorker, state: 'installing' } as ServiceWorker;
      
      mockRegistration.installing = installingWorker;

      await registerServiceWorker({ onUpdateAvailable });

      // Simulate updatefound event
      const updateFoundHandler = (mockRegistration.addEventListener as any).mock.calls.find(
        (call: any[]) => call[0] === 'updatefound'
      )?.[1];

      if (updateFoundHandler) {
        updateFoundHandler();
        expect(onUpdateAvailable).toHaveBeenCalled();
      }
    });

    it('should call onUpdateInstalled when new worker is installed', async () => {
      const onUpdateInstalled = vi.fn();
      const installingWorker = {
        ...mockServiceWorker,
        state: 'installing',
        addEventListener: vi.fn(),
      } as unknown as ServiceWorker;

      mockRegistration.installing = installingWorker;
      
      // Mock navigator.serviceWorker.controller to simulate existing worker
      Object.defineProperty(navigator.serviceWorker, 'controller', {
        value: mockServiceWorker,
        writable: true,
        configurable: true,
      });

      await registerServiceWorker({ onUpdateInstalled });

      // Simulate updatefound event
      const updateFoundHandler = (mockRegistration.addEventListener as any).mock.calls.find(
        (call: any[]) => call[0] === 'updatefound'
      )?.[1];

      if (updateFoundHandler) {
        updateFoundHandler();

        // Simulate statechange event
        const stateChangeHandler = (installingWorker.addEventListener as any).mock.calls.find(
          (call: any[]) => call[0] === 'statechange'
        )?.[1];

        if (stateChangeHandler) {
          // Change state to installed
          Object.defineProperty(installingWorker, 'state', {
            value: 'installed',
            writable: true,
          });

          stateChangeHandler();
          expect(onUpdateInstalled).toHaveBeenCalled();
        }
      }
    });

    it('should check for updates periodically', async () => {
      await registerServiceWorker();

      expect(mockRegistration.update).not.toHaveBeenCalled();

      // Fast-forward 1 hour
      vi.advanceTimersByTime(60 * 60 * 1000);

      expect(mockRegistration.update).toHaveBeenCalledTimes(1);

      // Fast-forward another hour
      vi.advanceTimersByTime(60 * 60 * 1000);

      expect(mockRegistration.update).toHaveBeenCalledTimes(2);
    });

    it('should handle registration errors', async () => {
      const error = new Error('Registration failed');
      (navigator.serviceWorker.register as any).mockRejectedValue(error);

      const registration = await registerServiceWorker();

      expect(registration).toBeNull();
    });
  });

  describe('unregisterServiceWorker', () => {
    it('should unregister service worker successfully', async () => {
      const result = await unregisterServiceWorker();

      expect(navigator.serviceWorker.getRegistration).toHaveBeenCalled();
      expect(mockRegistration.unregister).toHaveBeenCalled();
      expect(result).toBe(true);
    });

    it('should return false if service worker is not supported', async () => {
      // Remove service worker support
      const originalServiceWorker = navigator.serviceWorker;
      // @ts-expect-error - Testing unsupported browser
      delete navigator.serviceWorker;

      const result = await unregisterServiceWorker();

      expect(result).toBe(false);

      // Restore
      Object.defineProperty(navigator, 'serviceWorker', {
        value: originalServiceWorker,
        writable: true,
        configurable: true,
      });
    });

    it('should return false if no registration exists', async () => {
      (navigator.serviceWorker.getRegistration as any).mockResolvedValue(null);

      const result = await unregisterServiceWorker();

      expect(result).toBe(false);
    });

    it('should handle unregistration errors', async () => {
      const error = new Error('Unregistration failed');
      (navigator.serviceWorker.getRegistration as any).mockRejectedValue(error);

      const result = await unregisterServiceWorker();

      expect(result).toBe(false);
    });
  });

  describe('isServiceWorkerRegistered', () => {
    it('should return true if service worker is registered', async () => {
      const result = await isServiceWorkerRegistered();

      expect(result).toBe(true);
    });

    it('should return false if service worker is not registered', async () => {
      (navigator.serviceWorker.getRegistration as any).mockResolvedValue(null);

      const result = await isServiceWorkerRegistered();

      expect(result).toBe(false);
    });

    it('should return false if service worker is not supported', async () => {
      // Remove service worker support
      const originalServiceWorker = navigator.serviceWorker;
      // @ts-expect-error - Testing unsupported browser
      delete navigator.serviceWorker;

      const result = await isServiceWorkerRegistered();

      expect(result).toBe(false);

      // Restore
      Object.defineProperty(navigator, 'serviceWorker', {
        value: originalServiceWorker,
        writable: true,
        configurable: true,
      });
    });

    it('should handle errors gracefully', async () => {
      const error = new Error('Check failed');
      (navigator.serviceWorker.getRegistration as any).mockRejectedValue(error);

      const result = await isServiceWorkerRegistered();

      expect(result).toBe(false);
    });
  });

  describe('getServiceWorkerRegistration', () => {
    it('should return registration if service worker is registered', async () => {
      const result = await getServiceWorkerRegistration();

      expect(result).toBe(mockRegistration);
    });

    it('should return null if service worker is not registered', async () => {
      (navigator.serviceWorker.getRegistration as any).mockResolvedValue(null);

      const result = await getServiceWorkerRegistration();

      expect(result).toBeNull();
    });

    it('should return null if service worker is not supported', async () => {
      // Remove service worker support
      const originalServiceWorker = navigator.serviceWorker;
      // @ts-expect-error - Testing unsupported browser
      delete navigator.serviceWorker;

      const result = await getServiceWorkerRegistration();

      expect(result).toBeNull();

      // Restore
      Object.defineProperty(navigator, 'serviceWorker', {
        value: originalServiceWorker,
        writable: true,
        configurable: true,
      });
    });

    it('should handle errors gracefully', async () => {
      const error = new Error('Get registration failed');
      (navigator.serviceWorker.getRegistration as any).mockRejectedValue(error);

      const result = await getServiceWorkerRegistration();

      expect(result).toBeNull();
    });
  });
});
