package http

import (
	"flashsale-go/internal/delivery/http/handler"
	"flashsale-go/internal/delivery/http/middleware"
	"flashsale-go/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func SetupRouter(
	app *fiber.App,
	productUsecase usecase.ProductUsecase,
	orderUsecase usecase.OrderUsecase,
	authUsecase usecase.AuthUsecase,
) {
	// Middleware Setup
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "healthy",
			"service": "flashsale-go-api",
		})
	})

	api := app.Group("/api/v1")

	// Handlers
	productHandler := handler.NewProductHandler(productUsecase)
	orderHandler := handler.NewOrderHandler(orderUsecase)

	// Public Product Routes
	api.Get("/products", productHandler.ListProducts)
	api.Get("/products/:id", productHandler.GetProductByID)

	// Rate Limiter for Flash Sale Order API
	orderRateLimiter := middleware.NewRateLimiter(100, 1) // Max 100 requests/sec per IP

	// Protected Routes (Require Firebase JWT)
	protected := api.Group("", middleware.FirebaseAuthMiddleware(authUsecase))

	// Admin / Seller Endpoints
	protected.Post("/products", productHandler.CreateProduct)
	protected.Post("/products/prewarm", productHandler.PrewarmStock)
	protected.Get("/upload-url", productHandler.GetS3UploadURL)

	// Flash Sale High-Concurrency Order Endpoint
	protected.Post("/orders/flash-sale", orderRateLimiter, orderHandler.CreateFlashSaleOrder)
	protected.Get("/orders/:id", orderHandler.GetOrderByID)
}
