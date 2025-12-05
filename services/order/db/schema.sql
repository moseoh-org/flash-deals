-- Order Service Schema (sqlc용)
CREATE SCHEMA IF NOT EXISTS orders;

-- Order status enum
CREATE TYPE orders.order_status AS ENUM ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled');

-- Orders table
CREATE TABLE orders.orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    total_amount INTEGER NOT NULL CHECK (total_amount >= 0),
    status orders.order_status NOT NULL DEFAULT 'pending',
    -- Shipping address (denormalized for history)
    recipient_name VARCHAR(50),
    phone VARCHAR(20),
    address VARCHAR(200),
    address_detail VARCHAR(100),
    postal_code VARCHAR(10),
    -- Cancellation
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Order items table
CREATE TABLE orders.order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders.orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    deal_id UUID,
    product_name VARCHAR(200) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity >= 1),
    unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
    subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_orders_user_id ON orders.orders(user_id);
CREATE INDEX idx_orders_status ON orders.orders(status);
CREATE INDEX idx_orders_created_at ON orders.orders(created_at DESC);
CREATE INDEX idx_order_items_order_id ON orders.order_items(order_id);
