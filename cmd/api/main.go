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
	awsRepo "flashsale-go/internal/repository/aws"
	firebaseRepo "flashsale-go/internal/repository/firebase"
	mongoRepo "flashsale-go/internal/repository/mongodb"
	redisRepo "flashsale-go/internal/repository/redis"
	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/config"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
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

	log.Printf("Starting %s in %s mode...", cfg.AppName, cfg.Env)

	// 1. Initialize MongoDB Client
	mongoClient, err := mongoRepo.NewMongoClient(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("MongoDB initialization failed: %v", err)
	}

	// 2. Initialize Redis Client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Redis connection ping failed: %v. Continuing...", err)
	}

	// 3. Initialize Repositories
	productDBRepo := mongoRepo.NewProductRepository(mongoClient.Database)
	orderDBRepo := mongoRepo.NewOrderRepository(mongoClient.Database)
	cacheRepo := redisRepo.NewRedisRepository(redisClient)

	sqsQueueRepo, err := awsRepo.NewSQSRepository(
		cfg.AWSRegion,
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		cfg.AWSEndpoint,
		cfg.AWSSQSQueueURL,
	)
	if err != nil {
		log.Fatalf("AWS SQS initialization failed: %v", err)
	}

	s3StorageRepo, err := awsRepo.NewS3Repository(
		cfg.AWSRegion,
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		cfg.AWSEndpoint,
		cfg.AWSS3Bucket,
	)
	if err != nil {
		log.Printf("Warning: AWS S3 initialization failed: %v", err)
	}

	authClient := firebaseRepo.NewAuthClient(cfg.FirebaseDevMode)

	// 4. Initialize UseCases
	authUsecase := usecase.NewAuthUsecase(authClient)
	productUsecase := usecase.NewProductUsecase(productDBRepo, cacheRepo, s3StorageRepo)
	orderUsecase := usecase.NewOrderUsecase(orderDBRepo, productDBRepo, cacheRepo, sqsQueueRepo)

	// 5. Setup Fiber HTTP App
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	deliveryHTTP.SetupRouter(app, productUsecase, orderUsecase, authUsecase)

	// 6. Graceful Shutdown Listener
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("Fiber server failed to listen: %v", err)
		}
	}()

	log.Printf("Server successfully listening on port %s", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API server gracefully...")
	_ = app.Shutdown()
	log.Println("API Server gracefully stopped.")
}
