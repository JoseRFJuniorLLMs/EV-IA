package ports

import (
	"context"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, string, error) // token, refresh, err
	Register(ctx context.Context, user *domain.User) error
	RefreshToken(ctx context.Context, token string) (string, error)
	ValidateToken(ctx context.Context, token string) (*domain.User, error)
}

type DeviceService interface {
	GetDevice(ctx context.Context, id string) (*domain.ChargePoint, error)
	ListDevices(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error)
	UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error
	GetNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error)
}

type TransactionService interface {
	StartTransaction(ctx context.Context, deviceID string, connectorID int, userID string, idTag string) (*domain.Transaction, error)
	StopTransaction(ctx context.Context, transactionID string) (*domain.Transaction, error)
	GetTransaction(ctx context.Context, id string) (*domain.Transaction, error)
	GetActiveTransaction(ctx context.Context, userID string) (*domain.Transaction, error)
	GetTransactionHistory(ctx context.Context, userID string) ([]domain.Transaction, error)
}
