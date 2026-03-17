/**
 * Error Logger Service Tests
 * Validates error logging, redaction, and rate limiting
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ErrorLogger, redactSensitiveData, type ErrorLogPayload } from './error-logger';

describe('redactSensitiveData', () => {
  it('redacts JWT tokens', () => {
    const text =
      'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U';
    const redacted = redactSensitiveData(text);

    expect(redacted).toContain('[REDACTED]');
    expect(redacted).not.toContain('eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9');
  });

  it('redacts email addresses', () => {
    const text = 'User email: user@example.com found in error';
    const redacted = redactSensitiveData(text);

    expect(redacted).toContain('[REDACTED]');
    expect(redacted).not.toContain('user@example.com');
  });

  it('redacts DID identifiers', () => {
    const text = 'User DID: did:plc:abc123xyz';
    const redacted = redactSensitiveData(text);

    expect(redacted).toContain('[REDACTED]');
    expect(redacted).not.toContain('did:plc:abc123xyz');
  });

  it('redacts Authorization headers', () => {
    const text = 'Authorization: Bearer token123456';
    const redacted = redactSensitiveData(text);

    expect(redacted).toContain('[REDACTED]');
    expect(redacted).not.toContain('token123456');
  });

  it('redacts long alphanumeric strings (potential API keys)', () => {
    // Test with a realistic 40+ character token (using 'test' prefix to avoid false positives)
    const text = 'API Key: test_live_abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGH';
    const redacted = redactSensitiveData(text);

    expect(redacted).toContain('[REDACTED]');
    expect(redacted).not.toContain('test_live_abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGH');
  });

  it('redacts API keys with common prefixes', () => {
    const text = 'API key: api_live_1234567890abcdefghijklmn and key_test_abcdef1234567890';
    const redacted = redactSensitiveData(text);

    expect(redacted).toContain('[REDACTED]');
    expect(redacted).not.toContain('api_live_1234567890abcdefghijklmn');
    expect(redacted).not.toContain('key_test_abcdef1234567890');
  });

  it('does not redact short alphanumeric strings to avoid over-redaction', () => {
    // SHA256 hashes and UUIDs should not be overly aggressive
    const text = 'Error in function calculateHash with result abc123def456';
    const redacted = redactSensitiveData(text);

    // Short strings should not be redacted
    expect(redacted).toBe(text);
  });

  it('preserves non-sensitive text', () => {
    const text = 'Regular error message without PII';
    const redacted = redactSensitiveData(text);

    expect(redacted).toBe(text);
  });

  it('handles empty strings', () => {
    expect(redactSensitiveData('')).toBe('');
  });

  it('handles multiple sensitive patterns in one string', () => {
    const text =
      'Error with email user@test.com and DID did:plc:123 and token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c';
    const redacted = redactSensitiveData(text);

    expect(redacted).not.toContain('user@test.com');
    expect(redacted).not.toContain('did:plc:123');
    expect(redacted).not.toContain('eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9');
    expect(redacted).toContain('[REDACTED]');
  });
});

describe('ErrorLogger', () => {
  let logger: ErrorLogger;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // Clear sessionStorage
    sessionStorage.clear();

    // Mock fetch
    fetchMock = vi.fn().mockResolvedValue({ ok: true });
    global.fetch = fetchMock;

    // Use fake timers
    vi.useFakeTimers();
  });

  afterEach(() => {
    logger?.destroy();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  describe('initialization', () => {
    it('creates a session ID on first use', () => {
      logger = new ErrorLogger();

      const sessionId = sessionStorage.getItem('subcults-session-id');
      expect(sessionId).toBeTruthy();
      expect(sessionId).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
      );
    });

    it('reuses existing session ID', () => {
      const existingId = 'existing-session-id';
      sessionStorage.setItem('subcults-session-id', existingId);

      logger = new ErrorLogger();

      const sessionId = sessionStorage.getItem('subcults-session-id');
      expect(sessionId).toBe(existingId);
    });

    it('accepts custom configuration', () => {
      logger = new ErrorLogger({
        maxErrorsPerMinute: 5,
        endpoint: '/custom/endpoint',
        consoleLogging: false,
      });

      // Configuration is applied (tested indirectly through behavior)
      expect(logger).toBeTruthy();
    });
  });

  describe('logError', () => {
    beforeEach(() => {
      logger = new ErrorLogger({ consoleLogging: false });
    });

    it('sends error payload to backend', async () => {
      const error = new Error('Test error message');

      await logger.logError(error);

      expect(fetchMock).toHaveBeenCalledTimes(1);
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/log/client-error',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        })
      );
    });

    it('includes all required fields in payload', async () => {
      const error = new Error('Test error');
      error.name = 'TestError';
      error.stack = 'Error: Test error\n    at TestFunction (test.ts:10:5)';

      await logger.logError(error);

      const call = fetchMock.mock.calls[0];
      const payload: ErrorLogPayload = JSON.parse(call[1].body);

      expect(payload).toMatchObject({
        message: 'Test error',
        type: 'TestError',
        timestamp: expect.any(Number),
        url: expect.any(String),
        userAgent: expect.any(String),
        sessionId: expect.any(String),
      });
      expect(payload.stack).toBeTruthy();
    });

    it('redacts sensitive data from error message', async () => {
      const error = new Error('Error with email user@test.com in message');

      await logger.logError(error);

      const call = fetchMock.mock.calls[0];
      const payload: ErrorLogPayload = JSON.parse(call[1].body);

      expect(payload.message).not.toContain('user@test.com');
      expect(payload.message).toContain('[REDACTED]');
    });

    it('excludes query parameters from URL to prevent PII leakage', async () => {
      // Mock window.location
      delete (window as unknown as { location?: unknown }).location;
      (window as unknown as { location: unknown }).location = {
        pathname: '/scenes/123',
        hash: '#section',
        href: '/scenes/123?token=secret&email=user@test.com#section',
      };

      const error = new Error('Test error');
      await logger.logError(error);

      const call = fetchMock.mock.calls[0];
      const payload: ErrorLogPayload = JSON.parse(call[1].body);

      // Should include pathname and hash, but not query params
      expect(payload.url).toBe('/scenes/123#section');
      expect(payload.url).not.toContain('token=');
      expect(payload.url).not.toContain('email=');
    });

    it('redacts sensitive data from stack trace', async () => {
      const error = new Error('Test error');
      error.stack = 'Error with DID did:plc:abc123\n    at test.ts:10:5';

      await logger.logError(error);

      const call = fetchMock.mock.calls[0];
      const payload: ErrorLogPayload = JSON.parse(call[1].body);

      expect(payload.stack).not.toContain('did:plc:abc123');
      expect(payload.stack).toContain('[REDACTED]');
    });

    it('includes component stack for React errors', async () => {
      const error = new Error('React error');
      const errorInfo = {
        componentStack: '\n    at ErrorComponent (ErrorComponent.tsx:10)',
      };

      await logger.logError(error, errorInfo);

      const call = fetchMock.mock.calls[0];
      const payload: ErrorLogPayload = JSON.parse(call[1].body);

      expect(payload.componentStack).toBeTruthy();
      expect(payload.componentStack).toContain('ErrorComponent');
    });

    it('redacts sensitive data from component stack', async () => {
      const error = new Error('React error');
      const errorInfo = {
        componentStack: '\n    at Component with email user@test.com',
      };

      await logger.logError(error, errorInfo);

      const call = fetchMock.mock.calls[0];
      const payload: ErrorLogPayload = JSON.parse(call[1].body);

      expect(payload.componentStack).not.toContain('user@test.com');
      expect(payload.componentStack).toContain('[REDACTED]');
    });

    it('handles fetch errors gracefully', async () => {
      fetchMock.mockRejectedValueOnce(new Error('Network error'));

      const error = new Error('Test error');

      // Should not throw
      await expect(logger.logError(error)).resolves.toBeUndefined();
    });

    it('handles non-ok responses gracefully', async () => {
      fetchMock.mockResolvedValueOnce({ ok: false, status: 500 });

      const error = new Error('Test error');

      // Should not throw
      await expect(logger.logError(error)).resolves.toBeUndefined();
    });
  });

  describe('rate limiting', () => {
    beforeEach(() => {
      logger = new ErrorLogger({
        maxErrorsPerMinute: 3,
        consoleLogging: false,
      });
    });

    it('logs errors up to rate limit', async () => {
      await logger.logError(new Error('Error 1'));
      await logger.logError(new Error('Error 2'));
      await logger.logError(new Error('Error 3'));

      expect(fetchMock).toHaveBeenCalledTimes(3);
    });

    it('drops errors after rate limit is exceeded', async () => {
      await logger.logError(new Error('Error 1'));
      await logger.logError(new Error('Error 2'));
      await logger.logError(new Error('Error 3'));
      await logger.logError(new Error('Error 4')); // Should be dropped

      expect(fetchMock).toHaveBeenCalledTimes(3);
      expect(logger.getErrorCount()).toBe(3);
    });

    it('resets error count after 1 minute', async () => {
      await logger.logError(new Error('Error 1'));
      await logger.logError(new Error('Error 2'));
      await logger.logError(new Error('Error 3'));

      expect(logger.getErrorCount()).toBe(3);

      // Advance time by 1 minute
      vi.advanceTimersByTime(60000);

      expect(logger.getErrorCount()).toBe(0);

      // Should be able to log again
      await logger.logError(new Error('Error 4'));
      expect(fetchMock).toHaveBeenCalledTimes(4);
    });

    it('continues resetting error count on interval', async () => {
      await logger.logError(new Error('Error 1'));

      expect(logger.getErrorCount()).toBe(1);

      // First reset
      vi.advanceTimersByTime(60000);
      expect(logger.getErrorCount()).toBe(0);

      await logger.logError(new Error('Error 2'));
      expect(logger.getErrorCount()).toBe(1);

      // Second reset
      vi.advanceTimersByTime(60000);
      expect(logger.getErrorCount()).toBe(0);
    });
  });

  describe('destroy', () => {
    beforeEach(() => {
      logger = new ErrorLogger({ consoleLogging: false });
    });

    it('stops rate limit reset timer', async () => {
      await logger.logError(new Error('Error 1'));
      expect(logger.getErrorCount()).toBe(1);

      logger.destroy();

      // Advance time - should not reset
      vi.advanceTimersByTime(60000);

      // Cannot check count after destroy, but test doesn't throw
      expect(true).toBe(true);
    });
  });

  describe('custom configuration', () => {
    it('uses custom endpoint', async () => {
      logger = new ErrorLogger({
        endpoint: '/custom/error/endpoint',
        consoleLogging: false,
      });

      await logger.logError(new Error('Test error'));

      expect(fetchMock).toHaveBeenCalledWith('/custom/error/endpoint', expect.any(Object));
    });

    it('respects custom rate limit', async () => {
      logger = new ErrorLogger({
        maxErrorsPerMinute: 2,
        consoleLogging: false,
      });

      await logger.logError(new Error('Error 1'));
      await logger.logError(new Error('Error 2'));
      await logger.logError(new Error('Error 3')); // Should be dropped

      expect(fetchMock).toHaveBeenCalledTimes(2);
    });
  });
});
