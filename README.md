# Flash Deals

선착순 핫딜 시스템을 MSA로 구축하고, 성능 병목을 발견하여 단계별로 개선해 나가는 과정을 기록한 포트폴리오 프로젝트입니다.

각 성능 문제는 k6 부하 테스트로 재현하고, 개선 전후의 지표를 정량적으로 비교합니다.

## Performance Improvements

### [상품 목록 조회가 느림](docs/scenarios/product-list-slow.md)

> 상품 목록 조회 시 매번 DB를 호출하여 응답 지연 발생

| 개선 | 내용                     | p95 응답시간  | 처리량             |
| ---- | ------------------------ | ------------- | ------------------ |
| 기존 | DB 직접 조회             | 738ms         | 17 req/s           |
| 1차  | Redis 캐싱 (Cache-Aside) | 84ms (8.8배↓) | 60 req/s (3.5배↑)  |

### [주문 목록 조회가 느림](docs/scenarios/order-list-slow.md)

> 주문 목록 조회 시 N+1 쿼리로 인해 주문 수에 비례하여 응답 지연

| 개선 | 내용      | p95 응답시간   | 처리량            |
| ---- | --------- | -------------- | ----------------- |
| 기존 | N+1 쿼리  | 410ms          | 48 req/s          |
| 1차  | JOIN 쿼리 | 361ms (1.1배↓) | 52 req/s (1.1배↑) |

### [핫딜 트래픽 급증](docs/scenarios/deal-traffic-spike.md)

> 핫딜 오픈 시 트래픽이 급증하면 Gateway가 병목이 되어 전체 시스템 지연

| 개선 | 내용                             | p95 응답시간   | 처리량              |
| ---- | -------------------------------- | -------------- | ------------------- |
| 기존 | Python Gateway (httpx 매번 생성) | 1089ms         | 133 req/s           |
| 1차  | httpx 커넥션 풀 재사용           | 295ms (3.7배↓) | 498 req/s (3.7배↑)  |
| 2차  | Kong Gateway 도입                | 129ms (8.4배↓) | 982 req/s (7.4배↑)  |

### [인증 CPU 병목](docs/scenarios/auth-cpu-bottleneck.md)

> 모든 요청마다 Auth 서비스에서 JWT 검증을 수행하여 CPU 포화

| 개선 | 내용                    | p95 응답시간   | 처리량                |
| ---- | ----------------------- | -------------- | --------------------- |
| 기존 | Auth 서비스 직접 호출   | 235ms          | 900 req/s             |
| 1차  | Kong JWT + Proxy Cache  | 20ms (11.8배↓) | 11,337 req/s (12.6배↑) |

### [주문 처리량 한계](docs/scenarios/order-tps-limit.md)

> Python 기반 Order/Product 서비스의 처리량이 낮아 동시 주문 처리에 한계

| 개선 | 내용                    | p95 응답시간   | 처리량               |
| ---- | ----------------------- | -------------- | -------------------- |
| 기존 | HTTP/JSON 통신          | 867ms          | 46 req/s             |
| 1차  | gRPC 전환               | 367ms (2.4배↓) | 125 req/s (2.7배↑)   |
| 2차  | Uvicorn 2워커           | 224ms (3.9배↓) | 208 req/s (4.5배↑)   |
| 3차  | Product Go 전환         | 154ms (5.6배↓) | 253 req/s (5.5배↑)   |
| 4차  | Order Go 전환           | 20ms (43배↓)   | 295 req/s (6.4배↑)   |

### [핫딜 주문 시 Lock 경합](docs/scenarios/deal-lock-contention.md)

> 동시 주문 시 DB Row Lock 경합으로 인해 대기 시간 증가 및 처리량 저하

| 개선 | 내용                 | p95 응답시간 | 처리량             |
| ---- | -------------------- | ------------ | ------------------ |
| 기존 | SELECT FOR UPDATE    | 19ms         | 160 req/s          |
| 1차  | Go Channel (순차)    | 14ms (26%↓)  | 303 req/s (1.9배↑) |

### [핫딜 주문 지연](docs/scenarios/deal-order-latency.md)

> 대량 주문 시 DB 재고 차감과 주문 INSERT가 동기적으로 처리되어 응답 지연

| 개선 | 내용                      | p95 응답시간   | 처리량               |
| ---- | ------------------------- | -------------- | -------------------- |
| 기존 | Go Channel + DB 재고      | 313ms          | 949 req/s            |
| 1차  | Redis 재고 (Lua Script)   | 202ms (1.5배↓) | 1667 req/s (1.8배↑)  |
| 2차  | 비동기 주문 (Redis List)  | 162ms (1.9배↓) | 2668 req/s (2.8배↑)  |

## Documentation

- [Architecture](docs/architecture.md)
- [API Specifications](docs/api-spec/)
- [Project Plan](references/Plan.md)
