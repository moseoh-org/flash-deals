-- Product Service Schema (sqlc용)
CREATE SCHEMA IF NOT EXISTS product;

CREATE TABLE product.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price INTEGER NOT NULL CHECK (price >= 0),
    stock INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    category VARCHAR(50),
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE product.deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES product.products(id),
    deal_price INTEGER NOT NULL CHECK (deal_price >= 0),
    deal_stock INTEGER NOT NULL CHECK (deal_stock >= 1),
    remaining_stock INTEGER NOT NULL CHECK (remaining_stock >= 0),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_deal_period CHECK (ends_at > starts_at),
    CONSTRAINT valid_remaining_stock CHECK (remaining_stock <= deal_stock)
);

CREATE INDEX idx_products_category ON product.products(category);
CREATE INDEX idx_deals_product_id ON product.deals(product_id);
CREATE INDEX idx_deals_active ON product.deals(starts_at, ends_at, remaining_stock);
