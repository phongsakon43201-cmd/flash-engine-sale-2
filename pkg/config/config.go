package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                  string
	Env                   string
	AppName               string
	MongoURI              string
	MongoDBName           string
	RedisAddr             string
	RedisPassword         string
	AWSRegion             string
	AWSAccessKeyID        string
	AWSSecretAccessKey    string
	AWSEndpoint           string
	AWSS3Bucket           string
	AWSSQSQueueURL        string
	FirebaseDevMode       bool
	FirebaseCredsPath     string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, reading from environment variables")
	}

	return &Config{
		Port:                  getEnv("PORT", "8080"),
		Env:                   getEnv("ENV", "development"),
		AppName:               getEnv("APP_NAME", "flashsale-engine"),
		MongoURI:              getEnv("MONGO_URI", "mongodb://root:examplepassword@localhost:27017"),
		MongoDBName:           getEnv("MONGO_DB_NAME", "flashsale_db"),
		RedisAddr:             getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		AWSRegion:             getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:        getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey:    getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		AWSEndpoint:           getEnv("AWS_ENDPOINT", "http://127.0.0.1:4566"),
		AWSS3Bucket:           getEnv("AWS_S3_BUCKET", "flashsale-product-images"),
		AWSSQSQueueURL:        getEnv("AWS_SQS_QUEUE_URL", "http://127.0.0.1:4566/000000000000/flashsale-order-queue"),
		FirebaseDevMode:       getEnv("FIREBASE_DEV_MODE", "true") == "true",
		FirebaseCredsPath:     getEnv("FIREBASE_CREDENTIALS_PATH", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultValue
}
