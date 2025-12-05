from uuid import UUID

import httpx

from src.config import settings


class ProductClientError(Exception):
    def __init__(self, error: str, message: str, status_code: int = 400):
        self.error = error
        self.message = message
        self.status_code = status_code
        super().__init__(message)


async def get_product(product_id: UUID) -> dict:
    """Product Service에서 상품 정보 조회"""
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
