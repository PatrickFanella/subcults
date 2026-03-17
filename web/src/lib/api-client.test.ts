/**
 * API Client tests
 * Tests for HTTP client with automatic token refresh
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { apiClient, ApiClientError } from './api-client';

describe('ApiClient', () => {
  const mockBaseURL = 'http://localhost:3000';
  let mockGetAccessToken: ReturnType<typeof vi.fn>;
  let mockRefreshToken: ReturnType<typeof vi.fn>;
  let mockOnUnauthorized: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // Reset mocks
    mockGetAccessToken = vi.fn();
    mockRefreshToken = vi.fn();
    mockOnUnauthorized = vi.fn();

    // Initialize API client with mocks
    apiClient.initialize({
      baseURL: mockBaseURL,
      getAccessToken: mockGetAccessToken,
      refreshToken: mockRefreshToken,
      onUnauthorized: mockOnUnauthorized,
    });

    // Clear fetch mocks
    vi.clearAllMocks();
  });

  describe('request', () => {
    it('adds Authorization header when token is available', async () => {
      const mockToken = 'mock-access-token';
      mockGetAccessToken.mockReturnValue(mockToken);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'success' }),
      });

      await apiClient.get('/test');

      expect(global.fetch).toHaveBeenCalledWith(
        `${mockBaseURL}/test`,
        expect.objectContaining({
          headers: expect.any(Headers),
        })
      );

      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const headers = call[1].headers as Headers;
      expect(headers.get('Authorization')).toBe(`Bearer ${mockToken}`);
    });

    it('does not add Authorization header when skipAuth is true', async () => {
      mockGetAccessToken.mockReturnValue('mock-token');

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'success' }),
      });

      await apiClient.get('/test', { skipAuth: true });

      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const headers = call[1].headers as Headers;
      expect(headers.get('Authorization')).toBeNull();
    });

    it('refreshes token and retries on 401 response', async () => {
      const oldToken = 'old-token';
      const newToken = 'new-token';

      mockGetAccessToken.mockReturnValue(oldToken);
      mockRefreshToken.mockResolvedValue(newToken);

      // First call returns 401, second call succeeds
      global.fetch = vi
        .fn()
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'unauthorized', message: 'Token expired' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ data: 'success' }),
        });

      const result = await apiClient.get<{ data: string }>('/test');

      expect(mockRefreshToken).toHaveBeenCalledTimes(1);
      expect(global.fetch).toHaveBeenCalledTimes(2);
      expect(result).toEqual({ data: 'success' });

      // Second call should use new token
      const secondCall = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[1];
      const headers = secondCall[1].headers as Headers;
      expect(headers.get('Authorization')).toBe(`Bearer ${newToken}`);
    });

    it('calls onUnauthorized when refresh fails', async () => {
      mockGetAccessToken.mockReturnValue('old-token');
      mockRefreshToken.mockResolvedValue(null);

      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'unauthorized', message: 'Token expired' }),
      });

      await expect(apiClient.get('/test')).rejects.toThrow(ApiClientError);
      expect(mockRefreshToken).toHaveBeenCalledTimes(1);
      expect(mockOnUnauthorized).toHaveBeenCalledTimes(1);
    });

    it('does not retry when skipRetry is true', async () => {
      mockGetAccessToken.mockReturnValue('token');

      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'unauthorized', message: 'Token expired' }),
      });

      await expect(apiClient.get('/test', { skipRetry: true })).rejects.toThrow(ApiClientError);
      expect(mockRefreshToken).not.toHaveBeenCalled();
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });

    it('deduplicates concurrent refresh requests', async () => {
      mockGetAccessToken.mockReturnValue('old-token');
      mockRefreshToken.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve('new-token'), 100))
      );

      // All requests return 401
      global.fetch = vi
        .fn()
        .mockResolvedValue({
          ok: false,
          status: 401,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'unauthorized', message: 'Token expired' }),
        })
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'unauthorized', message: 'Token expired' }),
        })
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'unauthorized', message: 'Token expired' }),
        })
        // Then all retries succeed
        .mockResolvedValue({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ data: 'success' }),
        });

      // Make 3 concurrent requests
      const requests = [apiClient.get('/test1'), apiClient.get('/test2'), apiClient.get('/test3')];

      await Promise.all(requests);

      // Refresh should only be called once
      expect(mockRefreshToken).toHaveBeenCalledTimes(1);
    });

    it('throws ApiClientError on non-401 error responses', async () => {
      mockGetAccessToken.mockReturnValue('token');

      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'not_found', message: 'Resource not found' }),
      });

      await expect(apiClient.get('/test')).rejects.toThrow(ApiClientError);
      await expect(apiClient.get('/test')).rejects.toMatchObject({
        status: 404,
        code: 'not_found',
        message: 'Resource not found',
      });
    });

    it('returns empty object for 204 No Content', async () => {
      mockGetAccessToken.mockReturnValue('token');

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
        headers: new Headers(),
      });

      const result = await apiClient.delete('/test');
      expect(result).toEqual({});
    });
  });

  describe('convenience methods', () => {
    beforeEach(() => {
      mockGetAccessToken.mockReturnValue('token');
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'success' }),
      });
    });

    it('get method makes GET request', async () => {
      await apiClient.get('/test');

      expect(global.fetch).toHaveBeenCalledWith(
        `${mockBaseURL}/test`,
        expect.objectContaining({ method: 'GET' })
      );
    });

    it('post method makes POST request with JSON body', async () => {
      const data = { name: 'test' };
      await apiClient.post('/test', data);

      expect(global.fetch).toHaveBeenCalledWith(
        `${mockBaseURL}/test`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(data),
        })
      );

      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const headers = call[1].headers as Headers;
      expect(headers.get('Content-Type')).toBe('application/json');
    });

    it('put method makes PUT request with JSON body', async () => {
      const data = { name: 'test' };
      await apiClient.put('/test', data);

      expect(global.fetch).toHaveBeenCalledWith(
        `${mockBaseURL}/test`,
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(data),
        })
      );
    });

    it('patch method makes PATCH request with JSON body', async () => {
      const data = { name: 'test' };
      await apiClient.patch('/test', data);

      expect(global.fetch).toHaveBeenCalledWith(
        `${mockBaseURL}/test`,
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify(data),
        })
      );
    });

    it('delete method makes DELETE request', async () => {
      await apiClient.delete('/test');

      expect(global.fetch).toHaveBeenCalledWith(
        `${mockBaseURL}/test`,
        expect.objectContaining({ method: 'DELETE' })
      );
    });
  });

  describe('retry logic', () => {
    beforeEach(() => {
      mockGetAccessToken.mockReturnValue('token');
    });

    it('retries GET requests on 5xx errors with exponential backoff', async () => {
      vi.useFakeTimers();

      global.fetch = vi
        .fn()
        .mockResolvedValueOnce({
          ok: false,
          status: 503,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'service_unavailable', message: 'Service unavailable' }),
        })
        .mockResolvedValueOnce({
          ok: false,
          status: 502,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'bad_gateway', message: 'Bad gateway' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ data: 'success' }),
        });

      const requestPromise = apiClient.get<{ data: string }>('/test');

      // Run all timers (delays)
      await vi.runAllTimersAsync();

      const result = await requestPromise;

      expect(global.fetch).toHaveBeenCalledTimes(3);
      expect(result).toEqual({ data: 'success' });

      vi.useRealTimers();
    });

    it('does not retry POST requests on 5xx errors', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'internal_error', message: 'Internal server error' }),
      });

      await expect(apiClient.post('/test', { data: 'test' })).rejects.toThrow(ApiClientError);
      expect(global.fetch).toHaveBeenCalledTimes(1); // No retries
    });

    it('does not retry PATCH requests on 5xx errors', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'internal_error', message: 'Internal server error' }),
      });

      await expect(apiClient.patch('/test', { data: 'test' })).rejects.toThrow(ApiClientError);
      expect(global.fetch).toHaveBeenCalledTimes(1); // No retries
    });

    it('retries PUT and DELETE requests on 5xx errors', async () => {
      vi.useFakeTimers();

      // Mock for PUT
      global.fetch = vi
        .fn()
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'internal_error', message: 'Internal error' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ data: 'success' }),
        });

      const putPromise = apiClient.put('/test', { data: 'test' });
      await vi.runAllTimersAsync();
      await putPromise;

      expect(global.fetch).toHaveBeenCalledTimes(2);

      // Reset and test DELETE
      vi.clearAllMocks();
      global.fetch = vi
        .fn()
        .mockResolvedValueOnce({
          ok: false,
          status: 503,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'service_unavailable', message: 'Service unavailable' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 204,
          headers: new Headers(),
        });

      const deletePromise = apiClient.delete('/test');
      await vi.runAllTimersAsync();
      await deletePromise;

      expect(global.fetch).toHaveBeenCalledTimes(2);

      vi.useRealTimers();
    });

    it('stops retrying after max attempts and includes retry count in error', async () => {
      vi.useFakeTimers();

      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'internal_error', message: 'Internal error' }),
      });

      const requestPromise = apiClient.get('/test');

      // Run timers and await the promise rejection together
      const [result] = await Promise.allSettled([requestPromise, vi.runAllTimersAsync()]);

      expect(result.status).toBe('rejected');
      if (result.status === 'rejected') {
        expect(result.reason).toMatchObject({
          status: 500,
          code: 'internal_error',
          retryCount: 2, // 0, 1, 2 = 3 attempts
        });
      }

      expect(global.fetch).toHaveBeenCalledTimes(3); // Max 3 attempts

      vi.useRealTimers();
    });

    it('does not retry when skipAutoRetry is true', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'internal_error', message: 'Internal error' }),
      });

      await expect(apiClient.get('/test', { skipAutoRetry: true })).rejects.toThrow(ApiClientError);
      expect(global.fetch).toHaveBeenCalledTimes(1); // No retries
    });

    it('retries on network errors (status 0)', async () => {
      vi.useFakeTimers();

      global.fetch = vi
        .fn()
        .mockRejectedValueOnce(new Error('Network failure'))
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ data: 'success' }),
        });

      const requestPromise = apiClient.get('/test');
      await vi.runAllTimersAsync();
      const result = await requestPromise;

      expect(result).toEqual({ data: 'success' });
      expect(global.fetch).toHaveBeenCalledTimes(2);

      vi.useRealTimers();
    });
  });

  describe('timeout support', () => {
    beforeEach(() => {
      mockGetAccessToken.mockReturnValue('token');
    });

    it('aborts request after timeout', async () => {
      // Use real timers and a very short timeout for testing
      global.fetch = vi.fn().mockImplementation(
        (_, options: RequestInit) =>
          new Promise((resolve, reject) => {
            // Simulate AbortError when signal is aborted
            options.signal?.addEventListener('abort', () => {
              const error = new Error('The operation was aborted');
              error.name = 'AbortError';
              reject(error);
            });
            // Never resolve this promise - it will timeout
          })
      );

      await expect(apiClient.get('/test', { timeout: 100 })).rejects.toMatchObject({
        status: 0,
        code: 'timeout',
        message: expect.stringContaining('100ms'),
      });
    });

    it('completes request before timeout expires', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'success' }),
      });

      const result = await apiClient.get('/test', { timeout: 5000 });
      expect(result).toEqual({ data: 'success' });
    });
  });

  describe('telemetry', () => {
    let mockOnTelemetry: ReturnType<typeof vi.fn>;

    beforeEach(() => {
      mockGetAccessToken.mockReturnValue('token');
      mockOnTelemetry = vi.fn();

      // Reinitialize with telemetry callback
      apiClient.initialize({
        baseURL: mockBaseURL,
        getAccessToken: mockGetAccessToken,
        refreshToken: mockRefreshToken,
        onUnauthorized: mockOnUnauthorized,
        onTelemetry: mockOnTelemetry,
      });
    });

    it('emits api_request and api_response events on success', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'success' }),
      });

      await apiClient.get('/test');

      expect(mockOnTelemetry).toHaveBeenCalledTimes(2);

      // First call: request event
      expect(mockOnTelemetry).toHaveBeenNthCalledWith(1, {
        type: 'api_request',
        method: 'GET',
        endpoint: '/test',
        timestamp: expect.any(Number),
      });

      // Second call: response event
      expect(mockOnTelemetry).toHaveBeenNthCalledWith(2, {
        type: 'api_response',
        method: 'GET',
        endpoint: '/test',
        status: 200,
        duration: expect.any(Number),
        retryCount: 0,
        timestamp: expect.any(Number),
      });
    });

    it('emits api_response event with retry count on retries', async () => {
      vi.useFakeTimers();

      global.fetch = vi
        .fn()
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ code: 'internal_error', message: 'Internal error' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ data: 'success' }),
        });

      const requestPromise = apiClient.get('/test');
      await vi.runAllTimersAsync();
      await requestPromise;

      // Should emit: 1 request + 1 response (with retry count)
      expect(mockOnTelemetry).toHaveBeenCalledTimes(2);
      expect(mockOnTelemetry).toHaveBeenNthCalledWith(2, {
        type: 'api_response',
        method: 'GET',
        endpoint: '/test',
        status: 200,
        duration: expect.any(Number),
        retryCount: 1, // Second attempt succeeded
        timestamp: expect.any(Number),
      });

      vi.useRealTimers();
    });

    it('emits api_response event on error', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ code: 'not_found', message: 'Not found' }),
      });

      await expect(apiClient.get('/test')).rejects.toThrow(ApiClientError);

      expect(mockOnTelemetry).toHaveBeenCalledTimes(2);
      expect(mockOnTelemetry).toHaveBeenNthCalledWith(2, {
        type: 'api_response',
        method: 'GET',
        endpoint: '/test',
        status: 404,
        duration: expect.any(Number),
        retryCount: 0,
        timestamp: expect.any(Number),
      });
    });

    it('does not break request if telemetry callback throws', async () => {
      mockOnTelemetry.mockImplementation(() => {
        throw new Error('Telemetry error');
      });

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'success' }),
      });

      // Request should still succeed despite telemetry error
      const result = await apiClient.get('/test');
      expect(result).toEqual({ data: 'success' });
    });
  });
});
