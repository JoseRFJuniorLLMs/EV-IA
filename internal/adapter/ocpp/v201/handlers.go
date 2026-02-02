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

		tx, err := s.txService.StartTransaction(ctx, cpID, connID, uID, idTag)
		if err != nil {
			s.log.Error("Failed to start transaction", zap.Error(err))
			return &TransactionEventResponse{
				IdTokenInfo: &IdTokenInfo{Status: "Blocked"},
			}, nil
		}

		s.log.Info("Transaction Started via OCPP",
			zap.String("txID", tx.ID),
			zap.String("chargePointID", cpID),
			zap.String("userID", uID),
		)

	case "Updated":
		// Handle meter values update during charging
		if req.TransactionInfo != nil && req.MeterValue != nil {
			s.log.Info("Transaction Updated - Meter Values",
				zap.String("txID", req.TransactionInfo.TransactionId),
				zap.Any("meterValues", req.MeterValue),
			)
			// In production, update meter values in the transaction record
		}

	case "Ended":
		txID := req.TransactionInfo.TransactionId
		s.log.Info("Processing Transaction End", zap.String("txID", txID), zap.String("chargePointID", cpID))

		// Try to find the transaction by the OCPP transaction ID
		// If not found, try to find the active transaction for this charge point
		tx, err := s.txService.GetTransaction(ctx, txID)
		if err != nil || tx == nil {
			// Fallback: find active transaction for this user/device
			s.log.Warn("Transaction not found by ID, attempting to find active transaction",
				zap.String("txID", txID),
				zap.String("chargePointID", cpID),
			)

			// Get user ID from IdToken if available
			userID := "unknown"
			if req.IdToken != nil {
				userID = req.IdToken.IdToken
			}

			// Try to stop any active charging for this user
			if err := s.txService.StopActiveCharging(ctx, userID); err != nil {
				s.log.Error("Failed to stop active charging", zap.Error(err))
			} else {
				s.log.Info("Transaction Ended via StopActiveCharging",
					zap.String("userID", userID),
					zap.String("chargePointID", cpID),
				)
			}
		} else {
			// Stop the specific transaction
			stoppedTx, err := s.txService.StopTransaction(ctx, txID)
			if err != nil {
				s.log.Error("Failed to stop transaction", zap.Error(err), zap.String("txID", txID))
			} else {
				s.log.Info("Transaction Ended via OCPP",
					zap.String("txID", stoppedTx.ID),
					zap.Int("totalEnergy", stoppedTx.TotalEnergy),
					zap.Float64("cost", stoppedTx.Cost),
				)
			}
		}
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
