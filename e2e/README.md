# E2E Testing for Subcults Streaming

This directory contains end-to-end (E2E) tests for the Subcults streaming functionality using Playwright and load tests using k6.

## Overview

The E2E test suite provides comprehensive testing of the LiveKit streaming integration, including:

- **Stream Lifecycle**: Create, join, leave, and end streams
- **Organizer Controls**: Mute, kick, lock/unlock, end stream
- **Network Resilience**: Reconnection, quality degradation, packet loss handling
- **Load Testing**: 100+ concurrent listeners with performance metrics

## Architecture

### Mock LiveKit Server

The tests use a mock LiveKit server (`e2e/mocks/mock-livekit-server.ts`) that simulates LiveKit functionality without requiring actual LiveKit infrastructure. This provides:

- **WebSocket-based communication**: Mimics real LiveKit WebSocket protocol
- **Room management**: Create/join/leave rooms
- **Participant tracking**: Track participants, their states, and roles
- **Organizer controls**: Support for mute, kick, lock/unlock
- **Network simulation**: Simulate latency, packet loss, quality degradation
- **Fast execution**: No external dependencies, runs entirely locally

### Mock API Server

The mock API server (`e2e/mocks/mock-api-server.ts`) provides:

- **Token generation**: Mock LiveKit token endpoint
- **Testing utilities**: Endpoints to simulate network conditions
- **Room inspection**: Debug endpoints to verify room state

### Test Structure

```
e2e/
├── mocks/
│   ├── mock-livekit-server.ts  # Mock LiveKit WebSocket server
│   └── mock-api-server.ts      # Mock API for token generation
├── tests/
│   ├── stream-lifecycle.spec.ts    # Basic stream operations
│   ├── organizer-controls.spec.ts  # Organizer-specific features
│   └── network-resilience.spec.ts  # Network failure scenarios
├── fixtures/                    # Test data and helpers
└── package.json                 # E2E test dependencies
```

## Setup

### Prerequisites

- Node.js 18+ (for Playwright)
- k6 (for load testing) - Install from https://k6.io/docs/getting-started/installation/

### Installation

```bash
# Install Playwright and dependencies
cd e2e
npm install

# Install Playwright browsers
npx playwright install

# Install k6 (if not already installed)
# macOS
brew install k6

# Linux
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Windows
choco install k6
```

## Running Tests

### E2E Tests (Playwright)

```bash
# Run all E2E tests
npm run test:e2e

# Run tests in headed mode (see browser)
npm run test:e2e:headed

# Run tests in UI mode (interactive)
npm run test:e2e:ui

# Run specific test file
npx playwright test e2e/tests/stream-lifecycle.spec.ts

# Run tests in debug mode
npm run test:e2e:debug

# View test report
npm run test:e2e:report
```

### Load Tests (k6)

```bash
# Run full load test (100+ concurrent users)
npm run test:load

# Run smoke test (10 users for 30s)
npm run test:load:smoke

# Run with custom parameters
k6 run --vus 150 --duration 10m perf/k6/stream-load-test.js

# Run with custom environment variables
BASE_URL=http://localhost:3000 \
WS_URL=ws://localhost:7880 \
ROOM_ID=custom-room \
k6 run perf/k6/stream-load-test.js
```

## Test Scenarios

### 1. Stream Lifecycle Tests

**File**: `e2e/tests/stream-lifecycle.spec.ts`

Tests basic stream operations:
- ✓ Create and join stream
- ✓ Leave stream
- ✓ End stream
- ✓ Connection indicator visibility
- ✓ Multiple participants
- ✓ Volume persistence
- ✓ Latency overlay (dev mode)
- ✓ Connection error handling

### 2. Organizer Controls Tests

**File**: `e2e/tests/organizer-controls.spec.ts`

Tests organizer-specific functionality:
- ✓ Mute participant
- ✓ Kick participant
- ✓ Lock/unlock stream
- ✓ End stream for everyone
- ✓ Non-organizer permission validation

### 3. Network Resilience Tests

**File**: `e2e/tests/network-resilience.spec.ts`

Tests network failure scenarios:
- ✓ Automatic reconnection after temporary disconnection
- ✓ Multiple reconnection attempts
- ✓ Max reconnection limit
- ✓ Quality degradation handling
- ✓ Slow network tolerance
- ✓ Intermittent packet loss
- ✓ High latency handling

### 4. Load Tests

**File**: `perf/k6/stream-load-test.js`

Tests system performance under load:
- ✓ 100+ concurrent listeners
- ✓ Token fetch latency (p95 < 300ms)
- ✓ WebSocket connection time (p95 < 1s)
- ✓ Total join time (p95 < 2s)
- ✓ Success rate > 95%
- ✓ Error count < 10

## Test Configuration

### Playwright Configuration

The Playwright configuration (`playwright.config.ts`) includes:

- **Browsers**: Chromium, Firefox, WebKit, Mobile Chrome
- **Timeouts**: 60s per test, 30s navigation
- **Retries**: 2 retries in CI, 0 locally
- **Artifacts**: Screenshots on failure, videos on failure, traces on first retry
- **Reporters**: HTML, JSON, list
- **Permissions**: Microphone access for all tests
- **Fake Media**: Uses fake media streams for testing

### K6 Configuration

The k6 load test configuration includes:

- **Ramp-up stages**: 20 → 50 → 100 users over 2.5 minutes
- **Sustained load**: 100 users for 2 minutes
- **Ramp-down**: 100 → 0 users over 1 minute
- **Thresholds**: <10 errors, p95 latencies, >95% success rate
- **Custom metrics**: Connection errors, token fetch time, WS connection time, total join time

## Mock Server Features

### Simulating Network Conditions

The mock servers support simulating various network conditions:

```javascript
// Simulate high latency
await fetch('http://localhost:8080/api/test/simulate-latency', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ roomId: 'my-room', delayMs: 2000 }),
});

// Simulate packet loss
await fetch('http://localhost:8080/api/test/simulate-packet-loss', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ roomId: 'my-room', lossPercentage: 25 }),
});

// Get room state
const response = await fetch('http://localhost:8080/api/test/room/my-room');
const room = await response.json();
```

### WebSocket Protocol

The mock LiveKit server uses a simplified WebSocket protocol:

**Client → Server**:
```json
{
  "type": "mute",
  "muted": true
}
```

**Server → Client**:
```json
{
  "type": "participant_joined",
  "participant": {
    "identity": "user-123",
    "name": "User 123",
    "isOrganizer": false,
    "isMuted": false,
    "connectionQuality": "excellent"
  }
}
```

Supported message types:
- `room_state`: Initial room state
- `participant_joined`: New participant
- `participant_disconnected`: Participant left
- `participant_reconnected`: Participant reconnected
- `participant_muted`: Mute state changed
- `participant_kicked`: Participant kicked
- `room_locked`: Room lock state changed
- `stream_ended`: Stream ended by organizer
- `quality_changed`: Connection quality changed
- `kicked`: You were kicked
- `force_muted`: You were muted by organizer

## CI/CD Integration

### GitHub Actions

Example workflow for CI:

```yaml
name: E2E Tests

on:
  pull_request:
    paths:
      - 'web/**'
      - 'e2e/**'
      - 'internal/livekit/**'

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: |
          cd web && npm ci
          cd ../e2e && npm ci
      
      - name: Install Playwright browsers
        run: cd e2e && npx playwright install --with-deps
      
      - name: Run E2E tests
        run: cd e2e && npm run test:e2e
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: e2e-report/
          retention-days: 30

  load:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install k6
        run: |
          sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Run load tests
        run: k6 run --vus 50 --duration 2m perf/k6/stream-load-test.js
      
      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: k6-results
          path: perf/k6/stream-load-test-results.json
          retention-days: 30
```

## Debugging

### Playwright Debugging

```bash
# Run in debug mode
npm run test:e2e:debug

# Run with headed browser
npm run test:e2e:headed

# Run specific test
npx playwright test -g "should create, join, leave"

# Show last report
npm run test:e2e:report
```

### Mock Server Debugging

The mock servers log all connections and messages:

```
[MockLiveKit] Server started on port 7880
[MockLiveKit] Client connected: user-123 to room default
[MockLiveKit] Client disconnected: user-123
[MockAPI] Server started on port 8080
```

### Network Simulation

Test network conditions manually:

```bash
# Start mock servers (requires tsx or ts-node)
cd e2e && npx tsx mocks/mock-api-server.ts

# In another terminal, test latency
curl -X POST http://localhost:8080/api/test/simulate-latency \
  -H "Content-Type: application/json" \
  -d '{"roomId": "test-room", "delayMs": 2000}'

# Test packet loss
curl -X POST http://localhost:8080/api/test/simulate-packet-loss \
  -H "Content-Type: application/json" \
  -d '{"roomId": "test-room", "lossPercentage": 25}'

# Check room state
curl http://localhost:8080/api/test/room/test-room
```

## Performance Budgets

Based on acceptance criteria and performance requirements:

| Metric | Target | Measured By |
|--------|--------|-------------|
| Token Fetch (p95) | <300ms | k6 load test |
| WS Connection (p95) | <1000ms | k6 load test |
| Total Join Time (p95) | <2000ms | k6 load test |
| Concurrent Listeners | 100+ | k6 load test |
| Connection Errors | <10 total | k6 load test |
| Success Rate | >95% | k6 load test |
| Reconnection Time | <5s | Playwright test |
| Max Reconnect Attempts | 3 | Playwright test |

## Troubleshooting

### Common Issues

**Issue**: Tests fail with "WebSocket connection failed"
- **Solution**: Ensure mock servers are running, check port availability (7880, 8080)

**Issue**: Tests timeout waiting for elements
- **Solution**: Check that the frontend is running on port 5173, verify selectors match current UI

**Issue**: Load test shows high error rate
- **Solution**: Reduce concurrent VUs, check system resources, verify mock server can handle load

**Issue**: Browser permissions error
- **Solution**: Ensure Playwright config includes microphone permissions and fake media devices

### Getting Help

- Check Playwright docs: https://playwright.dev/docs/intro
- Check k6 docs: https://k6.io/docs/
- Review test logs in `e2e-report/` directory
- Run tests in headed mode to see what's happening

## Contributing

When adding new tests:

1. Follow existing test patterns
2. Use descriptive test names
3. Add comments for complex scenarios
4. Update this README with new test scenarios
5. Ensure tests are deterministic (no flaky tests)
6. Use appropriate timeouts and waits
7. Clean up resources after tests
8. Test both success and failure cases

## Future Enhancements

Potential improvements to the test suite:

- [ ] Add tests for audio quality metrics
- [ ] Test with real LiveKit server in staging
- [ ] Add visual regression testing
- [ ] Test screen reader compatibility
- [ ] Add chaos engineering tests
- [ ] Test with different network profiles (3G, 4G, LTE)
- [ ] Add CPU/memory profiling
- [ ] Test browser compatibility edge cases
- [ ] Add security testing (XSS, CSRF)
- [ ] Test internationalization (i18n)
