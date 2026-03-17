/**
 * NotificationBadge Component
 * Displays notification count with badge indicator
 */

import { useNotificationState } from '../stores/notificationStore';

export interface NotificationBadgeProps {
  /**
   * Additional CSS classes
   */
  className?: string;
  /**
   * Click handler
   */
  onClick?: () => void;
  /**
   * Number of unread notifications (optional, defaults to 0)
   */
  notificationCount?: number;
}

/**
 * NotificationBadge shows the current notification count
 */
export function NotificationBadge({
  className = '',
  onClick,
  notificationCount = 0,
}: NotificationBadgeProps) {
  const { isSubscribed } = useNotificationState();

  const hasNotifications = notificationCount > 0;

  return (
    <button
      onClick={onClick}
      aria-label={`Notifications${hasNotifications ? `, ${notificationCount} unread` : ''}`}
      className={`
        relative p-2 rounded-none min-h-touch min-w-touch
        text-foreground hover:bg-underground-lighter
        focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
        transition-none touch-manipulation
        ${className}
      `}
    >
      {/* Bell Icon */}
      <svg
        className="w-5 h-5 sm:w-6 sm:h-6"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
        />
      </svg>

      {/* Badge */}
      {hasNotifications && (
        <span
          className="
            absolute -top-1 -right-1
            inline-flex items-center justify-center
            min-w-[1.25rem] h-5 px-1
            text-xs font-bold text-white
            bg-status-error rounded-full
          "
          aria-hidden="true"
        >
          {notificationCount > 99 ? '99+' : notificationCount}
        </span>
      )}

      {/* Subscription indicator (dot) */}
      {isSubscribed && !hasNotifications && (
        <span
          className="
            absolute top-1 right-1
            w-2 h-2
            bg-brand-accent rounded-full
          "
          aria-hidden="true"
          title="Notifications enabled"
        />
      )}
    </button>
  );
}
