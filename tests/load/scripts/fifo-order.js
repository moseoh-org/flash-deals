import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import encoding from 'k6/encoding';
import { SharedArray } from 'k6/data';

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
const successCount = new Counter('success_count');
const failCount = new Counter('fail_count');

// 테스트 설정
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const NUM_USERS = parseInt(__ENV.NUM_USERS) || 20;
const INITIAL_STOCK = parseInt(__ENV.INITIAL_STOCK) || 10;

// 결과 저장용 (외부 파일에 기록)
let resultFile = '/tmp/fifo_results.json';

export const options = {
  scenarios: {
    fifo_test: {
      executor: 'per-vu-iterations',
      vus: NUM_USERS,
      iterations: 1,
      maxDuration: '60s',
    },
  },
  // threshold 없음 - 순서 보장 검증이 목적
};

export function setup() {
  console.log(`\n=== 선착순 주문 순서 테스트 ===`);
  console.log(`총 사용자: ${NUM_USERS}명`);
  console.log(`상품 재고: ${INITIAL_STOCK}개`);
  console.log(`기대 결과: User 1~${INITIAL_STOCK}이 구매 성공해야 함`);
  console.log(`목적: 요청 순서대로 처리되는지 검증\n`);

  const users = [];
  const timestamp = Date.now();

  // 테스트용 사용자 생성 (User 1, 2, 3, ... 순서대로)
  console.log(`${NUM_USERS}명의 테스트 사용자 생성 중...`);
  for (let i = 1; i <= NUM_USERS; i++) {
    const email = `fifo_${timestamp}_user${i}@test.com`;
    const password = 'test1234!';

    // 회원가입
    const registerRes = http.post(
      `${BASE_URL}/auth/register`,
      JSON.stringify({
        email: email,
        password: password,
        name: `FIFO User ${i}`,
      }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    if (registerRes.status !== 201 && registerRes.status !== 200) {
      console.error(`User ${i} 회원가입 실패: ${registerRes.status}`);
      continue;
    }

    // 로그인
    const loginRes = http.post(
      `${BASE_URL}/auth/login`,
      JSON.stringify({ email, password }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    if (loginRes.status !== 200) {
      console.error(`User ${i} 로그인 실패: ${loginRes.status}`);
      continue;
    }

    const loginData = JSON.parse(loginRes.body);
    const userId = parseJwtUserId(loginData.access_token);
    users.push({
      userNumber: i,
      token: loginData.access_token,
      userId,
      email,
    });
  }
  console.log(`${users.length}명의 사용자 생성 완료`);

  // 테스트용 상품 생성 (재고 제한)
  console.log(`재고 ${INITIAL_STOCK}개인 상품 생성 중...`);
  const productRes = http.post(
    `${BASE_URL}/products`,
    JSON.stringify({
      name: `선착순상품_${timestamp}`,
      description: '선착순 테스트용 상품 (한정 수량)',
      price: 10000,
      stock: INITIAL_STOCK,
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (productRes.status !== 201 && productRes.status !== 200) {
    console.error(`상품 생성 실패: ${productRes.status} - ${productRes.body}`);
    return { users: [], productId: null };
  }

  const productData = JSON.parse(productRes.body);
  console.log(`상품 생성 완료: ${productData.id} (재고: ${INITIAL_STOCK})`);
  console.log(`\n--- 테스트 시작 ---`);
  console.log(`${NUM_USERS}명이 동시에 주문 요청...\n`);

  return {
    users,
    productId: productData.id,
    startTime: Date.now(),
  };
}

export default function (data) {
  if (!data.users || data.users.length === 0 || !data.productId) {
    console.error('Setup 데이터 없음');
    return;
  }

  // VU 번호로 사용자 매핑 (VU 1 = User 1, VU 2 = User 2, ...)
  const userIndex = __VU - 1;
  if (userIndex >= data.users.length) {
    console.error(`VU ${__VU}: 매핑된 사용자 없음`);
    return;
  }

  const user = data.users[userIndex];
  const userNumber = user.userNumber;
  const token = user.token;
  const userId = user.userId;

  // 요청 시간 기록
  const requestTime = Date.now();
  const relativeTime = requestTime - data.startTime;

  // 주문 요청
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
        recipient_name: `User ${userNumber}`,
        phone: '010-1234-5678',
        address: '서울시 강남구',
        address_detail: `${userNumber}호`,
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

  const responseTime = Date.now();
  const duration = responseTime - requestTime;
  orderDuration.add(duration);

  const success = orderRes.status === 201;

  if (success) {
    successCount.add(1);
    console.log(
      `✅ User ${userNumber}: 주문 성공 (요청: +${relativeTime}ms, 응답: ${duration}ms)`
    );
  } else {
    failCount.add(1);
    const reason =
      orderRes.body && orderRes.body.includes('INSUFFICIENT_STOCK')
        ? '재고 부족'
        : `에러 ${orderRes.status}`;
    console.log(
      `❌ User ${userNumber}: ${reason} (요청: +${relativeTime}ms, 응답: ${duration}ms)`
    );
  }
}

export function handleSummary(data) {
  const successCountVal = data.metrics.success_count
    ? data.metrics.success_count.values['count']
    : 0;
  const failCountVal = data.metrics.fail_count
    ? data.metrics.fail_count.values['count']
    : 0;

  console.log('\n========================================');
  console.log('    선착순 주문 순서 테스트 결과');
  console.log('========================================');
  console.log(`총 요청: ${NUM_USERS}명`);
  console.log(`상품 재고: ${INITIAL_STOCK}개`);
  console.log('');
  console.log('--- 결과 ---');
  console.log(`주문 성공: ${successCountVal}명`);
  console.log(`주문 실패: ${failCountVal}명`);
  console.log('');
  console.log('--- 분석 ---');
  console.log(`기대값: User 1~${INITIAL_STOCK}이 성공해야 함`);
  console.log('실제값: 위 로그에서 성공한 User 번호 확인');
  console.log('');
  console.log('⚠️  순서가 보장되지 않는 경우:');
  console.log('   - User 1~10이 아닌 다른 조합이 성공');
  console.log('   - 먼저 요청한 User가 실패하고 나중 User가 성공');
  console.log('');
  console.log('💡 해결책:');
  console.log('   - Go Channel (단일 서버): 인메모리 FIFO 큐');
  console.log('   - Kafka (다중 서버): 분산 메시지 큐 + 파티션');
  console.log('========================================');

  return {};
}
