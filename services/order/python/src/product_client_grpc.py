import logging
from datetime import datetime
from uuid import UUID

import grpc

from src.config import settings
from src.grpc_gen import product_pb2, product_pb2_grpc
from src.product_client import ProductClientError

logger = logging.getLogger(__name__)

# gRPC 채널 재사용을 위한 전역 변수
_channel: grpc.aio.Channel | None = None
_stub: product_pb2_grpc.ProductServiceStub | None = None


async def get_stub() -> product_pb2_grpc.ProductServiceStub:
    """gRPC stub을 반환 (싱글톤 패턴으로 채널 재사용)"""
    global _channel, _stub
    if _stub is None:
        target = f"{settings.product_grpc_host}:{settings.product_grpc_port}"
        _channel = grpc.aio.insecure_channel(target)
        _stub = product_pb2_grpc.ProductServiceStub(_channel)
        logger.info(f"gRPC channel created: {target}")
    return _stub


def _parse_datetime(iso_string: str) -> datetime | None:
    """ISO 형식의 문자열을 datetime으로 변환"""
    if not iso_string:
        return None
    return datetime.fromisoformat(iso_string.replace("Z", "+00:00"))


async def get_product(product_id: UUID) -> dict:
    """Product Service에서 상품 정보 조회 (gRPC)"""
    try:
        stub = await get_stub()
        request = product_pb2.GetProductRequest(product_id=str(product_id))
        response = await stub.GetProduct(request, timeout=10.0)

        if not response.id:
            raise ProductClientError("PRODUCT_NOT_FOUND", f"상품을 찾을 수 없습니다: {product_id}", 404)

        return {
            "id": response.id,
            "name": response.name,
            "description": response.description,
            "price": response.price,
            "stock": response.stock,
            "created_at": response.created_at,
            "updated_at": response.updated_at,
        }
    except grpc.aio.AioRpcError as e:
        if e.code() == grpc.StatusCode.NOT_FOUND:
            raise ProductClientError("PRODUCT_NOT_FOUND", f"상품을 찾을 수 없습니다: {product_id}", 404)
        elif e.code() == grpc.StatusCode.INVALID_ARGUMENT:
            raise ProductClientError("INVALID_PRODUCT_ID", str(e.details()), 400)
        else:
            logger.error(f"gRPC error in get_product: {e.code()} - {e.details()}")
            raise ProductClientError("PRODUCT_SERVICE_ERROR", f"상품 서비스 오류: {e.details()}", 502)


async def get_deal(deal_id: UUID) -> dict:
    """Product Service에서 핫딜 정보 조회 (gRPC)"""
    try:
        stub = await get_stub()
        request = product_pb2.GetDealRequest(deal_id=str(deal_id))
        response = await stub.GetDeal(request, timeout=10.0)

        if not response.id:
            raise ProductClientError("DEAL_NOT_FOUND", f"핫딜을 찾을 수 없습니다: {deal_id}", 404)

        return {
            "id": response.id,
            "product_id": response.product_id,
            "deal_price": response.deal_price,
            "deal_stock": response.stock_limit,
            "starts_at": response.start_time,
            "ends_at": response.end_time,
            "status": response.status,
            "product": {
                "id": response.product.id,
                "name": response.product.name,
                "description": response.product.description,
                "price": response.product.price,
                "stock": response.product.stock,
                "created_at": response.product.created_at,
                "updated_at": response.product.updated_at,
            } if response.product.id else None,
        }
    except grpc.aio.AioRpcError as e:
        if e.code() == grpc.StatusCode.NOT_FOUND:
            raise ProductClientError("DEAL_NOT_FOUND", f"핫딜을 찾을 수 없습니다: {deal_id}", 404)
        elif e.code() == grpc.StatusCode.INVALID_ARGUMENT:
            raise ProductClientError("INVALID_DEAL_ID", str(e.details()), 400)
        else:
            logger.error(f"gRPC error in get_deal: {e.code()} - {e.details()}")
            raise ProductClientError("PRODUCT_SERVICE_ERROR", f"상품 서비스 오류: {e.details()}", 502)


async def decrease_stock(product_id: UUID, quantity: int) -> dict:
    """Product Service에서 재고 감소 (gRPC)"""
    try:
        stub = await get_stub()
        request = product_pb2.UpdateStockRequest(product_id=str(product_id), delta=-quantity)
        response = await stub.UpdateStock(request, timeout=10.0)

        if not response.id:
            raise ProductClientError("PRODUCT_NOT_FOUND", f"상품을 찾을 수 없습니다: {product_id}", 404)

        return {
            "product_id": response.id,
            "stock": response.stock,
        }
    except grpc.aio.AioRpcError as e:
        if e.code() == grpc.StatusCode.NOT_FOUND:
            raise ProductClientError("PRODUCT_NOT_FOUND", f"상품을 찾을 수 없습니다: {product_id}", 404)
        elif e.code() == grpc.StatusCode.FAILED_PRECONDITION:
            raise ProductClientError("INSUFFICIENT_STOCK", "재고가 부족합니다.", 400)
        elif e.code() == grpc.StatusCode.INVALID_ARGUMENT:
            raise ProductClientError("INVALID_PRODUCT_ID", str(e.details()), 400)
        else:
            logger.error(f"gRPC error in decrease_stock: {e.code()} - {e.details()}")
            raise ProductClientError("PRODUCT_SERVICE_ERROR", f"재고 감소 실패: {e.details()}", 502)


async def increase_stock(product_id: UUID, quantity: int) -> dict:
    """Product Service에서 재고 증가 (gRPC) - 취소 시 사용"""
    try:
        stub = await get_stub()
        request = product_pb2.UpdateStockRequest(product_id=str(product_id), delta=quantity)
        response = await stub.UpdateStock(request, timeout=10.0)

        if not response.id:
            raise ProductClientError("PRODUCT_NOT_FOUND", f"상품을 찾을 수 없습니다: {product_id}", 404)

        return {
            "product_id": response.id,
            "stock": response.stock,
        }
    except grpc.aio.AioRpcError as e:
        if e.code() == grpc.StatusCode.NOT_FOUND:
            raise ProductClientError("PRODUCT_NOT_FOUND", f"상품을 찾을 수 없습니다: {product_id}", 404)
        elif e.code() == grpc.StatusCode.INVALID_ARGUMENT:
            raise ProductClientError("INVALID_PRODUCT_ID", str(e.details()), 400)
        else:
            logger.error(f"gRPC error in increase_stock: {e.code()} - {e.details()}")
            raise ProductClientError("PRODUCT_SERVICE_ERROR", f"재고 복구 실패: {e.details()}", 502)
