// K6: Simple Burst Test (matches manual testing)

import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  vus: 1,
  iterations: 1, // Run once
};

export default function () {
  console.log('🧪 Burst Test: Sending 15 rapid requests');
  
  let allowed = 0;
  let denied = 0;

  for (let i = 1; i <= 15; i++) {
    const res = http.get(`${BASE_URL}/test/get`);
    
    if (res.status === 200) {
      allowed++;
    } else if (res.status === 429) {
      denied++;
    }

    if (i === 10 || i === 11 || i === 15) {
      console.log(`Request ${i}: ${res.status} | Remaining: ${res.headers['X-Ratelimit-Remaining']}`);
    }
  }

  console.log(`\n✅ Results: ${allowed} allowed, ${denied} denied`);
  console.log(`Expected: 10-11 allowed, 4-5 denied (Token Bucket)`);
  
  check({ allowed, denied }, {
    'burst: 10-11 requests allowed': () => allowed >= 10 && allowed <= 11,
    'burst: 4-5 requests denied': () => denied >= 4 && denied <= 5,
  });
}