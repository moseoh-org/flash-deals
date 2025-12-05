from datetime import datetime
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field


class HealthResponse(BaseModel):
    status: str
    service: str


class ErrorResponse(BaseModel):
    error: str
    message: str
    details: dict | None = None


# Product schemas
class CreateProductRequest(BaseModel):
    name: str = Field(min_length=1, max_length=200)
    description: str | None = Field(default=None, max_length=5000)
    price: int = Field(ge=0)
    stock: int = Field(ge=0)
    category: str | None = Field(default=None, max_length=50)
    image_url: str | None = None


class UpdateProductRequest(BaseModel):
    name: str | None = Field(default=None, min_length=1, max_length=200)
    description: str | None = Field(default=None, max_length=5000)
    price: int | None = Field(default=None, ge=0)
    category: str | None = Field(default=None, max_length=50)
    image_url: str | None = None


class ProductResponse(BaseModel):
    id: UUID
    name: str
    description: str | None
    price: int
    stock: int
    category: str | None
    image_url: str | None
    created_at: datetime
    updated_at: datetime


class ProductListResponse(BaseModel):
    items: list[ProductResponse]
    total: int
    page: int
    size: int


# Stock schemas
class UpdateStockRequest(BaseModel):
    delta: int


class StockResponse(BaseModel):
    product_id: UUID
    stock: int
    updated_at: datetime


# Deal schemas
class DealStatus(str, Enum):
    SCHEDULED = "scheduled"
    ACTIVE = "active"
    SOLD_OUT = "sold_out"
    ENDED = "ended"


class CreateDealRequest(BaseModel):
    product_id: UUID
    deal_price: int = Field(ge=0)
    deal_stock: int = Field(ge=1)
    starts_at: datetime
    ends_at: datetime


class DealResponse(BaseModel):
    id: UUID
    product_id: UUID
    product: ProductResponse | None = None
    deal_price: int
    original_price: int | None = None
    deal_stock: int
    remaining_stock: int
    starts_at: datetime
    ends_at: datetime
    status: DealStatus
    created_at: datetime


class DealListResponse(BaseModel):
    items: list[DealResponse]
    total: int
    page: int
    size: int
