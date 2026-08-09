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

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			return utils.JSONError(c, fiber.StatusUnauthorized, "Unauthorized", "Invalid Authorization header format")
		}
		tokenString := parts[1]

		claims, err := authUsecase.VerifyToken(c.Context(), tokenString)
		if err != nil {
			return utils.JSONError(c, fiber.StatusUnauthorized, "Unauthorized", "Invalid or expired authentication token")
		}

		// Store verified user claims in Fiber Context Locals
		c.Locals("userID", claims.UID)
		c.Locals("userEmail", claims.Email)
		c.Locals("userRole", claims.Role)

		return c.Next()
	}
}

func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("userRole").(string)
		role = strings.ToLower(strings.TrimSpace(role))
		if _, ok := allowed[role]; !ok {
			return utils.JSONError(c, fiber.StatusForbidden, "Forbidden", "Insufficient permissions")
		}
		return c.Next()
	}
}
