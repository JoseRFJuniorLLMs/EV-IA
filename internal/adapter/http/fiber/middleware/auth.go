package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

func AuthRequired(service ports.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authorization header format"})
		}

		token := parts[1]
		user, err := service.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		c.Locals("user_id", user.ID)
		c.Locals("user_role", user.Role)
		c.Locals("user", user)

		return c.Next()
	}
}
