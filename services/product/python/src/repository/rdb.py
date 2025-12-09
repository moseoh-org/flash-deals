from uuid import UUID

from src.database import get_connection
from src.generated.query import AsyncQuerier
from src.repository.base import ProductRepository
from src.schemas import ProductResponse


class RdbProductRepository(ProductRepository):
    """DB 직접 조회 Repository"""

    async def list_products(
        self, page: int, size: int, category: str | None
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
                    items.append(self._to_response(product))
            else:
                async for product in querier.list_products(limit=size, offset=offset):
                    items.append(self._to_response(product))

            return items, total

    async def get_product(self, product_id: UUID) -> ProductResponse | None:
        async with get_connection() as conn:
            querier = AsyncQuerier(conn)
            product = await querier.get_product_by_id(id=product_id)

            if product is None:
                return None

            return self._to_response(product)

    def _to_response(self, product) -> ProductResponse:
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
