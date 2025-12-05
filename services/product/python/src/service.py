from datetime import datetime, timezone
from uuid import UUID

from src.database import get_connection
from src.generated.query import (
    AsyncQuerier,
    CreateDealParams,
    CreateProductParams,
    GetDealByIDRow,
    ListActiveDealsRow,
    UpdateProductParams,
)
from src.schemas import (
    DealResponse,
    DealStatus,
    ProductResponse,
    StockResponse,
)


class ProductServiceError(Exception):
    def __init__(self, error: str, message: str, status_code: int = 400):
        self.error = error
        self.message = message
        self.status_code = status_code
        super().__init__(message)


def _get_deal_status(starts_at: datetime, ends_at: datetime, remaining_stock: int) -> DealStatus:
    now = datetime.now(timezone.utc)
    if remaining_stock <= 0:
        return DealStatus.SOLD_OUT
    if now < starts_at:
        return DealStatus.SCHEDULED
    if now > ends_at:
        return DealStatus.ENDED
    return DealStatus.ACTIVE


def _deal_row_to_response(row: GetDealByIDRow | ListActiveDealsRow) -> DealResponse:
    product = ProductResponse(
        id=row.p_id,
        name=row.p_name,
        description=row.p_description,
        price=row.p_price,
        stock=row.p_stock,
        category=row.p_category,
        image_url=row.p_image_url,
        created_at=row.p_created_at,
        updated_at=row.p_updated_at,
    )
    return DealResponse(
        id=row.id,
        product_id=row.product_id,
        product=product,
        deal_price=row.deal_price,
        original_price=product.price,
        deal_stock=row.deal_stock,
        remaining_stock=row.remaining_stock,
        starts_at=row.starts_at,
        ends_at=row.ends_at,
        status=_get_deal_status(row.starts_at, row.ends_at, row.remaining_stock),
        created_at=row.created_at,
    )


# Product operations
async def create_product(
    name: str,
    price: int,
    stock: int,
    description: str | None = None,
    category: str | None = None,
    image_url: str | None = None,
) -> ProductResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)
        product = await querier.create_product(
            CreateProductParams(
                name=name,
                description=description,
                price=price,
                stock=stock,
                category=category,
                image_url=image_url,
            )
        )
        await conn.commit()

        if product is None:
            raise ProductServiceError("CREATE_FAILED", "상품 생성에 실패했습니다.", 500)

        return ProductResponse(
            id=product.id,
            name=product.name,
            description=product.description,
            price=product.price,
            stock=product.stock,
            category=product.category,
            image_url=product.image_url,
            created_at=product.created_at,
            updated_at=product.updated_at,
        )


async def get_product(product_id: UUID) -> ProductResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)
        product = await querier.get_product_by_id(id=product_id)

        if product is None:
            raise ProductServiceError("NOT_FOUND", "상품을 찾을 수 없습니다.", 404)

        return ProductResponse(
            id=product.id,
            name=product.name,
            description=product.description,
            price=product.price,
            stock=product.stock,
            category=product.category,
            image_url=product.image_url,
            created_at=product.created_at,
            updated_at=product.updated_at,
        )


async def list_products(
    page: int = 1, size: int = 20, category: str | None = None
) -> tuple[list[ProductResponse], int]:
    offset = (page - 1) * size
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        # Count query
        if category:
            total = await querier.count_products_by_category(category=category)
        else:
            total = await querier.count_products()

        total = total or 0

        # List query
        items = []
        if category:
            async for product in querier.list_products_by_category(
                category=category, limit=size, offset=offset
            ):
                items.append(
                    ProductResponse(
                        id=product.id,
                        name=product.name,
                        description=product.description,
                        price=product.price,
                        stock=product.stock,
                        category=product.category,
                        image_url=product.image_url,
                        created_at=product.created_at,
                        updated_at=product.updated_at,
                    )
                )
        else:
            async for product in querier.list_products(limit=size, offset=offset):
                items.append(
                    ProductResponse(
                        id=product.id,
                        name=product.name,
                        description=product.description,
                        price=product.price,
                        stock=product.stock,
                        category=product.category,
                        image_url=product.image_url,
                        created_at=product.created_at,
                        updated_at=product.updated_at,
                    )
                )

        return items, total


async def update_product(
    product_id: UUID,
    name: str | None = None,
    description: str | None = None,
    price: int | None = None,
    category: str | None = None,
    image_url: str | None = None,
) -> ProductResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        # Check if product exists first
        existing = await querier.get_product_by_id(id=product_id)
        if existing is None:
            raise ProductServiceError("NOT_FOUND", "상품을 찾을 수 없습니다.", 404)

        product = await querier.update_product(
            UpdateProductParams(
                id=product_id,
                name=name if name is not None else existing.name,
                description=description if description is not None else existing.description,
                price=price if price is not None else existing.price,
                category=category if category is not None else existing.category,
                image_url=image_url if image_url is not None else existing.image_url,
            )
        )
        await conn.commit()

        if product is None:
            raise ProductServiceError("UPDATE_FAILED", "상품 수정에 실패했습니다.", 500)

        return ProductResponse(
            id=product.id,
            name=product.name,
            description=product.description,
            price=product.price,
            stock=product.stock,
            category=product.category,
            image_url=product.image_url,
            created_at=product.created_at,
            updated_at=product.updated_at,
        )


# Stock operations
async def get_stock(product_id: UUID) -> StockResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)
        product = await querier.get_product_by_id(id=product_id)

        if product is None:
            raise ProductServiceError("NOT_FOUND", "상품을 찾을 수 없습니다.", 404)

        return StockResponse(
            product_id=product.id,
            stock=product.stock,
            updated_at=product.updated_at,
        )


async def update_stock(product_id: UUID, delta: int) -> StockResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        # Check current stock with lock
        stock_row = await querier.get_stock_for_update(id=product_id)
        if stock_row is None:
            raise ProductServiceError("NOT_FOUND", "상품을 찾을 수 없습니다.", 404)

        new_stock = stock_row.stock + delta
        if new_stock < 0:
            raise ProductServiceError("INSUFFICIENT_STOCK", "재고가 부족합니다.", 400)

        result = await querier.update_stock(id=product_id, stock=new_stock)
        await conn.commit()

        if result is None:
            raise ProductServiceError("UPDATE_FAILED", "재고 수정에 실패했습니다.", 500)

        return StockResponse(
            product_id=result.id,
            stock=result.stock,
            updated_at=result.updated_at,
        )


# Deal operations
async def create_deal(
    product_id: UUID,
    deal_price: int,
    deal_stock: int,
    starts_at: datetime,
    ends_at: datetime,
) -> DealResponse:
    # Validate product exists and get original price
    product = await get_product(product_id)

    if starts_at >= ends_at:
        raise ProductServiceError("INVALID_PERIOD", "종료 시간은 시작 시간보다 커야 합니다.", 400)

    async with get_connection() as conn:
        querier = AsyncQuerier(conn)
        deal = await querier.create_deal(
            CreateDealParams(
                product_id=product_id,
                deal_price=deal_price,
                deal_stock=deal_stock,
                remaining_stock=deal_stock,
                starts_at=starts_at,
                ends_at=ends_at,
            )
        )
        await conn.commit()

        if deal is None:
            raise ProductServiceError("CREATE_FAILED", "핫딜 생성에 실패했습니다.", 500)

        return DealResponse(
            id=deal.id,
            product_id=deal.product_id,
            product=product,
            deal_price=deal.deal_price,
            original_price=product.price,
            deal_stock=deal.deal_stock,
            remaining_stock=deal.remaining_stock,
            starts_at=deal.starts_at,
            ends_at=deal.ends_at,
            status=_get_deal_status(deal.starts_at, deal.ends_at, deal.remaining_stock),
            created_at=deal.created_at,
        )


async def get_deal(deal_id: UUID) -> DealResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)
        row = await querier.get_deal_by_id(id=deal_id)

        if row is None:
            raise ProductServiceError("NOT_FOUND", "핫딜을 찾을 수 없습니다.", 404)

        return _deal_row_to_response(row)


async def list_active_deals(page: int = 1, size: int = 20) -> tuple[list[DealResponse], int]:
    offset = (page - 1) * size
    now = datetime.now(timezone.utc)

    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        # Count active deals
        total = await querier.count_active_deals(starts_at=now)
        total = total or 0

        # List active deals
        items = []
        async for row in querier.list_active_deals(starts_at=now, limit=size, offset=offset):
            items.append(_deal_row_to_response(row))

        return items, total
