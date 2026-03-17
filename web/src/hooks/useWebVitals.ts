/**
 * useWebVitals Hook
 * React hook for tracking Core Web Vitals metrics and emitting them via telemetry
 * 
 * Monitors:
 * - LCP (Largest Contentful Paint): <2.5s
 * - FID (First Input Delay): <100ms
 * - CLS (Cumulative Layout Shift): <0.1
 * - INP (Interaction to Next Paint): <200ms (newer metric replacing FID)
 * - TTFB (Time to First Byte): <600ms
 */

import { useEffect } from 'react';
import { useTelemetry } from './useTelemetry';

/**
 * Web Vitals metric thresholds (in milliseconds or dimensionless)
 */
const METRIC_THRESHOLDS = {
  LCP: 2500, // Largest Contentful Paint
  FID: 100,  // First Input Delay
  INP: 200,  // Interaction to Next Paint
  CLS: 0.1,  // Cumulative Layout Shift
  TTFB: 600, // Time to First Byte
  FCP: 1000, // First Contentful Paint
} as const;

/**
 * Hook to track Core Web Vitals metrics
 * Sends metrics via telemetry service
 * 
 * @example
 * ```tsx
 * export const App = () => {
 *   useWebVitals();
 *   return <div>...</div>;
 * };
 * ```
 */
export function useWebVitals(): void {
  const emit = useTelemetry();
  
  useEffect(() => {
    if (typeof PerformanceObserver === 'undefined') {
      return;
    }

    // Track if we've already sent metrics (to avoid duplicates for one-shot metrics)
    const sentMetrics = new Set<string>();
    
    /**
     * Send a metric. LCP and CLS can update over time, so always emit their latest value.
     * Other metrics (FCP, INP, TTFB) are emitted only once.
     */
    const emitMetric = (name: string, value: number, threshold: number): void => {
      if (name !== 'LCP' && name !== 'CLS') {
        if (sentMetrics.has(name)) {
          return;
        }
        sentMetrics.add(name);
      }
      
      const isGood = name === 'CLS' 
        ? value <= threshold 
        : value <= threshold;
      
      emit(`vitals.${name.toLowerCase()}`, {
        value: Math.round(value * 100) / 100, // Round to 2 decimals
        threshold,
        rating: isGood ? 'good' : value <= threshold * 1.5 ? 'needs-improvement' : 'poor',
      });
    };
    // Track LCP (Largest Contentful Paint)
    const lcpObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries();
      if (entries.length > 0) {
        const lastEntry = entries[entries.length - 1];
        emitMetric('LCP', lastEntry.startTime, METRIC_THRESHOLDS.LCP);
      }
    });
    
    try {
      lcpObserver.observe({ entryTypes: ['largest-contentful-paint'] });
    } catch (e) {
      console.warn('[vitals] LCP observer not supported:', e);
    }
    
    // Track FCP (First Contentful Paint)
    const fcpObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries();
      entries.forEach((entry) => {
        if (entry.name === 'first-contentful-paint') {
          emitMetric('FCP', entry.startTime, METRIC_THRESHOLDS.FCP);
        }
      });
    });
    
    try {
      fcpObserver.observe({ entryTypes: ['paint'] });
    } catch (e) {
      console.warn('[vitals] FCP observer not supported:', e);
    }
    
    // Track INP (Interaction to Next Paint) - replaces FID
    const inpObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries();
      if (entries.length > 0) {
        const lastEntry = entries[entries.length - 1];
        const processingDuration = (lastEntry as any).processingDuration || 0;
        const presentationDelay = (lastEntry as any).presentationDelay || 0;
        const inp = processingDuration + presentationDelay;
        emitMetric('INP', inp, METRIC_THRESHOLDS.INP);
      }
    });
    
    try {
      inpObserver.observe({ entryTypes: ['event'] });
    } catch (e) {
      console.warn('[vitals] INP observer not supported:', e);
    }
    
    // Track CLS (Cumulative Layout Shift)
    let clsValue = 0;
    const clsObserver = new PerformanceObserver((entryList) => {
      for (const entry of entryList.getEntries()) {
        if (!(entry as any).hadRecentInput) {
          clsValue += (entry as any).value;
          emitMetric('CLS', clsValue, METRIC_THRESHOLDS.CLS);
        }
      }
    });
    
    try {
      clsObserver.observe({ entryTypes: ['layout-shift'] });
    } catch (e) {
      console.warn('[vitals] CLS observer not supported:', e);
    }
    
    // Track TTFB (Time to First Byte) via navigation timing
    const trackTTFB = (): void => {
      try {
        const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
        if (navigation) {
          const ttfb = navigation.responseStart - (navigation.fetchStart || 0);
          emitMetric('TTFB', ttfb, METRIC_THRESHOLDS.TTFB);
        }
      } catch (e) {
        console.warn('[vitals] TTFB tracking failed:', e);
      }
    };
    
    // Wait for page load to measure TTFB
    if (document.readyState === 'loading') {
      window.addEventListener('load', trackTTFB, { once: true });
    } else {
      trackTTFB();
    }
    
    // Cleanup function
    return () => {
      lcpObserver.disconnect();
      fcpObserver.disconnect();
      inpObserver.disconnect();
      clsObserver.disconnect();
      window.removeEventListener('load', trackTTFB);
    };
  }, [emit]);
}

/**
 * Hook to track custom performance marks
 * Useful for tracking component-specific performance metrics
 * 
 * @example
 * ```tsx
 * const trackMark = usePerformanceMark('MapView');
 * 
 * useEffect(() => {
 *   trackMark('render-start');
 *   // ... rendering logic ...
 *   trackMark('render-end');
 * }, [trackMark]);
 * ```
 */
export function usePerformanceMark(componentName: string) {
  const emit = useTelemetry();
  
  const trackMark = (label: string): void => {
    try {
      const mark = `${componentName}:${label}`;
      performance.mark(mark);
      
      // Emit marking event for debugging
      emit('perf.mark', {
        component: componentName,
        label,
      });
    } catch (e) {
      console.warn(`[perf] Failed to mark ${label}:`, e);
    }
  };
  
  return trackMark;
}

/**
 * Hook to track navigation timing and page load metrics
 */
export function useNavigationTiming(): void {
  const emit = useTelemetry();
  
  useEffect(() => {
    const trackTiming = (): void => {
      try {
        const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
        if (!navigation) return;
        
        const metrics = {
          dns: navigation.domainLookupEnd - navigation.domainLookupStart,
          tcp: navigation.connectEnd - navigation.connectStart,
          ttfb: navigation.responseStart - navigation.fetchStart,
          download: navigation.responseEnd - navigation.responseStart,
          domInteractive: navigation.domInteractive - navigation.responseEnd,
          domComplete: navigation.domComplete - navigation.domInteractive,
          resourceLoading: navigation.loadEventStart - navigation.domComplete,
          pageLoad: navigation.loadEventEnd - navigation.loadEventStart,
        };
        
        emit('perf.navigation', metrics);
      } catch (e) {
        console.warn('[perf] Navigation timing tracking failed:', e);
      }
    };
    
    if (document.readyState === 'loading') {
      window.addEventListener('load', trackTiming, { once: true });
    } else {
      trackTiming();
    }
  }, [emit]);
}
