package v16

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Handlers processes OCPP 1.6 messages from charge points
type Handlers struct {
	deviceService ports.DeviceService
	txService     ports.TransactionService
	log           *zap.Logger
}

// NewHandlers creates a new OCPP 1.6 message handler
func NewHandlers(deviceService ports.DeviceService, txService ports.TransactionService, log *zap.Logger) *Handlers {
	return &Handlers{
		deviceService: deviceService,
		txService:     txService,
		log:           log,
	}
}

// HandleMessage routes an OCPP 1.6 action to the appropriate handler
func (h *Handlers) HandleMessage(chargePointID, action string, payload json.RawMessage) (interface{}, error) {
	ctx := context.Background()

	switch action {
	case "BootNotification":
		return h.handleBootNotification(ctx, chargePointID, payload)
	case "Heartbeat":
		return h.handleHeartbeat(ctx, chargePointID)
	case "StatusNotification":
		return h.handleStatusNotification(ctx, chargePointID, payload)
	case "StartTransaction":
		return h.handleStartTransaction(ctx, chargePointID, payload)
	case "StopTransaction":
		return h.handleStopTransaction(ctx, chargePointID, payload)
	case "MeterValues":
		return h.handleMeterValues(ctx, chargePointID, payload)
	case "Authorize":
		return h.handleAuthorize(ctx, chargePointID, payload)
	default:
		h.log.Warn("Unknown OCPP 1.6 action", zap.String("action", action))
		return map[string]string{}, nil
	}
}

// --- OCPP 1.6 Request/Response types ---

type bootNotificationReq struct {
	ChargePointVendor string `json:"chargePointVendor"`
	ChargePointModel  string `json:"chargePointModel"`
	ChargePointSerial string `json:"chargePointSerialNumber,omitempty"`
	FirmwareVersion   string `json:"firmwareVersion,omitempty"`
}

type bootNotificationResp struct {
	Status      string `json:"status"`
	CurrentTime string `json:"currentTime"`
	Interval    int    `json:"interval"`
}

func (h *Handlers) handleBootNotification(ctx context.Context, chargePointID string, payload json.RawMessage) (interface{}, error) {
	var req bootNotificationReq
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid BootNotification: %w", err)
	}

	h.log.Info("OCPP 1.6 BootNotification",
		zap.String("charge_point_id", chargePointID),
		zap.String("vendor", req.ChargePointVendor),
		zap.String("model", req.ChargePointModel),
	)

	if err := h.deviceService.UpdateStatus(ctx, chargePointID, domain.ChargePointStatusAvailable); err != nil {
		h.log.Warn("Failed to update device status on boot", zap.Error(err))
	}

	return bootNotificationResp{
		Status:      "Accepted",
		CurrentTime: time.Now().UTC().Format(time.RFC3339),
		Interval:    300,
	}, nil
}

func (h *Handlers) handleHeartbeat(ctx context.Context, chargePointID string) (interface{}, error) {
	h.log.Debug("OCPP 1.6 Heartbeat", zap.String("charge_point_id", chargePointID))

	return map[string]string{
		"currentTime": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

type statusNotificationReq struct {
	ConnectorId     int    `json:"connectorId"`
	ErrorCode       string `json:"errorCode"`
	Status          string `json:"status"`
	Timestamp       string `json:"timestamp,omitempty"`
	VendorErrorCode string `json:"vendorErrorCode,omitempty"`
}

func (h *Handlers) handleStatusNotification(ctx context.Context, chargePointID string, payload json.RawMessage) (interface{}, error) {
	var req statusNotificationReq
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid StatusNotification: %w", err)
	}

	h.log.Info("OCPP 1.6 StatusNotification",
		zap.String("charge_point_id", chargePointID),
		zap.Int("connector_id", req.ConnectorId),
		zap.String("status", req.Status),
		zap.String("error_code", req.ErrorCode),
	)

	// Map OCPP 1.6 status to internal status
	var internalStatus domain.ChargePointStatus
	switch req.Status {
	case "Available":
		internalStatus = domain.ChargePointStatusAvailable
	case "Occupied", "Charging", "SuspendedEV", "SuspendedEVSE":
		internalStatus = domain.ChargePointStatusOccupied
	case "Faulted":
		internalStatus = domain.ChargePointStatusFaulted
	case "Unavailable", "Reserved":
		internalStatus = domain.ChargePointStatusUnavailable
	default:
		internalStatus = domain.ChargePointStatusAvailable
	}

	if req.ConnectorId == 0 {
		if err := h.deviceService.UpdateStatus(ctx, chargePointID, internalStatus); err != nil {
			h.log.Warn("Failed to update status", zap.Error(err))
		}
	}

	return map[string]interface{}{}, nil
}

type startTransactionReq struct {
	ConnectorId   int    `json:"connectorId"`
	IdTag         string `json:"idTag"`
	MeterStart    int    `json:"meterStart"`
	Timestamp     string `json:"timestamp"`
	ReservationId *int   `json:"reservationId,omitempty"`
}

func (h *Handlers) handleStartTransaction(ctx context.Context, chargePointID string, payload json.RawMessage) (interface{}, error) {
	var req startTransactionReq
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid StartTransaction: %w", err)
	}

	h.log.Info("OCPP 1.6 StartTransaction",
		zap.String("charge_point_id", chargePointID),
		zap.Int("connector_id", req.ConnectorId),
		zap.String("id_tag", req.IdTag),
	)

	tx, err := h.txService.StartTransaction(ctx, chargePointID, req.ConnectorId, req.IdTag, req.IdTag)
	if err != nil {
		return map[string]interface{}{
			"transactionId": -1,
			"idTagInfo":     map[string]string{"status": "Invalid"},
		}, nil
	}

	return map[string]interface{}{
		"transactionId": tx.ID,
		"idTagInfo":     map[string]string{"status": "Accepted"},
	}, nil
}

type stopTransactionReq struct {
	TransactionId int    `json:"transactionId"`
	MeterStop     int    `json:"meterStop"`
	Timestamp     string `json:"timestamp"`
	IdTag         string `json:"idTag,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func (h *Handlers) handleStopTransaction(ctx context.Context, chargePointID string, payload json.RawMessage) (interface{}, error) {
	var req stopTransactionReq
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid StopTransaction: %w", err)
	}

	h.log.Info("OCPP 1.6 StopTransaction",
		zap.String("charge_point_id", chargePointID),
		zap.Int("transaction_id", req.TransactionId),
		zap.Int("meter_stop", req.MeterStop),
	)

	return map[string]interface{}{
		"idTagInfo": map[string]string{"status": "Accepted"},
	}, nil
}

func (h *Handlers) handleMeterValues(ctx context.Context, chargePointID string, payload json.RawMessage) (interface{}, error) {
	h.log.Debug("OCPP 1.6 MeterValues", zap.String("charge_point_id", chargePointID))
	return map[string]interface{}{}, nil
}

type authorizeReq struct {
	IdTag string `json:"idTag"`
}

func (h *Handlers) handleAuthorize(ctx context.Context, chargePointID string, payload json.RawMessage) (interface{}, error) {
	var req authorizeReq
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid Authorize: %w", err)
	}

	h.log.Info("OCPP 1.6 Authorize",
		zap.String("charge_point_id", chargePointID),
		zap.String("id_tag", req.IdTag),
	)

	return map[string]interface{}{
		"idTagInfo": map[string]string{"status": "Accepted"},
	}, nil
}
