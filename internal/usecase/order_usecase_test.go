package usecase_test

import (
	"context"
	"testing"
	"time"

	"flashsale-go/internal/domain"
	"flashsale-go/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Repositories
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	args := m.Called(ctx, product)
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *MockProductRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *MockProductRepository) ListProducts(ctx context.Context) ([]*domain.Product, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.Product), args.Error(1)
}

func (m *MockProductRepository) DecrementStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}

type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) PrewarmStock(ctx context.Context, productID string, stock int) error {
	args := m.Called(ctx, productID, stock)
	return args.Error(0)
}

func (m *MockCacheRepository) GetStock(ctx context.Context, productID string) (int, error) {
	args := m.Called(ctx, productID)
	return args.Int(0), args.Error(1)
}

func (m *MockCacheRepository) DecrementStockAtomic(ctx context.Context, productID string, quantity int) (bool, error) {
	args := m.Called(ctx, productID, quantity)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) AcquireLock(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, expiration)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) ReleaseLock(ctx context.Context, key string, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

type MockQueueRepository struct {
	mock.Mock
}

func (m *MockQueueRepository) PublishOrderEvent(ctx context.Context, event *domain.OrderEventPayload) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockQueueRepository) ReceiveOrderEvents(ctx context.Context, handler func(event *domain.OrderEventPayload) error) error {
	args := m.Called(ctx, handler)
	return args.Error(0)
}

// Test Cases
func TestCreateFlashSaleOrder_Success(t *testing.T) {
	ctx := context.Background()
	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockCacheRepo := new(MockCacheRepository)
	mockQueueRepo := new(MockQueueRepository)

	orderUsecase := usecase.NewOrderUsecase(mockOrderRepo, mockProductRepo, mockCacheRepo, mockQueueRepo)

	userID := "user-123"
	productID := "60d5ecb8b5c9c22b9c8b4567"
	dto := &domain.CreateOrderDTO{
		ProductID: productID,
		Quantity:  1,
	}

	// Expectations
	mockCacheRepo.On("AcquireLock", ctx, mock.Anything, "LOCKED", 10*time.Second).Return(true, nil)
	mockCacheRepo.On("DecrementStockAtomic", ctx, productID, 1).Return(true, nil)
	mockProductRepo.On("FindByID", ctx, productID).Return(&domain.Product{
		Price:       1000,
		IsFlashSale: true,
		FlashPrice:  199,
	}, nil)
	mockQueueRepo.On("PublishOrderEvent", ctx, mock.Anything).Return(nil)

	res, err := orderUsecase.CreateFlashSaleOrder(ctx, userID, dto)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, domain.OrderStatusPending, res.Status)
	assert.Contains(t, res.OrderID, "ORD-")
	mockCacheRepo.AssertExpectations(t)
	mockQueueRepo.AssertExpectations(t)
}

func TestCreateFlashSaleOrder_OutOfStock(t *testing.T) {
	ctx := context.Background()
	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockCacheRepo := new(MockCacheRepository)
	mockQueueRepo := new(MockQueueRepository)

	orderUsecase := usecase.NewOrderUsecase(mockOrderRepo, mockProductRepo, mockCacheRepo, mockQueueRepo)

	userID := "user-456"
	productID := "60d5ecb8b5c9c22b9c8b4567"
	dto := &domain.CreateOrderDTO{
		ProductID: productID,
		Quantity:  1,
	}

	mockCacheRepo.On("AcquireLock", ctx, mock.Anything, "LOCKED", 10*time.Second).Return(true, nil)
	// Return false for Atomic Decrement indicating 0 stock remaining in Redis
	mockCacheRepo.On("DecrementStockAtomic", ctx, productID, 1).Return(false, nil)

	res, err := orderUsecase.CreateFlashSaleOrder(ctx, userID, dto)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, "product out of stock", err.Error())
}
