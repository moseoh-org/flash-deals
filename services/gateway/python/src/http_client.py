import httpx

# 전역 HTTP 클라이언트 (커넥션 풀 재사용)
http_client = httpx.AsyncClient(
    limits=httpx.Limits(
        max_connections=100,
        max_keepalive_connections=20,
    ),
    timeout=httpx.Timeout(30.0),
)


async def close_http_client():
    """애플리케이션 종료 시 클라이언트 정리"""
    await http_client.aclose()
