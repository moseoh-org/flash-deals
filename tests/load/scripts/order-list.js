import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// 커스텀 메트릭
const listDuration = new Trend('order_list_duration');
const listFailRate = new Rate('order_list_fail_rate');

// 테스트 설정
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const DURATION = __ENV.DURATION || '30s';
const VUS = parseInt(__ENV.VUS) || 10;
const ORDERS_PER_USER = parseInt(__ENV.ORDERS_PER_USER) || 50;

export const options = {
  scenarios: {
    order_list: {
      executor: 'constant-vus',
      vus: VUS,
      duration: DURATION,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    order_list_fail_rate: ['rate<0.01'],
  },
};

export function setup() {
  const timestamp = Date.now();
  const email = `ordertest_${timestamp}@test.com`;
  const password = 'test1234!';

  // 1. 회원가입
  const registerRes = http.post(
    `${BASE_URL}/auth/register`,
    JSON.stringify({
      email: email,
      password: password,
      name: 'Order Test User',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (registerRes.status !== 201 && registerRes.status !== 409) {
    console.error(`회원가입 실패: ${registerRes.status} - ${registerRes.body}`);
  }

  // 2. 로그인
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({
      email: email,
      password: password,
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (loginRes.status !== 200) {
    console.error(`로그인 실패: ${loginRes.status} - ${loginRes.body}`);
    return { token: '' };
  }

  const loginData = JSON.parse(loginRes.body);
  const token = loginData.access_token;

  // 3. 상품 생성 (주문용)
  const productRes = http.post(
    `${BASE_URL}/products`,
    JSON.stringify({
      name: `주문 테스트 상품 ${timestamp}`,
      description: '주문 목록 부하 테스트용',
      price: 10000,
      stock: 100000,
      category: 'test',
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
    }
  );

  if (productRes.status !== 201) {
    console.error(`상품 생성 실패: ${productRes.status} - ${productRes.body}`);
    return { token: '' };
  }

  const productData = JSON.parse(productRes.body);
  const productId = productData.id;

  // 4. 주문 생성 (N+1 테스트를 위해 여러 개)
  console.log(`주문 ${ORDERS_PER_USER}건 생성 중...`);
  for (let i = 0; i < ORDERS_PER_USER; i++) {
    const orderRes = http.post(
      `${BASE_URL}/orders`,
      JSON.stringify({
        items: [
          {
            product_id: productId,
            quantity: 1,
          },
        ],
        shipping_address: {
          recipient_name: '테스트',
          phone: '010-1234-5678',
          address: '서울시 강남구',
          address_detail: '101호',
          postal_code: '12345',
        },
      }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
      }
    );

    if (orderRes.status !== 201) {
      console.error(`주문 생성 실패 (${i + 1}/${ORDERS_PER_USER}): ${orderRes.status}`);
    }

    if ((i + 1) % 10 === 0) {
      console.log(`주문 생성 진행: ${i + 1}/${ORDERS_PER_USER}`);
    }
  }

  console.log(`Setup 완료: 주문 ${ORDERS_PER_USER}건 생성`);

  return { token: token };
}

export default function (data) {
  const token = data.token;

  if (!token) {
    console.error('토큰이 없습니다.');
    return;
  }

  // 랜덤 페이지 조회
  const page = Math.floor(Math.random() * 3) + 1;
  const size = 20;

  const startTime = Date.now();

  const res = http.get(`${BASE_URL}/orders?page=${page}&size=${size}`, {
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`,
    },
  });

  const duration = Date.now() - startTime;
  listDuration.add(duration);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'has items': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.items && body.items.length > 0;
      } catch {
        return false;
      }
    },
  });

  listFailRate.add(!success);

  if (!success) {
    console.error(`주문 목록 조회 실패: ${res.status} - ${res.body}`);
  }

  sleep(0.1);
}

export function handleSummary(data) {
  const p95 = data.metrics.http_req_duration.values['p(95)'];
  const avg = data.metrics.http_req_duration.values['avg'];
  const rps = data.metrics.http_reqs.values['rate'];

  console.log('\n=== 테스트 결과 요약 ===');
  console.log(`p95 응답시간: ${p95.toFixed(2)}ms`);
  console.log(`평균 응답시간: ${avg.toFixed(2)}ms`);
  console.log(`처리량: ${rps.toFixed(2)} req/s`);

  return {};
}
