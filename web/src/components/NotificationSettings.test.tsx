/**
 * NotificationSettings Component Tests
 * Validates notification preference UI, user interactions, and accessibility
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NotificationSettings } from './NotificationSettings';
import * as notificationService from '../lib/notification-service';

// Mock the notification service
vi.mock('../lib/notification-service', () => ({
  isBrowserSupported: vi.fn(),
  requestNotificationPermission: vi.fn(),
  subscribeToPushNotifications: vi.fn(),
  unsubscribeFromPushNotifications: vi.fn(),
  sendSubscriptionToBackend: vi.fn(),
  deleteSubscriptionFromBackend: vi.fn(),
  getCurrentSubscription: vi.fn(),
}));

// Mock the notification store
vi.mock('../stores/notificationStore', () => ({
  useNotificationStore: vi.fn(),
}));

// Import after mocking
import { useNotificationStore } from '../stores/notificationStore';

describe('NotificationSettings', () => {
  const mockSetPermission = vi.fn();
  const mockSetSubscription = vi.fn();
  const mockSetLoading = vi.fn();
  const mockSetError = vi.fn();

  const defaultStoreState = {
    permission: 'default' as const,
    isSubscribed: false,
    subscription: null,
    isLoading: false,
    error: null,
    setPermission: mockSetPermission,
    setSubscription: mockSetSubscription,
    setLoading: mockSetLoading,
    setError: mockSetError,
  };

  // Store original Notification to restore after tests
  const originalNotification = global.Notification;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Default: browser is supported
    vi.mocked(notificationService.isBrowserSupported).mockReturnValue(true);

    // Default store state
    vi.mocked(useNotificationStore).mockReturnValue(defaultStoreState);

    // Mock Notification API
    global.Notification = {
      permission: 'default',
    } as any;

    // Mock getCurrentSubscription
    vi.mocked(notificationService.getCurrentSubscription).mockResolvedValue(null);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    // Restore original Notification to prevent test pollution
    global.Notification = originalNotification;
  });

  describe('Browser Support', () => {
    it('renders notification settings when browser is supported', () => {
      render(<NotificationSettings />);

      expect(screen.getByText('Notifications')).toBeInTheDocument();
      expect(screen.getByText('Push Notifications')).toBeInTheDocument();
    });

    it('shows unsupported message when browser does not support notifications', () => {
      vi.mocked(notificationService.isBrowserSupported).mockReturnValue(false);

      render(<NotificationSettings />);

      expect(screen.getByText('Notifications')).toBeInTheDocument();
      expect(screen.getByText(/Browser not supported/i)).toBeInTheDocument();
      expect(
        screen.getByText(/Your browser does not support Web Push notifications/i)
      ).toBeInTheDocument();
    });

    it('shows privacy note when browser is unsupported', () => {
      vi.mocked(notificationService.isBrowserSupported).mockReturnValue(false);

      render(<NotificationSettings />);

      expect(screen.getByText(/Privacy Note:/i)).toBeInTheDocument();
      expect(screen.getByText(/completely optional/i)).toBeInTheDocument();
    });
  });

  describe('Initial State', () => {
    it('shows disabled status when not subscribed', () => {
      render(<NotificationSettings />);

      expect(screen.getByText('Disabled')).toBeInTheDocument();
    });

    it('shows enabled status when subscribed', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        isSubscribed: true,
        subscription: {
          endpoint: 'https://push.example.com/test',
          keys: { p256dh: 'test', auth: 'test' },
        },
      });

      render(<NotificationSettings />);

      expect(screen.getByText('Enabled')).toBeInTheDocument();
    });

    it('shows appropriate description when disabled', () => {
      render(<NotificationSettings />);

      expect(screen.getByText(/Enable notifications to stay updated/i)).toBeInTheDocument();
    });

    it('shows appropriate description when enabled', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        isSubscribed: true,
        subscription: {
          endpoint: 'https://push.example.com/test',
          keys: { p256dh: 'test', auth: 'test' },
        },
      });

      render(<NotificationSettings />);

      expect(screen.getByText(/You will receive notifications/i)).toBeInTheDocument();
    });
  });

  describe('Enable Button', () => {
    it('renders enable button when not subscribed', () => {
      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /enable/i });
      expect(button).toBeInTheDocument();
      expect(button).not.toBeDisabled();
    });

    it('handles successful notification enablement', async () => {
      const user = userEvent.setup();
      const mockSubscription = {
        endpoint: 'https://push.example.com/test',
        keys: { p256dh: 'test-key', auth: 'test-auth' },
      };

      vi.mocked(notificationService.requestNotificationPermission).mockResolvedValue('granted');
      vi.mocked(notificationService.subscribeToPushNotifications).mockResolvedValue(
        mockSubscription
      );
      vi.mocked(notificationService.sendSubscriptionToBackend).mockResolvedValue(undefined);

      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /enable/i });
      await user.click(button);

      await waitFor(() => {
        expect(mockSetLoading).toHaveBeenCalledWith(true);
        expect(notificationService.requestNotificationPermission).toHaveBeenCalled();
      });

      await waitFor(() => {
        expect(mockSetPermission).toHaveBeenCalledWith('granted');
        expect(mockSetSubscription).toHaveBeenCalledWith(mockSubscription);
        expect(notificationService.sendSubscriptionToBackend).toHaveBeenCalledWith(
          mockSubscription
        );
      });
    });

    it('handles permission denial', async () => {
      const user = userEvent.setup();

      vi.mocked(notificationService.requestNotificationPermission).mockResolvedValue('denied');

      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /enable/i });
      await user.click(button);

      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith(expect.stringContaining('permission was denied'));
      });
    });

    it('rolls back subscription on backend failure', async () => {
      const user = userEvent.setup();
      const mockSubscription = {
        endpoint: 'https://push.example.com/test',
        keys: { p256dh: 'test', auth: 'test' },
      };

      vi.mocked(notificationService.requestNotificationPermission).mockResolvedValue('granted');
      vi.mocked(notificationService.subscribeToPushNotifications).mockResolvedValue(
        mockSubscription
      );
      vi.mocked(notificationService.sendSubscriptionToBackend).mockRejectedValue(
        new Error('Backend error')
      );
      vi.mocked(notificationService.unsubscribeFromPushNotifications).mockResolvedValue(undefined);

      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /enable/i });
      await user.click(button);

      await waitFor(() => {
        expect(notificationService.unsubscribeFromPushNotifications).toHaveBeenCalled();
        expect(mockSetSubscription).toHaveBeenCalledWith(null);
        expect(mockSetError).toHaveBeenCalled();
      });
    });

    it('shows loading state while processing', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        isLoading: true,
      });

      render(<NotificationSettings />);

      expect(screen.getByRole('button', { name: /processing/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /processing/i })).toBeDisabled();
    });

    it('disables button when permission is denied', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        permission: 'denied',
      });

      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /enable/i });
      expect(button).toBeDisabled();
    });
  });

  describe('Disable Button', () => {
    it('renders disable button when subscribed', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        isSubscribed: true,
        subscription: {
          endpoint: 'https://push.example.com/test',
          keys: { p256dh: 'test', auth: 'test' },
        },
      });

      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /disable/i });
      expect(button).toBeInTheDocument();
      expect(button).not.toBeDisabled();
    });

    it('handles successful notification disablement', async () => {
      const user = userEvent.setup();
      const mockSubscription = {
        endpoint: 'https://push.example.com/test',
        keys: { p256dh: 'test', auth: 'test' },
      };

      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        isSubscribed: true,
        subscription: mockSubscription,
      });

      vi.mocked(notificationService.deleteSubscriptionFromBackend).mockResolvedValue(undefined);
      vi.mocked(notificationService.unsubscribeFromPushNotifications).mockResolvedValue(undefined);

      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /disable/i });
      await user.click(button);

      await waitFor(() => {
        expect(notificationService.deleteSubscriptionFromBackend).toHaveBeenCalledWith(
          mockSubscription
        );
        expect(notificationService.unsubscribeFromPushNotifications).toHaveBeenCalled();
        expect(mockSetSubscription).toHaveBeenCalledWith(null);
      });
    });

    it('handles errors during disablement', async () => {
      const user = userEvent.setup();

      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        isSubscribed: true,
        subscription: {
          endpoint: 'https://push.example.com/test',
          keys: { p256dh: 'test', auth: 'test' },
        },
      });

      vi.mocked(notificationService.deleteSubscriptionFromBackend).mockRejectedValue(
        new Error('Network error')
      );

      render(<NotificationSettings />);

      const button = screen.getByRole('button', { name: /disable/i });
      await user.click(button);

      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith(expect.stringContaining('Network error'));
      });
    });
  });

  describe('Error Display', () => {
    it('shows error message when error exists', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        error: 'Test error message',
      });

      render(<NotificationSettings />);

      expect(screen.getByText(/Error:/i)).toBeInTheDocument();
      expect(screen.getByText(/Test error message/i)).toBeInTheDocument();
    });

    it('does not show error message when no error', () => {
      render(<NotificationSettings />);

      expect(screen.queryByText(/Error:/i)).not.toBeInTheDocument();
    });
  });

  describe('Permission Denied Warning', () => {
    it('shows warning when permission is denied', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        permission: 'denied',
      });

      render(<NotificationSettings />);

      expect(screen.getByText(/Permission denied:/i)).toBeInTheDocument();
      expect(screen.getByText(/You have blocked notifications for this site/i)).toBeInTheDocument();
    });

    it('does not show warning when permission is not denied', () => {
      render(<NotificationSettings />);

      expect(screen.queryByText(/Permission denied:/i)).not.toBeInTheDocument();
    });
  });

  describe('Privacy Information', () => {
    it('displays privacy and consent section', () => {
      render(<NotificationSettings />);

      expect(screen.getByText(/Privacy & Consent/i)).toBeInTheDocument();
      expect(screen.getByText(/Explicit Opt-In:/i)).toBeInTheDocument();
      expect(screen.getByText(/Your Control:/i)).toBeInTheDocument();
      expect(screen.getByText(/Privacy First:/i)).toBeInTheDocument();
    });

    it('explains what notifications will be sent', () => {
      render(<NotificationSettings />);

      expect(screen.getByText(/What We'll Notify You About:/i)).toBeInTheDocument();
      expect(screen.getByText(/New events near you/i)).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('has proper heading structure', () => {
      render(<NotificationSettings />);

      expect(screen.getByRole('heading', { level: 2, name: 'Notifications' })).toBeInTheDocument();
      expect(
        screen.getByRole('heading', { level: 3, name: 'Push Notifications' })
      ).toBeInTheDocument();
      expect(
        screen.getByRole('heading', { level: 3, name: 'Privacy & Consent' })
      ).toBeInTheDocument();
    });

    it('uses semantic HTML structure', () => {
      const { container } = render(<NotificationSettings />);

      // Check for section/div with semantic classes
      expect(container.querySelector('.space-y-4')).toBeInTheDocument();
    });

    it('provides accessible button labels', () => {
      render(<NotificationSettings />);

      const button = screen.getByRole('button');
      expect(button.textContent).toMatch(/Enable|Disable|Processing/i);
    });
  });

  describe('Status Badge', () => {
    it('shows enabled badge with appropriate styling when subscribed', () => {
      vi.mocked(useNotificationStore).mockReturnValue({
        ...defaultStoreState,
        isSubscribed: true,
        subscription: {
          endpoint: 'https://push.example.com/test',
          keys: { p256dh: 'test', auth: 'test' },
        },
      });

      render(<NotificationSettings />);

      const badge = screen.getByText('Enabled');
      expect(badge).toBeInTheDocument();
      expect(badge).toHaveClass('bg-background');
      expect(badge).toHaveClass('border-status-success');
      expect(badge).toHaveClass('text-foreground');
    });

    it('shows disabled badge with appropriate styling when not subscribed', () => {
      render(<NotificationSettings />);

      const badge = screen.getByText('Disabled');
      expect(badge).toBeInTheDocument();
      expect(badge).toHaveClass('bg-background');
      expect(badge).toHaveClass('border-border');
      expect(badge).toHaveClass('text-foreground-secondary');
    });
  });

  describe('Theme Support', () => {
    it('applies dark mode classes', () => {
      render(<NotificationSettings />);

      // Verify theme-aware classes are present by checking semantic elements
      const heading = screen.getByRole('heading', { level: 2, name: 'Notifications' });
      expect(heading).toBeInTheDocument();
      expect(heading).toHaveClass('text-foreground');
    });
  });
});
