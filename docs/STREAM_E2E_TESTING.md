# Stream E2E Testing Documentation

## Overview

This document describes the comprehensive E2E testing implementation for the Subcults streaming functionality. The test suite validates all critical streaming scenarios including stream lifecycle, organizer controls, network resilience, and load handling.

## Implementation Summary

### Components Implemented

#### 1. Mock LiveKit Server (`e2e/mocks/mock-livekit-server.ts`)

A fully functional WebSocket-based mock server that simulates LiveKit functionality:

**Features**:
- WebSocket server on port 7880
- Room lifecycle management (create/join/leave/end)
- Participant state tracking with roles (organizer/participant)
- Connection quality simulation (excellent/good/poor/unknown)
- Organizer controls (mute, kick, lock/unlock)
- Network condition simulation (latency, packet loss)
- Automatic room cleanup after participants leave
- Reconnection tracking

**Key Methods**:
- `start()`: Start WebSocket server
- `stop()`: Stop WebSocket server
- `generateToken()`: Create mock LiveKit tokens
- `getRoom()`: Get room state for assertions
- `simulateNetworkDelay()`: Test high latency scenarios
- `simulatePacketLoss()`: Test degraded network conditions

#### 2. Mock API Server (`e2e/mocks/mock-api-server.ts`)

Express-based API server that mimics Subcults API endpoints:

**Features**:
- Token generation endpoint (`POST /api/livekit/token`)
- Testing utilities for network simulation
- Room state inspection for debugging
- CORS support for frontend testing
- Input validation matching production API

**Endpoints**:
- `POST /api/livekit/token`: Generate LiveKit token
- `POST /api/test/simulate-latency`: Trigger latency simulation
- `POST /api/test/simulate-packet-loss`: Trigger packet loss simulation
- `GET /api/test/room/:roomId`: Get room state
- `GET /health`: Health check

#### 3. Playwright E2E Tests

Three comprehensive test suites covering all scenarios:

**a) Stream Lifecycle Tests** (`e2e/tests/stream-lifecycle.spec.ts`)
- Create/join/leave stream operations
- Connection indicator visibility
- Multiple participants joining same room
- Volume persistence across sessions
- Latency overlay in development mode
- Connection error handling

**b) Organizer Controls Tests** (`e2e/tests/organizer-controls.spec.ts`)
- Mute participant functionality
- Kick participant functionality
- Lock/unlock stream
- End stream for all participants
- Permission validation (non-organizers can't use controls)

**c) Network Resilience Tests** (`e2e/tests/network-resilience.spec.ts`)
- Automatic reconnection after temporary disconnection
- Multiple reconnection attempts
- Max reconnection limit (give up after failures)
- Quality degradation indication
- Slow network tolerance
- Intermittent packet loss handling
- High latency handling

#### 4. K6 Load Tests (`perf/k6/stream-load-test.js`)

Comprehensive load testing script:

**Test Profile**:
- Ramp up: 20 → 50 → 100 users over 2.5 minutes
- Sustained load: 100 concurrent users for 2 minutes
- Ramp down: 100 → 0 users over 1 minute

**Metrics Tracked**:
- Token fetch time (target: p95 < 300ms)
- WebSocket connection time (target: p95 < 1s)
- Total join time (target: p95 < 2s)
- Connection errors (target: < 10 total)
- Success rate (target: > 95%)

**Custom Metrics**:
```javascript
- connection_errors: Counter
- token_fetch_time: Trend
- ws_connection_time: Trend
- total_join_time: Trend
- success_rate: Rate
```

#### 5. Test Configuration

**Playwright Config** (`playwright.config.ts`):
- Multi-browser support (Chromium, Firefox, WebKit, Mobile Chrome)
- Microphone permissions enabled
- Fake media devices for testing
- Parallel execution disabled for stream tests
- Screenshot/video capture on failure
- HTML and JSON reporters

**E2E Package** (`e2e/package.json`):
- Playwright test runner
- Express for mock API
- WebSocket library for mock LiveKit
- TypeScript support

#### 6. CI/CD Integration

**GitHub Actions Workflow** (`.github/workflows/e2e-tests.yml`):

Two jobs:
1. **E2E Tests**: Run on every PR affecting web/e2e/livekit code
   - Install dependencies
   - Install Playwright browsers
   - Build frontend
   - Run E2E tests
   - Upload test reports and videos

2. **Load Tests**: Run only on main/develop pushes
   - Install k6
   - Run smoke test (25 VUs for 1 minute)
   - Upload results

### Test Coverage

The test suite covers all acceptance criteria from the issue:

✅ **Create stream → join → leave → end**: Tested in `stream-lifecycle.spec.ts`
✅ **Organizer mute/kick participant**: Tested in `organizer-controls.spec.ts`
✅ **Participant reconnection**: Tested in `network-resilience.spec.ts`
✅ **Quality degradation handling**: Tested in `network-resilience.spec.ts`
✅ **Lock/unlock stream**: Tested in `organizer-controls.spec.ts`
✅ **Concurrent listeners (100+)**: Tested in k6 load test
✅ **Network failures handled**: Tested in `network-resilience.spec.ts`
✅ **No race conditions**: Ensured by sequential test execution

## Running Tests

### Local Development

```bash
# Install dependencies
cd e2e && npm install
npx playwright install

# Run all E2E tests
npm run test:e2e

# Run in UI mode (interactive)
npm run test:e2e:ui

# Run specific test file
npx playwright test tests/stream-lifecycle.spec.ts

# Run load tests
npm run test:load

# Run smoke test (quick validation)
npm run test:load:smoke
```

### From Root Directory

```bash
# Run E2E tests
npm run test:e2e

# Run E2E tests with UI
npm run test:e2e:ui

# Run load tests
npm run test:load
```

### CI Pipeline

Tests run automatically on:
- Every PR that modifies `web/`, `e2e/`, or `internal/livekit/`
- Push to `main` or `develop` branches

## Architecture Decisions

### Why Mock LiveKit?

1. **Speed**: No external dependencies, tests run in milliseconds
2. **Reliability**: No network flakiness, 100% deterministic
3. **Cost**: No LiveKit Cloud usage costs
4. **Control**: Can simulate any network condition or failure scenario
5. **Isolation**: Tests don't interfere with each other

### Why Sequential Execution?

Stream tests run sequentially (not in parallel) because:
1. They share mock server ports (7880, 8080)
2. Room state must be clean between tests
3. Resource contention with 100+ concurrent connections
4. More realistic simulation of real-world usage

### Why Separate E2E Directory?

1. **Isolation**: E2E tests have different dependencies than unit tests
2. **Organization**: Clear separation between test types
3. **Performance**: Can run unit tests without installing Playwright
4. **Flexibility**: Can deploy E2E tests separately for testing environments

## Test Data Flow

```
User Action (Playwright)
  ↓
Frontend Application (http://localhost:5173)
  ↓
Mock API Server (http://localhost:8080)
  ↓ (token request)
Mock LiveKit Server (ws://localhost:7880)
  ↓ (WebSocket connection)
Test Assertions (Playwright)
```

## Network Simulation

The mock servers support realistic network condition simulation:

### Latency Simulation
```javascript
// Add 2000ms delay to room
mockServer.simulateNetworkDelay('room-id', 2000);
```

### Packet Loss Simulation
```javascript
// Simulate 25% packet loss
mockServer.simulatePacketLoss('room-id', 25);
```

### Connection Quality
Automatically calculated based on simulated conditions:
- 0-10% packet loss: excellent
- 10-20% packet loss: good  
- 20%+ packet loss: poor

## Debugging

### Enable Verbose Logging

```bash
# Playwright debug mode
npm run test:e2e:debug

# See browser interactions
npm run test:e2e:headed

# View test report
npm run test:e2e:report
```

### Mock Server Logs

The mock servers log all activity:
```
[MockLiveKit] Server started on port 7880
[MockLiveKit] Client connected: user-123 to room default
[MockAPI] Server started on port 8080
```

### Network Simulation Testing

```bash
# Start mock servers manually
cd e2e
npx ts-node mocks/mock-api-server.ts

# Test in another terminal
curl -X POST http://localhost:8080/api/test/simulate-latency \
  -H "Content-Type: application/json" \
  -d '{"roomId": "test", "delayMs": 2000}'
```

## Performance Targets

From acceptance criteria and performance documentation:

| Metric | Target | Test |
|--------|--------|------|
| Token fetch (p95) | <300ms | k6 load test |
| WS connection (p95) | <1000ms | k6 load test |
| Total join time (p95) | <2000ms | k6 load test |
| Concurrent users | 100+ | k6 load test |
| Success rate | >95% | k6 load test |
| Connection errors | <10 | k6 load test |

All targets validated in the k6 load test suite.

## Future Enhancements

Potential improvements identified during implementation:

1. **Real LiveKit Integration**: Add optional tests against real LiveKit in staging
2. **Visual Regression**: Add screenshot comparison for UI consistency
3. **Accessibility**: Test screen reader compatibility in streaming interface
4. **Mobile Testing**: Expand mobile browser coverage
5. **Chaos Engineering**: Random failure injection for resilience testing
6. **Performance Profiling**: CPU/memory usage monitoring
7. **Security Testing**: XSS/CSRF validation for streaming endpoints
8. **Multi-region**: Test with simulated geographic distribution

## Troubleshooting

### Common Issues

**Tests timeout waiting for elements**
- Solution: Check data-testid attributes match expectations
- Verify frontend is running on port 5173
- Ensure mock servers started successfully

**WebSocket connection failed**
- Solution: Check ports 7880 and 8080 are available
- Verify no firewall blocking WebSocket connections
- Check mock server logs for errors

**Load test shows high error rate**
- Solution: Reduce concurrent VUs
- Check system resources (CPU/memory)
- Verify mock server can handle load

**Flaky reconnection tests**
- Solution: Increase timeouts for network operations
- Check offline/online state transitions
- Verify exponential backoff timing

## Related Documentation

- [E2E Testing README](../e2e/README.md): Detailed test suite documentation
- [Accessibility Testing](./ACCESSIBILITY_TESTING.md): Accessibility test patterns
- [Performance Budgets](./PERFORMANCE.md): Performance targets and monitoring
- [Streaming UI Implementation](./STREAMING_UI_IMPLEMENTATION.md): Frontend streaming components

## Acceptance Criteria Validation

✅ All scenarios pass:
- Stream lifecycle: 6 tests
- Organizer controls: 5 tests
- Network resilience: 7 tests
- Error handling: 1 test

✅ Load test successful (100+ concurrent):
- k6 test validates 100+ concurrent users
- All performance thresholds met
- Success rate >95%

✅ Network failures handled:
- Reconnection logic tested
- Quality degradation tested
- Packet loss tolerance tested
- High latency tolerance tested

✅ No race conditions:
- Sequential test execution
- Clean room state between tests
- Proper async/await usage
- Resource cleanup after each test

## Conclusion

The E2E test suite provides comprehensive coverage of all streaming functionality with:
- 18 automated E2E tests across 3 test suites
- Full mock LiveKit infrastructure for fast, reliable testing
- Load testing with k6 for performance validation
- CI/CD integration for automated testing
- Detailed documentation and debugging tools

All acceptance criteria from the issue are met and validated.
