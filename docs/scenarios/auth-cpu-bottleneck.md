# 인증 CPU 병목

## 상황

트래픽이 증가하면서 Auth 서비스의 CPU 사용률이 100%를 초과한다.
Protected route 접근 시 매번 Auth 서비스를 호출하여 JWT를 검증하므로, 트래픽에 비례해 Auth 부하가 증가한다.

## 테스트 환경

- 부하 테스트: `make load-auth-stress`
- 최대 VU: 100
- 테스트 대상: `GET /auth/users/me` (Protected route)

---

## 기존 시스템

### 성능

| 지표          | 값        |
| ------------- | --------- |
| p95 응답시간  | 235ms     |
| 평균 응답시간 | 79ms      |
| 처리량        | 900req/s  |
| 에러율        | 0%        |

### 리소스 사용량

| 컨테이너 | CPU  |
| -------- | ---- |
| auth     | 105% |
| gateway  | 16%  |

### 문제점

- Auth 서비스가 CPU 100%를 초과하여 병목 지점
- 모든 Protected route 요청이 Auth 서비스로 집중

---

## 개선 1: Kong JWT + Proxy Cache

### 변경 내용

- Kong JWT 플러그인: Gateway에서 JWT 검증
- Kong Proxy Cache 플러그인: 응답 캐싱 (Authorization 헤더별)
- Auth 서비스 호출 없이 캐시 응답

### 성능

| 지표          | Before   | After      | 개선      |
| ------------- | -------- | ---------- | --------- |
| p95 응답시간  | 235ms    | 20ms       | 11.8배 ↓  |
| 평균 응답시간 | 79ms     | 6ms        | 13배 ↓    |
| 처리량        | 900req/s | 11337req/s | 12.6배 ↑  |
| 에러율        | 0%       | 0%         | -         |

### 리소스 사용량 비교

| 컨테이너 | Before | After |
| -------- | ------ | ----- |
| auth     | 105%   | 0.4%  |
| gateway  | 16%    | 2%    |

### 분석

- Kong Proxy Cache로 동일 JWT 요청은 Auth 호출 없이 캐시 응답
- Auth 서비스 부하가 사실상 제거됨 (첫 요청만 처리)
- Gateway에서 JWT 검증 + 캐시 응답으로 응답시간 대폭 단축
