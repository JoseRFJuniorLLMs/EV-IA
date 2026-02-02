package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Provider defines the interface for payment providers
type Provider interface {
	// CreatePaymentIntent creates a payment intent
	CreatePaymentIntent(ctx context.Context, amount float64, currency string, metadata map[string]string) (*domain.PaymentIntent, error)

	// ProcessPayment processes a payment
	ProcessPayment(ctx context.Context, amount float64, currency string, paymentMethodID string, metadata map[string]string) (string, error)

	// CreatePixPayment creates a PIX payment
	CreatePixPayment(ctx context.Context, amount float64, description string, expiresIn time.Duration) (*domain.PixPayment, string, error)

	// CreateBoletoPayment creates a Boleto payment
	CreateBoletoPayment(ctx context.Context, amount float64, customerInfo map[string]string, expiresAt time.Time) (*domain.BoletoPayment, string, error)

	// RefundPayment refunds a payment
	RefundPayment(ctx context.Context, paymentID string, amount float64) (string, error)

	// GetPayment retrieves payment details from provider
	GetPayment(ctx context.Context, paymentID string) (*ProviderPayment, error)

	// ValidateWebhook validates webhook signature
	ValidateWebhook(payload []byte, signature string) error

	// ParseWebhook parses webhook payload
	ParseWebhook(payload []byte) (*WebhookEvent, error)

	// Name returns the provider name
	Name() string
}

// ProviderPayment represents payment info from provider
type ProviderPayment struct {
	ID       string
	Status   domain.PaymentStatus
	Amount   float64
	Currency string
}

// WebhookEvent represents a webhook event from provider
type WebhookEvent struct {
	Type      string // payment.completed, payment.failed, etc
	PaymentID string
	Status    domain.PaymentStatus
	Amount    float64
	Metadata  map[string]string
}

// Config holds payment service configuration
type Config struct {
	DefaultProvider domain.PaymentProvider
	DefaultCurrency string

	// Stripe config
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string

	// PagSeguro config
	PagSeguroEmail   string
	PagSeguroToken   string
	PagSeguroSandbox bool
}

// Service implements PaymentService interface
type Service struct {
	config    *Config
	providers map[domain.PaymentProvider]Provider
	repo      ports.PaymentRepository
	walletSvc ports.WalletService
	log       *zap.Logger
}

// NewService creates a new payment service
func NewService(config *Config, repo ports.PaymentRepository, walletSvc ports.WalletService, log *zap.Logger) (*Service, error) {
	s := &Service{
		config:    config,
		providers: make(map[domain.PaymentProvider]Provider),
		repo:      repo,
		walletSvc: walletSvc,
		log:       log,
	}

	// Initialize Stripe provider if configured
	if config.StripeSecretKey != "" {
		stripeProvider := NewStripeProvider(config.StripeSecretKey, config.StripeWebhookSecret)
		s.providers[domain.PaymentProviderStripe] = stripeProvider
		log.Info("Stripe payment provider initialized")
	}

	// Initialize PagSeguro provider if configured
	if config.PagSeguroToken != "" {
		pagSeguroProvider := NewPagSeguroProvider(config.PagSeguroEmail, config.PagSeguroToken, config.PagSeguroSandbox)
		s.providers[domain.PaymentProviderPagSeguro] = pagSeguroProvider
		log.Info("PagSeguro payment provider initialized")
	}

	if len(s.providers) == 0 {
		log.Warn("No payment providers configured")
	}

	return s, nil
}

// getProvider returns the appropriate provider
func (s *Service) getProvider(provider domain.PaymentProvider) (Provider, error) {
	if provider == "" {
		provider = s.config.DefaultProvider
	}
	p, ok := s.providers[provider]
	if !ok {
		return nil, fmt.Errorf("payment provider not configured: %s", provider)
	}
	return p, nil
}

// CreatePaymentIntent creates a payment intent for client-side confirmation
func (s *Service) CreatePaymentIntent(ctx context.Context, userID string, amount float64, currency string) (*domain.PaymentIntent, error) {
	if currency == "" {
		currency = s.config.DefaultCurrency
	}

	provider, err := s.getProvider(s.config.DefaultProvider)
	if err != nil {
		return nil, err
	}

	metadata := map[string]string{
		"user_id": userID,
	}

	intent, err := provider.CreatePaymentIntent(ctx, amount, currency, metadata)
	if err != nil {
		s.log.Error("Failed to create payment intent",
			zap.String("user_id", userID),
			zap.Float64("amount", amount),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	return intent, nil
}

// ProcessPayment processes a payment
func (s *Service) ProcessPayment(ctx context.Context, req *ports.PaymentRequest) (*domain.Payment, error) {
	provider, err := s.getProvider(s.config.DefaultProvider)
	if err != nil {
		return nil, err
	}

	currency := req.Currency
	if currency == "" {
		currency = s.config.DefaultCurrency
	}

	// Create payment record
	payment := &domain.Payment{
		ID:            uuid.New().String(),
		UserID:        req.UserID,
		TransactionID: req.TransactionID,
		Provider:      s.config.DefaultProvider,
		Method:        req.Method,
		Status:        domain.PaymentStatusProcessing,
		Amount:        req.Amount,
		Currency:      currency,
		Description:   req.Description,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save initial payment record
	if err := s.repo.SavePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	// Process with provider
	metadata := map[string]string{
		"user_id":        req.UserID,
		"payment_id":     payment.ID,
		"transaction_id": req.TransactionID,
	}

	providerID, err := provider.ProcessPayment(ctx, req.Amount, currency, req.CardID, metadata)
	if err != nil {
		payment.Status = domain.PaymentStatusFailed
		payment.FailureReason = err.Error()
		payment.UpdatedAt = time.Now()
		s.repo.SavePayment(ctx, payment)

		s.log.Error("Payment processing failed",
			zap.String("payment_id", payment.ID),
			zap.Error(err),
		)
		return payment, fmt.Errorf("payment processing failed: %w", err)
	}

	// Update payment with provider ID
	payment.ProviderID = providerID
	payment.Status = domain.PaymentStatusCompleted
	now := time.Now()
	payment.CompletedAt = &now
	payment.UpdatedAt = now

	if err := s.repo.SavePayment(ctx, payment); err != nil {
		s.log.Error("Failed to update payment record",
			zap.String("payment_id", payment.ID),
			zap.Error(err),
		)
	}

	s.log.Info("Payment processed successfully",
		zap.String("payment_id", payment.ID),
		zap.String("provider_id", providerID),
		zap.Float64("amount", req.Amount),
	)

	return payment, nil
}

// ProcessChargingPayment processes payment for a charging transaction
func (s *Service) ProcessChargingPayment(ctx context.Context, userID string, transactionID string, amount float64) (*domain.Payment, error) {
	// First try to use wallet balance
	if s.walletSvc != nil {
		hasFunds, err := s.walletSvc.HasSufficientBalance(ctx, userID, amount)
		if err == nil && hasFunds {
			// Deduct from wallet
			err = s.walletSvc.DeductFunds(ctx, userID, amount, "Charging session payment", transactionID)
			if err == nil {
				// Create payment record for wallet payment
				payment := &domain.Payment{
					ID:            uuid.New().String(),
					UserID:        userID,
					TransactionID: transactionID,
					Provider:      "wallet",
					Method:        domain.PaymentMethodWallet,
					Status:        domain.PaymentStatusCompleted,
					Amount:        amount,
					Currency:      s.config.DefaultCurrency,
					Description:   "Charging session payment from wallet",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				now := time.Now()
				payment.CompletedAt = &now

				if err := s.repo.SavePayment(ctx, payment); err != nil {
					s.log.Error("Failed to save wallet payment record", zap.Error(err))
				}

				s.log.Info("Charging payment processed from wallet",
					zap.String("user_id", userID),
					zap.String("transaction_id", transactionID),
					zap.Float64("amount", amount),
				)
				return payment, nil
			}
		}
	}

	// Fall back to card payment
	return s.ProcessPayment(ctx, &ports.PaymentRequest{
		UserID:        userID,
		Amount:        amount,
		Currency:      s.config.DefaultCurrency,
		Method:        domain.PaymentMethodCreditCard,
		TransactionID: transactionID,
		Description:   "Charging session payment",
	})
}

// GetPayment retrieves a payment by ID
func (s *Service) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	return s.repo.GetPayment(ctx, paymentID)
}

// GetPaymentHistory retrieves payment history for a user
func (s *Service) GetPaymentHistory(ctx context.Context, userID string, limit, offset int) ([]domain.Payment, error) {
	return s.repo.GetPaymentsByUser(ctx, userID, limit, offset)
}

// RefundPayment refunds a payment
func (s *Service) RefundPayment(ctx context.Context, paymentID string, amount float64, reason string) (*domain.Refund, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	if payment.Status != domain.PaymentStatusCompleted {
		return nil, fmt.Errorf("can only refund completed payments")
	}

	if amount <= 0 {
		amount = payment.Amount // Full refund
	}

	if amount > payment.Amount {
		return nil, fmt.Errorf("refund amount exceeds payment amount")
	}

	provider, err := s.getProvider(payment.Provider)
	if err != nil {
		return nil, err
	}

	// Process refund with provider
	refundID, err := provider.RefundPayment(ctx, payment.ProviderID, amount)
	if err != nil {
		s.log.Error("Refund failed",
			zap.String("payment_id", paymentID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("refund failed: %w", err)
	}

	// Create refund record
	refund := &domain.Refund{
		ID:         uuid.New().String(),
		PaymentID:  paymentID,
		ProviderID: refundID,
		Amount:     amount,
		Status:     domain.PaymentStatusCompleted,
		Reason:     reason,
		CreatedAt:  time.Now(),
	}
	now := time.Now()
	refund.CompletedAt = &now

	if err := s.repo.SaveRefund(ctx, refund); err != nil {
		s.log.Error("Failed to save refund record", zap.Error(err))
	}

	// Update payment status if full refund
	if amount == payment.Amount {
		payment.Status = domain.PaymentStatusRefunded
		payment.UpdatedAt = time.Now()
		s.repo.SavePayment(ctx, payment)
	}

	s.log.Info("Refund processed",
		zap.String("payment_id", paymentID),
		zap.String("refund_id", refund.ID),
		zap.Float64("amount", amount),
	)

	return refund, nil
}

// CreatePixPayment creates a PIX payment
func (s *Service) CreatePixPayment(ctx context.Context, userID string, amount float64) (*domain.PixPayment, *domain.Payment, error) {
	provider, err := s.getProvider(domain.PaymentProviderPagSeguro)
	if err != nil {
		// Fallback to Stripe if available
		provider, err = s.getProvider(domain.PaymentProviderStripe)
		if err != nil {
			return nil, nil, fmt.Errorf("no provider available for PIX payments")
		}
	}

	// Create payment record
	payment := &domain.Payment{
		ID:        uuid.New().String(),
		UserID:    userID,
		Provider:  domain.PaymentProviderPagSeguro,
		Method:    domain.PaymentMethodPix,
		Status:    domain.PaymentStatusPending,
		Amount:    amount,
		Currency:  "BRL",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create PIX with provider
	pixPayment, providerID, err := provider.CreatePixPayment(ctx, amount, "SIGEC-VE Recarga", 30*time.Minute)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create PIX payment: %w", err)
	}

	payment.ProviderID = providerID
	if err := s.repo.SavePayment(ctx, payment); err != nil {
		s.log.Error("Failed to save PIX payment record", zap.Error(err))
	}

	s.log.Info("PIX payment created",
		zap.String("payment_id", payment.ID),
		zap.Float64("amount", amount),
	)

	return pixPayment, payment, nil
}

// CreateBoletoPayment creates a Boleto payment
func (s *Service) CreateBoletoPayment(ctx context.Context, userID string, amount float64) (*domain.BoletoPayment, *domain.Payment, error) {
	provider, err := s.getProvider(domain.PaymentProviderPagSeguro)
	if err != nil {
		return nil, nil, fmt.Errorf("PagSeguro required for Boleto payments")
	}

	// Create payment record
	payment := &domain.Payment{
		ID:        uuid.New().String(),
		UserID:    userID,
		Provider:  domain.PaymentProviderPagSeguro,
		Method:    domain.PaymentMethodBoleto,
		Status:    domain.PaymentStatusPending,
		Amount:    amount,
		Currency:  "BRL",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create Boleto with provider - expires in 3 days
	expiresAt := time.Now().AddDate(0, 0, 3)
	customerInfo := map[string]string{
		"user_id": userID,
	}

	boletoPayment, providerID, err := provider.CreateBoletoPayment(ctx, amount, customerInfo, expiresAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Boleto: %w", err)
	}

	payment.ProviderID = providerID
	if err := s.repo.SavePayment(ctx, payment); err != nil {
		s.log.Error("Failed to save Boleto payment record", zap.Error(err))
	}

	s.log.Info("Boleto payment created",
		zap.String("payment_id", payment.ID),
		zap.Float64("amount", amount),
	)

	return boletoPayment, payment, nil
}

// HandleWebhook handles payment provider webhooks
func (s *Service) HandleWebhook(ctx context.Context, providerName string, payload []byte, signature string) error {
	var provider Provider
	var providerType domain.PaymentProvider

	switch providerName {
	case "stripe":
		providerType = domain.PaymentProviderStripe
	case "pagseguro":
		providerType = domain.PaymentProviderPagSeguro
	default:
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	var err error
	provider, err = s.getProvider(providerType)
	if err != nil {
		return err
	}

	// Validate signature
	if err := provider.ValidateWebhook(payload, signature); err != nil {
		s.log.Warn("Invalid webhook signature",
			zap.String("provider", providerName),
			zap.Error(err),
		)
		return fmt.Errorf("invalid webhook signature: %w", err)
	}

	// Parse event
	event, err := provider.ParseWebhook(payload)
	if err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	s.log.Info("Webhook received",
		zap.String("provider", providerName),
		zap.String("type", event.Type),
		zap.String("payment_id", event.PaymentID),
	)

	// Find payment by provider ID
	payment, err := s.repo.GetPaymentByProviderID(ctx, event.PaymentID)
	if err != nil {
		s.log.Warn("Payment not found for webhook",
			zap.String("provider_id", event.PaymentID),
		)
		return nil // Don't error, might be a test event
	}

	// Update payment status
	payment.Status = event.Status
	payment.UpdatedAt = time.Now()

	if event.Status == domain.PaymentStatusCompleted {
		now := time.Now()
		payment.CompletedAt = &now

		// Add funds to wallet if this is a wallet top-up
		if s.walletSvc != nil && payment.TransactionID == "" {
			if err := s.walletSvc.AddFunds(ctx, payment.UserID, payment.Amount, payment.ID); err != nil {
				s.log.Error("Failed to add funds to wallet", zap.Error(err))
			}
		}
	}

	if err := s.repo.SavePayment(ctx, payment); err != nil {
		s.log.Error("Failed to update payment from webhook", zap.Error(err))
		return err
	}

	return nil
}
