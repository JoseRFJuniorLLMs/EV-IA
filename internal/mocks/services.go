package mocks

import (
	"context"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// MockDeviceService is a mock implementation of DeviceService interface
type MockDeviceService struct {
	GetDeviceFunc            func(ctx context.Context, id string) (*domain.ChargePoint, error)
	ListDevicesFunc          func(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error)
	UpdateStatusFunc         func(ctx context.Context, id string, status domain.ChargePointStatus) error
	GetNearbyFunc            func(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error)
	ListAvailableDevicesFunc func(ctx context.Context) ([]domain.ChargePoint, error)
}

func (m *MockDeviceService) GetDevice(ctx context.Context, id string) (*domain.ChargePoint, error) {
	if m.GetDeviceFunc != nil {
		return m.GetDeviceFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockDeviceService) ListDevices(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
	if m.ListDevicesFunc != nil {
		return m.ListDevicesFunc(ctx, filter)
	}
	return []domain.ChargePoint{}, nil
}

func (m *MockDeviceService) UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockDeviceService) GetNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error) {
	if m.GetNearbyFunc != nil {
		return m.GetNearbyFunc(ctx, lat, lon, radius)
	}
	return []domain.ChargePoint{}, nil
}

func (m *MockDeviceService) ListAvailableDevices(ctx context.Context) ([]domain.ChargePoint, error) {
	if m.ListAvailableDevicesFunc != nil {
		return m.ListAvailableDevicesFunc(ctx)
	}
	return []domain.ChargePoint{}, nil
}

// MockTransactionService is a mock implementation of TransactionService interface
type MockTransactionService struct {
	StartTransactionFunc      func(ctx context.Context, deviceID string, connectorID int, userID string, idTag string) (*domain.Transaction, error)
	StopTransactionFunc       func(ctx context.Context, transactionID string) (*domain.Transaction, error)
	GetTransactionFunc        func(ctx context.Context, id string) (*domain.Transaction, error)
	GetActiveTransactionFunc  func(ctx context.Context, userID string) (*domain.Transaction, error)
	GetTransactionHistoryFunc func(ctx context.Context, userID string) ([]domain.Transaction, error)
	StartChargingFunc         func(ctx context.Context, userID string, stationID string) (*domain.Transaction, error)
	StopActiveChargingFunc    func(ctx context.Context, userID string) error
	GetCurrentSessionCostFunc func(ctx context.Context, userID string) (float64, error)
}

func (m *MockTransactionService) StartTransaction(ctx context.Context, deviceID string, connectorID int, userID string, idTag string) (*domain.Transaction, error) {
	if m.StartTransactionFunc != nil {
		return m.StartTransactionFunc(ctx, deviceID, connectorID, userID, idTag)
	}
	return nil, nil
}

func (m *MockTransactionService) StopTransaction(ctx context.Context, transactionID string) (*domain.Transaction, error) {
	if m.StopTransactionFunc != nil {
		return m.StopTransactionFunc(ctx, transactionID)
	}
	return nil, nil
}

func (m *MockTransactionService) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	if m.GetTransactionFunc != nil {
		return m.GetTransactionFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTransactionService) GetActiveTransaction(ctx context.Context, userID string) (*domain.Transaction, error) {
	if m.GetActiveTransactionFunc != nil {
		return m.GetActiveTransactionFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockTransactionService) GetTransactionHistory(ctx context.Context, userID string) ([]domain.Transaction, error) {
	if m.GetTransactionHistoryFunc != nil {
		return m.GetTransactionHistoryFunc(ctx, userID)
	}
	return []domain.Transaction{}, nil
}

func (m *MockTransactionService) StartCharging(ctx context.Context, userID string, stationID string) (*domain.Transaction, error) {
	if m.StartChargingFunc != nil {
		return m.StartChargingFunc(ctx, userID, stationID)
	}
	return nil, nil
}

func (m *MockTransactionService) StopActiveCharging(ctx context.Context, userID string) error {
	if m.StopActiveChargingFunc != nil {
		return m.StopActiveChargingFunc(ctx, userID)
	}
	return nil
}

func (m *MockTransactionService) GetCurrentSessionCost(ctx context.Context, userID string) (float64, error) {
	if m.GetCurrentSessionCostFunc != nil {
		return m.GetCurrentSessionCostFunc(ctx, userID)
	}
	return 0, nil
}

// MockEmailService is a mock implementation of EmailService interface
type MockEmailService struct {
	SendFunc              func(ctx context.Context, to, subject, body string) error
	SendHTMLFunc          func(ctx context.Context, to, subject, htmlBody string) error
	SendTemplateFunc      func(ctx context.Context, to, templateName string, data map[string]interface{}) error
	SendWelcomeFunc       func(ctx context.Context, user *domain.User) error
	SendChargingStartedFunc   func(ctx context.Context, user *domain.User, tx *domain.Transaction, station *domain.ChargePoint) error
	SendChargingCompletedFunc func(ctx context.Context, user *domain.User, tx *domain.Transaction, cost float64) error
	SendPasswordResetFunc func(ctx context.Context, user *domain.User, resetToken string) error
	SendInvoiceFunc       func(ctx context.Context, user *domain.User, invoice *Invoice) error
	SendLowBalanceFunc    func(ctx context.Context, user *domain.User, balance float64) error

	// Track sent emails for assertions
	SentEmails []SentEmail
}

// SentEmail represents a sent email for testing
type SentEmail struct {
	To          string
	Subject     string
	Body        string
	Template    string
	Data        map[string]interface{}
}

// Invoice for mock testing
type Invoice struct {
	ID            string
	TransactionID string
	Amount        float64
	Currency      string
	EnergyKWh     float64
	Duration      string
	StationName   string
	Date          string
}

func (m *MockEmailService) Send(ctx context.Context, to, subject, body string) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: to, Subject: subject, Body: body})
	if m.SendFunc != nil {
		return m.SendFunc(ctx, to, subject, body)
	}
	return nil
}

func (m *MockEmailService) SendHTML(ctx context.Context, to, subject, htmlBody string) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: to, Subject: subject, Body: htmlBody})
	if m.SendHTMLFunc != nil {
		return m.SendHTMLFunc(ctx, to, subject, htmlBody)
	}
	return nil
}

func (m *MockEmailService) SendTemplate(ctx context.Context, to, templateName string, data map[string]interface{}) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: to, Template: templateName, Data: data})
	if m.SendTemplateFunc != nil {
		return m.SendTemplateFunc(ctx, to, templateName, data)
	}
	return nil
}

func (m *MockEmailService) SendWelcome(ctx context.Context, user *domain.User) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: user.Email, Template: "welcome"})
	if m.SendWelcomeFunc != nil {
		return m.SendWelcomeFunc(ctx, user)
	}
	return nil
}

func (m *MockEmailService) SendChargingStarted(ctx context.Context, user *domain.User, tx *domain.Transaction, station *domain.ChargePoint) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: user.Email, Template: "charging_started"})
	if m.SendChargingStartedFunc != nil {
		return m.SendChargingStartedFunc(ctx, user, tx, station)
	}
	return nil
}

func (m *MockEmailService) SendChargingCompleted(ctx context.Context, user *domain.User, tx *domain.Transaction, cost float64) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: user.Email, Template: "charging_completed"})
	if m.SendChargingCompletedFunc != nil {
		return m.SendChargingCompletedFunc(ctx, user, tx, cost)
	}
	return nil
}

func (m *MockEmailService) SendPasswordReset(ctx context.Context, user *domain.User, resetToken string) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: user.Email, Template: "password_reset"})
	if m.SendPasswordResetFunc != nil {
		return m.SendPasswordResetFunc(ctx, user, resetToken)
	}
	return nil
}

func (m *MockEmailService) SendInvoice(ctx context.Context, user *domain.User, invoice *Invoice) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: user.Email, Template: "invoice"})
	if m.SendInvoiceFunc != nil {
		return m.SendInvoiceFunc(ctx, user, invoice)
	}
	return nil
}

func (m *MockEmailService) SendLowBalance(ctx context.Context, user *domain.User, balance float64) error {
	m.SentEmails = append(m.SentEmails, SentEmail{To: user.Email, Template: "low_balance"})
	if m.SendLowBalanceFunc != nil {
		return m.SendLowBalanceFunc(ctx, user, balance)
	}
	return nil
}

// GetSentEmails returns all sent emails for assertions
func (m *MockEmailService) GetSentEmails() []SentEmail {
	return m.SentEmails
}

// ClearSentEmails clears the sent emails list
func (m *MockEmailService) ClearSentEmails() {
	m.SentEmails = nil
}
