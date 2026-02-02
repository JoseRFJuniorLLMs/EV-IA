package ports

import (
	"context"

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
}
