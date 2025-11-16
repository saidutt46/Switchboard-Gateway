import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Specific test for connection pooling
export const options = {
  scenarios: {
    // Test connection reuse
    sustained_load: {
      executor: 'constant-vus',
      vus: 50,
      duration: '2m',
    },
  },
  
  thresholds: {
    'http_req_duration': ['p(95)<150'],
    'http_req_failed': ['rate<0.01'],
  },
};

export default function() {
  const res = http.get(`${BASE_URL}/get`);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'has keep-alive': (r) => r.headers['Connection']?.toLowerCase() !== 'close',
  });
  
  // Small delay to simulate realistic usage
  sleep(0.1); // 100ms between requests
}