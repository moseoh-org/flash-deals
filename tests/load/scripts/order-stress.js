import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import encoding from 'k6/encoding';

// JWT에서 user_id(sub) 추출
function parseJwtUserId(token) {
  const parts = token.split('.');
  if (parts.length !== 3) return null;
  try {
    const payload = encoding.b64decode(parts[1], 'rawurl', 's');
    const decoded = JSON.parse(payload);
    return decoded.sub;
  } catch (e) {
    return null;
  }
}

// 커스텀 메트릭
const orderDuration = new Trend('order_duration');
const errorRate = new Rate('error_rate');
const successCount = new Counter('success_count');
const stockErrorCount = new Counter('stock_error_count');

// 테스트 설정
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const MAX_VUS = parseInt(__ENV.MAX_VUS) || 50;
const RAMP_DURATION = __ENV.RAMP_DURATION || '30s';
const HOLD_DURATION = __ENV.HOLD_DURATION || '60s';
const NUM_USERS = parseInt(__ENV.NUM_USERS) || 50;

export const options = {
  scenarios: {
    order_stress: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: RAMP_DURATION, target: MAX_VUS },
        { duration: HOLD_DURATION, target: MAX_VUS },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    error_rate: ['rate<0.1'],
  },
};

export function setup() {
  console.log(`\n=== 주문 처리량 한계 테스트 ===`);
  console.log(`최대 VU: ${MAX_VUS}`);
  console.log(`테스트 사용자 수: ${NUM_USERS}`);
  console.log(`측정 대상: POST /orders (주문 생성)`);
  console.log(`목적: Order Service 처리량 한계 측정\n`);

  const users = [];
  const timestamp = Date.now();

  // 테스트용 사용자 생성
  console.log(`${NUM_USERS}명의 테스트 사용자 생성 중...`);
  for (let i = 0; i < NUM_USERS; i++) {
    const email = `ordertest_${timestamp}_${i}@test.com`;
    const password = 'test1234!';

    // 회원가입
    const registerRes = http.post(
      `${BASE_URL}/auth/register`,
      JSON.stringify({
        email: email,
        password: password,
        name: `Order Test User ${i}`,
      }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    if (registerRes.status !== 201 && registerRes.status !== 200) {
      console.error(`사용자 ${i} 회원가입 실패: ${registerRes.status}`);
      continue;
    }

    // 로그인
    const loginRes = http.post(
      `${BASE_URL}/auth/login`,
      JSON.stringify({ email, password }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    if (loginRes.status !== 200) {
      console.error(`사용자 ${i} 로그인 실패: ${loginRes.status}`);
      continue;
    }

    const loginData = JSON.parse(loginRes.body);
    const userId = parseJwtUserId(loginData.access_token);
    users.push({ token: loginData.access_token, userId, email });
  }
  console.log(`${users.length}명의 사용자 생성 완료`);

  // 테스트용 상품 생성 (재고 충분히)
  console.log('테스트용 상품 생성 중...');
  const productRes = http.post(
    `${BASE_URL}/products`,
    JSON.stringify({
      name: `주문테스트상품_${timestamp}`,
      description: '주문 부하 테스트용 상품',
      price: 10000,
      stock: 1000000, // 재고 100만개 (재고 부족 방지)
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (productRes.status !== 201 && productRes.status !== 200) {
    console.error(`상품 생성 실패: ${productRes.status} - ${productRes.body}`);
    return { users: [], productId: null };
  }

  const productData = JSON.parse(productRes.body);
  console.log(`상품 생성 완료: ${productData.id}`);

  return {
    users,
    productId: productData.id,
  };
}

export default function (data) {
  if (!data.users || data.users.length === 0 || !data.productId) {
    console.error('Setup 데이터 없음');
    return;
  }

  // VU 번호에 따라 사용자 선택 (다양한 사용자가 주문)
  const userIndex = __VU % data.users.length;
  const user = data.users[userIndex];
  const token = user.token;
  const userId = user.userId;

  const startTime = Date.now();

  // 주문 생성
  const orderRes = http.post(
    `${BASE_URL}/orders`,
    JSON.stringify({
      items: [
        {
          product_id: data.productId,
          quantity: 1,
        },
      ],
      shipping_address: {
        recipient_name: '테스트',
        phone: '010-1234-5678',
        address: '서울시 강남구',
        address_detail: '테스트동 123호',
        postal_code: '12345',
      },
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
        'X-User-ID': userId,
      },
      timeout: '30s',
    }
  );

  const duration = Date.now() - startTime;
  orderDuration.add(duration);

  const success = check(orderRes, {
    'status is 201': (r) => r.status === 201,
  });

  if (success) {
    successCount.add(1);
  } else {
    // 재고 부족 에러 별도 카운트
    if (orderRes.body && orderRes.body.includes('INSUFFICIENT_STOCK')) {
      stockErrorCount.add(1);
    }
  }
  errorRate.add(!success);

  if (!success && orderRes.status !== 201) {
    const bodyPreview = (orderRes.body || '').substring(0, 100);
    if (!bodyPreview.includes('INSUFFICIENT_STOCK')) {
      console.error(`[VU ${__VU}] 주문 실패 (${orderRes.status}): ${bodyPreview}`);
    }
  }

  // 약간의 think time
  sleep(0.1);
}

export function handleSummary(data) {
  const p95 = data.metrics.http_req_duration.values['p(95)'];
  const avg = data.metrics.http_req_duration.values['avg'];
  const totalReqs = data.metrics.http_reqs.values['count'];
  const errorRateVal = data.metrics.error_rate ? data.metrics.error_rate.values['rate'] : 0;
  const testDuration = data.state.testRunDurationMs / 1000;
  const throughput = (totalReqs / testDuration).toFixed(2);
  const successCountVal = data.metrics.success_count ? data.metrics.success_count.values['count'] : 0;
  const orderThroughput = (successCountVal / testDuration).toFixed(2);

  console.log('\n========================================');
  console.log('    주문 처리량 한계 테스트 결과');
  console.log('========================================');
  console.log(`최대 VU: ${MAX_VUS}`);
  console.log(`테스트 시간: ${testDuration.toFixed(1)}s`);
  console.log('');
  console.log('--- 성능 지표 ---');
  console.log(`총 요청 수: ${totalReqs}`);
  console.log(`성공한 주문: ${successCountVal}`);
  console.log(`전체 처리량: ${throughput} req/s`);
  console.log(`주문 처리량: ${orderThroughput} orders/s`);
  console.log(`평균 응답시간: ${avg.toFixed(2)}ms`);
  console.log(`p95 응답시간: ${p95.toFixed(2)}ms`);
  console.log(`에러율: ${(errorRateVal * 100).toFixed(2)}%`);
  console.log('');
  console.log('--- CPU 사용률 (별도 측정 필요) ---');
  console.log('docker stats --no-stream 으로 확인');
  console.log('========================================');

  return {};
}
