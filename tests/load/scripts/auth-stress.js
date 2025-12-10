import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 커스텀 메트릭
const reqDuration = new Trend('req_duration');
const errorRate = new Rate('error_rate');
const successCount = new Counter('success_count');

// 테스트 설정
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const MAX_VUS = parseInt(__ENV.MAX_VUS) || 100;
const RAMP_DURATION = __ENV.RAMP_DURATION || '30s';
const HOLD_DURATION = __ENV.HOLD_DURATION || '30s';

export const options = {
  scenarios: {
    auth_stress: {
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
    http_req_duration: ['p(95)<500'],
    error_rate: ['rate<0.05'],
  },
};

export function setup() {
  console.log(`\n=== 인증 CPU 병목 테스트 ===`);
  console.log(`최대 VU: ${MAX_VUS}`);
  console.log(`측정 대상: Protected route (/auth/users/me)`);
  console.log(`목적: Auth Service CPU 병목 측정\n`);

  const timestamp = Date.now();
  const email = `authtest_${timestamp}@test.com`;
  const password = 'test1234!';

  // 회원가입
  const registerRes = http.post(
    `${BASE_URL}/auth/register`,
    JSON.stringify({
      email: email,
      password: password,
      name: 'Auth Test User',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (registerRes.status !== 201 && registerRes.status !== 200) {
    console.error(`회원가입 실패: ${registerRes.status}`);
  }

  // 로그인
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email, password }),
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
  if (!token) {
    return;
  }

  const startTime = Date.now();

  // Protected route 호출 (100% Auth 검증 필요)
  const res = http.get(`${BASE_URL}/auth/users/me`, {
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`,
    },
    timeout: '10s',
  });

  const duration = Date.now() - startTime;
  reqDuration.add(duration);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
  });

  if (success) {
    successCount.add(1);
  }
  errorRate.add(!success);

  if (!success && res.status !== 200) {
    console.error(`[VU ${__VU}] 실패 (${res.status}): ${(res.body || '').substring(0, 100)}`);
  }
}

export function handleSummary(data) {
  const p95 = data.metrics.http_req_duration.values['p(95)'];
  const avg = data.metrics.http_req_duration.values['avg'];
  const totalReqs = data.metrics.http_reqs.values['count'];
  const errorRateVal = data.metrics.error_rate ? data.metrics.error_rate.values['rate'] : 0;
  const testDuration = data.state.testRunDurationMs / 1000;
  const throughput = (totalReqs / testDuration).toFixed(2);

  console.log('\n========================================');
  console.log('    인증 CPU 병목 테스트 결과');
  console.log('========================================');
  console.log(`최대 VU: ${MAX_VUS}`);
  console.log(`테스트 시간: ${testDuration.toFixed(1)}s`);
  console.log('');
  console.log('--- 성능 지표 ---');
  console.log(`총 요청 수: ${totalReqs}`);
  console.log(`처리량: ${throughput} req/s`);
  console.log(`평균 응답시간: ${avg.toFixed(2)}ms`);
  console.log(`p95 응답시간: ${p95.toFixed(2)}ms`);
  console.log(`에러율: ${(errorRateVal * 100).toFixed(2)}%`);
  console.log('');
  console.log('--- CPU 사용률 (별도 측정 필요) ---');
  console.log('docker stats --no-stream 으로 확인');
  console.log('========================================');

  return {};
}
