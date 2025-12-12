# 상품 목록 조회가 느림

## 상황

메인 페이지 트래픽이 증가하면서 상품 목록 API의 응답 시간이 급격히 느려졌다.
사용자들이 상품 목록 페이지에서 로딩이 오래 걸린다는 불만이 접수되고 있다.

## 테스트 환경

- 부하 테스트: `make load-product-list`
- 최대 VU: 10
- 테스트 대상: `GET /products` (상품 목록 조회)
- 상품 데이터: 1,000만 건

---

## 기존 시스템

### 성능

| 지표          | 값       |
| ------------- | -------- |
| p95 응답시간  | 738ms    |
| 평균 응답시간 | 469ms    |
| 처리량        | 17req/s  |
| 에러율        | 0%       |

### 리소스 사용량

| 컨테이너 | CPU  |
| -------- | ---- |
| postgres | 573% |
| product  | 5%   |
| gateway  | 16%  |

### 문제점

- PostgreSQL CPU 573%로 DB 과부하
- p95 738ms로 목표치(500ms) 초과
- 매 요청마다 COUNT + SELECT 쿼리 실행

---

## 개선 1: 목록 조회 + Total Count 캐싱

### 변경 내용

- Repository 패턴 도입 (`RdbProductRepository`, `CachedProductRepository`)
- Cache-Aside 패턴으로 목록 + total 함께 Redis 캐싱
- 캐시 키: `products:list:{page}:{size}:{category}`, TTL 60초

### 성능

| 지표          | Before  | After   | 개선    |
| ------------- | ------- | ------- | ------- |
| p95 응답시간  | 738ms   | 84ms    | 8.8배 ↓ |
| 평균 응답시간 | 469ms   | 66ms    | 7.1배 ↓ |
| 처리량        | 17req/s | 60req/s | 3.5배 ↑ |
| 에러율        | 0%      | 0%      | -       |

### 리소스 사용량 비교

| 컨테이너 | Before | After |
| -------- | ------ | ----- |
| postgres | 573%   | 0%    |
| product  | 5%     | 8%    |
| gateway  | 16%    | 44%   |
| redis    | -      | 0.4%  |

### 분석

- PostgreSQL CPU 573% → 0%: 캐시 히트로 DB 쿼리 제거
- Redis 메모리 8MB로 안정적
- Gateway CPU 증가: DB 병목 해소로 처리량 증가에 따른 자연스러운 현상
