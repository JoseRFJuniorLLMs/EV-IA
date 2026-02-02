package ports

import (
	"context"
)

type PaymentGateway interface {
	CreatePaymentIntent(ctx context.Context, amount float64, currency string, customerID string) (string, error)
	ConfirmPayment(ctx context.Context, paymentID string) error
	RefundPayment(ctx context.Context, paymentID string) error
}
