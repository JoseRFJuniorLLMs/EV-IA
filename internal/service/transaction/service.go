package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/seu-repo/sigec-ve/internal/adapter/queue"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Default pricing (should come from config in production)
const (
	defaultPricePerKWh = 0.75 // R$/kWh
	defaultCurrency    = "BRL"
)

type Service struct {
	repo          ports.TransactionRepository
	deviceService ports.DeviceService
	mq            queue.MessageQueue
	log           *zap.Logger
}

func NewService(repo ports.TransactionRepository, deviceService ports.DeviceService, mq queue.MessageQueue, log *zap.Logger) ports.TransactionService {
	return &Service{
		repo:          repo,
		deviceService: deviceService,
		mq:            mq,
		log:           log,
	}
}

func (s *Service) StartTransaction(ctx context.Context, deviceID string, connectorID int, userID string, idTag string) (*domain.Transaction, error) {
	// Check if device is available
	device, err := s.deviceService.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, errors.New("device not found")
	}

	// Check if device is available
	if device.Status != domain.ChargePointStatusAvailable {
		return nil, fmt.Errorf("device is not available, current status: %s", device.Status)
	}

	// Check if user already has an active transaction
	existingTx, _ := s.repo.FindActiveByUserID(ctx, userID)
	if existingTx != nil {
		return nil, errors.New("user already has an active charging session")
	}

	// Create transaction
	tx := &domain.Transaction{
		ID:            uuid.New().String(),
		ChargePointID: deviceID,
		ConnectorID:   connectorID,
		UserID:        userID,
		IdTag:         idTag,
		StartTime:     time.Now(),
		Status:        domain.TransactionStatusStarted,
		Currency:      defaultCurrency,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Save(ctx, tx); err != nil {
		return nil, err
	}

	// Update device status to Occupied
	if err := s.deviceService.UpdateStatus(ctx, deviceID, domain.ChargePointStatusOccupied); err != nil {
		s.log.Warn("Failed to update device status", zap.Error(err))
	}

	// Publish event
	if s.mq != nil {
		event := map[string]interface{}{
			"transaction_id": tx.ID,
			"device_id":      deviceID,
			"user_id":        userID,
			"start_time":     tx.StartTime.Format(time.RFC3339),
		}
		if data, err := json.Marshal(event); err == nil {
			if err := s.mq.Publish("transaction.started", data); err != nil {
				s.log.Warn("Failed to publish transaction started event", zap.Error(err))
			}
		}
	}

	s.log.Info("Transaction started",
		zap.String("tx_id", tx.ID),
		zap.String("device_id", deviceID),
		zap.String("user_id", userID),
	)

	return tx, nil
}

func (s *Service) StopTransaction(ctx context.Context, transactionID string) (*domain.Transaction, error) {
	tx, err := s.repo.FindByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, errors.New("transaction not found")
	}

	if tx.Status != domain.TransactionStatusStarted {
		return nil, fmt.Errorf("transaction is not active, current status: %s", tx.Status)
	}

	now := time.Now()
	tx.EndTime = &now
	tx.Status = domain.TransactionStatusStopped
	tx.UpdatedAt = now

	// Calculate energy and cost
	if tx.MeterStop > tx.MeterStart {
		tx.TotalEnergy = tx.MeterStop - tx.MeterStart
		tx.Cost = float64(tx.TotalEnergy) / 1000.0 * defaultPricePerKWh // Convert Wh to kWh
	}

	if err := s.repo.Update(ctx, tx); err != nil {
		return nil, err
	}

	// Update device status to Available
	if err := s.deviceService.UpdateStatus(ctx, tx.ChargePointID, domain.ChargePointStatusAvailable); err != nil {
		s.log.Warn("Failed to update device status", zap.Error(err))
	}

	// Publish event for billing (if message queue available)
	if s.mq != nil {
		event := map[string]interface{}{
			"transaction_id": tx.ID,
			"device_id":      tx.ChargePointID,
			"user_id":        tx.UserID,
			"total_energy":   tx.TotalEnergy,
			"cost":           tx.Cost,
			"currency":       tx.Currency,
			"end_time":       now.Format(time.RFC3339),
		}
		if data, err := json.Marshal(event); err == nil {
			if err := s.mq.Publish("transaction.completed", data); err != nil {
				s.log.Warn("Failed to publish transaction completed event", zap.Error(err))
			}
			if err := s.mq.Publish("billing.events", data); err != nil {
				s.log.Warn("Failed to publish billing event", zap.Error(err))
			}
		}
	}

	s.log.Info("Transaction stopped",
		zap.String("tx_id", tx.ID),
		zap.Int("energy_wh", tx.TotalEnergy),
		zap.Float64("cost", tx.Cost),
	)

	return tx, nil
}

func (s *Service) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) GetActiveTransaction(ctx context.Context, userID string) (*domain.Transaction, error) {
	return s.repo.FindActiveByUserID(ctx, userID)
}

func (s *Service) GetTransactionHistory(ctx context.Context, userID string) ([]domain.Transaction, error) {
	return s.repo.FindHistoryByUserID(ctx, userID)
}

// StartCharging starts a charging session for the voice assistant
// It finds an available connector on the specified station
func (s *Service) StartCharging(ctx context.Context, userID string, stationID string) (*domain.Transaction, error) {
	// If no station specified, find the nearest available one
	if stationID == "" {
		availableDevices, err := s.deviceService.ListAvailableDevices(ctx)
		if err != nil || len(availableDevices) == 0 {
			return nil, errors.New("no available charging stations found")
		}
		stationID = availableDevices[0].ID
	}

	// Use default connector 1
	return s.StartTransaction(ctx, stationID, 1, userID, userID)
}

// StopActiveCharging stops the active charging session for a user
func (s *Service) StopActiveCharging(ctx context.Context, userID string) error {
	tx, err := s.repo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if tx == nil {
		return errors.New("no active charging session found")
	}

	_, err = s.StopTransaction(ctx, tx.ID)
	return err
}

// GetCurrentSessionCost returns the estimated cost of the active session
func (s *Service) GetCurrentSessionCost(ctx context.Context, userID string) (float64, error) {
	tx, err := s.repo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if tx == nil {
		return 0, errors.New("no active charging session found")
	}

	// Calculate estimated cost based on time elapsed
	// In a real implementation, this would query the meter values
	elapsed := time.Since(tx.StartTime)
	estimatedKWh := elapsed.Hours() * 7.0 // Assume average 7kW charging rate
	estimatedCost := estimatedKWh * defaultPricePerKWh

	return estimatedCost, nil
}
