package notification

import (
	"context"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
	"github.com/seu-repo/sigec-ve/internal/service/email"
)

// EmailAdapter bridges the notification layer to the email service
type EmailAdapter struct {
	svc *email.Service
	log *zap.Logger
}

// NewEmailAdapter creates an email adapter wrapping the email service
func NewEmailAdapter(cfg *email.Config, log *zap.Logger) (*EmailAdapter, error) {
	svc, err := email.NewService(cfg, log)
	if err != nil {
		return nil, err
	}
	return &EmailAdapter{svc: svc, log: log}, nil
}

// Ensure EmailAdapter implements ports.EmailService
var _ ports.EmailService = (*EmailAdapter)(nil)

func (a *EmailAdapter) Send(ctx context.Context, to, subject, body string) error {
	return a.svc.Send(ctx, to, subject, body)
}

func (a *EmailAdapter) SendHTML(ctx context.Context, to, subject, htmlBody string) error {
	return a.svc.SendHTML(ctx, to, subject, htmlBody)
}

func (a *EmailAdapter) SendTemplate(ctx context.Context, to, templateName string, data map[string]interface{}) error {
	return a.svc.SendTemplate(ctx, to, templateName, data)
}

func (a *EmailAdapter) SendWelcome(ctx context.Context, user *domain.User) error {
	return a.svc.SendWelcome(ctx, user)
}

func (a *EmailAdapter) SendChargingStarted(ctx context.Context, user *domain.User, tx *domain.Transaction, station *domain.ChargePoint) error {
	return a.svc.SendChargingStarted(ctx, user, tx, station)
}

func (a *EmailAdapter) SendChargingCompleted(ctx context.Context, user *domain.User, tx *domain.Transaction, cost float64) error {
	return a.svc.SendChargingCompleted(ctx, user, tx, cost)
}

func (a *EmailAdapter) SendPasswordReset(ctx context.Context, user *domain.User, resetToken string) error {
	return a.svc.SendPasswordReset(ctx, user, resetToken)
}

func (a *EmailAdapter) SendInvoice(ctx context.Context, user *domain.User, invoice *ports.Invoice) error {
	return a.svc.SendInvoice(ctx, user, invoice)
}

func (a *EmailAdapter) SendLowBalance(ctx context.Context, user *domain.User, balance float64) error {
	return a.svc.SendLowBalance(ctx, user, balance)
}
