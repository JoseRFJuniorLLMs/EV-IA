package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// Global logger variable, assuming it is initialized elsewhere or should be passed.
var logger *zap.Logger

func CircuitBreaker() fiber.Handler {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "sigec-api",
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			if logger != nil {
				logger.Warn("Circuit breaker state changed",
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			}
		},
	})

	return func(c *fiber.Ctx) error {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, c.Next()
		})

		if err == gobreaker.ErrOpenState {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Service temporarily unavailable",
			})
		}

		return err
	}
}
