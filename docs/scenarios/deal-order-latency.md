# 핫딜 주문 지연

## 상황

핫딜 오픈 시 대량의 주문 요청이 몰리면 응답 지연이 발생한다.
서버는 에러 없이 모든 요청을 처리하지만, 동시 요청 증가에 따라 대기 시간이 선형적으로 증가한다.

> **참고**: 이 시나리오는 [핫딜 트래픽 급증](deal-traffic-spike.md)에서 Gateway 병목을 해결한 이후, 주문(POST /orders) 엔드포인트에서 발생하는 지연 문제를 다룹니다.

## 테스트 환경

- 부하 테스트: `make load-order-overload`
- 최대 VU: 500
- 테스트 대상: `POST /orders` (핫딜 주문)
- Product Service: Go Channel 버퍼 10,000 (단일 워커)
- 환경 변수: `HOTDEAL_STOCK_REDIS` (true/false)

---

## 기존 시스템 (DB 재고 차감)

### 현재 구조

```
POST /orders → Order Service → Product Service (gRPC) → DB
                    ↓                   ↓
              Order INSERT       Go Channel (단일 워커)
              OrderItem INSERT          ↓
                                 순차적 재고 차감
```

### 병목 지점

- **Go Channel 단일 워커**: 재고 차감 요청을 순차 처리
- **I/O Bound 병목**: 워커가 DB 쿼리 완료를 대기하는 시간이 대부분
- **동기 응답**: 주문 완료까지 클라이언트가 대기

### 성능

| 지표          | 값       |
| ------------- | -------- |
| p95 응답시간  | 313ms    |
| 평균 응답시간 | 231ms    |
| 처리량        | 949req/s |
| 에러율        | 0%       |

### 리소스 사용량

| 컨테이너        | CPU |
| --------------- | --- |
| Order Service   | 69% |
| Product Service | 83% |
| PostgreSQL      | 73% |

---

## 개선 1: Redis 재고 차감 시스템

### 변경 내용

- 핫딜 시작 전 DB 재고를 Redis에 로드 (`POST /products/:id/hotdeal/start`)
- 재고 차감을 Redis Lua Script로 원자적 처리
- 핫딜 종료 시 Redis → DB 동기화 (`POST /products/:id/hotdeal/end`)
- 환경 변수 `HOTDEAL_STOCK_REDIS=true`로 활성화

### 동작 방식

```
[핫딜 시작]
POST /products/:id/hotdeal/start
  → DB 재고 조회 → Redis SET hotdeal:stock:{id}

[주문 요청]
POST /orders → Product Service (gRPC)
                    ↓
              Redis Lua Script (원자적 차감)
                    ↓
              성공: 새 재고값 반환
              실패: insufficient stock

[핫딜 종료]
POST /products/:id/hotdeal/end
  → Redis 재고 조회 → DB 동기화 → Redis 키 삭제
```

### 성능

| 지표          | Before   | After     | 개선    |
| ------------- | -------- | --------- | ------- |
| p95 응답시간  | 313ms    | 202ms     | 1.5배 ↓ |
| 평균 응답시간 | 231ms    | 131ms     | 1.8배 ↓ |
| 처리량        | 949req/s | 1667req/s | 1.8배 ↑ |
| 에러율        | 0%       | 0%        | -       |

### 리소스 사용량 비교

| 컨테이너        | Before | After | 변화             |
| --------------- | ------ | ----- | ---------------- |
| Order Service   | 69%    | 98%   | +29% (병목 이동) |
| Product Service | 83%    | 71%   | -12%             |
| PostgreSQL      | 73%    | 80%   | +7%              |
| Redis           | -      | 17%   | -                |

### 분석

- **처리량 1.8배 향상**: Redis Lua Script의 빠른 재고 차감으로 Product Service 병목 해소
- **Product Service CPU 83% → 71%**: DB I/O 대기 없이 Redis에서 즉시 처리
- **Order Service CPU 69% → 98% 포화**: 병목이 Product → Order로 이동
  - 재고 차감은 빨라졌지만, **Order/OrderItem INSERT가 여전히 동기적으로 DB에 저장**
  - Order Service가 DB INSERT 완료를 기다리면서 새로운 병목 발생
- **PostgreSQL CPU 73% → 80%**: 처리량 증가로 주문 INSERT 부하도 증가

### 다음 병목

```
POST /orders 흐름:
  1. Product Service gRPC 호출 (재고 차감) ← ✅ Redis로 개선됨
  2. Order INSERT (DB)      ← ⚠️ 동기 처리 (병목)
  3. OrderItem INSERT (DB)  ← ⚠️ 동기 처리 (병목)
  4. 202 Accepted 반환
```

Order Service CPU가 98%로 포화 상태이며, 이는 **주문 생성(INSERT)이 동기적으로 처리**되기 때문이다.

---

## 개선 2: 주문 생성 비동기 처리

### 변경 내용

- 주문 ID 선발급 (UUID) 후 즉시 응답
- 재고 차감 후 주문 정보를 Redis List에 등록 (`LPUSH`)
- 백그라운드 워커가 큐에서 주문을 꺼내 DB INSERT (`BRPOP`)
- 환경 변수 `ASYNC_ORDER_ENABLED=true`로 활성화

### 동작 방식

```
[주문 요청]
POST /orders
  → UUID 선발급
  → 재고 차감 (Redis Lua Script)
  → Redis LPUSH (order:queue)
  → 202 Accepted + 주문 ID 즉시 반환

[백그라운드 워커]
  → Redis BRPOP (order:queue) 대기
  → Order INSERT (DB)
  → OrderItem INSERT (DB)
  → 상태: CONFIRMED
```

### 기술 선택: Redis List

| 방법                       | 복잡도 | 안정성 | 다중 서버 | 선택  |
| -------------------------- | ------ | ------ | --------- | ----- |
| Go Channel + Goroutine     | 낮음   | 낮음   | X         |       |
| **Redis List**             | 중간   | 중간   | **O**     | **✓** |
| DB Outbox Pattern          | 중간   | 높음   | O         |       |
| 메시지 큐 (Kafka/RabbitMQ) | 높음   | 높음   | O         |       |

**Redis List 선택 이유:**

1. **이미 Redis 사용 중**: 추가 인프라 없이 구현 가능
2. **다중 서버 지원**: Go Channel은 단일 인스턴스에서만 동작하지만, Redis List는 여러 Order Service 인스턴스가 동일한 큐를 공유 가능
3. **서버 재시작 시 데이터 보존**: Go Channel은 메모리에만 존재하여 재시작 시 유실되지만, Redis는 영속성 보장
4. **구현 복잡도**: 메시지 큐(Kafka/RabbitMQ) 대비 간단한 구현

### 성능

| 지표          | Before    | After     | 개선    |
| ------------- | --------- | --------- | ------- |
| p95 응답시간  | 202ms     | 162ms     | 1.2배 ↓ |
| 평균 응답시간 | 131ms     | 83ms      | 1.6배 ↓ |
| 처리량        | 1667req/s | 2668req/s | 1.6배 ↑ |
| 에러율        | 0%        | 0%        | -       |

### 리소스 사용량 비교

| 컨테이너        | Before | After | 변화     |
| --------------- | ------ | ----- | -------- |
| Order Service   | 98%    | 107%  | +9%      |
| Product Service | 71%    | 85%   | +14%     |
| PostgreSQL      | 80%    | 11%   | **-69%** |
| Redis           | 17%    | 29%   | +12%     |

### 분석

- **처리량 1.6배 향상**: DB INSERT 대기 없이 즉시 응답하여 더 많은 요청 처리 가능
- **PostgreSQL CPU 80% → 11%**: DB INSERT가 백그라운드로 이동하여 동기 요청 처리에서 제외
- **Redis CPU 증가**: 주문 큐 처리 부하 증가 (17% → 29%)
- **Order/Product Service CPU 증가**: 더 많은 요청을 처리하면서 자연스럽게 증가
- **응답시간 단축**: DB INSERT 대기 시간이 제거되어 p95 202ms → 162ms로 감소

---

## 전체 개선 요약

| 단계                 | 처리량    | p95 응답 | Order CPU | Product CPU | Postgres CPU |
| -------------------- | --------- | -------- | --------- | ----------- | ------------ |
| 기존 (DB 재고)       | 949req/s  | 313ms    | 69%       | 83%         | 73%          |
| 개선 1 (Redis 재고)  | 1667req/s | 202ms    | Order 98% | 71%         | 80%          |
| 개선 2 (비동기 주문) | 2668req/s | 162ms    | 107%      | 85%         | 11%          |

### 누적 개선 효과

| 지표           | 기존 → 최종      | 개선율      |
| -------------- | ---------------- | ----------- |
| 처리량         | 949 → 2668 req/s | **2.8배 ↑** |
| p95 응답시간   | 313 → 162ms      | **1.9배 ↓** |
| PostgreSQL CPU | 73% → 11%        | **-62%**    |
