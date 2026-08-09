package usecase_test

import (
	"context"
	"testing"

	"flashsale-go/internal/domain"
	"flashsale-go/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateProduct_Success(t *testing.T) {
	ctx := context.Background()
	mockProductRepo := new(MockProductRepository)
	mockCacheRepo := new(MockCacheRepository)

	productUsecase := usecase.NewProductUsecase(mockProductRepo, mockCacheRepo, nil)

	dto := &domain.CreateProductDTO{
		Title:       "iPhone 15 Pro Flash Sale",
		Description: "Special deal",
		Price:       35000,
		Stock:       10,
		IsFlashSale: true,
		FlashPrice:  19900,
	}

	objID := primitive.NewObjectID()
	mockProductRepo.On("CreateProduct", ctx, mock.Anything).Return(&domain.Product{
		ID:          objID,
		Title:       dto.Title,
		Price:       dto.Price,
		Stock:       dto.Stock,
		IsFlashSale: true,
		FlashPrice:  dto.FlashPrice,
	}, nil)

	// Prewarm stock in Redis expectation
	mockCacheRepo.On("PrewarmStock", ctx, objID.Hex(), 10).Return(nil)

	result, err := productUsecase.CreateProduct(ctx, dto)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "iPhone 15 Pro Flash Sale", result.Title)
	mockProductRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestCreateProduct_InvalidInput(t *testing.T) {
	ctx := context.Background()
	productUsecase := usecase.NewProductUsecase(nil, nil, nil)

	// Invalid payload: Title is empty
	dto := &domain.CreateProductDTO{
		Title: "",
		Price: 100,
		Stock: 5,
	}

	result, err := productUsecase.CreateProduct(ctx, dto)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "invalid product input payload", err.Error())
}
