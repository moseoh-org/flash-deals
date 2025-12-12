package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flash-deals/product/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

type CachedRepository struct {
	dbRepo *DBRepository
	redis  *redis.Client
	ttl    time.Duration
}

func NewCachedRepository(dbRepo *DBRepository, redisClient *redis.Client, ttlSeconds int) *CachedRepository {
	return &CachedRepository{
		dbRepo: dbRepo,
		redis:  redisClient,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

func (r *CachedRepository) productKey(id uuid.UUID) string {
	return fmt.Sprintf("product:%s", id.String())
}

func (r *CachedRepository) productListKey(limit, offset int32) string {
	return fmt.Sprintf("products:list:%d:%d", limit, offset)
}

// Products

func (r *CachedRepository) CreateProduct(ctx context.Context, arg db.CreateProductParams) (*db.ProductProduct, error) {
	product, err := r.dbRepo.CreateProduct(ctx, arg)
	if err != nil {
		return nil, err
	}
	// Invalidate list cache
	r.InvalidateProductList(ctx)
	return product, nil
}

func (r *CachedRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*db.ProductProduct, error) {
	key := r.productKey(id)

	// Try cache
	cached, err := r.redis.Get(ctx, key).Bytes()
	if err == nil {
		var product db.ProductProduct
		if json.Unmarshal(cached, &product) == nil {
			return &product, nil
		}
	}

	// Cache miss - get from DB
	product, err := r.dbRepo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if data, err := json.Marshal(product); err == nil {
		r.redis.Set(ctx, key, data, r.ttl)
	}

	return product, nil
}

func (r *CachedRepository) ListProducts(ctx context.Context, limit, offset int32) ([]db.ProductProduct, error) {
	key := r.productListKey(limit, offset)

	// Try cache
	cached, err := r.redis.Get(ctx, key).Bytes()
	if err == nil {
		var products []db.ProductProduct
		if json.Unmarshal(cached, &products) == nil {
			return products, nil
		}
	}

	// Cache miss - get from DB
	products, err := r.dbRepo.ListProducts(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if data, err := json.Marshal(products); err == nil {
		r.redis.Set(ctx, key, data, r.ttl)
	}

	return products, nil
}

func (r *CachedRepository) ListProductsByCategory(ctx context.Context, category string, limit, offset int32) ([]db.ProductProduct, error) {
	// No cache for category filtering
	return r.dbRepo.ListProductsByCategory(ctx, category, limit, offset)
}

func (r *CachedRepository) CountProducts(ctx context.Context) (int64, error) {
	return r.dbRepo.CountProducts(ctx)
}

func (r *CachedRepository) CountProductsByCategory(ctx context.Context, category string) (int64, error) {
	return r.dbRepo.CountProductsByCategory(ctx, category)
}

func (r *CachedRepository) UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (*db.ProductProduct, error) {
	product, err := r.dbRepo.UpdateProduct(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	var id uuid.UUID
	copy(id[:], arg.ID.Bytes[:])
	r.InvalidateProduct(ctx, id)
	r.InvalidateProductList(ctx)

	return product, nil
}

func (r *CachedRepository) GetStockForUpdate(ctx context.Context, id uuid.UUID) (*db.GetStockForUpdateRow, error) {
	// Always go to DB for transactional operations
	return r.dbRepo.GetStockForUpdate(ctx, id)
}

func (r *CachedRepository) UpdateStock(ctx context.Context, id uuid.UUID, stock int32) (*db.UpdateStockRow, error) {
	row, err := r.dbRepo.UpdateStock(ctx, id, stock)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	r.InvalidateProduct(ctx, id)

	return row, nil
}

// Deals (no caching for deals)

func (r *CachedRepository) CreateDeal(ctx context.Context, arg db.CreateDealParams) (*db.ProductDeal, error) {
	return r.dbRepo.CreateDeal(ctx, arg)
}

func (r *CachedRepository) GetDealByID(ctx context.Context, id uuid.UUID) (*db.GetDealByIDRow, error) {
	return r.dbRepo.GetDealByID(ctx, id)
}

func (r *CachedRepository) ListActiveDeals(ctx context.Context, now pgtype.Timestamptz, limit, offset int32) ([]db.ListActiveDealsRow, error) {
	return r.dbRepo.ListActiveDeals(ctx, now, limit, offset)
}

func (r *CachedRepository) CountActiveDeals(ctx context.Context, now pgtype.Timestamptz) (int64, error) {
	return r.dbRepo.CountActiveDeals(ctx, now)
}

// Cache invalidation

func (r *CachedRepository) InvalidateProduct(ctx context.Context, id uuid.UUID) error {
	return r.redis.Del(ctx, r.productKey(id)).Err()
}

func (r *CachedRepository) InvalidateProductList(ctx context.Context) error {
	// Delete all list cache keys (pattern match)
	iter := r.redis.Scan(ctx, 0, "products:list:*", 100).Iterator()
	for iter.Next(ctx) {
		r.redis.Del(ctx, iter.Val())
	}
	return iter.Err()
}
