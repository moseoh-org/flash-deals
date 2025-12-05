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


# Order status enum
class OrderStatus(str, Enum):
    PENDING = "pending"
    CONFIRMED = "confirmed"
    SHIPPED = "shipped"
    DELIVERED = "delivered"
    CANCELLED = "cancelled"


# Shipping address
class ShippingAddress(BaseModel):
    recipient_name: str = Field(max_length=50)
    phone: str = Field(pattern=r"^01[0-9]-?[0-9]{3,4}-?[0-9]{4}$")
    address: str = Field(max_length=200)
    address_detail: str | None = Field(default=None, max_length=100)
    postal_code: str = Field(pattern=r"^[0-9]{5}$")


# Order item
class OrderItemRequest(BaseModel):
    product_id: UUID
    deal_id: UUID | None = None
    quantity: int = Field(ge=1, le=10)


class OrderItemResponse(BaseModel):
    id: UUID
    product_id: UUID
    deal_id: UUID | None = None
    product_name: str
    quantity: int
    unit_price: int
    subtotal: int


# Create order
class CreateOrderRequest(BaseModel):
    items: list[OrderItemRequest] = Field(min_length=1, max_length=10)
    shipping_address: ShippingAddress | None = None


# Cancel order
class CancelOrderRequest(BaseModel):
    reason: str | None = Field(default=None, max_length=500)


# Order response
class OrderResponse(BaseModel):
    id: UUID
    user_id: UUID
    items: list[OrderItemResponse]
    total_amount: int
    status: OrderStatus
    shipping_address: ShippingAddress | None = None
    cancelled_at: datetime | None = None
    cancel_reason: str | None = None
    created_at: datetime
    updated_at: datetime


class OrderListResponse(BaseModel):
    items: list[OrderResponse]
    total: int
    page: int
    size: int
