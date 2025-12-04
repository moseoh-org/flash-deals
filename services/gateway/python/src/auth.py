import httpx
from fastapi import Request
from starlette.routing import Match, Route, Router

from src.config import settings


class PublicRouteValidator:
    """Starlette Router 기반 공개 경로 검증"""

    def __init__(self):
        self.router = Router(
            routes=[
                # Health
                Route("/health", endpoint=lambda: None, methods=["GET"]),
                Route("/auth/health", endpoint=lambda: None, methods=["GET"]),
                Route("/products/health", endpoint=lambda: None, methods=["GET"]),
                Route("/orders/health", endpoint=lambda: None, methods=["GET"]),
                # Auth (public)
                Route("/auth/register", endpoint=lambda: None, methods=["POST"]),
                Route("/auth/login", endpoint=lambda: None, methods=["POST"]),
                Route("/auth/refresh", endpoint=lambda: None, methods=["POST"]),
                # Products (public read)
                Route("/products", endpoint=lambda: None, methods=["GET"]),
                Route("/products/{product_id}", endpoint=lambda: None, methods=["GET"]),
                Route("/products/{product_id}/stock", endpoint=lambda: None, methods=["GET"]),
                Route("/products/deals", endpoint=lambda: None, methods=["GET"]),
                Route("/products/deals/{deal_id}", endpoint=lambda: None, methods=["GET"]),
            ]
        )

    def is_public(self, path: str, method: str) -> bool:
        """공개 경로인지 확인"""
        scope = {"type": "http", "path": path, "method": method}
        for route in self.router.routes:
            match, _ = route.matches(scope)
            if match == Match.FULL:
                return True
        return False


public_route_validator = PublicRouteValidator()


def extract_token(request: Request) -> str | None:
    """Authorization 헤더에서 Bearer 토큰 추출"""
    auth_header = request.headers.get("authorization")
    if not auth_header:
        return None
    if not auth_header.startswith("Bearer "):
        return None
    return auth_header[7:]


async def verify_token(token: str) -> dict | None:
    """Auth Service를 통해 토큰 검증"""
    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(
                f"{settings.auth_service_url}/auth/verify",
                headers={"Authorization": f"Bearer {token}"},
                timeout=5.0,
            )
            if response.status_code == 200:
                return response.json()
            return None
        except Exception:
            return None
