package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "flashsale-go/docs"
	deliveryHTTP "flashsale-go/internal/delivery/http"
	"flashsale-go/internal/domain"
	awsRepo "flashsale-go/internal/repository/aws"
	firebaseRepo "flashsale-go/internal/repository/firebase"
	mongoRepo "flashsale-go/internal/repository/mongodb"
	queueRepo "flashsale-go/internal/repository/queue"
	redisRepo "flashsale-go/internal/repository/redis"
	storageRepo "flashsale-go/internal/repository/storage"
	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/config"

	"github.com/gofiber/fiber/v2"
)

// @title Flash Sale Engine API
// @version 1.0
// @description High-Concurrency Flash Sale Engine REST API Server with Redis Distributed Locks and AWS SQS Event Queue.
// @termsOfService http://swagger.io/terms/

// @contact.name Flash Sale Support Team
// @contact.url https://github.com/phongsakontle/flash-engine-sale-

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer <Firebase-JWT-Token>" to authenticate protected endpoints.
func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Starting %s in %s mode...", cfg.AppName, cfg.Env)

	// 1. Initialize MongoDB Client
	mongoClient, err := mongoRepo.NewMongoClient(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("MongoDB initialization failed: %v", err)
	}
	defer func() {
		if err := mongoClient.Client.Disconnect(context.Background()); err != nil {
			log.Printf("MongoDB disconnect failed: %v", err)
		}
	}()

	// 2. Initialize Redis Client
	redisClient, err := redisRepo.NewClient(cfg.RedisURL, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("Redis initialization failed: %v", err)
	}
	defer func() { _ = redisClient.Close() }()
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Redis connection ping failed: %v. Continuing...", err)
	}

	// 3. Initialize Repositories
	productDBRepo := mongoRepo.NewProductRepository(mongoClient.Database)
	orderDBRepo, err := mongoRepo.NewOrderRepository(mongoClient.Database)
	if err != nil {
		log.Fatalf("Order repository initialization failed: %v", err)
	}
	cacheRepo := redisRepo.NewRedisRepository(redisClient)

	var queueRepository domain.QueueRepository
	switch cfg.QueueDriver {
	case "memory":
		queueRepository = queueRepo.NewMemoryQueueRepository(1000)
		log.Println("Using in-memory order queue; queued work is not durable across restarts")
	case "sqs":
		queueRepository, err = awsRepo.NewSQSRepository(
			cfg.AWSRegion,
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			cfg.AWSEndpoint,
			cfg.AWSSQSQueueURL,
		)
		if err != nil {
			log.Fatalf("AWS SQS initialization failed: %v", err)
		}
	}

	var storageRepository domain.StorageRepository
	switch cfg.StorageDriver {
	case "disabled":
		storageRepository = storageRepo.NewDisabledRepository()
		log.Println("Object storage uploads are disabled for this deployment")
	case "s3":
		storageRepository, err = awsRepo.NewS3Repository(
			cfg.AWSRegion,
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			cfg.AWSEndpoint,
			cfg.AWSS3Bucket,
		)
		if err != nil {
			log.Fatalf("AWS S3 initialization failed: %v", err)
		}
	}

	authClient, err := firebaseRepo.NewAuthClient(context.Background(), cfg.FirebaseDevMode, cfg.FirebaseCredsPath, cfg.FirebaseCredsJSON)
	if err != nil {
		log.Fatalf("Firebase initialization failed: %v", err)
	}

	// 4. Initialize UseCases
	authUsecase := usecase.NewAuthUsecase(authClient)
	productUsecase := usecase.NewProductUsecase(productDBRepo, cacheRepo, storageRepository)
	orderUsecase := usecase.NewOrderUsecase(orderDBRepo, productDBRepo, cacheRepo, queueRepository)

	runtimeCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.QueueDriver == "memory" {
		go func() {
			err := queueRepository.ReceiveOrderEvents(runtimeCtx, func(event *domain.OrderEventPayload) error {
				return orderUsecase.ProcessOrderFromQueue(runtimeCtx, event)
			})
			if err != nil {
				log.Printf("In-memory order worker stopped with error: %v", err)
			}
		}()
	}

	// 5. Setup Fiber HTTP App
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		BodyLimit:    2 * 1024 * 1024,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	deliveryHTTP.SetupRouter(app, productUsecase, orderUsecase, authUsecase, cfg.AllowedOrigins)
	app.Get("/ready", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		if err := mongoClient.Client.Ping(ctx, nil); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not_ready", "dependency": "mongodb"})
		}
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not_ready", "dependency": "redis"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// 6. Serve until startup fails or a shutdown signal is received.
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Listen(":" + cfg.Port)
	}()

	log.Printf("Server successfully listening on port %s", cfg.Port)

	select {
	case err := <-serverErrors:
		log.Printf("Fiber server stopped unexpectedly: %v", err)
		return
	case <-runtimeCtx.Done():
	}

	log.Println("Shutting down API server gracefully...")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("API server shutdown failed: %v", err)
	}
	log.Println("API Server gracefully stopped.")
}
