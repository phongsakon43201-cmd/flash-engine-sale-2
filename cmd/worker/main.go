package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	awsRepo "flashsale-go/internal/repository/aws"
	mongoRepo "flashsale-go/internal/repository/mongodb"
	redisRepo "flashsale-go/internal/repository/redis"
	"flashsale-go/internal/domain"
	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/config"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.LoadConfig()

	log.Println("Starting Flash Sale Async SQS Consumer Worker...")

	// 1. Initialize MongoDB Client
	mongoClient, err := mongoRepo.NewMongoClient(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("[Worker] MongoDB connection failed: %v", err)
	}

	// 2. Initialize Redis Client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

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
		log.Fatalf("[Worker] AWS SQS initialization failed: %v", err)
	}

	// 4. Initialize Order Usecase
	orderUsecase := usecase.NewOrderUsecase(orderDBRepo, productDBRepo, cacheRepo, sqsQueueRepo)

	ctx, cancel := context.WithCancel(context.Background())

	// 5. Graceful shutdown handler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("[Worker] Shutting down worker consumer...")
		cancel()
	}()

	// 6. Start listening to SQS queue messages
	log.Println("[Worker] Worker loop listening for SQS messages...")
	err = sqsQueueRepo.ReceiveOrderEvents(ctx, func(event *domain.OrderEventPayload) error {
		return orderUsecase.ProcessOrderFromQueue(ctx, event)
	})

	if err != nil {
		log.Printf("[Worker] Consumer loop terminated with error: %v", err)
	}

	log.Println("[Worker] Worker successfully shut down.")
}
