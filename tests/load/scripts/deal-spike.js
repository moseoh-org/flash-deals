import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 커스텀 메트릭
const reqDuration = new Trend('req_duration');
const errorRate = new Rate('error_rate');
const poolExhaustedErrors = new Counter('pool_exhausted_errors');

// 상태 코드별 카운터
const status2xx = new Counter('status_2xx');
const status4xx = new Counter('status_4xx');
const status5xx = new Counter('status_5xx');
const status502 = new Counter('status_502_bad_gateway');
const status503 = new Counter('status_503_unavailable');
const statusTimeout = new Counter('status_timeout');

// 테스트 설정
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const MAX_VUS = parseInt(__ENV.MAX_VUS) || 100;
const RAMP_DURATION = __ENV.RAMP_DURATION || '30s';
const HOLD_DURATION = __ENV.HOLD_DURATION || '30s';

export const options = {
  scenarios: {
    deal_spike: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: RAMP_DURATION, target: MAX_VUS }, // VU 점진적 증가
        { duration: HOLD_DURATION, target: MAX_VUS }, // 최대 VU 유지
        { duration: '10s', target: 0 }, // 종료
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    error_rate: ['rate<0.05'], // 에러율 5% 미만
  },
};

export function setup() {
  const timestamp = Date.now();
  const email = `dealtest_${timestamp}@test.com`;
  const password = 'test1234!';

  // 회원가입
  http.post(
    `${BASE_URL}/auth/register`,
    JSON.stringify({
      email: email,
      password: password,
      name: 'Deal Test User',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  // 로그인
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({
      email: email,
      password: password,
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (loginRes.status !== 200) {
    console.error(`로그인 실패: ${loginRes.status}`);
    return { token: '' };
  }

  const loginData = JSON.parse(loginRes.body);
  console.log('Setup 완료: 토큰 획득');

  return { token: loginData.access_token };
}

export default function (data) {
  const token = data.token;

  // 핫딜 트래픽 시뮬레이션: 상품 목록 조회 (가장 빈번한 요청)
  const endpoints = [
    { method: 'GET', url: `${BASE_URL}/products?page=1&size=20`, weight: 70 },
    { method: 'GET', url: `${BASE_URL}/auth/users/me`, weight: 30 },
  ];

  // 가중치 기반 엔드포인트 선택
  const rand = Math.random() * 100;
  let cumWeight = 0;
  let endpoint = endpoints[0];
  for (const ep of endpoints) {
    cumWeight += ep.weight;
    if (rand < cumWeight) {
      endpoint = ep;
      break;
    }
  }

  const startTime = Date.now();

  const res = http.request(endpoint.method, endpoint.url, null, {
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`,
    },
    timeout: '10s',
  });

  const duration = Date.now() - startTime;
  reqDuration.add(duration);

  const success = check(res, {
    'status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'no timeout': (r) => r.status !== 0,
  });

  errorRate.add(!success);

  // 상태 코드별 카운터 업데이트
  if (res.status === 0) {
    statusTimeout.add(1);
    console.error(`[VU ${__VU}] Timeout/Connection error`);
  } else if (res.status >= 200 && res.status < 300) {
    status2xx.add(1);
  } else if (res.status >= 400 && res.status < 500) {
    status4xx.add(1);
    // 4xx 에러 상세 로깅 (샘플링)
    if (Math.random() < 0.01) {
      console.warn(`[VU ${__VU}] ${res.status} ${endpoint.url}: ${(res.body || '').substring(0, 100)}`);
    }
  } else if (res.status >= 500) {
    status5xx.add(1);
    if (res.status === 502) {
      status502.add(1);
    } else if (res.status === 503) {
      status503.add(1);
    }
    // 5xx 에러는 모두 로깅
    console.error(`[VU ${__VU}] ${res.status}: ${(res.body || '').substring(0, 200)}`);
  }

  // 커넥션 풀 고갈 또는 서버 과부하 에러 감지
  if (res.status === 500 || res.status === 503) {
    const body = res.body || '';
    if (
      body.includes('pool') ||
      body.includes('connection') ||
      body.includes('timeout') ||
      body.includes('QueuePool')
    ) {
      poolExhaustedErrors.add(1);
    }
  }

  // 짧은 대기 (실제 사용자 시뮬레이션)
  sleep(0.05);
}

export function handleSummary(data) {
  const p95 = data.metrics.http_req_duration.values['p(95)'];
  const avg = data.metrics.http_req_duration.values['avg'];
  const errorRateVal = data.metrics.error_rate ? data.metrics.error_rate.values['rate'] : 0;
  const poolErrors = data.metrics.pool_exhausted_errors
    ? data.metrics.pool_exhausted_errors.values['count']
    : 0;
  const totalReqs = data.metrics.http_reqs.values['count'];

  // 상태 코드별 카운트
  const count2xx = data.metrics.status_2xx ? data.metrics.status_2xx.values['count'] : 0;
  const count4xx = data.metrics.status_4xx ? data.metrics.status_4xx.values['count'] : 0;
  const count5xx = data.metrics.status_5xx ? data.metrics.status_5xx.values['count'] : 0;
  const count502 = data.metrics.status_502_bad_gateway ? data.metrics.status_502_bad_gateway.values['count'] : 0;
  const count503 = data.metrics.status_503_unavailable ? data.metrics.status_503_unavailable.values['count'] : 0;
  const countTimeout = data.metrics.status_timeout ? data.metrics.status_timeout.values['count'] : 0;

  console.log('\n=== 테스트 결과 요약 ===');
  console.log(`최대 VU: ${MAX_VUS}`);
  console.log(`총 요청 수: ${totalReqs}`);
  console.log(`p95 응답시간: ${p95.toFixed(2)}ms`);
  console.log(`평균 응답시간: ${avg.toFixed(2)}ms`);
  console.log(`에러율: ${(errorRateVal * 100).toFixed(2)}%`);
  console.log('');
  console.log('=== 상태 코드 분포 ===');
  console.log(`2xx (성공): ${count2xx}`);
  console.log(`4xx (클라이언트 에러): ${count4xx}`);
  console.log(`5xx (서버 에러): ${count5xx}`);
  console.log(`  - 502 Bad Gateway: ${count502}`);
  console.log(`  - 503 Unavailable: ${count503}`);
  console.log(`Timeout/Connection 에러: ${countTimeout}`);
  console.log(`커넥션 풀 관련 에러: ${poolErrors}건`);

  return {};
}
