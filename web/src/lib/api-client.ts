/**
 * API Client
 * HTTP client with automatic token refresh and retry logic
 */

import type { Scene, Event } from '../types/scene';
import type { Post } from '../types/search';

export interface RequestConfig extends RequestInit {
  skipAuth?: boolean; // Skip adding Authorization header
  skipRetry?: boolean; // Skip retry on 401
  timeout?: number; // Request timeout in milliseconds (default: 10000)
  skipAutoRetry?: boolean; // Skip automatic retry on network/5xx errors
}

export interface ApiError {
  status: number;
  code: string;
  message: string;
  retryCount: number; // Number of retry attempts made
}

export class ApiClientError extends Error {
  public status: number;
  public code: string;
  public retryCount: number;

  constructor(status: number, code: string, message: string, retryCount: number = 0) {
    super(message);
    this.name = 'ApiClientError';
    this.status = status;
    this.code = code;
    this.retryCount = retryCount;
  }
}

/**
 * Token refresh callback type
 * Should return new access token or null if refresh failed
 */
export type RefreshTokenCallback = () => Promise<string | null>;

/**
 * Telemetry event types
 */
export interface ApiRequestEvent {
  type: 'api_request';
  method: string;
  endpoint: string;
  timestamp: number;
}

export interface ApiResponseEvent {
  type: 'api_response';
  method: string;
  endpoint: string;
  status: number;
  duration: number; // milliseconds
  retryCount: number;
  timestamp: number;
}

export type TelemetryEvent = ApiRequestEvent | ApiResponseEvent;
export type TelemetryCallback = (event: TelemetryEvent) => void;

/**
 * Default retry configuration
 */
const RETRY_CONFIG = {
  maxAttempts: 3,
  initialDelay: 100, // 100ms
  maxDelay: 2000, // 2s
  jitterFactor: 0.3, // 30% random jitter
};

/**
 * Default timeout for requests (10 seconds)
 */
const DEFAULT_TIMEOUT = 10000;

/**
 * Check if HTTP method is idempotent and can be safely retried
 */
function isIdempotentMethod(method: string): boolean {
  const idempotentMethods = ['GET', 'PUT', 'DELETE', 'HEAD', 'OPTIONS'];
  return idempotentMethods.includes(method.toUpperCase());
}

/**
 * Check if error/response should trigger a retry
 * Retries on network errors (status 0) and 5xx server errors
 */
function shouldRetry(status: number, method: string, skipAutoRetry: boolean): boolean {
  if (skipAutoRetry) return false;
  if (!isIdempotentMethod(method)) return false;
  // Retry on network errors (0) or 5xx server errors
  return status === 0 || (status >= 500 && status < 600);
}

/**
 * Calculate delay with exponential backoff and jitter
 */
function calculateDelay(attempt: number): number {
  const exponentialDelay = Math.min(
    RETRY_CONFIG.initialDelay * Math.pow(2, attempt),
    RETRY_CONFIG.maxDelay
  );

  // Add random jitter to prevent thundering herd
  const jitter = exponentialDelay * RETRY_CONFIG.jitterFactor * (Math.random() - 0.5) * 2;

  return Math.max(0, exponentialDelay + jitter);
}

/**
 * Sleep utility for retry delays
 */
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * API Client configuration
 */
interface ApiClientConfig {
  baseURL: string;
  getAccessToken: () => string | null;
  refreshToken: RefreshTokenCallback;
  onUnauthorized: () => void; // Called when refresh fails
  onTelemetry?: TelemetryCallback; // Optional telemetry callback
}

class ApiClient {
  private config: ApiClientConfig | null = null;
  private refreshPromise: Promise<string | null> | null = null;

  /**
   * Initialize the API client
   * Must be called before making any requests
   */
  initialize(config: ApiClientConfig): void {
    this.config = config;
  }

  /**
   * Make an authenticated API request
   * Automatically adds Authorization header, handles 401 with token refresh,
   * implements retry logic for idempotent methods, and emits telemetry events
   */
  async request<T>(endpoint: string, options: RequestConfig = {}): Promise<T> {
    if (!this.config) {
      throw new Error('ApiClient not initialized. Call initialize() first.');
    }

    const {
      skipAuth,
      skipRetry,
      timeout = DEFAULT_TIMEOUT,
      skipAutoRetry = false,
      ...fetchOptions
    } = options;

    const method = fetchOptions.method?.toUpperCase() || 'GET';
    const url = `${this.config.baseURL}${endpoint}`;
    const startTime = Date.now();

    // Emit request telemetry
    this.emitTelemetry({
      type: 'api_request',
      method,
      endpoint,
      timestamp: startTime,
    });

    // Execute request with retry logic
    let lastError: ApiClientError | null = null;

    for (let attempt = 0; attempt < RETRY_CONFIG.maxAttempts; attempt++) {
      try {
        // Create AbortController for timeout
        const abortController = new AbortController();
        const timeoutId = setTimeout(() => abortController.abort(), timeout);

        try {
          const response = await this.executeRequest(
            url,
            fetchOptions,
            skipAuth,
            skipRetry,
            abortController.signal
          );

          clearTimeout(timeoutId);

          // Emit success telemetry
          this.emitTelemetry({
            type: 'api_response',
            method,
            endpoint,
            status: response.status,
            duration: Date.now() - startTime,
            retryCount: attempt,
            timestamp: Date.now(),
          });

          return response.data as T;
        } catch (error) {
          clearTimeout(timeoutId);

          // Handle AbortError (timeout)
          if (error instanceof Error && error.name === 'AbortError') {
            lastError = new ApiClientError(
              0,
              'timeout',
              `Request timeout after ${timeout}ms`,
              attempt
            );
          } else if (error instanceof ApiClientError) {
            lastError = error;
            lastError.retryCount = attempt;
          } else {
            // Network error or other fetch failure
            lastError = new ApiClientError(
              0,
              'network_error',
              error instanceof Error ? error.message : 'Network request failed',
              attempt
            );
          }

          // Determine if we should retry
          const shouldRetryRequest = shouldRetry(lastError.status, method, skipAutoRetry);

          // Don't retry on last attempt
          if (!shouldRetryRequest || attempt === RETRY_CONFIG.maxAttempts - 1) {
            // Emit error telemetry
            this.emitTelemetry({
              type: 'api_response',
              method,
              endpoint,
              status: lastError.status,
              duration: Date.now() - startTime,
              retryCount: attempt,
              timestamp: Date.now(),
            });

            throw lastError;
          }

          // Calculate delay and wait before retry
          const delay = calculateDelay(attempt);
          await sleep(delay);
        }
      } catch (error) {
        // If error is thrown from the inner try-catch, rethrow it
        if (error instanceof ApiClientError) {
          throw error;
        }
        // Unexpected error, wrap and throw
        throw new ApiClientError(
          0,
          'unknown_error',
          error instanceof Error ? error.message : 'Unknown error occurred',
          attempt
        );
      }
    }

    // Should never reach here, but TypeScript requires a return
    throw lastError || new ApiClientError(0, 'unknown_error', 'Unknown error occurred', 0);
  }

  /**
   * Execute a single request attempt
   * Handles authentication, 401 refresh, and error parsing
   */
  private async executeRequest(
    url: string,
    fetchOptions: RequestInit,
    skipAuth: boolean | undefined,
    skipRetry: boolean | undefined,
    signal: AbortSignal
  ): Promise<{ data: Record<string, unknown> | string | object; status: number }> {
    if (!this.config) {
      throw new Error('ApiClient not initialized');
    }

    // Prepare headers
    const headers = new Headers(fetchOptions.headers);

    // Add Authorization header if not skipped and token exists
    if (!skipAuth) {
      const token = this.config.getAccessToken();
      if (token) {
        headers.set('Authorization', `Bearer ${token}`);
      }
    }

    // Set default Content-Type for POST/PUT/PATCH if not specified
    if (
      ['POST', 'PUT', 'PATCH'].includes(fetchOptions.method?.toUpperCase() || 'GET') &&
      !headers.has('Content-Type') &&
      fetchOptions.body
    ) {
      headers.set('Content-Type', 'application/json');
    }

    // Make request
    let response = await fetch(url, {
      ...fetchOptions,
      headers,
      credentials: fetchOptions.credentials || 'include', // Include cookies by default
      signal,
    });

    // Handle 401 Unauthorized
    if (response.status === 401 && !skipRetry && !skipAuth) {
      // Attempt to refresh token
      const newToken = await this.handleTokenRefresh();

      if (newToken) {
        // Retry original request with new token
        headers.set('Authorization', `Bearer ${newToken}`);
        response = await fetch(url, {
          ...fetchOptions,
          headers,
          credentials: fetchOptions.credentials || 'include',
          signal,
        });
      } else {
        // Refresh failed, onUnauthorized was called and will handle redirect
        throw new ApiClientError(401, 'unauthorized', 'Session expired. Redirecting to login.');
      }
    }

    // Check for error responses
    if (!response.ok) {
      const error = await this.parseError(response);
      throw new ApiClientError(error.status, error.code, error.message);
    }

    // Parse response
    const contentType = response.headers.get('content-type');
    let data: Record<string, unknown> | string | object;

    if (contentType?.includes('application/json')) {
      data = await response.json();
    } else if (response.status === 204) {
      // Return empty object for 204 No Content
      data = {};
    } else {
      // For non-JSON responses, return as text
      data = await response.text();
    }

    return { data, status: response.status };
  }

  /**
   * Emit telemetry event if callback is configured
   */
  private emitTelemetry(event: TelemetryEvent): void {
    if (this.config?.onTelemetry) {
      try {
        this.config.onTelemetry(event);
      } catch (error) {
        // Never let telemetry errors break the request
        console.warn('[apiClient] Telemetry callback error:', error);
      }
    }
  }

  /**
   * Handle token refresh with deduplication
   * Multiple concurrent requests will share the same refresh promise
   */
  private async handleTokenRefresh(): Promise<string | null> {
    if (!this.config) {
      return null;
    }

    // If refresh is already in progress, wait for it
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    // Start new refresh
    this.refreshPromise = this.config.refreshToken();

    try {
      const newToken = await this.refreshPromise;

      // Clear refresh promise on success or failure
      this.refreshPromise = null;

      // If refresh failed, notify the app
      if (!newToken) {
        this.config.onUnauthorized();
      }

      return newToken;
    } catch (error) {
      // Catch any errors during refresh (network failures, etc.)
      // Log error for debugging while maintaining user experience
      console.warn('[apiClient] Token refresh failed:', error);

      // Clear refresh promise on error
      this.refreshPromise = null;

      // Notify the app of unauthorized state
      this.config.onUnauthorized();

      return null;
    }
  }

  /**
   * Parse error response
   */
  private async parseError(response: Response): Promise<ApiError> {
    const contentType = response.headers.get('content-type');

    if (contentType?.includes('application/json')) {
      try {
        const body = await response.json();
        return {
          status: response.status,
          code: body.code || 'unknown_error',
          message: body.message || response.statusText,
          retryCount: 0, // Will be set by caller
        };
      } catch {
        // Fall through to default error
      }
    }

    return {
      status: response.status,
      code: 'unknown_error',
      message: response.statusText || 'An error occurred',
      retryCount: 0, // Will be set by caller
    };
  }

  /**
   * Convenience methods
   */
  async get<T>(endpoint: string, options?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'GET' });
  }

  async post<T>(endpoint: string, data?: unknown, options?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: unknown, options?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T>(endpoint: string, data?: unknown, options?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string, options?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'DELETE' });
  }

  /**
   * Fetch LiveKit token for joining an audio room
   * @param roomId - Room identifier
   * @param sceneId - Optional scene ID
   * @param eventId - Optional event ID
   */
  async getLiveKitToken(
    roomId: string,
    sceneId?: string,
    eventId?: string
  ): Promise<{ token: string; expires_at: string }> {
    return this.post('/livekit/token', {
      room_id: roomId,
      scene_id: sceneId,
      event_id: eventId,
    });
  }

  /**
   * Search scenes by query string
   * @param query - Search query
   * @param limit - Maximum results (default: 10)
   * @param signal - AbortSignal for request cancellation
   * @returns Promise resolving to array of scenes matching the query
   * @note Results respect visibility settings and location privacy. Private scenes are excluded.
   */
  async searchScenes(query: string, limit: number = 10, signal?: AbortSignal): Promise<Scene[]> {
    return this.get(`/search/scenes?q=${encodeURIComponent(query)}&limit=${limit}`, {
      signal,
      skipAutoRetry: true, // Don't retry searches
    });
  }

  /**
   * Search events by query string
   * @param query - Search query
   * @param limit - Maximum results (default: 10)
   * @param signal - AbortSignal for request cancellation
   * @returns Promise resolving to array of events matching the query
   * @note Results respect visibility settings and location privacy. Private events are excluded.
   */
  async searchEvents(query: string, limit: number = 10, signal?: AbortSignal): Promise<Event[]> {
    return this.get(`/search/events?q=${encodeURIComponent(query)}&limit=${limit}`, {
      signal,
      skipAutoRetry: true, // Don't retry searches
    });
  }

  /**
   * Search posts by query string
   * @param query - Search query
   * @param limit - Maximum results (default: 10)
   * @param signal - AbortSignal for request cancellation
   * @returns Promise resolving to array of posts matching the query
   * @note Results respect visibility settings and location privacy. Private posts are excluded.
   */
  async searchPosts(query: string, limit: number = 10, signal?: AbortSignal): Promise<Post[]> {
    return this.get(`/search/posts?q=${encodeURIComponent(query)}&limit=${limit}`, {
      signal,
      skipAutoRetry: true, // Don't retry searches
    });
  }
}

// Export singleton instance
export const apiClient = new ApiClient();
