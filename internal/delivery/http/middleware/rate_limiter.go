package middleware

import (
	"strconv"
	"time"

	"flashsale-go/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func NewRateLimiter(maxRequests int, expirationSeconds int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: time.Duration(expirationSeconds) * time.Second,
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", strconv.Itoa(expirationSeconds))
			return utils.JSONError(c, fiber.StatusTooManyRequests, "Too many requests", "Rate limit exceeded for your IP address. Please retry shortly.")
		},
	})
}
