package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func ErrorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		if code == fiber.StatusInternalServerError {
			log.Error("Internal Server Error", zap.Error(err), zap.String("path", c.Path()))
		}

		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}
