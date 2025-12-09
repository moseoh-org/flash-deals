-- 상품 더미 데이터 시드 (1000만 건)
-- 용도: UUID v4 INSERT 성능 저하 테스트를 위한 대규모 데이터 적재
-- 실행: make seed-products-10m
-- 예상 소요: 10-20분

-- 기존 데이터 삭제
TRUNCATE product.products CASCADE;

-- 1000만 건 상품 데이터 생성 (100만 건씩 10회)
DO $$
DECLARE
    batch_size INTEGER := 1000000;
    total_batches INTEGER := 10;
    i INTEGER;
BEGIN
    FOR i IN 1..total_batches LOOP
        RAISE NOTICE 'Inserting batch %/% (% rows)...', i, total_batches, batch_size;

        INSERT INTO product.products (id, name, description, price, stock, category, created_at, updated_at)
        SELECT
            gen_random_uuid(),
            'Product ' || ((i-1) * batch_size + j),
            'Description for product ' || ((i-1) * batch_size + j),
            (random() * 100000 + 10000)::INTEGER,
            (random() * 100 + 1)::INTEGER,
            (ARRAY['electronics', 'fashion', 'food', 'home', 'sports'])[1 + (random() * 4)::INTEGER],
            NOW() - (random() * INTERVAL '365 days'),
            NOW() - (random() * INTERVAL '30 days')
        FROM generate_series(1, batch_size) AS j;

        RAISE NOTICE 'Batch % complete. Total: % rows', i, i * batch_size;
    END LOOP;
END $$;

-- 적재 결과 확인
SELECT
    COUNT(*) AS total_products,
    pg_size_pretty(pg_total_relation_size('product.products')) AS table_size
FROM product.products;
