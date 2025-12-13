package service

import (
	"context"
	"fmt"
	"time"

	"github.com/flash-deals/order/internal/db"
	"github.com/flash-deals/order/internal/product"
	"github.com/flash-deals/order/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderService struct {
	pool       *pgxpool.Pool
	orderQueue *queue.OrderQueue
}

func NewOrderService(pool *pgxpool.Pool) *OrderService {
	return &OrderService{pool: pool}
}

// SetOrderQueue sets the order queue for async processing
func (s *OrderService) SetOrderQueue(q *queue.OrderQueue) {
	s.orderQueue = q
}

type ServiceError struct {
	Code    string
	Message string
	Status  int
}

func (e *ServiceError) Error() string {
	return e.Message
}

type OrderItemRequest struct {
	ProductID uuid.UUID
	DealID    *uuid.UUID
	Quantity  int32
}

type ShippingAddress struct {
	RecipientName string
	Phone         string
	Address       string
	AddressDetail *string
	PostalCode    string
}

type OrderItemResponse struct {
	ID          uuid.UUID
	ProductID   uuid.UUID
	DealID      *uuid.UUID
	ProductName string
	Quantity    int32
	UnitPrice   int32
	Subtotal    int32
}

type OrderResponse struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Items           []OrderItemResponse
	TotalAmount     int32
	Status          string
	ShippingAddress *ShippingAddress
	CancelledAt     *time.Time
	CancelReason    *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, items []OrderItemRequest, shipping *ShippingAddress) (*OrderResponse, error) {
	// 1. 상품 정보 조회 및 가격 계산
	type orderItemData struct {
		ProductID   uuid.UUID
		DealID      *uuid.UUID
		ProductName string
		Quantity    int32
		UnitPrice   int32
		Subtotal    int32
	}
	orderItemsData := make([]orderItemData, 0, len(items))
	var totalAmount int32

	for _, item := range items {
		var unitPrice int32
		var productName string

		if item.DealID != nil {
			// 핫딜 주문
			deal, err := product.GetDeal(ctx, *item.DealID)
			if err != nil {
				if ce, ok := err.(*product.ClientError); ok {
					return nil, &ServiceError{Code: ce.Code, Message: ce.Message, Status: ce.Status}
				}
				return nil, err
			}
			if deal.ProductID != item.ProductID.String() {
				return nil, &ServiceError{Code: "INVALID_DEAL", Message: "핫딜과 상품이 일치하지 않습니다.", Status: 400}
			}
			if deal.Status != "active" {
				return nil, &ServiceError{Code: "DEAL_NOT_ACTIVE", Message: "핫딜이 진행 중이 아닙니다.", Status: 400}
			}
			unitPrice = deal.DealPrice
			productName = deal.Product.Name
		} else {
			// 일반 주문
			prod, err := product.GetProduct(ctx, item.ProductID)
			if err != nil {
				if ce, ok := err.(*product.ClientError); ok {
					return nil, &ServiceError{Code: ce.Code, Message: ce.Message, Status: ce.Status}
				}
				return nil, err
			}
			unitPrice = prod.Price
			productName = prod.Name
		}

		subtotal := unitPrice * item.Quantity
		totalAmount += subtotal

		orderItemsData = append(orderItemsData, orderItemData{
			ProductID:   item.ProductID,
			DealID:      item.DealID,
			ProductName: productName,
			Quantity:    item.Quantity,
			UnitPrice:   unitPrice,
			Subtotal:    subtotal,
		})
	}

	// 2. 재고 차감 (보상 트랜잭션 패턴)
	type decreasedItem struct {
		ProductID uuid.UUID
		Quantity  int32
	}
	decreasedItems := make([]decreasedItem, 0, len(items))

	for _, item := range items {
		_, err := product.DecreaseStock(ctx, item.ProductID, item.Quantity)
		if err != nil {
			// 이미 차감한 재고 복구
			for _, d := range decreasedItems {
				product.IncreaseStock(ctx, d.ProductID, d.Quantity)
			}
			if ce, ok := err.(*product.ClientError); ok {
				return nil, &ServiceError{Code: ce.Code, Message: ce.Message, Status: ce.Status}
			}
			return nil, err
		}
		decreasedItems = append(decreasedItems, decreasedItem{ProductID: item.ProductID, Quantity: item.Quantity})
	}

	// 3. 주문 생성 (트랜잭션)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		// 재고 복구
		for _, d := range decreasedItems {
			product.IncreaseStock(ctx, d.ProductID, d.Quantity)
		}
		return nil, &ServiceError{Code: "CREATE_FAILED", Message: "주문 생성에 실패했습니다.", Status: 500}
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)

	// 주문 생성
	createParams := db.CreateOrderParams{
		UserID:      pgtype.UUID{Bytes: userID, Valid: true},
		TotalAmount: totalAmount,
		Status:      db.OrdersOrderStatusPending,
	}
	if shipping != nil {
		createParams.RecipientName = pgtype.Text{String: shipping.RecipientName, Valid: true}
		createParams.Phone = pgtype.Text{String: shipping.Phone, Valid: true}
		createParams.Address = pgtype.Text{String: shipping.Address, Valid: true}
		if shipping.AddressDetail != nil {
			createParams.AddressDetail = pgtype.Text{String: *shipping.AddressDetail, Valid: true}
		}
		createParams.PostalCode = pgtype.Text{String: shipping.PostalCode, Valid: true}
	}

	order, err := queries.CreateOrder(ctx, createParams)
	if err != nil {
		for _, d := range decreasedItems {
			product.IncreaseStock(ctx, d.ProductID, d.Quantity)
		}
		return nil, &ServiceError{Code: "CREATE_FAILED", Message: fmt.Sprintf("주문 생성에 실패했습니다: %v", err), Status: 500}
	}

	// 주문 아이템 생성
	createdItems := make([]db.OrdersOrderItem, 0, len(orderItemsData))
	for _, itemData := range orderItemsData {
		itemParams := db.CreateOrderItemParams{
			OrderID:     order.ID,
			ProductID:   pgtype.UUID{Bytes: itemData.ProductID, Valid: true},
			ProductName: itemData.ProductName,
			Quantity:    itemData.Quantity,
			UnitPrice:   itemData.UnitPrice,
			Subtotal:    itemData.Subtotal,
		}
		if itemData.DealID != nil {
			itemParams.DealID = pgtype.UUID{Bytes: *itemData.DealID, Valid: true}
		}

		orderItem, err := queries.CreateOrderItem(ctx, itemParams)
		if err != nil {
			for _, d := range decreasedItems {
				product.IncreaseStock(ctx, d.ProductID, d.Quantity)
			}
			return nil, &ServiceError{Code: "CREATE_FAILED", Message: "주문 아이템 생성에 실패했습니다.", Status: 500}
		}
		createdItems = append(createdItems, orderItem)
	}

	if err := tx.Commit(ctx); err != nil {
		for _, d := range decreasedItems {
			product.IncreaseStock(ctx, d.ProductID, d.Quantity)
		}
		return nil, &ServiceError{Code: "CREATE_FAILED", Message: "주문 커밋에 실패했습니다.", Status: 500}
	}

	return toOrderResponse(&order, createdItems, shipping), nil
}

// AsyncCreateOrder creates an order asynchronously via Redis queue
// Returns immediately with order ID after stock deduction
func (s *OrderService) AsyncCreateOrder(ctx context.Context, userID uuid.UUID, items []OrderItemRequest, shipping *ShippingAddress) (*OrderResponse, error) {
	if s.orderQueue == nil {
		return nil, &ServiceError{Code: "QUEUE_NOT_INITIALIZED", Message: "주문 큐가 초기화되지 않았습니다.", Status: 500}
	}

	// 1. Generate order ID upfront
	orderID := uuid.New()

	// 2. Get product info and calculate prices
	orderItems := make([]queue.OrderItemReq, 0, len(items))
	var totalAmount int32

	for _, item := range items {
		var unitPrice int32
		var productName string

		if item.DealID != nil {
			// Hot deal order
			deal, err := product.GetDeal(ctx, *item.DealID)
			if err != nil {
				if ce, ok := err.(*product.ClientError); ok {
					return nil, &ServiceError{Code: ce.Code, Message: ce.Message, Status: ce.Status}
				}
				return nil, err
			}
			if deal.ProductID != item.ProductID.String() {
				return nil, &ServiceError{Code: "INVALID_DEAL", Message: "핫딜과 상품이 일치하지 않습니다.", Status: 400}
			}
			if deal.Status != "active" {
				return nil, &ServiceError{Code: "DEAL_NOT_ACTIVE", Message: "핫딜이 진행 중이 아닙니다.", Status: 400}
			}
			unitPrice = deal.DealPrice
			productName = deal.Product.Name
		} else {
			// Regular order
			prod, err := product.GetProduct(ctx, item.ProductID)
			if err != nil {
				if ce, ok := err.(*product.ClientError); ok {
					return nil, &ServiceError{Code: ce.Code, Message: ce.Message, Status: ce.Status}
				}
				return nil, err
			}
			unitPrice = prod.Price
			productName = prod.Name
		}

		totalAmount += unitPrice * item.Quantity
		orderItems = append(orderItems, queue.OrderItemReq{
			ProductID:   item.ProductID,
			DealID:      item.DealID,
			Quantity:    item.Quantity,
			ProductName: productName,
			UnitPrice:   unitPrice,
		})
	}

	// 3. Deduct stock (compensating transaction pattern)
	type decreasedItem struct {
		ProductID uuid.UUID
		Quantity  int32
	}
	decreasedItems := make([]decreasedItem, 0, len(items))

	for _, item := range items {
		_, err := product.DecreaseStock(ctx, item.ProductID, item.Quantity)
		if err != nil {
			// Rollback already deducted stock
			for _, d := range decreasedItems {
				product.IncreaseStock(ctx, d.ProductID, d.Quantity)
			}
			if ce, ok := err.(*product.ClientError); ok {
				return nil, &ServiceError{Code: ce.Code, Message: ce.Message, Status: ce.Status}
			}
			return nil, err
		}
		decreasedItems = append(decreasedItems, decreasedItem{ProductID: item.ProductID, Quantity: item.Quantity})
	}

	// 4. Enqueue order for async processing
	var shippingAddr *queue.ShippingAddr
	if shipping != nil {
		shippingAddr = &queue.ShippingAddr{
			RecipientName: shipping.RecipientName,
			Phone:         shipping.Phone,
			Address:       shipping.Address,
			AddressDetail: shipping.AddressDetail,
			PostalCode:    shipping.PostalCode,
		}
	}

	orderReq := &queue.OrderRequest{
		OrderID:         orderID,
		UserID:          userID,
		Items:           orderItems,
		ShippingAddress: shippingAddr,
		CreatedAt:       time.Now(),
	}

	if err := s.orderQueue.Enqueue(ctx, orderReq); err != nil {
		// Rollback stock on queue failure
		for _, d := range decreasedItems {
			product.IncreaseStock(ctx, d.ProductID, d.Quantity)
		}
		return nil, &ServiceError{Code: "QUEUE_FAILED", Message: "주문 큐 등록에 실패했습니다.", Status: 500}
	}

	// 5. Return immediately with pending status
	itemResponses := make([]OrderItemResponse, 0, len(orderItems))
	for _, item := range orderItems {
		itemResponses = append(itemResponses, OrderItemResponse{
			ProductID:   item.ProductID,
			DealID:      item.DealID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.UnitPrice * item.Quantity,
		})
	}

	return &OrderResponse{
		ID:              orderID,
		UserID:          userID,
		Items:           itemResponses,
		TotalAmount:     totalAmount,
		Status:          "processing",
		ShippingAddress: shipping,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID, userID uuid.UUID) (*OrderResponse, error) {
	queries := db.New(s.pool)

	order, err := queries.GetOrderByID(ctx, pgtype.UUID{Bytes: orderID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &ServiceError{Code: "NOT_FOUND", Message: "주문을 찾을 수 없습니다.", Status: 404}
		}
		return nil, err
	}

	orderUserID := uuid.UUID(order.UserID.Bytes)
	if orderUserID != userID {
		return nil, &ServiceError{Code: "FORBIDDEN", Message: "접근 권한이 없습니다.", Status: 403}
	}

	items, err := queries.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	return toOrderResponse(&order, items, nil), nil
}

func (s *OrderService) ListOrders(ctx context.Context, userID uuid.UUID, page, size int32, status *string) ([]OrderResponse, int64, error) {
	queries := db.New(s.pool)
	offset := (page - 1) * size

	var total int64
	var err error

	if status != nil {
		params := db.CountOrdersByUserIDAndStatusParams{
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
			Status: db.OrdersOrderStatus(*status),
		}
		total, err = queries.CountOrdersByUserIDAndStatus(ctx, params)
	} else {
		total, err = queries.CountOrdersByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	}
	if err != nil {
		return nil, 0, err
	}

	// 간단한 리스트 조회 (N+1 방지를 위해 JOIN 쿼리 사용)
	var orders []db.OrdersOrder
	if status != nil {
		params := db.ListOrdersByUserIDAndStatusParams{
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
			Status: db.OrdersOrderStatus(*status),
			Limit:  size,
			Offset: offset,
		}
		orders, err = queries.ListOrdersByUserIDAndStatus(ctx, params)
	} else {
		params := db.ListOrdersByUserIDParams{
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
			Limit:  size,
			Offset: offset,
		}
		orders, err = queries.ListOrdersByUserID(ctx, params)
	}
	if err != nil {
		return nil, 0, err
	}

	results := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		items, _ := queries.GetOrderItemsByOrderID(ctx, order.ID)
		results = append(results, *toOrderResponse(&order, items, nil))
	}

	return results, total, nil
}

func (s *OrderService) ConfirmOrder(ctx context.Context, orderID, userID uuid.UUID) (*OrderResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, &ServiceError{Code: "CONFIRM_FAILED", Message: "주문 확정에 실패했습니다.", Status: 500}
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)

	// 주문 조회 및 잠금
	order, err := queries.GetOrderForUpdate(ctx, pgtype.UUID{Bytes: orderID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &ServiceError{Code: "NOT_FOUND", Message: "주문을 찾을 수 없습니다.", Status: 404}
		}
		return nil, err
	}

	orderUserID := uuid.UUID(order.UserID.Bytes)
	if orderUserID != userID {
		return nil, &ServiceError{Code: "FORBIDDEN", Message: "접근 권한이 없습니다.", Status: 403}
	}

	// PENDING 상태만 확정 가능
	if order.Status != db.OrdersOrderStatusPending {
		return nil, &ServiceError{
			Code:    "CANNOT_CONFIRM",
			Message: fmt.Sprintf("확정할 수 없는 주문 상태입니다: %s", order.Status),
			Status:  400,
		}
	}

	// 주문 확정
	confirmedOrder, err := queries.ConfirmOrder(ctx, pgtype.UUID{Bytes: orderID, Valid: true})
	if err != nil {
		return nil, &ServiceError{Code: "CONFIRM_FAILED", Message: "주문 확정에 실패했습니다.", Status: 500}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, &ServiceError{Code: "CONFIRM_FAILED", Message: "주문 확정 커밋에 실패했습니다.", Status: 500}
	}

	items, _ := queries.GetOrderItemsByOrderID(ctx, confirmedOrder.ID)
	return toOrderResponse(&confirmedOrder, items, nil), nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID uuid.UUID, reason *string) (*OrderResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, &ServiceError{Code: "CANCEL_FAILED", Message: "주문 취소에 실패했습니다.", Status: 500}
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)

	// 주문 조회 및 잠금
	orderLock, err := queries.GetOrderForUpdate(ctx, pgtype.UUID{Bytes: orderID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &ServiceError{Code: "NOT_FOUND", Message: "주문을 찾을 수 없습니다.", Status: 404}
		}
		return nil, err
	}

	lockUserID := uuid.UUID(orderLock.UserID.Bytes)
	if lockUserID != userID {
		return nil, &ServiceError{Code: "FORBIDDEN", Message: "접근 권한이 없습니다.", Status: 403}
	}

	// 취소 가능한 상태 확인
	if orderLock.Status != db.OrdersOrderStatusPending && orderLock.Status != db.OrdersOrderStatusConfirmed {
		return nil, &ServiceError{
			Code:    "CANNOT_CANCEL",
			Message: fmt.Sprintf("취소할 수 없는 주문 상태입니다: %s", orderLock.Status),
			Status:  400,
		}
	}

	// 주문 아이템 조회 (재고 복구용)
	items, err := queries.GetOrderItemsByOrderID(ctx, pgtype.UUID{Bytes: orderID, Valid: true})
	if err != nil {
		return nil, err
	}

	// 주문 취소
	cancelParams := db.CancelOrderParams{
		ID: pgtype.UUID{Bytes: orderID, Valid: true},
	}
	if reason != nil {
		cancelParams.CancelReason = pgtype.Text{String: *reason, Valid: true}
	}

	order, err := queries.CancelOrder(ctx, cancelParams)
	if err != nil {
		return nil, &ServiceError{Code: "CANCEL_FAILED", Message: "주문 취소에 실패했습니다.", Status: 500}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, &ServiceError{Code: "CANCEL_FAILED", Message: "주문 취소 커밋에 실패했습니다.", Status: 500}
	}

	// 재고 복구 (트랜잭션 외부)
	for _, item := range items {
		productID := uuid.UUID(item.ProductID.Bytes)
		product.IncreaseStock(ctx, productID, item.Quantity)
	}

	return toOrderResponse(&order, items, nil), nil
}

func toOrderResponse(order *db.OrdersOrder, items []db.OrdersOrderItem, shipping *ShippingAddress) *OrderResponse {
	resp := &OrderResponse{
		ID:          uuid.UUID(order.ID.Bytes),
		UserID:      uuid.UUID(order.UserID.Bytes),
		TotalAmount: order.TotalAmount,
		Status:      string(order.Status),
		CreatedAt:   order.CreatedAt.Time,
		UpdatedAt:   order.UpdatedAt.Time,
		Items:       make([]OrderItemResponse, 0, len(items)),
	}

	if order.CancelledAt.Valid {
		t := order.CancelledAt.Time
		resp.CancelledAt = &t
	}
	if order.CancelReason.Valid {
		resp.CancelReason = &order.CancelReason.String
	}

	// Shipping address from order
	if order.RecipientName.Valid {
		resp.ShippingAddress = &ShippingAddress{
			RecipientName: order.RecipientName.String,
			Phone:         order.Phone.String,
			Address:       order.Address.String,
			PostalCode:    order.PostalCode.String,
		}
		if order.AddressDetail.Valid {
			resp.ShippingAddress.AddressDetail = &order.AddressDetail.String
		}
	} else if shipping != nil {
		resp.ShippingAddress = shipping
	}

	for _, item := range items {
		itemResp := OrderItemResponse{
			ID:          uuid.UUID(item.ID.Bytes),
			ProductID:   uuid.UUID(item.ProductID.Bytes),
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.Subtotal,
		}
		if item.DealID.Valid {
			dealID := uuid.UUID(item.DealID.Bytes)
			itemResp.DealID = &dealID
		}
		resp.Items = append(resp.Items, itemResp)
	}

	return resp
}
