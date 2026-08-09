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

func TestMemoryQueueDoesNotRequireSQS(t *testing.T) {
	cfg := validConfig()
	cfg.QueueDriver = "memory"
	cfg.AWSSQSQueueURL = ""
	assert.NoError(t, cfg.Validate())
}

func TestManagedRedisURLDoesNotRequireAddress(t *testing.T) {
	cfg := validConfig()
	cfg.RedisURL = "rediss://default:secret@example.com:6380"
	cfg.RedisAddr = ""
	assert.NoError(t, cfg.Validate())
}

func TestRejectsUnsupportedRuntimeDrivers(t *testing.T) {
	cfg := validConfig()
	cfg.QueueDriver = "unknown"
	assert.ErrorContains(t, cfg.Validate(), "QUEUE_DRIVER")

	cfg = validConfig()
	cfg.StorageDriver = "filesystem"
	assert.ErrorContains(t, cfg.Validate(), "STORAGE_DRIVER")
}

func TestRenderHostnameBecomesDefaultCORSOrigin(t *testing.T) {
	t.Setenv("PORT", "10000")
	t.Setenv("RENDER_EXTERNAL_HOSTNAME", "flashsale-demo.onrender.com")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg := LoadConfig()
	assert.Equal(t, "https://flashsale-demo.onrender.com", cfg.AllowedOrigins)
}
