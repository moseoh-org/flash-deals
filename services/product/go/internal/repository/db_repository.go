package repository

import (
	"context"

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
