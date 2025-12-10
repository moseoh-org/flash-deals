# PLAN.md: Project Flash Deal

> **High Concurrency & Polyglot MSA Architecture Portfolio**
> 4년 차 백엔드 개발자의 기술 검증 및 성능 최적화 R&D 프로젝트

## 1. Project Overview

**목표:** 대규모 트래픽이 발생하는 '선착순 핫딜' 시스템을 MSA로 구축하며, 단계별 아키텍처 진화와 언어별 성능 차이를 정량적으로 증명한다.

### 🛠 Core Tech Stack (Phase 1)

- **Language:** Python 3.12 (FastAPI)
- **Database:** PostgreSQL 16 (Asyncpg driver)
- **ORM/Query:** `sqlc` (SQL First Code Generation)
- **Migration:** `golang-migrate` (언어 독립적 SQL 마이그레이션)
- **Architecture:** Microservices (Docker Compose base)
- **Communication:** REST API (HTTP 1.1)

---

## 2. Roadmap & Milestones

### 🚀 Phase 1: MVP 구축 (The Foundation)

**목표:** 가장 단순한 구조로 핵심 비즈니스 로직(로그인, 상품조회, 주문)이 동작해야 한다.

- [x] **1. API Contract Definition (API First)**
  - [x] api-spec/auth-service.v1.yaml 작성
  - [x] api-spec/product-service.v1.yaml 작성
  - [x] api-spec/order-service.v1.yaml 작성
  - [x] Review: 각 명세서의 Request/Response 필드 확정
- [x] **2. 인프라 기본 설정**
  - [x] Monorepo 폴더 구조 생성 (services/, infra/, tests/)
  - [x] Docker Compose 환경 구성 (PostgreSQL, Redis)
- [x] **3. Gateway (Python FastAPI)** - 기본 라우팅
  - [x] Dockerfile, pyproject.toml 설정 (uv 사용)
  - [x] `httpx`를 이용한 요청 라우팅 (Proxy)
  - [x] 공개 경로(`/auth/*`) 설정
- [x] **4. Auth Service (Python)**
  - [x] DB Schema/Query 작성 및 sqlc 설정
  - [x] JWT 발급 및 검증 로직 구현
  - [x] User 회원가입/로그인 API
- [x] **5. Gateway 고도화** - JWT 검증 추가
  - [x] JWT 검증 Middleware 구현
  - [x] 인증 필요 경로에 Middleware 적용
- [x] **6. Product Service (Python)**
  - [x] DB Schema/Query 작성 및 sqlc 설정
  - [x] 상품 등록 및 재고 조회 API
  - [x] 핫딜 API
- [x] **7. Order Service (Python)**
  - [x] DB Schema/Query 작성 및 sqlc 설정
  - [x] 주문 생성 API (단순 DB Transaction)
- [x] **8. 통합 Makefile 완성**
  - [x] 빌드, 실행, 테스트 자동화 명령어

### 📊 Phase 2: 측정 환경 구축 (The Baseline)

**목표:** 최적화 전, 현재 시스템의 성능 지표를 수집하여 비교군(Control Group)을 형성한다.

- [x] **Integration Testing (API)**
  - [x] **Tool:** `pytest-playwright` (Python, uv 사용)
  - [x] **Location:** `tests/integration/` (서비스 독립적, API 명세 기반)
  - [x] API Spec(Swagger) 기반의 통합 테스트 시나리오 작성 (Code-based)
  - [ ] CI 파이프라인 연동 준비 (Github Actions 등)
- [x] **Observability (Monitoring)**
  - [x] **Tool:** OpenTelemetry + Grafana Stack (Tempo, Prometheus)
  - [x] Python 앱에 OTel Auto-Instrumentation 적용
  - [x] 통합 대시보드 구성 (Traces, Metrics)
- [x] **Load Testing (Performance)**
  - [x] **Tool:** `k6`
  - [x] 시나리오 스크립트 작성 (health-check, auth-flow, order-scenario)
  - [x] Baseline 성능 측정 환경 구축 완료

### ⚡ Phase 3: 비즈니스 고도화 (Concurrency & Caching)

**목표:** '선착순 구매'의 핵심인 동시성 문제를 해결하고 성능을 최적화한다.

- [ ] **Redis 도입**
  - [ ] Global Caching 전략 수립 (Product 조회 성능 개선)
  - [ ] **Distributed Lock (Redlock/Lua Script)** 구현: 재고 차감 동시성 이슈 해결
- [ ] **Database Optimization**
  - [ ] PostgreSQL Index 튜닝 및 실행 계획 분석 (`EXPLAIN ANALYZE`)

### 🧪 Phase 4: 언어 전환 실험 (The R&D Experiment)

**목표:** 특정 서비스(Auth 또는 Order)를 고성능 언어로 교체하여 **"비용 대비 성능 효율"**을 증명한다.

- [ ] **Target 선정:** CPU 연산이 많은 `Auth Service` 또는 트래픽이 몰리는 `Order Service`
- [ ] **Re-implementation (Polyglot)**
  - [ ] **Golang** 또는 **Kotlin(Spring Boot)** 으로 해당 서비스 재작성
  - [ ] `sqlc` 설정을 변경하여 동일한 SQL로 Go/Kotlin 코드 생성
- [ ] **Performance Battle**
  - [ ] 동일한 `k6` 테스트 수행
  - [ ] Python 버전 vs Go/Kotlin 버전 성능 비교 리포트 작성 (Throughput, Memory Usage 등)

### 🏗 Phase 5: 아키텍처 진화 (Event-Driven & Infra)

**목표:** 시스템 결합도를 낮추고 운영 안정성을 확보한다.

- [ ] **Message Queue 도입**
  - [ ] Kafka (or RabbitMQ) 구축
  - [ ] 주문 프로세스 리팩토링: 동기 호출(HTTP) -> 비동기 이벤트(Event-Driven)
- [ ] **Gateway 고도화**
  - [ ] Python Gateway -> **Kong Gateway** (또는 Spring Cloud Gateway) 전환
  - [ ] Rate Limiting(유량 제어) 적용

---

## 3. Development Conventions

### 📜 API First Design

- 코딩 전 **OpenAPI(Swagger) Spec**을 먼저 작성한다.
- 언어가 바뀌어도 API 인터페이스(Endpoint, Request/Response Body)는 절대 변경하지 않는다.

### 💾 SQL First Development

- ORM(JPA, SQLAlchemy)의 객체 매핑 기능에 의존하지 않는다.
- 모든 비즈니스 쿼리는 `.sql` 파일에 작성하고 `sqlc`로 코드를 생성해서 사용한다.
- **Why?** 언어 교체 시 비즈니스 로직(SQL)의 이식성을 100% 보장하기 위함.

### 🐳 Database Strategy

- **Physical:** 1개의 PostgreSQL 인스턴스 (리소스 절약).
- **Logical:** 서비스별로 엄격하게 `Database` (또는 `Schema`)를 분리.
- 타 서비스의 테이블에 직접 `JOIN` 금지. 오직 API를 통해서만 데이터 접근.

---

## 4. 성능 개선 시나리오 (Performance Optimization Roadmap)

> 각 시나리오는 **k6 부하테스트**로 Before/After 성능을 정량적으로 측정한다.
> 실무 스토리: **문제 발견 조건 → 원인 분석 → 해결책 적용 → 성능 검증**

---

### 🟢 Easy (난이도 하) - 빠른 성과 달성

#### E1. UUID v4 → v7 전환

| 항목          | 내용                                                                                                   |
| ------------- | ------------------------------------------------------------------------------------------------------ |
| **문제 발견** | 핫딜 이벤트로 **상품 등록이 급증**하면서 INSERT 성능이 저하됨. 동일 시간 내 등록 가능한 상품 수가 감소 |
| **증상**      | 상품 대량 등록 시 응답시간 점진적 증가, DB CPU 사용률 상승                                             |
| **원인 분석** | UUID v4는 랜덤값이라 B-Tree 인덱스에서 페이지 분할(page split)이 빈번하게 발생                         |
| **해결책**    | UUID v7 (시간 기반 정렬)으로 전환하여 순차적 INSERT 유도                                               |
| **측정 방법** | 10,000건 상품 INSERT 후 p95 응답시간 비교                                                              |
| **k6 테스트** | `product-insert.js` - 대량 상품 생성 부하                                                              |

#### E2. 상품 목록 캐싱 (Redis)

| 항목          | 내용                                                                            |
| ------------- | ------------------------------------------------------------------------------- |
| **문제 발견** | 메인 페이지 트래픽 증가로 **상품 목록 조회 API가 병목**이 됨. DB 커넥션 풀 고갈 |
| **증상**      | GET /products 응답시간 500ms 초과, DB 커넥션 대기 발생                          |
| **원인 분석** | 동일한 상품 목록을 매 요청마다 DB에서 조회. 상품 데이터는 자주 변경되지 않음    |
| **해결책**    | Redis에 상품 목록 캐싱 (TTL 60초), Cache-Aside 패턴 적용                        |
| **측정 방법** | 100 VU로 상품 목록 조회, Cache Hit 시 p95 < 50ms 목표                           |
| **k6 테스트** | `product-list.js` - 상품 목록 반복 조회 부하                                    |

#### E3. 핫딜 목록 캐싱 (Redis)

| 항목          | 내용                                                                                |
| ------------- | ----------------------------------------------------------------------------------- |
| **문제 발견** | 핫딜 시작 시간에 **GET /products/deals 트래픽이 10배 급증**. 서비스 응답 지연       |
| **증상**      | 핫딜 조회 API Timeout 발생, 에러율 상승                                             |
| **원인 분석** | 핫딜은 시간 조건 + JOIN 쿼리로 상품 목록보다 무거움. 동시 사용자가 같은 데이터 요청 |
| **해결책**    | Redis 캐싱 + 핫딜 시작 전 Cache Warming 전략                                        |
| **측정 방법** | 핫딜 시작 시점 시뮬레이션, 200 VU 동시 접속                                         |
| **k6 테스트** | `deal-list.js` - 핫딜 목록 집중 조회 부하                                           |

#### E4. N+1 쿼리 해결 (Order Items)

| 항목          | 내용                                                             |
| ------------- | ---------------------------------------------------------------- |
| **문제 발견** | 주문 내역 페이지에서 **주문이 많은 사용자의 조회가 극도로 느림** |
| **증상**      | 주문 20건 조회 시 응답시간 2초 이상, 주문 수에 비례해 느려짐     |
| **원인 분석** | 주문 목록 조회 후 각 주문마다 order_items를 별도 쿼리 (N+1 문제) |
| **해결책**    | JOIN 쿼리로 한 번에 조회하거나, IN 절로 배치 로딩                |
| **측정 방법** | 주문 50건 보유 사용자의 주문 목록 조회 성능                      |
| **k6 테스트** | `order-list.js` - 주문 목록 조회 부하 (다량 주문 사용자)         |

#### E5. Connection Pool 튜닝

| 항목          | 내용                                                                 |
| ------------- | -------------------------------------------------------------------- |
| **문제 발견** | 동시 사용자 100명 이상에서 **"connection pool exhausted" 에러** 발생 |
| **증상**      | 간헐적 500 에러, 응답시간 불안정                                     |
| **원인 분석** | 기본 Connection Pool 크기가 동시 요청 수를 감당 못함                 |
| **해결책**    | Pool 크기 조정 (min/max), Connection 재사용 최적화                   |
| **측정 방법** | 동시 VU 증가시키며 에러 발생 임계점 확인                             |
| **k6 테스트** | `health-check.js` stress 모드 - 동시 연결 한계 테스트                |

#### E6. Response 압축 (gzip)

| 항목          | 내용                                                       |
| ------------- | ---------------------------------------------------------- |
| **문제 발견** | 모바일 사용자에서 **상품 목록 로딩이 느림**. 네트워크 지연 |
| **증상**      | 상품 100건 응답이 500KB 이상, 3G 환경에서 체감 느림        |
| **원인 분석** | JSON 응답을 압축 없이 전송. 특히 목록 API에서 데이터량 큼  |
| **해결책**    | FastAPI에 GZip 미들웨어 적용 (minimum_size=1000)           |
| **측정 방법** | 응답 크기 비교 (압축 전/후), 네트워크 전송 시간            |
| **k6 테스트** | `product-list.js` - 응답 크기 및 전송 시간 측정            |

#### E7. Pagination COUNT 캐싱

| 항목          | 내용                                                             |
| ------------- | ---------------------------------------------------------------- |
| **문제 발견** | 상품이 100만 건 이상일 때 **목록 API가 느려짐**                  |
| **증상**      | 페이지 이동마다 300ms+ 소요, 첫 페이지도 느림                    |
| **원인 분석** | 매 요청마다 COUNT(\*) 쿼리 실행. 대량 데이터에서 Full Scan       |
| **해결책**    | Total Count를 Redis에 캐싱, 상품 변경 시 갱신 (또는 근사값 사용) |
| **측정 방법** | 100만 건 상품 테이블에서 페이지네이션 성능                       |
| **k6 테스트** | `product-list.js` - 대량 데이터 페이지네이션                     |

---

### 🟡 Medium (난이도 중) - 아키텍처 개선

#### ~~M1. 동시 주문 락 최적화 (FOR UPDATE → Redlock)~~ ❌ 불필요

| 항목          | 내용                                                                    |
| ------------- | ----------------------------------------------------------------------- |
| **문제 발견** | 인기 핫딜 동시 주문 시 **DB 락 대기로 응답시간 지연**                   |
| **증상**      | 100명 동시 주문 시 p95 응답시간 1200ms+, 락 직렬화로 인한 병목          |
| **원인 분석** | `FOR UPDATE`는 DB 레벨 락으로 트랜잭션 전체 시간 동안 다른 요청 대기    |
| **해결책**    | Redis 분산 락 (Redlock)으로 애플리케이션 레벨 락 전환, 락 범위 최소화   |
| **측정 방법** | 100명 동시 주문 시 p95 응답시간, 평균 응답시간 비교                     |
| **k6 테스트** | `concurrent-order.js` - 동시 주문 응답시간 테스트                       |
| **결론**      | ❌ **개선 불필요** - 단일 DB 환경에서 `FOR UPDATE`가 이미 동시성을 안전하게 처리. Redlock은 락 획득 실패 시 "재고 부족"이 아닌 "나중에 재시도" 응답을 줘야 하므로 UX 저하. 다중 DB 샤딩 환경에서만 의미 있음. |

#### M2. JWT 검증 캐싱

| 항목          | 내용                                                 |
| ------------- | ---------------------------------------------------- |
| **문제 발견** | 인증된 API 호출마다 **JWT 검증 오버헤드**가 누적됨   |
| **증상**      | 동일 토큰으로 100번 요청해도 매번 서명 검증 수행     |
| **원인 분석** | JWT RS256 서명 검증은 CPU 연산 비용이 높음           |
| **해결책**    | 검증된 토큰을 Redis에 캐싱 (토큰 해시 → 사용자 정보) |
| **측정 방법** | 인증 API 1000회 호출 시 Gateway CPU 사용률 비교      |
| **k6 테스트** | `auth-flow.js` - 반복적인 인증 API 호출              |

#### M3. DB Index 튜닝

| 항목          | 내용                                                      |
| ------------- | --------------------------------------------------------- |
| **문제 발견** | 특정 조회 쿼리에서 **Slow Query 로그** 다수 발생          |
| **증상**      | 주문 상태별 조회, 날짜 범위 조회 등에서 1초 이상 소요     |
| **원인 분석** | 복합 조건 쿼리에 적절한 인덱스 부재, Full Table Scan 발생 |
| **해결책**    | EXPLAIN ANALYZE로 실행계획 분석, 복합 인덱스 추가         |
| **측정 방법** | 문제 쿼리 Before/After 실행 시간 비교                     |
| **k6 테스트** | `order-scenario.js` - 다양한 조회 조건 테스트             |

#### M4. Rate Limiting

| 항목          | 내용                                                             |
| ------------- | ---------------------------------------------------------------- |
| **문제 발견** | 핫딜 시작 시 **트래픽 폭주로 전체 서비스 다운**                  |
| **증상**      | 정상 사용자도 서비스 이용 불가, 연쇄 장애 발생                   |
| **원인 분석** | 트래픽 제한 없이 모든 요청 처리 시도, 리소스 고갈                |
| **해결책**    | Gateway에 Rate Limiting 적용 (IP당 100 req/s, 사용자당 50 req/s) |
| **측정 방법** | 제한 초과 시 429 응답, 제한 내에서 안정적 TPS 유지 확인          |
| **k6 테스트** | `stress-test.js` - 과부하 상황에서 안정성 테스트                 |

#### M5. Circuit Breaker

| 항목          | 내용                                                                |
| ------------- | ------------------------------------------------------------------- |
| **문제 발견** | Product Service 장애 시 **Order Service도 연쇄 장애** 발생          |
| **증상**      | Product 응답 지연 → Order 타임아웃 → Gateway 타임아웃 → 전체 마비   |
| **원인 분석** | 동기 HTTP 호출에서 장애 서비스로 계속 요청 시도                     |
| **해결책**    | Circuit Breaker 패턴 적용 (실패율 50% 이상 시 차단, 30초 후 재시도) |
| **측정 방법** | Product Service 강제 다운 후 Order Service 응답 확인                |
| **k6 테스트** | `service-failure.js` - 서비스 장애 시뮬레이션                       |

#### M6. Kong JWT 플러그인 (Gateway 인증)

| 항목          | 내용                                                                   |
| ------------- | ---------------------------------------------------------------------- |
| **문제 발견** | Protected route 접근 시 **매번 Auth 서비스 호출**로 Auth CPU 병목      |
| **증상**      | Auth 서비스 CPU 80%+ 도달, 인증 요청 응답시간 증가                     |
| **원인 분석** | Gateway가 JWT 검증을 Auth 서비스에 위임 (`/auth/verify` HTTP 호출)     |
| **해결책**    | Kong JWT 플러그인으로 Gateway에서 직접 JWT 서명 검증                   |
| **측정 방법** | 100% Protected route 부하 테스트 시 Auth CPU 사용률 비교               |
| **k6 테스트** | `auth-stress.js` - Protected route 집중 호출                           |

---

### 🔴 Hard (난이도 상) - 대규모 아키텍처 변경

#### H1. gRPC 전환 (서비스 간 통신)

| 항목          | 내용                                                                  |
| ------------- | --------------------------------------------------------------------- |
| **문제 발견** | 주문 생성 시 **Product Service 호출 지연이 전체 응답시간의 60%** 차지 |
| **증상**      | 주문 API p95가 800ms인데, 그 중 Product API 호출이 500ms              |
| **원인 분석** | HTTP/1.1 + JSON 직렬화 오버헤드, 매 요청마다 연결 설정                |
| **해결책**    | gRPC (HTTP/2 + Protocol Buffers) 전환, 연결 재사용                    |
| **측정 방법** | 서비스 간 통신 지연 시간 측정 (HTTP vs gRPC)                          |
| **k6 테스트** | `inter-service.js` - 서비스 간 호출 포함 시나리오                     |

#### H2. Kafka 비동기 처리 (주문)

| 항목          | 내용                                                                    |
| ------------- | ----------------------------------------------------------------------- |
| **문제 발견** | 핫딜 시작 시 **주문 처리량 한계 (100 TPS)에 도달**                      |
| **증상**      | 동기 처리로 인해 DB 트랜잭션 대기, TPS 증가 불가                        |
| **원인 분석** | 주문 생성 = 재고확인 + 재고차감 + 주문저장이 동기적으로 연결            |
| **해결책**    | 주문 요청을 Kafka로 비동기 처리, 최종 일관성(Eventual Consistency) 모델 |
| **측정 방법** | 주문 요청 TPS vs 실제 처리 완료 TPS 비교                                |
| **k6 테스트** | `async-order.js` - 대량 주문 처리량 테스트                              |

#### H3. Python → Go (Auth Service)

| 항목          | 내용                                                     |
| ------------- | -------------------------------------------------------- |
| **문제 발견** | 로그인 트래픽 증가 시 **Auth Service CPU 100%** 도달     |
| **증상**      | JWT 생성/검증 요청이 많아지면 Python GIL로 인한 병목     |
| **원인 분석** | JWT 암호화는 CPU-bound 작업, Python은 멀티코어 활용 제한 |
| **해결책**    | Go로 Auth Service 재작성 (goroutine으로 병렬 처리)       |
| **측정 방법** | 동일 하드웨어에서 로그인 API TPS 비교                    |
| **k6 테스트** | `auth-flow.js` - 인증 API 최대 처리량 테스트             |

#### H4. Python → Go (Order Service)

| 항목          | 내용                                                             |
| ------------- | ---------------------------------------------------------------- |
| **문제 발견** | 주문 서비스가 **전체 시스템의 병목** (가장 낮은 TPS)             |
| **증상**      | Auth, Product는 여유 있는데 Order에서 대기열 발생                |
| **원인 분석** | 주문은 여러 서비스 호출 + DB 트랜잭션으로 복잡, Python 성능 한계 |
| **해결책**    | Go로 Order Service 재작성, I/O 멀티플렉싱 최적화                 |
| **측정 방법** | 전체 주문 플로우 TPS 비교 (Python vs Go)                         |
| **k6 테스트** | `order-scenario.js` - 전체 주문 시나리오 처리량                  |

#### H5. Kong Gateway 전환

| 항목          | 내용                                                                 |
| ------------- | -------------------------------------------------------------------- |
| **문제 발견** | Python Gateway가 **높은 트래픽에서 병목**이 됨                       |
| **증상**      | 모든 요청이 Gateway를 경유하므로, Gateway TPS = 전체 시스템 TPS 상한 |
| **원인 분석** | Python + httpx 기반 프록시의 처리량 한계                             |
| **해결책**    | Kong Gateway (Nginx + Lua 기반) 또는 Envoy로 전환                    |
| **측정 방법** | Gateway 단독 처리량 측정 (프록시 성능)                               |
| **k6 테스트** | `gateway-throughput.js` - Gateway 최대 처리량 테스트                 |

---

## 5. 성능 테스트 시나리오 (k6 Scripts)

> `tests/load/scripts/` 디렉토리

### 시나리오별 테스트 스크립트

| 완료 | 시나리오                                                              | 스크립트              | 타겟 개선사항                         |
| :--: | --------------------------------------------------------------------- | --------------------- | ------------------------------------- |
|  ✅  | [상품 목록 조회가 느림](../docs/scenarios/product-list-slow.md)       | `product-list.js`     | E2, E6(로컬 효과 없음), E7(E2에 포함) |
|  ✅  | [상품 대량 등록이 느림](../docs/scenarios/product-insert-slow.md)     | `product-insert.js`   | E1(유의미한 차이 없음)                |
|  ✅  | [주문 목록 조회가 느림](../docs/scenarios/order-list-slow.md)         | `order-list.js`       | E4, M3(이미 인덱스 적용됨)            |
|  ✅  | [핫딜 트래픽 급증](../docs/scenarios/deal-traffic-spike.md)           | `deal-spike.js`       | E5, H5                                |
|  ❌  | ~~동시 주문 락 지연~~ | ~~`concurrent-order.js`~~ | ~~M1~~ (FOR UPDATE로 충분, Redlock 불필요) |
|      | [서비스 장애 연쇄 전파](../docs/scenarios/service-cascade-failure.md) | `service-failure.js`  | M5                                    |
|      | [인증 CPU 병목](../docs/scenarios/auth-cpu-bottleneck.md)             | `auth-stress.js`      | M2, M6, H3                            |
|      | [주문 처리량 한계](../docs/scenarios/order-tps-limit.md)              | `order-stress.js`     | H1, H2, H4                            |

### 기존 스크립트

| 스크립트            | 용도                            |
| ------------------- | ------------------------------- |
| `health-check.js`   | 서비스 헬스체크                 |
| `auth-flow.js`      | 회원가입→로그인→내정보→토큰갱신 |
| `order-scenario.js` | 전체 주문 플로우                |

### 측정 지표

- **Throughput (TPS)**: 초당 처리량
- **Latency**: p50, p95, p99 응답시간
- **Error Rate**: 에러 비율
- **Resource Usage**: CPU, Memory (Docker stats)
