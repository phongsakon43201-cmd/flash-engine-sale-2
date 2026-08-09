package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"flashsale-go/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type orderRepository struct {
	client            *mongo.Client
	collection        *mongo.Collection
	productCollection *mongo.Collection
}

func NewOrderRepository(db *mongo.Database) (domain.OrderRepository, error) {
	collection := db.Collection("orders")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "order_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_order_id"),
	})
	if err != nil {
		return nil, fmt.Errorf("create order indexes: %w", err)
	}

	return &orderRepository{
		client:            db.Client(),
		collection:        collection,
		productCollection: db.Collection("products"),
	}, nil
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

func (r *orderRepository) CreateOrderAndDecrementStock(ctx context.Context, order *domain.Order) (bool, error) {
	session, err := r.client.StartSession()
	if err != nil {
		return false, fmt.Errorf("start MongoDB session: %w", err)
	}
	defer session.EndSession(ctx)

	txnOptions := options.Transaction().SetWriteConcern(writeconcern.Majority())
	result, err := session.WithTransaction(ctx, func(sessionCtx mongo.SessionContext) (interface{}, error) {
		var existing domain.Order
		err := r.collection.FindOne(sessionCtx, bson.M{"order_id": order.OrderID}).Decode(&existing)
		if err == nil {
			return false, nil
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("check existing order: %w", err)
		}

		productID, err := primitive.ObjectIDFromHex(order.ProductID)
		if err != nil {
			return nil, errors.New("invalid product ID format")
		}

		stockResult, err := r.productCollection.UpdateOne(
			sessionCtx,
			bson.M{"_id": productID, "stock": bson.M{"$gte": order.Quantity}},
			bson.M{
				"$inc": bson.M{"stock": -order.Quantity},
				"$set": bson.M{"updated_at": time.Now().UTC()},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("decrement product stock: %w", err)
		}
		if stockResult.ModifiedCount != 1 {
			return nil, errors.New("insufficient MongoDB stock or product not found")
		}

		now := time.Now().UTC()
		order.Status = domain.OrderStatusCompleted
		if order.CreatedAt.IsZero() {
			order.CreatedAt = now
		}
		order.UpdatedAt = now
		if _, err := r.collection.InsertOne(sessionCtx, order); err != nil {
			return nil, fmt.Errorf("insert order: %w", err)
		}

		return true, nil
	}, txnOptions)
	if err != nil {
		return false, fmt.Errorf("process order transaction: %w", err)
	}

	created, ok := result.(bool)
	if !ok {
		return false, errors.New("unexpected MongoDB transaction result")
	}
	return created, nil
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

	if res.MatchedCount == 0 {
		return errors.New("order not found")
	}

	return nil
}
