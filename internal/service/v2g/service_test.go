package v2g

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// MockV2GRepository is a mock implementation of V2GRepository
type MockV2GRepository struct {
	sessions    map[string]*domain.V2GSession
	preferences map[string]*domain.V2GPreferences
}

func NewMockV2GRepository() *MockV2GRepository {
	return &MockV2GRepository{
		sessions:    make(map[string]*domain.V2GSession),
		preferences: make(map[string]*domain.V2GPreferences),
	}
}

func (m *MockV2GRepository) CreateSession(ctx context.Context, session *domain.V2GSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *MockV2GRepository) UpdateSession(ctx context.Context, session *domain.V2GSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *MockV2GRepository) GetSession(ctx context.Context, sessionID string) (*domain.V2GSession, error) {
	if session, ok := m.sessions[sessionID]; ok {
		return session, nil
	}
	return nil, nil
}

func (m *MockV2GRepository) GetSessionsByChargePoint(ctx context.Context, chargePointID string, limit int) ([]domain.V2GSession, error) {
	var result []domain.V2GSession
	for _, s := range m.sessions {
		if s.ChargePointID == chargePointID {
			result = append(result, *s)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MockV2GRepository) GetSessionsByUser(ctx context.Context, userID string, limit int) ([]domain.V2GSession, error) {
	var result []domain.V2GSession
	for _, s := range m.sessions {
		if s.UserID == userID {
			result = append(result, *s)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MockV2GRepository) GetActiveSessions(ctx context.Context) ([]domain.V2GSession, error) {
	var result []domain.V2GSession
	for _, s := range m.sessions {
		if s.Status == domain.V2GStatusActive {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *MockV2GRepository) SavePreferences(ctx context.Context, prefs *domain.V2GPreferences) error {
	m.preferences[prefs.UserID] = prefs
	return nil
}

func (m *MockV2GRepository) GetPreferences(ctx context.Context, userID string) (*domain.V2GPreferences, error) {
	if prefs, ok := m.preferences[userID]; ok {
		return prefs, nil
	}
	return &domain.V2GPreferences{
		UserID:          userID,
		AutoDischarge:   false,
		MinGridPrice:    0.80,
		MaxDischargeKWh: 50.0,
		PreserveSOC:     20,
	}, nil
}

func (m *MockV2GRepository) CreateEvent(ctx context.Context, event *domain.V2GEvent) error {
	return nil
}

func (m *MockV2GRepository) GetEventsBySession(ctx context.Context, sessionID string) ([]domain.V2GEvent, error) {
	return nil, nil
}

func (m *MockV2GRepository) GetUserStats(ctx context.Context, userID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	return &domain.V2GStats{
		EntityID:                userID,
		EntityType:              "user",
		TotalSessions:           10,
		TotalEnergyDischargedKWh: 150.0,
		TotalCompensation:       120.0,
	}, nil
}

func (m *MockV2GRepository) GetChargePointStats(ctx context.Context, chargePointID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	return &domain.V2GStats{
		EntityID:                chargePointID,
		EntityType:              "charge_point",
		TotalSessions:           50,
		TotalEnergyDischargedKWh: 500.0,
		TotalCompensation:       400.0,
	}, nil
}

func (m *MockV2GRepository) GetGlobalStats(ctx context.Context, startDate, endDate time.Time) (*domain.V2GStats, error) {
	return &domain.V2GStats{
		EntityType:              "global",
		TotalSessions:           1000,
		TotalEnergyDischargedKWh: 10000.0,
		TotalCompensation:       8000.0,
	}, nil
}

func (m *MockV2GRepository) GetPendingCompensations(ctx context.Context) ([]domain.V2GSession, error) {
	return nil, nil
}

func (m *MockV2GRepository) MarkCompensationPaid(ctx context.Context, sessionID string, paymentID string) error {
	return nil
}

// MockGridPriceService is a mock implementation of GridPriceService
type MockGridPriceService struct {
	currentPrice float64
	isPeak       bool
}

func NewMockGridPriceService() *MockGridPriceService {
	return &MockGridPriceService{
		currentPrice: 0.85,
		isPeak:       false,
	}
}

func (m *MockGridPriceService) GetCurrentPrice(ctx context.Context) (float64, error) {
	return m.currentPrice, nil
}

func (m *MockGridPriceService) GetPriceForecast(ctx context.Context, hours int) ([]domain.GridPricePoint, error) {
	forecast := make([]domain.GridPricePoint, hours)
	now := time.Now()
	for i := 0; i < hours; i++ {
		forecast[i] = domain.GridPricePoint{
			Timestamp: now.Add(time.Duration(i) * time.Hour),
			Price:     m.currentPrice,
			IsPeak:    i >= 18 && i < 21,
		}
	}
	return forecast, nil
}

func (m *MockGridPriceService) IsPeakHour(ctx context.Context) (bool, error) {
	return m.isPeak, nil
}

func (m *MockGridPriceService) CalculateV2GCompensation(ctx context.Context, energyKWh float64, startTime, endTime time.Time) (float64, error) {
	return energyKWh * m.currentPrice * 0.9, nil
}

// MockOCPPCommandService is a mock OCPP command service
type MockOCPPCommandService struct {
	connected map[string]bool
}

func NewMockOCPPCommandService() *MockOCPPCommandService {
	return &MockOCPPCommandService{
		connected: map[string]bool{
			"CP001": true,
			"CP002": true,
		},
	}
}

func (m *MockOCPPCommandService) IsConnected(chargePointID string) bool {
	return m.connected[chargePointID]
}

func (m *MockOCPPCommandService) GetConnectedClients() []string {
	var clients []string
	for id, connected := range m.connected {
		if connected {
			clients = append(clients, id)
		}
	}
	return clients
}

func (m *MockOCPPCommandService) SetV2GChargingProfile(ctx context.Context, chargePointID string, evseID int, dischargePowerKW float64, durationSeconds int) error {
	return nil
}

func (m *MockOCPPCommandService) ClearV2GChargingProfile(ctx context.Context, chargePointID string, evseID int) error {
	return nil
}

func (m *MockOCPPCommandService) GetV2GCapability(ctx context.Context, chargePointID string) (*domain.V2GCapability, error) {
	return &domain.V2GCapability{
		Supported:             true,
		MaxDischargePowerKW:   50.0,
		MaxDischargeCurrentA:  125.0,
		BidirectionalCharging: true,
		ISO15118Support:       true,
	}, nil
}

// Test helper to create V2G service with mocks
func createTestV2GService() (*V2GService, *MockV2GRepository) {
	logger := zap.NewNop()
	repo := NewMockV2GRepository()
	gridPrice := NewMockGridPriceService()
	ocpp := NewMockOCPPCommandService()

	service := NewV2GService(repo, gridPrice, ocpp, nil, logger, nil)
	return service, repo
}

func TestV2GService_CheckV2GCapability(t *testing.T) {
	service, _ := createTestV2GService()
	ctx := context.Background()

	capability, err := service.CheckV2GCapability(ctx, "CP001")
	if err != nil {
		t.Fatalf("CheckV2GCapability failed: %v", err)
	}

	if !capability.Supported {
		t.Error("Expected V2G to be supported")
	}

	if capability.MaxDischargePowerKW != 50.0 {
		t.Errorf("Expected max discharge power 50.0, got %f", capability.MaxDischargePowerKW)
	}

	if !capability.BidirectionalCharging {
		t.Error("Expected bidirectional charging to be supported")
	}

	if !capability.ISO15118Support {
		t.Error("Expected ISO 15118 support")
	}
}

func TestV2GService_SetUserPreferences(t *testing.T) {
	service, repo := createTestV2GService()
	ctx := context.Background()

	prefs := &domain.V2GPreferences{
		UserID:          "user123",
		AutoDischarge:   true,
		MinGridPrice:    1.00,
		MaxDischargeKWh: 30.0,
		PreserveSOC:     30,
		NotifyOnStart:   true,
		NotifyOnEnd:     true,
	}

	err := service.SetUserPreferences(ctx, "user123", prefs)
	if err != nil {
		t.Fatalf("SetUserPreferences failed: %v", err)
	}

	// Verify preferences were saved
	savedPrefs := repo.preferences["user123"]
	if savedPrefs == nil {
		t.Fatal("Preferences not saved")
	}

	if savedPrefs.MinGridPrice != 1.00 {
		t.Errorf("Expected min grid price 1.00, got %f", savedPrefs.MinGridPrice)
	}

	if savedPrefs.MaxDischargeKWh != 30.0 {
		t.Errorf("Expected max discharge 30.0, got %f", savedPrefs.MaxDischargeKWh)
	}
}

func TestV2GService_GetUserPreferences(t *testing.T) {
	service, _ := createTestV2GService()
	ctx := context.Background()

	// Get default preferences for user without saved preferences
	prefs, err := service.GetUserPreferences(ctx, "newuser")
	if err != nil {
		t.Fatalf("GetUserPreferences failed: %v", err)
	}

	// Check defaults
	if prefs.AutoDischarge != false {
		t.Error("Expected AutoDischarge to be false by default")
	}

	if prefs.MinGridPrice != 0.80 {
		t.Errorf("Expected default min grid price 0.80, got %f", prefs.MinGridPrice)
	}

	if prefs.PreserveSOC != 20 {
		t.Errorf("Expected default preserve SOC 20, got %d", prefs.PreserveSOC)
	}
}

func TestV2GService_GetUserStats(t *testing.T) {
	service, _ := createTestV2GService()
	ctx := context.Background()

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	stats, err := service.GetUserStats(ctx, "user123", startDate, endDate)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	if stats.TotalSessions != 10 {
		t.Errorf("Expected 10 sessions, got %d", stats.TotalSessions)
	}

	if stats.TotalEnergyDischargedKWh != 150.0 {
		t.Errorf("Expected 150 kWh, got %f", stats.TotalEnergyDischargedKWh)
	}

	if stats.TotalCompensation != 120.0 {
		t.Errorf("Expected 120.0 compensation, got %f", stats.TotalCompensation)
	}
}

func TestV2GService_CalculateCompensation(t *testing.T) {
	service, _ := createTestV2GService()
	ctx := context.Background()

	session := &domain.V2GSession{
		ID:               "session123",
		ChargePointID:    "CP001",
		UserID:           "user123",
		Direction:        domain.V2GDirectionDischarging,
		Status:           domain.V2GStatusCompleted,
		EnergyTransferred: -20.0, // 20 kWh discharged (negative)
		GridPriceAtStart: 0.80,
		CurrentGridPrice: 0.90,
		StartTime:        time.Now().Add(-2 * time.Hour),
	}

	compensation, err := service.CalculateCompensation(ctx, session)
	if err != nil {
		t.Fatalf("CalculateCompensation failed: %v", err)
	}

	// Verify compensation was calculated
	if compensation.EnergyKWh <= 0 {
		t.Error("Expected positive energy value")
	}

	if compensation.Amount <= 0 {
		t.Error("Expected positive compensation amount")
	}

	if compensation.Currency != "BRL" {
		t.Errorf("Expected currency BRL, got %s", compensation.Currency)
	}
}
