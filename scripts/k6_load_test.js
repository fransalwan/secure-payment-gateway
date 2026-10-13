import http from 'k6/http';
import { check, sleep } from 'k6';
import crypto from 'k6/crypto';

export const options = {
  stages: [
    { duration: '10s', target: 50 },  // Ramp-up to 50 concurrent users
    { duration: '30s', target: 200 }, // Peak load 200 users
    { duration: '10s', target: 0 },   // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<50', 'p(99)<100'], // Latency targets for high performance
    http_req_failed: ['rate<0.01'],              // Less than 1% errors
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const HMAC_SECRET = 'spg-secret-hmac-key-for-signatures';

export default function () {
  // Test Health endpoint
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'status is 200': (r) => r.status === 200,
    'latency is under 30ms': (r) => r.timings.duration < 30,
  });

  sleep(0.1);
}

