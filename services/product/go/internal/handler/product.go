package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/flash-deals/product/internal/db"
	"github.com/flash-deals/product/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type ProductHandler struct {
	repo repository.ProductRepository
}

func NewProductHandler(repo repository.ProductRepository) *ProductHandler {
	return &ProductHandler{repo: repo}
}

// Request/Response DTOs

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Price       int32   `json:"price"`
	Stock       int32   `json:"stock"`
	Category    *string `json:"category,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type UpdateProductRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Price       *int32  `json:"price,omitempty"`
	Category    *string `json:"category,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type UpdateStockRequest struct {
	StockDelta int32 `json:"stock_delta"`
}

type ProductResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Price       int32     `json:"price"`
	Stock       int32     `json:"stock"`
	Category    *string   `json:"category,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProductListResponse struct {
	Items []ProductResponse `json:"items"`
	Total int64             `json:"total"`
	Limit int32             `json:"limit"`
	Skip  int32             `json:"skip"`
}

type StockResponse struct {
	ID        string    `json:"id"`
	Stock     int32     `json:"stock"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateDealRequest struct {
	ProductID      string    `json:"product_id"`
	DealPrice      int32     `json:"deal_price"`
	DealStock      int32     `json:"deal_stock"`
	RemainingStock int32     `json:"remaining_stock"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
}

type DealResponse struct {
	ID             string           `json:"id"`
	ProductID      string           `json:"product_id"`
	DealPrice      int32            `json:"deal_price"`
	DealStock      int32            `json:"deal_stock"`
	RemainingStock int32            `json:"remaining_stock"`
	StartsAt       time.Time        `json:"starts_at"`
	EndsAt         time.Time        `json:"ends_at"`
	CreatedAt      time.Time        `json:"created_at"`
	Product        *ProductResponse `json:"product,omitempty"`
}

type DealListResponse struct {
	Items []DealResponse `json:"items"`
	Total int64          `json:"total"`
	Limit int32          `json:"limit"`
	Skip  int32          `json:"skip"`
}

// Helpers

func toProductResponse(p *db.ProductProduct) ProductResponse {
	resp := ProductResponse{
		ID:        uuidToString(p.ID),
		Name:      p.Name,
		Price:     p.Price,
		Stock:     p.Stock,
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
	}
	if p.Description.Valid {
		resp.Description = &p.Description.String
	}
	if p.Category.Valid {
		resp.Category = &p.Category.String
	}
	if p.ImageUrl.Valid {
		resp.ImageURL = &p.ImageUrl.String
	}
	return resp
}

func uuidToString(id pgtype.UUID) string {
	u := uuid.UUID(id.Bytes)
	return u.String()
}

// Health Check

func (h *ProductHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}

// Product Handlers

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	var req CreateProductRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	params := db.CreateProductParams{
		Name:  req.Name,
		Price: req.Price,
		Stock: req.Stock,
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Category != nil {
		params.Category = pgtype.Text{String: *req.Category, Valid: true}
	}
	if req.ImageURL != nil {
		params.ImageUrl = pgtype.Text{String: *req.ImageURL, Valid: true}
	}

	product, err := h.repo.CreateProduct(c.Request().Context(), params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	resp := toProductResponse(product)
	return c.JSON(http.StatusCreated, resp)
}

func (h *ProductHandler) GetProduct(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	product, err := h.repo.GetProductByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	resp := toProductResponse(product)
	return c.JSON(http.StatusOK, resp)
}

func (h *ProductHandler) ListProducts(c echo.Context) error {
	ctx := c.Request().Context()

	limit := int32(20)
	offset := int32(0)
	category := c.QueryParam("category")

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = int32(parsed)
		}
	}
	if s := c.QueryParam("skip"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil {
			offset = int32(parsed)
		}
	}

	var products []db.ProductProduct
	var total int64
	var err error

	if category != "" {
		products, err = h.repo.ListProductsByCategory(ctx, category, limit, offset)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		total, _ = h.repo.CountProductsByCategory(ctx, category)
	} else {
		products, err = h.repo.ListProducts(ctx, limit, offset)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		total, _ = h.repo.CountProducts(ctx)
	}

	items := make([]ProductResponse, len(products))
	for i, p := range products {
		items[i] = toProductResponse(&p)
	}

	return c.JSON(http.StatusOK, ProductListResponse{
		Items: items,
		Total: total,
		Limit: limit,
		Skip:  offset,
	})
}

func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	var req UpdateProductRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	params := db.UpdateProductParams{
		ID: pgtype.UUID{Bytes: id, Valid: true},
	}
	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Price != nil {
		params.Price = *req.Price
	}
	if req.Category != nil {
		params.Category = pgtype.Text{String: *req.Category, Valid: true}
	}
	if req.ImageURL != nil {
		params.ImageUrl = pgtype.Text{String: *req.ImageURL, Valid: true}
	}

	product, err := h.repo.UpdateProduct(c.Request().Context(), params)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	resp := toProductResponse(product)
	return c.JSON(http.StatusOK, resp)
}

// Stock Handlers

func (h *ProductHandler) GetStock(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	product, err := h.repo.GetProductByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	return c.JSON(http.StatusOK, StockResponse{
		ID:        uuidToString(product.ID),
		Stock:     product.Stock,
		UpdatedAt: product.UpdatedAt.Time,
	})
}

func (h *ProductHandler) UpdateStock(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	var req UpdateStockRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	ctx := c.Request().Context()

	// Get current stock
	current, err := h.repo.GetStockForUpdate(ctx, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	newStock := current.Stock + req.StockDelta
	if newStock < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Insufficient stock"})
	}

	row, err := h.repo.UpdateStock(ctx, id, newStock)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, StockResponse{
		ID:        uuidToString(row.ID),
		Stock:     row.Stock,
		UpdatedAt: row.UpdatedAt.Time,
	})
}

// Deal Handlers

func (h *ProductHandler) CreateDeal(c echo.Context) error {
	var req CreateDealRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	params := db.CreateDealParams{
		ProductID:      pgtype.UUID{Bytes: productID, Valid: true},
		DealPrice:      req.DealPrice,
		DealStock:      req.DealStock,
		RemainingStock: req.RemainingStock,
		StartsAt:       pgtype.Timestamptz{Time: req.StartsAt, Valid: true},
		EndsAt:         pgtype.Timestamptz{Time: req.EndsAt, Valid: true},
	}

	deal, err := h.repo.CreateDeal(c.Request().Context(), params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, DealResponse{
		ID:             uuidToString(deal.ID),
		ProductID:      uuidToString(deal.ProductID),
		DealPrice:      deal.DealPrice,
		DealStock:      deal.DealStock,
		RemainingStock: deal.RemainingStock,
		StartsAt:       deal.StartsAt.Time,
		EndsAt:         deal.EndsAt.Time,
		CreatedAt:      deal.CreatedAt.Time,
	})
}

func (h *ProductHandler) GetDeal(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid deal ID"})
	}

	row, err := h.repo.GetDealByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Deal not found"})
	}

	resp := DealResponse{
		ID:             uuidToString(row.ID),
		ProductID:      uuidToString(row.ProductID),
		DealPrice:      row.DealPrice,
		DealStock:      row.DealStock,
		RemainingStock: row.RemainingStock,
		StartsAt:       row.StartsAt.Time,
		EndsAt:         row.EndsAt.Time,
		CreatedAt:      row.CreatedAt.Time,
		Product: &ProductResponse{
			ID:        uuidToString(row.PID),
			Name:      row.PName,
			Price:     row.PPrice,
			Stock:     row.PStock,
			CreatedAt: row.PCreatedAt.Time,
			UpdatedAt: row.PUpdatedAt.Time,
		},
	}

	if row.PDescription.Valid {
		resp.Product.Description = &row.PDescription.String
	}
	if row.PCategory.Valid {
		resp.Product.Category = &row.PCategory.String
	}
	if row.PImageUrl.Valid {
		resp.Product.ImageURL = &row.PImageUrl.String
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *ProductHandler) ListActiveDeals(c echo.Context) error {
	ctx := c.Request().Context()

	limit := int32(20)
	offset := int32(0)

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = int32(parsed)
		}
	}
	if s := c.QueryParam("skip"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil {
			offset = int32(parsed)
		}
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	rows, err := h.repo.ListActiveDeals(ctx, now, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	total, _ := h.repo.CountActiveDeals(ctx, now)

	items := make([]DealResponse, len(rows))
	for i, row := range rows {
		resp := DealResponse{
			ID:             uuidToString(row.ID),
			ProductID:      uuidToString(row.ProductID),
			DealPrice:      row.DealPrice,
			DealStock:      row.DealStock,
			RemainingStock: row.RemainingStock,
			StartsAt:       row.StartsAt.Time,
			EndsAt:         row.EndsAt.Time,
			CreatedAt:      row.CreatedAt.Time,
			Product: &ProductResponse{
				ID:        uuidToString(row.PID),
				Name:      row.PName,
				Price:     row.PPrice,
				Stock:     row.PStock,
				CreatedAt: row.PCreatedAt.Time,
				UpdatedAt: row.PUpdatedAt.Time,
			},
		}
		if row.PDescription.Valid {
			resp.Product.Description = &row.PDescription.String
		}
		if row.PCategory.Valid {
			resp.Product.Category = &row.PCategory.String
		}
		if row.PImageUrl.Valid {
			resp.Product.ImageURL = &row.PImageUrl.String
		}
		items[i] = resp
	}

	return c.JSON(http.StatusOK, DealListResponse{
		Items: items,
		Total: total,
		Limit: limit,
		Skip:  offset,
	})
}
