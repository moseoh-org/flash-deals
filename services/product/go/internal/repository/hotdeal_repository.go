package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Lua Script for atomic stock decrement
const decrementStockScript = `
local stock = redis.call('GET', KEYS[1])
if stock == false then
    return -1  -- key not found
end
local currentStock = tonumber(stock)
local delta = tonumber(ARGV[1])
if currentStock < delta then
    return -2  -- insufficient stock
end
local newStock = currentStock - delta
redis.call('SET', KEYS[1], newStock)
return newStock
`

// HotdealRepository handles Redis-based stock management for hotdeals
type HotdealRepository struct {
	redis           *redis.Client
	decrementScript *redis.Script
	enabled         bool
}

// NewHotdealRepository creates a new HotdealRepository
func NewHotdealRepository(redisClient *redis.Client, enabled bool) *HotdealRepository {
	return &HotdealRepository{
		redis:           redisClient,
		decrementScript: redis.NewScript(decrementStockScript),
		enabled:         enabled,
	}
}

// IsEnabled returns whether Redis hotdeal stock is enabled
func (r *HotdealRepository) IsEnabled() bool {
	return r.enabled && r.redis != nil
}

// stockKey returns the Redis key for product stock
func (r *HotdealRepository) stockKey(productID uuid.UUID) string {
	return fmt.Sprintf("hotdeal:stock:%s", productID.String())
}

// LoadStock loads stock from DB to Redis for a hotdeal product
func (r *HotdealRepository) LoadStock(ctx context.Context, productID uuid.UUID, stock int32) error {
	if !r.IsEnabled() {
		return nil // No-op when disabled
	}

	key := r.stockKey(productID)
	if err := r.redis.Set(ctx, key, stock, 0).Err(); err != nil {
		return fmt.Errorf("failed to load stock to Redis: %w", err)
	}

	log.Printf("[Hotdeal] Loaded stock to Redis: product=%s, stock=%d", productID, stock)
	return nil
}

// UnloadStock removes stock from Redis and returns the current stock
func (r *HotdealRepository) UnloadStock(ctx context.Context, productID uuid.UUID) (int32, error) {
	if !r.IsEnabled() {
		return 0, nil // No-op when disabled
	}

	key := r.stockKey(productID)

	// Get current stock before deleting
	stock, err := r.redis.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // Key doesn't exist
		}
		return 0, fmt.Errorf("failed to get stock from Redis: %w", err)
	}

	// Delete the key
	if err := r.redis.Del(ctx, key).Err(); err != nil {
		return 0, fmt.Errorf("failed to delete stock from Redis: %w", err)
	}

	log.Printf("[Hotdeal] Unloaded stock from Redis: product=%s, remaining=%d", productID, stock)
	return int32(stock), nil
}

// DecrementStock atomically decrements stock using Lua script
// Returns new stock on success, or error if insufficient stock
func (r *HotdealRepository) DecrementStock(ctx context.Context, productID uuid.UUID, quantity int32) (int32, error) {
	if !r.IsEnabled() {
		return 0, fmt.Errorf("hotdeal stock redis not enabled")
	}

	key := r.stockKey(productID)
	result, err := r.decrementScript.Run(ctx, r.redis, []string{key}, quantity).Int()
	if err != nil {
		return 0, fmt.Errorf("failed to run decrement script: %w", err)
	}

	switch result {
	case -1:
		return 0, fmt.Errorf("hotdeal stock not found in Redis")
	case -2:
		return 0, fmt.Errorf("insufficient stock")
	default:
		return int32(result), nil
	}
}

// IncrementStock increments stock (for order cancellation)
func (r *HotdealRepository) IncrementStock(ctx context.Context, productID uuid.UUID, quantity int32) (int32, error) {
	if !r.IsEnabled() {
		return 0, fmt.Errorf("hotdeal stock redis not enabled")
	}

	key := r.stockKey(productID)
	result, err := r.redis.IncrBy(ctx, key, int64(quantity)).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment stock: %w", err)
	}

	return int32(result), nil
}

// GetStock returns current stock in Redis
func (r *HotdealRepository) GetStock(ctx context.Context, productID uuid.UUID) (int32, error) {
	if !r.IsEnabled() {
		return 0, fmt.Errorf("hotdeal stock redis not enabled")
	}

	key := r.stockKey(productID)
	stock, err := r.redis.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("hotdeal stock not found")
		}
		return 0, fmt.Errorf("failed to get stock: %w", err)
	}

	return int32(stock), nil
}

// HasStock checks if product has hotdeal stock in Redis
func (r *HotdealRepository) HasStock(ctx context.Context, productID uuid.UUID) bool {
	if !r.IsEnabled() {
		return false
	}

	key := r.stockKey(productID)
	exists, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		return false
	}

	return exists > 0
}
