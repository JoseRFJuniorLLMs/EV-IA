package reservation

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Handler handles reservation HTTP requests
type Handler struct {
	service ports.ReservationService
}

// NewHandler creates a new reservation handler
func NewHandler(service ports.ReservationService) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers reservation routes
func (h *Handler) RegisterRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	reservations := app.Group("/api/v1/reservations", authMiddleware)

	reservations.Post("/", h.CreateReservation)
	reservations.Get("/", h.GetUserReservations)
	reservations.Get("/:id", h.GetReservation)
	reservations.Delete("/:id", h.CancelReservation)
	reservations.Post("/:id/confirm", h.ConfirmReservation)

	// Station availability
	app.Get("/api/v1/stations/:id/availability", h.GetStationAvailability)
	app.Get("/api/v1/stations/:id/reservations", authMiddleware, h.GetStationReservations)
}

// CreateReservationRequest represents the request body
type CreateReservationRequest struct {
	ChargePointID string    `json:"charge_point_id" validate:"required"`
	ConnectorID   int       `json:"connector_id" validate:"required,min=1"`
	StartTime     time.Time `json:"start_time" validate:"required"`
	Duration      int       `json:"duration" validate:"required,min=30,max=180"`
	Notes         string    `json:"notes"`
}

// CreateReservation handles POST /api/v1/reservations
func (h *Handler) CreateReservation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req CreateReservationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	reservation, err := h.service.CreateReservation(c.Context(), &ports.ReservationRequest{
		UserID:        userID,
		ChargePointID: req.ChargePointID,
		ConnectorID:   req.ConnectorID,
		StartTime:     req.StartTime,
		Duration:      req.Duration,
		Notes:         req.Notes,
	})

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(reservation)
}

// GetReservation handles GET /api/v1/reservations/:id
func (h *Handler) GetReservation(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(string)

	reservation, err := h.service.GetReservation(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if reservation == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Reservation not found",
		})
	}

	// Verify ownership (unless admin)
	if reservation.UserID != userID {
		role := c.Locals("role")
		if role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied",
			})
		}
	}

	return c.JSON(reservation)
}

// GetUserReservations handles GET /api/v1/reservations
func (h *Handler) GetUserReservations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	status := c.Query("status", "")
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	reservations, err := h.service.GetUserReservations(c.Context(), userID, status, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"reservations": reservations,
		"limit":        limit,
		"offset":       offset,
	})
}

// CancelReservation handles DELETE /api/v1/reservations/:id
func (h *Handler) CancelReservation(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(string)

	var body struct {
		Reason string `json:"reason"`
	}
	c.BodyParser(&body)

	if err := h.service.CancelReservation(c.Context(), id, userID, body.Reason); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Reservation cancelled successfully",
	})
}

// ConfirmReservation handles POST /api/v1/reservations/:id/confirm
func (h *Handler) ConfirmReservation(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.service.ConfirmReservation(c.Context(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Reservation confirmed",
	})
}

// GetStationAvailability handles GET /api/v1/stations/:id/availability
func (h *Handler) GetStationAvailability(c *fiber.Ctx) error {
	stationID := c.Params("id")
	dateStr := c.Query("date", time.Now().Format("2006-01-02"))

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid date format (use YYYY-MM-DD)",
		})
	}

	slots, err := h.service.GetAvailableSlots(c.Context(), stationID, date)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"station_id": stationID,
		"date":       dateStr,
		"slots":      slots,
	})
}

// GetStationReservations handles GET /api/v1/stations/:id/reservations
func (h *Handler) GetStationReservations(c *fiber.Ctx) error {
	stationID := c.Params("id")
	dateStr := c.Query("date", time.Now().Format("2006-01-02"))

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid date format (use YYYY-MM-DD)",
		})
	}

	reservations, err := h.service.GetStationReservations(c.Context(), stationID, date)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"station_id":   stationID,
		"date":         dateStr,
		"reservations": reservations,
	})
}
