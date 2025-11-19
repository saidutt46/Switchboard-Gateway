// K6 Load Test: Rate Limiting - Realistic Scenario
//
// This test simulates real-world usage:
// - Multiple users making requests
// - Varied request patterns
// - Different endpoints

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Gauge } from 'k6/metrics';

// Custom metrics
const rateLimitHits = new Counter('rate_limit_429s');
const successfulRequests = new Counter('successful_requests');
const remainingTokens = new Gauge('remaining_tokens');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  scenarios: {
    // Realistic user behavior: gradual ramp-up
    normal_usage: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 5 },   // Ramp to 5 users
        { duration: '1m', target: 5 },    // Stay at 5 users
        { duration: '30s', target: 10 },  // Ramp to 10 users
        { duration: '1m', target: 10 },   // Stay at 10 users
        { duration: '30s', target: 0 },   // Ramp down
      ],
    },
  },

  thresholds: {
    http_req_duration: ['p(95)<1000'],  // 95% under 1s
    // Don't fail on 429s - they're expected with rate limiting!
    // http_req_failed: ['rate<0.5'],  // ← Removed
    checks: ['rate>0.90'],               // 90% of checks pass
    successful_requests: ['count>0'],    // At least some requests succeed
    rate_limit_429s: ['count>0'],        // Rate limiting is active},
  },
};

export default function () {
  // Simulate different endpoints
  const endpoints = [
    '/test/get',
    '/test/post',
    '/test/uuid',
    '/test/headers',
  ];

  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  const response = http.get(`${BASE_URL}${endpoint}`);

  // Track results
  if (response.status === 200) {
    successfulRequests.add(1);
  } else if (response.status === 429) {
    rateLimitHits.add(1);
  }

  // Track remaining tokens
  const remaining = parseInt(response.headers['X-Ratelimit-Remaining'] || '0');
  remainingTokens.add(remaining);

  // Check response
  const checks = check(response, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'has rate limit headers': (r) => r.headers['X-Ratelimit-Limit'] !== undefined,
    'response time OK': (r) => r.timings.duration < 2000,
  });

  // Random delay between requests (1-5 seconds)
  // Simulates real user behavior
  sleep(Math.random() * 4 + 1);
}

export function handleSummary(data) {
  const total = (data.metrics.successful_requests?.values?.count || 0) + 
                (data.metrics.rate_limit_429s?.values?.count || 0);
  const allowed = data.metrics.successful_requests?.values?.count || 0;
  const denied = data.metrics.rate_limit_429s?.values?.count || 0;
  const allowedPercent = total > 0 ? (allowed / total * 100).toFixed(2) : 0;

  console.log('\n📊 Realistic Load Test Results');
  console.log('==============================');
  console.log(`Total Requests: ${total}`);
  console.log(`✅ Allowed (200): ${allowed} (${allowedPercent}%)`);
  console.log(`❌ Rate Limited (429): ${denied} (${(100 - allowedPercent).toFixed(2)}%)`);
  console.log('');

  return {
    'stdout': textSummary(data, { indent: '  ', enableColors: true }),
    'tests/results/rate_limit_realistic.json': JSON.stringify(data, null, 2),
  };
}

function textSummary(data, options) {
  return '\n✅ Test complete!\n';
}