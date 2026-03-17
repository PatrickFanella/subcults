/**
 * Error Mapping Utilities Tests
 * Validates error message mapping and filtering
 */

import { describe, it, expect } from 'vitest';
import { getErrorMessage, getErrorMessageForStatus, shouldShowToast } from './errorMapping';
import { ApiClientError } from '../lib/api-client';

describe('errorMapping', () => {
  describe('getErrorMessage', () => {
    it('returns mapped message for known error codes', () => {
      const error = new ApiClientError(401, 'unauthorized', 'Auth failed');
      expect(getErrorMessage(error)).toBe('Your session has expired. Please log in again.');
    });

    it('returns mapped message for validation errors', () => {
      const error = new ApiClientError(400, 'invalid_scene_name', 'Invalid name');
      expect(getErrorMessage(error)).toContain('Scene name is invalid');
    });

    it('returns original message for unknown error codes', () => {
      const error = new ApiClientError(500, 'custom_error', 'Custom error message');
      expect(getErrorMessage(error)).toBe('Custom error message');
    });

    it('returns fallback for errors without code', () => {
      const error = new Error('Generic error');
      expect(getErrorMessage(error)).toBe('Generic error');
    });

    it('returns fallback when message is empty', () => {
      const error = new ApiClientError(500, 'unknown_error', '');
      expect(getErrorMessage(error)).toBe('An unexpected error occurred. Please try again.');
    });

    it('handles network errors', () => {
      const error = new ApiClientError(0, 'network_error', 'Network failed');
      expect(getErrorMessage(error)).toContain('Network error');
    });

    it('handles timeout errors', () => {
      const error = new ApiClientError(0, 'timeout', 'Request timeout');
      expect(getErrorMessage(error)).toContain('Request timed out');
    });

    it('handles duplicate scene name errors', () => {
      const error = new ApiClientError(409, 'duplicate_scene_name', 'Duplicate');
      expect(getErrorMessage(error)).toContain('scene with this name already exists');
    });

    it('handles invalid time range errors', () => {
      const error = new ApiClientError(400, 'invalid_time_range', 'Bad time');
      expect(getErrorMessage(error)).toContain('End time must be after start time');
    });
  });

  describe('getErrorMessageForStatus', () => {
    it('returns network error for status 0', () => {
      expect(getErrorMessageForStatus(0)).toContain('Network error');
    });

    it('returns unauthorized message for 401', () => {
      expect(getErrorMessageForStatus(401)).toContain('session has expired');
    });

    it('returns forbidden message for 403', () => {
      expect(getErrorMessageForStatus(403)).toContain('do not have permission');
    });

    it('returns not found message for 404', () => {
      expect(getErrorMessageForStatus(404)).toContain('not found');
    });

    it('returns conflict message for 409', () => {
      expect(getErrorMessageForStatus(409)).toContain('conflict');
    });

    it('returns validation message for 4xx errors', () => {
      expect(getErrorMessageForStatus(400)).toContain('Invalid input');
      expect(getErrorMessageForStatus(422)).toContain('Invalid input');
    });

    it('returns internal error message for 5xx errors', () => {
      expect(getErrorMessageForStatus(500)).toContain('internal server error');
      expect(getErrorMessageForStatus(503)).toContain('internal server error');
    });

    it('returns fallback error for other status codes', () => {
      // 200 should return unknown error fallback
      expect(getErrorMessageForStatus(200)).toContain('unexpected error');
      // 999 is >= 500, so it returns internal error message
      expect(getErrorMessageForStatus(999)).toContain('internal server error');
    });
  });

  describe('shouldShowToast', () => {
    it('returns false for unauthorized errors', () => {
      const error = new ApiClientError(401, 'unauthorized', 'Auth failed');
      expect(shouldShowToast(error)).toBe(false);
    });

    it('returns false for invalid token errors', () => {
      const error = new ApiClientError(401, 'invalid_token', 'Invalid token');
      expect(shouldShowToast(error)).toBe(false);
    });

    it('returns true for validation errors', () => {
      const error = new ApiClientError(400, 'validation', 'Validation failed');
      expect(shouldShowToast(error)).toBe(true);
    });

    it('returns true for network errors', () => {
      const error = new ApiClientError(0, 'network_error', 'Network failed');
      expect(shouldShowToast(error)).toBe(true);
    });

    it('returns true for server errors', () => {
      const error = new ApiClientError(500, 'internal_error', 'Server error');
      expect(shouldShowToast(error)).toBe(true);
    });

    it('returns true for generic errors', () => {
      const error = new Error('Generic error');
      expect(shouldShowToast(error)).toBe(true);
    });

    it('returns true for not found errors', () => {
      const error = new ApiClientError(404, 'not_found', 'Not found');
      expect(shouldShowToast(error)).toBe(true);
    });
  });
});
