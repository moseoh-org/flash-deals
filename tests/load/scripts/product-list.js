import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// 커스텀 메트릭
const listDuration = new Trend('product_list_duration');
const listFailRate = new Rate('product_list_fail_rate');

// 테스트 설정
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const DURATION = __ENV.DURATION || '30s';
const VUS = parseInt(__ENV.VUS) || 10;

export const options = {
  scenarios: {
    product_list: {
      executor: 'constant-vus',
      vus: VUS,
      duration: DURATION,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    product_list_fail_rate: ['rate<0.01'],
  },
};

export default function () {
  // 랜덤 페이지 조회 (1~10 페이지)
  const page = Math.floor(Math.random() * 10) + 1;
  const size = 20;

  const startTime = Date.now();

  const res = http.get(`${BASE_URL}/products?page=${page}&size=${size}`, {
    headers: {
      'Accept': 'application/json',
      'Accept-Encoding': 'gzip, deflate',
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
    console.error(`조회 실패: ${res.status} - ${res.body}`);
  }

  // 요청 간 짧은 대기
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
