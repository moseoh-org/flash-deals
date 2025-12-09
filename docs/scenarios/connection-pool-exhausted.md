# DB 커넥션 풀 고갈

## 상황

트래픽이 증가하면 "connection pool exhausted" 에러가 발생하며 요청이 실패한다.
서버 CPU와 메모리는 여유가 있는데, DB 연결을 기다리다가 타임아웃이 발생한다.
