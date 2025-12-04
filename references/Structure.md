# Architecture & Directory Structure Guide

이 문서는 프로젝트의 폴더 구조 원칙과 확장성(언어 전환 및 성능 최적화) 전략을 정의합니다.

## 1. Directory Structure (Polyglot Monorepo)

핵심 원칙:

- Interface Separation: API 명세(api/)와 데이터 쿼리(db/)는 특정 언어에 종속되지 않고 공유된다.

- Implementation Isolation: 각 서비스의 구현 언어(python, go)는 물리적으로 완벽히 격리된다.

- Infrastructure as Code: 전체 실행 및 테스트 환경은 루트 레벨에서 통합 관리된다.

```text
flash-deal/
├── docker-compose.yml # [Orchestration] 전체 서비스 실행 및 Env 제어
├── Makefile # [Automation] 빌드, 실행, 테스트 자동화
├── README.md
│
├── api/ # [Contract] OpenAPI(Swagger) Specs
│ ├── auth-service.v1.yaml # 언어가 바뀌어도 클라이언트와의 계약(Interface)은 유지
│ └── product-service.v1.yaml
│
├── infra/ # [Infrastructure]
│ ├── signoz/ # Monitoring (APM, Logs, Traces)
│ └── k6/ # Performance Test Scripts
│
├── tests/ # [Verification] E2E Integration Tests
│ ├── scenarios/ # Tavern YAML (언어 무관 테스트 시나리오)
│ └── conftest.py
│
└── services/ # [Microservices]
├── gateway/ # API Gateway (Python FastAPI)
│
└── product/ # ★ Service Structure Example
├── db/ # [Source of Truth] Shared Data Layer
│ ├── schema.sql # Database Schema (DDL)
│ └── query.sql # Business Logic Queries
│
├── sqlc.yaml # [Code Gen Config] SQL -> Python/Go Code 매핑
│
├── python/ # [Impl A] Python Implementation (Phase 1)
│ ├── Dockerfile
│ ├── pyproject.toml
│ ├── src/
│ │ ├── main.py # Entrypoint (DI Container 역할)
│ │ ├── config.py # Env Loading
│ │ ├── service.py # Business Logic
│ │ ├── repository/ # [Pattern] Data Access Layer
│ │ │ ├── base.py # Interface (Abstract Class)
│ │ │ ├── rdb.py # Impl: DB Only
│ │ │ └── cached.py # Impl: Redis + DB (Decorator/Proxy)
│ │ └── generated/ # sqlc output (Auto-generated)
│
└── go/ # [Impl B] Golang Implementation (Phase 4)
├── Dockerfile
├── go.mod
└── internal/
└── generated/ # sqlc output (Auto-generated)
```

## 2. Scalability & Optimization Strategy

이 프로젝트는 4년 차 개발자의 역량을 보여주기 위해 3단계의 확장/최적화 전략을 사용합니다.

### Strategy A. Internal Logic Optimization (Design Pattern)

"같은 언어(Python) 내에서 캐싱 적용 유무를 구조적으로 분리한다."

- Problem: 캐싱 로직이 비즈니스 로직과 뒤섞이면 유지보수가 어렵고, 성능 비교(A/B Test)가 힘들다.

- Solution: Strategy Pattern + Dependency Injection (DI)

  - Interface (base.py): get_product(id) 메서드 정의.

  - Basic Strategy (rdb.py): 단순히 DB에서 데이터를 조회.

  - Cached Strategy (cached.py): Redis를 먼저 확인하고(Look aside), 없으면 rdb.py를 호출한 뒤 Write Back.

  - Execution: docker-compose.yml의 ENABLE_CACHE 환경변수에 따라 main.py에서 주입되는 객체가 달라짐.

### Strategy B. Polyglot Switching (Language Performance)

"비즈니스 로직(SQL)을 유지한 채 런타임 언어를 교체하여 성능을 극대화한다."

- Problem: 언어를 변경(Python -> Go)할 때 비즈니스 로직(SQL)을 다시 작성하다가 휴먼 에러가 발생한다.

- Solution: SQL First Development (sqlc)

  - 모든 쿼리는 services/{service}/db/query.sql에 작성.

  - sqlc generate 명령 하나로 Python용 aio-pg 코드와 Go용 pgx 코드를 동시에 생성.

  - 개발자는 생성된 Type-safe 함수를 호출하기만 하면 됨. -> 로직의 100% 이식성 보장.

### Strategy C. Architecture Evolution (Sync to Async)

"강한 결합을 끊고 대용량 트래픽을 처리할 수 있는 구조로 진화한다."

- Phase 1 (Sync): HTTP/REST

  - 구현이 쉽고 직관적이지만, 한 서비스의 장애가 전체로 전파됨.

- Phase 2 (Async): Event-Driven (Kafka)

  - 주문 서비스는 "주문 접수됨" 이벤트만 발행하고 즉시 응답.

  - 재고 서비스 등이 이벤트를 구독하여 처리.

  - 확장성: 트래픽 폭주 시 메시지 큐가 버퍼 역할을 하여 DB 부하를 조절(Backpressure).
