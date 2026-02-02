package ports

import (
	"context"
	"time"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

type ChargePointRepository interface {
	Save(ctx context.Context, cp *domain.ChargePoint) error
	FindByID(ctx context.Context, id string) (*domain.ChargePoint, error)
	FindAll(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error)
	UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error
	FindNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error)
}

type TransactionRepository interface {
	Save(ctx context.Context, tx *domain.Transaction) error
	FindByID(ctx context.Context, id string) (*domain.Transaction, error)
	FindActiveByUserID(ctx context.Context, userID string) (*domain.Transaction, error)
	FindHistoryByUserID(ctx context.Context, userID string) ([]domain.Transaction, error)
	Update(ctx context.Context, tx *domain.Transaction) error
}

type UserRepository interface {
	Save(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
}

// PaymentRepository handles payment persistence
type PaymentRepository interface {
	SavePayment(ctx context.Context, payment *domain.Payment) error
	GetPayment(ctx context.Context, id string) (*domain.Payment, error)
	GetPaymentByProviderID(ctx context.Context, providerID string) (*domain.Payment, error)
	GetPaymentsByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Payment, error)
	GetPaymentsByTransaction(ctx context.Context, transactionID string) ([]domain.Payment, error)
	SaveRefund(ctx context.Context, refund *domain.Refund) error
	GetRefundsByPayment(ctx context.Context, paymentID string) ([]domain.Refund, error)
}

// CardRepository handles payment card persistence
type CardRepository interface {
	Save(ctx context.Context, card *domain.PaymentCard) error
	GetByID(ctx context.Context, id string) (*domain.PaymentCard, error)
	GetByUserID(ctx context.Context, userID string) ([]domain.PaymentCard, error)
	Delete(ctx context.Context, id string) error
	SetDefault(ctx context.Context, userID, cardID string) error
}

// WalletRepository handles wallet persistence
type WalletRepository interface {
	Save(ctx context.Context, wallet *domain.Wallet) error
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
	GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error)
	SaveTransaction(ctx context.Context, tx *domain.WalletTransaction) error
	GetTransactions(ctx context.Context, walletID string, limit, offset int) ([]domain.WalletTransaction, error)
}

// ReservationRepository handles reservation persistence
type ReservationRepository interface {
	Save(ctx context.Context, reservation *domain.Reservation) error
	GetByID(ctx context.Context, id string) (*domain.Reservation, error)
	GetByUserID(ctx context.Context, userID string, status string, limit, offset int) ([]domain.Reservation, error)
	GetByChargePointID(ctx context.Context, chargePointID string, date time.Time) ([]domain.Reservation, error)
	GetByTimeRange(ctx context.Context, chargePointID string, connectorID int, startTime, endTime time.Time) ([]domain.Reservation, error)
	GetActiveByUserID(ctx context.Context, userID string) ([]domain.Reservation, error)
	GetExpired(ctx context.Context, gracePeriod time.Duration) ([]domain.Reservation, error)
	UpdateStatus(ctx context.Context, id string, status domain.ReservationStatus) error
	Delete(ctx context.Context, id string) error
	CountByUserAndStatus(ctx context.Context, userID string, statuses []domain.ReservationStatus) (int, error)
}

// AlertRepository handles alert persistence
type AlertRepository interface {
	Save(ctx context.Context, alert *Alert) error
	GetByID(ctx context.Context, id string) (*Alert, error)
	GetAll(ctx context.Context, acknowledged bool, limit, offset int) ([]Alert, error)
	Acknowledge(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	CountUnacknowledged(ctx context.Context) (int, error)
}

// Alert entity for repository
type Alert struct {
	ID           string
	Type         string
	Severity     string
	Title        string
	Message      string
	Source       string
	SourceID     string
	Acknowledged bool
	CreatedAt    time.Time
}

// TripRepository handles trip persistence
type TripRepository interface {
	Save(ctx context.Context, trip *domain.Trip) error
	GetByID(ctx context.Context, id string) (*domain.Trip, error)
	GetByUserID(ctx context.Context, userID string, status string, limit, offset int) ([]domain.Trip, error)
	Delete(ctx context.Context, id string) error
}
