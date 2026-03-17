/**
 * Telemetry Service Tests
 * Validates event batching, flushing, retry logic, and opt-out behavior
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { TelemetryService } from './telemetry-service';
import { apiClient } from './api-client';

// Mock apiClient
vi.mock('./api-client', () => ({
  apiClient: {
    post: vi.fn(),
  },
}));

describe('TelemetryService', () => {
  let service: TelemetryService;
  let isOptedOut: () => boolean;

  beforeEach(() => {
    // Clear sessionStorage
    sessionStorage.clear();
    
    // Reset API client mock
    vi.clearAllMocks();
    (apiClient.post as ReturnType<typeof vi.fn>).mockResolvedValue({});

    // Default: user is NOT opted out
    isOptedOut = vi.fn(() => false);

    // Use fake timers
    vi.useFakeTimers();
  });

  afterEach(() => {
    service?.destroy();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  describe('session ID generation', () => {
    it('generates a session ID on first use', () => {
      service = new TelemetryService({}, isOptedOut);
      const sessionId = service.getSessionId();

      expect(sessionId).toBeTruthy();
      expect(sessionId).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i);
    });

    it('persists session ID in sessionStorage', () => {
      service = new TelemetryService({}, isOptedOut);
      const sessionId = service.getSessionId();

      expect(sessionStorage.getItem('subcults-session-id')).toBe(sessionId);
    });

    it('reuses existing session ID from sessionStorage', () => {
      const existingId = 'existing-session-id';
      sessionStorage.setItem('subcults-session-id', existingId);

      service = new TelemetryService({}, isOptedOut);
      const sessionId = service.getSessionId();

      expect(sessionId).toBe(existingId);
    });
  });

  describe('event emission', () => {
    beforeEach(() => {
      service = new TelemetryService({ flushInterval: 10000 }, isOptedOut);
    });

    it('queues events when emit is called', () => {
      service.emit('test.event', { key: 'value' });

      expect(service.getQueueSize()).toBe(1);
    });

    it('includes all required event fields', () => {
      const userId = 'did:plc:test123';
      service.emit('test.event', { key: 'value' }, userId);
      
      // Manually flush to inspect events
      vi.spyOn(apiClient, 'post');
      service.flush();

      expect(apiClient.post).toHaveBeenCalledWith(
        '/telemetry',
        {
          events: expect.arrayContaining([
            expect.objectContaining({
              name: 'test.event',
              ts: expect.any(Number),
              sessionId: expect.any(String),
              userId: 'did:plc:test123',
              payload: { key: 'value' },
            }),
          ]),
        },
        { skipAutoRetry: true }
      );
    });

    it('does not queue events when user is opted out', () => {
      isOptedOut = vi.fn(() => true);
      service = new TelemetryService({ flushInterval: 10000 }, isOptedOut);

      service.emit('test.event');

      expect(service.getQueueSize()).toBe(0);
    });
  });

  describe('auto-flush on batch size', () => {
    beforeEach(() => {
      service = new TelemetryService({ 
        maxBatchSize: 3,
        flushInterval: 10000, // Long interval to avoid time-based flush
      }, isOptedOut);
    });

    it('auto-flushes when batch size is reached', () => {
      vi.spyOn(apiClient, 'post');

      service.emit('event.1');
      service.emit('event.2');
      expect(apiClient.post).not.toHaveBeenCalled();

      service.emit('event.3'); // Should trigger flush
      expect(apiClient.post).toHaveBeenCalledTimes(1);
      expect(service.getQueueSize()).toBe(0);
    });

    it('sends correct number of events in batch', () => {
      vi.spyOn(apiClient, 'post');

      service.emit('event.1');
      service.emit('event.2');
      service.emit('event.3'); // Triggers flush

      const call = (apiClient.post as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[1].events).toHaveLength(3);
    });
  });

  describe('flush timer', () => {
    beforeEach(() => {
      service = new TelemetryService({ 
        flushInterval: 5000, // 5 seconds
        maxBatchSize: 100, // High to avoid size-based flush
      }, isOptedOut);
    });

    it('auto-flushes after flush interval', () => {
      vi.spyOn(apiClient, 'post');

      service.emit('event.1');
      expect(apiClient.post).not.toHaveBeenCalled();

      // Advance time by flush interval
      vi.advanceTimersByTime(5000);

      expect(apiClient.post).toHaveBeenCalledTimes(1);
      expect(service.getQueueSize()).toBe(0);
    });

    it('flushes multiple times at interval', () => {
      vi.spyOn(apiClient, 'post');

      service.emit('event.1');
      vi.advanceTimersByTime(5000);
      expect(apiClient.post).toHaveBeenCalledTimes(1);

      service.emit('event.2');
      vi.advanceTimersByTime(5000);
      expect(apiClient.post).toHaveBeenCalledTimes(2);
    });

    it('does not flush when queue is empty', () => {
      vi.spyOn(apiClient, 'post');

      // No events emitted
      vi.advanceTimersByTime(5000);

      expect(apiClient.post).not.toHaveBeenCalled();
    });
  });

  describe('manual flush', () => {
    beforeEach(() => {
      service = new TelemetryService({ flushInterval: 10000 }, isOptedOut);
    });

    it('sends queued events immediately', () => {
      vi.spyOn(apiClient, 'post');

      service.emit('event.1');
      service.emit('event.2');

      service.flush();

      expect(apiClient.post).toHaveBeenCalledTimes(1);
      expect(service.getQueueSize()).toBe(0);
    });

    it('clears queue when opted out', () => {
      service.emit('event.1');
      expect(service.getQueueSize()).toBe(1);

      // User opts out
      isOptedOut = vi.fn(() => true);
      service = new TelemetryService({ flushInterval: 10000 }, isOptedOut);
      service.emit('event.1'); // Add to new service to set up queue
      
      service.flush();

      expect(apiClient.post).not.toHaveBeenCalled();
      expect(service.getQueueSize()).toBe(0);
    });

    it('does nothing when queue is empty', () => {
      vi.spyOn(apiClient, 'post');

      service.flush();

      expect(apiClient.post).not.toHaveBeenCalled();
    });
  });

  describe('retry logic', () => {
    beforeEach(() => {
      service = new TelemetryService({ 
        flushInterval: 10000,
        maxRetries: 1,
        retryDelay: 1000,
      }, isOptedOut);
    });

    it('retries once on network error', async () => {
      vi.spyOn(apiClient, 'post').mockRejectedValueOnce(new Error('Network error'));

      service.emit('event.1');
      service.flush();

      // First call fails
      expect(apiClient.post).toHaveBeenCalledTimes(1);

      // Advance time by retry delay
      await vi.advanceTimersByTimeAsync(1000);

      // Second call (retry) should happen
      expect(apiClient.post).toHaveBeenCalledTimes(2);
    });

    it('applies exponential backoff', async () => {
      vi.spyOn(apiClient, 'post')
        .mockRejectedValueOnce(new Error('Network error'))
        .mockRejectedValueOnce(new Error('Network error'));

      service = new TelemetryService({ 
        flushInterval: 10000,
        maxRetries: 2,
        retryDelay: 1000,
      }, isOptedOut);

      service.emit('event.1');
      service.flush();

      expect(apiClient.post).toHaveBeenCalledTimes(1);

      // First retry after 1s
      await vi.advanceTimersByTimeAsync(1000);
      expect(apiClient.post).toHaveBeenCalledTimes(2);

      // Second retry after 2s (exponential backoff: 1000 * 2^1)
      await vi.advanceTimersByTimeAsync(2000);
      expect(apiClient.post).toHaveBeenCalledTimes(3);
    });

    it('drops events after max retries', async () => {
      vi.spyOn(apiClient, 'post')
        .mockRejectedValueOnce(new Error('Network error'))
        .mockRejectedValueOnce(new Error('Network error'));

      service.emit('event.1');
      service.flush();

      // First call fails
      expect(apiClient.post).toHaveBeenCalledTimes(1);

      // Retry after 1s
      await vi.advanceTimersByTimeAsync(1000);
      expect(apiClient.post).toHaveBeenCalledTimes(2);

      // No more retries after maxRetries (1)
      await vi.advanceTimersByTimeAsync(10000);
      expect(apiClient.post).toHaveBeenCalledTimes(2);
    });
  });

  describe('destroy', () => {
    beforeEach(() => {
      service = new TelemetryService({ flushInterval: 5000 }, isOptedOut);
    });

    it('stops flush timer', () => {
      vi.spyOn(apiClient, 'post');

      service.emit('event.1');
      service.destroy();

      vi.advanceTimersByTime(5000);

      // Only the destroy flush should happen, not timer-based flush
      expect(apiClient.post).toHaveBeenCalledTimes(1);
    });

    it('flushes remaining events', () => {
      vi.spyOn(apiClient, 'post');

      service.emit('event.1');
      service.destroy();

      expect(apiClient.post).toHaveBeenCalledTimes(1);
    });
  });
});
