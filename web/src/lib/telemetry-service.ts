/**
 * Telemetry Service
 * Lightweight event bus for collecting structured analytics events
 * with batching, retry logic, and privacy opt-out support
 */

import type { TelemetryEvent } from '../types/telemetry';
import { apiClient } from './api-client';

/**
 * Configuration for telemetry service
 */
interface TelemetryConfig {
  /** Flush interval in milliseconds (default: 5000ms = 5s) */
  flushInterval: number;
  /** Maximum events before auto-flush (default: 20) */
  maxBatchSize: number;
  /** Maximum retry attempts on network errors (default: 1) */
  maxRetries: number;
  /** Base delay for retry backoff in milliseconds (default: 1000ms) */
  retryDelay: number;
}

/**
 * Default telemetry configuration
 */
const DEFAULT_CONFIG: TelemetryConfig = {
  flushInterval: 5000, // 5 seconds
  maxBatchSize: 20,
  maxRetries: 1,
  retryDelay: 1000, // 1 second
};

/**
 * Session ID storage key
 */
const SESSION_ID_KEY = 'subcults-session-id';

/**
 * Generate a UUID v4
 * Uses crypto.randomUUID() for better entropy, falls back to Math.random()
 */
function generateUUID(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  // Fallback for environments without crypto.randomUUID
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/**
 * Get or create session ID for this tab
 * Session ID persists for the browser tab session (until tab closes)
 */
function getSessionId(): string {
  // Use sessionStorage (persists across navigation, clears when tab closes)
  let sessionId = sessionStorage.getItem(SESSION_ID_KEY);
  
  if (!sessionId) {
    sessionId = generateUUID();
    sessionStorage.setItem(SESSION_ID_KEY, sessionId);
  }
  
  return sessionId;
}

/**
 * Telemetry service class
 */
class TelemetryService {
  private config: TelemetryConfig;
  private eventQueue: TelemetryEvent[] = [];
  private flushTimer: number | null = null;
  private sessionId: string;
  private isOptedOut: () => boolean;

  constructor(config: Partial<TelemetryConfig> = {}, isOptedOut: () => boolean = () => false) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.sessionId = getSessionId();
    this.isOptedOut = isOptedOut;
    this.startFlushTimer();
  }

  /**
   * Emit a telemetry event
   * @param name - Event name (use dot-notation: e.g., 'search.scene')
   * @param payload - Optional event-specific data (keep minimal)
   * @param userId - Optional user DID (only if authenticated)
   */
  emit(name: string, payload?: Record<string, unknown>, userId?: string): void {
    // Skip if user has opted out
    if (this.isOptedOut()) {
      return;
    }

    const event: TelemetryEvent = {
      name,
      ts: Date.now(),
      sessionId: this.sessionId,
      userId,
      payload,
    };

    this.eventQueue.push(event);

    // Auto-flush if batch size reached
    if (this.eventQueue.length >= this.config.maxBatchSize) {
      this.flush();
    }
  }

  /**
   * Manually flush all queued events
   */
  flush(): void {
    if (this.eventQueue.length === 0) {
      return;
    }

    // Skip if user has opted out
    if (this.isOptedOut()) {
      this.eventQueue = [];
      return;
    }

    // Take current queue and clear it
    const eventsToSend = [...this.eventQueue];
    this.eventQueue = [];

    // Send events with retry
    this.sendEvents(eventsToSend, 0);
  }

  /**
   * Send events to telemetry endpoint with retry logic
   */
  private async sendEvents(events: TelemetryEvent[], retryCount: number): Promise<void> {
    try {
      await apiClient.post('/telemetry', { events }, { skipAutoRetry: true });
    } catch (error) {
      // Retry on network errors if we haven't exceeded max retries
      if (retryCount < this.config.maxRetries) {
        const delay = this.config.retryDelay * Math.pow(2, retryCount);
        setTimeout(() => {
          this.sendEvents(events, retryCount + 1);
        }, delay);
      } else {
        // Drop events after max retries
        console.warn('[telemetry] Failed to send events after retries:', error);
      }
    }
  }

  /**
   * Start the periodic flush timer
   */
  private startFlushTimer(): void {
    this.flushTimer = window.setInterval(() => {
      this.flush();
    }, this.config.flushInterval);
  }

  /**
   * Stop the periodic flush timer
   */
  private stopFlushTimer(): void {
    if (this.flushTimer !== null) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    this.stopFlushTimer();
    this.flush(); // Final flush
  }

  /**
   * Get current queue size (for testing)
   */
  getQueueSize(): number {
    return this.eventQueue.length;
  }

  /**
   * Get session ID (for testing)
   */
  getSessionId(): string {
    return this.sessionId;
  }
}

/**
 * Get opt-out status from settings store
 * Uses dynamic import to avoid circular dependency
 */
function getOptOutStatus(): boolean {
  try {
    // Dynamic import to avoid module loading issues
    // In production, this will be replaced by the actual store value
    if (typeof window !== 'undefined' && window.localStorage) {
      const settings = localStorage.getItem('subcults-settings');
      if (settings) {
        const parsed = JSON.parse(settings);
        return parsed.telemetryOptOut ?? false;
      }
    }
  } catch (error) {
    // Fail gracefully
    console.warn('[telemetry] Failed to read opt-out status:', error);
  }
  return false;
}

// Export singleton instance with opt-out callback
export const telemetryService = new TelemetryService({}, getOptOutStatus);

// Export class for testing
export { TelemetryService };
