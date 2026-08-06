package middleware

import (
	"strings"

	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

func FirebaseAuthMiddleware(authUsecase usecase.AuthUsecase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.JSONError(c, fiber.StatusUnauthorized, "Unauthorized", "Missing Authorization header")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			return utils.JSONError(c, fiber.StatusUnauthorized, "Unauthorized", "Invalid Authorization header format")
		}

		claims, err := authUsecase.VerifyToken(c.Context(), tokenString)
		if err != nil {
			return utils.JSONError(c, fiber.StatusUnauthorized, "Unauthorized", err.Error())
		}

		// Store verified user claims in Fiber Context Locals
		c.Locals("userID", claims.UID)
		c.Locals("userEmail", claims.Email)
		c.Locals("userRole", claims.Role)

		return c.Next()
	}
}
