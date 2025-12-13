package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	OrderQueueKey = "order:queue"
)

// OrderRequest represents a queued order request
type OrderRequest struct {
	OrderID         uuid.UUID       `json:"order_id"`
	UserID          uuid.UUID       `json:"user_id"`
	Items           []OrderItemReq  `json:"items"`
	ShippingAddress *ShippingAddr   `json:"shipping_address,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type OrderItemReq struct {
	ProductID   uuid.UUID  `json:"product_id"`
	DealID      *uuid.UUID `json:"deal_id,omitempty"`
	Quantity    int32      `json:"quantity"`
	ProductName string     `json:"product_name"`
	UnitPrice   int32      `json:"unit_price"`
}

type ShippingAddr struct {
	RecipientName string  `json:"recipient_name"`
	Phone         string  `json:"phone"`
	Address       string  `json:"address"`
	AddressDetail *string `json:"address_detail,omitempty"`
	PostalCode    string  `json:"postal_code"`
}

// OrderQueue handles async order processing via Redis List
type OrderQueue struct {
	client *redis.Client
}

// NewOrderQueue creates a new OrderQueue
func NewOrderQueue(client *redis.Client) *OrderQueue {
	return &OrderQueue{client: client}
}

// Enqueue adds an order request to the queue
func (q *OrderQueue) Enqueue(ctx context.Context, req *OrderRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, OrderQueueKey, data).Err()
}

// Dequeue removes and returns an order request from the queue (blocking)
func (q *OrderQueue) Dequeue(ctx context.Context, timeout time.Duration) (*OrderRequest, error) {
	result, err := q.client.BRPop(ctx, timeout, OrderQueueKey).Result()
	if err != nil {
		return nil, err
	}

	// result[0] is key, result[1] is value
	var req OrderRequest
	if err := json.Unmarshal([]byte(result[1]), &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// QueueLength returns the current queue length
func (q *OrderQueue) QueueLength(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, OrderQueueKey).Result()
}

// LogQueueStats periodically logs queue statistics
func (q *OrderQueue) LogQueueStats(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			length, err := q.QueueLength(ctx)
			if err != nil {
				log.Printf("[OrderQueue] Failed to get queue length: %v", err)
				continue
			}
			if length > 0 {
				log.Printf("[OrderQueue] Queue length: %d", length)
			}
		}
	}
}
