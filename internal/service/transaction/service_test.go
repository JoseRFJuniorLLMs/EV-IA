package transaction

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/mocks"
)

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestStartTransaction_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "device-123"
	userID := "user-456"

	mockDevice := &domain.ChargePoint{
		ID:     deviceID,
		Status: domain.ChargePointStatusAvailable,
	}

	var savedTx *domain.Transaction

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, userID string) (*domain.Transaction, error) {
			return nil, nil // No active transaction
		},
		SaveFunc: func(ctx context.Context, tx *domain.Transaction) error {
			savedTx = tx
			return nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{
		GetDeviceFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			if id == deviceID {
				return mockDevice, nil
			}
			return nil, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.ChargePointStatus) error {
			return nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	tx, err := service.StartTransaction(ctx, deviceID, 1, userID, "rfid-tag")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tx == nil {
		t.Fatal("expected transaction, got nil")
	}
	if tx.ChargePointID != deviceID {
		t.Errorf("expected device ID '%s', got '%s'", deviceID, tx.ChargePointID)
	}
	if tx.UserID != userID {
		t.Errorf("expected user ID '%s', got '%s'", userID, tx.UserID)
	}
	if tx.Status != domain.TransactionStatusStarted {
		t.Errorf("expected status 'Started', got '%s'", tx.Status)
	}
	if savedTx == nil {
		t.Error("expected transaction to be saved")
	}

	// Check event was published
	messages := mockQueue.GetPublishedMessages("transaction.started")
	if len(messages) != 1 {
		t.Errorf("expected 1 message published, got %d", len(messages))
	}
}

func TestStartTransaction_DeviceNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockTxRepo := &mocks.MockTransactionRepository{}

	mockDeviceService := &mocks.MockDeviceService{
		GetDeviceFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return nil, nil // Device not found
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	_, err := service.StartTransaction(ctx, "nonexistent", 1, "user-123", "rfid")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "device not found" {
		t.Errorf("expected 'device not found', got '%s'", err.Error())
	}
}

func TestStartTransaction_DeviceNotAvailable(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockDevice := &domain.ChargePoint{
		ID:     "device-123",
		Status: domain.ChargePointStatusOccupied, // Not available
	}

	mockTxRepo := &mocks.MockTransactionRepository{}

	mockDeviceService := &mocks.MockDeviceService{
		GetDeviceFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return mockDevice, nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	_, err := service.StartTransaction(ctx, "device-123", 1, "user-123", "rfid")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStartTransaction_UserAlreadyCharging(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockDevice := &domain.ChargePoint{
		ID:     "device-123",
		Status: domain.ChargePointStatusAvailable,
	}

	existingTx := &domain.Transaction{
		ID:     "existing-tx",
		UserID: "user-123",
		Status: domain.TransactionStatusStarted,
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, userID string) (*domain.Transaction, error) {
			return existingTx, nil // User already has active transaction
		},
	}

	mockDeviceService := &mocks.MockDeviceService{
		GetDeviceFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return mockDevice, nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	_, err := service.StartTransaction(ctx, "device-123", 1, "user-123", "rfid")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "user already has an active charging session" {
		t.Errorf("expected 'user already has an active charging session', got '%s'", err.Error())
	}
}

func TestStopTransaction_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	txID := "tx-123"

	existingTx := &domain.Transaction{
		ID:            txID,
		ChargePointID: "device-123",
		UserID:        "user-456",
		Status:        domain.TransactionStatusStarted,
		StartTime:     time.Now().Add(-30 * time.Minute),
		MeterStart:    0,
		MeterStop:     10000, // 10 kWh
	}

	var updatedTx *domain.Transaction

	mockTxRepo := &mocks.MockTransactionRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.Transaction, error) {
			if id == txID {
				return existingTx, nil
			}
			return nil, nil
		},
		UpdateFunc: func(ctx context.Context, tx *domain.Transaction) error {
			updatedTx = tx
			return nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.ChargePointStatus) error {
			return nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	tx, err := service.StopTransaction(ctx, txID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tx == nil {
		t.Fatal("expected transaction, got nil")
	}
	if tx.Status != domain.TransactionStatusStopped {
		t.Errorf("expected status 'Stopped', got '%s'", tx.Status)
	}
	if tx.EndTime == nil {
		t.Error("expected EndTime to be set")
	}
	if updatedTx == nil {
		t.Error("expected transaction to be updated")
	}

	// Check events were published
	completedMessages := mockQueue.GetPublishedMessages("transaction.completed")
	if len(completedMessages) != 1 {
		t.Errorf("expected 1 transaction.completed message, got %d", len(completedMessages))
	}

	billingMessages := mockQueue.GetPublishedMessages("billing.events")
	if len(billingMessages) != 1 {
		t.Errorf("expected 1 billing.events message, got %d", len(billingMessages))
	}
}

func TestStopTransaction_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockTxRepo := &mocks.MockTransactionRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.Transaction, error) {
			return nil, nil // Not found
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	_, err := service.StopTransaction(ctx, "nonexistent")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "transaction not found" {
		t.Errorf("expected 'transaction not found', got '%s'", err.Error())
	}
}

func TestStopTransaction_AlreadyStopped(t *testing.T) {
	// Arrange
	ctx := context.Background()

	stoppedTx := &domain.Transaction{
		ID:     "tx-123",
		Status: domain.TransactionStatusStopped, // Already stopped
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.Transaction, error) {
			return stoppedTx, nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	_, err := service.StopTransaction(ctx, "tx-123")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTransaction_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	txID := "tx-123"

	expectedTx := &domain.Transaction{
		ID:     txID,
		UserID: "user-456",
		Status: domain.TransactionStatusStarted,
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.Transaction, error) {
			if id == txID {
				return expectedTx, nil
			}
			return nil, nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	tx, err := service.GetTransaction(ctx, txID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tx == nil {
		t.Fatal("expected transaction, got nil")
	}
	if tx.ID != txID {
		t.Errorf("expected ID '%s', got '%s'", txID, tx.ID)
	}
}

func TestGetActiveTransaction_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := "user-123"

	activeTx := &domain.Transaction{
		ID:     "tx-active",
		UserID: userID,
		Status: domain.TransactionStatusStarted,
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			if uid == userID {
				return activeTx, nil
			}
			return nil, nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	tx, err := service.GetActiveTransaction(ctx, userID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tx == nil {
		t.Fatal("expected transaction, got nil")
	}
	if tx.UserID != userID {
		t.Errorf("expected user ID '%s', got '%s'", userID, tx.UserID)
	}
}

func TestGetTransactionHistory_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := "user-123"

	history := []domain.Transaction{
		{ID: "tx-1", UserID: userID, Status: domain.TransactionStatusCompleted},
		{ID: "tx-2", UserID: userID, Status: domain.TransactionStatusCompleted},
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindHistoryByUserIDFunc: func(ctx context.Context, uid string) ([]domain.Transaction, error) {
			if uid == userID {
				return history, nil
			}
			return []domain.Transaction{}, nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	txs, err := service.GetTransactionHistory(ctx, userID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(txs))
	}
}

func TestStartCharging_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := "user-123"
	stationID := "station-456"

	mockDevice := &domain.ChargePoint{
		ID:     stationID,
		Status: domain.ChargePointStatusAvailable,
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			return nil, nil
		},
		SaveFunc: func(ctx context.Context, tx *domain.Transaction) error {
			return nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{
		GetDeviceFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return mockDevice, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.ChargePointStatus) error {
			return nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	tx, err := service.StartCharging(ctx, userID, stationID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tx == nil {
		t.Fatal("expected transaction, got nil")
	}
}

func TestStartCharging_NoStationSpecified_FindsAvailable(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := "user-123"

	availableDevice := &domain.ChargePoint{
		ID:     "available-device",
		Status: domain.ChargePointStatusAvailable,
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			return nil, nil
		},
		SaveFunc: func(ctx context.Context, tx *domain.Transaction) error {
			return nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{
		ListAvailableDevicesFunc: func(ctx context.Context) ([]domain.ChargePoint, error) {
			return []domain.ChargePoint{*availableDevice}, nil
		},
		GetDeviceFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return availableDevice, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.ChargePointStatus) error {
			return nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	tx, err := service.StartCharging(ctx, userID, "") // Empty station ID

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tx == nil {
		t.Fatal("expected transaction, got nil")
	}
}

func TestStopActiveCharging_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := "user-123"

	activeTx := &domain.Transaction{
		ID:            "tx-active",
		UserID:        userID,
		ChargePointID: "device-123",
		Status:        domain.TransactionStatusStarted,
		StartTime:     time.Now().Add(-30 * time.Minute),
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			return activeTx, nil
		},
		FindByIDFunc: func(ctx context.Context, id string) (*domain.Transaction, error) {
			return activeTx, nil
		},
		UpdateFunc: func(ctx context.Context, tx *domain.Transaction) error {
			return nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.ChargePointStatus) error {
			return nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	err := service.StopActiveCharging(ctx, userID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStopActiveCharging_NoActiveSession(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			return nil, nil // No active session
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	err := service.StopActiveCharging(ctx, "user-123")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "no active charging session found" {
		t.Errorf("expected 'no active charging session found', got '%s'", err.Error())
	}
}

func TestGetCurrentSessionCost_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := "user-123"

	activeTx := &domain.Transaction{
		ID:        "tx-active",
		UserID:    userID,
		StartTime: time.Now().Add(-1 * time.Hour), // 1 hour ago
		Status:    domain.TransactionStatusStarted,
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			return activeTx, nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	cost, err := service.GetCurrentSessionCost(ctx, userID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Expected: ~1 hour * 7kW * 0.75 R$/kWh = ~5.25
	if cost < 5.0 || cost > 6.0 {
		t.Errorf("expected cost around 5.25, got %f", cost)
	}
}

func TestGetCurrentSessionCost_NoActiveSession(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			return nil, nil
		},
	}

	mockDeviceService := &mocks.MockDeviceService{}
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	_, err := service.GetCurrentSessionCost(ctx, "user-123")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStartTransaction_RepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockDevice := &domain.ChargePoint{
		ID:     "device-123",
		Status: domain.ChargePointStatusAvailable,
	}

	mockTxRepo := &mocks.MockTransactionRepository{
		FindActiveByUserIDFunc: func(ctx context.Context, uid string) (*domain.Transaction, error) {
			return nil, nil
		},
		SaveFunc: func(ctx context.Context, tx *domain.Transaction) error {
			return errors.New("database error")
		},
	}

	mockDeviceService := &mocks.MockDeviceService{
		GetDeviceFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return mockDevice, nil
		},
	}

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockTxRepo, mockDeviceService, mockQueue, newTestLogger())

	// Act
	_, err := service.StartTransaction(ctx, "device-123", 1, "user-123", "rfid")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
