/**
 * K6 Load Test: Stream Concurrent Listeners
 * 
 * Tests the streaming infrastructure with 100+ concurrent listeners
 * to verify performance under load.
 * 
 * Run with:
 *   k6 run perf/k6/stream-load-test.js
 * 
 * Or with custom parameters:
 *   k6 run --vus 100 --duration 5m perf/k6/stream-load-test.js
 */

import http from 'k6/http';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const connectionErrors = new Counter('connection_errors');
const tokenFetchTime = new Trend('token_fetch_time');
const wsConnectionTime = new Trend('ws_connection_time');
const totalJoinTime = new Trend('total_join_time');
const successRate = new Rate('success_rate');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 20 },  // Ramp up to 20 users
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '2m', target: 100 },  // Stay at 100 users
    { duration: '30s', target: 50 },  // Ramp down to 50
    { duration: '30s', target: 0 },   // Ramp down to 0
  ],
  thresholds: {
    'connection_errors': ['count<10'], // Less than 10 connection errors
    'token_fetch_time': ['p(95)<300'], // 95% of token fetches under 300ms
    'ws_connection_time': ['p(95)<1000'], // 95% of WS connections under 1s
    'total_join_time': ['p(95)<2000'], // 95% of total joins under 2s
    'success_rate': ['rate>0.95'], // 95% success rate
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const WS_URL = __ENV.WS_URL || 'ws://localhost:7880';
const ROOM_ID = __ENV.ROOM_ID || 'load-test-room';

export default function () {
  const userId = `user-${__VU}-${__ITER}`;
  const joinStartTime = new Date();
  
  // Step 1: Fetch LiveKit token (using production API contract)
  const tokenStartTime = new Date();
  const tokenResponse = http.post(`${BASE_URL}/api/livekit/token`, JSON.stringify({
    room_id: ROOM_ID,
    scene_id: `test-scene-${__VU}`,
  }), {
    headers: {
      'Content-Type': 'application/json',
    },
  });
  
  const tokenFetchDuration = new Date() - tokenStartTime;
  tokenFetchTime.add(tokenFetchDuration);
  
  const tokenSuccess = check(tokenResponse, {
    'token fetch status is 200': (r) => r.status === 200,
    'token is returned': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token && body.token.length > 0;
      } catch (e) {
        return false;
      }
    },
    'expires_at is returned': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.expires_at && body.expires_at.length > 0;
      } catch (e) {
        return false;
      }
    },
  });
  
  if (!tokenSuccess) {
    connectionErrors.add(1);
    successRate.add(0);
    sleep(1);
    return;
  }
  
  // Step 2: Connect to WebSocket
  const wsStartTime = new Date();
  const wsUrl = `${WS_URL}?room=${ROOM_ID}&identity=${userId}`;
  
  let wsConnected = false;
  let participantJoined = false;
  let messageCount = 0;
  
  const wsResponse = ws.connect(wsUrl, {}, function (socket) {
    socket.on('open', () => {
      wsConnected = true;
      const wsConnectionDuration = new Date() - wsStartTime;
      wsConnectionTime.add(wsConnectionDuration);
    });
    
    socket.on('message', (data) => {
      messageCount++;
      
      try {
        const message = JSON.parse(data);
        
        // Check for room state message (confirms join)
        if (message.type === 'room_state') {
          participantJoined = true;
          
          const totalJoinDuration = new Date() - joinStartTime;
          totalJoinTime.add(totalJoinDuration);
          
          // Log performance for monitoring
          if (totalJoinDuration > 2000) {
            console.log(`WARNING: Slow join time for ${userId}: ${totalJoinDuration}ms`);
          }
        }
      } catch (e) {
        console.error(`Failed to parse message: ${e}`);
      }
    });
    
    socket.on('error', (e) => {
      console.error(`WebSocket error for ${userId}: ${e}`);
      connectionErrors.add(1);
    });
    
    socket.on('close', () => {
      // Connection closed
    });
    
    // Stay connected for 10-30 seconds (simulate real user behavior)
    const stayDuration = Math.random() * 20 + 10;
    socket.setTimeout(() => {
      socket.close();
    }, stayDuration * 1000);
  });
  
  // Record success/failure
  const overallSuccess = wsConnected && participantJoined;
  successRate.add(overallSuccess ? 1 : 0);
  
  if (!overallSuccess) {
    connectionErrors.add(1);
  }
  
  // Check WebSocket connection
  check(wsResponse, {
    'websocket connected': () => wsConnected,
    'participant joined room': () => participantJoined,
    'received messages': () => messageCount > 0,
  });
  
  // Brief pause before next iteration
  sleep(1);
}

export function handleSummary(data) {
  return {
    'stream-load-test-results.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: ' ' }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  
  let summary = '\n' + indent + '=== Load Test Summary ===\n\n';
  
  summary += indent + `Total Requests: ${data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0}\n`;
  summary += indent + `Total Errors: ${data.metrics.connection_errors ? data.metrics.connection_errors.values.count : 0}\n`;
  summary += indent + `Success Rate: ${data.metrics.success_rate ? (data.metrics.success_rate.values.rate * 100).toFixed(2) : 0}%\n\n`;
  
  summary += indent + 'Latency Metrics:\n';
  summary += indent + `  Token Fetch (p95): ${data.metrics.token_fetch_time ? data.metrics.token_fetch_time.values['p(95)'].toFixed(2) : 0}ms\n`;
  summary += indent + `  WS Connection (p95): ${data.metrics.ws_connection_time ? data.metrics.ws_connection_time.values['p(95)'].toFixed(2) : 0}ms\n`;
  summary += indent + `  Total Join Time (p95): ${data.metrics.total_join_time ? data.metrics.total_join_time.values['p(95)'].toFixed(2) : 0}ms\n\n`;
  
  // Check if thresholds passed
  const thresholdsPassed = !data.metrics.connection_errors || data.metrics.connection_errors.values.count < 10;
  summary += indent + `Thresholds: ${thresholdsPassed ? '✓ PASSED' : '✗ FAILED'}\n`;
  
  return summary;
}
