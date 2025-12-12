# 주문 처리량 한계

## 상황

서비스가 인기를 얻으면서 사용자와 요청이 급증했다.
핫딜 이벤트 시작 시 동시 주문 요청이 폭주하면서 주문 처리량의 한계에 도달한다.

## 테스트 환경

- k6 부하 테스트: `make load-order-stress`
- 최대 VU: 50
- Ramp-up: 30초, Hold: 60초

---

## 기존 시스템

### 성능

| 지표          | 값       |
| ------------- | -------- |
| p95 응답시간  | 867ms    |
| 평균 응답시간 | 626ms    |
| 처리량        | 46 req/s |
| 에러율        | 0%       |

### 리소스 사용량

| 컨테이너 | CPU  |
| -------- | ---- |
| order    | 86%  |
| product  | 40%  |
| postgres | ~20% |

### 문제점

- Order Service CPU 86%로 병목 발생
- 매 주문마다 Product Service로 HTTP 호출 (상품조회 + 재고차감)
- `httpx.AsyncClient`를 매 요청마다 생성/해제
- 50 VU에서 이미 처리량 한계 (46 req/s) 도달

---

## 개선 시도: httpx 커넥션 풀 재사용

### 변경 내용

- 전역 `httpx.AsyncClient` 생성하여 커넥션 풀 재사용
- `max_connections=100`, `max_keepalive_connections=20` 설정

### 성능

| 지표          | Before | After | 개선  |
| ------------- | ------ | ----- | ----- |
| p95 응답시간  | 867ms  | 912ms | -5% ↑ |
| 평균 응답시간 | 626ms  | 639ms | -2% ↑ |
| 처리량        | 46     | 45    | -2% ↓ |
| 에러율        | 0%     | 0%    | -     |

### 분석

- **효과 없음** - 오히려 약간 느려짐
- 원인: Order Service의 병목이 HTTP 커넥션이 아님
  - Gateway → Auth: 단순 검증이라 커넥션 풀 효과 큼 (deal-traffic-spike 시나리오)
  - Order → Product: DB 트랜잭션과 함께 처리되어 커넥션 풀 효과 제한적
- HTTP/1.1의 근본적 한계: Head-of-line blocking, 매 요청 헤더 전송

---

## 개선 1: gRPC 전환

### 변경 내용

1. **gRPC + Protocol Buffers**: HTTP/JSON → gRPC/Protobuf
2. **HTTP/2 연결 재사용**: 단일 연결로 멀티플렉싱
3. **바이너리 직렬화**: JSON 대비 빠른 직렬화/역직렬화
4. **헤더 압축**: HPACK으로 헤더 오버헤드 감소

### 성능

| 지표          | Before (HTTP) | After (gRPC) | 개선        |
| ------------- | ------------- | ------------ | ----------- |
| p95 응답시간  | 867ms         | 367ms        | **58% ↓**   |
| 평균 응답시간 | 626ms         | 163ms        | **74% ↓**   |
| 처리량        | 46            | 125          | **2.7배 ↑** |
| 에러율        | 0%            | 0%           | -           |

### 리소스 사용량 비교

| 컨테이너 | Before (CPU) | After (CPU) |
| -------- | ------------ | ----------- |
| order    | 86%          | 64%         |
| product  | 40%          | 79%         |

### 분석

- gRPC HTTP/2의 멀티플렉싱으로 단일 연결에서 다중 요청 동시 처리
- 바이너리 직렬화로 JSON 파싱 오버헤드 제거
- 병목이 Order Service(86%) → Product Service(79%)로 이동
- Product Service가 새로운 병목 지점 (다음 개선 대상)
