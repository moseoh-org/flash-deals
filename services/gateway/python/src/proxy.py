from fastapi import Request, Response

from src.config import settings
from src.http_client import http_client

# 서비스 라우팅 맵
SERVICE_MAP = {
    "/auth": settings.auth_service_url,
    "/products": settings.product_service_url,
    "/orders": settings.order_service_url,
}


def get_target_url(path: str) -> str | None:
    """경로에 해당하는 서비스 URL 반환"""
    for prefix, service_url in SERVICE_MAP.items():
        if path.startswith(prefix):
            return service_url
    return None


async def proxy_request(request: Request, target_url: str, user_id: str | None = None) -> Response:
    """요청을 대상 서비스로 프록시"""
    # 원본 요청 정보 추출
    url = f"{target_url}{request.url.path}"
    if request.url.query:
        url = f"{url}?{request.url.query}"

    # 헤더 복사 (hop-by-hop 헤더 제외)
    headers = dict(request.headers)
    headers.pop("host", None)
    headers.pop("content-length", None)

    # 인증된 사용자 ID 추가
    if user_id:
        headers["X-User-ID"] = user_id

    # 요청 본문 읽기
    body = await request.body()

    # 프록시 요청 수행 (전역 클라이언트 사용)
    response = await http_client.request(
        method=request.method,
        url=url,
        headers=headers,
        content=body,
    )

    # 응답 헤더 복사 (hop-by-hop 헤더 제외)
    response_headers = dict(response.headers)
    response_headers.pop("content-length", None)
    response_headers.pop("content-encoding", None)
    response_headers.pop("transfer-encoding", None)

    return Response(
        content=response.content,
        status_code=response.status_code,
        headers=response_headers,
    )
