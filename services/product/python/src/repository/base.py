from abc import ABC, abstractmethod
from uuid import UUID

from src.schemas import ProductResponse


class ProductRepository(ABC):
    """상품 저장소 인터페이스"""

    @abstractmethod
    async def list_products(
        self, page: int, size: int, category: str | None
    ) -> tuple[list[ProductResponse], int]:
        """상품 목록 조회"""
        pass

    @abstractmethod
    async def get_product(self, product_id: UUID) -> ProductResponse | None:
        """상품 단건 조회"""
        pass
