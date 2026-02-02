package v2g

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// MockWalletService is a mock implementation of WalletService
type MockWalletService struct {
	wallets      map[string]float64
	transactions []WalletTransaction
}

type WalletTransaction struct {
	UserID      string
	Amount      float64
	Description string
}

func NewMockWalletService() *MockWalletService {
	return &MockWalletService{
		wallets:      make(map[string]float64),
		transactions: []WalletTransaction{},
	}
}

func (m *MockWalletService) GetWallet(ctx context.Context, userID string) (*domain.Wallet, error) {
	balance := m.wallets[userID]
	return &domain.Wallet{
		UserID:   userID,
		Balance:  balance,
		Currency: "BRL",
	}, nil
}

func (m *MockWalletService) AddFunds(ctx context.Context, userID string, amount float64, paymentID string) error {
	m.wallets[userID] += amount
	m.transactions = append(m.transactions, WalletTransaction{
		UserID:      userID,
		Amount:      amount,
		Description: paymentID,
	})
	return nil
}

func (m *MockWalletService) DeductFunds(ctx context.Context, userID string, amount float64, description string, referenceID string) error {
	m.wallets[userID] -= amount
	m.transactions = append(m.transactions, WalletTransaction{
		UserID:      userID,
		Amount:      -amount,
		Description: description,
	})
	return nil
}

func (m *MockWalletService) GetTransactions(ctx context.Context, userID string, limit, offset int) ([]domain.WalletTransaction, error) {
	return nil, nil
}

func (m *MockWalletService) HasSufficientBalance(ctx context.Context, userID string, amount float64) (bool, error) {
	return m.wallets[userID] >= amount, nil
}

// MockMessageQueue is a mock message queue
type MockMessageQueue struct {
	messages []MockMessage
}

type MockMessage struct {
	Topic string
	Data  interface{}
}

func NewMockMessageQueue() *MockMessageQueue {
	return &MockMessageQueue{
		messages: []MockMessage{},
	}
}

func (m *MockMessageQueue) Publish(topic string, message interface{}) error {
	m.messages = append(m.messages, MockMessage{
		Topic: topic,
		Data:  message,
	})
	return nil
}

func (m *MockMessageQueue) Subscribe(topic string, handler func(message []byte)) error {
	return nil
}

func (m *MockMessageQueue) Close() error {
	return nil
}

func createTestPaymentService() (*V2GPaymentService, *MockWalletService, *MockMessageQueue) {
	logger := zap.NewNop()
	wallet := NewMockWalletService()
	mq := NewMockMessageQueue()
	config := DefaultPaymentConfig()

	service := NewV2GPaymentService(nil, wallet, mq, logger, config)
	return service, wallet, mq
}

func TestPaymentConfig_Defaults(t *testing.T) {
	config := DefaultPaymentConfig()

	if config.OperatorMargin != 0.10 {
		t.Errorf("Expected operator margin 0.10, got %f", config.OperatorMargin)
	}

	if config.MinPayoutAmount != 5.00 {
		t.Errorf("Expected min payout 5.00, got %f", config.MinPayoutAmount)
	}

	if config.PayoutCurrency != "BRL" {
		t.Errorf("Expected currency BRL, got %s", config.PayoutCurrency)
	}

	if !config.AutoPayoutEnabled {
		t.Error("Expected auto payout to be enabled")
	}
}

func TestV2GPaymentService_CalculateAndRecordCompensation(t *testing.T) {
	service, _, mq := createTestPaymentService()
	ctx := context.Background()

	// Create a completed discharge session
	endTime := time.Now()
	session := &domain.V2GSession{
		ID:                "session123",
		ChargePointID:     "CP001",
		UserID:            "user123",
		Direction:         domain.V2GDirectionDischarging,
		Status:            domain.V2GStatusCompleted,
		EnergyTransferred: -25.0, // 25 kWh discharged (negative)
		GridPriceAtStart:  0.80,
		CurrentGridPrice:  0.90,
		StartTime:         endTime.Add(-2 * time.Hour),
		EndTime:           &endTime,
	}

	record, err := service.CalculateAndRecordCompensation(ctx, session)
	if err != nil {
		t.Fatalf("CalculateAndRecordCompensation failed: %v", err)
	}

	// Verify record
	if record.ID == "" {
		t.Error("Record should have an ID")
	}

	if record.SessionID != "session123" {
		t.Errorf("Expected session ID 'session123', got '%s'", record.SessionID)
	}

	if record.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", record.UserID)
	}

	// Energy should be positive (absolute value)
	if record.EnergyDischargedKWh != 25.0 {
		t.Errorf("Expected 25.0 kWh, got %f", record.EnergyDischargedKWh)
	}

	// Average price should be (0.80 + 0.90) / 2 = 0.85
	expectedAvgPrice := 0.85
	if record.AverageGridPrice != expectedAvgPrice {
		t.Errorf("Expected avg price %f, got %f", expectedAvgPrice, record.AverageGridPrice)
	}

	// Gross = 25 * 0.85 = 21.25
	expectedGross := 25.0 * expectedAvgPrice
	if record.GrossAmount != expectedGross {
		t.Errorf("Expected gross %f, got %f", expectedGross, record.GrossAmount)
	}

	// Net = Gross * (1 - 0.10) = 21.25 * 0.90 = 19.125
	expectedNet := expectedGross * 0.90
	if record.NetAmount != expectedNet {
		t.Errorf("Expected net %f, got %f", expectedNet, record.NetAmount)
	}

	if record.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", record.Status)
	}

	if record.Currency != "BRL" {
		t.Errorf("Expected currency BRL, got %s", record.Currency)
	}

	// Verify event was published
	if len(mq.messages) == 0 {
		t.Error("Expected compensation event to be published")
	}

	if mq.messages[0].Topic != "v2g.compensation.calculated" {
		t.Errorf("Expected topic 'v2g.compensation.calculated', got '%s'", mq.messages[0].Topic)
	}
}

func TestV2GPaymentService_CalculateCompensation_NonDischarge(t *testing.T) {
	service, _, _ := createTestPaymentService()
	ctx := context.Background()

	// Create a charging session (not discharge)
	session := &domain.V2GSession{
		ID:                "session456",
		Direction:         domain.V2GDirectionCharging,
		Status:            domain.V2GStatusCompleted,
		EnergyTransferred: 30.0, // Charging
	}

	_, err := service.CalculateAndRecordCompensation(ctx, session)
	if err == nil {
		t.Error("Expected error for non-discharge session")
	}
}

func TestV2GPaymentService_CalculateCompensation_NotCompleted(t *testing.T) {
	service, _, _ := createTestPaymentService()
	ctx := context.Background()

	// Create an active (not completed) session
	session := &domain.V2GSession{
		ID:                "session789",
		Direction:         domain.V2GDirectionDischarging,
		Status:            domain.V2GStatusActive,
		EnergyTransferred: -20.0,
	}

	_, err := service.CalculateAndRecordCompensation(ctx, session)
	if err == nil {
		t.Error("Expected error for non-completed session")
	}
}

func TestV2GPaymentService_ProcessPayout(t *testing.T) {
	service, wallet, mq := createTestPaymentService()
	ctx := context.Background()

	// Create a compensation record
	record := &V2GCompensationRecord{
		ID:                  "comp123",
		SessionID:           "session123",
		UserID:              "user123",
		EnergyDischargedKWh: 20.0,
		AverageGridPrice:    0.85,
		OperatorMargin:      0.10,
		GrossAmount:         17.0,
		NetAmount:           15.30, // Above minimum
		Currency:            "BRL",
		Status:              "pending",
	}

	err := service.ProcessPayout(ctx, record)
	if err != nil {
		t.Fatalf("ProcessPayout failed: %v", err)
	}

	// Verify wallet was credited
	if wallet.wallets["user123"] != 15.30 {
		t.Errorf("Expected wallet balance 15.30, got %f", wallet.wallets["user123"])
	}

	// Verify record was updated
	if record.Status != "paid" {
		t.Errorf("Expected status 'paid', got '%s'", record.Status)
	}

	if record.PaidAt == nil {
		t.Error("Expected PaidAt to be set")
	}

	if record.PaymentID == "" {
		t.Error("Expected PaymentID to be set")
	}

	// Verify events
	found := false
	for _, msg := range mq.messages {
		if msg.Topic == "v2g.compensation.paid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected compensation paid event")
	}
}

func TestV2GPaymentService_ProcessPayout_BelowMinimum(t *testing.T) {
	service, wallet, _ := createTestPaymentService()
	ctx := context.Background()

	// Create a compensation record below minimum
	record := &V2GCompensationRecord{
		ID:        "comp456",
		SessionID: "session456",
		UserID:    "user456",
		NetAmount: 3.00, // Below minimum (5.00)
		Status:    "pending",
	}

	err := service.ProcessPayout(ctx, record)
	if err != nil {
		t.Fatalf("ProcessPayout failed: %v", err)
	}

	// Wallet should not be credited
	if wallet.wallets["user456"] != 0 {
		t.Errorf("Expected wallet balance 0, got %f", wallet.wallets["user456"])
	}

	// Status should remain pending
	if record.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", record.Status)
	}
}

func TestV2GPaymentService_ProcessPayout_AlreadyProcessed(t *testing.T) {
	service, _, _ := createTestPaymentService()
	ctx := context.Background()

	// Create an already processed record
	paidAt := time.Now()
	record := &V2GCompensationRecord{
		ID:        "comp789",
		SessionID: "session789",
		UserID:    "user789",
		NetAmount: 20.00,
		Status:    "paid",
		PaidAt:    &paidAt,
	}

	err := service.ProcessPayout(ctx, record)
	if err == nil {
		t.Error("Expected error for already processed compensation")
	}
}

func TestV2GPaymentService_ProcessSessionCompensation(t *testing.T) {
	service, wallet, _ := createTestPaymentService()
	ctx := context.Background()

	endTime := time.Now()
	session := &domain.V2GSession{
		ID:                "session_full",
		ChargePointID:     "CP001",
		UserID:            "user_full",
		Direction:         domain.V2GDirectionDischarging,
		Status:            domain.V2GStatusCompleted,
		EnergyTransferred: -30.0, // 30 kWh
		GridPriceAtStart:  0.90,
		CurrentGridPrice:  1.00,
		StartTime:         endTime.Add(-3 * time.Hour),
		EndTime:           &endTime,
	}

	err := service.ProcessSessionCompensation(ctx, session)
	if err != nil {
		t.Fatalf("ProcessSessionCompensation failed: %v", err)
	}

	// Verify wallet was credited
	// Energy: 30 kWh, Avg Price: 0.95, Gross: 28.5, Net: 25.65
	expectedNet := 30.0 * 0.95 * 0.90
	if wallet.wallets["user_full"] != expectedNet {
		t.Errorf("Expected wallet balance %f, got %f", expectedNet, wallet.wallets["user_full"])
	}
}

func TestV2GPaymentService_CompensationReport(t *testing.T) {
	service, _, _ := createTestPaymentService()
	ctx := context.Background()

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	report, err := service.GenerateCompensationReport(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("GenerateCompensationReport failed: %v", err)
	}

	if report.StartDate != startDate {
		t.Error("Report start date mismatch")
	}

	if report.EndDate != endDate {
		t.Error("Report end date mismatch")
	}

	if report.GeneratedAt.IsZero() {
		t.Error("Report should have generation timestamp")
	}
}

func TestV2GPaymentService_OperatorMarginCalculation(t *testing.T) {
	tests := []struct {
		energy       float64
		price        float64
		margin       float64
		expectedNet  float64
	}{
		{10.0, 1.00, 0.10, 9.0},
		{20.0, 0.80, 0.15, 13.6},
		{50.0, 0.90, 0.05, 42.75},
		{100.0, 0.75, 0.20, 60.0},
	}

	for _, tt := range tests {
		gross := tt.energy * tt.price
		net := gross * (1 - tt.margin)

		if net != tt.expectedNet {
			t.Errorf("Energy %.1f, Price %.2f, Margin %.2f: expected %.2f, got %.2f",
				tt.energy, tt.price, tt.margin, tt.expectedNet, net)
		}
	}
}

func TestV2GPaymentService_MultiplePayouts(t *testing.T) {
	service, wallet, _ := createTestPaymentService()
	ctx := context.Background()

	// Process multiple payouts for the same user
	records := []*V2GCompensationRecord{
		{ID: "c1", SessionID: "s1", UserID: "user_multi", NetAmount: 10.0, Status: "pending"},
		{ID: "c2", SessionID: "s2", UserID: "user_multi", NetAmount: 15.0, Status: "pending"},
		{ID: "c3", SessionID: "s3", UserID: "user_multi", NetAmount: 20.0, Status: "pending"},
	}

	for _, record := range records {
		err := service.ProcessPayout(ctx, record)
		if err != nil {
			t.Fatalf("ProcessPayout failed for %s: %v", record.ID, err)
		}
	}

	// Total should be 10 + 15 + 20 = 45
	expectedTotal := 45.0
	if wallet.wallets["user_multi"] != expectedTotal {
		t.Errorf("Expected total balance %f, got %f", expectedTotal, wallet.wallets["user_multi"])
	}
}
