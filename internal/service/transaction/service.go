package transaction

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/seu-repo/sigec-ve/internal/adapter/queue"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
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
	// Check connector status (simplified)

	// Create transaction
	tx := &domain.Transaction{
		ID:            uuid.New().String(),
		ChargePointID: deviceID,
		ConnectorID:   connectorID,
		UserID:        userID,
		IdTag:         idTag,
		StartTime:     time.Now(),
		Status:        domain.TransactionStatusStarted,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Save(ctx, tx); err != nil {
		return nil, err
	}

	// Publish event
	// s.mq.Publish("transaction.started", ...)

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

	now := time.Now()
	tx.EndTime = &now
	tx.Status = domain.TransactionStatusStopped
	tx.UpdatedAt = now

	if err := s.repo.Update(ctx, tx); err != nil {
		return nil, err
	}

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
