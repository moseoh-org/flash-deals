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
