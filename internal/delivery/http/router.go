package http

import (
	"flashsale-go/internal/delivery/http/handler"
	"flashsale-go/internal/delivery/http/middleware"
	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/metrics"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
)

func SetupRouter(
	app *fiber.App,
	productUsecase usecase.ProductUsecase,
	orderUsecase usecase.OrderUsecase,
	authUsecase usecase.AuthUsecase,
	allowedOrigins string,
) {
	// Middleware Setup
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(metrics.PrometheusMiddleware())
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Serve Web Dashboard Static Files
	app.Static("/", "./web")

	// Swagger Endpoint
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Prometheus Metrics Endpoint
	app.Get("/metrics", metrics.PrometheusHandler())

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "healthy",
			"service": "flashsale-go-api",
		})
	})

	api := app.Group("/api/v1")
	// Apply a coarse per-IP API limit before route-specific protections.
	api.Use(middleware.NewRateLimiter(300, 60))

	// Handlers
	productHandler := handler.NewProductHandler(productUsecase)
	orderHandler := handler.NewOrderHandler(orderUsecase)

	// Public Product Routes
	api.Get("/products", productHandler.ListProducts)
	api.Get("/products/:id", productHandler.GetProductByID)
	api.Get("/products/:id/stock", productHandler.GetRedisStock)

	// Rate Limiter for Flash Sale Order API
	orderRateLimiter := middleware.NewRateLimiter(100, 1) // Max 100 requests/sec per IP

	// Protected Routes (Require Firebase JWT)
	protected := api.Group("", middleware.FirebaseAuthMiddleware(authUsecase))
	adminOnly := middleware.RequireRole("admin", "seller")

	// Admin / Seller Endpoints
	protected.Post("/products", adminOnly, productHandler.CreateProduct)
	protected.Post("/products/prewarm", adminOnly, productHandler.PrewarmStock)
	protected.Get("/upload-url", adminOnly, productHandler.GetS3UploadURL)

	// Flash Sale High-Concurrency Order Endpoint
	protected.Post("/orders/flash-sale", orderRateLimiter, orderHandler.CreateFlashSaleOrder)
	protected.Get("/orders/:id", orderHandler.GetOrderByID)
	protected.Get("/orders/:id/stream", orderHandler.StreamOrderStatus)
}
