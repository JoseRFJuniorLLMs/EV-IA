package health

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// FiberHandler creates Fiber routes for health checks
type FiberHandler struct {
	service *Service
}

// NewFiberHandler creates a new Fiber health handler
func NewFiberHandler(service *Service) *FiberHandler {
	return &FiberHandler{service: service}
}

// RegisterRoutes registers health check routes
func (h *FiberHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/health", h.Health)
	app.Get("/healthz", h.Health)  // Kubernetes alias
	app.Get("/ready", h.Ready)
	app.Get("/readyz", h.Ready)    // Kubernetes alias
	app.Get("/live", h.Health)     // Kubernetes liveness
	app.Get("/livez", h.Health)    // Kubernetes alias
}

// Health handles the liveness probe
func (h *FiberHandler) Health(c *fiber.Ctx) error {
	response := h.service.Health(c.Context())
	return c.Status(fiber.StatusOK).JSON(response)
}

// Ready handles the readiness probe
func (h *FiberHandler) Ready(c *fiber.Ctx) error {
	response := h.service.Ready(c.Context())

	status := fiber.StatusOK
	if !response.Ready {
		status = fiber.StatusServiceUnavailable
	}

	return c.Status(status).JSON(response)
}

// HTTPHandler creates standard HTTP handlers for health checks
type HTTPHandler struct {
	service *Service
}

// NewHTTPHandler creates a new HTTP health handler
func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// RegisterRoutes registers health check routes on a ServeMux
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/healthz", h.Health)
	mux.HandleFunc("/ready", h.Ready)
	mux.HandleFunc("/readyz", h.Ready)
	mux.HandleFunc("/live", h.Health)
	mux.HandleFunc("/livez", h.Health)
}

// Health handles the liveness probe
func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := h.service.Health(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Ready handles the readiness probe
func (h *HTTPHandler) Ready(w http.ResponseWriter, r *http.Request) {
	response := h.service.Ready(r.Context())

	w.Header().Set("Content-Type", "application/json")

	if response.Ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// Middleware creates a health check middleware that can short-circuit requests
// when the service is not ready
func Middleware(service *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip health check endpoints
		path := c.Path()
		if path == "/health" || path == "/healthz" ||
		   path == "/ready" || path == "/readyz" ||
		   path == "/live" || path == "/livez" {
			return c.Next()
		}

		// Check readiness for other endpoints
		response := service.Ready(c.Context())
		if !response.Ready {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "service unavailable",
				"message": "service is not ready to accept requests",
			})
		}

		return c.Next()
	}
}
