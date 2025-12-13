package repository

import (
	"context"

	"github.com/flash-deals/product/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductRepository interface {
	// Products
	CreateProduct(ctx context.Context, arg db.CreateProductParams) (*db.ProductProduct, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*db.ProductProduct, error)
	ListProducts(ctx context.Context, limit, offset int32) ([]db.ProductProduct, error)
	ListProductsByCategory(ctx context.Context, category string, limit, offset int32) ([]db.ProductProduct, error)
	CountProducts(ctx context.Context) (int64, error)
	CountProductsByCategory(ctx context.Context, category string) (int64, error)
	UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (*db.ProductProduct, error)
	GetStockForUpdate(ctx context.Context, id uuid.UUID) (*db.GetStockForUpdateRow, error)
	UpdateStock(ctx context.Context, id uuid.UUID, stock int32) (*db.UpdateStockRow, error)
	UpdateStockWithLock(ctx context.Context, id uuid.UUID, delta int32) (*db.UpdateStockRow, error)

	// Deals
	CreateDeal(ctx context.Context, arg db.CreateDealParams) (*db.ProductDeal, error)
	GetDealByID(ctx context.Context, id uuid.UUID) (*db.GetDealByIDRow, error)
	ListActiveDeals(ctx context.Context, now pgtype.Timestamptz, limit, offset int32) ([]db.ListActiveDealsRow, error)
	CountActiveDeals(ctx context.Context, now pgtype.Timestamptz) (int64, error)

	// Cache invalidation
	InvalidateProduct(ctx context.Context, id uuid.UUID) error
	InvalidateProductList(ctx context.Context) error
}
