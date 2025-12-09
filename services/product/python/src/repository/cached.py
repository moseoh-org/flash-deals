import json
from uuid import UUID

from redis.asyncio import Redis

from src.repository.base import ProductRepository
from src.schemas import ProductResponse


class CachedProductRepository(ProductRepository):
    """Redis 캐싱 Repository (Decorator 패턴)"""

    def __init__(self, inner: ProductRepository, redis: Redis, ttl: int = 60):
        self.inner = inner
        self.redis = redis
        self.ttl = ttl

    async def list_products(
        self, page: int, size: int, category: str | None
    ) -> tuple[list[ProductResponse], int]:
        cache_key = f"products:list:{page}:{size}:{category or 'all'}"

        # 캐시 조회
        cached = await self.redis.get(cache_key)
        if cached:
            data = json.loads(cached)
            items = [ProductResponse(**item) for item in data["items"]]
            return items, data["total"]

        # DB 조회 (inner repository)
        items, total = await self.inner.list_products(page, size, category)

        # 캐시 저장
        cache_data = {
            "items": [item.model_dump(mode="json") for item in items],
            "total": total,
        }
        await self.redis.set(cache_key, json.dumps(cache_data), ex=self.ttl)

        return items, total

    async def get_product(self, product_id: UUID) -> ProductResponse | None:
        cache_key = f"products:detail:{product_id}"

        # 캐시 조회
        cached = await self.redis.get(cache_key)
        if cached:
            return ProductResponse(**json.loads(cached))

        # DB 조회
        product = await self.inner.get_product(product_id)
        if product is None:
            return None

        # 캐시 저장
        await self.redis.set(
            cache_key, json.dumps(product.model_dump(mode="json")), ex=self.ttl
        )

        return product
