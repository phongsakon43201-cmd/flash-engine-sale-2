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
)

type productRepository struct {
	collection *mongo.Collection
}

func NewProductRepository(db *mongo.Database) domain.ProductRepository {
	return &productRepository{
		collection: db.Collection("products"),
	}
}

func (r *productRepository) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	res, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return nil, fmt.Errorf("failed to insert product: %w", err)
	}

	product.ID = res.InsertedID.(primitive.ObjectID)
	return product, nil
}

func (r *productRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid product ID format")
	}

	var product domain.Product
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &product, nil
}

func (r *productRepository) ListProducts(ctx context.Context) ([]*domain.Product, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*domain.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepository) DecrementStock(ctx context.Context, productID string, quantity int) error {
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return errors.New("invalid product ID format")
	}

	filter := bson.M{
		"_id":   objID,
		"stock": bson.M{"$gte": quantity},
	}
	update := bson.M{
		"$inc": bson.M{"stock": -quantity},
		"$set": bson.M{"updated_at": time.Now()},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update product stock: %w", err)
	}

	if res.ModifiedCount == 0 {
		return errors.New("failed to decrement DB stock: insufficient stock or product not found")
	}

	return nil
}
