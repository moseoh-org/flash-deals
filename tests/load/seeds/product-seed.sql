-- 상품 더미 데이터 시드 (100만 건)
-- 용도: UUID v4 INSERT 성능 저하 테스트를 위한 사전 데이터 적재
-- 실행: make seed-products

-- 기존 데이터 삭제 (선택적)
-- TRUNCATE product.products CASCADE;

-- 100만 건 상품 데이터 생성
-- gen_random_uuid()는 UUID v4를 생성
INSERT INTO product.products (id, name, description, price, stock, category, created_at, updated_at)
SELECT
    gen_random_uuid(),
    'Product ' || i,
    'Description for product ' || i || '. This is a dummy product for load testing.',
    (random() * 100000 + 10000)::INTEGER,
    (random() * 100 + 1)::INTEGER,
    (ARRAY['electronics', 'fashion', 'food', 'home', 'sports'])[1 + (random() * 4)::INTEGER],
    NOW() - (random() * INTERVAL '365 days'),
    NOW() - (random() * INTERVAL '30 days')
FROM generate_series(1, 1000000) AS i;

-- 적재 결과 확인
SELECT
    COUNT(*) AS total_products,
    pg_size_pretty(pg_total_relation_size('product.products')) AS table_size
FROM product.products;
