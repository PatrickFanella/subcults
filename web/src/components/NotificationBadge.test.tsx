/**
 * NotificationBadge Tests
 * Validates notification badge rendering and behavior
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NotificationBadge } from './NotificationBadge';
import { useNotificationStore } from '../stores/notificationStore';

describe('NotificationBadge', () => {
  beforeEach(() => {
    // Reset notification store
    useNotificationStore.setState({
      permission: 'default',
      isSubscribed: false,
      subscription: null,
      isLoading: false,
      error: null,
    });
  });

  it('renders bell icon', () => {
    render(<NotificationBadge />);

    const button = screen.getByRole('button', { name: /notifications/i });
    expect(button).toBeInTheDocument();

    // Check for SVG icon
    const svg = button.querySelector('svg');
    expect(svg).toBeInTheDocument();
  });

  it('shows no badge when notification count is 0', () => {
    const { container } = render(<NotificationBadge notificationCount={0} />);

    // Badge should not be present
    const badge = container.querySelector('.bg-status-error');
    expect(badge).not.toBeInTheDocument();
  });

  it('shows badge with count when notifications > 0', () => {
    render(<NotificationBadge notificationCount={5} />);

    const button = screen.getByRole('button', { name: /notifications, 5 unread/i });
    expect(button).toBeInTheDocument();

    // Badge should show count
    expect(screen.getByText('5')).toBeInTheDocument();
  });

  it('shows "99+" for counts over 99', () => {
    render(<NotificationBadge notificationCount={150} />);

    expect(screen.getByText('99+')).toBeInTheDocument();
  });

  it('shows exactly 99 for count of 99', () => {
    render(<NotificationBadge notificationCount={99} />);

    expect(screen.getByText('99')).toBeInTheDocument();
  });

  it('shows subscription indicator when subscribed but no notifications', () => {
    useNotificationStore.setState({ isSubscribed: true });

    const { container } = render(<NotificationBadge notificationCount={0} />);

    // Should show subscription dot
    const dot = container.querySelector('.bg-brand-accent');
    expect(dot).toBeInTheDocument();
  });

  it('hides subscription indicator when has notifications', () => {
    useNotificationStore.setState({ isSubscribed: true });

    const { container } = render(<NotificationBadge notificationCount={3} />);

    // Should show badge, not dot
    expect(screen.getByText('3')).toBeInTheDocument();
    const dot = container.querySelector('.bg-brand-accent');
    expect(dot).not.toBeInTheDocument();
  });

  it('calls onClick handler when clicked', async () => {
    const handleClick = vi.fn();
    const user = userEvent.setup();

    render(<NotificationBadge onClick={handleClick} />);

    const button = screen.getByRole('button', { name: /notifications/i });
    await user.click(button);

    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('applies custom className', () => {
    render(<NotificationBadge className="custom-class" />);

    const button = screen.getByRole('button', { name: /notifications/i });
    expect(button).toHaveClass('custom-class');
  });

  it('has proper accessibility attributes', () => {
    render(<NotificationBadge notificationCount={7} />);

    const button = screen.getByRole('button', { name: /notifications, 7 unread/i });
    expect(button).toBeInTheDocument();

    // SVG should be hidden from screen readers
    const svg = button.querySelector('svg');
    expect(svg).toHaveAttribute('aria-hidden', 'true');
  });

  it('updates aria-label based on notification count', () => {
    const { rerender } = render(<NotificationBadge notificationCount={0} />);

    expect(screen.getByRole('button', { name: 'Notifications' })).toBeInTheDocument();

    rerender(<NotificationBadge notificationCount={3} />);

    expect(screen.getByRole('button', { name: 'Notifications, 3 unread' })).toBeInTheDocument();
  });
});
