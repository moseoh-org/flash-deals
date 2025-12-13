# TODO: 핫딜 주문 지연 시나리오 구현

이 문서는 `docs/scenarios/deal-order-latency.md` 시나리오를 완성하기 위한 작업 목록입니다.

---

## 1. 문서 양식 정리

### 목표
`docs/scenarios/deal-order-latency.md`를 `docs/scenarios/TEMPLATE.md` 양식에 맞게 재작성

### 참고 파일
- 템플릿: `docs/scenarios/TEMPLATE.md`
- 예시: `docs/scenarios/deal-traffic-spike.md`, `docs/scenarios/order-tps-limit.md`

### 작업 내용
- 기존 시스템 성능 테이블 추가 (p95, 평균, 처리량, 에러율)
- 리소스 사용량 테이블 추가 (CPU)
- 개선 1 섹션을 Redis 재고 시스템으로 작성
- 전체 개선 요약 테이블 추가

---

## 2. 주문 시스템 아키텍처 변경

### 현재 구조 (동기 처리)
```
POST /orders
    ↓
Order Service
    ↓ (gRPC)
Product Service - Go Channel 단일 워커 - DB 재고 차감
    ↓
주문 생성 (DB)
    ↓
응답 반환 (200 OK + 주문 정보)
```

**문제점**: 재고 차감(DB)이 병목. 단일 워커가 순차 처리하므로 I/O bound.

### 변경할 구조 (요청 분리)
```
[Phase 1: 재고 예약 - 즉시 응답]
POST /orders
    ↓
Order Service
    ↓ (gRPC)
Product Service - 재고 차감 (현재: DB, 개선 후: Redis)
    ↓
주문 생성 (DB) - 상태: PENDING
    ↓
응답 반환 (202 Accepted + 주문 ID)

[Phase 2: 주문 확정 - 별도 요청]
POST /orders/{id}/confirm  (또는 결제 완료 콜백)
    ↓
주문 상태 변경: PENDING → CONFIRMED
    ↓
응답 반환 (200 OK)
```

### 구현 필요 사항

#### Order Service 변경
- `POST /orders` 응답을 202 Accepted로 변경
- 주문 상태 필드 추가: `PENDING`, `CONFIRMED`, `CANCELLED`
- 주문 확정 엔드포인트 추가: `POST /orders/{id}/confirm`

#### Product Service 변경 (4단계에서 진행)
- Redis 재고 관리 추가
- Lua Script로 원자적 재고 차감

#### 데이터베이스 변경
- orders 테이블에 status 컬럼 추가 (없다면)

---

## 3. 기존 시스템 부하 테스트

### 목표
현재 Go Channel + DB 재고 시스템의 성능 측정

### 테스트 명령
```bash
make load-order-overload MAX_VUS=100 NUM_USERS=100
```

### 측정 항목
- p95 응답시간
- 평균 응답시간
- 처리량 (req/s)
- 에러율 (%)
- CPU 사용량 (Order, Product, Gateway, Postgres)

### 결과 기록 위치
`docs/scenarios/deal-order-latency.md` → 기존 시스템 → 성능 테이블

---

## 4. Redis 재고 시스템 구현

### 목표
Product Service의 재고 관리를 Redis로 변경

### 구현 내용

#### 4.1 Redis 재고 로드 (핫딜 시작 전)
```go
// 핫딜 상품의 DB 재고를 Redis에 로드
HSET hotdeal:stock:{product_id} quantity {stock}
```

엔드포인트 또는 스케줄러:
- `POST /products/{id}/hotdeal/start` - 핫딜 시작 (Redis에 재고 로드)
- `POST /products/{id}/hotdeal/end` - 핫딜 종료 (Redis → DB 동기화)

#### 4.2 Redis 재고 차감 (Lua Script)
```lua
-- 원자적 재고 차감
local stock = redis.call('HGET', KEYS[1], 'quantity')
if stock and tonumber(stock) >= tonumber(ARGV[1]) then
    redis.call('HINCRBY', KEYS[1], 'quantity', -tonumber(ARGV[1]))
    return 1  -- 성공
else
    return 0  -- 재고 부족
end
```

#### 4.3 Product Service 변경
- `DecreaseStock` 메서드에서 핫딜 상품 여부 확인
- 핫딜 상품: Redis Lua Script로 차감
- 일반 상품: 기존 Go Channel + DB 방식 유지

#### 4.4 환경 변수
```yaml
# docker-compose.yml
HOTDEAL_STOCK_REDIS: true  # Redis 재고 사용 여부
```

### 파일 변경 예상
- `services/product/go/internal/service/product.go` - DecreaseStock 수정
- `services/product/go/internal/repository/` - Redis 재고 레포지토리 추가
- `services/product/go/cmd/server/main.go` - Redis 클라이언트 초기화

---

## 5. 개선 후 부하 테스트 및 비교

### 목표
Redis 재고 시스템의 성능 측정 및 기존 대비 비교

### 테스트 절차
1. 핫딜 상품 생성 및 Redis 재고 로드
2. 동일 조건으로 부하 테스트 실행
3. 결과 측정 및 기록

### 기대 결과
| 항목 | 기존 (DB) | 개선 (Redis) | 개선율 |
|------|-----------|--------------|--------|
| 재고 차감 | ~5ms | ~0.1ms | 50배↓ |
| p95 응답시간 | TODO | TODO | ?배↓ |
| 처리량 | TODO | TODO | ?배↑ |

### 결과 기록 위치
`docs/scenarios/deal-order-latency.md` → 개선 1: Redis 재고 시스템

---

## 체크리스트

- [ ] 1. 문서 양식 정리 (TEMPLATE.md 참고)
- [ ] 2. 주문 시스템 아키텍처 변경
  - [ ] 2.1 Order status 추가 (PENDING, CONFIRMED)
  - [ ] 2.2 응답 코드 202 Accepted로 변경
  - [ ] 2.3 주문 확정 엔드포인트 추가
- [ ] 3. 기존 시스템 부하 테스트 및 기록
- [ ] 4. Redis 재고 시스템 구현
  - [ ] 4.1 Redis 재고 로드 기능
  - [ ] 4.2 Lua Script 재고 차감
  - [ ] 4.3 Product Service 수정
- [ ] 5. 개선 후 부하 테스트 및 비교 분석

---

## 참고 사항

### 현재 코드 위치
- Order Service: `services/order/go/`
- Product Service: `services/product/go/`
- 재고 큐: `services/product/go/internal/queue/stock_queue.go`

### 관련 문서
- 시나리오: `docs/scenarios/deal-order-latency.md`
- 아키텍처: `docs/architecture.md`
