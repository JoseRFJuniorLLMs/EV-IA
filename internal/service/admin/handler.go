package admin

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Handler handles admin HTTP requests
type Handler struct {
	service ports.AdminService
}

// NewHandler creates a new admin handler
func NewHandler(service ports.AdminService) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers admin routes
func (h *Handler) RegisterRoutes(app *fiber.App, authMiddleware, adminMiddleware fiber.Handler) {
	admin := app.Group("/api/v1/admin", authMiddleware, adminMiddleware)

	// Dashboard
	admin.Get("/dashboard", h.GetDashboard)
	admin.Get("/stats/revenue", h.GetRevenueStats)
	admin.Get("/stats/usage", h.GetUsageStats)

	// Users
	admin.Get("/users", h.GetUsers)
	admin.Get("/users/:id", h.GetUserDetails)
	admin.Patch("/users/:id/status", h.UpdateUserStatus)
	admin.Patch("/users/:id/role", h.UpdateUserRole)

	// Stations
	admin.Get("/stations", h.GetStations)
	admin.Get("/stations/:id", h.GetStationDetails)
	admin.Patch("/stations/:id/status", h.UpdateStationStatus)

	// Transactions
	admin.Get("/transactions", h.GetTransactions)
	admin.Get("/transactions/:id", h.GetTransactionDetails)

	// Alerts
	admin.Get("/alerts", h.GetAlerts)
	admin.Post("/alerts/:id/acknowledge", h.AcknowledgeAlert)

	// Reports
	admin.Get("/reports/:type", h.GenerateReport)
}

// GetDashboard handles GET /api/v1/admin/dashboard
func (h *Handler) GetDashboard(c *fiber.Ctx) error {
	stats, err := h.service.GetDashboardStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// GetRevenueStats handles GET /api/v1/admin/stats/revenue
func (h *Handler) GetRevenueStats(c *fiber.Ctx) error {
	startDate, endDate := parseDateRange(c)

	stats, err := h.service.GetRevenueStats(c.Context(), startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// GetUsageStats handles GET /api/v1/admin/stats/usage
func (h *Handler) GetUsageStats(c *fiber.Ctx) error {
	startDate, endDate := parseDateRange(c)

	stats, err := h.service.GetUsageStats(c.Context(), startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// GetUsers handles GET /api/v1/admin/users
func (h *Handler) GetUsers(c *fiber.Ctx) error {
	filter := ports.UserFilter{
		Status: c.Query("status"),
		Role:   c.Query("role"),
		Search: c.Query("search"),
	}
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	users, total, err := h.service.GetUsers(c.Context(), filter, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetUserDetails handles GET /api/v1/admin/users/:id
func (h *Handler) GetUserDetails(c *fiber.Ctx) error {
	userID := c.Params("id")

	details, err := h.service.GetUserDetails(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(details)
}

// UpdateUserStatus handles PATCH /api/v1/admin/users/:id/status
func (h *Handler) UpdateUserStatus(c *fiber.Ctx) error {
	userID := c.Params("id")

	var body struct {
		Status string `json:"status" validate:"required,oneof=Active Inactive Blocked"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateUserStatus(c.Context(), userID, body.Status); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User status updated",
	})
}

// UpdateUserRole handles PATCH /api/v1/admin/users/:id/role
func (h *Handler) UpdateUserRole(c *fiber.Ctx) error {
	userID := c.Params("id")

	var body struct {
		Role string `json:"role" validate:"required,oneof=admin operator user"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateUserRole(c.Context(), userID, domain.UserRole(body.Role)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User role updated",
	})
}

// GetStations handles GET /api/v1/admin/stations
func (h *Handler) GetStations(c *fiber.Ctx) error {
	filter := ports.StationFilter{
		Status: c.Query("status"),
		Vendor: c.Query("vendor"),
		Search: c.Query("search"),
	}
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	stations, total, err := h.service.GetStations(c.Context(), filter, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"stations": stations,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetStationDetails handles GET /api/v1/admin/stations/:id
func (h *Handler) GetStationDetails(c *fiber.Ctx) error {
	stationID := c.Params("id")

	details, err := h.service.GetStationDetails(c.Context(), stationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(details)
}

// UpdateStationStatus handles PATCH /api/v1/admin/stations/:id/status
func (h *Handler) UpdateStationStatus(c *fiber.Ctx) error {
	stationID := c.Params("id")

	var body struct {
		Status string `json:"status" validate:"required"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateStationStatus(c.Context(), stationID, domain.ChargePointStatus(body.Status)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Station status updated",
	})
}

// GetTransactions handles GET /api/v1/admin/transactions
func (h *Handler) GetTransactions(c *fiber.Ctx) error {
	filter := ports.TransactionFilter{
		Status:        c.Query("status"),
		UserID:        c.Query("user_id"),
		ChargePointID: c.Query("station_id"),
	}

	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			filter.StartDate = t
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			filter.EndDate = t
		}
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	transactions, total, err := h.service.GetTransactions(c.Context(), filter, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"transactions": transactions,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// GetTransactionDetails handles GET /api/v1/admin/transactions/:id
func (h *Handler) GetTransactionDetails(c *fiber.Ctx) error {
	txID := c.Params("id")

	details, err := h.service.GetTransactionDetails(c.Context(), txID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(details)
}

// GetAlerts handles GET /api/v1/admin/alerts
func (h *Handler) GetAlerts(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	alerts, err := h.service.GetAlerts(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"alerts": alerts,
		"limit":  limit,
		"offset": offset,
	})
}

// AcknowledgeAlert handles POST /api/v1/admin/alerts/:id/acknowledge
func (h *Handler) AcknowledgeAlert(c *fiber.Ctx) error {
	alertID := c.Params("id")

	if err := h.service.AcknowledgeAlert(c.Context(), alertID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Alert acknowledged",
	})
}

// GenerateReport handles GET /api/v1/admin/reports/:type
func (h *Handler) GenerateReport(c *fiber.Ctx) error {
	reportType := c.Params("type")
	startDate, endDate := parseDateRange(c)

	report, err := h.service.GenerateReport(c.Context(), reportType, startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set content type based on report type
	format := c.Query("format", "json")
	switch format {
	case "csv":
		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", "attachment; filename=report.csv")
	case "pdf":
		c.Set("Content-Type", "application/pdf")
		c.Set("Content-Disposition", "attachment; filename=report.pdf")
	default:
		c.Set("Content-Type", "application/json")
	}

	return c.Send(report)
}

// AdminMiddleware checks if user is admin
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		if role != "admin" && role != domain.UserRoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}
		return c.Next()
	}
}

// parseDateRange parses start and end dates from query parameters
func parseDateRange(c *fiber.Ctx) (time.Time, time.Time) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -30) // Default: last 30 days
	endDate := now

	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = t
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = t
		}
	}

	return startDate, endDate
}
