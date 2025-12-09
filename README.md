# Flash Deals

선착순 핫딜 시스템을 MSA로 구축하고, 성능 병목을 발견하여 단계별로 개선해 나가는 과정을 기록한 포트폴리오 프로젝트입니다.

각 성능 문제는 k6 부하 테스트로 재현하고, 개선 전후의 지표를 정량적으로 비교합니다.

## Performance Scenarios

| Scenario                                                           | Problem               | Solution              |
| ------------------------------------------------------------------ | --------------------- | --------------------- |
| [상품 목록 조회가 느림](docs/scenarios/product-list-slow.md)       | DB 반복 조회          | Redis 캐싱, gzip 압축 |
| [상품 대량 등록이 느림](docs/scenarios/product-insert-slow.md)     | UUID v4 fragmentation | UUID v7 전환          |
| [주문 목록 조회가 느림](docs/scenarios/order-list-slow.md)         | N+1 쿼리              | JOIN 쿼리             |
| [핫딜 트래픽 급증](docs/scenarios/deal-traffic-spike.md)           | 순간 트래픽 폭주      | 캐싱, Rate Limiting   |
| [동시 주문 재고 초과](docs/scenarios/concurrent-order-oversell.md) | Race Condition        | 분산 락 (Redlock)     |
| [서비스 장애 연쇄 전파](docs/scenarios/service-cascade-failure.md) | 장애 전파             | Circuit Breaker       |
| [인증 CPU 병목](docs/scenarios/auth-cpu-bottleneck.md)             | Python GIL            | Go 전환               |
| [주문 처리량 한계](docs/scenarios/order-tps-limit.md)              | 동기 처리             | gRPC, Kafka           |
| [Gateway 처리량 한계](docs/scenarios/gateway-throughput-limit.md)  | Python 한계           | Kong Gateway          |

## Documentation

- [Architecture](docs/architecture.md)
- [API Specifications](docs/api-spec/)
- [Project Plan](references/Plan.md)
