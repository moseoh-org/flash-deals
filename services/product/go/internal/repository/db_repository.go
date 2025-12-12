package repository

import (
	"context"
	"fmt"

	"github.com/flash-deals/product/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Products

func (r *DBRepository) CreateProduct(ctx context.Context, arg db.CreateProductParams) (*db.ProductProduct, error) {
	product, err := r.queries.CreateProduct(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *DBRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*db.ProductProduct, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	product, err := r.queries.GetProductByID(ctx, pgID)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *DBRepository) ListProducts(ctx context.Context, limit, offset int32) ([]db.ProductProduct, error) {
	return r.queries.ListProducts(ctx, db.ListProductsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *DBRepository) ListProductsByCategory(ctx context.Context, category string, limit, offset int32) ([]db.ProductProduct, error) {
	return r.queries.ListProductsByCategory(ctx, db.ListProductsByCategoryParams{
		Category: pgtype.Text{String: category, Valid: true},
		Limit:    limit,
		Offset:   offset,
	})
}

func (r *DBRepository) CountProducts(ctx context.Context) (int64, error) {
	return r.queries.CountProducts(ctx)
}

func (r *DBRepository) CountProductsByCategory(ctx context.Context, category string) (int64, error) {
	return r.queries.CountProductsByCategory(ctx, pgtype.Text{String: category, Valid: true})
}

func (r *DBRepository) UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (*db.ProductProduct, error) {
	product, err := r.queries.UpdateProduct(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *DBRepository) GetStockForUpdate(ctx context.Context, id uuid.UUID) (*db.GetStockForUpdateRow, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	row, err := r.queries.GetStockForUpdate(ctx, pgID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DBRepository) UpdateStock(ctx context.Context, id uuid.UUID, stock int32) (*db.UpdateStockRow, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	row, err := r.queries.UpdateStock(ctx, db.UpdateStockParams{
		ID:    pgID,
		Stock: stock,
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// UpdateStockWithLock: 트랜잭션 내에서 FOR UPDATE 락과 재고 업데이트를 원자적으로 수행
func (r *DBRepository) UpdateStockWithLock(ctx context.Context, id uuid.UUID, delta int32) (*db.UpdateStockRow, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}

	// 트랜잭션 시작
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 트랜잭션 내에서 쿼리 실행
	queries := db.New(tx)

	// 1. SELECT FOR UPDATE로 락 획득
	current, err := queries.GetStockForUpdate(ctx, pgID)
	if err != nil {
		return nil, err
	}

	// 2. 재고 검증
	newStock := current.Stock + delta
	if newStock < 0 {
		return nil, fmt.Errorf("insufficient stock: current=%d, delta=%d", current.Stock, delta)
	}

	// 3. 재고 업데이트 (같은 트랜잭션 내에서)
	row, err := queries.UpdateStock(ctx, db.UpdateStockParams{
		ID:    pgID,
		Stock: newStock,
	})
	if err != nil {
		return nil, err
	}

	// 4. 트랜잭션 커밋
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &row, nil
}

// Deals

func (r *DBRepository) CreateDeal(ctx context.Context, arg db.CreateDealParams) (*db.ProductDeal, error) {
	deal, err := r.queries.CreateDeal(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &deal, nil
}

func (r *DBRepository) GetDealByID(ctx context.Context, id uuid.UUID) (*db.GetDealByIDRow, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	row, err := r.queries.GetDealByID(ctx, pgID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DBRepository) ListActiveDeals(ctx context.Context, now pgtype.Timestamptz, limit, offset int32) ([]db.ListActiveDealsRow, error) {
	return r.queries.ListActiveDeals(ctx, db.ListActiveDealsParams{
		StartsAt: now,
		Limit:    limit,
		Offset:   offset,
	})
}

func (r *DBRepository) CountActiveDeals(ctx context.Context, now pgtype.Timestamptz) (int64, error) {
	return r.queries.CountActiveDeals(ctx, now)
}

// Cache invalidation (no-op for DB-only repository)

func (r *DBRepository) InvalidateProduct(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *DBRepository) InvalidateProductList(ctx context.Context) error {
	return nil
}
