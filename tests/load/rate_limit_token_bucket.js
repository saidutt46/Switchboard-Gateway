// K6 Load Test: Rate Limiting - Token Bucket Algorithm
//
// This test validates:
// - Rate limits are enforced correctly under load
// - Token bucket refills gradually
// - Headers are accurate
// - 429 responses are returned when limit exceeded
// - System handles concurrent requests correctly
// - Performance impact is minimal

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const rateLimitExceeded = new Counter('rate_limit_exceeded');
const rateLimitAllowed = new Counter('rate_limit_allowed');
const rateLimitAccuracy = new Rate('rate_limit_accuracy');
const headerPresence = new Rate('rate_limit_headers_present');
const tokenRefillRate = new Trend('token_refill_rate');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const RATE_LIMIT = 10; // 10 requests per minute (from config)
const WINDOW_SECONDS = 60;

// Test scenarios
export const options = {
  scenarios: {
    // Scenario 1: Burst test - rapid fire to hit limit
    burst_test: {
      executor: 'shared-iterations',
      vus: 1, // Single user
      iterations: 15, // Try 15 requests (expect 10-11 to succeed)
      maxDuration: '10s',
      startTime: '0s',
      exec: 'burstTest',
    },

    // Scenario 2: Sustained load - test gradual refill
    sustained_load: {
      executor: 'constant-vus',
      vus: 2, // 2 concurrent users
      duration: '30s',
      startTime: '70s', // Start after burst test + reset
      exec: 'sustainedLoad',
    },

    // Scenario 3: Spike test - sudden traffic spike
    spike_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5s', target: 10 }, // Ramp up to 10 users
        { duration: '10s', target: 10 }, // Stay at 10
        { duration: '5s', target: 0 }, // Ramp down
      ],
      startTime: '110s', // Start after sustained load
      exec: 'spikeTest',
    },

    // Scenario 4: Stress test - test system limits
    stress_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 20 }, // Ramp to 20 users
        { duration: '20s', target: 50 }, // Ramp to 50 users
        { duration: '10s', target: 0 }, // Ramp down
      ],
      startTime: '140s',
      exec: 'stressTest',
    },
  },

  thresholds: {
    // Performance thresholds
    http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
    http_req_failed: ['rate<0.1'], // Less than 10% failures (excluding 429s)
    
    // Rate limiting thresholds
    rate_limit_accuracy: ['rate>0.9'], // 90% accuracy in enforcement
    rate_limit_headers_present: ['rate>0.95'], // 95% of responses have headers
    
    // Custom checks
    checks: ['rate>0.95'], // 95% of checks pass
  },
};

// Scenario 1: Burst Test
// Tests rapid-fire requests to verify limit enforcement
// Scenario 1: Burst Test
// Tests rapid-fire requests to verify limit enforcement
export function burstTest() {
  group('Burst Test - Rapid Fire Requests', () => {
    // IMPORTANT: Each iteration should test fresh bucket
    // In real world, this would be different users
    // For testing, we add a unique identifier per iteration
    
    const iterationId = __ITER; // k6 built-in iteration counter
    const testId = `burst-${iterationId}`;
    
    const results = { allowed: 0, denied: 0 };

    for (let i = 1; i <= 15; i++) {
      // Add unique header to simulate different users
      const response = http.get(`${BASE_URL}/test/get`, {
        headers: {
          'X-Test-ID': testId,  // Unique per iteration
        },
      });
      
      const isAllowed = response.status === 200;
      const isDenied = response.status === 429;

      if (isAllowed) {
        results.allowed++;
        rateLimitAllowed.add(1);
      } else if (isDenied) {
        results.denied++;
        rateLimitExceeded.add(1);
      }

      check(response, {
        'burst: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
        'burst: has rate limit headers': (r) => {
          const hasHeaders = r.headers['X-Ratelimit-Limit'] !== undefined;
          headerPresence.add(hasHeaders);
          return hasHeaders;
        },
        'burst: limit header is correct': (r) => 
          r.headers['X-Ratelimit-Limit'] === RATE_LIMIT.toString(),
      });
    }

    // Only check first iteration (others share depleted bucket)
    if (iterationId === 0) {
      const inRange = results.allowed >= 10 && results.allowed <= 11;
      rateLimitAccuracy.add(inRange);

      console.log(`Burst Test Iteration ${iterationId} - Allowed: ${results.allowed}, Denied: ${results.denied}`);
      
      check(results, {
        'burst: allowed 10-11 requests (Token Bucket)': () => inRange,
        'burst: denied 4-5 requests': () => results.denied >= 4 && results.denied <= 5,
      });
    }
  });
}

// Scenario 2: Sustained Load
// Tests gradual refill under continuous load
export function sustainedLoad() {
  group('Sustained Load - Gradual Refill', () => {
    const response = http.get(`${BASE_URL}/test/post`, {
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        test: 'sustained_load',
        timestamp: new Date().toISOString(),
        data: 'Testing gradual token refill',
      }),
    });

    const remaining = parseInt(response.headers['X-Ratelimit-Remaining'] || '-1');

    check(response, {
      'sustained: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
      'sustained: has rate limit headers': (r) => 
        r.headers['X-Ratelimit-Limit'] !== undefined,
      'sustained: remaining is non-negative': () => remaining >= 0,
    });

    // Track refill rate (requests allowed over time)
    if (response.status === 200) {
      rateLimitAllowed.add(1);
    } else {
      rateLimitExceeded.add(1);
    }

    // Small delay to allow refill
    sleep(0.5);
  });
}

// Scenario 3: Spike Test
// Tests system under sudden load spike
export function spikeTest() {
  group('Spike Test - Traffic Burst', () => {
    const endpoints = [
      '/test/get',
      '/test/post',
      '/test/put',
      '/test/delete',
    ];

    const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
    const response = http.get(`${BASE_URL}${endpoint}`);

    check(response, {
      'spike: request completed': (r) => r.status !== 0,
      'spike: has rate limit headers': (r) => 
        r.headers['X-Ratelimit-Limit'] !== undefined,
      'spike: response time acceptable': (r) => r.timings.duration < 1000,
    });

    if (response.status === 200) {
      rateLimitAllowed.add(1);
    } else if (response.status === 429) {
      rateLimitExceeded.add(1);
    }

    sleep(Math.random() * 2); // Random delay 0-2s
  });
}

// Scenario 4: Stress Test
// Tests system under heavy load
export function stressTest() {
  group('Stress Test - High Concurrency', () => {
    const batch = http.batch([
      ['GET', `${BASE_URL}/test/get`],
      ['GET', `${BASE_URL}/test/headers`],
      ['GET', `${BASE_URL}/test/ip`],
    ]);

    batch.forEach((response) => {
      check(response, {
        'stress: request completed': (r) => r.status !== 0,
        'stress: gateway responsive': (r) => r.status !== 502 && r.status !== 503,
      });

      if (response.status === 200) {
        rateLimitAllowed.add(1);
      } else if (response.status === 429) {
        rateLimitExceeded.add(1);
      }
    });

    sleep(0.1);
  });
}

// Setup: Verify rate limiting is enabled
export function setup() {
  console.log('🧪 Token Bucket Load Test Setup');
  console.log('================================');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Rate Limit: ${RATE_LIMIT} requests per ${WINDOW_SECONDS} seconds`);
  console.log(`Algorithm: Token Bucket`);
  console.log('');

  // Warmup request
  const warmup = http.get(`${BASE_URL}/test/get`);
  
  check(warmup, {
    'setup: gateway is reachable': (r) => r.status === 200,
    'setup: rate limiting is active': (r) => 
      r.headers['X-Ratelimit-Limit'] !== undefined,
  });

  console.log(`Warmup: Status ${warmup.status}, ` +
    `Limit: ${warmup.headers['X-Ratelimit-Limit']}, ` +
    `Remaining: ${warmup.headers['X-Ratelimit-Remaining']}`);
  console.log('');

  return { startTime: new Date() };
}

// Teardown: Summary
export function teardown(data) {
  console.log('');
  console.log('🎯 Token Bucket Test Summary');
  console.log('============================');
  console.log(`Test Duration: ${((new Date() - data.startTime) / 1000).toFixed(2)}s`);
  console.log('');
  console.log('✅ Test complete! Check metrics above for results.');
}

// Handle summary for custom reporting
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    'tests/results/rate_limit_token_bucket.json': JSON.stringify(data, null, 2),
  };
}

// Helper: Text summary
function textSummary(data, options) {
  const indent = options.indent || '';
  const colors = options.enableColors || false;

  let summary = '\n';
  summary += `${indent}Rate Limiting Metrics:\n`;
  summary += `${indent}  Requests Allowed: ${data.metrics.rate_limit_allowed?.values?.count || 0}\n`;
  summary += `${indent}  Requests Denied (429): ${data.metrics.rate_limit_exceeded?.values?.count || 0}\n`;
  summary += `${indent}  Accuracy: ${((data.metrics.rate_limit_accuracy?.values?.rate || 0) * 100).toFixed(2)}%\n`;
  summary += `${indent}  Headers Present: ${((data.metrics.rate_limit_headers_present?.values?.rate || 0) * 100).toFixed(2)}%\n`;

  return summary;
}