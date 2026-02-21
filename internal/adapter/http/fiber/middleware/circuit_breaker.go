package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// CircuitBreakerConfig holds configuration for the circuit breaker middleware
type CircuitBreakerConfig struct {
	Logger      *zap.Logger
	Name        string
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Logger:      zap.NewNop(), // Safe no-op logger as default
		Name:        "sigec-api",
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
	}
}

// CircuitBreaker creates a circuit breaker middleware with default config
func CircuitBreaker() fiber.Handler {
	return CircuitBreakerWithConfig(DefaultCircuitBreakerConfig())
}

// CircuitBreakerWithLogger creates a circuit breaker middleware with a specific logger
func CircuitBreakerWithLogger(log *zap.Logger) fiber.Handler {
	cfg := DefaultCircuitBreakerConfig()
	if log != nil {
		cfg.Logger = log
	}
	return CircuitBreakerWithConfig(cfg)
}

// CircuitBreakerWithConfig creates a circuit breaker middleware with custom config
func CircuitBreakerWithConfig(cfg CircuitBreakerConfig) fiber.Handler {
	// Ensure logger is never nil
	log := cfg.Logger
	if log == nil {
		log = zap.NewNop()
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests == 0 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Warn("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	})

	return func(c *fiber.Ctx) error {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, c.Next()
		})

		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			log.Warn("Circuit breaker rejecting request",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.String("state", cb.State().String()),
				zap.Error(err),
			)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Service temporarily unavailable",
			})
		}

		if err != nil {
			log.Error("Request failed through circuit breaker",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.Error(err),
			)
		}

		return err
	}
}
