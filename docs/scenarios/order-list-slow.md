# 주문 목록 조회가 느림

## 상황

주문이 많은 VIP 고객이 주문 내역 페이지를 열 때 로딩이 매우 느리다.
주문이 적은 일반 사용자는 정상인데, 주문이 많을수록 느려지는 패턴이 발견되었다.

## 테스트 환경

- PostgreSQL 16 (Docker)
- 사용자당 주문 100건
- 부하: 20 VU, 30초

---

## 기존 시스템

주문 목록 조회 시 N+1 쿼리 발생:

1. 주문 20건 조회 (1번 쿼리)
2. 각 주문의 아이템 조회 (20번 쿼리)
3. **총 21번 쿼리**

```python
async for order in orders_iter:
    items = await querier.get_order_items_by_order_id(order_id=order.id)  # N+1!
```

### 성능

| 지표          | 값          |
| ------------- | ----------- |
| p95 응답시간  | 410ms       |
| 평균 응답시간 | 285ms       |
| TPS           | 47.58 req/s |

### 리소스 사용량

| 컨테이너 | CPU | RAM   |
| -------- | --- | ----- |
| postgres | 27% | 168MB |
| order    | 57% | 90MB  |
| gateway  | 57% | 142MB |

---

## 개선 1: JOIN 쿼리로 N+1 해결

### 변경 내용

- 주문 + 아이템을 LEFT JOIN으로 한 번에 조회
- Python에서 결과를 주문별로 그룹핑
- **21번 쿼리 → 1번 쿼리**

```sql
SELECT o.*, i.*
FROM orders.orders o
LEFT JOIN orders.order_items i ON o.id = i.order_id
WHERE o.user_id = $1
ORDER BY o.created_at DESC
LIMIT $2 OFFSET $3;
```

### 성능

| 지표          | Before | After | 개선  |
| ------------- | ------ | ----- | ----- |
| p95 응답시간  | 410ms  | 361ms | 12% ↓ |
| 평균 응답시간 | 285ms  | 252ms | 12% ↓ |
| TPS           | 47.58  | 51.74 | 9% ↑  |

### 리소스 사용량 비교

| 컨테이너 | Before (CPU) | After (CPU) | Before (RAM) | After (RAM) |
| -------- | ------------ | ----------- | ------------ | ----------- |
| postgres | 27%          | 32%         | 168MB        | 69MB        |
| order    | 57%          | 49%         | 90MB         | 91MB        |
| gateway  | 57%          | 47%         | 142MB        | 100MB       |

### 분석

- 쿼리 횟수 21번 → 1번으로 감소
- 개선폭이 12%로 제한적인 이유:
  - 인덱스(`idx_order_items_order_id`)가 있어서 개별 쿼리도 빠름 (0.4ms)
  - 실제 병목은 Gateway 경유, JWT 검증, Python 처리 오버헤드
