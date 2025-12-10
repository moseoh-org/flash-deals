# 인증 CPU 병목

## 상황

트래픽 증가하면서 Auth 서비스의 CPU 사용률이 높아진다.
Protected route 접근 시 매번 Auth 서비스를 호출하여 JWT를 검증하므로, 트래픽에 비례해 Auth 부하가 증가한다.

### 현재 아키텍처

```
Client → Kong Gateway → Auth Service (/auth/verify) → JWT 검증
                      ↘ Product/Order Service
```

- 모든 Protected route 요청마다 `/auth/verify` HTTP 호출
- Auth Service가 JWT 검증
- Auth Service CPU가 병목 지점

## 테스트 환경

- k6 부하 테스트: `make load-auth-stress`
- 최대 VU: 100
- 테스트 대상: Protected route (`/auth/users/me`)
- 측정: Auth Service CPU 사용률, 응답시간

---

## 기존 시스템 (Before)

### 성능

| 지표               | 값       |
| ------------------ | -------- |
| p95 응답시간       | 235ms    |
| 평균 응답시간      | 79ms     |
| 처리량 (req/s)     | 900      |
| Auth CPU 사용률    | **105%** |
| Gateway CPU 사용률 | 16%      |

### 분석

- Auth 서비스가 CPU 100%를 초과하여 **병목 지점**으로 확인됨
- 모든 Protected route 요청이 Auth 서비스로 집중되어 트래픽에 비례해 부하가 증가
- Gateway는 16%로 여유 있음 - Auth에서 인증 처리를 분담하면 전체 처리량 향상 가능

---

## 1차 개선: Kong JWT + Proxy Cache

### 개선 내용

1. **Kong JWT 플러그인**: Gateway에서 JWT 검증
2. **Kong Proxy Cache 플러그인**: 응답 캐싱 (Authorization 헤더별)

### 개선된 아키텍처

```
Client → Kong Gateway (JWT 검증 + 캐시) → Auth Service
              ↓
         캐시 히트 시 Auth 호출 없이 응답
```

### 성능

| 지표               | Before | After  | 개선        |
| ------------------ | ------ | ------ | ----------- |
| p95 응답시간       | 235ms  | 20ms   | **11.8배↓** |
| 평균 응답시간      | 79ms   | 6ms    | **13배↓**   |
| 처리량 (req/s)     | 900    | 11,337 | **12.6배↑** |
| Auth CPU 사용률    | 105%   | 0.4%   | **99.6%↓**  |
| Gateway CPU 사용률 | 16%    | 2%     | 87%↓        |

### 분석

- Kong Proxy Cache로 **동일 JWT 요청은 Auth 호출 없이 캐시 응답**
- Auth 서비스 부하가 사실상 제거됨 (첫 요청만 처리)
- Gateway에서 JWT 검증 + 캐시 응답으로 응답시간 대폭 단축
