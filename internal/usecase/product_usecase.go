package usecase

import (
	"context"
	"errors"
	"fmt"

	"flashsale-go/internal/domain"
)

type ProductUsecase interface {
	CreateProduct(ctx context.Context, dto *domain.CreateProductDTO) (*domain.Product, error)
	GetProductByID(ctx context.Context, id string) (*domain.Product, error)
	ListProducts(ctx context.Context) ([]*domain.Product, error)
	PrewarmStock(ctx context.Context, productID string, stock int) error
	GetRedisStock(ctx context.Context, productID string) (int, error)
	GetPresignedUploadURL(ctx context.Context, filename, contentType string) (uploadURL string, fileURL string, err error)
}

type productUsecase struct {
	productRepo domain.ProductRepository
	cacheRepo   domain.CacheRepository
	storageRepo domain.StorageRepository
}

func NewProductUsecase(productRepo domain.ProductRepository, cacheRepo domain.CacheRepository, storageRepo domain.StorageRepository) ProductUsecase {
	return &productUsecase{
		productRepo: productRepo,
		cacheRepo:   cacheRepo,
		storageRepo: storageRepo,
	}
}

func (u *productUsecase) CreateProduct(ctx context.Context, dto *domain.CreateProductDTO) (*domain.Product, error) {
	if dto.Title == "" || dto.Stock < 0 || dto.Price < 0 {
		return nil, errors.New("invalid product input payload")
	}

	product := &domain.Product{
		Title:       dto.Title,
		Description: dto.Description,
		ImageURL:    dto.ImageURL,
		Price:       dto.Price,
		Stock:       dto.Stock,
		IsFlashSale: dto.IsFlashSale,
		FlashPrice:  dto.FlashPrice,
	}

	createdProduct, err := u.productRepo.CreateProduct(ctx, product)
	if err != nil {
		return nil, err
	}

	// Auto pre-warm Redis stock if Flash Sale product
	if createdProduct.IsFlashSale {
		if err := u.cacheRepo.PrewarmStock(ctx, createdProduct.ID.Hex(), createdProduct.Stock); err != nil {
			fmt.Printf("Warning: failed to auto pre-warm stock in Redis: %v\n", err)
		}
	}

	return createdProduct, nil
}

func (u *productUsecase) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	return u.productRepo.FindByID(ctx, id)
}

func (u *productUsecase) ListProducts(ctx context.Context) ([]*domain.Product, error) {
	return u.productRepo.ListProducts(ctx)
}

func (u *productUsecase) PrewarmStock(ctx context.Context, productID string, stock int) error {
	product, err := u.productRepo.FindByID(ctx, productID)
	if err != nil {
		return err
	}

	return u.cacheRepo.PrewarmStock(ctx, product.ID.Hex(), stock)
}

func (u *productUsecase) GetRedisStock(ctx context.Context, productID string) (int, error) {
	return u.cacheRepo.GetStock(ctx, productID)
}

func (u *productUsecase) GetPresignedUploadURL(ctx context.Context, filename, contentType string) (string, string, error) {
	if u.storageRepo == nil {
		return "", "", errors.New("storage repository not configured")
	}
	return u.storageRepo.GeneratePresignedUploadURL(ctx, filename, contentType)
}
