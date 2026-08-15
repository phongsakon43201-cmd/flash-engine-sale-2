package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Env                string
	AppName            string
	MongoURI           string
	MongoDBName        string
	RedisURL           string
	RedisAddr          string
	RedisPassword      string
	QueueDriver        string
	StorageDriver      string
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSEndpoint        string
	AWSS3Bucket        string
	AWSSQSQueueURL     string
	FirebaseDevMode    bool
	FirebaseCredsPath  string
	FirebaseCredsJSON  string
	AllowedOrigins     string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, reading from environment variables")
	}
	port := getEnv("PORT", "8080")
	allowedOrigins := "http://localhost:" + port
	if renderHostname := strings.TrimSpace(os.Getenv("RENDER_EXTERNAL_HOSTNAME")); renderHostname != "" {
		allowedOrigins = "https://" + renderHostname
	}

	env := getEnv("ENV", "development")
	awsEndpoint := getEnvAllowEmpty("AWS_ENDPOINT", "")
	awsAccessKeyID := getEnvAllowEmpty("AWS_ACCESS_KEY_ID", "")
	awsSecretAccessKey := getEnvAllowEmpty("AWS_SECRET_ACCESS_KEY", "")
	awsSQSQueueURL := getEnvAllowEmpty("AWS_SQS_QUEUE_URL", "")
	awsS3Bucket := getEnvAllowEmpty("AWS_S3_BUCKET", "")
	if !strings.EqualFold(env, "production") && awsEndpoint == "" {
		awsEndpoint = "http://127.0.0.1:4566"
	}
	if !strings.EqualFold(env, "production") {
		if awsAccessKeyID == "" {
			awsAccessKeyID = "test"
		}
		if awsSecretAccessKey == "" {
			awsSecretAccessKey = "test"
		}
		if awsSQSQueueURL == "" {
			awsSQSQueueURL = "http://127.0.0.1:4566/000000000000/flashsale-order-queue"
		}
		if awsS3Bucket == "" {
			awsS3Bucket = "flashsale-product-images"
		}
	}

	return &Config{
		Port:               port,
		Env:                env,
		AppName:            getEnv("APP_NAME", "flashsale-engine"),
		MongoURI:           getEnv("MONGO_URI", "mongodb://root:examplepassword@localhost:27017"),
		MongoDBName:        getEnv("MONGO_DB_NAME", "flashsale_db"),
		RedisURL:           getEnvAllowEmpty("REDIS_URL", ""),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnvAllowEmpty("REDIS_PASSWORD", ""),
		QueueDriver:        strings.ToLower(strings.TrimSpace(getEnv("QUEUE_DRIVER", "sqs"))),
		StorageDriver:      strings.ToLower(strings.TrimSpace(getEnv("STORAGE_DRIVER", "s3"))),
		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:     awsAccessKeyID,
		AWSSecretAccessKey: awsSecretAccessKey,
		AWSEndpoint:        awsEndpoint,
		AWSS3Bucket:        awsS3Bucket,
		AWSSQSQueueURL:     awsSQSQueueURL,
		FirebaseDevMode:    getEnvBool("FIREBASE_DEV_MODE", true),
		FirebaseCredsPath:  getEnvAllowEmpty("FIREBASE_CREDENTIALS_PATH", ""),
		FirebaseCredsJSON:  getEnvAllowEmpty("FIREBASE_CREDENTIALS_JSON", ""),
		AllowedOrigins:     getEnv("ALLOWED_ORIGINS", allowedOrigins),
	}
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Port) == "" || strings.TrimSpace(c.MongoURI) == "" || strings.TrimSpace(c.MongoDBName) == "" {
		return errors.New("PORT, MONGO_URI and MONGO_DB_NAME are required")
	}
	if strings.TrimSpace(c.RedisURL) == "" && strings.TrimSpace(c.RedisAddr) == "" {
		return errors.New("REDIS_URL or REDIS_ADDR is required")
	}
	if strings.TrimSpace(c.AllowedOrigins) == "" {
		return errors.New("ALLOWED_ORIGINS must not be empty")
	}

	queueDriver := strings.ToLower(strings.TrimSpace(c.QueueDriver))
	if queueDriver == "" {
		queueDriver = "sqs"
	}
	if queueDriver != "sqs" && queueDriver != "memory" {
		return errors.New("QUEUE_DRIVER must be 'sqs' or 'memory'")
	}
	if queueDriver == "sqs" && strings.TrimSpace(c.AWSSQSQueueURL) == "" {
		return errors.New("AWS_SQS_QUEUE_URL is required when QUEUE_DRIVER=sqs")
	}

	storageDriver := strings.ToLower(strings.TrimSpace(c.StorageDriver))
	if storageDriver == "" {
		storageDriver = "s3"
	}
	if storageDriver != "s3" && storageDriver != "disabled" {
		return errors.New("STORAGE_DRIVER must be 's3' or 'disabled'")
	}

	if strings.EqualFold(c.Env, "production") {
		if c.FirebaseDevMode {
			return errors.New("FIREBASE_DEV_MODE must be false in production")
		}
		if strings.Contains(c.AllowedOrigins, "*") {
			return errors.New("ALLOWED_ORIGINS must not be '*' in production")
		}
		if strings.TrimSpace(c.FirebaseCredsPath) == "" && strings.TrimSpace(c.FirebaseCredsJSON) == "" {
			return errors.New("FIREBASE_CREDENTIALS_PATH or FIREBASE_CREDENTIALS_JSON is required in production")
		}
		if (queueDriver == "sqs" || storageDriver == "s3") && (strings.TrimSpace(c.AWSAccessKeyID) == "" || strings.TrimSpace(c.AWSSecretAccessKey) == "") {
			return errors.New("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are required for production AWS drivers")
		}
		if storageDriver == "s3" && strings.TrimSpace(c.AWSS3Bucket) == "" {
			return errors.New("AWS_S3_BUCKET is required when STORAGE_DRIVER=s3")
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultValue
}

func getEnvAllowEmpty(key, defaultValue string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Invalid boolean value for %s; using %t", key, defaultValue)
		return defaultValue
	}
	return parsed
}

func (c *Config) String() string {
	return fmt.Sprintf("app=%s env=%s port=%s", c.AppName, c.Env, c.Port)
}
