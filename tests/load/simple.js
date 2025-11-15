import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 10 },  // Ramp up
    { duration: '30s', target: 50 },  // Sustained load
    { duration: '10s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'], // 95% of requests under 100ms
    http_req_failed: ['rate<0.01'],   // Less than 1% errors
  },
};

export default function () {
  const endpoints = ['/get', '/anything', '/headers'];
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  
  const res = http.get(`http://localhost:8080${endpoint}`);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'has X-Request-ID': (r) => r.headers['X-Request-Id'] !== undefined,
  });
  
  sleep(0.1);
}