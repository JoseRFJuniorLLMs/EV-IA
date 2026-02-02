package email

import (
	"context"
	"errors"
	"html/template"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// MockProvider is a mock email provider for testing
type MockProvider struct {
	SentEmails []MockEmail
	ShouldFail bool
	FailError  error
}

type MockEmail struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

func (m *MockProvider) Send(ctx context.Context, to, subject, body string, isHTML bool) error {
	if m.ShouldFail {
		if m.FailError != nil {
			return m.FailError
		}
		return errors.New("mock send failed")
	}

	m.SentEmails = append(m.SentEmails, MockEmail{
		To:      to,
		Subject: subject,
		Body:    body,
		IsHTML:  isHTML,
	})
	return nil
}

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func newTestService(provider *MockProvider) *Service {
	return &Service{
		config: &Config{
			Provider:  "mock",
			FromEmail: "test@sigec-ve.com",
			FromName:  "SIGEC-VE Test",
			BaseURL:   "http://localhost:3000",
		},
		provider:  provider,
		templates: make(map[string]*template.Template),
		log:       newTestLogger(),
	}
}

func TestService_Send_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)

	// Act
	err := service.Send(context.Background(), "user@example.com", "Test Subject", "Test Body")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if email.To != "user@example.com" {
		t.Errorf("expected to 'user@example.com', got '%s'", email.To)
	}
	if email.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got '%s'", email.Subject)
	}
	if email.Body != "Test Body" {
		t.Errorf("expected body 'Test Body', got '%s'", email.Body)
	}
	if email.IsHTML {
		t.Error("expected plain text email, got HTML")
	}
}

func TestService_Send_Failure(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{
		ShouldFail: true,
		FailError:  errors.New("SMTP connection failed"),
	}
	service := newTestService(mockProvider)

	// Act
	err := service.Send(context.Background(), "user@example.com", "Test Subject", "Test Body")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "SMTP connection failed") {
		t.Errorf("expected error to contain 'SMTP connection failed', got '%s'", err.Error())
	}
}

func TestService_SendHTML_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)

	htmlBody := "<h1>Hello World</h1>"

	// Act
	err := service.SendHTML(context.Background(), "user@example.com", "HTML Subject", htmlBody)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if !email.IsHTML {
		t.Error("expected HTML email, got plain text")
	}
	if email.Body != htmlBody {
		t.Errorf("expected body '%s', got '%s'", htmlBody, email.Body)
	}
}

func TestService_SendWelcome_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)
	service.loadTemplates()

	user := &domain.User{
		ID:    "user-123",
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// Act
	err := service.SendWelcome(context.Background(), user)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if email.To != "john@example.com" {
		t.Errorf("expected to 'john@example.com', got '%s'", email.To)
	}
	if !strings.Contains(email.Body, "John Doe") {
		t.Error("expected body to contain user name")
	}
	if !strings.Contains(email.Body, "Welcome") {
		t.Error("expected body to contain welcome message")
	}
}

func TestService_SendChargingStarted_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)
	service.loadTemplates()

	user := &domain.User{
		ID:    "user-123",
		Name:  "John Doe",
		Email: "john@example.com",
	}
	tx := &domain.Transaction{
		ID:        "tx-123",
		StartTime: time.Now(),
	}
	station := &domain.ChargePoint{
		ID:     "station-1",
		Vendor: "ABB",
		Model:  "Terra 184",
	}

	// Act
	err := service.SendChargingStarted(context.Background(), user, tx, station)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if !strings.Contains(email.Body, "tx-123") {
		t.Error("expected body to contain transaction ID")
	}
	if !strings.Contains(email.Body, "ABB Terra 184") {
		t.Error("expected body to contain station name")
	}
}

func TestService_SendChargingCompleted_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)
	service.loadTemplates()

	user := &domain.User{
		ID:    "user-123",
		Name:  "John Doe",
		Email: "john@example.com",
	}
	endTime := time.Now()
	tx := &domain.Transaction{
		ID:         "tx-123",
		StartTime:  endTime.Add(-90 * time.Minute),
		EndTime:    &endTime,
		MeterStart: 1000,
		MeterStop:  1025.5,
	}

	// Act
	err := service.SendChargingCompleted(context.Background(), user, tx, 45.50)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if !strings.Contains(email.Body, "25.50") {
		t.Error("expected body to contain energy delivered")
	}
	if !strings.Contains(email.Body, "45.50") {
		t.Error("expected body to contain cost")
	}
}

func TestService_SendPasswordReset_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)
	service.loadTemplates()

	user := &domain.User{
		ID:    "user-123",
		Name:  "John Doe",
		Email: "john@example.com",
	}
	resetToken := "abc123xyz"

	// Act
	err := service.SendPasswordReset(context.Background(), user, resetToken)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if !strings.Contains(email.Body, "reset-password?token=abc123xyz") {
		t.Error("expected body to contain reset URL with token")
	}
}

func TestService_SendInvoice_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)
	service.loadTemplates()

	user := &domain.User{
		ID:    "user-123",
		Name:  "John Doe",
		Email: "john@example.com",
	}
	invoice := &ports.Invoice{
		ID:            "inv-123",
		TransactionID: "tx-123",
		Amount:        45.50,
		Currency:      "BRL",
		EnergyKWh:     25.5,
		Duration:      "1h 30m",
		StationName:   "ABB Terra 184",
		Date:          "2024-01-15",
	}

	// Act
	err := service.SendInvoice(context.Background(), user, invoice)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if !strings.Contains(email.Body, "inv-123") {
		t.Error("expected body to contain invoice ID")
	}
	if !strings.Contains(email.Body, "45.50") {
		t.Error("expected body to contain amount")
	}
}

func TestService_SendLowBalance_Success(t *testing.T) {
	// Arrange
	mockProvider := &MockProvider{}
	service := newTestService(mockProvider)
	service.loadTemplates()

	user := &domain.User{
		ID:    "user-123",
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// Act
	err := service.SendLowBalance(context.Background(), user, 15.00)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mockProvider.SentEmails))
	}
	email := mockProvider.SentEmails[0]
	if !strings.Contains(email.Body, "15.00") {
		t.Error("expected body to contain balance amount")
	}
	if !strings.Contains(email.Body, "Low Balance") {
		t.Error("expected body to contain low balance warning")
	}
}

func TestNewService_SendGridProvider(t *testing.T) {
	// Arrange
	config := &Config{
		Provider:       "sendgrid",
		SendGridAPIKey: "test-api-key",
		FromEmail:      "test@example.com",
		FromName:       "Test",
	}

	// Act
	service, err := NewService(config, newTestLogger())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if service == nil {
		t.Fatal("expected service, got nil")
	}
	if _, ok := service.provider.(*SendGridProvider); !ok {
		t.Error("expected SendGridProvider")
	}
}

func TestNewService_SMTPProvider(t *testing.T) {
	// Arrange
	config := &Config{
		Provider:  "smtp",
		SMTPHost:  "localhost",
		SMTPPort:  1025,
		FromEmail: "test@example.com",
		FromName:  "Test",
	}

	// Act
	service, err := NewService(config, newTestLogger())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if service == nil {
		t.Fatal("expected service, got nil")
	}
	if _, ok := service.provider.(*SMTPProvider); !ok {
		t.Error("expected SMTPProvider")
	}
}

func TestNewService_UnknownProvider(t *testing.T) {
	// Arrange
	config := &Config{
		Provider: "unknown",
	}

	// Act
	_, err := NewService(config, newTestLogger())

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown email provider") {
		t.Errorf("expected 'unknown email provider' error, got '%s'", err.Error())
	}
}

func TestNewService_SendGridMissingAPIKey(t *testing.T) {
	// Arrange
	config := &Config{
		Provider:       "sendgrid",
		SendGridAPIKey: "", // Missing
	}

	// Act
	_, err := NewService(config, newTestLogger())

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "API key is required") {
		t.Errorf("expected 'API key is required' error, got '%s'", err.Error())
	}
}

func TestDefaultConfig(t *testing.T) {
	// Act
	config := DefaultConfig()

	// Assert
	if config.Provider != "smtp" {
		t.Errorf("expected provider 'smtp', got '%s'", config.Provider)
	}
	if config.SMTPHost != "localhost" {
		t.Errorf("expected SMTP host 'localhost', got '%s'", config.SMTPHost)
	}
	if config.SMTPPort != 1025 {
		t.Errorf("expected SMTP port 1025, got %d", config.SMTPPort)
	}
}
