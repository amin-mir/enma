package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/amin-mir/enma/internal/auth"
)

func Protected(a *auth.Auth) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed token"})
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := a.ValidateAccessToken(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		c.Locals("user_id", claims.UserID)
		return c.Next()
	}
}
