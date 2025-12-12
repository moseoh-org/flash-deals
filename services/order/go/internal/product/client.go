package product

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/flash-deals/order/internal/proto"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var (
	client     proto.ProductServiceClient
	conn       *grpc.ClientConn
	clientOnce sync.Once
)

type ProductInfo struct {
	ID          string
	Name        string
	Description string
	Price       int32
	Stock       int32
}

type DealInfo struct {
	ID         string
	ProductID  string
	DealPrice  int32
	StockLimit int32
	Status     string
	Product    *ProductInfo
}

type StockResult struct {
	ProductID string
	Stock     int32
}

type ClientError struct {
	Code    string
	Message string
	Status  int
}

func (e *ClientError) Error() string {
	return e.Message
}

func InitClient(addr string, otelEnabled bool) error {
	var initErr error
	clientOnce.Do(func() {
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		if otelEnabled {
			opts = append(opts,
				grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
			)
		}

		var err error
		conn, err = grpc.NewClient(addr, opts...)
		if err != nil {
			initErr = fmt.Errorf("failed to connect to product service: %w", err)
			return
		}
		client = proto.NewProductServiceClient(conn)
		log.Printf("Product gRPC client connected: %s", addr)
	})
	return initErr
}

func Close() {
	if conn != nil {
		conn.Close()
	}
}

func GetProduct(ctx context.Context, productID uuid.UUID) (*ProductInfo, error) {
	if client == nil {
		return nil, &ClientError{Code: "CLIENT_NOT_INITIALIZED", Message: "Product client not initialized", Status: 500}
	}

	resp, err := client.GetProduct(ctx, &proto.GetProductRequest{ProductId: productID.String()})
	if err != nil {
		return nil, handleGRPCError(err, "Product", productID.String())
	}

	return &ProductInfo{
		ID:          resp.Id,
		Name:        resp.Name,
		Description: resp.Description,
		Price:       resp.Price,
		Stock:       resp.Stock,
	}, nil
}

func GetDeal(ctx context.Context, dealID uuid.UUID) (*DealInfo, error) {
	if client == nil {
		return nil, &ClientError{Code: "CLIENT_NOT_INITIALIZED", Message: "Product client not initialized", Status: 500}
	}

	resp, err := client.GetDeal(ctx, &proto.GetDealRequest{DealId: dealID.String()})
	if err != nil {
		return nil, handleGRPCError(err, "Deal", dealID.String())
	}

	deal := &DealInfo{
		ID:         resp.Id,
		ProductID:  resp.ProductId,
		DealPrice:  resp.DealPrice,
		StockLimit: resp.StockLimit,
		Status:     resp.Status,
	}

	if resp.Product != nil {
		deal.Product = &ProductInfo{
			ID:          resp.Product.Id,
			Name:        resp.Product.Name,
			Description: resp.Product.Description,
			Price:       resp.Product.Price,
			Stock:       resp.Product.Stock,
		}
	}

	return deal, nil
}

func DecreaseStock(ctx context.Context, productID uuid.UUID, quantity int32) (*StockResult, error) {
	if client == nil {
		return nil, &ClientError{Code: "CLIENT_NOT_INITIALIZED", Message: "Product client not initialized", Status: 500}
	}

	resp, err := client.UpdateStock(ctx, &proto.UpdateStockRequest{
		ProductId: productID.String(),
		Delta:     -quantity,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.FailedPrecondition {
			return nil, &ClientError{Code: "INSUFFICIENT_STOCK", Message: "재고가 부족합니다.", Status: 400}
		}
		return nil, handleGRPCError(err, "Product", productID.String())
	}

	return &StockResult{
		ProductID: resp.Id,
		Stock:     resp.Stock,
	}, nil
}

func IncreaseStock(ctx context.Context, productID uuid.UUID, quantity int32) (*StockResult, error) {
	if client == nil {
		return nil, &ClientError{Code: "CLIENT_NOT_INITIALIZED", Message: "Product client not initialized", Status: 500}
	}

	resp, err := client.UpdateStock(ctx, &proto.UpdateStockRequest{
		ProductId: productID.String(),
		Delta:     quantity,
	})
	if err != nil {
		return nil, handleGRPCError(err, "Product", productID.String())
	}

	return &StockResult{
		ProductID: resp.Id,
		Stock:     resp.Stock,
	}, nil
}

func handleGRPCError(err error, resourceType, resourceID string) *ClientError {
	st, ok := status.FromError(err)
	if !ok {
		return &ClientError{
			Code:    "PRODUCT_SERVICE_ERROR",
			Message: fmt.Sprintf("%s 서비스 오류: %v", resourceType, err),
			Status:  502,
		}
	}

	switch st.Code() {
	case codes.NotFound:
		return &ClientError{
			Code:    fmt.Sprintf("%s_NOT_FOUND", resourceType),
			Message: fmt.Sprintf("%s을(를) 찾을 수 없습니다: %s", resourceType, resourceID),
			Status:  404,
		}
	case codes.InvalidArgument:
		return &ClientError{
			Code:    fmt.Sprintf("INVALID_%s_ID", resourceType),
			Message: st.Message(),
			Status:  400,
		}
	default:
		return &ClientError{
			Code:    "PRODUCT_SERVICE_ERROR",
			Message: fmt.Sprintf("%s 서비스 오류: %s", resourceType, st.Message()),
			Status:  502,
		}
	}
}
