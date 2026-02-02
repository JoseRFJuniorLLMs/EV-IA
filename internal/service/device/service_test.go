package device

import (
	"context"
	"encoding/json"
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

func TestGetDevice_Success_CacheMiss(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "device-123"

	expectedDevice := &domain.ChargePoint{
		ID:     deviceID,
		Vendor: "ABB",
		Model:  "Terra 184",
		Status: domain.ChargePointStatusAvailable,
	}

	mockRepo := &mocks.MockChargePointRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			if id == deviceID {
				return expectedDevice, nil
			}
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	device, err := service.GetDevice(ctx, deviceID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if device == nil {
		t.Fatal("expected device, got nil")
	}
	if device.ID != deviceID {
		t.Errorf("expected device ID '%s', got '%s'", deviceID, device.ID)
	}
	if device.Vendor != "ABB" {
		t.Errorf("expected vendor 'ABB', got '%s'", device.Vendor)
	}
}

func TestGetDevice_Success_CacheHit(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "device-123"

	cachedDevice := &domain.ChargePoint{
		ID:     deviceID,
		Vendor: "ABB",
		Model:  "Terra 184",
		Status: domain.ChargePointStatusAvailable,
	}
	cachedJSON, _ := json.Marshal(cachedDevice)

	mockRepo := &mocks.MockChargePointRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			t.Error("repository should not be called on cache hit")
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	mockCache.Set(ctx, "device:"+deviceID, string(cachedJSON), 30*time.Second)

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	device, err := service.GetDevice(ctx, deviceID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if device == nil {
		t.Fatal("expected device, got nil")
	}
	if device.ID != deviceID {
		t.Errorf("expected device ID '%s', got '%s'", deviceID, device.ID)
	}
}

func TestGetDevice_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockChargePointRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return nil, nil
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	device, err := service.GetDevice(ctx, "nonexistent")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if device != nil {
		t.Error("expected nil device for nonexistent ID")
	}
}

func TestGetDevice_RepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockChargePointRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*domain.ChargePoint, error) {
			return nil, errors.New("database error")
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	_, err := service.GetDevice(ctx, "device-123")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListDevices_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()

	expectedDevices := []domain.ChargePoint{
		{ID: "device-1", Vendor: "ABB", Status: domain.ChargePointStatusAvailable},
		{ID: "device-2", Vendor: "Siemens", Status: domain.ChargePointStatusOccupied},
	}

	mockRepo := &mocks.MockChargePointRepository{
		FindAllFunc: func(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
			return expectedDevices, nil
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	devices, err := service.ListDevices(ctx, nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestListDevices_WithFilter(t *testing.T) {
	// Arrange
	ctx := context.Background()
	var receivedFilter map[string]interface{}

	mockRepo := &mocks.MockChargePointRepository{
		FindAllFunc: func(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
			receivedFilter = filter
			return []domain.ChargePoint{
				{ID: "device-1", Status: domain.ChargePointStatusAvailable},
			}, nil
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	filter := map[string]interface{}{
		"status": domain.ChargePointStatusAvailable,
	}

	// Act
	devices, err := service.ListDevices(ctx, filter)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if receivedFilter == nil {
		t.Error("expected filter to be passed to repository")
	}
	if len(devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(devices))
	}
}

func TestUpdateStatus_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "device-123"
	newStatus := domain.ChargePointStatusOccupied

	var updatedID string
	var updatedStatus domain.ChargePointStatus

	mockRepo := &mocks.MockChargePointRepository{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.ChargePointStatus) error {
			updatedID = id
			updatedStatus = status
			return nil
		},
	}

	mockCache := mocks.NewMockCache()
	// Pre-populate cache
	mockCache.Set(ctx, "device:"+deviceID, "{}", 30*time.Second)

	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	err := service.UpdateStatus(ctx, deviceID, newStatus)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedID != deviceID {
		t.Errorf("expected device ID '%s', got '%s'", deviceID, updatedID)
	}
	if updatedStatus != newStatus {
		t.Errorf("expected status '%s', got '%s'", newStatus, updatedStatus)
	}

	// Check cache was invalidated
	cached, _ := mockCache.Get(ctx, "device:"+deviceID)
	if cached != "" {
		t.Error("expected cache to be invalidated")
	}

	// Check event was published
	messages := mockQueue.GetPublishedMessages("device.status.changed")
	if len(messages) != 1 {
		t.Errorf("expected 1 message published, got %d", len(messages))
	}
}

func TestUpdateStatus_RepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockChargePointRepository{
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.ChargePointStatus) error {
			return errors.New("database error")
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	err := service.UpdateStatus(ctx, "device-123", domain.ChargePointStatusOccupied)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetNearby_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()

	expectedDevices := []domain.ChargePoint{
		{ID: "device-1", Vendor: "ABB"},
		{ID: "device-2", Vendor: "Siemens"},
	}

	var receivedLat, receivedLon, receivedRadius float64

	mockRepo := &mocks.MockChargePointRepository{
		FindNearbyFunc: func(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error) {
			receivedLat = lat
			receivedLon = lon
			receivedRadius = radius
			return expectedDevices, nil
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	devices, err := service.GetNearby(ctx, -23.55, -46.63, 5.0)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
	if receivedLat != -23.55 {
		t.Errorf("expected lat -23.55, got %f", receivedLat)
	}
	if receivedLon != -46.63 {
		t.Errorf("expected lon -46.63, got %f", receivedLon)
	}
	if receivedRadius != 5.0 {
		t.Errorf("expected radius 5.0, got %f", receivedRadius)
	}
}

func TestListAvailableDevices_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()

	availableDevices := []domain.ChargePoint{
		{ID: "device-1", Status: domain.ChargePointStatusAvailable},
		{ID: "device-2", Status: domain.ChargePointStatusAvailable},
	}

	var receivedFilter map[string]interface{}

	mockRepo := &mocks.MockChargePointRepository{
		FindAllFunc: func(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
			receivedFilter = filter
			return availableDevices, nil
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	devices, err := service.ListAvailableDevices(ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
	if receivedFilter["status"] != domain.ChargePointStatusAvailable {
		t.Error("expected filter to include status=Available")
	}
}

func TestListAvailableDevices_RepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()

	mockRepo := &mocks.MockChargePointRepository{
		FindAllFunc: func(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
			return nil, errors.New("database error")
		},
	}

	mockCache := mocks.NewMockCache()
	mockQueue := mocks.NewMockMessageQueue()

	service := NewService(mockRepo, mockCache, mockQueue, newTestLogger())

	// Act
	_, err := service.ListAvailableDevices(ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
