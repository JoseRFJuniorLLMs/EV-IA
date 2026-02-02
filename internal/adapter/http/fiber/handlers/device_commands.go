package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

// DeviceCommandHandler handles OCPP device command endpoints
type DeviceCommandHandler struct {
	ocppService     ports.OCPPCommandService
	firmwareService ports.FirmwareService
	log             *zap.Logger
}

// NewDeviceCommandHandler creates a new device command handler
func NewDeviceCommandHandler(
	ocppService ports.OCPPCommandService,
	firmwareService ports.FirmwareService,
	log *zap.Logger,
) *DeviceCommandHandler {
	return &DeviceCommandHandler{
		ocppService:     ocppService,
		firmwareService: firmwareService,
		log:             log,
	}
}

// --- Remote Start/Stop ---

// RemoteStartRequest represents a remote start request
type RemoteStartRequest struct {
	IdToken     string `json:"id_token"`
	EvseID      *int   `json:"evse_id,omitempty"`
	ConnectorID *int   `json:"connector_id,omitempty"`
}

// RemoteStart handles POST /api/v1/devices/:id/remote-start
func (h *DeviceCommandHandler) RemoteStart(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var req RemoteStartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.IdToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id_token is required",
		})
	}

	// Check if device is connected
	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	err := h.ocppService.RemoteStartTransaction(c.Context(), deviceID, req.IdToken, req.EvseID)
	if err != nil {
		h.log.Error("Remote start failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Accepted",
		"message": "Remote start command sent successfully",
	})
}

// RemoteStopRequest represents a remote stop request
type RemoteStopRequest struct {
	TransactionID string `json:"transaction_id"`
}

// RemoteStop handles POST /api/v1/devices/:id/remote-stop
func (h *DeviceCommandHandler) RemoteStop(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var req RemoteStopRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.TransactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "transaction_id is required",
		})
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	err := h.ocppService.RemoteStopTransaction(c.Context(), deviceID, req.TransactionID)
	if err != nil {
		h.log.Error("Remote stop failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Accepted",
		"message": "Remote stop command sent successfully",
	})
}

// --- Reset ---

// ResetRequest represents a reset request
type ResetRequest struct {
	Type   string `json:"type"` // Immediate, OnIdle
	EvseID *int   `json:"evse_id,omitempty"`
}

// Reset handles POST /api/v1/devices/:id/reset
func (h *DeviceCommandHandler) Reset(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var req ResetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Type == "" {
		req.Type = "Immediate"
	}

	if req.Type != "Immediate" && req.Type != "OnIdle" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "type must be 'Immediate' or 'OnIdle'",
		})
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	err := h.ocppService.Reset(c.Context(), deviceID, req.Type, req.EvseID)
	if err != nil {
		h.log.Error("Reset failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Accepted",
		"message": "Reset command sent successfully",
	})
}

// --- Trigger Message ---

// TriggerMessage handles POST /api/v1/devices/:id/trigger/:message
func (h *DeviceCommandHandler) TriggerMessage(c *fiber.Ctx) error {
	deviceID := c.Params("id")
	message := c.Params("message")

	validMessages := map[string]bool{
		"BootNotification":           true,
		"Heartbeat":                  true,
		"StatusNotification":         true,
		"MeterValues":                true,
		"FirmwareStatusNotification": true,
		"LogStatusNotification":      true,
	}

	if !validMessages[message] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":          "Invalid message type",
			"valid_messages": []string{"BootNotification", "Heartbeat", "StatusNotification", "MeterValues", "FirmwareStatusNotification", "LogStatusNotification"},
		})
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	var evseID *int
	if evse := c.QueryInt("evse_id", 0); evse > 0 {
		evseID = &evse
	}

	err := h.ocppService.TriggerMessage(c.Context(), deviceID, message, evseID)
	if err != nil {
		h.log.Error("Trigger message failed",
			zap.String("deviceID", deviceID),
			zap.String("message", message),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Accepted",
		"message": "Trigger command sent successfully",
	})
}

// --- Charging Profile ---

// SetChargingProfileRequest represents a charging profile request
type SetChargingProfileRequest struct {
	EvseID          int              `json:"evse_id"`
	ChargingProfile *ChargingProfile `json:"charging_profile"`
}

// ChargingProfile for REST API
type ChargingProfile struct {
	ID                     int                `json:"id"`
	StackLevel             int                `json:"stack_level"`
	ChargingProfilePurpose string             `json:"purpose"` // ChargePointMaxProfile, TxDefaultProfile, TxProfile
	ChargingProfileKind    string             `json:"kind"`    // Absolute, Recurring, Relative
	ValidFrom              *string            `json:"valid_from,omitempty"`
	ValidTo                *string            `json:"valid_to,omitempty"`
	ChargingSchedule       []ChargingSchedule `json:"charging_schedule"`
}

// ChargingSchedule for REST API
type ChargingSchedule struct {
	Duration         *int                      `json:"duration,omitempty"`
	StartSchedule    *string                   `json:"start_schedule,omitempty"`
	ChargingRateUnit string                    `json:"charging_rate_unit"` // W, A
	Periods          []ChargingSchedulePeriod  `json:"periods"`
}

// ChargingSchedulePeriod for REST API
type ChargingSchedulePeriod struct {
	StartPeriod  int     `json:"start_period"`
	Limit        float64 `json:"limit"`
	NumberPhases *int    `json:"number_phases,omitempty"`
}

// SetChargingProfile handles POST /api/v1/devices/:id/charging-profile
func (h *DeviceCommandHandler) SetChargingProfile(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var req SetChargingProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.ChargingProfile == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "charging_profile is required",
		})
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	err := h.ocppService.SetChargingProfile(c.Context(), deviceID, req.EvseID, req.ChargingProfile)
	if err != nil {
		h.log.Error("Set charging profile failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Accepted",
		"message": "Charging profile set successfully",
	})
}

// ClearChargingProfile handles DELETE /api/v1/devices/:id/charging-profile
func (h *DeviceCommandHandler) ClearChargingProfile(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var profileID *int
	if pid := c.QueryInt("profile_id", 0); pid > 0 {
		profileID = &pid
	}

	var evseID *int
	if eid := c.QueryInt("evse_id", 0); eid > 0 {
		evseID = &eid
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	err := h.ocppService.ClearChargingProfile(c.Context(), deviceID, profileID, evseID)
	if err != nil {
		h.log.Error("Clear charging profile failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Accepted",
		"message": "Charging profile cleared successfully",
	})
}

// --- Unlock Connector ---

// UnlockConnectorRequest represents an unlock request
type UnlockConnectorRequest struct {
	EvseID      int `json:"evse_id"`
	ConnectorID int `json:"connector_id"`
}

// UnlockConnector handles POST /api/v1/devices/:id/unlock
func (h *DeviceCommandHandler) UnlockConnector(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var req UnlockConnectorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.EvseID == 0 || req.ConnectorID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "evse_id and connector_id are required",
		})
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	err := h.ocppService.UnlockConnector(c.Context(), deviceID, req.EvseID, req.ConnectorID)
	if err != nil {
		h.log.Error("Unlock connector failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Unlocked",
		"message": "Connector unlock command sent successfully",
	})
}

// --- Change Availability ---

// ChangeAvailabilityRequest represents availability change request
type ChangeAvailabilityRequest struct {
	OperationalStatus string `json:"operational_status"` // Operative, Inoperative
	EvseID            *int   `json:"evse_id,omitempty"`
}

// ChangeAvailability handles POST /api/v1/devices/:id/availability
func (h *DeviceCommandHandler) ChangeAvailability(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var req ChangeAvailabilityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.OperationalStatus != "Operative" && req.OperationalStatus != "Inoperative" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "operational_status must be 'Operative' or 'Inoperative'",
		})
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	err := h.ocppService.ChangeAvailability(c.Context(), deviceID, req.OperationalStatus, req.EvseID)
	if err != nil {
		h.log.Error("Change availability failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Accepted",
		"message": "Availability change command sent successfully",
	})
}

// --- Firmware ---

// UpdateFirmwareRequest represents a firmware update request
type UpdateFirmwareRequest struct {
	FirmwareURL      string     `json:"firmware_url"`
	Version          string     `json:"version"`
	RetrieveDateTime *time.Time `json:"retrieve_datetime,omitempty"`
	InstallDateTime  *time.Time `json:"install_datetime,omitempty"`
	Retries          *int       `json:"retries,omitempty"`
	RetryInterval    *int       `json:"retry_interval,omitempty"`
}

// UpdateFirmware handles POST /api/v1/devices/:id/firmware/update
func (h *DeviceCommandHandler) UpdateFirmware(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	var req UpdateFirmwareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.FirmwareURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "firmware_url is required",
		})
	}

	if !h.ocppService.IsConnected(deviceID) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Device is not connected",
		})
	}

	fwReq := &ports.FirmwareUpdateRequest{
		ChargePointID:    deviceID,
		FirmwareURL:      req.FirmwareURL,
		Version:          req.Version,
		RetrieveDateTime: req.RetrieveDateTime,
		InstallDateTime:  req.InstallDateTime,
		Retries:          req.Retries,
		RetryInterval:    req.RetryInterval,
	}

	status, err := h.firmwareService.UpdateFirmware(c.Context(), fwReq)
	if err != nil {
		h.log.Error("Firmware update failed",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(status)
}

// GetFirmwareStatus handles GET /api/v1/devices/:id/firmware/status
func (h *DeviceCommandHandler) GetFirmwareStatus(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	status, err := h.firmwareService.GetFirmwareStatus(c.Context(), deviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if status == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No firmware update in progress",
		})
	}

	return c.JSON(status)
}

// CancelFirmwareUpdate handles DELETE /api/v1/devices/:id/firmware/update
func (h *DeviceCommandHandler) CancelFirmwareUpdate(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	err := h.firmwareService.CancelFirmwareUpdate(c.Context(), deviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "Cancelled",
		"message": "Firmware update cancelled",
	})
}

// --- Connection Status ---

// GetConnectionStatus handles GET /api/v1/devices/:id/connection
func (h *DeviceCommandHandler) GetConnectionStatus(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	isConnected := h.ocppService.IsConnected(deviceID)

	return c.JSON(fiber.Map{
		"device_id":    deviceID,
		"connected":    isConnected,
		"protocol":     "OCPP 2.0.1",
	})
}

// GetConnectedDevices handles GET /api/v1/devices/connected
func (h *DeviceCommandHandler) GetConnectedDevices(c *fiber.Ctx) error {
	clients := h.ocppService.GetConnectedClients()

	return c.JSON(fiber.Map{
		"count":   len(clients),
		"devices": clients,
	})
}
