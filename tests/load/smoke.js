import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Ultra-light smoke test - just verify gateway works
export const options = {
  vus: 5,
  duration: '30s',
  
  thresholds: {
    'http_req_failed': ['rate<0.01'],
    'http_req_duration': ['p(95)<100'],
  },
};

export default function() {
  // Test basic proxying
  const res1 = http.get(`${BASE_URL}/get`);
  check(res1, { 'GET status is 200': (r) => r.status === 200 });
  
  sleep(1);
  
  // Test POST
  const res2 = http.post(`${BASE_URL}/post`, JSON.stringify({ test: 'data' }), {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res2, { 'POST status is 200': (r) => r.status === 200 });
  
  sleep(1);
  
  // Test CORS
  const res3 = http.options(`${BASE_URL}/get`, null, {
    headers: {
      'Origin': 'https://example.com',
      'Access-Control-Request-Method': 'POST',
    },
  });
  check(res3, { 'OPTIONS status is 204': (r) => r.status === 204 });
  
  sleep(1);
}