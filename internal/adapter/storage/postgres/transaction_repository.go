package postgres

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

type TransactionRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewTransactionRepository(db *gorm.DB, log *zap.Logger) ports.TransactionRepository {
	return &TransactionRepository{
		db:  db,
		log: log,
	}
}

func (r *TransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	return r.db.WithContext(ctx).Save(tx).Error
}

func (r *TransactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	var tx domain.Transaction
	err := r.db.WithContext(ctx).First(&tx, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepository) FindActiveByUserID(ctx context.Context, userID string) (*domain.Transaction, error) {
	var tx domain.Transaction
	// Assuming "Started" means active. Also might define "Charging", "Suspended" etc.
	err := r.db.WithContext(ctx).Where("user_id = ? AND status IN ?", userID, []domain.TransactionStatus{domain.TransactionStatusStarted}).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepository) FindHistoryByUserID(ctx context.Context, userID string) ([]domain.Transaction, error) {
	var txs []domain.Transaction
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&txs).Error
	return txs, err
}

func (r *TransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	return r.db.WithContext(ctx).Save(tx).Error
}

func (r *TransactionRepository) FindByDate(ctx context.Context, date time.Time) ([]domain.Transaction, error) {
	var txs []domain.Transaction
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	err := r.db.WithContext(ctx).Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).Find(&txs).Error
	return txs, err
}
