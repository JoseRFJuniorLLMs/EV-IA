// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package nietzsche

import (
	"context"
	"time"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"go.uber.org/zap"
)

// CertificateStats holds aggregated certificate statistics.
type CertificateStats struct {
	Total        int64
	Valid        int64
	Revoked      int64
	Expired      int64
	ExpiringSoon int64
	V2GCapable   int64
}

type ISO15118Repository struct {
	db  *DB
	log *zap.Logger
}

func NewISO15118Repository(db *DB, log *zap.Logger) *ISO15118Repository {
	return &ISO15118Repository{db: db, log: log}
}

func (r *ISO15118Repository) StoreCertificate(ctx context.Context, cert *domain.ISO15118Certificate) error {
	m, err := ToMap(cert)
	if err != nil {
		return err
	}
	_, err = r.db.Insert(ctx, "iso15118_certificates", m)
	return err
}

func (r *ISO15118Repository) GetCertificateByEMAID(ctx context.Context, emaid string) (*domain.ISO15118Certificate, error) {
	m, err := r.db.QueryFirst(ctx, "iso15118_certificates",
		" AND n.emaid = $emaid",
		map[string]interface{}{"emaid": emaid})
	if err != nil || m == nil {
		return nil, err
	}
	c := &domain.ISO15118Certificate{}
	return c, FromMap(m, c)
}

func (r *ISO15118Repository) GetCertificateByContractID(ctx context.Context, contractID string) (*domain.ISO15118Certificate, error) {
	m, err := r.db.QueryFirst(ctx, "iso15118_certificates",
		" AND n.contract_id = $cid",
		map[string]interface{}{"cid": contractID})
	if err != nil || m == nil {
		return nil, err
	}
	c := &domain.ISO15118Certificate{}
	return c, FromMap(m, c)
}

func (r *ISO15118Repository) GetCertificateByVIN(ctx context.Context, vin string) ([]*domain.ISO15118Certificate, error) {
	rows, err := r.db.QueryByLabel(ctx, "iso15118_certificates",
		" AND n.vehicle_vin = $vin",
		map[string]interface{}{"vin": vin})
	if err != nil {
		return nil, err
	}
	var certs []*domain.ISO15118Certificate
	for _, m := range rows {
		c := &domain.ISO15118Certificate{}
		if err := FromMap(m, c); err == nil {
			certs = append(certs, c)
		}
	}
	return certs, nil
}

func (r *ISO15118Repository) UpdateCertificate(ctx context.Context, cert *domain.ISO15118Certificate) error {
	m, err := ToMap(cert)
	if err != nil {
		return err
	}
	delete(m, "id")
	delete(m, "node_label")
	delete(m, "created_at")
	return r.db.UpdateFields(ctx, "iso15118_certificates", cert.ID, m)
}

func (r *ISO15118Repository) GetExpiringCertificates(ctx context.Context, daysUntilExpiry int) ([]*domain.ISO15118Certificate, error) {
	cutoff := time.Now().Add(time.Duration(daysUntilExpiry) * 24 * time.Hour)
	now := time.Now()

	rows, err := r.db.QueryByLabel(ctx, "iso15118_certificates", "", nil)
	if err != nil {
		return nil, err
	}
	var certs []*domain.ISO15118Certificate
	for _, m := range rows {
		if GetBool(m, "revoked") {
			continue
		}
		validTo := GetTime(m, "valid_to")
		if validTo.After(now) && validTo.Before(cutoff) {
			c := &domain.ISO15118Certificate{}
			if err := FromMap(m, c); err == nil {
				certs = append(certs, c)
			}
		}
	}
	return certs, nil
}

func (r *ISO15118Repository) GetV2GCapableCertificates(ctx context.Context) ([]*domain.ISO15118Certificate, error) {
	rows, err := r.db.QueryByLabel(ctx, "iso15118_certificates", "", nil)
	if err != nil {
		return nil, err
	}
	var certs []*domain.ISO15118Certificate
	for _, m := range rows {
		if GetBool(m, "v2g_capable") && !GetBool(m, "revoked") {
			c := &domain.ISO15118Certificate{}
			if err := FromMap(m, c); err == nil {
				certs = append(certs, c)
			}
		}
	}
	return certs, nil
}

func (r *ISO15118Repository) GetValidCertificates(ctx context.Context) ([]domain.ISO15118Certificate, error) {
	now := time.Now()
	rows, err := r.db.QueryByLabel(ctx, "iso15118_certificates", "", nil)
	if err != nil {
		return nil, err
	}
	var certs []domain.ISO15118Certificate
	for _, m := range rows {
		validFrom := GetTime(m, "valid_from")
		validTo := GetTime(m, "valid_to")
		if !validFrom.After(now) && validTo.After(now) && !GetBool(m, "revoked") {
			c := domain.ISO15118Certificate{}
			if err := FromMap(m, &c); err == nil {
				certs = append(certs, c)
			}
		}
	}
	return certs, nil
}

func (r *ISO15118Repository) CountCertificates(ctx context.Context, v2gCapable *bool, revoked *bool) (int64, error) {
	rows, err := r.db.QueryByLabel(ctx, "iso15118_certificates", "", nil)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, m := range rows {
		if v2gCapable != nil && GetBool(m, "v2g_capable") != *v2gCapable {
			continue
		}
		if revoked != nil && GetBool(m, "revoked") != *revoked {
			continue
		}
		count++
	}
	return count, nil
}

func (r *ISO15118Repository) DeleteCertificate(ctx context.Context, emaid, reason string) error {
	m, err := r.db.QueryFirst(ctx, "iso15118_certificates",
		" AND n.emaid = $emaid",
		map[string]interface{}{"emaid": emaid})
	if err != nil || m == nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	return r.db.UpdateFields(ctx, "iso15118_certificates", GetString(m, "id"), map[string]interface{}{
		"revoked":           true,
		"revoked_at":        now,
		"revocation_reason": reason,
	})
}

func (r *ISO15118Repository) GetCertificateStats(ctx context.Context) (*CertificateStats, error) {
	now := time.Now()
	soon := now.Add(30 * 24 * time.Hour)
	rows, err := r.db.QueryByLabel(ctx, "iso15118_certificates", "", nil)
	if err != nil {
		return nil, err
	}
	stats := &CertificateStats{}
	for _, m := range rows {
		stats.Total++
		revoked := GetBool(m, "revoked")
		validTo := GetTime(m, "valid_to")
		validFrom := GetTime(m, "valid_from")
		v2g := GetBool(m, "v2g_capable")

		if revoked {
			stats.Revoked++
		} else if validTo.Before(now) {
			stats.Expired++
		} else if !validFrom.After(now) && validTo.After(now) {
			stats.Valid++
			if validTo.Before(soon) {
				stats.ExpiringSoon++
			}
		}
		if v2g {
			stats.V2GCapable++
		}
	}
	return stats, nil
}
