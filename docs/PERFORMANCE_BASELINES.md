# Performance Baselines and Bundle Size Tracking

This document tracks historical performance metrics and bundle sizes to help detect regressions.

## Current Performance Budgets

See [PERFORMANCE_MONITORING.md](./PERFORMANCE_MONITORING.md) for detailed budget definitions.

### Core Web Vitals Targets
- FCP: <1.0s
- LCP: <2.5s
- CLS: <0.1
- INP: <200ms (via TBT)
- TTFB: <600ms

### Bundle Size Targets
- JavaScript: <300KB
- CSS: <50KB
- HTML: <20KB
- Total: <500KB

## Baseline Measurements

### Initial Implementation (2025-02-01)

**Bundle Sizes** (after implementing performance monitoring):
- JavaScript: TBD - Run `npm run build` to measure
- CSS: TBD
- HTML: TBD
- Total: TBD

**Lighthouse Scores** (Desktop, No Throttling):
- Performance: TBD - Run `npm run lighthouse` to measure
- Accessibility: TBD
- Best Practices: TBD
- SEO: TBD

**Core Web Vitals** (from initial Lighthouse run):
- FCP: TBD
- LCP: TBD
- CLS: TBD
- TBT: TBD
- Speed Index: TBD

### How to Update Baselines

After making significant changes to the frontend:

1. **Build the frontend**:
   ```bash
   cd web
   npm run build
   ```

2. **Check bundle stats**:
   - Open `web/dist/stats.html` in a browser
   - Record sizes for JS, CSS, and total
   - Compare gzip/brotli sizes

3. **Run Lighthouse CI locally**:
   ```bash
   npm run lighthouse
   ```

4. **Extract metrics**:
   - Open `.lighthouseci/lhr-*.html` reports
   - Record scores and metrics
   - Compare against previous baselines

5. **Update this document**:
   - Add new entry with date
   - Record all relevant metrics
   - Note any significant changes or optimizations

## Historical Tracking

### Version History

| Date | Version | Total Size | Performance Score | Notes |
|------|---------|-----------|------------------|-------|
| 2025-02-01 | Initial | TBD | TBD | Performance monitoring added |

### Significant Changes

**2025-02-01**: Performance Monitoring Implementation
- Added web-vitals library (+~5KB gzipped)
- Added rollup-plugin-visualizer (build-time only)
- Added Lighthouse CI workflow
- Implemented telemetry endpoint

Expected impact: Minimal (<10KB increase from web-vitals library)

## Regression Detection

If CI fails due to budget violations:

1. **Review the Lighthouse report** in the CI artifacts
2. **Compare bundle size** using stats.html
3. **Identify culprits**:
   - New dependencies added
   - Large assets not lazy-loaded
   - Inefficient code patterns
4. **Take action**:
   - Code split large dependencies
   - Optimize images and assets
   - Review and remove unused code
5. **Re-run tests** to verify improvements

## Optimization Opportunities

### Current Optimization Strategies

1. **Code Splitting**:
   - Lazy load route components
   - Dynamic imports for large libraries
   - Separate vendor chunks

2. **Asset Optimization**:
   - Compress images (WebP, AVIF)
   - Minify and tree-shake
   - Use CDN for static assets

3. **Caching Strategy**:
   - Service worker for offline support
   - HTTP cache headers
   - Versioned asset URLs

### Future Optimizations

- [ ] Implement route-based code splitting
- [ ] Convert images to modern formats (WebP/AVIF)
- [ ] Enable Brotli compression in production
- [ ] Implement resource hints (preload, prefetch)
- [ ] Optimize font loading strategy
- [ ] Enable HTTP/2 server push for critical resources
- [ ] Implement skeleton screens for perceived performance

## Monitoring in Production

Once deployed to production:

1. **Set up RUM (Real User Monitoring)**:
   - Aggregate web-vitals data from real users
   - Track performance by region, device, network
   - Set up percentile tracking (p50, p75, p95, p99)

2. **Create dashboards**:
   - Time-series graphs for each metric
   - Breakdown by page/route
   - Alert on regressions

3. **Regular audits**:
   - Monthly Lighthouse audits
   - Quarterly bundle size reviews
   - Performance budget updates as needed

## Resources

- Bundle analysis: `web/dist/stats.html` (generated after build)
- Lighthouse reports: `.lighthouseci/` (generated after `npm run lighthouse`)
- Web Vitals library: https://github.com/GoogleChrome/web-vitals
- Performance budgets: https://web.dev/performance-budgets-101/
