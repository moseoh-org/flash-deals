from uuid import UUID

from fastapi import FastAPI, Header, Query
from fastapi.responses import JSONResponse

from src.product_client import ProductClientError
from src.schemas import (
    CancelOrderRequest,
    CreateOrderRequest,
    ErrorResponse,
    HealthResponse,
    OrderListResponse,
    OrderResponse,
)
from src.service import (
    OrderServiceError,
    cancel_order,
    create_order,
    get_order,
    list_orders,
)
from src.telemetry import setup_telemetry

app = FastAPI(
    title="Order Service",
    description="주문 생성 및 관리 서비스",
    version="1.0.0",
)

setup_telemetry(app)


@app.exception_handler(OrderServiceError)
async def order_service_error_handler(request, exc: OrderServiceError):
    return JSONResponse(
        status_code=exc.status_code,
        content=ErrorResponse(error=exc.error, message=exc.message).model_dump(),
    )


@app.exception_handler(ProductClientError)
async def product_client_error_handler(request, exc: ProductClientError):
    return JSONResponse(
        status_code=exc.status_code,
        content=ErrorResponse(error=exc.error, message=exc.message).model_dump(),
    )


# Health check
@app.get("/health", response_model=HealthResponse)
@app.get("/orders/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(status="healthy", service="order-service")


# Order endpoints
@app.get("/orders", response_model=OrderListResponse)
async def list_orders_endpoint(
    x_user_id: str = Header(..., alias="X-User-ID"),
    page: int = Query(default=1, ge=1),
    size: int = Query(default=20, ge=1, le=100),
    status: str | None = Query(default=None),
):
    user_id = UUID(x_user_id)
    items, total = await list_orders(user_id=user_id, page=page, size=size, status=status)
    return OrderListResponse(items=items, total=total, page=page, size=size)


@app.post("/orders", response_model=OrderResponse, status_code=201)
async def create_order_endpoint(
    request: CreateOrderRequest,
    x_user_id: str = Header(..., alias="X-User-ID"),
):
    user_id = UUID(x_user_id)
    return await create_order(
        user_id=user_id,
        items=request.items,
        shipping_address=request.shipping_address,
    )


@app.get("/orders/{order_id}", response_model=OrderResponse)
async def get_order_endpoint(
    order_id: UUID,
    x_user_id: str = Header(..., alias="X-User-ID"),
):
    user_id = UUID(x_user_id)
    return await get_order(order_id=order_id, user_id=user_id)


@app.post("/orders/{order_id}/cancel", response_model=OrderResponse)
async def cancel_order_endpoint(
    order_id: UUID,
    request: CancelOrderRequest | None = None,
    x_user_id: str = Header(..., alias="X-User-ID"),
):
    user_id = UUID(x_user_id)
    reason = request.reason if request else None
    return await cancel_order(order_id=order_id, user_id=user_id, reason=reason)
