-- Product queries

-- name: CreateProduct :one
INSERT INTO product.products (name, description, price, stock, category, image_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, description, price, stock, category, image_url, created_at, updated_at;

-- name: GetProductByID :one
SELECT id, name, description, price, stock, category, image_url, created_at, updated_at
FROM product.products
WHERE id = $1;

-- name: ListProducts :many
SELECT id, name, description, price, stock, category, image_url, created_at, updated_at
FROM product.products
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListProductsByCategory :many
SELECT id, name, description, price, stock, category, image_url, created_at, updated_at
FROM product.products
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountProducts :one
SELECT COUNT(*) FROM product.products;

-- name: CountProductsByCategory :one
SELECT COUNT(*) FROM product.products WHERE category = $1;

-- name: UpdateProduct :one
UPDATE product.products
SET name = COALESCE($2, name),
    description = COALESCE($3, description),
    price = COALESCE($4, price),
    category = COALESCE($5, category),
    image_url = COALESCE($6, image_url),
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, price, stock, category, image_url, created_at, updated_at;

-- name: GetStockForUpdate :one
SELECT id, stock FROM product.products WHERE id = $1 FOR UPDATE;

-- name: UpdateStock :one
UPDATE product.products
SET stock = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, stock, updated_at;

-- Deal queries

-- name: CreateDeal :one
INSERT INTO product.deals (product_id, deal_price, deal_stock, remaining_stock, starts_at, ends_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, product_id, deal_price, deal_stock, remaining_stock, starts_at, ends_at, created_at;

-- name: GetDealByID :one
SELECT d.id, d.product_id, d.deal_price, d.deal_stock, d.remaining_stock,
       d.starts_at, d.ends_at, d.created_at,
       p.id as p_id, p.name as p_name, p.description as p_description,
       p.price as p_price, p.stock as p_stock, p.category as p_category,
       p.image_url as p_image_url, p.created_at as p_created_at, p.updated_at as p_updated_at
FROM product.deals d
JOIN product.products p ON d.product_id = p.id
WHERE d.id = $1;

-- name: ListActiveDeals :many
SELECT d.id, d.product_id, d.deal_price, d.deal_stock, d.remaining_stock,
       d.starts_at, d.ends_at, d.created_at,
       p.id as p_id, p.name as p_name, p.description as p_description,
       p.price as p_price, p.stock as p_stock, p.category as p_category,
       p.image_url as p_image_url, p.created_at as p_created_at, p.updated_at as p_updated_at
FROM product.deals d
JOIN product.products p ON d.product_id = p.id
WHERE d.starts_at <= $1 AND d.ends_at >= $1 AND d.remaining_stock > 0
ORDER BY d.starts_at DESC
LIMIT $2 OFFSET $3;

-- name: CountActiveDeals :one
SELECT COUNT(*) FROM product.deals
WHERE starts_at <= $1 AND ends_at >= $1 AND remaining_stock > 0;
