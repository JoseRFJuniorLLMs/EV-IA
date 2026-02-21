package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	fibercors "github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/seu-repo/sigec-ve/pkg/config"
)

// NewCORS creates a CORS middleware from application config
func NewCORS(cfg config.CORSConfig) fiber.Handler {
	allowedOrigins := "*"
	if len(cfg.AllowedOrigins) > 0 {
		allowedOrigins = strings.Join(cfg.AllowedOrigins, ",")
	}

	allowedMethods := "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	if len(cfg.AllowedMethods) > 0 {
		allowedMethods = strings.Join(cfg.AllowedMethods, ",")
	}

	allowedHeaders := "Origin,Content-Type,Accept,Authorization,X-Request-ID"
	if len(cfg.AllowedHeaders) > 0 {
		allowedHeaders = strings.Join(cfg.AllowedHeaders, ",")
	}

	exposeHeaders := "Content-Length,Content-Range"
	if len(cfg.ExposeHeaders) > 0 {
		exposeHeaders = strings.Join(cfg.ExposeHeaders, ",")
	}

	maxAge := 86400 // 24 hours default
	if cfg.MaxAge > 0 {
		maxAge = cfg.MaxAge
	}

	return fibercors.New(fibercors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     allowedMethods,
		AllowHeaders:     allowedHeaders,
		ExposeHeaders:    exposeHeaders,
		AllowCredentials: cfg.Credentials,
		MaxAge:           maxAge,
	})
}

// DefaultCORS creates a CORS middleware with sensible defaults for development
func DefaultCORS() fiber.Handler {
	return NewCORS(config.CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		MaxAge:         int((24 * time.Hour).Seconds()),
		Credentials:    false,
	})
}
