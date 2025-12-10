-- Order queries

-- name: CreateOrder :one
INSERT INTO orders.orders (user_id, total_amount, status, recipient_name, phone, address, address_detail, postal_code)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, total_amount, status, recipient_name, phone, address, address_detail, postal_code, cancelled_at, cancel_reason, created_at, updated_at;

-- name: CreateOrderItem :one
INSERT INTO orders.order_items (order_id, product_id, deal_id, product_name, quantity, unit_price, subtotal)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, order_id, product_id, deal_id, product_name, quantity, unit_price, subtotal, created_at;

-- name: GetOrderByID :one
SELECT id, user_id, total_amount, status, recipient_name, phone, address, address_detail, postal_code, cancelled_at, cancel_reason, created_at, updated_at
FROM orders.orders
WHERE id = $1;

-- name: GetOrderItemsByOrderID :many
SELECT id, order_id, product_id, deal_id, product_name, quantity, unit_price, subtotal, created_at
FROM orders.order_items
WHERE order_id = $1
ORDER BY created_at;

-- name: ListOrdersByUserID :many
SELECT id, user_id, total_amount, status, recipient_name, phone, address, address_detail, postal_code, cancelled_at, cancel_reason, created_at, updated_at
FROM orders.orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListOrdersByUserIDAndStatus :many
SELECT id, user_id, total_amount, status, recipient_name, phone, address, address_detail, postal_code, cancelled_at, cancel_reason, created_at, updated_at
FROM orders.orders
WHERE user_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountOrdersByUserID :one
SELECT COUNT(*) FROM orders.orders WHERE user_id = $1;

-- name: CountOrdersByUserIDAndStatus :one
SELECT COUNT(*) FROM orders.orders WHERE user_id = $1 AND status = $2;

-- name: UpdateOrderStatus :one
UPDATE orders.orders
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, total_amount, status, recipient_name, phone, address, address_detail, postal_code, cancelled_at, cancel_reason, created_at, updated_at;

-- name: CancelOrder :one
UPDATE orders.orders
SET status = 'cancelled', cancelled_at = NOW(), cancel_reason = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, total_amount, status, recipient_name, phone, address, address_detail, postal_code, cancelled_at, cancel_reason, created_at, updated_at;

-- name: GetOrderForUpdate :one
SELECT id, user_id, status FROM orders.orders WHERE id = $1 FOR UPDATE;

-- name: ListOrdersWithItemsByUserID :many
-- N+1 문제 해결: 주문 + 아이템을 한 번에 조회
SELECT
    o.id AS o_id,
    o.user_id AS o_user_id,
    o.total_amount AS o_total_amount,
    o.status AS o_status,
    o.recipient_name AS o_recipient_name,
    o.phone AS o_phone,
    o.address AS o_address,
    o.address_detail AS o_address_detail,
    o.postal_code AS o_postal_code,
    o.cancelled_at AS o_cancelled_at,
    o.cancel_reason AS o_cancel_reason,
    o.created_at AS o_created_at,
    o.updated_at AS o_updated_at,
    i.id AS i_id,
    i.product_id AS i_product_id,
    i.deal_id AS i_deal_id,
    i.product_name AS i_product_name,
    i.quantity AS i_quantity,
    i.unit_price AS i_unit_price,
    i.subtotal AS i_subtotal,
    i.created_at AS i_created_at
FROM orders.orders o
LEFT JOIN orders.order_items i ON o.id = i.order_id
WHERE o.user_id = $1
ORDER BY o.created_at DESC, i.created_at
LIMIT $2 OFFSET $3;

-- name: ListOrdersWithItemsByUserIDAndStatus :many
SELECT
    o.id AS o_id,
    o.user_id AS o_user_id,
    o.total_amount AS o_total_amount,
    o.status AS o_status,
    o.recipient_name AS o_recipient_name,
    o.phone AS o_phone,
    o.address AS o_address,
    o.address_detail AS o_address_detail,
    o.postal_code AS o_postal_code,
    o.cancelled_at AS o_cancelled_at,
    o.cancel_reason AS o_cancel_reason,
    o.created_at AS o_created_at,
    o.updated_at AS o_updated_at,
    i.id AS i_id,
    i.product_id AS i_product_id,
    i.deal_id AS i_deal_id,
    i.product_name AS i_product_name,
    i.quantity AS i_quantity,
    i.unit_price AS i_unit_price,
    i.subtotal AS i_subtotal,
    i.created_at AS i_created_at
FROM orders.orders o
LEFT JOIN orders.order_items i ON o.id = i.order_id
WHERE o.user_id = $1 AND o.status = $2
ORDER BY o.created_at DESC, i.created_at
LIMIT $3 OFFSET $4;
