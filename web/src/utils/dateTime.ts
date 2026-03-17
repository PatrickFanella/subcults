/**
 * Date and Time Formatting Utilities
 * Provides locale-aware date and time formatting
 */

import i18n from '../i18n';

/**
 * Format a date according to the current locale
 * @param date - Date to format
 * @param format - Format type ('short', 'long', 'time', 'dateTime')
 * @returns Formatted date string
 */
export function formatDate(date: Date | string | number, format: 'short' | 'long' | 'time' | 'dateTime' = 'short'): string {
  const dateObj = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date;
  
  if (isNaN(dateObj.getTime())) {
    return i18n.t('dateTime.invalidDate');
  }

  const locale = i18n.language || 'en';

  switch (format) {
    case 'short':
      return new Intl.DateTimeFormat(locale, {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
      }).format(dateObj);

    case 'long':
      return new Intl.DateTimeFormat(locale, {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      }).format(dateObj);

    case 'time':
      return new Intl.DateTimeFormat(locale, {
        hour: '2-digit',
        minute: '2-digit',
      }).format(dateObj);

    case 'dateTime':
      return new Intl.DateTimeFormat(locale, {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }).format(dateObj);

    default:
      return dateObj.toLocaleDateString(locale);
  }
}

/**
 * Format a time according to the current locale
 * @param date - Date to format
 * @returns Formatted time string
 */
export function formatTime(date: Date | string | number): string {
  return formatDate(date, 'time');
}

/**
 * Format a date and time according to the current locale
 * @param date - Date to format
 * @returns Formatted date and time string
 */
export function formatDateTime(date: Date | string | number): string {
  return formatDate(date, 'dateTime');
}

/**
 * Format a relative time (e.g., "2 hours ago", "in 3 days")
 * @param date - Date to format
 * @returns Formatted relative time string
 */
export function formatRelativeTime(date: Date | string | number): string {
  const dateObj = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date;
  
  if (isNaN(dateObj.getTime())) {
    return i18n.t('dateTime.invalidDate');
  }

  const locale = i18n.language || 'en';
  const now = new Date();
  const diffInSeconds = Math.floor((now.getTime() - dateObj.getTime()) / 1000);
  
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });

  if (Math.abs(diffInSeconds) < 60) {
    return rtf.format(-diffInSeconds, 'second');
  }

  const diffInMinutes = Math.floor(diffInSeconds / 60);
  if (Math.abs(diffInMinutes) < 60) {
    return rtf.format(-diffInMinutes, 'minute');
  }

  const diffInHours = Math.floor(diffInMinutes / 60);
  if (Math.abs(diffInHours) < 24) {
    return rtf.format(-diffInHours, 'hour');
  }

  const diffInDays = Math.floor(diffInHours / 24);
  if (Math.abs(diffInDays) < 30) {
    return rtf.format(-diffInDays, 'day');
  }

  const diffInMonths = Math.floor(diffInDays / 30);
  if (Math.abs(diffInMonths) < 12) {
    return rtf.format(-diffInMonths, 'month');
  }

  const diffInYears = Math.floor(diffInMonths / 12);
  return rtf.format(-diffInYears, 'year');
}
