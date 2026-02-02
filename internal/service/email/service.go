package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Provider defines the interface for email providers
type Provider interface {
	Send(ctx context.Context, to, subject, body string, isHTML bool) error
}

// Config holds email service configuration
type Config struct {
	// Provider type: "sendgrid" or "smtp"
	Provider string

	// From email address
	FromEmail string
	FromName  string

	// SendGrid configuration
	SendGridAPIKey string

	// SMTP configuration (for Mailhog or other SMTP servers)
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPUseTLS   bool

	// Template configuration
	TemplateDir string
	BaseURL     string // Base URL for links in emails
}

// DefaultConfig returns a default configuration for development (Mailhog)
func DefaultConfig() *Config {
	return &Config{
		Provider:   "smtp",
		FromEmail:  "noreply@sigec-ve.com",
		FromName:   "SIGEC-VE",
		SMTPHost:   "localhost",
		SMTPPort:   1025, // Mailhog default port
		SMTPUseTLS: false,
		BaseURL:    "http://localhost:3000",
	}
}

// Service implements the EmailService interface
type Service struct {
	config    *Config
	provider  Provider
	templates map[string]*template.Template
	log       *zap.Logger
}

// NewService creates a new email service
func NewService(config *Config, log *zap.Logger) (*Service, error) {
	if config == nil {
		config = DefaultConfig()
	}

	s := &Service{
		config:    config,
		templates: make(map[string]*template.Template),
		log:       log,
	}

	// Initialize provider
	switch config.Provider {
	case "sendgrid":
		if config.SendGridAPIKey == "" {
			return nil, fmt.Errorf("SendGrid API key is required")
		}
		s.provider = NewSendGridProvider(config.SendGridAPIKey, config.FromEmail, config.FromName)
	case "smtp":
		s.provider = NewSMTPProvider(
			config.SMTPHost,
			config.SMTPPort,
			config.SMTPUsername,
			config.SMTPPassword,
			config.FromEmail,
			config.FromName,
			config.SMTPUseTLS,
		)
	default:
		return nil, fmt.Errorf("unknown email provider: %s", config.Provider)
	}

	// Load templates
	s.loadTemplates()

	return s, nil
}

// loadTemplates loads all email templates
func (s *Service) loadTemplates() {
	s.templates["welcome"] = template.Must(template.New("welcome").Parse(welcomeTemplate))
	s.templates["charging_started"] = template.Must(template.New("charging_started").Parse(chargingStartedTemplate))
	s.templates["charging_completed"] = template.Must(template.New("charging_completed").Parse(chargingCompletedTemplate))
	s.templates["password_reset"] = template.Must(template.New("password_reset").Parse(passwordResetTemplate))
	s.templates["invoice"] = template.Must(template.New("invoice").Parse(invoiceTemplate))
	s.templates["low_balance"] = template.Must(template.New("low_balance").Parse(lowBalanceTemplate))
}

// Send sends a generic email
func (s *Service) Send(ctx context.Context, to, subject, body string) error {
	s.log.Info("Sending email",
		zap.String("to", to),
		zap.String("subject", subject),
	)

	if err := s.provider.Send(ctx, to, subject, body, false); err != nil {
		s.log.Error("Failed to send email",
			zap.String("to", to),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendHTML sends an HTML email
func (s *Service) SendHTML(ctx context.Context, to, subject, htmlBody string) error {
	s.log.Info("Sending HTML email",
		zap.String("to", to),
		zap.String("subject", subject),
	)

	if err := s.provider.Send(ctx, to, subject, htmlBody, true); err != nil {
		s.log.Error("Failed to send HTML email",
			zap.String("to", to),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send HTML email: %w", err)
	}

	return nil
}

// SendTemplate sends an email using a template
func (s *Service) SendTemplate(ctx context.Context, to, templateName string, data map[string]interface{}) error {
	tmpl, ok := s.templates[templateName]
	if !ok {
		return fmt.Errorf("template not found: %s", templateName)
	}

	// Add base URL to data
	if data == nil {
		data = make(map[string]interface{})
	}
	data["BaseURL"] = s.config.BaseURL

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	subject, ok := data["Subject"].(string)
	if !ok {
		subject = "Notification from SIGEC-VE"
	}

	return s.SendHTML(ctx, to, subject, buf.String())
}

// SendWelcome sends a welcome email to a new user
func (s *Service) SendWelcome(ctx context.Context, user *domain.User) error {
	data := map[string]interface{}{
		"Subject":  "Welcome to SIGEC-VE!",
		"UserName": user.Name,
		"Email":    user.Email,
	}

	return s.SendTemplate(ctx, user.Email, "welcome", data)
}

// SendChargingStarted sends a notification when charging starts
func (s *Service) SendChargingStarted(ctx context.Context, user *domain.User, tx *domain.Transaction, station *domain.ChargePoint) error {
	stationName := ""
	if station != nil {
		stationName = fmt.Sprintf("%s %s", station.Vendor, station.Model)
	}

	data := map[string]interface{}{
		"Subject":       "Charging Session Started",
		"UserName":      user.Name,
		"TransactionID": tx.ID,
		"StationName":   stationName,
		"StartTime":     tx.StartTime.Format("2006-01-02 15:04:05"),
	}

	return s.SendTemplate(ctx, user.Email, "charging_started", data)
}

// SendChargingCompleted sends a notification when charging completes
func (s *Service) SendChargingCompleted(ctx context.Context, user *domain.User, tx *domain.Transaction, cost float64) error {
	duration := ""
	if tx.EndTime != nil {
		dur := tx.EndTime.Sub(tx.StartTime)
		hours := int(dur.Hours())
		minutes := int(dur.Minutes()) % 60
		if hours > 0 {
			duration = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			duration = fmt.Sprintf("%dm", minutes)
		}
	}

	data := map[string]interface{}{
		"Subject":       "Charging Session Completed",
		"UserName":      user.Name,
		"TransactionID": tx.ID,
		"EnergyKWh":     fmt.Sprintf("%.2f", tx.MeterStop-tx.MeterStart),
		"Duration":      duration,
		"Cost":          fmt.Sprintf("%.2f", cost),
		"Currency":      "BRL",
	}

	return s.SendTemplate(ctx, user.Email, "charging_completed", data)
}

// SendPasswordReset sends a password reset email
func (s *Service) SendPasswordReset(ctx context.Context, user *domain.User, resetToken string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.BaseURL, resetToken)

	data := map[string]interface{}{
		"Subject":  "Reset Your Password",
		"UserName": user.Name,
		"ResetURL": resetURL,
	}

	return s.SendTemplate(ctx, user.Email, "password_reset", data)
}

// SendInvoice sends an invoice email
func (s *Service) SendInvoice(ctx context.Context, user *domain.User, invoice *ports.Invoice) error {
	data := map[string]interface{}{
		"Subject":       fmt.Sprintf("Invoice #%s", invoice.ID),
		"UserName":      user.Name,
		"InvoiceID":     invoice.ID,
		"TransactionID": invoice.TransactionID,
		"Amount":        fmt.Sprintf("%.2f", invoice.Amount),
		"Currency":      invoice.Currency,
		"EnergyKWh":     fmt.Sprintf("%.2f", invoice.EnergyKWh),
		"Duration":      invoice.Duration,
		"StationName":   invoice.StationName,
		"Date":          invoice.Date,
	}

	return s.SendTemplate(ctx, user.Email, "invoice", data)
}

// SendLowBalance sends a low balance warning
func (s *Service) SendLowBalance(ctx context.Context, user *domain.User, balance float64) error {
	data := map[string]interface{}{
		"Subject":  "Low Balance Warning",
		"UserName": user.Name,
		"Balance":  fmt.Sprintf("%.2f", balance),
		"Currency": "BRL",
	}

	return s.SendTemplate(ctx, user.Email, "low_balance", data)
}
