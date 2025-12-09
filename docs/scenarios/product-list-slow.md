# 상품 목록 조회가 느림

## 상황

메인 페이지 트래픽이 증가하면서 상품 목록 API(`GET /products`)의 응답 시간이 급격히 느려졌다.
사용자들이 상품 목록 페이지에서 로딩이 오래 걸린다는 불만이 접수되고 있다.

## 테스트 환경

- PostgreSQL 16 (Docker)
- 상품 데이터: 1,000만 건
- 부하: 10 VU, 30초

---

## 기존 시스템

매 요청마다 PostgreSQL에서 COUNT + SELECT 쿼리를 실행한다.

### 성능

| 지표          | 값       |
| ------------- | -------- |
| p95 응답시간  | 738ms    |
| 평균 응답시간 | 469ms    |
| TPS           | 17 req/s |

### 리소스 사용량

| 컨테이너 | CPU    | RAM   |
| -------- | ------ | ----- |
| postgres | 573%   | 512MB |
| product  | 5.10%  | 85MB  |
| gateway  | 16.46% | 128MB |

### 문제점

- PostgreSQL CPU 573%로 DB 과부하
- p95 738ms로 목표치(500ms) 초과

---

## 개선 1: 목록 조회 + Total Count 캐싱

### 변경 내용

- Repository 패턴 도입 (`RdbProductRepository`, `CachedProductRepository`)
- Cache-Aside 패턴으로 목록 + total 함께 Redis 캐싱
- 캐시 키: `products:list:{page}:{size}:{category}`, TTL 60초
- 환경변수 `ENABLE_CACHE`로 캐싱 On/Off 제어

### 성능

| 지표          | Before | After | 개선    |
| ------------- | ------ | ----- | ------- |
| p95 응답시간  | 738ms  | 84ms  | 8.8배 ↓ |
| 평균 응답시간 | 469ms  | 66ms  | 7.1배 ↓ |
| TPS           | 17     | 60    | 3.5배 ↑ |

### 리소스 사용량 비교

| 컨테이너 | Before (CPU) | After (CPU) | Before (RAM) | After (RAM) |
| -------- | ------------ | ----------- | ------------ | ----------- |
| postgres | 573%         | 0%          | 512MB        | 139MB       |
| product  | 5.10%        | 7.89%       | 85MB         | 87MB        |
| gateway  | 16.46%       | 43.61%      | 128MB        | 121MB       |
| redis    | -            | 0.37%       | -            | 8MB         |

### 분석

- PostgreSQL CPU 573% → 0%: 캐시 히트로 DB 쿼리 제거
- Redis 메모리 8MB: 페이지당 20개 × 10페이지 캐싱, 정상 수준
- Gateway CPU 증가: DB 병목 해소로 처리량 증가에 따른 자연스러운 현상

---

## 추가 개선 계획

- [ ] E6: gzip 압축 적용
