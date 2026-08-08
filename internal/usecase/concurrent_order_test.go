package usecase_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"flashsale-go/internal/domain"
	"flashsale-go/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ThreadSafeMockCache simulates real Redis atomic Lua stock decrement and lock behavior
type ThreadSafeMockCache struct {
	stock int64
	locks sync.Map
}

func NewThreadSafeMockCache(initialStock int64) *ThreadSafeMockCache {
	return &ThreadSafeMockCache{
		stock: initialStock,
	}
}

func (m *ThreadSafeMockCache) PrewarmStock(ctx context.Context, productID string, stock int) error {
	atomic.StoreInt64(&m.stock, int64(stock))
	return nil
}

func (m *ThreadSafeMockCache) GetStock(ctx context.Context, productID string) (int, error) {
	return int(atomic.LoadInt64(&m.stock)), nil
}

func (m *ThreadSafeMockCache) DecrementStockAtomic(ctx context.Context, productID string, quantity int) (bool, error) {
	for {
		current := atomic.LoadInt64(&m.stock)
		if current < int64(quantity) {
			return false, nil
		}
		if atomic.CompareAndSwapInt64(&m.stock, current, current-int64(quantity)) {
			return true, nil
		}
	}
}

func (m *ThreadSafeMockCache) IncrementStock(ctx context.Context, productID string, quantity int) error {
	atomic.AddInt64(&m.stock, int64(quantity))
	return nil
}

func (m *ThreadSafeMockCache) AcquireLock(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	_, loaded := m.locks.LoadOrStore(key, value)
	return !loaded, nil
}

func (m *ThreadSafeMockCache) ReleaseLock(ctx context.Context, key string, value string) error {
	m.locks.Delete(key)
	return nil
}

func (m *ThreadSafeMockCache) PublishEvent(ctx context.Context, channel string, message string) error {
	return nil
}

func (m *ThreadSafeMockCache) SubscribeEvent(ctx context.Context, channel string) (<-chan string, func(), error) {
	ch := make(chan string)
	close(ch)
	return ch, func() {}, nil
}

func TestCreateFlashSaleOrder_HighConcurrency_ZeroOverselling(t *testing.T) {
	ctx := context.Background()
	const initialStock = 10
	const concurrentUsers = 100

	threadSafeCache := NewThreadSafeMockCache(initialStock)
	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockQueueRepo := new(MockQueueRepository)

	prodID := primitive.NewObjectID()
	prodHex := prodID.Hex()

	// Mock product repo find product
	mockProductRepo.On("FindByID", ctx, prodHex).Return(&domain.Product{
		ID:          prodID,
		Price:       999.0,
		IsFlashSale: true,
		FlashPrice:  99.0,
	}, nil)

	// Mock queue publish
	mockQueueRepo.On("PublishOrderEvent", ctx, mock.Anything).Return(nil)

	orderUsecase := usecase.NewOrderUsecase(mockOrderRepo, mockProductRepo, threadSafeCache, mockQueueRepo)

	var wg sync.WaitGroup
	var successCount int64
	var outOfStockCount int64
	var doubleClickCount int64

	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()

			userID := fmt.Sprintf("user-%d", userIndex)
			dto := &domain.CreateOrderDTO{
				ProductID: prodHex,
				Quantity:  1,
			}

			res, err := orderUsecase.CreateFlashSaleOrder(ctx, userID, dto)
			if err == nil && res != nil {
				atomic.AddInt64(&successCount, 1)
			} else if err != nil {
				if err.Error() == "product out of stock" {
					atomic.AddInt64(&outOfStockCount, 1)
				} else if err.Error() == "order processing in progress, please do not double click" {
					atomic.AddInt64(&doubleClickCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	remainingStock, _ := threadSafeCache.GetStock(ctx, prodHex)

	t.Logf("Concurrency Test Results:")
	t.Logf("  Initial Stock: %d", initialStock)
	t.Logf("  Concurrent Requests: %d", concurrentUsers)
	t.Logf("  Successful Orders: %d", successCount)
	t.Logf("  Out of Stock Rejections: %d", outOfStockCount)
	t.Logf("  Remaining Stock in Cache: %d", remainingStock)

	// Assertions for zero overselling
	assert.Equal(t, int64(initialStock), successCount, "Successful orders must exactly equal initial stock count")
	assert.Equal(t, 0, remainingStock, "Stock remaining must be zero (no negative stock)")
	assert.Equal(t, int64(concurrentUsers-initialStock), outOfStockCount+doubleClickCount, "Remaining requests must be rejected safely")
}
