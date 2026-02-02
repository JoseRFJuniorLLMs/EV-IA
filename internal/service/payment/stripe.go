package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// StripeProvider implements the Provider interface for Stripe
type StripeProvider struct {
	secretKey     string
	webhookSecret string
}

// NewStripeProvider creates a new Stripe provider
func NewStripeProvider(secretKey, webhookSecret string) *StripeProvider {
	stripe.Key = secretKey
	return &StripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// Name returns the provider name
func (p *StripeProvider) Name() string {
	return "stripe"
}

// CreatePaymentIntent creates a Stripe payment intent
func (p *StripeProvider) CreatePaymentIntent(ctx context.Context, amount float64, currency string, metadata map[string]string) (*domain.PaymentIntent, error) {
	// Stripe expects amount in cents
	amountCents := int64(amount * 100)

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	// Add metadata
	if metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = v
		}
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe error: %w", err)
	}

	return &domain.PaymentIntent{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Amount:       amount,
		Currency:     currency,
		Status:       string(pi.Status),
	}, nil
}

// ProcessPayment processes a payment with Stripe
func (p *StripeProvider) ProcessPayment(ctx context.Context, amount float64, currency string, paymentMethodID string, metadata map[string]string) (string, error) {
	amountCents := int64(amount * 100)

	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(amountCents),
		Currency:      stripe.String(currency),
		PaymentMethod: stripe.String(paymentMethodID),
		Confirm:       stripe.Bool(true),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled:        stripe.Bool(true),
			AllowRedirects: stripe.String("never"),
		},
	}

	if metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range metadata {
			params.Metadata[k] = v
		}
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe payment error: %w", err)
	}

	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		return "", fmt.Errorf("payment not succeeded: %s", pi.Status)
	}

	return pi.ID, nil
}

// CreatePixPayment creates a PIX payment via Stripe (Brazil)
func (p *StripeProvider) CreatePixPayment(ctx context.Context, amount float64, description string, expiresIn time.Duration) (*domain.PixPayment, string, error) {
	// Stripe doesn't natively support PIX, this would need Stripe Brazil or a local provider
	// For now, return an error suggesting to use PagSeguro
	return nil, "", fmt.Errorf("PIX payments require PagSeguro provider")
}

// CreateBoletoPayment creates a Boleto payment via Stripe
func (p *StripeProvider) CreateBoletoPayment(ctx context.Context, amount float64, customerInfo map[string]string, expiresAt time.Time) (*domain.BoletoPayment, string, error) {
	// Stripe doesn't natively support Boleto
	return nil, "", fmt.Errorf("Boleto payments require PagSeguro provider")
}

// RefundPayment refunds a Stripe payment
func (p *StripeProvider) RefundPayment(ctx context.Context, paymentID string, amount float64) (string, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentID),
	}

	if amount > 0 {
		params.Amount = stripe.Int64(int64(amount * 100))
	}

	r, err := refund.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe refund error: %w", err)
	}

	return r.ID, nil
}

// GetPayment retrieves payment details from Stripe
func (p *StripeProvider) GetPayment(ctx context.Context, paymentID string) (*ProviderPayment, error) {
	pi, err := paymentintent.Get(paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get payment error: %w", err)
	}

	status := domain.PaymentStatusPending
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		status = domain.PaymentStatusCompleted
	case stripe.PaymentIntentStatusCanceled:
		status = domain.PaymentStatusCancelled
	case stripe.PaymentIntentStatusProcessing:
		status = domain.PaymentStatusProcessing
	case stripe.PaymentIntentStatusRequiresPaymentMethod, stripe.PaymentIntentStatusRequiresAction:
		status = domain.PaymentStatusPending
	}

	return &ProviderPayment{
		ID:       pi.ID,
		Status:   status,
		Amount:   float64(pi.Amount) / 100,
		Currency: string(pi.Currency),
	}, nil
}

// ValidateWebhook validates Stripe webhook signature
func (p *StripeProvider) ValidateWebhook(payload []byte, signature string) error {
	_, err := webhook.ConstructEvent(payload, signature, p.webhookSecret)
	return err
}

// ParseWebhook parses Stripe webhook payload
func (p *StripeProvider) ParseWebhook(payload []byte) (*WebhookEvent, error) {
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	webhookEvent := &WebhookEvent{
		Type:     string(event.Type),
		Metadata: make(map[string]string),
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, err
		}
		webhookEvent.PaymentID = pi.ID
		webhookEvent.Status = domain.PaymentStatusCompleted
		webhookEvent.Amount = float64(pi.Amount) / 100
		for k, v := range pi.Metadata {
			webhookEvent.Metadata[k] = v
		}

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, err
		}
		webhookEvent.PaymentID = pi.ID
		webhookEvent.Status = domain.PaymentStatusFailed
		webhookEvent.Amount = float64(pi.Amount) / 100

	case "charge.refunded":
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
			return nil, err
		}
		webhookEvent.PaymentID = charge.PaymentIntent.ID
		webhookEvent.Status = domain.PaymentStatusRefunded
		webhookEvent.Amount = float64(charge.AmountRefunded) / 100

	default:
		webhookEvent.Status = domain.PaymentStatusPending
	}

	return webhookEvent, nil
}

// CreateCustomer creates a Stripe customer
func (p *StripeProvider) CreateCustomer(ctx context.Context, email, name string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	// Import the customer package when needed
	// c, err := customer.New(params)
	// if err != nil {
	// 	return "", fmt.Errorf("stripe customer error: %w", err)
	// }
	// return c.ID, nil

	_ = params
	return "", fmt.Errorf("not implemented")
}

// AttachPaymentMethod attaches a payment method to a customer
func (p *StripeProvider) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	// pm, err := paymentmethod.Attach(paymentMethodID, &stripe.PaymentMethodAttachParams{
	// 	Customer: stripe.String(customerID),
	// })
	// return err
	return fmt.Errorf("not implemented")
}
