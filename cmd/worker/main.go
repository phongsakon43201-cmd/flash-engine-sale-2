package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"flashsale-go/internal/domain"
	awsRepo "flashsale-go/internal/repository/aws"
	mongoRepo "flashsale-go/internal/repository/mongodb"
	redisRepo "flashsale-go/internal/repository/redis"
	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/config"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Println("Starting Flash Sale Async SQS Consumer Worker...")

	// 1. Initialize MongoDB Client
	mongoClient, err := mongoRepo.NewMongoClient(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("[Worker] MongoDB connection failed: %v", err)
	}
	defer func() {
		if err := mongoClient.Client.Disconnect(context.Background()); err != nil {
			log.Printf("[Worker] MongoDB disconnect failed: %v", err)
		}
	}()

	// 2. Initialize Redis Client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	defer func() { _ = redisClient.Close() }()

	// 3. Initialize Repositories
	productDBRepo := mongoRepo.NewProductRepository(mongoClient.Database)
	orderDBRepo, err := mongoRepo.NewOrderRepository(mongoClient.Database)
	if err != nil {
		log.Fatalf("[Worker] Order repository initialization failed: %v", err)
	}
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Start listening to SQS queue messages
	log.Println("[Worker] Worker loop listening for SQS messages...")
	err = sqsQueueRepo.ReceiveOrderEvents(ctx, func(event *domain.OrderEventPayload) error {
		return orderUsecase.ProcessOrderFromQueue(ctx, event)
	})

	if err != nil {
		log.Printf("[Worker] Consumer loop terminated with error: %v", err)
	}

	log.Println("[Worker] Shutting down worker consumer...")
	log.Println("[Worker] Worker successfully shut down.")
}
