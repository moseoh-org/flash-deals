from uuid import UUID

from fastapi import FastAPI, Query
from fastapi.responses import JSONResponse

from src.schemas import (
    CreateDealRequest,
    CreateProductRequest,
    DealListResponse,
    DealResponse,
    ErrorResponse,
    HealthResponse,
    ProductListResponse,
    ProductResponse,
    StockResponse,
    UpdateProductRequest,
    UpdateStockRequest,
)
from src.service import (
    ProductServiceError,
    create_deal,
    create_product,
    get_deal,
    get_product,
    get_stock,
    list_active_deals,
    list_products,
    update_product,
    update_stock,
)
from src.telemetry import setup_telemetry

app = FastAPI(
    title="Product Service",
    description="상품 및 재고 관리 서비스",
    version="1.0.0",
)

setup_telemetry(app)


@app.exception_handler(ProductServiceError)
async def product_service_error_handler(request, exc: ProductServiceError):
    return JSONResponse(
        status_code=exc.status_code,
        content=ErrorResponse(error=exc.error, message=exc.message).model_dump(),
    )


# Health check
@app.get("/health", response_model=HealthResponse)
@app.get("/products/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(status="healthy", service="product-service")


# Product endpoints
@app.get("/products", response_model=ProductListResponse)
async def list_products_endpoint(
    page: int = Query(default=1, ge=1),
    size: int = Query(default=20, ge=1, le=100),
    category: str | None = None,
):
    items, total = await list_products(page=page, size=size, category=category)
    return ProductListResponse(items=items, total=total, page=page, size=size)


@app.post("/products", response_model=ProductResponse, status_code=201)
async def create_product_endpoint(request: CreateProductRequest):
    return await create_product(
        name=request.name,
        price=request.price,
        stock=request.stock,
        description=request.description,
        category=request.category,
        image_url=request.image_url,
    )


# Deal endpoints (must be before /products/{product_id} to avoid route conflict)
@app.get("/products/deals", response_model=DealListResponse)
async def list_deals_endpoint(
    page: int = Query(default=1, ge=1),
    size: int = Query(default=20, ge=1, le=100),
):
    items, total = await list_active_deals(page=page, size=size)
    return DealListResponse(items=items, total=total, page=page, size=size)


@app.post("/products/deals", response_model=DealResponse, status_code=201)
async def create_deal_endpoint(request: CreateDealRequest):
    return await create_deal(
        product_id=request.product_id,
        deal_price=request.deal_price,
        deal_stock=request.deal_stock,
        starts_at=request.starts_at,
        ends_at=request.ends_at,
    )


@app.get("/products/deals/{deal_id}", response_model=DealResponse)
async def get_deal_endpoint(deal_id: UUID):
    return await get_deal(deal_id)


@app.get("/products/{product_id}", response_model=ProductResponse)
async def get_product_endpoint(product_id: UUID):
    return await get_product(product_id)


@app.patch("/products/{product_id}", response_model=ProductResponse)
async def update_product_endpoint(product_id: UUID, request: UpdateProductRequest):
    return await update_product(
        product_id=product_id,
        name=request.name,
        description=request.description,
        price=request.price,
        category=request.category,
        image_url=request.image_url,
    )


# Stock endpoints
@app.get("/products/{product_id}/stock", response_model=StockResponse)
async def get_stock_endpoint(product_id: UUID):
    return await get_stock(product_id)


@app.patch("/products/{product_id}/stock", response_model=StockResponse)
async def update_stock_endpoint(product_id: UUID, request: UpdateStockRequest):
    return await update_stock(product_id=product_id, delta=request.delta)
