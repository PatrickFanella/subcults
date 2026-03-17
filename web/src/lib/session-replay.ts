/**
 * Session Replay Service
 * Minimal event recorder for capturing DOM mutations and user interactions
 * with opt-in privacy controls and performance-aware buffering
 */

/**
 * Replay event types
 */
const ReplayEventType = {
  DOMChange: 'dom_change',
  Click: 'click',
  Navigation: 'navigation',
  Scroll: 'scroll',
} as const;

export type ReplayEventType = typeof ReplayEventType[keyof typeof ReplayEventType];

/**
 * Replay event structure
 */
export interface ReplayEvent {
  /** Event type */
  type: ReplayEventType;
  /** Timestamp when event occurred */
  timestamp: number;
  /** Event-specific data (sanitized) */
  data: Record<string, unknown>;
}

/**
 * Configuration for session replay
 */
interface SessionReplayConfig {
  /** Maximum events to buffer before dropping old events (default: 100) */
  maxBufferSize: number;
  /** Sample rate for DOM mutations (0-1, default: 0.5 = 50%) */
  domMutationSampleRate: number;
  /** Sample rate for clicks (0-1, default: 1.0 = 100%) */
  clickSampleRate: number;
  /** Whether to enable performance monitoring (default: true) */
  performanceMonitoring: boolean;
}

/**
 * Default configuration
 */
const DEFAULT_CONFIG: SessionReplayConfig = {
  maxBufferSize: 100,
  domMutationSampleRate: 0.5, // Sample 50% of DOM mutations
  clickSampleRate: 1.0, // Capture all clicks
  performanceMonitoring: true,
};

/**
 * Sanitize element data to remove PII
 * Removes text content, values, and sensitive attributes
 */
function sanitizeElementData(element: Element): Record<string, unknown> {
  return {
    tagName: element.tagName.toLowerCase(),
    id: element.id || undefined,
    className: element.className || undefined,
    // Don't include text content, values, or other potentially sensitive data
  };
}

/**
 * Session Replay Service
 * Records user interactions and DOM changes in a ring buffer
 */
class SessionReplay {
  private config: SessionReplayConfig;
  private buffer: ReplayEvent[] = [];
  private isRecording = false;
  private mutationObserver: MutationObserver | null = null;
  private isOptedIn: () => boolean;

  constructor(config: Partial<SessionReplayConfig> = {}, isOptedIn: () => boolean = () => false) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.isOptedIn = isOptedIn;
  }

  /**
   * Start recording user interactions
   */
  start(): void {
    // Only record if user has opted in
    if (!this.isOptedIn()) {
      return;
    }

    if (this.isRecording) {
      return;
    }

    this.isRecording = true;

    // Set up event listeners
    this.setupClickListener();
    this.setupNavigationListener();
    this.setupScrollListener();
    this.setupMutationObserver();
  }

  /**
   * Stop recording
   */
  stop(): void {
    this.isRecording = false;

    // Remove event listeners
    document.removeEventListener('click', this.handleClick);
    window.removeEventListener('popstate', this.handleNavigation);
    window.removeEventListener('scroll', this.handleScroll as EventListener);

    // Disconnect mutation observer
    if (this.mutationObserver) {
      this.mutationObserver.disconnect();
      this.mutationObserver = null;
    }
  }

  /**
   * Set up click event listener
   */
  private setupClickListener(): void {
    document.addEventListener('click', this.handleClick);
  }

  /**
   * Handle click events
   */
  private handleClick = (event: MouseEvent): void => {
    if (!this.isRecording || !this.isOptedIn()) return;

    // Apply sampling
    if (Math.random() > this.config.clickSampleRate) {
      return;
    }

    const target = event.target as Element;
    this.addEvent({
      type: ReplayEventType.Click,
      timestamp: Date.now(),
      data: {
        element: sanitizeElementData(target),
        x: Math.round(event.clientX),
        y: Math.round(event.clientY),
      },
    });
  };

  /**
   * Set up navigation listener
   */
  private setupNavigationListener(): void {
    window.addEventListener('popstate', this.handleNavigation);
  }

  /**
   * Handle navigation events
   */
  private handleNavigation = (): void => {
    if (!this.isRecording || !this.isOptedIn()) return;

    this.addEvent({
      type: ReplayEventType.Navigation,
      timestamp: Date.now(),
      data: {
        url: window.location.pathname, // Only pathname, no query params
      },
    });
  };

  /**
   * Set up scroll listener with passive flag for performance
   */
  private setupScrollListener(): void {
    window.addEventListener('scroll', this.handleScroll, { passive: true });
  }

  /**
   * Handle scroll events (throttled via sampling)
   */
  private handleScroll = (): void => {
    if (!this.isRecording || !this.isOptedIn()) return;

    // Heavy sampling for scroll events to avoid overwhelming the buffer
    if (Math.random() > 0.1) { // 10% sample rate for scrolls
      return;
    }

    this.addEvent({
      type: ReplayEventType.Scroll,
      timestamp: Date.now(),
      data: {
        x: Math.round(window.scrollX),
        y: Math.round(window.scrollY),
      },
    });
  };

  /**
   * Set up mutation observer for DOM changes
   */
  private setupMutationObserver(): void {
    this.mutationObserver = new MutationObserver(this.handleMutations);
    
    this.mutationObserver.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeOldValue: false,
      characterData: false, // Don't track text changes (privacy)
    });
  }

  /**
   * Handle DOM mutations
   */
  private handleMutations = (mutations: MutationRecord[]): void => {
    if (!this.isRecording || !this.isOptedIn()) return;

    // Apply sampling to reduce buffer pressure
    if (Math.random() > this.config.domMutationSampleRate) {
      return;
    }

    // Check buffer size before processing (performance threshold)
    if (this.config.performanceMonitoring && this.buffer.length >= this.config.maxBufferSize * 0.9) {
      // Buffer is 90% full - early exit to preserve performance
      return;
    }

    const mutationData = mutations
      .slice(0, 5) // Limit to first 5 mutations to avoid large payloads
      .map((mutation) => ({
        type: mutation.type,
        target: mutation.target instanceof Element 
          ? sanitizeElementData(mutation.target) 
          : { nodeType: mutation.target.nodeType },
      }));

    if (mutationData.length > 0) {
      this.addEvent({
        type: ReplayEventType.DOMChange,
        timestamp: Date.now(),
        data: {
          mutations: mutationData,
          count: mutations.length,
        },
      });
    }
  };

  /**
   * Add an event to the buffer (ring buffer)
   */
  private addEvent(event: ReplayEvent): void {
    // Early exit if buffer is at max capacity
    if (this.buffer.length >= this.config.maxBufferSize) {
      // Remove oldest event (ring buffer behavior)
      this.buffer.shift();
    }

    this.buffer.push(event);
  }

  /**
   * Get all buffered events and clear the buffer
   */
  getAndClearBuffer(): ReplayEvent[] {
    if (!this.isOptedIn()) {
      // If user has opted out, clear buffer without returning
      this.buffer = [];
      return [];
    }

    const events = [...this.buffer];
    this.buffer = [];
    return events;
  }

  /**
   * Get current buffer size (for testing and monitoring)
   */
  getBufferSize(): number {
    return this.buffer.length;
  }

  /**
   * Clear the buffer without returning events
   */
  clearBuffer(): void {
    this.buffer = [];
  }

  /**
   * Check if recording is active
   */
  isActive(): boolean {
    return this.isRecording;
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    this.stop();
    this.clearBuffer();
  }
}

/**
 * Get opt-in status from settings store
 * Reads dynamically from localStorage on each check to respect real-time changes
 */
function getSessionReplayOptIn(): boolean {
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      const settings = localStorage.getItem('subcults-settings');
      if (settings) {
        const parsed = JSON.parse(settings);
        return parsed.sessionReplayOptIn ?? false;
      }
    }
  } catch (error) {
    console.warn('[SessionReplay] Failed to read opt-in status:', error);
  }
  return false; // Default to false (opt-out)
}

// Export singleton instance with dynamic opt-in check
export const sessionReplay = new SessionReplay({}, getSessionReplayOptIn);

// Export class for testing
export { SessionReplay };
