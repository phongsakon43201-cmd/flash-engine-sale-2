package redis

import (
	"fmt"
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

// NewClient accepts either a managed Redis URL or a host/password pair.
func NewClient(rawURL, addr, password string) (*goredis.Client, error) {
	if strings.TrimSpace(rawURL) != "" {
		options, err := goredis.ParseURL(rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_URL: %w", err)
		}
		return goredis.NewClient(options), nil
	}

	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("REDIS_ADDR is required when REDIS_URL is not set")
	}

	return goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	}), nil
}
