package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// V2GHandler handles V2G (Vehicle-to-Grid) endpoints
type V2GHandler struct {
	v2gService       ports.V2GService
	gridPriceService ports.GridPriceService
	log              *zap.Logger
}

// NewV2GHandler creates a new V2G handler
func NewV2GHandler(
	v2gService ports.V2GService,
	gridPriceService ports.GridPriceService,
	log *zap.Logger,
) *V2GHandler {
	return &V2GHandler{
		v2gService:       v2gService,
		gridPriceService: gridPriceService,
		log:              log,
	}
}

// --- Discharge Operations ---

// StartDischargeRequest represents a V2G discharge start request
type StartDischargeRequest struct {
	ChargePointID string     `json:"charge_point_id"`
	ConnectorID   int        `json:"connector_id"`
	MaxPowerKW    float64    `json:"max_power_kw,omitempty"`
	MaxEnergyKWh  float64    `json:"max_energy_kwh,omitempty"`
	MinBatterySOC int        `json:"min_battery_soc,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
}

// StartDischarge handles POST /api/v1/v2g/discharge/start
func (h *V2GHandler) StartDischarge(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(string)

	var req StartDischargeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.ChargePointID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "charge_point_id is required",
		})
	}

	dischargeReq := &ports.V2GDischargeRequest{
		ChargePointID: req.ChargePointID,
		ConnectorID:   req.ConnectorID,
		UserID:        userID,
		MaxPowerKW:    req.MaxPowerKW,
		MaxEnergyKWh:  req.MaxEnergyKWh,
		MinBatterySOC: req.MinBatterySOC,
		EndTime:       req.EndTime,
	}

	session, err := h.v2gService.StartDischarge(c.Context(), dischargeReq)
	if err != nil {
		h.log.Error("Failed to start V2G discharge",
			zap.String("chargePointID", req.ChargePointID),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(session)
}

// StopDischargeRequest represents a V2G discharge stop request
type StopDischargeRequest struct {
	SessionID string `json:"session_id"`
}

// StopDischarge handles POST /api/v1/v2g/discharge/stop
func (h *V2GHandler) StopDischarge(c *fiber.Ctx) error {
	var req StopDischargeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.SessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id is required",
		})
	}

	err := h.v2gService.StopDischarge(c.Context(), req.SessionID)
	if err != nil {
		h.log.Error("Failed to stop V2G discharge",
			zap.String("sessionID", req.SessionID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Stopped",
		"message": "V2G discharge session stopped successfully",
	})
}

// --- Session Operations ---

// GetSession handles GET /api/v1/v2g/session/:id
func (h *V2GHandler) GetSession(c *fiber.Ctx) error {
	sessionID := c.Params("id")

	session, err := h.v2gService.GetSession(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(session)
}

// GetActiveSession handles GET /api/v1/v2g/session/active/:deviceId
func (h *V2GHandler) GetActiveSession(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")

	session, err := h.v2gService.GetActiveSession(c.Context(), deviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if session == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No active V2G session for this device",
		})
	}

	return c.JSON(session)
}

// --- V2G Capability ---

// GetCapability handles GET /api/v1/v2g/capability/:deviceId
func (h *V2GHandler) GetCapability(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")

	capability, err := h.v2gService.CheckV2GCapability(c.Context(), deviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(capability)
}

// --- Grid Price ---

// GetCurrentGridPrice handles GET /api/v1/v2g/grid-price
func (h *V2GHandler) GetCurrentGridPrice(c *fiber.Ctx) error {
	price, err := h.gridPriceService.GetCurrentPrice(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	isPeak, _ := h.gridPriceService.IsPeakHour(c.Context())

	return c.JSON(fiber.Map{
		"price":    price,
		"currency": "BRL",
		"unit":     "kWh",
		"is_peak":  isPeak,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetPriceForecast handles GET /api/v1/v2g/grid-price/forecast
func (h *V2GHandler) GetPriceForecast(c *fiber.Ctx) error {
	hours := c.QueryInt("hours", 24)
	if hours < 1 || hours > 48 {
		hours = 24
	}

	forecast, err := h.gridPriceService.GetPriceForecast(c.Context(), hours)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"forecast": forecast,
		"currency": "BRL",
		"unit":     "kWh",
		"hours":    hours,
	})
}

// --- User Preferences ---

// SetPreferencesRequest represents V2G preferences update
type SetPreferencesRequest struct {
	AutoDischarge   bool    `json:"auto_discharge"`
	MinGridPrice    float64 `json:"min_grid_price"`
	MaxDischargeKWh float64 `json:"max_discharge_kwh"`
	PreserveSOC     int     `json:"preserve_soc"`
	NotifyOnStart   bool    `json:"notify_on_start"`
	NotifyOnEnd     bool    `json:"notify_on_end"`
}

// GetPreferences handles GET /api/v1/v2g/preferences
func (h *V2GHandler) GetPreferences(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	prefs, err := h.v2gService.GetUserPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(prefs)
}

// SetPreferences handles POST /api/v1/v2g/preferences
func (h *V2GHandler) SetPreferences(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req SetPreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	prefs := &domain.V2GPreferences{
		UserID:          userID,
		AutoDischarge:   req.AutoDischarge,
		MinGridPrice:    req.MinGridPrice,
		MaxDischargeKWh: req.MaxDischargeKWh,
		PreserveSOC:     req.PreserveSOC,
		NotifyOnStart:   req.NotifyOnStart,
		NotifyOnEnd:     req.NotifyOnEnd,
	}

	err := h.v2gService.SetUserPreferences(c.Context(), userID, prefs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Updated",
		"message": "V2G preferences updated successfully",
	})
}

// --- Statistics ---

// GetUserStats handles GET /api/v1/v2g/stats
func (h *V2GHandler) GetUserStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	// Default to last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if start := c.Query("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = t
		}
	}

	if end := c.Query("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			endDate = t
		}
	}

	stats, err := h.v2gService.GetUserStats(c.Context(), userID, startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// --- Compensation ---

// CalculateCompensationRequest represents a compensation calculation request
type CalculateCompensationRequest struct {
	SessionID string `json:"session_id"`
}

// CalculateCompensation handles POST /api/v1/v2g/compensation/calculate
func (h *V2GHandler) CalculateCompensation(c *fiber.Ctx) error {
	var req CalculateCompensationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	session, err := h.v2gService.GetSession(c.Context(), req.SessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	compensation, err := h.v2gService.CalculateCompensation(c.Context(), session)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(compensation)
}

// --- Optimization ---

// OptimizeV2GRequest represents an optimization request
type OptimizeV2GRequest struct {
	ChargePointID string `json:"charge_point_id"`
}

// OptimizeV2G handles POST /api/v1/v2g/optimize
func (h *V2GHandler) OptimizeV2G(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req OptimizeV2GRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.ChargePointID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "charge_point_id is required",
		})
	}

	err := h.v2gService.OptimizeV2G(c.Context(), req.ChargePointID, userID)
	if err != nil {
		h.log.Error("V2G optimization failed",
			zap.String("chargePointID", req.ChargePointID),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Optimizing",
		"message": "V2G optimization started based on user preferences and grid prices",
	})
}
