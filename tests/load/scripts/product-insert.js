import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// 커스텀 메트릭
const insertDuration = new Trend('product_insert_duration');
const insertFailRate = new Rate('product_insert_fail_rate');

// 테스트 설정
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const TOTAL_PRODUCTS = parseInt(__ENV.TOTAL_PRODUCTS) || 1000;

export const options = {
  scenarios: {
    product_insert: {
      executor: 'shared-iterations',
      vus: 10,
      iterations: TOTAL_PRODUCTS,
      maxDuration: '10m',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    product_insert_fail_rate: ['rate<0.01'],
  },
};

// 테스트 사용자 정보 (setup에서 로그인)
let authToken = '';

export function setup() {
  const timestamp = Date.now();
  const email = `loadtest_admin_${timestamp}@test.com`;
  const password = 'test1234!';

  // 1. 회원가입
  const registerRes = http.post(
    `${BASE_URL}/auth/register`,
    JSON.stringify({
      email: email,
      password: password,
      name: 'Load Test Admin',
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

  check(loginRes, {
    'login successful': (r) => r.status === 200,
  });

  if (loginRes.status !== 200) {
    console.error(`로그인 실패: ${loginRes.status} - ${loginRes.body}`);
    return { token: '' };
  }

  const loginData = JSON.parse(loginRes.body);
  console.log(`Setup 완료: 토큰 획득`);

  return { token: loginData.access_token };
}

export default function (data) {
  const token = data.token;

  if (!token) {
    console.error('토큰이 없습니다. 테스트를 중단합니다.');
    return;
  }

  const vuId = __VU;
  const iterationId = __ITER;
  const uniqueId = `${vuId}_${iterationId}_${Date.now()}`;

  // 상품 생성 요청
  const payload = JSON.stringify({
    name: `테스트 상품 ${uniqueId}`,
    description: `부하 테스트용 상품입니다. ID: ${uniqueId}`,
    price: Math.floor(Math.random() * 100000) + 10000,
    stock: Math.floor(Math.random() * 100) + 1,
    category: ['electronics', 'fashion', 'food', 'home'][Math.floor(Math.random() * 4)],
  });

  const startTime = Date.now();

  const res = http.post(`${BASE_URL}/products`, payload, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  });

  const duration = Date.now() - startTime;
  insertDuration.add(duration);

  const success = check(res, {
    'product created': (r) => r.status === 201,
  });

  insertFailRate.add(!success);

  if (!success) {
    console.error(`상품 생성 실패: ${res.status} - ${res.body}`);
  }

  // 요청 간 짧은 대기 (DB 부하 분산)
  sleep(0.01);
}

export function teardown(data) {
  console.log('=== 테스트 완료 ===');
  console.log(`총 상품 등록 시도: ${TOTAL_PRODUCTS}건`);
}
