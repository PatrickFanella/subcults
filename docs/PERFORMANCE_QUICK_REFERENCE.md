# Performance Monitoring Quick Reference

## For Developers

### Running Performance Audits Locally

```bash
# 1. Build the frontend
cd web
npm run build
cd ..

# 2. Run Lighthouse CI
npm run lighthouse

# 3. View results
# Open .lighthouseci/lhr-*.html files in your browser
```

### Viewing Bundle Analysis

```bash
# Build generates stats automatically
cd web
npm run build

# Open bundle visualization
open dist/stats.html  # macOS
xdg-open dist/stats.html  # Linux
start dist/stats.html  # Windows
```

### Testing Web Vitals Collection

```bash
# 1. Start dev server
cd web
npm run dev

# 2. Open browser to http://localhost:5173
# 3. Open DevTools Console
# 4. Look for [PerformanceMetrics] logs
# 5. Check Network tab for POST to /api/telemetry/metrics
```

### Opting Out of Telemetry (for testing)

```javascript
// In browser console:
localStorage.setItem('subcults-settings', JSON.stringify({
  telemetryOptOut: true,
  sessionReplayOptIn: false
}));

// Reload page - metrics should not be sent
```

## For CI/CD

### GitHub Actions Workflow

The Lighthouse CI workflow runs automatically on:
- Pull requests that modify `web/**` or `lighthouserc.js`
- Pushes to `main` or `develop` branches

**View Results:**
1. Go to Actions tab in GitHub
2. Click on the Lighthouse CI workflow run
3. Download `lighthouse-results` artifact
4. Open HTML reports locally

**PR Comments:**
- Performance scores posted automatically as PR comment
- Includes scores for Performance, Accessibility, Best Practices, SEO

### Troubleshooting CI Failures

**Budget Violation:**
```bash
# The error will show which metric failed
# Example: "largest-contentful-paint" exceeded budget

# Local investigation:
npm run lighthouse
# Open .lighthouseci/lhr-*.html
# Look for specific metric details
```

**Bundle Too Large:**
```bash
# Check bundle visualization
cd web
npm run build
open dist/stats.html

# Look for:
# - Large dependencies that could be code-split
# - Duplicate modules
# - Unused code
```

## Performance Budget Thresholds

| Metric | Budget | Severity |
|--------|--------|----------|
| FCP | 1000ms | Error |
| LCP | 2500ms | Error |
| CLS | 0.1 | Error |
| TBT | 200ms | Error |
| TTFB | 600ms | Error |
| Speed Index | 3000ms | Error |
| Performance Score | 90% | Error |
| JS Bundle | 300KB | Warning |
| CSS Bundle | 50KB | Warning |
| Total Size | 500KB | Warning |

**Regression Allowance:** 10% increase allowed before CI fails

## Common Issues

### "Lighthouse CI Failed"

**Check:**
1. Which metric failed? (see workflow logs)
2. How much over budget? (>10% regression?)
3. What changed? (review recent commits)

**Fix:**
1. Run lighthouse locally to reproduce
2. Identify root cause (bundle size, blocking resources, etc.)
3. Apply optimizations
4. Re-run lighthouse to verify
5. Push changes

### "Bundle Size Too Large"

**Quick Wins:**
1. Code splitting: Use `React.lazy()` for route components
2. Dynamic imports: `import('module').then(...)`
3. Remove unused dependencies: Check package.json
4. Tree shaking: Ensure side-effects are declared in package.json

**Example:**
```typescript
// Before:
import { LargeComponent } from './LargeComponent';

// After (lazy load):
const LargeComponent = React.lazy(() => import('./LargeComponent'));
```

### "Web Vitals Not Reporting"

**Checklist:**
1. Is telemetry opted out? (check localStorage `subcults-settings`)
2. Is the endpoint responding? (check /api/telemetry/metrics in Network tab)
3. Browser console errors? (look for fetch failures)
4. Backend logs? (search for `performance_metric` entries)

## Best Practices

### When to Run Lighthouse

- ✅ Before opening a PR with frontend changes
- ✅ After adding new dependencies
- ✅ When optimizing performance
- ❌ Don't run on every commit (CI handles this)

### Interpreting Results

**Good Signs:**
- All metrics green in Lighthouse report
- Performance score >90%
- Bundle size under budget
- No regressions vs. baseline

**Warning Signs:**
- Yellow/orange metrics (needs improvement)
- Performance score 50-89% (investigate)
- Bundle size warnings (plan optimizations)

**Red Flags:**
- Red metrics (over budget)
- Performance score <50% (critical)
- Multiple budget violations
- Large regressions from baseline

### Optimization Workflow

1. **Measure**: Run lighthouse locally
2. **Analyze**: Review report, check bundle stats
3. **Prioritize**: Fix critical issues first
4. **Implement**: Apply optimizations
5. **Verify**: Re-run lighthouse
6. **Document**: Update baselines if needed

## Resources

- [Web Vitals Documentation](https://web.dev/vitals/)
- [Lighthouse Scoring Guide](https://web.dev/performance-scoring/)
- [Bundle Optimization Tips](https://web.dev/reduce-javascript-payloads-with-code-splitting/)
- [Performance Budgets Guide](https://web.dev/performance-budgets-101/)

## Getting Help

- Check [PERFORMANCE_MONITORING.md](./PERFORMANCE_MONITORING.md) for detailed docs
- Review [PERFORMANCE_BASELINES.md](./PERFORMANCE_BASELINES.md) for historical data
- Ask in #performance channel (if available)
- Create issue with `performance` label
