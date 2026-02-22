package postgres

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// ISO15118Repository implements ISO 15118 certificate persistence
type ISO15118Repository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewISO15118Repository creates a new ISO 15118 repository
func NewISO15118Repository(db *gorm.DB, log *zap.Logger) *ISO15118Repository {
	return &ISO15118Repository{
		db:  db,
		log: log,
	}
}

// StoreCertificate stores a new certificate
func (r *ISO15118Repository) StoreCertificate(ctx context.Context, cert *domain.ISO15118Certificate) error {
	result := r.db.WithContext(ctx).Create(cert)
	if result.Error != nil {
		r.log.Error("Failed to store ISO 15118 certificate",
			zap.String("emaid", cert.EMAID),
			zap.Error(result.Error),
		)
		return result.Error
	}

	r.log.Debug("Stored ISO 15118 certificate",
		zap.String("emaid", cert.EMAID),
		zap.String("contractID", cert.ContractID),
	)
	return nil
}

// GetCertificateByEMAID retrieves a certificate by EMAID
func (r *ISO15118Repository) GetCertificateByEMAID(ctx context.Context, emaid string) (*domain.ISO15118Certificate, error) {
	var cert domain.ISO15118Certificate
	result := r.db.WithContext(ctx).First(&cert, "emaid = ?", emaid)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("certificate not found")
		}
		return nil, result.Error
	}
	return &cert, nil
}

// GetCertificateByContractID retrieves a certificate by contract ID
func (r *ISO15118Repository) GetCertificateByContractID(ctx context.Context, contractID string) (*domain.ISO15118Certificate, error) {
	var cert domain.ISO15118Certificate
	result := r.db.WithContext(ctx).First(&cert, "contract_id = ?", contractID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("certificate not found")
		}
		return nil, result.Error
	}
	return &cert, nil
}

// GetCertificateByVIN retrieves certificates by vehicle VIN
func (r *ISO15118Repository) GetCertificateByVIN(ctx context.Context, vin string) ([]*domain.ISO15118Certificate, error) {
	var certs []domain.ISO15118Certificate
	result := r.db.WithContext(ctx).
		Where("vehicle_vin = ?", vin).
		Order("created_at DESC").
		Find(&certs)
	if result.Error != nil {
		return nil, result.Error
	}

	results := make([]*domain.ISO15118Certificate, len(certs))
	for i := range certs {
		results[i] = &certs[i]
	}
	return results, nil
}

// UpdateCertificate updates an existing certificate
func (r *ISO15118Repository) UpdateCertificate(ctx context.Context, cert *domain.ISO15118Certificate) error {
	cert.UpdatedAt = time.Now()
	result := r.db.WithContext(ctx).Save(cert)
	if result.Error != nil {
		r.log.Error("Failed to update ISO 15118 certificate",
			zap.String("emaid", cert.EMAID),
			zap.Error(result.Error),
		)
		return result.Error
	}
	return nil
}

// GetExpiringCertificates retrieves certificates expiring within N days
func (r *ISO15118Repository) GetExpiringCertificates(ctx context.Context, daysUntilExpiry int) ([]*domain.ISO15118Certificate, error) {
	expiryDate := time.Now().AddDate(0, 0, daysUntilExpiry)

	var certs []domain.ISO15118Certificate
	result := r.db.WithContext(ctx).
		Where("valid_to <= ? AND revoked = false", expiryDate).
		Order("valid_to ASC").
		Find(&certs)
	if result.Error != nil {
		return nil, result.Error
	}

	results := make([]*domain.ISO15118Certificate, len(certs))
	for i := range certs {
		results[i] = &certs[i]
	}
	return results, nil
}

// GetV2GCapableCertificates retrieves all V2G-capable certificates
func (r *ISO15118Repository) GetV2GCapableCertificates(ctx context.Context) ([]*domain.ISO15118Certificate, error) {
	now := time.Now()

	var certs []domain.ISO15118Certificate
	result := r.db.WithContext(ctx).
		Where("v2g_capable = true AND revoked = false AND valid_to > ?", now).
		Order("created_at DESC").
		Find(&certs)
	if result.Error != nil {
		return nil, result.Error
	}

	results := make([]*domain.ISO15118Certificate, len(certs))
	for i := range certs {
		results[i] = &certs[i]
	}
	return results, nil
}

// GetValidCertificates retrieves all valid (non-revoked, non-expired) certificates
func (r *ISO15118Repository) GetValidCertificates(ctx context.Context) ([]domain.ISO15118Certificate, error) {
	now := time.Now()

	var certs []domain.ISO15118Certificate
	result := r.db.WithContext(ctx).
		Where("revoked = false AND valid_from <= ? AND valid_to > ?", now, now).
		Order("created_at DESC").
		Find(&certs)
	if result.Error != nil {
		return nil, result.Error
	}

	return certs, nil
}

// CountCertificates counts certificates with optional filters
func (r *ISO15118Repository) CountCertificates(ctx context.Context, v2gCapable *bool, revoked *bool) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.ISO15118Certificate{})

	if v2gCapable != nil {
		query = query.Where("v2g_capable = ?", *v2gCapable)
	}
	if revoked != nil {
		query = query.Where("revoked = ?", *revoked)
	}

	var count int64
	result := query.Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

// DeleteCertificate soft-deletes a certificate (marks as revoked)
func (r *ISO15118Repository) DeleteCertificate(ctx context.Context, emaid, reason string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&domain.ISO15118Certificate{}).
		Where("emaid = ?", emaid).
		Updates(map[string]interface{}{
			"revoked":           true,
			"revoked_at":        now,
			"revocation_reason": reason,
			"updated_at":        now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("certificate not found")
	}
	return nil
}

// CertificateStats represents certificate statistics
type CertificateStats struct {
	Total        int64 `json:"total"`
	Valid        int64 `json:"valid"`
	Revoked      int64 `json:"revoked"`
	Expired      int64 `json:"expired"`
	ExpiringSoon int64 `json:"expiring_soon"`
	V2GCapable   int64 `json:"v2g_capable"`
}

// GetCertificateStats returns certificate statistics
func (r *ISO15118Repository) GetCertificateStats(ctx context.Context) (*CertificateStats, error) {
	stats := &CertificateStats{}
	now := time.Now()
	thirtyDaysFromNow := now.AddDate(0, 0, 30)

	r.db.WithContext(ctx).Model(&domain.ISO15118Certificate{}).Count(&stats.Total)

	r.db.WithContext(ctx).Model(&domain.ISO15118Certificate{}).
		Where("revoked = false AND valid_from <= ? AND valid_to > ?", now, now).
		Count(&stats.Valid)

	r.db.WithContext(ctx).Model(&domain.ISO15118Certificate{}).
		Where("revoked = true").
		Count(&stats.Revoked)

	r.db.WithContext(ctx).Model(&domain.ISO15118Certificate{}).
		Where("revoked = false AND valid_to <= ?", now).
		Count(&stats.Expired)

	r.db.WithContext(ctx).Model(&domain.ISO15118Certificate{}).
		Where("revoked = false AND valid_to > ? AND valid_to <= ?", now, thirtyDaysFromNow).
		Count(&stats.ExpiringSoon)

	r.db.WithContext(ctx).Model(&domain.ISO15118Certificate{}).
		Where("v2g_capable = true AND revoked = false AND valid_to > ?", now).
		Count(&stats.V2GCapable)

	return stats, nil
}
