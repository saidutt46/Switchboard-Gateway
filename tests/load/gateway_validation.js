import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const pluginOverhead = new Trend('plugin_overhead_ms');
const corsPreflightRate = new Rate('cors_preflight_success_rate');
const proxySuccessRate = new Rate('proxy_success_rate');
const requestsWithPlugins = new Counter('requests_with_plugins');
const requestsWithoutPlugins = new Counter('requests_without_plugins');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const BACKEND_URL = __ENV.BACKEND_URL || 'http://localhost:8081';

// Test stages - moderate load to avoid port exhaustion
export const options = {
  stages: [
    // Warm up
    { duration: '30s', target: 20 },   // Ramp up to 20 users
    
    // Sustained load
    { duration: '1m', target: 50 },    // Increase to 50 users
    { duration: '2m', target: 50 },    // Hold at 50 users
    
    // Peak load (moderate)
    { duration: '30s', target: 100 },  // Increase to 100 users
    { duration: '1m', target: 100 },   // Hold at 100 users
    
    // Cool down
    { duration: '30s', target: 0 },    // Ramp down to 0
  ],
  
  thresholds: {
    // Overall success rate should be > 95%
    'http_req_failed': ['rate<0.05'],
    
    // 95th percentile response time should be < 200ms (excluding backend delay)
    'http_req_duration': ['p(95)<200'],
    
    // Proxy success rate should be > 98%
    'proxy_success_rate': ['rate>0.98'],
    
    // CORS preflight should be > 99%
    'cors_preflight_success_rate': ['rate>0.99'],
  },
};

// Test scenarios
export default function() {
  // Sleep between requests to avoid port exhaustion
  // This simulates realistic user behavior
  const thinkTime = Math.random() * 1000 + 500; // 500-1500ms
  
  // Randomly choose a test scenario
  const scenario = Math.random();
  
  if (scenario < 0.3) {
    testBasicProxying();
  } else if (scenario < 0.5) {
    testCORS();
  } else if (scenario < 0.7) {
    testDifferentMethods();
  } else if (scenario < 0.85) {
    testWithHeaders();
  } else {
    testPathParameters();
  }
  
  // Think time between requests (prevents port exhaustion)
  sleep(thinkTime / 1000);
}

// Test 1: Basic GET request proxying
function testBasicProxying() {
  const startTime = new Date();
  
  const res = http.get(`${BASE_URL}/get`, {
    tags: { scenario: 'basic_proxying' },
  });
  
  const passed = check(res, {
    'status is 200': (r) => r.status === 200,
    'response has json': (r) => r.headers['Content-Type']?.includes('application/json'),
    'response has X-Request-ID': (r) => r.headers['X-Request-Id'] !== undefined,
    'response has X-Upstream-Latency': (r) => r.headers['X-Upstream-Latency'] !== undefined,
    'body contains headers': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.headers !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  if (passed) {
    proxySuccessRate.add(1);
    requestsWithPlugins.add(1);
  } else {
    proxySuccessRate.add(0);
  }
}

// Test 2: CORS preflight
function testCORS() {
  // Preflight request
  const preflightRes = http.options(`${BASE_URL}/get`, null, {
    headers: {
      'Origin': 'https://example.com',
      'Access-Control-Request-Method': 'POST',
    },
    tags: { scenario: 'cors_preflight' },
  });
  
  const preflightPassed = check(preflightRes, {
    'preflight status is 204': (r) => r.status === 204,
    'has CORS allow-origin': (r) => r.headers['Access-Control-Allow-Origin'] !== undefined,
    'has CORS allow-methods': (r) => r.headers['Access-Control-Allow-Methods'] !== undefined,
    'has CORS max-age': (r) => r.headers['Access-Control-Max-Age'] !== undefined,
  });
  
  corsPreflightRate.add(preflightPassed ? 1 : 0);
  
  // Actual request with Origin
  const actualRes = http.get(`${BASE_URL}/get`, {
    headers: {
      'Origin': 'https://example.com',
    },
    tags: { scenario: 'cors_actual' },
  });
  
  check(actualRes, {
    'actual request status is 200': (r) => r.status === 200,
    'has CORS headers': (r) => r.headers['Access-Control-Allow-Origin'] !== undefined,
  });
}

// Test 3: Different HTTP methods
function testDifferentMethods() {
  const method = ['POST', 'PUT', 'PATCH', 'DELETE'][Math.floor(Math.random() * 4)];
  const endpoint = method.toLowerCase();
  
  const payload = JSON.stringify({
    test: 'data',
    timestamp: new Date().toISOString(),
    method: method,
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { scenario: `method_${method}` },
  };
  
  let res;
  switch(method) {
    case 'POST':
      res = http.post(`${BASE_URL}/${endpoint}`, payload, params);
      break;
    case 'PUT':
      res = http.put(`${BASE_URL}/${endpoint}`, payload, params);
      break;
    case 'PATCH':
      res = http.patch(`${BASE_URL}/${endpoint}`, payload, params);
      break;
    case 'DELETE':
      res = http.del(`${BASE_URL}/${endpoint}`, null, params);
      break;
  }
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'method forwarded correctly': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.method === method;
      } catch {
        return false;
      }
    },
  });
}

// Test 4: Custom headers forwarding
function testWithHeaders() {
  const res = http.get(`${BASE_URL}/headers`, {
    headers: {
      'X-Custom-Header': `test-${__VU}-${__ITER}`,
      'Authorization': 'Bearer test-token-12345',
      'X-Request-Source': 'k6-load-test',
    },
    tags: { scenario: 'custom_headers' },
  });
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'custom header forwarded': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.headers['X-Custom-Header'] !== undefined;
      } catch {
        return false;
      }
    },
    'auth header forwarded': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.headers['Authorization'] !== undefined;
      } catch {
        return false;
      }
    },
  });
}

// Test 5: Path parameters (if route exists)
function testPathParameters() {
  const userId = Math.floor(Math.random() * 10000);
  
  // This will 404 if route doesn't exist, but we're testing the gateway behavior
  const res = http.get(`${BASE_URL}/user/${userId}`, {
    tags: { scenario: 'path_parameters' },
    expected_response: 'any'
  });
  
  // Accept both 200 (if route exists) and 404 (if not configured)
  check(res, {
    'status is 200 or 404': (r) => r.status === 200 || r.status === 404,
    'response is valid': (r) => r.body !== undefined,
  });
}

// Setup function - runs once per VU
export function setup() {
  console.log('🚀 Starting Gateway Validation Test');
  console.log(`📍 Gateway URL: ${BASE_URL}`);
  console.log(`📍 Backend URL: ${BACKEND_URL}`);
  
  // Verify gateway is responding
  const healthCheck = http.get(`${BASE_URL}/health`);
  if (healthCheck.status !== 200) {
    throw new Error('Gateway health check failed!');
  }
  
  console.log('✅ Gateway is healthy');
  
  return { startTime: Date.now() };
}

// Teardown function - runs once after all VUs finish
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`\n🏁 Test completed in ${duration.toFixed(2)} seconds`);
}

// Handle summary - custom summary output
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'tests/load/results/gateway_validation_summary.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  let summary = '\n';
  summary += `${indent}📊 Gateway Validation Test Results\n`;
  summary += `${indent}${'='.repeat(50)}\n\n`;
  
  // Scenario breakdown
  summary += `${indent}Scenarios Executed:\n`;
  Object.keys(data.metrics.http_reqs.values).forEach(key => {
    if (key.startsWith('scenario:')) {
      const scenario = key.split(':')[1];
      const count = data.metrics.http_reqs.values[key] || 0;
      summary += `${indent}  - ${scenario}: ${count} requests\n`;
    }
  });
  
  summary += `\n${indent}Overall Metrics:\n`;
  summary += `${indent}  - Total Requests: ${data.metrics.http_reqs.values.count}\n`;
  summary += `${indent}  - Failed Requests: ${data.metrics.http_req_failed.values.fails}\n`;
  summary += `${indent}  - Success Rate: ${(100 - data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%\n`;
  
  summary += `\n${indent}Response Times:\n`;
  summary += `${indent}  - Average: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += `${indent}  - Median: ${data.metrics.http_req_duration.values.med.toFixed(2)}ms\n`;
  summary += `${indent}  - 95th percentile: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `${indent}  - 99th percentile: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
  
  summary += `\n${indent}Gateway-Specific:\n`;
  summary += `${indent}  - Proxy Success Rate: ${(data.metrics.proxy_success_rate.values.rate * 100).toFixed(2)}%\n`;
  summary += `${indent}  - CORS Preflight Success: ${(data.metrics.cors_preflight_success_rate.values.rate * 100).toFixed(2)}%\n`;
  
  return summary;
}