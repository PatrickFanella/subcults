/**
 * Date and Time Formatting Utilities Tests
 * Tests for locale-aware date and time formatting functions
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { formatDate, formatTime, formatDateTime, formatRelativeTime } from './dateTime';
import i18n from '../i18n';

describe('dateTime utilities', () => {
  beforeEach(() => {
    // Set a fixed locale for consistent tests
    vi.spyOn(i18n, 'language', 'get').mockReturnValue('en');
  });

  describe('formatDate', () => {
    const testDate = new Date('2026-02-15T14:30:00Z');

    it('formats date in short format', () => {
      const result = formatDate(testDate, 'short');
      // Format: MM/DD/YYYY or similar depending on locale
      expect(result).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
    });

    it('formats date in long format', () => {
      const result = formatDate(testDate, 'long');
      // Should contain month name and year
      expect(result).toContain('2026');
      expect(result.length).toBeGreaterThan(10);
    });

    it('formats time only', () => {
      const result = formatDate(testDate, 'time');
      // Should contain time
      expect(result).toMatch(/\d{1,2}:\d{2}/);
    });

    it('formats date and time', () => {
      const result = formatDate(testDate, 'dateTime');
      // Should contain both date and time
      expect(result).toContain('2026');
      expect(result).toMatch(/\d{1,2}:\d{2}/);
    });

    it('handles string date input', () => {
      const result = formatDate('2026-02-15T14:30:00Z', 'short');
      expect(result).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
    });

    it('handles number timestamp input', () => {
      const timestamp = testDate.getTime();
      const result = formatDate(timestamp, 'short');
      expect(result).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
    });

    it('returns internationalized error for invalid date', () => {
      const result = formatDate('invalid-date', 'short');
      // With i18n mock, it returns the translation key
      expect(result).toBe('dateTime.invalidDate');
    });

    it('defaults to short format when no format specified', () => {
      const result = formatDate(testDate);
      expect(result).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
    });
  });

  describe('formatTime', () => {
    const testDate = new Date('2026-02-15T14:30:00Z');

    it('formats time correctly', () => {
      const result = formatTime(testDate);
      expect(result).toMatch(/\d{1,2}:\d{2}/);
    });

    it('handles invalid date', () => {
      const result = formatTime('invalid');
      expect(result).toBe('dateTime.invalidDate');
    });
  });

  describe('formatDateTime', () => {
    const testDate = new Date('2026-02-15T14:30:00Z');

    it('formats date and time together', () => {
      const result = formatDateTime(testDate);
      expect(result).toContain('2026');
      expect(result).toMatch(/\d{1,2}:\d{2}/);
    });

    it('handles invalid date', () => {
      const result = formatDateTime('invalid');
      expect(result).toBe('dateTime.invalidDate');
    });
  });

  describe('formatRelativeTime', () => {
    it('returns internationalized error for invalid date', () => {
      const result = formatRelativeTime('invalid-date');
      expect(result).toBe('dateTime.invalidDate');
    });

    it('formats seconds ago correctly', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 30 * 1000); // 30 seconds ago
      const result = formatRelativeTime(past);
      expect(result).toContain('30');
      expect(result.toLowerCase()).toMatch(/second|ago/);
    });

    it('formats minutes ago correctly', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 5 * 60 * 1000); // 5 minutes ago
      const result = formatRelativeTime(past);
      expect(result).toContain('5');
      expect(result.toLowerCase()).toMatch(/minute|ago/);
    });

    it('formats hours ago correctly', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 3 * 60 * 60 * 1000); // 3 hours ago
      const result = formatRelativeTime(past);
      expect(result).toContain('3');
      expect(result.toLowerCase()).toMatch(/hour|ago/);
    });

    it('formats days ago correctly', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000); // 2 days ago
      const result = formatRelativeTime(past);
      expect(result).toContain('2');
      expect(result.toLowerCase()).toMatch(/day|ago/);
    });

    it('formats months ago correctly', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 35 * 24 * 60 * 60 * 1000); // ~1 month ago
      const result = formatRelativeTime(past);
      expect(result.toLowerCase()).toMatch(/month|ago/);
    });

    it('formats years ago correctly', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 400 * 24 * 60 * 60 * 1000); // ~1 year ago
      const result = formatRelativeTime(past);
      expect(result.toLowerCase()).toMatch(/year|ago/);
    });

    it('formats future dates correctly', () => {
      const now = new Date();
      const future = new Date(now.getTime() + 5 * 60 * 1000); // 5 minutes in future
      const result = formatRelativeTime(future);
      expect(result).toContain('5');
      expect(result.toLowerCase()).toMatch(/minute|in/);
    });

    it('handles string input', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 2 * 60 * 60 * 1000); // 2 hours ago
      const result = formatRelativeTime(past.toISOString());
      expect(result).toContain('2');
      expect(result.toLowerCase()).toMatch(/hour|ago/);
    });

    it('handles number timestamp input', () => {
      const now = new Date();
      const past = new Date(now.getTime() - 10 * 60 * 1000); // 10 minutes ago
      const result = formatRelativeTime(past.getTime());
      expect(result).toContain('10');
      expect(result.toLowerCase()).toMatch(/minute|ago/);
    });
  });

  describe('locale handling', () => {
    it('uses current i18n language', () => {
      const testDate = new Date('2026-02-15T14:30:00Z');
      
      // The actual formatting will use the mocked locale
      const result = formatDate(testDate, 'long');
      
      // Just verify it returns a formatted string
      expect(typeof result).toBe('string');
      expect(result.length).toBeGreaterThan(0);
    });
  });
});
