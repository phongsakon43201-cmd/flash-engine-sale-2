package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"flashsale-go/internal/domain"
	"flashsale-go/pkg/metrics"

	"github.com/google/uuid"
)

type OrderUsecase interface {
	CreateFlashSaleOrder(ctx context.Context, userID string, dto *domain.CreateOrderDTO) (*domain.OrderResponseDTO, error)
	ProcessOrderFromQueue(ctx context.Context, event *domain.OrderEventPayload) error
	GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error)
}

type orderUsecase struct {
	orderRepo   domain.OrderRepository
	productRepo domain.ProductRepository
	cacheRepo   domain.CacheRepository
	queueRepo   domain.QueueRepository
}

func NewOrderUsecase(
	orderRepo domain.OrderRepository,
	productRepo domain.ProductRepository,
	cacheRepo domain.CacheRepository,
	queueRepo domain.QueueRepository,
) OrderUsecase {
	return &orderUsecase{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		cacheRepo:   cacheRepo,
		queueRepo:   queueRepo,
	}
}

func (u *orderUsecase) CreateFlashSaleOrder(ctx context.Context, userID string, dto *domain.CreateOrderDTO) (*domain.OrderResponseDTO, error) {
	if dto.ProductID == "" {
		return nil, errors.New("product ID is required")
	}
	if dto.Quantity <= 0 {
		dto.Quantity = 1
	}

	// 1. Prevent duplicate purchase per user for the flash sale item (10s TTL Lock)
	userLockKey := fmt.Sprintf("user:%s:product:%s:lock", userID, dto.ProductID)
	acquired, err := u.cacheRepo.AcquireLock(ctx, userLockKey, "LOCKED", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed checking user lock: %w", err)
	}
	if !acquired {
		return nil, errors.New("order processing in progress, please do not double click")
	}

	// 2. Atomic Stock Decrement via Redis Lua Script (Zero Race Condition / Zero Over-selling)
	decremented, err := u.cacheRepo.DecrementStockAtomic(ctx, dto.ProductID, dto.Quantity)
	if err != nil {
		metrics.OrdersPlacedTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("error during atomic stock check: %w", err)
	}
	if !decremented {
		metrics.OrdersPlacedTotal.WithLabelValues("out_of_stock").Inc()
		return nil, errors.New("product out of stock")
	}

	// 3. Get product price details (or use cached price)
	product, err := u.productRepo.FindByID(ctx, dto.ProductID)
	price := 0.0
	if err == nil && product != nil {
		if product.IsFlashSale && product.FlashPrice > 0 {
			price = product.FlashPrice
		} else {
			price = product.Price
		}
	}

	// 4. Construct Order Event Payload & Publish to AWS SQS Queue
	orderID := "ORD-" + uuid.New().String()
	event := &domain.OrderEventPayload{
		OrderID:   orderID,
		UserID:    userID,
		ProductID: dto.ProductID,
		Quantity:  dto.Quantity,
		Price:     price,
		Timestamp: time.Now(),
	}

	if err := u.queueRepo.PublishOrderEvent(ctx, event); err != nil {
		log.Printf("CRITICAL: Failed to publish order event to SQS [OrderID: %s]: %v", orderID, err)
		// Return stock to Redis in emergency failure case (increment back by quantity)
		_ = u.cacheRepo.IncrementStock(ctx, dto.ProductID, dto.Quantity)
		metrics.OrdersPlacedTotal.WithLabelValues("queue_failed").Inc()
		return nil, fmt.Errorf("failed to process order queue: %w", err)
	}

	metrics.OrdersPlacedTotal.WithLabelValues("accepted").Inc()

	// 5. Return HTTP 202 Accepted response immediately (~15-30ms latency)
	return &domain.OrderResponseDTO{
		OrderID:   orderID,
		Status:    domain.OrderStatusPending,
		Message:   "Order received and is being processed in background queue",
		Timestamp: time.Now(),
	}, nil
}

// ProcessOrderFromQueue executes asynchronously inside worker engine
func (u *orderUsecase) ProcessOrderFromQueue(ctx context.Context, event *domain.OrderEventPayload) error {
	log.Printf("[Worker] Processing order %s for user %s, product %s", event.OrderID, event.UserID, event.ProductID)

	// Idempotency check: Ensure order is not processed multiple times by SQS At-Least-Once delivery
	idempotencyKey := fmt.Sprintf("idempotency:order:%s", event.OrderID)
	acquired, err := u.cacheRepo.AcquireLock(ctx, idempotencyKey, "PROCESSED", 24*time.Hour)
	if err == nil && !acquired {
		log.Printf("[Worker Idempotency] Order %s was already processed. Skipping duplicate execution.", event.OrderID)
		metrics.OrdersPlacedTotal.WithLabelValues("duplicate_skipped").Inc()
		return nil
	}

	// 1. Create order record in MongoDB
	order := &domain.Order{
		OrderID:    event.OrderID,
		UserID:     event.UserID,
		ProductID:  event.ProductID,
		Quantity:   event.Quantity,
		TotalPrice: event.Price * float64(event.Quantity),
		Status:     domain.OrderStatusCompleted,
		CreatedAt:  event.Timestamp,
		UpdatedAt:  time.Now(),
	}

	if err := u.orderRepo.CreateOrder(ctx, order); err != nil {
		log.Printf("[Worker Error] Failed creating order in DB [OrderID: %s]: %v", event.OrderID, err)
		metrics.OrdersPlacedTotal.WithLabelValues("db_error").Inc()
		return err
	}

	// 2. Decrement persistent MongoDB database stock
	if err := u.productRepo.DecrementStock(ctx, event.ProductID, event.Quantity); err != nil {
		log.Printf("[Worker Warning] Stock decrement in DB failed [ProductID: %s]: %v", event.ProductID, err)
		_ = u.orderRepo.UpdateOrderStatus(ctx, event.OrderID, domain.OrderStatusFailed)
		metrics.OrdersPlacedTotal.WithLabelValues("stock_db_failed").Inc()
		return err
	}

	metrics.OrdersPlacedTotal.WithLabelValues("completed").Inc()
	log.Printf("[Worker Success] Order %s successfully persisted to MongoDB", event.OrderID)
	return nil
}

func (u *orderUsecase) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	return u.orderRepo.FindByOrderID(ctx, orderID)
}
