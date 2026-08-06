package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"flashsale-go/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type orderRepository struct {
	collection *mongo.Collection
}

func NewOrderRepository(db *mongo.Database) domain.OrderRepository {
	return &orderRepository{
		collection: db.Collection("orders"),
	}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	return nil
}

func (r *orderRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Order, error) {
	var order domain.Order
	err := r.collection.FindOne(ctx, bson.M{"order_id": orderID}).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}

	return &order, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	filter := bson.M{"order_id": orderID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if res.ModifiedCount == 0 {
		return errors.New("order not found or status unchanged")
	}

	return nil
}
