package v201

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// handleAction routes the message to the appropriate handler based on the Action field
func (s *Server) handleAction(chargePointID string, msgID string, action string, payload []byte) {
	var err error
	var responsePayload interface{}

	s.log.Info("Handling OCPP Action", zap.String("action", action), zap.String("chargePointID", chargePointID))

	switch action {
	case "BootNotification":
		responsePayload, err = s.handleBootNotification(payload)
	case "Heartbeat":
		responsePayload, err = s.handleHeartbeat(payload)
	case "StatusNotification":
		responsePayload, err = s.handleStatusNotification(chargePointID, payload)
	case "TransactionEvent":
		responsePayload, err = s.handleTransactionEvent(chargePointID, payload)
	default:
		s.sendError(chargePointID, msgID, "NotImplemented", fmt.Sprintf("Action %s not implemented", action), nil)
		return
	}

	if err != nil {
		s.log.Error("Error handling action", zap.String("action", action), zap.Error(err))
		s.sendError(chargePointID, msgID, "InternalError", "An internal error occurred", nil)
		return
	}

	s.sendCallResult(chargePointID, msgID, responsePayload)
}

func (s *Server) handleBootNotification(payload []byte) (*BootNotificationResponse, error) {
	var req BootNotificationRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	// Logic: Register or update charge point in DB (via DeviceService ideally, but keeping it simple here)
	s.log.Info("BootNotification received", zap.String("vendor", req.ChargingStation.VendorName), zap.String("model", req.ChargingStation.Model))

	// In a real scenario, we would validate credentials here.

	return &BootNotificationResponse{
		CurrentTime: time.Now().Format(time.RFC3339),
		Interval:    300,        // 5 minutes heartbeat
		Status:      "Accepted", // Accepted, Pending, Rejected
	}, nil
}

func (s *Server) handleHeartbeat(payload []byte) (*HeartbeatResponse, error) {
	// Heartbeat acts as a keep-alive
	// Update last seen status in DeviceService

	return &HeartbeatResponse{
		CurrentTime: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Server) handleStatusNotification(cpID string, payload []byte) (*StatusNotificationResponse, error) {
	var req StatusNotificationRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	s.log.Info("Status Notification", zap.String("cpID", cpID), zap.String("status", req.ConnectorStatus))

	// Map OCPP status to Domain status
	status := domain.ChargePointStatus(req.ConnectorStatus) // Simplified mapping

	// Update device status in DB via Service
	ctx := context.Background()
	_ = s.deviceService.UpdateStatus(ctx, cpID, status)

	return &StatusNotificationResponse{}, nil
}

func (s *Server) handleTransactionEvent(cpID string, payload []byte) (*TransactionEventResponse, error) {
	var req TransactionEventRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	ctx := context.Background()

	switch req.EventType {
	case "Started":
		// User ID from IdToken
		uID := "unknown"
		idTag := ""
		if req.IdToken != nil {
			idTag = req.IdToken.IdToken
			// In production, verify auth cache or service to get real UserUUID
			uID = idTag // simplified
		}

		connID := 1
		if req.Evse != nil {
			connID = req.Evse.ConnectorId
		}

		_, err := s.txService.StartTransaction(ctx, cpID, connID, uID, idTag)
		if err != nil {
			s.log.Error("Failed to start transaction", zap.Error(err))
			// Even if local start fails, we might acknowledge but with Blocked status
		}

	case "Ended":
		txID := req.TransactionInfo.TransactionId
		// Find transaction by internal ID mapped to device ID or use device generated ID
		// For syncing, we might use remote transaction ID logic.
		// Detailed implementation is complex. Simplified:

		// Typically we'd need to look up the transaction that matches this device's transactionId
		// Here assuming req.TransactionInfo.TransactionId maps to our UUID if we sent it, OR we rely on GetActiveTransaction

		// s.txService.StopTransaction(ctx, txID)
		s.log.Info("Transaction Ended", zap.String("txID", txID))
	}

	return &TransactionEventResponse{
		IdTokenInfo: &IdTokenInfo{Status: "Accepted"},
	}, nil
}

func (s *Server) sendCallResult(id string, msgID string, payload interface{}) {
	response := []interface{}{CallResult, msgID, payload}
	data, _ := json.Marshal(response)
	s.Send(id, data)
}

func (s *Server) sendError(id string, msgID string, code string, desc string, details interface{}) {
	response := []interface{}{CallError, msgID, code, desc, details}
	data, _ := json.Marshal(response)
	s.Send(id, data)
}
