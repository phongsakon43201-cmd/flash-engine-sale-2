package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"flashsale-go/internal/domain"

	"github.com/redis/go-redis/v9"
)

// Lua script to safely decrement stock atomically without over-selling
const atomicDecrementLuaScript = `
local stockKey = KEYS[1]
local quantity = tonumber(ARGV[1])
local currentStock = tonumber(redis.call('GET', stockKey) or '0')

if currentStock >= quantity then
    redis.call('DECRBY', stockKey, quantity)
    return 1
else
    return 0
end
`

// Lua script to safely release distributed lock only if token matches
const releaseLockLuaScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
else
    return 0
end
`

type redisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) domain.CacheRepository {
	return &redisRepository{
		client: client,
	}
}

func (r *redisRepository) PrewarmStock(ctx context.Context, productID string, stock int) error {
	key := fmt.Sprintf("product:%s:stock", productID)
	err := r.client.Set(ctx, key, stock, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to prewarm stock in Redis: %w", err)
	}
	return nil
}

func (r *redisRepository) GetStock(ctx context.Context, productID string) (int, error) {
	key := fmt.Sprintf("product:%s:stock", productID)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	stock, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return stock, nil
}

func (r *redisRepository) DecrementStockAtomic(ctx context.Context, productID string, quantity int) (bool, error) {
	key := fmt.Sprintf("product:%s:stock", productID)
	res, err := r.client.Eval(ctx, atomicDecrementLuaScript, []string{key}, quantity).Result()
	if err != nil {
		return false, fmt.Errorf("redis atomic decrement script error: %w", err)
	}

	resultInt, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected script return type")
	}

	return resultInt == 1, nil
}

func (r *redisRepository) IncrementStock(ctx context.Context, productID string, quantity int) error {
	key := fmt.Sprintf("product:%s:stock", productID)
	return r.client.IncrBy(ctx, key, int64(quantity)).Err()
}

func (r *redisRepository) AcquireLock(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	success, err := r.client.SetNX(ctx, lockKey, value, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("redis distributed lock error: %w", err)
	}
	return success, nil
}

func (r *redisRepository) ReleaseLock(ctx context.Context, key string, value string) error {
	lockKey := fmt.Sprintf("lock:%s", key)
	return r.client.Eval(ctx, releaseLockLuaScript, []string{lockKey}, value).Err()
}
