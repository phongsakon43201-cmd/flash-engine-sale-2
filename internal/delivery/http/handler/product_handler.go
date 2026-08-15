package handler

import (
	"errors"
	"log"

	"flashsale-go/internal/domain"
	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ProductHandler struct {
	productUsecase usecase.ProductUsecase
}

func NewProductHandler(productUsecase usecase.ProductUsecase) *ProductHandler {
	return &ProductHandler{
		productUsecase: productUsecase,
	}
}

// CreateProduct creates a new product
// @Summary Create a new product
// @Description Create product details for store catalog
// @Tags products
// @Accept json
// @Produce json
// @Param product body domain.CreateProductDTO true "Product payload"
// @Success 201 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	var dto domain.CreateProductDTO
	if err := c.BodyParser(&dto); err != nil {
		return utils.JSONError(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	product, err := h.productUsecase.CreateProduct(c.Context(), &dto)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidProduct) {
			return utils.JSONError(c, fiber.StatusBadRequest, "Invalid product", err.Error())
		}
		log.Printf("Failed to create product: %v", err)
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to create product", "Unable to create product")
	}

	return utils.JSONSuccess(c, fiber.StatusCreated, "Product created successfully", product)
}

// GetProductByID retrieves a product by ID
// @Summary Get product by ID
// @Description Retrieve single product detail by ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /products/{id} [get]
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.JSONError(c, fiber.StatusBadRequest, "Missing product ID parameter", "")
	}

	product, err := h.productUsecase.GetProductByID(c.Context(), id)
	if err != nil {
		log.Printf("Failed to fetch product %s: %v", id, err)
		return utils.JSONError(c, fiber.StatusNotFound, "Product not found", "The requested product was not found")
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Product fetched successfully", product)
}

// ListProducts retrieves all products
// @Summary List all products
// @Description Fetch list of all active products
// @Tags products
// @Produce json
// @Success 200 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /products [get]
func (h *ProductHandler) ListProducts(c *fiber.Ctx) error {
	products, err := h.productUsecase.ListProducts(c.Context())
	if err != nil {
		log.Printf("Failed to fetch products: %v", err)
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to fetch products", "Unable to load products")
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Products list fetched successfully", products)
}

// PrewarmStock pre-populates product stock into Redis cache for Flash Sale
// @Summary Prewarm product stock in Redis
// @Description Pre-populate product stock into Redis cache before Flash Sale starts
// @Tags products
// @Accept json
// @Produce json
// @Param payload body domain.PrewarmStockDTO true "Prewarm Stock DTO"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /products/prewarm [post]
func (h *ProductHandler) PrewarmStock(c *fiber.Ctx) error {
	var dto domain.PrewarmStockDTO
	if err := c.BodyParser(&dto); err != nil {
		return utils.JSONError(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.productUsecase.PrewarmStock(c.Context(), dto.ProductID, dto.Stock); err != nil {
		if errors.Is(err, usecase.ErrInvalidPrewarmStock) {
			return utils.JSONError(c, fiber.StatusBadRequest, "Invalid stock", err.Error())
		}
		log.Printf("Failed to prewarm Redis stock: %v", err)
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to prewarm Redis stock", "Unable to prewarm stock")
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Flash sale stock pre-warmed in Redis successfully", fiber.Map{
		"product_id": dto.ProductID,
		"stock":      dto.Stock,
	})
}

// GetRedisStock gets the active Redis stock count for a product
// @Summary Get live Redis stock
// @Description Fetch real-time available stock count from Redis cache
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /products/{id}/stock [get]
func (h *ProductHandler) GetRedisStock(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.JSONError(c, fiber.StatusBadRequest, "Missing product ID parameter", "")
	}

	stock, err := h.productUsecase.GetRedisStock(c.Context(), id)
	if err != nil {
		log.Printf("Failed to fetch Redis stock for %s: %v", id, err)
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to get Redis stock", "Unable to load stock")
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Redis stock fetched", fiber.Map{
		"product_id": id,
		"stock":      stock,
	})
}

// GetS3UploadURL generates Presigned Put URL for S3 image uploads
// @Summary Generate S3 presigned upload URL
// @Description Get presigned S3 upload URL for product image upload
// @Tags products
// @Produce json
// @Param filename query string true "Filename (e.g. image.jpg)"
// @Param content_type query string false "Content-Type (e.g. image/jpeg)"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /upload-url [get]
func (h *ProductHandler) GetS3UploadURL(c *fiber.Ctx) error {
	filename := c.Query("filename")
	contentType := c.Query("content_type", "image/jpeg")

	if filename == "" {
		return utils.JSONError(c, fiber.StatusBadRequest, "Query parameter 'filename' is required", "")
	}

	uploadURL, fileURL, err := h.productUsecase.GetPresignedUploadURL(c.Context(), filename, contentType)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidUpload) {
			return utils.JSONError(c, fiber.StatusBadRequest, "Invalid upload request", err.Error())
		}
		log.Printf("Failed to generate presigned upload URL: %v", err)
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to generate presigned upload URL", "Unable to generate upload URL")
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Presigned URL generated", fiber.Map{
		"upload_url": uploadURL,
		"file_url":   fileURL,
	})
}
