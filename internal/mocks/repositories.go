package mocks

import (
	"context"
	"time"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	SaveFunc           func(ctx context.Context, user *domain.User) error
	FindByIDFunc       func(ctx context.Context, id string) (*domain.User, error)
	FindByEmailFunc    func(ctx context.Context, email string) (*domain.User, error)
	FindByDocumentFunc func(ctx context.Context, document string) (*domain.User, error)
}

func (m *MockUserRepository) Save(ctx context.Context, user *domain.User) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) FindByDocument(ctx context.Context, document string) (*domain.User, error) {
	if m.FindByDocumentFunc != nil {
		return m.FindByDocumentFunc(ctx, document)
	}
	return nil, nil
}

// MockChargePointRepository is a mock implementation of ChargePointRepository
type MockChargePointRepository struct {
	SaveFunc         func(ctx context.Context, cp *domain.ChargePoint) error
	FindByIDFunc     func(ctx context.Context, id string) (*domain.ChargePoint, error)
	FindAllFunc      func(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error)
	UpdateStatusFunc func(ctx context.Context, id string, status domain.ChargePointStatus) error
	FindNearbyFunc   func(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error)
}

func (m *MockChargePointRepository) Save(ctx context.Context, cp *domain.ChargePoint) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, cp)
	}
	return nil
}

func (m *MockChargePointRepository) FindByID(ctx context.Context, id string) (*domain.ChargePoint, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockChargePointRepository) FindAll(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc(ctx, filter)
	}
	return []domain.ChargePoint{}, nil
}

func (m *MockChargePointRepository) UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockChargePointRepository) FindNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error) {
	if m.FindNearbyFunc != nil {
		return m.FindNearbyFunc(ctx, lat, lon, radius)
	}
	return []domain.ChargePoint{}, nil
}

// MockTransactionRepository is a mock implementation of TransactionRepository
type MockTransactionRepository struct {
	SaveFunc                func(ctx context.Context, tx *domain.Transaction) error
	FindByIDFunc            func(ctx context.Context, id string) (*domain.Transaction, error)
	FindActiveByUserIDFunc  func(ctx context.Context, userID string) (*domain.Transaction, error)
	FindHistoryByUserIDFunc func(ctx context.Context, userID string) ([]domain.Transaction, error)
	FindByDateFunc          func(ctx context.Context, date time.Time) ([]domain.Transaction, error)
	UpdateFunc              func(ctx context.Context, tx *domain.Transaction) error
}

func (m *MockTransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, tx)
	}
	return nil
}

func (m *MockTransactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTransactionRepository) FindActiveByUserID(ctx context.Context, userID string) (*domain.Transaction, error) {
	if m.FindActiveByUserIDFunc != nil {
		return m.FindActiveByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockTransactionRepository) FindHistoryByUserID(ctx context.Context, userID string) ([]domain.Transaction, error) {
	if m.FindHistoryByUserIDFunc != nil {
		return m.FindHistoryByUserIDFunc(ctx, userID)
	}
	return []domain.Transaction{}, nil
}

func (m *MockTransactionRepository) FindByDate(ctx context.Context, date time.Time) ([]domain.Transaction, error) {
	if m.FindByDateFunc != nil {
		return m.FindByDateFunc(ctx, date)
	}
	return []domain.Transaction{}, nil
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, tx)
	}
	return nil
}
