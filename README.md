# Flash Deals

선착순 핫딜 시스템을 MSA로 구축하고, 성능 병목을 발견하여 단계별로 개선해 나가는 과정을 기록한 포트폴리오 프로젝트입니다.

각 성능 문제는 k6 부하 테스트로 재현하고, 개선 전후의 지표를 정량적으로 비교합니다.

## Performance Improvements

### [상품 목록 조회가 느림](docs/scenarios/product-list-slow.md)

| 개선 | 내용                     | p95 응답시간  | 처리량             |
| ---- | ------------------------ | ------------- | ------------------ |
| 기존 | DB 직접 조회             | 738ms         | 17 req/s           |
| 1차  | Redis 캐싱 (Cache-Aside) | 84ms (8.8배↓) | 60 req/s (3.5배↑)  |

### [주문 목록 조회가 느림](docs/scenarios/order-list-slow.md)

| 개선 | 내용      | p95 응답시간  | 처리량            |
| ---- | --------- | ------------- | ----------------- |
| 기존 | N+1 쿼리  | 274ms         | 36 req/s          |
| 1차  | JOIN 쿼리 | 242ms (12%↓)  | 41 req/s (14%↑)   |

### [핫딜 트래픽 급증](docs/scenarios/deal-traffic-spike.md)

| 개선 | 내용                             | p95 응답시간   | 처리량              |
| ---- | -------------------------------- | -------------- | ------------------- |
| 기존 | Python Gateway (httpx 매번 생성) | 1089ms         | 133 req/s           |
| 1차  | httpx 커넥션 풀 재사용           | 295ms (3.7배↓) | 498 req/s (3.7배↑)  |
| 2차  | Kong Gateway 도입                | 129ms (8.4배↓) | 982 req/s (7.4배↑)  |

### [인증 CPU 병목](docs/scenarios/auth-cpu-bottleneck.md)

| 개선 | 내용                    | p95 응답시간   | 처리량                |
| ---- | ----------------------- | -------------- | --------------------- |
| 기존 | Auth 서비스 직접 호출   | 235ms          | 900 req/s             |
| 1차  | Kong JWT + Proxy Cache  | 20ms (11.8배↓) | 11,337 req/s (12.6배↑) |

### (예정) [서비스 장애 연쇄 전파](docs/scenarios/service-cascade-failure.md)

## Documentation

- [Architecture](docs/architecture.md)
- [API Specifications](docs/api-spec/)
- [Project Plan](references/Plan.md)
