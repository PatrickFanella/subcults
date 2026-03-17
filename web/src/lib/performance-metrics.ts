/**
 * Performance Metrics Service
 * 
 * Captures and reports Core Web Vitals using the web-vitals library.
 * Respects user privacy settings (telemetry opt-out).
 * 
 * Core Web Vitals tracked:
 * - FCP (First Contentful Paint): <1.0s target
 * - LCP (Largest Contentful Paint): <2.5s target
 * - CLS (Cumulative Layout Shift): <0.1 target
 * - INP (Interaction to Next Paint): <200ms target
 * - TTFB (Time to First Byte): <600ms target
 */

import { onCLS, onFCP, onINP, onLCP, onTTFB, type Metric } from 'web-vitals';

/**
 * Performance metric data structure for reporting
 */
export interface PerformanceMetric {
  name: string;
  value: number;
  rating: 'good' | 'needs-improvement' | 'poor';
  delta: number;
  id: string;
  navigationType: string;
  timestamp: number;
}

/**
 * Telemetry endpoint configuration
 */
interface TelemetryConfig {
  endpoint: string;
  enabled: boolean;
}

/**
 * Default telemetry configuration
 */
const DEFAULT_CONFIG: TelemetryConfig = {
  endpoint: '/api/telemetry/metrics',
  enabled: true,
};

let config: TelemetryConfig = { ...DEFAULT_CONFIG };

/**
 * Initialize performance monitoring
 * 
 * @param telemetryOptOut - User's telemetry opt-out preference
 * @param customConfig - Optional custom configuration
 */
export function initPerformanceMonitoring(
  telemetryOptOut: boolean,
  customConfig?: Partial<TelemetryConfig>
): void {
  config = {
    ...DEFAULT_CONFIG,
    ...customConfig,
    enabled: !telemetryOptOut && (customConfig?.enabled ?? DEFAULT_CONFIG.enabled),
  };

  if (!config.enabled) {
    console.log('[PerformanceMetrics] Telemetry disabled by user preference');
    return;
  }

  // Register web vitals listeners
  onCLS(handleMetric);
  onFCP(handleMetric);
  onINP(handleMetric);
  onLCP(handleMetric);
  onTTFB(handleMetric);

  console.log('[PerformanceMetrics] Monitoring initialized');
}

/**
 * Handle individual metric measurement
 */
function handleMetric(metric: Metric): void {
  if (!config.enabled) {
    return;
  }

  const performanceMetric: PerformanceMetric = {
    name: metric.name,
    value: metric.value,
    rating: metric.rating,
    delta: metric.delta,
    id: metric.id,
    navigationType: metric.navigationType,
    timestamp: Date.now(),
  };

  // Log to console in development
  if (import.meta.env.DEV) {
    console.log(`[PerformanceMetrics] ${metric.name}:`, {
      value: metric.value,
      rating: metric.rating,
    });
  }

  // Send to telemetry endpoint
  sendMetric(performanceMetric);
}

/**
 * Send metric to telemetry endpoint
 * 
 * Uses sendBeacon API for reliability (non-blocking, survives page unload)
 * Falls back to fetch if sendBeacon is unavailable
 */
function sendMetric(metric: PerformanceMetric): void {
  const payload = JSON.stringify({
    metrics: [metric],
    userAgent: navigator.userAgent,
    url: window.location.href,
  });

  // Prefer sendBeacon for reliability
  if (navigator.sendBeacon) {
    const blob = new Blob([payload], { type: 'application/json' });
    navigator.sendBeacon(config.endpoint, blob);
  } else {
    // Fallback to fetch (best effort)
    fetch(config.endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: payload,
      keepalive: true, // Allows request to complete even if page unloads
    }).catch((error) => {
      // Silent failure - don't disrupt user experience
      console.warn('[PerformanceMetrics] Failed to send metric:', error);
    });
  }
}

/**
 * Manually report a custom performance metric
 * 
 * @param name - Metric name
 * @param value - Metric value (milliseconds)
 */
export function reportCustomMetric(name: string, value: number): void {
  if (!config.enabled) {
    return;
  }

  const metric: PerformanceMetric = {
    name,
    value,
    rating: 'good', // Custom metrics don't have automatic rating
    delta: value,
    id: `custom-${Date.now()}-${Math.random()}`,
    navigationType: 'custom',
    timestamp: Date.now(),
  };

  if (import.meta.env.DEV) {
    console.log(`[PerformanceMetrics] Custom metric ${name}:`, value);
  }

  sendMetric(metric);
}

/**
 * Get current telemetry configuration
 */
export function getConfig(): Readonly<TelemetryConfig> {
  return { ...config };
}
