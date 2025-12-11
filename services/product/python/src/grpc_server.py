import asyncio
import logging
from concurrent import futures
from uuid import UUID

import grpc
from grpc_reflection.v1alpha import reflection

from src.config import settings
from src.service import ProductServiceError, get_deal, get_product, update_stock

# gRPC generated code (생성 후 사용)
from src.grpc_gen import product_pb2, product_pb2_grpc

logger = logging.getLogger(__name__)


class ProductServicer(product_pb2_grpc.ProductServiceServicer):
    """Product gRPC Service 구현"""

    async def GetProduct(self, request, context):
        """상품 정보 조회"""
        try:
            product_id = UUID(request.product_id)
            product = await get_product(product_id)

            return product_pb2.Product(
                id=str(product.id),
                name=product.name,
                description=product.description or "",
                price=product.price,
                stock=product.stock,
                created_at=product.created_at.isoformat() if product.created_at else "",
                updated_at=product.updated_at.isoformat() if product.updated_at else "",
            )
        except ProductServiceError as e:
            if e.status_code == 404:
                context.set_code(grpc.StatusCode.NOT_FOUND)
            else:
                context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(e.message)
            return product_pb2.Product()
        except ValueError:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Invalid product_id format")
            return product_pb2.Product()

    async def GetDeal(self, request, context):
        """핫딜 정보 조회"""
        try:
            deal_id = UUID(request.deal_id)
            deal = await get_deal(deal_id)

            product_msg = product_pb2.Product(
                id=str(deal.product.id),
                name=deal.product.name,
                description=deal.product.description or "",
                price=deal.product.price,
                stock=deal.product.stock,
                created_at=deal.product.created_at.isoformat() if deal.product.created_at else "",
                updated_at=deal.product.updated_at.isoformat() if deal.product.updated_at else "",
            )

            return product_pb2.Deal(
                id=str(deal.id),
                product_id=str(deal.product_id),
                deal_price=deal.deal_price,
                stock_limit=deal.deal_stock,
                start_time=deal.starts_at.isoformat() if deal.starts_at else "",
                end_time=deal.ends_at.isoformat() if deal.ends_at else "",
                status=deal.status.value,
                product=product_msg,
            )
        except ProductServiceError as e:
            if e.status_code == 404:
                context.set_code(grpc.StatusCode.NOT_FOUND)
            else:
                context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(e.message)
            return product_pb2.Deal()
        except ValueError:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Invalid deal_id format")
            return product_pb2.Deal()

    async def UpdateStock(self, request, context):
        """재고 변경"""
        try:
            product_id = UUID(request.product_id)
            stock_result = await update_stock(product_id, request.delta)

            # 재고 변경 후 상품 정보 반환
            product = await get_product(product_id)
            return product_pb2.Product(
                id=str(product.id),
                name=product.name,
                description=product.description or "",
                price=product.price,
                stock=product.stock,
                created_at=product.created_at.isoformat() if product.created_at else "",
                updated_at=product.updated_at.isoformat() if product.updated_at else "",
            )
        except ProductServiceError as e:
            if e.status_code == 404:
                context.set_code(grpc.StatusCode.NOT_FOUND)
            elif e.error == "INSUFFICIENT_STOCK":
                context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
            else:
                context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(e.message)
            return product_pb2.Product()
        except ValueError:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Invalid product_id format")
            return product_pb2.Product()


async def serve_grpc():
    """gRPC 서버 시작"""
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    product_pb2_grpc.add_ProductServiceServicer_to_server(ProductServicer(), server)

    # gRPC reflection 활성화 (디버깅용)
    service_names = (
        product_pb2.DESCRIPTOR.services_by_name["ProductService"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(service_names, server)

    listen_addr = f"[::]:{settings.grpc_port}"
    server.add_insecure_port(listen_addr)

    logger.info(f"gRPC server starting on port {settings.grpc_port}")
    await server.start()
    await server.wait_for_termination()


def run_grpc_server():
    """gRPC 서버 실행 (별도 프로세스/스레드용)"""
    asyncio.run(serve_grpc())
