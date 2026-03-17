# Performance Monitoring and Budget Management

This document describes the performance monitoring and budget management setup for Subcults.

## Overview

The project uses a comprehensive performance monitoring stack:

1. **Web Vitals Monitoring**: Real-time Core Web Vitals tracking using the `web-vitals` library
2. **Telemetry Backend**: API endpoint for collecting and aggregating performance metrics
3. **Bundle Analysis**: Automatic bundle size visualization with rollup-plugin-visualizer
4. **Lighthouse CI**: Automated performance audits in CI/CD with budget enforcement

## Performance Budgets

The following performance budgets are enforced in Lighthouse CI:

| Metric | Target | Status |
|--------|--------|--------|
| **First Contentful Paint (FCP)** | <1.0s | Error if exceeded |
| **Largest Contentful Paint (LCP)** | <2.5s | Error if exceeded |
| **Cumulative Layout Shift (CLS)** | <0.1 | Error if exceeded |
| **Interaction to Next Paint (INP)** | <200ms (via TBT) | Error if exceeded |
| **Time to First Byte (TTFB)** | <600ms | Error if exceeded |
| **Speed Index** | <3.0s | Error if exceeded |
| **Total Blocking Time (TBT)** | <200ms | Error if exceeded |
| **Performance Score** | >90% | Error if below |

### Bundle Size Budgets

| Resource | Budget | Status |
|----------|--------|--------|
| **JavaScript** | <300KB | Warning if exceeded |
| **CSS** | <50KB | Warning if exceeded |
| **HTML** | <20KB | Warning if exceeded |
| **Total** | <500KB | Warning if exceeded |

## Web Vitals Monitoring

### Client-Side Collection

The frontend automatically collects Core Web Vitals metrics using the `web-vitals` library:

```typescript
import { initPerformanceMonitoring } from './lib/performance-metrics';
import { useSettingsStore } from './stores/settingsStore';

// Initialize on app startup
const { telemetryOptOut } = useSettingsStore.getState();
initPerformanceMonitoring(telemetryOptOut);
```

**Privacy**: Users can opt-out of telemetry collection via the settings store. When opted out, no metrics are collected or sent.

### Metrics Collected

- **FCP (First Contentful Paint)**: Time until first DOM content is rendered
- **LCP (Largest Contentful Paint)**: Time until largest content element is rendered
- **CLS (Cumulative Layout Shift)**: Visual stability metric
- **INP (Interaction to Next Paint)**: Responsiveness metric (via web-vitals)
- **TTFB (Time to First Byte)**: Server response time

### Backend Telemetry Endpoint

Metrics are sent to `POST /api/telemetry/metrics` with the following payload:

```json
{
  "metrics": [
    {
      "name": "LCP",
      "value": 1234.56,
      "rating": "good",
      "delta": 1234.56,
      "id": "v3-1234567890-1234567890",
      "navigationType": "navigate",
      "timestamp": 1234567890000
    }
  ],
  "userAgent": "Mozilla/5.0...",
  "url": "https://example.com/page"
}
```

Metrics are logged to structured logs for aggregation and analysis.

## Bundle Analysis

### Viewing Bundle Size

Bundle analysis is automatically generated during the build process:

```bash
cd web
npm run build
```

This creates `web/dist/stats.html` with an interactive visualization of:
- Bundle size by module
- Tree map of dependencies
- Gzip and Brotli compressed sizes

### Reducing Bundle Size

If bundle size exceeds budgets:

1. **Analyze the bundle**: Open `web/dist/stats.html` in a browser
2. **Identify large dependencies**: Look for unexpectedly large modules
3. **Code splitting**: Use dynamic imports for large components
4. **Tree shaking**: Ensure dead code elimination is working
5. **Compression**: Verify Gzip/Brotli compression is enabled

## Lighthouse CI

### Local Testing

Run Lighthouse audits locally:

```bash
# Build and run local server
npm run lighthouse:local
```

In another terminal:

```bash
# Run Lighthouse CI
npm run lighthouse
```

### CI/CD Integration

Lighthouse CI runs automatically on:
- Pull requests that modify frontend code
- Pushes to `main` or `develop` branches

The workflow:
1. Builds the frontend in production mode
2. Starts a local HTTP server
3. Runs Lighthouse audits (3 runs, desktop preset)
4. Uploads results as artifacts
5. Posts summary comment on PRs
6. **Fails the build** if performance budgets are exceeded by >10%

### Regression Threshold

The CI allows a **10% regression** on numeric metrics before failing. This prevents false positives from minor fluctuations while catching significant performance degradations.

### Viewing Results

**In Pull Requests**:
- Lighthouse posts a comment with performance scores
- Click the "View detailed results" link to see full report

**In GitHub Actions**:
- Go to Actions → Lighthouse CI workflow
- Download `lighthouse-results` artifact
- Open HTML reports in `.lighthouseci/` directory

## Troubleshooting

### Metrics Not Appearing

1. **Check telemetry opt-out**: Verify user hasn't opted out in settings
2. **Browser console**: Look for `[PerformanceMetrics]` logs in development
3. **Network tab**: Check for requests to `/api/telemetry/metrics`
4. **Backend logs**: Search for `performance_metric` log entries

### Lighthouse CI Failures

1. **Local testing**: Run `npm run lighthouse` to reproduce locally
2. **Bundle size**: Check `web/dist/stats.html` for large dependencies
3. **Performance bottlenecks**: Review Lighthouse suggestions in HTML report
4. **Throttling settings**: Adjust `lighthouserc.js` throttling if needed

### Bundle Analysis Not Generated

1. **Build command**: Ensure you're running `npm run build` in `web/` directory
2. **Plugin configuration**: Verify `rollup-plugin-visualizer` is in `vite.config.ts`
3. **Output location**: Check `web/dist/stats.html` after build

## Future Enhancements

### Telemetry Aggregation

Currently, metrics are logged to stdout. Future enhancements:

- Store metrics in time-series database (e.g., Prometheus, InfluxDB)
- Create Grafana dashboards for visualization
- Set up alerting for performance regressions
- A/B testing performance impact

### Advanced Budgets

- Per-route budgets (e.g., map page vs. settings page)
- Network-specific budgets (3G, 4G, WiFi)
- Device-specific budgets (mobile, tablet, desktop)

### Continuous Monitoring

- Real User Monitoring (RUM) with percentile tracking
- Synthetic monitoring from multiple geographic locations
- Performance regression detection across versions

## References

- [Web Vitals](https://web.dev/vitals/)
- [Lighthouse CI Documentation](https://github.com/GoogleChrome/lighthouse-ci)
- [rollup-plugin-visualizer](https://github.com/btd/rollup-plugin-visualizer)
- [Core Web Vitals Thresholds](https://web.dev/defining-core-web-vitals-thresholds/)
