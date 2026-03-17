# Testing Guide

This document provides an overview of testing in the Subcults project.

## Test Types

### 1. Unit Tests

**Location**: `web/src/**/*.test.ts`, `internal/**/*_test.go`

**Purpose**: Test individual components and functions in isolation

**Running**:
```bash
# Frontend unit tests
cd web && npm test

# Backend unit tests
go test -v ./...

# Or use make
make test
```

**Coverage Targets**:
- Backend: >80%
- Frontend: >70%

### 2. E2E Tests (End-to-End)

**Location**: `e2e/tests/*.spec.ts`

**Purpose**: Test complete user workflows with real browser interactions

**Running**:
```bash
# Run all E2E tests
npm run test:e2e

# Interactive UI mode
npm run test:e2e:ui

# Or use make
make test-e2e
```

**Test Suites**:
- Stream Lifecycle (6 tests)
- Organizer Controls (5 tests)
- Network Resilience (7 tests)

**Quick Start**: See [e2e/QUICKSTART.md](e2e/QUICKSTART.md)

**Detailed Guide**: See [e2e/README.md](e2e/README.md)

### 3. Load Tests

**Location**: `perf/k6/*.js`

**Purpose**: Validate performance under high concurrent load

**Running**:
```bash
# Run load tests
npm run test:load

# Or use make
make test-load
```

**Targets**:
- 100+ concurrent users
- p95 latency <2s
- Success rate >95%

### 4. Accessibility Tests

**Location**: `web/src/**/*.a11y.test.tsx`

**Purpose**: Ensure WCAG 2.1 Level AA compliance

**Running**:
```bash
cd web
npm test -- --run src/pages/*.a11y.test.tsx
```

**Guide**: See [docs/ACCESSIBILITY_TESTING.md](docs/ACCESSIBILITY_TESTING.md)

## Quick Commands

```bash
# Run all tests
make test            # Unit tests only
npm run test:e2e     # E2E tests
npm run test:load    # Load tests

# Run specific test types
cd web && npm test                      # Frontend unit tests
go test -v ./...                        # Backend unit tests
cd e2e && npx playwright test           # E2E tests
k6 run perf/k6/stream-load-test.js     # Load tests

# Interactive testing
npm run test:e2e:ui                     # Playwright UI mode
cd web && npm run test:ui               # Vitest UI mode

# View reports
npm run test:e2e:report                 # Playwright HTML report
cd web && npm run test:coverage         # Coverage report
```

## Test Infrastructure

### Mock Servers

For E2E testing, we use mock servers that simulate external services:

- **Mock LiveKit Server**: WebSocket server simulating LiveKit functionality
- **Mock API Server**: Express server simulating Subcults API

These run automatically when E2E tests start.

### Test Data

- Unit tests use inline test data
- E2E tests use mock servers with dynamic data
- Load tests simulate realistic user behavior

## CI/CD Integration

Tests run automatically in GitHub Actions:

- **Unit Tests**: On every PR
- **E2E Tests**: On PRs affecting web/e2e/livekit
- **Load Tests**: On push to main/develop only
- **Accessibility Tests**: On PRs affecting web code

Workflows: `.github/workflows/`

## Writing New Tests

### Unit Tests

```typescript
// Frontend (Vitest)
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { MyComponent } from './MyComponent';

describe('MyComponent', () => {
  it('should render correctly', () => {
    const { getByText } = render(<MyComponent />);
    expect(getByText('Hello')).toBeInTheDocument();
  });
});
```

```go
// Backend (Go)
func TestMyFunction(t *testing.T) {
    result := MyFunction("input")
    if result != "expected" {
        t.Errorf("got %v, want %v", result, "expected")
    }
}
```

### E2E Tests

```typescript
// Playwright
import { test, expect } from '@playwright/test';

test('should do something', async ({ page }) => {
  await page.goto('/path');
  await page.click('button');
  await expect(page.locator('.result')).toBeVisible();
});
```

See [e2e/README.md](e2e/README.md) for detailed guidelines.

## Performance Budgets

| Metric | Target | Test Type |
|--------|--------|-----------|
| Unit Test Suite | <2 min | Unit |
| E2E Test Suite | <10 min | E2E |
| Load Test | <5 min | Load |
| API Latency (p95) | <300ms | Load |
| Stream Join (p95) | <2s | Load/E2E |

## Debugging Tests

### Unit Tests

```bash
# Run in watch mode
cd web && npm test

# Run specific test
cd web && npm test MyComponent.test.ts

# View coverage
cd web && npm run test:coverage
```

### E2E Tests

```bash
# Debug mode (step through tests)
cd e2e && npx playwright test --debug

# Headed mode (see browser)
cd e2e && npx playwright test --headed

# Trace viewer (analyze after failure)
cd e2e && npx playwright show-trace trace.zip
```

### Load Tests

```bash
# Run with fewer VUs for debugging
k6 run --vus 5 --duration 30s perf/k6/stream-load-test.js

# View detailed logs
k6 run --http-debug perf/k6/stream-load-test.js
```

## Test Documentation

- [E2E Testing Guide](e2e/README.md) - Comprehensive E2E testing documentation
- [E2E Quick Start](e2e/QUICKSTART.md) - Get started with E2E tests quickly
- [Stream E2E Implementation](docs/STREAM_E2E_TESTING.md) - Implementation details
- [Accessibility Testing](docs/ACCESSIBILITY_TESTING.md) - A11y testing guide
- [Performance Testing](docs/PERFORMANCE.md) - Performance benchmarks

## Contributing

When adding new features:

1. **Write tests first** (TDD approach recommended)
2. **Maintain coverage** (meet or exceed targets)
3. **Test all scenarios** (success, failure, edge cases)
4. **Update documentation** (this file and specific guides)
5. **Run full test suite** before submitting PR

## Troubleshooting

### Common Issues

**Tests fail locally but pass in CI**
- Check Node.js version (use 18+)
- Clear node_modules and reinstall
- Check for environment-specific issues

**E2E tests timeout**
- Increase timeout in playwright.config.ts
- Check frontend is running properly
- Verify mock servers start correctly

**Load tests show high error rate**
- Reduce concurrent VUs
- Check system resources
- Verify mock server capacity

### Getting Help

- Check test documentation (links above)
- Review test logs and error messages
- Run tests in debug/headed mode
- Ask in team chat or create discussion issue

## Summary

- ✅ **Unit Tests**: Fast, isolated component testing
- ✅ **E2E Tests**: Real browser testing with mocks
- ✅ **Load Tests**: Performance validation at scale
- ✅ **A11y Tests**: Accessibility compliance
- ✅ **CI/CD**: Automated testing on every change

For detailed information on each test type, see the linked documentation above.
