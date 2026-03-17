# Stream E2E Testing - Quick Start Guide

## Prerequisites

- Node.js 18+
- npm or yarn
- (Optional) k6 for load testing

## Installation

### 1. Install Frontend Dependencies

```bash
cd web
npm install
```

### 2. Install E2E Test Dependencies

```bash
cd ../e2e
npm install
```

### 3. Install Playwright Browsers

```bash
cd e2e
npx playwright install chromium
# Or install all browsers for full test coverage
npx playwright install
```

### 4. Install k6 (Optional - for load testing)

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows:**
```bash
choco install k6
```

## Running Tests

### Quick Test (From Root Directory)

```bash
# Run all E2E tests
npm run test:e2e

# Run E2E tests with interactive UI
npm run test:e2e:ui

# Run load tests
npm run test:load
```

### Detailed Testing (From e2e Directory)

```bash
cd e2e

# List all available tests
npx playwright test --list

# Run all tests
npm run test:e2e

# Run specific test file
npx playwright test tests/stream-lifecycle.spec.ts

# Run tests in headed mode (see browser)
npx playwright test --headed

# Run tests in UI mode (interactive)
npx playwright test --ui

# Run tests in debug mode
npx playwright test --debug

# Run specific browser
npx playwright test --project=chromium

# Run with specific test pattern
npx playwright test -g "should create, join, leave"
```

### Load Testing

```bash
cd e2e

# Run full load test (100+ concurrent users)
npm run test:load

# Run smoke test (quick validation)
npm run test:load:smoke

# Run with custom parameters
k6 run --vus 50 --duration 2m ../perf/k6/stream-load-test.js
```

### Using Makefile (From Root)

```bash
# Run E2E tests
make test-e2e

# Run load tests
make test-load
```

## Test Structure

```
e2e/
├── mocks/
│   ├── mock-livekit-server.ts  # Mock LiveKit WebSocket server
│   └── mock-api-server.ts      # Mock API server
├── tests/
│   ├── stream-lifecycle.spec.ts     # Basic lifecycle tests (6 tests)
│   ├── organizer-controls.spec.ts   # Organizer features (5 tests)
│   └── network-resilience.spec.ts   # Network failure tests (7 tests)
└── playwright.config.ts         # Playwright configuration
```

## Test Scenarios Covered

### Stream Lifecycle (6 tests)
- ✅ Create, join, leave, end stream
- ✅ Connection indicator visibility
- ✅ Multiple participants
- ✅ Volume persistence
- ✅ Latency overlay (dev mode)
- ✅ Connection error handling

### Organizer Controls (5 tests)
- ✅ Mute participant
- ✅ Kick participant
- ✅ Lock/unlock stream
- ✅ End stream for everyone
- ✅ Permission validation

### Network Resilience (7 tests)
- ✅ Automatic reconnection
- ✅ Multiple reconnection attempts
- ✅ Max reconnection limit
- ✅ Quality degradation
- ✅ Slow network handling
- ✅ Packet loss tolerance
- ✅ High latency handling

### Load Test
- ✅ 100+ concurrent users
- ✅ Performance metrics (token fetch, WS connection, total join time)
- ✅ Success rate >95%
- ✅ Error count <10

## Viewing Test Results

### Playwright Reports

```bash
# After running tests, view HTML report
cd e2e
npx playwright show-report ../e2e-report
```

Or from root:
```bash
npm run test:e2e:report
```

### Load Test Results

Results are saved to:
```
perf/k6/stream-load-test-results.json
```

View with:
```bash
cat perf/k6/stream-load-test-results.json | jq
```

## Troubleshooting

### Tests can't find elements

**Problem**: Tests timeout waiting for elements
**Solution**: 
- Verify frontend is running on port 5173
- Check data-testid attributes match expectations
- Run tests in headed mode to see what's happening: `npx playwright test --headed`

### WebSocket connection failed

**Problem**: Mock server connection errors
**Solution**:
- Check ports 7880 and 8080 are available
- Verify no firewall blocking connections
- Check mock server logs in test output

### Load test high error rate

**Problem**: k6 shows many connection errors
**Solution**:
- Reduce VUs: `k6 run --vus 25 --duration 1m ...`
- Check system resources (CPU/memory)
- Verify mock server can handle load

### Playwright browsers not installed

**Problem**: Error about missing browsers
**Solution**:
```bash
cd e2e
npx playwright install chromium
```

## CI/CD

Tests run automatically in GitHub Actions:
- **E2E Tests**: On every PR affecting web/e2e/livekit code
- **Load Tests**: On push to main/develop branches only

View workflow file: `.github/workflows/e2e-tests.yml`

## Performance Targets

| Metric | Target | Test |
|--------|--------|------|
| Token fetch (p95) | <300ms | k6 |
| WS connection (p95) | <1000ms | k6 |
| Total join time (p95) | <2000ms | k6 |
| Concurrent users | 100+ | k6 |
| Success rate | >95% | k6 |
| Connection errors | <10 | k6 |

## Next Steps

1. **Run your first test**:
   ```bash
   npm run test:e2e
   ```

2. **Explore test results**:
   ```bash
   npm run test:e2e:report
   ```

3. **Add new tests**: See `e2e/README.md` for contributing guidelines

4. **Run load tests**:
   ```bash
   npm run test:load
   ```

## Documentation

- **Detailed Testing Guide**: `e2e/README.md`
- **Implementation Summary**: `docs/STREAM_E2E_TESTING.md`
- **Accessibility Testing**: `docs/ACCESSIBILITY_TESTING.md`

## Getting Help

- Check Playwright docs: https://playwright.dev/docs/intro
- Check k6 docs: https://k6.io/docs/
- Review test logs and reports
- Run tests in debug mode: `npx playwright test --debug`

## Summary

✅ **18 E2E tests** across 3 test suites
✅ **Mock infrastructure** for fast, reliable testing
✅ **Load testing** with k6 for performance validation
✅ **CI/CD integration** for automated testing
✅ **Comprehensive documentation** and debugging tools

All acceptance criteria met! 🎉
