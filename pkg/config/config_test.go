package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validConfig() *Config {
	return &Config{
		Port:           "8080",
		Env:            "development",
		MongoURI:       "mongodb://localhost:27017",
		MongoDBName:    "flashsale",
		RedisAddr:      "localhost:6379",
		AWSSQSQueueURL: "http://localhost/queue",
		AllowedOrigins: "http://localhost:8080",
	}
}

func TestProductionRejectsDevelopmentAuthentication(t *testing.T) {
	cfg := validConfig()
	cfg.Env = "production"
	cfg.FirebaseDevMode = true
	assert.ErrorContains(t, cfg.Validate(), "FIREBASE_DEV_MODE")
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	cfg := validConfig()
	cfg.Env = "production"
	cfg.AllowedOrigins = "*"
	assert.ErrorContains(t, cfg.Validate(), "ALLOWED_ORIGINS")
}
