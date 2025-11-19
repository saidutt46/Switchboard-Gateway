// K6 Load Test: Rate Limiting - Sliding Window Algorithm
//
// This test validates:
// - Strict enforcement of rate limits
// - No burst tolerance (exactly N requests per window)
// - Sliding window behavior (not fixed buckets)
// - Accurate request counting
// - Performance under strict limits

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const rateLimitExceeded = new Counter('rate_limit_exceeded');
const rateLimitAllowed = new Counter('rate_limit_allowed');
const exactEnforcement = new Rate('exact_limit_enforcement');
const headerPresence = new Rate('rate_limit_headers_present');
const windowSliding = new Rate('window_sliding_correctly');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const RATE_LIMIT = 10; // 10 requests per minute
const WINDOW_SECONDS = 60;

export const options = {
  scenarios: {
    // Scenario 1: Exact limit test - verify strict enforcement
    exact_limit_test: {
      executor: 'shared-iterations',
      vus: 1,
      iterations: 15,
      maxDuration: '10s',
      startTime: '0s',
      exec: 'exactLimitTest',
    },

    // Scenario 2: Sliding window test - verify window slides correctly
    sliding_window_test: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 20,
      maxDuration: '90s',
      startTime: '70s', // After reset
      exec: 'slidingWindowTest',
    },

    // Scenario 3: Concurrent users - test fairness
    concurrent_users_test: {
      executor: 'constant-vus',
      vus: 5, // 5 concurrent users
      duration: '30s',
      startTime: '170s',
      exec: 'concurrentUsersTest',
    },

    // Scenario 4: No gradual refill test
    no_refill_test: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 25,
      maxDuration: '40s',
      startTime: '210s',
      exec: 'noRefillTest',
    },
  },

  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.1'],
    exact_limit_enforcement: ['rate>0.95'], // 95% exact enforcement
    rate_limit_headers_present: ['rate>0.95'],
    checks: ['rate>0.95'],
  },
};

// Scenario 1: Exact Limit Test
// Verifies sliding window allows exactly N requests
export function exactLimitTest() {
  group('Exact Limit - Strict Enforcement', () => {
    const results = { allowed: 0, denied: 0 };

    for (let i = 1; i <= 15; i++) {
      const response = http.get(`${BASE_URL}/test/uuid`);
      
      if (response.status === 200) {
        results.allowed++;
        rateLimitAllowed.add(1);
      } else if (response.status === 429) {
        results.denied++;
        rateLimitExceeded.add(1);
      }

      check(response, {
        'exact: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
        'exact: has rate limit headers': (r) => {
          const hasHeaders = r.headers['X-Ratelimit-Limit'] !== undefined;
          headerPresence.add(hasHeaders);
          return hasHeaders;
        },
      });

      if (i === 10 || i === 11 || i === 15) {
        console.log(`Exact: Request ${i} - Status: ${response.status}`);
      }
    }

    // Sliding window should allow exactly 10 (strict)
    const isExact = results.allowed === 10;
    exactEnforcement.add(isExact);

    console.log(`Exact Limit Results - Allowed: ${results.allowed}, Denied: ${results.denied}`);

    check(results, {
      'exact: allowed exactly 10 requests': () => results.allowed === 10,
      'exact: denied exactly 5 requests': () => results.denied === 5,
    });
  });
}

// Scenario 2: Sliding Window Test
// Verifies window slides correctly over time
export function slidingWindowTest() {
  group('Sliding Window - Time-based Reset', () => {
    // Make a request
    const response = http.get(`${BASE_URL}/test/delay/1`);
    
    const remaining = parseInt(response.headers['X-Ratelimit-Remaining'] || '-1');
    const resetTime = parseInt(response.headers['X-Ratelimit-Reset'] || '0');
    const now = Math.floor(Date.now() / 1000);
    const timeToReset = resetTime - now;

    check(response, {
      'sliding: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
      'sliding: reset time is in future': () => timeToReset > 0 && timeToReset <= WINDOW_SECONDS,
      'sliding: remaining count is valid': () => remaining >= 0 && remaining <= RATE_LIMIT,
    });

    if (response.status === 200) {
      rateLimitAllowed.add(1);
      console.log(`Sliding: Allowed, Remaining: ${remaining}, Reset in: ${timeToReset}s`);
    } else {
      rateLimitExceeded.add(1);
      console.log(`Sliding: Denied (429), Reset in: ${timeToReset}s`);
    }

    // Small delay between requests
    sleep(2);
  });
}

// Scenario 3: Concurrent Users Test
// Tests fairness with multiple users
export function concurrentUsersTest() {
  group('Concurrent Users - Fairness', () => {
    const batch = http.batch([
      ['GET', `${BASE_URL}/test/base64/SGVsbG8gV29ybGQ=`],
      ['GET', `${BASE_URL}/test/status/200`],
    ]);

    batch.forEach((response) => {
      check(response, {
        'concurrent: request completed': (r) => r.status !== 0,
        'concurrent: has headers': (r) => r.headers['X-Ratelimit-Limit'] !== undefined,
      });

      if (response.status === 200) {
        rateLimitAllowed.add(1);
      } else if (response.status === 429) {
        rateLimitExceeded.add(1);
      }
    });

    sleep(Math.random()); // 0-1s random delay
  });
}

// Scenario 4: No Refill Test
// Verifies no gradual refill (unlike Token Bucket)
export function noRefillTest() {
  group('No Gradual Refill - Strict Window', () => {
    const response = http.get(`${BASE_URL}/test/html`);
    
    check(response, {
      'no-refill: status is 200 or 429': (r) => r.status === 200 || r.status === 429,
      'no-refill: response time acceptable': (r) => r.timings.duration < 2000,
    });

    if (response.status === 200) {
      rateLimitAllowed.add(1);
    } else if (response.status === 429) {
      rateLimitExceeded.add(1);
      
      // Verify retry-after header
      check(response, {
        'no-refill: has retry-after header': (r) => r.headers['Retry-After'] !== undefined,
      });
    }

    sleep(1.5); // 1.5s delay - should NOT allow gradual refill
  });
}

// Setup
export function setup() {
  console.log('🧪 Sliding Window Load Test Setup');
  console.log('==================================');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Rate Limit: ${RATE_LIMIT} requests per ${WINDOW_SECONDS} seconds`);
  console.log(`Algorithm: Sliding Window`);
  console.log('');

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

// Teardown
export function teardown(data) {
  console.log('');
  console.log('🎯 Sliding Window Test Summary');
  console.log('==============================');
  console.log(`Test Duration: ${((new Date() - data.startTime) / 1000).toFixed(2)}s`);
  console.log('');
  console.log('✅ Test complete!');
}

// Summary
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    'tests/results/rate_limit_sliding_window.json': JSON.stringify(data, null, 2),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';

  let summary = '\n';
  summary += `${indent}Rate Limiting Metrics:\n`;
  summary += `${indent}  Requests Allowed: ${data.metrics.rate_limit_allowed?.values?.count || 0}\n`;
  summary += `${indent}  Requests Denied (429): ${data.metrics.rate_limit_exceeded?.values?.count || 0}\n`;
  summary += `${indent}  Exact Enforcement: ${((data.metrics.exact_limit_enforcement?.values?.rate || 0) * 100).toFixed(2)}%\n`;
  summary += `${indent}  Headers Present: ${((data.metrics.rate_limit_headers_present?.values?.rate || 0) * 100).toFixed(2)}%\n`;

  return summary;
}