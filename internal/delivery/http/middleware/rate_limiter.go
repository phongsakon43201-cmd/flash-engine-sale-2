package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func NewRateLimiter(maxRequests int, expirationSeconds int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: time.Duration(expirationSeconds) * time.Second,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Too many requests. Rate limit exceeded.",
			})
		},
	})
}
