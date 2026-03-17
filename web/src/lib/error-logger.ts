/**
 * Error Logger Service
 * Captures and logs client-side errors with PII redaction and rate limiting
 */

import { sessionReplay, type ReplayEvent } from './session-replay';

/**
 * Error payload structure for logging endpoint
 */
export interface ErrorLogPayload {
  /** Error message (redacted) */
  message: string;
  /** Stack trace (redacted) */
  stack?: string;
  /** Error type/name */
  type: string;
  /** Timestamp when error occurred */
  timestamp: number;
  /** URL where error occurred */
  url: string;
  /** User agent string */
  userAgent: string;
  /** Component stack (React errors only, redacted) */
  componentStack?: string;
  /** Session ID for grouping related errors */
  sessionId: string;
  /** Session replay events (only if opted in and available) */
  replayEvents?: ReplayEvent[];
}

/**
 * Configuration for error logger
 */
interface ErrorLoggerConfig {
  /** Maximum errors to log per minute (default: 10) */
  maxErrorsPerMinute: number;
  /** Endpoint to send errors to (default: /api/log/client-error) */
  endpoint: string;
  /** Whether to log to console in development (default: true) */
  consoleLogging: boolean;
}

/**
 * Default configuration
 */
const DEFAULT_CONFIG: ErrorLoggerConfig = {
  maxErrorsPerMinute: 10,
  endpoint: '/api/log/client-error',
  consoleLogging: true,
};

/**
 * Sensitive patterns to redact from error messages and stack traces
 * These patterns match common PII and authentication tokens
 */
const SENSITIVE_PATTERNS = [
  // JWT tokens (eyJ... format)
  /eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}/g,
  
  // Email addresses
  /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g,
  
  // DID identifiers (did:plc:... or did:web:...)
  /did:[a-z]+:[a-zA-Z0-9._-]+/g,
  
  // Authorization headers
  /Authorization:\s*Bearer\s+[A-Za-z0-9._-]+/gi,
  
  // API keys / access tokens
  // 1) Keys with common prefixes (Stripe, generic "api_", "key_", etc.)
  /(?:sk|pk|api|key|rk|r2|lk)_[A-Za-z0-9_-]{16,}/gi,
  // 2) Generic high-entropy tokens (more conservative than 32+ chars, uses word boundaries)
  /\b[A-Za-z0-9_-]{40,}\b/g,
];

/**
 * Redact sensitive information from a string
 * @param text - Text to redact
 * @returns Redacted text with sensitive data replaced
 */
export function redactSensitiveData(text: string): string {
  if (!text) return text;
  
  let redacted = text;
  
  // Apply all redaction patterns
  for (const pattern of SENSITIVE_PATTERNS) {
    redacted = redacted.replace(pattern, '[REDACTED]');
  }
  
  return redacted;
}

/**
 * Get or create session ID for error grouping
 * Reuses telemetry session ID if available
 */
function getSessionId(): string {
  const SESSION_ID_KEY = 'subcults-session-id';
  
  try {
    let sessionId = sessionStorage.getItem(SESSION_ID_KEY);
    
    if (!sessionId) {
      // Generate UUID v4
      if (typeof crypto !== 'undefined' && crypto.randomUUID) {
        sessionId = crypto.randomUUID();
      } else {
        // Fallback
        sessionId = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
          const r = (Math.random() * 16) | 0;
          const v = c === 'x' ? r : (r & 0x3) | 0x8;
          return v.toString(16);
        });
      }
      
      sessionStorage.setItem(SESSION_ID_KEY, sessionId);
    }
    
    return sessionId;
  } catch {
    // Fallback if sessionStorage is unavailable
    return 'unknown-session';
  }
}

/**
 * Error Logger Service
 * Handles client-side error logging with redaction and rate limiting
 */
class ErrorLogger {
  private config: ErrorLoggerConfig;
  private errorCount = 0;
  private resetTimer: number | null = null;
  private sessionId: string;

  constructor(config: Partial<ErrorLoggerConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.sessionId = getSessionId();
    this.startRateLimitReset();
  }

  /**
   * Log an error to the backend
   * @param error - Error object
   * @param errorInfo - Additional error info (for React errors)
   */
  async logError(error: Error, errorInfo?: { componentStack?: string }): Promise<void> {
    // Check rate limit
    if (this.errorCount >= this.config.maxErrorsPerMinute) {
      if (this.config.consoleLogging && import.meta.env.DEV) {
        console.warn('[ErrorLogger] Rate limit exceeded, dropping error:', error.message);
      }
      return;
    }

    // Increment error count
    this.errorCount++;

    // Get session replay events if available (user must be opted in)
    const replayEvents = sessionReplay.getAndClearBuffer();

    // Build error payload with redaction
    const payload: ErrorLogPayload = {
      message: redactSensitiveData(error.message),
      stack: error.stack ? redactSensitiveData(error.stack) : undefined,
      type: error.name,
      timestamp: Date.now(),
      url: window.location.pathname + window.location.hash,
      userAgent: navigator.userAgent,
      componentStack: errorInfo?.componentStack 
        ? redactSensitiveData(errorInfo.componentStack) 
        : undefined,
      sessionId: this.sessionId,
      replayEvents: replayEvents.length > 0 ? replayEvents : undefined,
    };

    // Log to console in development
    if (this.config.consoleLogging && import.meta.env.DEV) {
      console.error('[ErrorLogger] Logging error:', payload);
    }

    // Send to backend
    try {
      const response = await fetch(this.config.endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(`Error logging failed: ${response.status}`);
      }
    } catch (err) {
      // Don't throw - logging errors shouldn't crash the app
      if (this.config.consoleLogging && import.meta.env.DEV) {
        console.error('[ErrorLogger] Failed to send error log:', err);
      }
    }
  }

  /**
   * Start the rate limit reset timer
   * Resets error count every minute
   */
  private startRateLimitReset(): void {
    this.resetTimer = window.setInterval(() => {
      this.errorCount = 0;
    }, 60000); // 60 seconds
  }

  /**
   * Stop the rate limit reset timer
   */
  private stopRateLimitReset(): void {
    if (this.resetTimer !== null) {
      clearInterval(this.resetTimer);
      this.resetTimer = null;
    }
  }

  /**
   * Get current error count (for testing)
   */
  getErrorCount(): number {
    return this.errorCount;
  }

  /**
   * Reset error count (for testing)
   */
  resetErrorCount(): void {
    this.errorCount = 0;
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    this.stopRateLimitReset();
  }
}

// Export singleton instance
export const errorLogger = new ErrorLogger();

// Export class for testing
export { ErrorLogger };
