/**
 * Error Mapping Utilities
 * Map API error codes to user-friendly messages
 */

import type { ApiClientError } from '../lib/api-client';

/**
 * Error message map
 * Maps error codes to user-friendly messages
 */
const ERROR_MESSAGES: Record<string, string> = {
  // Auth errors
  unauthorized: 'Your session has expired. Please log in again.',
  invalid_token: 'Invalid authentication token. Please log in again.',
  auth_failed: 'Authentication failed. Please check your credentials.',
  
  // Network errors
  network_error: 'Network error. Please check your connection and try again.',
  timeout: 'Request timed out. Please try again.',
  
  // Validation errors
  validation: 'Invalid input. Please check your data and try again.',
  invalid_scene_name: 'Scene name is invalid. Use only letters, numbers, spaces, dashes, underscores, and periods.',
  duplicate_scene_name: 'A scene with this name already exists.',
  invalid_time_range: 'Invalid time range. End time must be after start time.',
  
  // Resource errors
  not_found: 'The requested resource was not found.',
  conflict: 'A conflict occurred. The resource may have been modified.',
  
  // Permission errors
  forbidden: 'You do not have permission to perform this action.',
  
  // Server errors
  internal_error: 'An internal server error occurred. Please try again later.',
  service_unavailable: 'Service temporarily unavailable. Please try again later.',
  
  // Generic fallback
  unknown_error: 'An unexpected error occurred. Please try again.',
};

/**
 * Get user-friendly message for API error
 * 
 * @param error - API client error
 * @returns User-friendly error message
 */
export function getErrorMessage(error: ApiClientError | Error): string {
  if (error instanceof Error && 'code' in error) {
    const apiError = error as ApiClientError;
    return ERROR_MESSAGES[apiError.code] || apiError.message || ERROR_MESSAGES.unknown_error;
  }
  
  return error.message || ERROR_MESSAGES.unknown_error;
}

/**
 * Get error message for status code
 * Fallback when error code is not available
 * 
 * @param status - HTTP status code
 * @returns User-friendly error message
 */
export function getErrorMessageForStatus(status: number): string {
  if (status === 0) {
    return ERROR_MESSAGES.network_error;
  }
  
  if (status === 401) {
    return ERROR_MESSAGES.unauthorized;
  }
  
  if (status === 403) {
    return ERROR_MESSAGES.forbidden;
  }
  
  if (status === 404) {
    return ERROR_MESSAGES.not_found;
  }
  
  if (status === 409) {
    return ERROR_MESSAGES.conflict;
  }
  
  if (status >= 400 && status < 500) {
    return ERROR_MESSAGES.validation;
  }
  
  if (status >= 500) {
    return ERROR_MESSAGES.internal_error;
  }
  
  return ERROR_MESSAGES.unknown_error;
}

/**
 * Check if error should be auto-toasted
 * Some errors are handled by the application and should not show toasts
 * 
 * @param error - API client error
 * @returns Whether to show toast
 */
export function shouldShowToast(error: ApiClientError | Error): boolean {
  if (error instanceof Error && 'code' in error) {
    const apiError = error as ApiClientError;
    
    // Don't toast unauthorized errors (handled by auth system)
    if (apiError.code === 'unauthorized' || apiError.code === 'invalid_token') {
      return false;
    }
  }
  
  return true;
}
