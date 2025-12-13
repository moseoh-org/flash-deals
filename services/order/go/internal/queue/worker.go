package queue

import (
	"context"
	"log"
	"time"

	"github.com/flash-deals/order/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// OrderWorker processes orders from the queue
type OrderWorker struct {
	queue *OrderQueue
	pool  *pgxpool.Pool
}

// NewOrderWorker creates a new OrderWorker
func NewOrderWorker(redisClient *redis.Client, pool *pgxpool.Pool) *OrderWorker {
	return &OrderWorker{
		queue: NewOrderQueue(redisClient),
		pool:  pool,
	}
}

// Start begins processing orders from the queue
func (w *OrderWorker) Start(ctx context.Context) {
	log.Println("[OrderWorker] Worker started - processing orders from Redis queue")

	// Start queue stats logging in background
	go w.queue.LogQueueStats(ctx, 5*time.Second)

	for {
		select {
		case <-ctx.Done():
			log.Println("[OrderWorker] Worker shutting down")
			return
		default:
			// Blocking wait for order (5 second timeout to check context)
			req, err := w.queue.Dequeue(ctx, 5*time.Second)
			if err != nil {
				if err == redis.Nil {
					// Timeout, no items in queue
					continue
				}
				if ctx.Err() != nil {
					// Context cancelled
					return
				}
				log.Printf("[OrderWorker] Dequeue error: %v", err)
				continue
			}

			// Process the order
			if err := w.processOrder(ctx, req); err != nil {
				log.Printf("[OrderWorker] Failed to process order %s: %v", req.OrderID, err)
				// TODO: Add to dead letter queue or retry logic
				continue
			}
		}
	}
}

// processOrder saves the order to the database
func (w *OrderWorker) processOrder(ctx context.Context, req *OrderRequest) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)

	// Calculate total amount
	var totalAmount int32
	for _, item := range req.Items {
		totalAmount += item.UnitPrice * item.Quantity
	}

	// Create order with pre-generated ID
	createParams := db.CreateOrderWithIDParams{
		ID:          pgtype.UUID{Bytes: req.OrderID, Valid: true},
		UserID:      pgtype.UUID{Bytes: req.UserID, Valid: true},
		TotalAmount: totalAmount,
		Status:      db.OrdersOrderStatusConfirmed, // Directly confirmed since stock already deducted
	}

	if req.ShippingAddress != nil {
		createParams.RecipientName = pgtype.Text{String: req.ShippingAddress.RecipientName, Valid: true}
		createParams.Phone = pgtype.Text{String: req.ShippingAddress.Phone, Valid: true}
		createParams.Address = pgtype.Text{String: req.ShippingAddress.Address, Valid: true}
		if req.ShippingAddress.AddressDetail != nil {
			createParams.AddressDetail = pgtype.Text{String: *req.ShippingAddress.AddressDetail, Valid: true}
		}
		createParams.PostalCode = pgtype.Text{String: req.ShippingAddress.PostalCode, Valid: true}
	}

	order, err := queries.CreateOrderWithID(ctx, createParams)
	if err != nil {
		return err
	}

	// Create order items
	for _, item := range req.Items {
		itemParams := db.CreateOrderItemParams{
			OrderID:     order.ID,
			ProductID:   pgtype.UUID{Bytes: item.ProductID, Valid: true},
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.UnitPrice * item.Quantity,
		}
		if item.DealID != nil {
			itemParams.DealID = pgtype.UUID{Bytes: *item.DealID, Valid: true}
		}

		if _, err := queries.CreateOrderItem(ctx, itemParams); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetQueue returns the underlying queue for direct access
func (w *OrderWorker) GetQueue() *OrderQueue {
	return w.queue
}
