package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	FindByFirebaseID(ctx context.Context, firebaseID string) (*User, error)
}

type ProductRepository interface {
	CreateProduct(ctx context.Context, product *Product) (*Product, error)
	FindByID(ctx context.Context, id string) (*Product, error)
	ListProducts(ctx context.Context) ([]*Product, error)
	DecrementStock(ctx context.Context, productID string, quantity int) error
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	FindByOrderID(ctx context.Context, orderID string) (*Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status OrderStatus) error
}

type CacheRepository interface {
	PrewarmStock(ctx context.Context, productID string, stock int) error
	GetStock(ctx context.Context, productID string) (int, error)
	// DecrementStockAtomic executes Redis Lua Script ensuring zero over-selling
	DecrementStockAtomic(ctx context.Context, productID string, quantity int) (bool, error)
	// IncrementStock adds back stock to Redis when order placement/queue fails
	IncrementStock(ctx context.Context, productID string, quantity int) error
	// AcquireLock attempts to set a distributed lock with TTL
	AcquireLock(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string, value string) error
}

type QueueRepository interface {
	PublishOrderEvent(ctx context.Context, event *OrderEventPayload) error
	ReceiveOrderEvents(ctx context.Context, handler func(event *OrderEventPayload) error) error
}

type StorageRepository interface {
	GeneratePresignedUploadURL(ctx context.Context, filename string, contentType string) (string, string, error)
}
