import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 500 },   // Ramp up to 500 virtual users
    { duration: '15s', target: 2000 }, // Spike to 2,000 concurrent users competing for stock
    { duration: '5s', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'], // 95% of requests must complete below 100ms
    http_req_failed: ['rate<0.01'],    // Error rate must be less than 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';
const PRODUCT_ID = __ENV.PRODUCT_ID || '60d5ecb8b5c9c22b9c8b4567';

export default function () {
  const userId = `vu-user-${__VU}-${Math.floor(Math.random() * 100000)}`;

  const url = `${BASE_URL}/orders/flash-sale`;
  const payload = JSON.stringify({
    product_id: PRODUCT_ID,
    quantity: 1,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${userId}`, // Uses Dev Firebase JWT Token
    },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'status is 202 Accepted or 400 Out of Stock': (r) => r.status === 202 || (r.status === 400 && r.body.includes('out of stock')),
    'response time < 50ms': (r) => r.timings.duration < 50,
  });

  sleep(0.1);
}
