package grpc

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/flash-deals/product/internal/config"
	"github.com/flash-deals/product/internal/proto"
	"github.com/flash-deals/product/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductServer struct {
	proto.UnimplementedProductServiceServer
	repo repository.ProductRepository
}

func NewProductServer(repo repository.ProductRepository) *ProductServer {
	return &ProductServer{repo: repo}
}

func (s *ProductServer) GetProduct(ctx context.Context, req *proto.GetProductRequest) (*proto.Product, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
	}

	product, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "product not found: %v", err)
	}

	resp := &proto.Product{
		Id:        uuidToString(product.ID),
		Name:      product.Name,
		Price:     product.Price,
		Stock:     product.Stock,
		CreatedAt: product.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: product.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
	if product.Description.Valid {
		resp.Description = product.Description.String
	}

	return resp, nil
}

func (s *ProductServer) GetDeal(ctx context.Context, req *proto.GetDealRequest) (*proto.Deal, error) {
	id, err := uuid.Parse(req.DealId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid deal ID: %v", err)
	}

	deal, err := s.repo.GetDealByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "deal not found: %v", err)
	}

	resp := &proto.Deal{
		Id:         uuidToString(deal.ID),
		ProductId:  uuidToString(deal.ProductID),
		DealPrice:  deal.DealPrice,
		StockLimit: deal.DealStock,
		StartTime:  deal.StartsAt.Time.Format("2006-01-02T15:04:05Z"),
		EndTime:    deal.EndsAt.Time.Format("2006-01-02T15:04:05Z"),
		Status:     getDealStatus(deal),
		Product: &proto.Product{
			Id:        uuidToString(deal.PID),
			Name:      deal.PName,
			Price:     deal.PPrice,
			Stock:     deal.PStock,
			CreatedAt: deal.PCreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: deal.PUpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		},
	}

	if deal.PDescription.Valid {
		resp.Product.Description = deal.PDescription.String
	}

	return resp, nil
}

func (s *ProductServer) UpdateStock(ctx context.Context, req *proto.UpdateStockRequest) (*proto.Product, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
	}

	// 트랜잭션 내에서 FOR UPDATE 락과 재고 업데이트를 원자적으로 수행
	row, err := s.repo.UpdateStockWithLock(ctx, id, req.Delta)
	if err != nil {
		// 재고 부족 에러 처리
		errMsg := err.Error()
		if len(errMsg) >= 18 && errMsg[:18] == "insufficient stock" {
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient stock")
		}
		return nil, status.Errorf(codes.Internal, "failed to update stock: %v", err)
	}

	// Get full product for response
	product, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}

	resp := &proto.Product{
		Id:        uuidToString(product.ID),
		Name:      product.Name,
		Price:     product.Price,
		Stock:     row.Stock,
		CreatedAt: product.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
	if product.Description.Valid {
		resp.Description = product.Description.String
	}

	return resp, nil
}

func StartGRPCServer(cfg *config.Config, repo repository.ProductRepository, otelEnabled bool) error {
	addr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	var opts []grpc.ServerOption
	if otelEnabled {
		opts = append(opts,
			grpc.StatsHandler(otelgrpc.NewServerHandler()),
		)
	}

	server := grpc.NewServer(opts...)
	proto.RegisterProductServiceServer(server, NewProductServer(repo))

	log.Printf("gRPC server starting on %s", addr)
	return server.Serve(lis)
}

// Helpers

func uuidToString(id interface{}) string {
	switch v := id.(type) {
	case pgtype.UUID:
		return uuid.UUID(v.Bytes).String()
	case [16]byte:
		return uuid.UUID(v).String()
	default:
		return fmt.Sprintf("%v", id)
	}
}

func getDealStatus(deal interface{}) string {
	return "active"
}
