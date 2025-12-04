# PLAN.md: Project Flash Deal

> **High Concurrency & Polyglot MSA Architecture Portfolio**
> 4년 차 백엔드 개발자의 기술 검증 및 성능 최적화 R&D 프로젝트

## 1. Project Overview

**목표:** 대규모 트래픽이 발생하는 '선착순 핫딜' 시스템을 MSA로 구축하며, 단계별 아키텍처 진화와 언어별 성능 차이를 정량적으로 증명한다.

### 🛠 Core Tech Stack (Phase 1)

- **Language:** Python 3.12 (FastAPI)
- **Database:** PostgreSQL 16 (Asyncpg driver)
- **ORM/Query:** `sqlc` (SQL First Code Generation)
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
- [ ] **2. 인프라 기본 설정**
  - [ ] Monorepo 폴더 구조 생성 (services/, infra/, tests/)
  - [ ] Docker Compose 환경 구성 (PostgreSQL, Redis)
- [ ] **3. Gateway (Python FastAPI)** - 기본 라우팅
  - [ ] Dockerfile, pyproject.toml 설정
  - [ ] `httpx`를 이용한 요청 라우팅 (Proxy)
  - [ ] 공개 경로(`/auth/*`) 설정
- [ ] **4. Auth Service (Python)**
  - [ ] DB Schema/Query 작성 및 sqlc 설정
  - [ ] JWT 발급 및 검증 로직 구현
  - [ ] User 회원가입/로그인 API
- [ ] **5. Gateway 고도화** - JWT 검증 추가
  - [ ] JWT 검증 Middleware 구현
  - [ ] 인증 필요 경로에 Middleware 적용
- [ ] **6. Product Service (Python)**
  - [ ] DB Schema/Query 작성 및 sqlc 설정
  - [ ] 상품 등록 및 재고 조회 API
  - [ ] 핫딜 API
- [ ] **7. Order Service (Python)**
  - [ ] DB Schema/Query 작성 및 sqlc 설정
  - [ ] 주문 생성 API (단순 DB Transaction)
- [ ] **8. 통합 Makefile 완성**
  - [ ] 빌드, 실행, 테스트 자동화 명령어

### 📊 Phase 2: 측정 환경 구축 (The Baseline)

**목표:** 최적화 전, 현재 시스템의 성능 지표를 수집하여 비교군(Control Group)을 형성한다.

- [ ] **E2E Testing (Functional)**
  - [ ] **Tool:** `Pytest` + `Tavern`
  - [ ] API Spec(Swagger) 기반의 통합 테스트 시나리오 작성 (Code-based)
  - [ ] CI 파이프라인 연동 준비 (Github Actions 등)
- [ ] **Observability (Monitoring)**
  - [ ] **Tool:** **SigNoz** (OpenTelemetry Native APM)
  - [ ] Python 앱에 OTel Auto-Instrumentation 적용 및 SigNoz 연동
  - [ ] 통합 대시보드 구성 (Traces, Metrics, Logs 일원화)
- [ ] **Load Testing (Performance)**
  - [ ] **Tool:** `k6`
  - [ ] 시나리오 스크립트 작성 (로그인 -> 목록조회 -> 주문)
  - [ ] Baseline 성능 측정 및 리포트 작성 (RPS, p99 Latency)

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
