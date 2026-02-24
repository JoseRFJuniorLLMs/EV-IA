// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package nietzsche

import (
	"context"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
	"go.uber.org/zap"
)

type UserRepository struct {
	db  *DB
	log *zap.Logger
}

func NewUserRepository(db *DB, log *zap.Logger) ports.UserRepository {
	return &UserRepository{db: db, log: log}
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	m, err := ToMap(user)
	if err != nil {
		return err
	}
	_, err = r.db.Insert(ctx, "users", m)
	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	m, err := r.db.QueryFirst(ctx, "users", " AND n.id = $id", map[string]interface{}{"id": id})
	if err != nil || m == nil {
		return nil, err
	}
	u := &domain.User{}
	if err := FromMap(m, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	m, err := r.db.QueryFirst(ctx, "users", " AND n.email = $email", map[string]interface{}{"email": email})
	if err != nil || m == nil {
		return nil, err
	}
	u := &domain.User{}
	if err := FromMap(m, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByDocument(ctx context.Context, document string) (*domain.User, error) {
	m, err := r.db.QueryFirst(ctx, "users", " AND n.document = $doc", map[string]interface{}{"doc": document})
	if err != nil || m == nil {
		return nil, err
	}
	u := &domain.User{}
	if err := FromMap(m, u); err != nil {
		return nil, err
	}
	return u, nil
}
