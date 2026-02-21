package payment

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

type StripeService struct {
	apiKey string
	log    *zap.Logger
}

func NewStripeService(apiKey string, log *zap.Logger) ports.PaymentGateway {
	stripe.Key = apiKey
	return &StripeService{
		apiKey: apiKey,
		log:    log,
	}
}

func (s *StripeService) CreatePaymentIntent(ctx context.Context, amount float64, currency string, customerID string) (string, error) {
	if amount <= 0 {
		return "", errors.New("invalid amount")
	}

	s.log.Info("Creating payment intent",
		zap.Float64("amount", amount),
		zap.String("currency", currency),
		zap.String("customer_id", customerID),
	)

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount * 100)),
		Currency: stripe.String(currency),
	}
	if customerID != "" {
		params.Customer = stripe.String(customerID)
	}
	params.Context = ctx

	pi, err := paymentintent.New(params)
	if err != nil {
		s.log.Error("Failed to create payment intent", zap.Error(err))
		return "", fmt.Errorf("stripe: create payment intent: %w", err)
	}

	s.log.Info("Payment intent created",
		zap.String("payment_intent_id", pi.ID),
		zap.String("status", string(pi.Status)),
	)

	return pi.ID, nil
}

func (s *StripeService) ConfirmPayment(ctx context.Context, paymentID string) error {
	if paymentID == "" {
		return errors.New("payment ID is required")
	}

	s.log.Info("Confirming payment", zap.String("payment_id", paymentID))

	params := &stripe.PaymentIntentConfirmParams{}
	params.Context = ctx

	pi, err := paymentintent.Confirm(paymentID, params)
	if err != nil {
		s.log.Error("Failed to confirm payment", zap.String("payment_id", paymentID), zap.Error(err))
		return fmt.Errorf("stripe: confirm payment: %w", err)
	}

	s.log.Info("Payment confirmed",
		zap.String("payment_id", pi.ID),
		zap.String("status", string(pi.Status)),
	)

	return nil
}

func (s *StripeService) RefundPayment(ctx context.Context, paymentID string) error {
	if paymentID == "" {
		return errors.New("payment ID is required")
	}

	s.log.Info("Refunding payment", zap.String("payment_id", paymentID))

	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentID),
	}
	params.Context = ctx

	r, err := refund.New(params)
	if err != nil {
		s.log.Error("Failed to refund payment", zap.String("payment_id", paymentID), zap.Error(err))
		return fmt.Errorf("stripe: refund payment: %w", err)
	}

	s.log.Info("Payment refunded",
		zap.String("refund_id", r.ID),
		zap.String("status", string(r.Status)),
	)

	return nil
}
