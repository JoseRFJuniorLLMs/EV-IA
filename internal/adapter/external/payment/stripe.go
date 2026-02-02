package payment

import (
	"context"
	"errors"

	"go.uber.org/zap"

	// "github.com/stripe/stripe-go/v76"
	// "github.com/stripe/stripe-go/v76/paymentintent"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

type StripeService struct {
	apiKey string
	log    *zap.Logger
}

func NewStripeService(apiKey string, log *zap.Logger) ports.PaymentGateway {
	// stripe.Key = apiKey
	return &StripeService{
		apiKey: apiKey,
		log:    log,
	}
}

func (s *StripeService) CreatePaymentIntent(ctx context.Context, amount float64, currency string, customerID string) (string, error) {
	s.log.Info("Creating mock payment intent", zap.Float64("amount", amount), zap.String("currency", currency))

	// Mock implementation for now as we don't have the stripe dependency in go.mod and want to avoid fetch errors
	if amount <= 0 {
		return "", errors.New("invalid amount")
	}

	// In real impl:
	// params := &stripe.PaymentIntentParams{
	//     Amount:   stripe.Int64(int64(amount * 100)),
	//     Currency: stripe.String(currency),
	//     Customer: stripe.String(customerID),
	// }
	// pi, err := paymentintent.New(params)
	// return pi.ID, err

	return "pi_mock_123456789", nil
}

func (s *StripeService) ConfirmPayment(ctx context.Context, paymentID string) error {
	s.log.Info("Confirming mock payment", zap.String("paymentID", paymentID))
	return nil
}

func (s *StripeService) RefundPayment(ctx context.Context, paymentID string) error {
	s.log.Info("Refunding mock payment", zap.String("paymentID", paymentID))
	return nil
}
