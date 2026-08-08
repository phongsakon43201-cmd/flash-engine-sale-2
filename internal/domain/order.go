package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusCompleted OrderStatus = "COMPLETED"
	OrderStatusFailed    OrderStatus = "FAILED"
)

type Order struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID     string             `bson:"order_id" json:"order_id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	ProductID   string             `bson:"product_id" json:"product_id"`
	Quantity    int                `bson:"quantity" json:"quantity"`
	TotalPrice  float64            `bson:"total_price" json:"total_price"`
	Status      OrderStatus        `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type OrderEventPayload struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	TraceID   string    `json:"trace_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type CreateOrderDTO struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type OrderResponseDTO struct {
	OrderID   string      `json:"order_id"`
	Status    OrderStatus `json:"status"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}
