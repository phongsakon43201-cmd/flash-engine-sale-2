package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"flashsale-go/internal/delivery/http/handler"
	"flashsale-go/internal/domain"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockProductUsecase struct {
	mock.Mock
}

func (m *MockProductUsecase) CreateProduct(ctx context.Context, dto *domain.CreateProductDTO) (*domain.Product, error) {
	args := m.Called(ctx, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *MockProductUsecase) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *MockProductUsecase) ListProducts(ctx context.Context) ([]*domain.Product, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Product), args.Error(1)
}

func (m *MockProductUsecase) PrewarmStock(ctx context.Context, productID string, stock int) error {
	args := m.Called(ctx, productID, stock)
	return args.Error(0)
}

func (m *MockProductUsecase) GetRedisStock(ctx context.Context, productID string) (int, error) {
	args := m.Called(ctx, productID)
	return args.Int(0), args.Error(1)
}

func (m *MockProductUsecase) GetPresignedUploadURL(ctx context.Context, filename string, contentType string) (string, string, error) {
	args := m.Called(ctx, filename, contentType)
	return args.String(0), args.String(1), args.Error(2)
}

func TestProductHandler_ListProducts(t *testing.T) {
	mockUsecase := new(MockProductUsecase)
	productHandler := handler.NewProductHandler(mockUsecase)

	app := fiber.New()
	app.Get("/products", productHandler.ListProducts)

	pID := primitive.NewObjectID()

	mockUsecase.On("ListProducts", mock.Anything).Return([]*domain.Product{
		{ID: pID, Title: "Phone"},
	}, nil)

	req := httptest.NewRequest("GET", "/products", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestProductHandler_GetProductByID_Success(t *testing.T) {
	mockUsecase := new(MockProductUsecase)
	productHandler := handler.NewProductHandler(mockUsecase)

	app := fiber.New()
	app.Get("/products/:id", productHandler.GetProductByID)

	pID := primitive.NewObjectID()
	pIDHex := pID.Hex()

	mockUsecase.On("GetProductByID", mock.Anything, pIDHex).Return(&domain.Product{
		ID:    pID,
		Title: "Gaming Laptop",
	}, nil)

	req := httptest.NewRequest("GET", "/products/"+pIDHex, nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestProductHandler_GetProductByID_NotFound(t *testing.T) {
	mockUsecase := new(MockProductUsecase)
	productHandler := handler.NewProductHandler(mockUsecase)

	app := fiber.New()
	app.Get("/products/:id", productHandler.GetProductByID)

	mockUsecase.On("GetProductByID", mock.Anything, "non-existent").Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/products/non-existent", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestProductHandler_PrewarmStock(t *testing.T) {
	mockUsecase := new(MockProductUsecase)
	productHandler := handler.NewProductHandler(mockUsecase)

	app := fiber.New()
	app.Post("/products/prewarm", productHandler.PrewarmStock)

	dto := domain.PrewarmStockDTO{
		ProductID: "prod-123",
		Stock:     500,
	}
	body, _ := json.Marshal(dto)

	mockUsecase.On("PrewarmStock", mock.Anything, "prod-123", 500).Return(nil)

	req := httptest.NewRequest("POST", "/products/prewarm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
