from uuid import UUID

from src.database import get_connection
from src.generated.models import OrdersOrder, OrdersOrderItem
from src.generated.query import (
    AsyncQuerier,
    CreateOrderItemParams,
    CreateOrderParams,
    ListOrdersWithItemsByUserIDAndStatusRow,
    ListOrdersWithItemsByUserIDRow,
)
from src.product_client import ProductClientError, decrease_stock, get_deal, get_product, increase_stock
from src.schemas import (
    OrderItemRequest,
    OrderItemResponse,
    OrderResponse,
    OrderStatus,
    ShippingAddress,
)


class OrderServiceError(Exception):
    def __init__(self, error: str, message: str, status_code: int = 400):
        self.error = error
        self.message = message
        self.status_code = status_code
        super().__init__(message)


def _order_to_response(order: OrdersOrder, items: list[OrdersOrderItem]) -> OrderResponse:
    shipping_address = None
    if order.recipient_name:
        shipping_address = ShippingAddress(
            recipient_name=order.recipient_name,
            phone=order.phone or "",
            address=order.address or "",
            address_detail=order.address_detail,
            postal_code=order.postal_code or "",
        )

    return OrderResponse(
        id=order.id,
        user_id=order.user_id,
        items=[
            OrderItemResponse(
                id=item.id,
                product_id=item.product_id,
                deal_id=item.deal_id,
                product_name=item.product_name,
                quantity=item.quantity,
                unit_price=item.unit_price,
                subtotal=item.subtotal,
            )
            for item in items
        ],
        total_amount=order.total_amount,
        status=OrderStatus(order.status),
        shipping_address=shipping_address,
        cancelled_at=order.cancelled_at,
        cancel_reason=order.cancel_reason,
        created_at=order.created_at,
        updated_at=order.updated_at,
    )


async def create_order(
    user_id: UUID,
    items: list[OrderItemRequest],
    shipping_address: ShippingAddress | None = None,
) -> OrderResponse:
    # 1. 상품 정보 조회 및 가격 계산
    order_items_data: list[dict] = []
    total_amount = 0

    for item in items:
        try:
            if item.deal_id:
                # 핫딜 주문
                deal = await get_deal(item.deal_id)
                if deal["product_id"] != str(item.product_id):
                    raise OrderServiceError(
                        "INVALID_DEAL",
                        "핫딜과 상품이 일치하지 않습니다.",
                        400,
                    )
                if deal["status"] != "active":
                    raise OrderServiceError(
                        "DEAL_NOT_ACTIVE",
                        "핫딜이 진행 중이 아닙니다.",
                        400,
                    )
                unit_price = deal["deal_price"]
                product_name = deal["product"]["name"]
            else:
                # 일반 주문
                product = await get_product(item.product_id)
                unit_price = product["price"]
                product_name = product["name"]

            subtotal = unit_price * item.quantity
            total_amount += subtotal

            order_items_data.append({
                "product_id": item.product_id,
                "deal_id": item.deal_id,
                "product_name": product_name,
                "quantity": item.quantity,
                "unit_price": unit_price,
                "subtotal": subtotal,
            })
        except ProductClientError as e:
            raise OrderServiceError(e.error, e.message, e.status_code) from e

    # 2. 재고 차감 (트랜잭션 외부에서 - 보상 트랜잭션 패턴)
    decreased_items: list[tuple[UUID, int]] = []
    try:
        for item in items:
            await decrease_stock(item.product_id, item.quantity)
            decreased_items.append((item.product_id, item.quantity))
    except ProductClientError as e:
        # 이미 차감한 재고 복구
        for product_id, quantity in decreased_items:
            try:
                await increase_stock(product_id, quantity)
            except ProductClientError:
                pass  # 재고 복구 실패는 로깅만 (실제로는 알림 필요)
        raise OrderServiceError(e.error, e.message, e.status_code) from e

    # 3. 주문 생성
    try:
        async with get_connection() as conn:
            querier = AsyncQuerier(conn)

            # 주문 생성
            order = await querier.create_order(
                CreateOrderParams(
                    user_id=user_id,
                    total_amount=total_amount,
                    status="confirmed",
                    recipient_name=shipping_address.recipient_name if shipping_address else None,
                    phone=shipping_address.phone if shipping_address else None,
                    address=shipping_address.address if shipping_address else None,
                    address_detail=shipping_address.address_detail if shipping_address else None,
                    postal_code=shipping_address.postal_code if shipping_address else None,
                )
            )

            if order is None:
                raise OrderServiceError("CREATE_FAILED", "주문 생성에 실패했습니다.", 500)

            # 주문 아이템 생성
            created_items: list[OrdersOrderItem] = []
            for item_data in order_items_data:
                order_item = await querier.create_order_item(
                    CreateOrderItemParams(
                        order_id=order.id,
                        product_id=item_data["product_id"],
                        deal_id=item_data["deal_id"],
                        product_name=item_data["product_name"],
                        quantity=item_data["quantity"],
                        unit_price=item_data["unit_price"],
                        subtotal=item_data["subtotal"],
                    )
                )
                if order_item:
                    created_items.append(order_item)

            await conn.commit()
            return _order_to_response(order, created_items)

    except OrderServiceError:
        # 주문 생성 실패 시 재고 복구
        for product_id, quantity in decreased_items:
            try:
                await increase_stock(product_id, quantity)
            except ProductClientError:
                pass
        raise


async def get_order(order_id: UUID, user_id: UUID) -> OrderResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        order = await querier.get_order_by_id(id=order_id)
        if order is None:
            raise OrderServiceError("NOT_FOUND", "주문을 찾을 수 없습니다.", 404)

        if order.user_id != user_id:
            raise OrderServiceError("FORBIDDEN", "접근 권한이 없습니다.", 403)

        items: list[OrdersOrderItem] = []
        async for item in querier.get_order_items_by_order_id(order_id=order_id):
            items.append(item)

        return _order_to_response(order, items)


def _rows_to_orders(
    rows: list[ListOrdersWithItemsByUserIDRow | ListOrdersWithItemsByUserIDAndStatusRow],
) -> list[OrderResponse]:
    """JOIN 결과를 주문 목록으로 변환 (주문별로 아이템 그룹핑)"""
    orders_dict: dict[UUID, tuple[OrdersOrder, list[OrdersOrderItem]]] = {}

    for row in rows:
        order_id = row.o_id

        if order_id not in orders_dict:
            # 주문 정보 생성
            order = OrdersOrder(
                id=row.o_id,
                user_id=row.o_user_id,
                total_amount=row.o_total_amount,
                status=row.o_status,
                recipient_name=row.o_recipient_name,
                phone=row.o_phone,
                address=row.o_address,
                address_detail=row.o_address_detail,
                postal_code=row.o_postal_code,
                cancelled_at=row.o_cancelled_at,
                cancel_reason=row.o_cancel_reason,
                created_at=row.o_created_at,
                updated_at=row.o_updated_at,
            )
            orders_dict[order_id] = (order, [])

        # 아이템 추가 (LEFT JOIN이므로 아이템이 없을 수 있음)
        if row.i_id is not None:
            item = OrdersOrderItem(
                id=row.i_id,
                order_id=order_id,
                product_id=row.i_product_id,
                deal_id=row.i_deal_id,
                product_name=row.i_product_name,
                quantity=row.i_quantity,
                unit_price=row.i_unit_price,
                subtotal=row.i_subtotal,
                created_at=row.i_created_at,
            )
            orders_dict[order_id][1].append(item)

    return [_order_to_response(order, items) for order, items in orders_dict.values()]


async def list_orders(
    user_id: UUID,
    page: int = 1,
    size: int = 20,
    status: str | None = None,
) -> tuple[list[OrderResponse], int]:
    offset = (page - 1) * size
    # JOIN 쿼리는 주문당 아이템 수만큼 행이 반환되므로 limit 조정
    # 주문 20개 × 아이템 최대 10개 = 200행으로 넉넉하게
    join_limit = size * 10

    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        # Count
        if status:
            total = await querier.count_orders_by_user_id_and_status(user_id=user_id, status=status)
        else:
            total = await querier.count_orders_by_user_id(user_id=user_id)
        total = total or 0

        # List with JOIN (N+1 해결)
        rows: list[ListOrdersWithItemsByUserIDRow | ListOrdersWithItemsByUserIDAndStatusRow] = []
        if status:
            async for row in querier.list_orders_with_items_by_user_id_and_status(
                user_id=user_id, status=status, limit=join_limit, offset=offset
            ):
                rows.append(row)
        else:
            async for row in querier.list_orders_with_items_by_user_id(
                user_id=user_id, limit=join_limit, offset=offset
            ):
                rows.append(row)

        orders_list = _rows_to_orders(rows)

        # 페이지 크기에 맞게 자르기
        return orders_list[:size], total


async def cancel_order(order_id: UUID, user_id: UUID, reason: str | None = None) -> OrderResponse:
    async with get_connection() as conn:
        querier = AsyncQuerier(conn)

        # 주문 조회 및 잠금
        order_lock = await querier.get_order_for_update(id=order_id)
        if order_lock is None:
            raise OrderServiceError("NOT_FOUND", "주문을 찾을 수 없습니다.", 404)

        if order_lock.user_id != user_id:
            raise OrderServiceError("FORBIDDEN", "접근 권한이 없습니다.", 403)

        # 취소 가능한 상태 확인
        if order_lock.status not in ("pending", "confirmed"):
            raise OrderServiceError(
                "CANNOT_CANCEL",
                f"취소할 수 없는 주문 상태입니다: {order_lock.status}",
                400,
            )

        # 주문 아이템 조회 (재고 복구용)
        items: list[OrdersOrderItem] = []
        async for item in querier.get_order_items_by_order_id(order_id=order_id):
            items.append(item)

        # 주문 취소
        order = await querier.cancel_order(id=order_id, cancel_reason=reason)
        if order is None:
            raise OrderServiceError("CANCEL_FAILED", "주문 취소에 실패했습니다.", 500)

        await conn.commit()

    # 재고 복구 (트랜잭션 외부)
    for item in items:
        try:
            await increase_stock(item.product_id, item.quantity)
        except ProductClientError:
            pass  # 재고 복구 실패는 로깅만

    return _order_to_response(order, items)
