package handler

import (
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
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	var dto domain.CreateProductDTO
	if err := c.BodyParser(&dto); err != nil {
		return utils.JSONError(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	product, err := h.productUsecase.CreateProduct(c.Context(), &dto)
	if err != nil {
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to create product", err.Error())
	}

	return utils.JSONSuccess(c, fiber.StatusCreated, "Product created successfully", product)
}

// GetProductByID retrieves a product by ID
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.JSONError(c, fiber.StatusBadRequest, "Missing product ID parameter", "")
	}

	product, err := h.productUsecase.GetProductByID(c.Context(), id)
	if err != nil {
		return utils.JSONError(c, fiber.StatusNotFound, "Product not found", err.Error())
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Product fetched successfully", product)
}

// ListProducts retrieves all products
func (h *ProductHandler) ListProducts(c *fiber.Ctx) error {
	products, err := h.productUsecase.ListProducts(c.Context())
	if err != nil {
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to fetch products", err.Error())
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Products list fetched successfully", products)
}

// PrewarmStock pre-populates product stock into Redis cache for Flash Sale
func (h *ProductHandler) PrewarmStock(c *fiber.Ctx) error {
	var dto domain.PrewarmStockDTO
	if err := c.BodyParser(&dto); err != nil {
		return utils.JSONError(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.productUsecase.PrewarmStock(c.Context(), dto.ProductID, dto.Stock); err != nil {
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to prewarm Redis stock", err.Error())
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Flash sale stock pre-warmed in Redis successfully", fiber.Map{
		"product_id": dto.ProductID,
		"stock":      dto.Stock,
	})
}

// GetS3UploadURL generates Presigned Put URL for S3 image uploads
func (h *ProductHandler) GetS3UploadURL(c *fiber.Ctx) error {
	filename := c.Query("filename")
	contentType := c.Query("content_type", "image/jpeg")

	if filename == "" {
		return utils.JSONError(c, fiber.StatusBadRequest, "Query parameter 'filename' is required", "")
	}

	uploadURL, fileURL, err := h.productUsecase.GetPresignedUploadURL(c.Context(), filename, contentType)
	if err != nil {
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to generate presigned upload URL", err.Error())
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Presigned URL generated", fiber.Map{
		"upload_url": uploadURL,
		"file_url":   fileURL,
	})
}
