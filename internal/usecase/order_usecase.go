package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"flashsale-go/internal/domain"
	"flashsale-go/pkg/metrics"
	"flashsale-go/pkg/tracer"

	"github.com/google/uuid"
)

var (
	ErrInvalidOrder    = errors.New("invalid order request")
	ErrInvalidQuantity = errors.New("quantity must be between 1 and 10")
	ErrNotFlashSale    = errors.New("product is not available for flash sale")
	ErrOutOfStock      = errors.New("product out of stock")
	ErrDuplicateOrder  = errors.New("order processing in progress, please do not double click")
)

type OrderUsecase interface {
	CreateFlashSaleOrder(ctx context.Context, userID string, dto *domain.CreateOrderDTO) (*domain.OrderResponseDTO, error)
	ProcessOrderFromQueue(ctx context.Context, event *domain.OrderEventPayload) error
	GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error)
	SubscribeOrderStatusStream(ctx context.Context, orderID string) (<-chan string, func(), error)
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
	if dto == nil || userID == "" || dto.ProductID == "" {
		return nil, ErrInvalidOrder
	}
	if dto.Quantity < 1 || dto.Quantity > 10 {
		return nil, ErrInvalidQuantity
	}

	// Load and validate the product before reserving inventory. Never enqueue a zero-price order.
	product, err := u.productRepo.FindByID(ctx, dto.ProductID)
	if err != nil {
		return nil, fmt.Errorf("load product: %w", err)
	}
	if product == nil || !product.IsFlashSale || product.FlashPrice <= 0 {
		return nil, ErrNotFlashSale
	}

	// Prevent accidental double clicks while the first request is being accepted.
	userLockKey := fmt.Sprintf("user:%s:product:%s:lock", userID, dto.ProductID)
	lockToken := uuid.NewString()
	acquired, err := u.cacheRepo.AcquireLock(ctx, userLockKey, lockToken, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed checking user lock: %w", err)
	}
	if !acquired {
		return nil, ErrDuplicateOrder
	}
	keepLock := false
	defer func() {
		if !keepLock {
			_ = u.cacheRepo.ReleaseLock(context.Background(), userLockKey, lockToken)
		}
	}()

	// Reserve inventory atomically in Redis.
	decremented, err := u.cacheRepo.DecrementStockAtomic(ctx, dto.ProductID, dto.Quantity)
	if err != nil {
		metrics.OrdersPlacedTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("error during atomic stock check: %w", err)
	}
	if !decremented {
		metrics.OrdersPlacedTotal.WithLabelValues("out_of_stock").Inc()
		return nil, ErrOutOfStock
	}

	// Construct the immutable order event and publish it to SQS.
	orderID := "ORD-" + uuid.New().String()
	traceID := tracer.FromContext(ctx)
	if traceID == "" {
		traceID = tracer.NewTraceID()
	}

	event := &domain.OrderEventPayload{
		OrderID:   orderID,
		UserID:    userID,
		ProductID: dto.ProductID,
		Quantity:  dto.Quantity,
		Price:     product.FlashPrice,
		TraceID:   traceID,
		Timestamp: time.Now(),
	}

	if err := u.queueRepo.PublishOrderEvent(ctx, event); err != nil {
		log.Printf("[TraceID: %s] CRITICAL: Failed to publish order event to SQS [OrderID: %s]: %v", traceID, orderID, err)
		// Return stock to Redis when the durable queue did not accept the event.
		if rollbackErr := u.cacheRepo.IncrementStock(ctx, dto.ProductID, dto.Quantity); rollbackErr != nil {
			log.Printf("[TraceID: %s] CRITICAL: Failed to restore Redis stock [OrderID: %s]: %v", traceID, orderID, rollbackErr)
			err = errors.Join(err, fmt.Errorf("restore reserved stock: %w", rollbackErr))
		}
		metrics.OrdersPlacedTotal.WithLabelValues("queue_failed").Inc()
		return nil, fmt.Errorf("failed to process order queue: %w", err)
	}

	keepLock = true
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
	if event == nil || event.OrderID == "" || event.UserID == "" || event.ProductID == "" || event.Quantity < 1 || event.Quantity > 10 || event.Price <= 0 {
		return errors.New("invalid order event payload")
	}

	traceID := event.TraceID
	if traceID == "" {
		traceID = "tr-worker-auto"
	}
	log.Printf("[Worker TraceID: %s] Processing order %s for user %s, product %s", traceID, event.OrderID, event.UserID, event.ProductID)

	// MongoDB transaction + unique order_id index provide durable idempotency.
	order := &domain.Order{
		OrderID:    event.OrderID,
		UserID:     event.UserID,
		ProductID:  event.ProductID,
		Quantity:   event.Quantity,
		TotalPrice: event.Price * float64(event.Quantity),
		Status:     domain.OrderStatusPending,
		CreatedAt:  event.Timestamp,
		UpdatedAt:  time.Now(),
	}

	created, err := u.orderRepo.CreateOrderAndDecrementStock(ctx, order)
	if err != nil {
		log.Printf("[Worker Error] Failed processing order transaction [OrderID: %s]: %v", event.OrderID, err)
		metrics.OrdersPlacedTotal.WithLabelValues("db_error").Inc()
		return err
	}
	if !created {
		metrics.OrdersPlacedTotal.WithLabelValues("duplicate_skipped").Inc()
		_ = u.cacheRepo.PublishEvent(ctx, fmt.Sprintf("order:status:%s", event.OrderID), string(domain.OrderStatusCompleted))
		return nil
	}

	metrics.OrdersPlacedTotal.WithLabelValues("completed").Inc()
	_ = u.cacheRepo.PublishEvent(ctx, fmt.Sprintf("order:status:%s", event.OrderID), string(domain.OrderStatusCompleted))
	log.Printf("[Worker Success] Order %s successfully persisted to MongoDB", event.OrderID)
	return nil
}

func (u *orderUsecase) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	return u.orderRepo.FindByOrderID(ctx, orderID)
}

func (u *orderUsecase) SubscribeOrderStatusStream(ctx context.Context, orderID string) (<-chan string, func(), error) {
	channel := fmt.Sprintf("order:status:%s", orderID)
	return u.cacheRepo.SubscribeEvent(ctx, channel)
}
