import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  initPerformanceMonitoring,
  reportCustomMetric,
  getConfig,
  type PerformanceMetric,
} from './performance-metrics';

describe('Performance Metrics Service', () => {
  let sendBeaconSpy: ReturnType<typeof vi.fn>;
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // Mock navigator.sendBeacon
    sendBeaconSpy = vi.fn().mockReturnValue(true);
    Object.defineProperty(navigator, 'sendBeacon', {
      value: sendBeaconSpy,
      writable: true,
      configurable: true,
    });

    // Mock fetch
    fetchSpy = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
    });
    global.fetch = fetchSpy;

    // Mock console methods
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initPerformanceMonitoring', () => {
    it('should enable monitoring when telemetry is not opted out', () => {
      initPerformanceMonitoring(false);
      const config = getConfig();
      expect(config.enabled).toBe(true);
      expect(console.log).toHaveBeenCalledWith(
        '[PerformanceMetrics] Monitoring initialized'
      );
    });

    it('should disable monitoring when telemetry is opted out', () => {
      initPerformanceMonitoring(true);
      const config = getConfig();
      expect(config.enabled).toBe(false);
      expect(console.log).toHaveBeenCalledWith(
        '[PerformanceMetrics] Telemetry disabled by user preference'
      );
    });

    it('should accept custom configuration', () => {
      initPerformanceMonitoring(false, {
        endpoint: '/custom/endpoint',
      });
      const config = getConfig();
      expect(config.endpoint).toBe('/custom/endpoint');
      expect(config.enabled).toBe(true);
    });

    it('should respect custom enabled flag even when not opted out', () => {
      initPerformanceMonitoring(false, {
        enabled: false,
      });
      const config = getConfig();
      expect(config.enabled).toBe(false);
    });
  });

  describe('reportCustomMetric', () => {
    it('should send metric when telemetry is enabled', () => {
      initPerformanceMonitoring(false);
      reportCustomMetric('custom-metric', 123.45);

      expect(sendBeaconSpy).toHaveBeenCalledTimes(1);
      const [endpoint, blob] = sendBeaconSpy.mock.calls[0];
      expect(endpoint).toBe('/api/telemetry/metrics');
      
      // Read blob content
      const reader = new FileReader();
      reader.onload = () => {
        const payload = JSON.parse(reader.result as string);
        expect(payload.metrics).toHaveLength(1);
        expect(payload.metrics[0].name).toBe('custom-metric');
        expect(payload.metrics[0].value).toBe(123.45);
        expect(payload.userAgent).toBe(navigator.userAgent);
      };
      reader.readAsText(blob);
    });

    it('should not send metric when telemetry is disabled', () => {
      initPerformanceMonitoring(true);
      reportCustomMetric('custom-metric', 123.45);

      expect(sendBeaconSpy).not.toHaveBeenCalled();
      expect(fetchSpy).not.toHaveBeenCalled();
    });

    it('should fallback to fetch when sendBeacon is unavailable', () => {
      // Remove sendBeacon
      Object.defineProperty(navigator, 'sendBeacon', {
        value: undefined,
        writable: true,
        configurable: true,
      });

      initPerformanceMonitoring(false);
      reportCustomMetric('custom-metric', 456.78);

      expect(fetchSpy).toHaveBeenCalledTimes(1);
      const [endpoint, options] = fetchSpy.mock.calls[0];
      expect(endpoint).toBe('/api/telemetry/metrics');
      expect(options.method).toBe('POST');
      expect(options.headers['Content-Type']).toBe('application/json');
      expect(options.keepalive).toBe(true);

      const body = JSON.parse(options.body);
      expect(body.metrics[0].name).toBe('custom-metric');
      expect(body.metrics[0].value).toBe(456.78);
    });

    it('should handle fetch errors gracefully', async () => {
      // Remove sendBeacon
      Object.defineProperty(navigator, 'sendBeacon', {
        value: undefined,
        writable: true,
        configurable: true,
      });

      fetchSpy.mockRejectedValue(new Error('Network error'));

      initPerformanceMonitoring(false);
      reportCustomMetric('custom-metric', 789.01);

      // Wait for async operation
      await vi.waitFor(() => {
        expect(console.warn).toHaveBeenCalledWith(
          '[PerformanceMetrics] Failed to send metric:',
          expect.any(Error)
        );
      });
    });
  });

  describe('getConfig', () => {
    it('should return a copy of configuration', () => {
      initPerformanceMonitoring(false, {
        endpoint: '/test/endpoint',
      });

      const config1 = getConfig();
      const config2 = getConfig();

      expect(config1).toEqual(config2);
      expect(config1).not.toBe(config2); // Different objects
    });

    it('should return default configuration when freshly initialized', () => {
      // Reset to defaults
      initPerformanceMonitoring(false);
      
      const config = getConfig();
      expect(config.endpoint).toBe('/api/telemetry/metrics');
      expect(config.enabled).toBe(true);
    });
  });
});
