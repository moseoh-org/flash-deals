import logging
from uuid import UUID

import httpx

from src.config import settings

logger = logging.getLogger(__name__)


class ProductClientError(Exception):
    def __init__(self, error: str, message: str, status_code: int = 400):
        self.error = error
        self.message = message
        self.status_code = status_code
        super().__init__(message)


# HTTP 커넥션 풀 (http_pool 모드용)
_http_client: httpx.AsyncClient | None = None


async def get_http_client() -> httpx.AsyncClient:
    """HTTP 클라이언트 반환 (커넥션 풀 재사용)"""
    global _http_client
    if _http_client is None:
        _http_client = httpx.AsyncClient(
            base_url=settings.product_service_url,
            timeout=10.0,
            limits=httpx.Limits(max_connections=100, max_keepalive_connections=20),
        )
        logger.info("HTTP connection pool created")
    return _http_client


async def get_product(product_id: UUID) -> dict:
    """Product Service에서 상품 정보 조회"""
    if settings.product_client_type == "grpc":
        from src.product_client_grpc import get_product as grpc_get_product
        return await grpc_get_product(product_id)

    if settings.product_client_type == "http_pool":
        client = await get_http_client()
        response = await client.get(f"/products/{product_id}")
    else:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.product_service_url}/products/{product_id}",
                timeout=10.0,
            )

    if response.status_code == 404:
        raise ProductClientError("PRODUCT_NOT_FOUND", f"상품을 찾을 수 없습니다: {product_id}", 404)

    if response.status_code != 200:
        raise ProductClientError(
            "PRODUCT_SERVICE_ERROR",
            f"상품 서비스 오류: {response.status_code}",
            502,
        )

    return response.json()


async def get_deal(deal_id: UUID) -> dict:
    """Product Service에서 핫딜 정보 조회"""
    if settings.product_client_type == "grpc":
        from src.product_client_grpc import get_deal as grpc_get_deal
        return await grpc_get_deal(deal_id)

    if settings.product_client_type == "http_pool":
        client = await get_http_client()
        response = await client.get(f"/products/deals/{deal_id}")
    else:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.product_service_url}/products/deals/{deal_id}",
                timeout=10.0,
            )

    if response.status_code == 404:
        raise ProductClientError("DEAL_NOT_FOUND", f"핫딜을 찾을 수 없습니다: {deal_id}", 404)

    if response.status_code != 200:
        raise ProductClientError(
            "PRODUCT_SERVICE_ERROR",
            f"상품 서비스 오류: {response.status_code}",
            502,
        )

    return response.json()


async def decrease_stock(product_id: UUID, quantity: int) -> dict:
    """Product Service에서 재고 감소"""
    if settings.product_client_type == "grpc":
        from src.product_client_grpc import decrease_stock as grpc_decrease_stock
        return await grpc_decrease_stock(product_id, quantity)

    if settings.product_client_type == "http_pool":
        client = await get_http_client()
        response = await client.patch(
            f"/products/{product_id}/stock",
            json={"delta": -quantity},
        )
    else:
        async with httpx.AsyncClient() as client:
            response = await client.patch(
                f"{settings.product_service_url}/products/{product_id}/stock",
                json={"delta": -quantity},
                timeout=10.0,
            )

    if response.status_code == 400:
        data = response.json()
        raise ProductClientError(
            data.get("error", "INSUFFICIENT_STOCK"),
            data.get("message", "재고가 부족합니다."),
            400,
        )

    if response.status_code != 200:
        raise ProductClientError(
            "PRODUCT_SERVICE_ERROR",
            f"재고 감소 실패: {response.status_code}",
            502,
        )

    return response.json()


async def increase_stock(product_id: UUID, quantity: int) -> dict:
    """Product Service에서 재고 증가 (취소 시)"""
    if settings.product_client_type == "grpc":
        from src.product_client_grpc import increase_stock as grpc_increase_stock
        return await grpc_increase_stock(product_id, quantity)

    if settings.product_client_type == "http_pool":
        client = await get_http_client()
        response = await client.patch(
            f"/products/{product_id}/stock",
            json={"delta": quantity},
        )
    else:
        async with httpx.AsyncClient() as client:
            response = await client.patch(
                f"{settings.product_service_url}/products/{product_id}/stock",
                json={"delta": quantity},
                timeout=10.0,
            )

    if response.status_code != 200:
        raise ProductClientError(
            "PRODUCT_SERVICE_ERROR",
            f"재고 복구 실패: {response.status_code}",
            502,
        )

    return response.json()
