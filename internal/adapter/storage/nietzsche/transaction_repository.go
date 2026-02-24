// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package nietzsche

import (
	"context"
	"sort"
	"time"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
	"go.uber.org/zap"
)

type TransactionRepository struct {
	db  *DB
	log *zap.Logger
}

func NewTransactionRepository(db *DB, log *zap.Logger) ports.TransactionRepository {
	return &TransactionRepository{db: db, log: log}
}

func (r *TransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	m, err := ToMap(tx)
	if err != nil {
		return err
	}
	_, err = r.db.Insert(ctx, "transactions", m)
	return err
}

func (r *TransactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	m, err := r.db.QueryFirst(ctx, "transactions", " AND n.id = $id", map[string]interface{}{"id": id})
	if err != nil || m == nil {
		return nil, err
	}
	tx := &domain.Transaction{}
	if err := FromMap(m, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *TransactionRepository) FindActiveByUserID(ctx context.Context, userID string) (*domain.Transaction, error) {
	rows, err := r.db.QueryByLabel(ctx, "transactions",
		" AND n.user_id = $uid AND n.status = $st",
		map[string]interface{}{"uid": userID, "st": string(domain.TransactionStatusStarted)})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	tx := &domain.Transaction{}
	if err := FromMap(rows[0], tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *TransactionRepository) FindHistoryByUserID(ctx context.Context, userID string) ([]domain.Transaction, error) {
	rows, err := r.db.QueryByLabel(ctx, "transactions",
		" AND n.user_id = $uid",
		map[string]interface{}{"uid": userID})
	if err != nil {
		return nil, err
	}
	var txs []domain.Transaction
	for _, m := range rows {
		var tx domain.Transaction
		if err := FromMap(m, &tx); err == nil {
			txs = append(txs, tx)
		}
	}
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].CreatedAt.After(txs[j].CreatedAt)
	})
	return txs, nil
}

func (r *TransactionRepository) FindByDate(ctx context.Context, date time.Time) ([]domain.Transaction, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := r.db.QueryByLabel(ctx, "transactions", "", nil)
	if err != nil {
		return nil, err
	}
	var txs []domain.Transaction
	for _, m := range rows {
		createdAt := GetTime(m, "created_at")
		if !createdAt.Before(dayStart) && createdAt.Before(dayEnd) {
			var tx domain.Transaction
			if err := FromMap(m, &tx); err == nil {
				txs = append(txs, tx)
			}
		}
	}
	return txs, nil
}

func (r *TransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	m, err := ToMap(tx)
	if err != nil {
		return err
	}
	delete(m, "id")
	delete(m, "node_label")
	delete(m, "created_at")
	return r.db.UpdateFields(ctx, "transactions", tx.ID, m)
}
