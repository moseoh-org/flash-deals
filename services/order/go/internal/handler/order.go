package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/flash-deals/order/internal/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// Request/Response DTOs

type CreateOrderItemRequest struct {
	ProductID string  `json:"product_id"`
	DealID    *string `json:"deal_id,omitempty"`
	Quantity  int32   `json:"quantity"`
}

type ShippingAddressRequest struct {
	RecipientName string  `json:"recipient_name"`
	Phone         string  `json:"phone"`
	Address       string  `json:"address"`
	AddressDetail *string `json:"address_detail,omitempty"`
	PostalCode    string  `json:"postal_code"`
}

type CreateOrderRequest struct {
	Items           []CreateOrderItemRequest `json:"items"`
	ShippingAddress *ShippingAddressRequest  `json:"shipping_address,omitempty"`
}

type CancelOrderRequest struct {
	Reason *string `json:"reason,omitempty"`
}

type OrderItemResponse struct {
	ID          string  `json:"id"`
	ProductID   string  `json:"product_id"`
	DealID      *string `json:"deal_id,omitempty"`
	ProductName string  `json:"product_name"`
	Quantity    int32   `json:"quantity"`
	UnitPrice   int32   `json:"unit_price"`
	Subtotal    int32   `json:"subtotal"`
}

type ShippingAddressResponse struct {
	RecipientName string  `json:"recipient_name"`
	Phone         string  `json:"phone"`
	Address       string  `json:"address"`
	AddressDetail *string `json:"address_detail,omitempty"`
	PostalCode    string  `json:"postal_code"`
}

type OrderResponse struct {
	ID              string                   `json:"id"`
	UserID          string                   `json:"user_id"`
	Items           []OrderItemResponse      `json:"items"`
	TotalAmount     int32                    `json:"total_amount"`
	Status          string                   `json:"status"`
	ShippingAddress *ShippingAddressResponse `json:"shipping_address,omitempty"`
	CancelledAt     *time.Time               `json:"cancelled_at,omitempty"`
	CancelReason    *string                  `json:"cancel_reason,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

type OrderListResponse struct {
	Items []OrderResponse `json:"items"`
	Total int64           `json:"total"`
	Page  int32           `json:"page"`
	Size  int32           `json:"size"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Health Check

func (h *OrderHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "order-service",
	})
}

// Order Handlers

func (h *OrderHandler) CreateOrder(c echo.Context) error {
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: "X-User-ID header required"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER_ID", Message: "Invalid user ID format"})
	}

	var req CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid request body"})
	}

	if len(req.Items) == 0 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "At least one item is required"})
	}

	// Convert request to service types
	items := make([]service.OrderItemRequest, len(req.Items))
	for i, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_PRODUCT_ID", Message: "Invalid product ID format"})
		}
		items[i] = service.OrderItemRequest{
			ProductID: productID,
			Quantity:  item.Quantity,
		}
		if item.DealID != nil {
			dealID, err := uuid.Parse(*item.DealID)
			if err != nil {
				return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_DEAL_ID", Message: "Invalid deal ID format"})
			}
			items[i].DealID = &dealID
		}
	}

	var shipping *service.ShippingAddress
	if req.ShippingAddress != nil {
		shipping = &service.ShippingAddress{
			RecipientName: req.ShippingAddress.RecipientName,
			Phone:         req.ShippingAddress.Phone,
			Address:       req.ShippingAddress.Address,
			AddressDetail: req.ShippingAddress.AddressDetail,
			PostalCode:    req.ShippingAddress.PostalCode,
		}
	}

	order, err := h.svc.CreateOrder(c.Request().Context(), userID, items, shipping)
	if err != nil {
		if svcErr, ok := err.(*service.ServiceError); ok {
			return c.JSON(svcErr.Status, ErrorResponse{Error: svcErr.Code, Message: svcErr.Message})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "INTERNAL_ERROR", Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, toOrderResponse(order))
}

func (h *OrderHandler) GetOrder(c echo.Context) error {
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: "X-User-ID header required"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER_ID", Message: "Invalid user ID format"})
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_ORDER_ID", Message: "Invalid order ID format"})
	}

	order, err := h.svc.GetOrder(c.Request().Context(), orderID, userID)
	if err != nil {
		if svcErr, ok := err.(*service.ServiceError); ok {
			return c.JSON(svcErr.Status, ErrorResponse{Error: svcErr.Code, Message: svcErr.Message})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "INTERNAL_ERROR", Message: err.Error()})
	}

	return c.JSON(http.StatusOK, toOrderResponse(order))
}

func (h *OrderHandler) ListOrders(c echo.Context) error {
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: "X-User-ID header required"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER_ID", Message: "Invalid user ID format"})
	}

	page := int32(1)
	size := int32(20)
	var status *string

	if p := c.QueryParam("page"); p != "" {
		if pageInt, err := strconv.Atoi(p); err == nil && pageInt > 0 {
			page = int32(pageInt)
		}
	}
	if s := c.QueryParam("size"); s != "" {
		if sizeInt, err := strconv.Atoi(s); err == nil && sizeInt > 0 && sizeInt <= 100 {
			size = int32(sizeInt)
		}
	}
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}

	orders, total, err := h.svc.ListOrders(c.Request().Context(), userID, page, size, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "INTERNAL_ERROR", Message: err.Error()})
	}

	items := make([]OrderResponse, len(orders))
	for i, order := range orders {
		items[i] = *toOrderResponse(&order)
	}

	return c.JSON(http.StatusOK, OrderListResponse{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

func (h *OrderHandler) CancelOrder(c echo.Context) error {
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: "X-User-ID header required"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER_ID", Message: "Invalid user ID format"})
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_ORDER_ID", Message: "Invalid order ID format"})
	}

	var req CancelOrderRequest
	c.Bind(&req) // Ignore bind error - reason is optional

	order, err := h.svc.CancelOrder(c.Request().Context(), orderID, userID, req.Reason)
	if err != nil {
		if svcErr, ok := err.(*service.ServiceError); ok {
			return c.JSON(svcErr.Status, ErrorResponse{Error: svcErr.Code, Message: svcErr.Message})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "INTERNAL_ERROR", Message: err.Error()})
	}

	return c.JSON(http.StatusOK, toOrderResponse(order))
}

// Helpers

func toOrderResponse(order *service.OrderResponse) *OrderResponse {
	resp := &OrderResponse{
		ID:          order.ID.String(),
		UserID:      order.UserID.String(),
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
		Items:       make([]OrderItemResponse, len(order.Items)),
	}

	if order.CancelledAt != nil {
		resp.CancelledAt = order.CancelledAt
	}
	if order.CancelReason != nil {
		resp.CancelReason = order.CancelReason
	}
	if order.ShippingAddress != nil {
		resp.ShippingAddress = &ShippingAddressResponse{
			RecipientName: order.ShippingAddress.RecipientName,
			Phone:         order.ShippingAddress.Phone,
			Address:       order.ShippingAddress.Address,
			AddressDetail: order.ShippingAddress.AddressDetail,
			PostalCode:    order.ShippingAddress.PostalCode,
		}
	}

	for i, item := range order.Items {
		resp.Items[i] = OrderItemResponse{
			ID:          item.ID.String(),
			ProductID:   item.ProductID.String(),
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.Subtotal,
		}
		if item.DealID != nil {
			dealIDStr := item.DealID.String()
			resp.Items[i].DealID = &dealIDStr
		}
	}

	return resp
}
