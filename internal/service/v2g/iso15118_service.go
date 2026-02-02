package v2g

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// ISO15118Service handles ISO 15118 Plug & Charge operations
type ISO15118Service struct {
	repo           ports.ISO15118Repository
	log            *zap.Logger
	config         *ISO15118Config
	certCache      *certCache
	ocspClient     *OCSPClient
}

// ISO15118Config holds ISO 15118 configuration
type ISO15118Config struct {
	RootCACertPath      string        // Path to V2G Root CA certificate
	SubCACertPath       string        // Path to Sub CA certificate
	CPOCertPath         string        // Path to CPO (Charge Point Operator) certificate
	CPOKeyPath          string        // Path to CPO private key
	OCSPResponderURL    string        // OCSP responder URL for certificate status
	CertCacheDuration   time.Duration // How long to cache certificate validation
	EnableOCSP          bool          // Enable OCSP checking
	EnableCRL           bool          // Enable CRL checking
	AllowExpiredCerts   bool          // Allow expired certs (for testing)
}

// DefaultISO15118Config returns default ISO 15118 configuration
func DefaultISO15118Config() *ISO15118Config {
	return &ISO15118Config{
		CertCacheDuration: 1 * time.Hour,
		EnableOCSP:        true,
		EnableCRL:         false, // CRL checking is more expensive
		AllowExpiredCerts: false,
	}
}

// certCache caches certificate validation results
type certCache struct {
	validations map[string]*certValidation
	mu          sync.RWMutex
}

type certValidation struct {
	valid     bool
	reason    string
	validAt   time.Time
	expiresAt time.Time
}

// OCSPClient handles OCSP certificate status checking
type OCSPClient struct {
	responderURL string
	httpTimeout  time.Duration
}

// NewISO15118Service creates a new ISO 15118 service
func NewISO15118Service(
	repo ports.ISO15118Repository,
	log *zap.Logger,
	config *ISO15118Config,
) *ISO15118Service {
	if config == nil {
		config = DefaultISO15118Config()
	}

	return &ISO15118Service{
		repo:   repo,
		log:    log,
		config: config,
		certCache: &certCache{
			validations: make(map[string]*certValidation),
		},
		ocspClient: &OCSPClient{
			responderURL: config.OCSPResponderURL,
			httpTimeout:  10 * time.Second,
		},
	}
}

// ISO15118Certificate represents a stored ISO 15118 certificate
type ISO15118Certificate struct {
	ID                   string    `json:"id"`
	EMAID                string    `json:"emaid"` // E-Mobility Account Identifier
	ContractID           string    `json:"contract_id"`
	VehicleVIN           string    `json:"vehicle_vin,omitempty"`
	CertificatePEM       string    `json:"certificate_pem"`
	CertificateChain     string    `json:"certificate_chain,omitempty"`
	PrivateKeyEncrypted  string    `json:"private_key_encrypted,omitempty"`
	V2GCapable           bool      `json:"v2g_capable"`
	ValidFrom            time.Time `json:"valid_from"`
	ValidTo              time.Time `json:"valid_to"`
	Revoked              bool      `json:"revoked"`
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
	RevocationReason     string    `json:"revocation_reason,omitempty"`
	ProviderID           string    `json:"provider_id,omitempty"`
	MaxChargePowerKW     float64   `json:"max_charge_power_kw,omitempty"`
	MaxDischargePowerKW  float64   `json:"max_discharge_power_kw,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// AuthenticateVehicle authenticates a vehicle using its ISO 15118 certificate
func (s *ISO15118Service) AuthenticateVehicle(ctx context.Context, certChain []byte) (*domain.ISO15118VehicleIdentity, error) {
	// Parse the certificate chain
	certs, err := s.parseCertificateChain(certChain)
	if err != nil {
		s.log.Error("Failed to parse certificate chain",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to parse certificate chain: %w", err)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates in chain")
	}

	// The first certificate should be the leaf (vehicle/contract certificate)
	leafCert := certs[0]

	// Validate the certificate
	if err := s.ValidateCertificate(ctx, certChain); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	// Extract vehicle identity from certificate
	identity := s.extractVehicleIdentity(leafCert)

	// Check if certificate exists in database
	storedCert, err := s.repo.GetCertificateByEMAID(ctx, identity.EMAID)
	if err != nil {
		s.log.Warn("Certificate not found in database, creating new entry",
			zap.String("emaid", identity.EMAID),
		)

		// Store new certificate
		newCert := &ISO15118Certificate{
			EMAID:            identity.EMAID,
			ContractID:       identity.ContractID,
			VehicleVIN:       identity.VehicleVIN,
			CertificatePEM:   string(certChain),
			V2GCapable:       identity.V2GCapable,
			ValidFrom:        leafCert.NotBefore,
			ValidTo:          leafCert.NotAfter,
			Revoked:          false,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := s.repo.StoreCertificate(ctx, newCert); err != nil {
			s.log.Error("Failed to store certificate",
				zap.String("emaid", identity.EMAID),
				zap.Error(err),
			)
			// Don't fail authentication just because we couldn't store it
		}
	} else {
		// Check if stored certificate is revoked
		if storedCert.Revoked {
			return nil, fmt.Errorf("certificate has been revoked: %s", storedCert.RevocationReason)
		}

		// Update V2G capability info from stored cert
		identity.V2GCapable = storedCert.V2GCapable
	}

	s.log.Info("Vehicle authenticated via ISO 15118",
		zap.String("emaid", identity.EMAID),
		zap.String("contractID", identity.ContractID),
		zap.Bool("v2gCapable", identity.V2GCapable),
	)

	return identity, nil
}

// ValidateCertificate validates an ISO 15118 certificate chain
func (s *ISO15118Service) ValidateCertificate(ctx context.Context, certPEM []byte) error {
	// Check cache first
	cacheKey := string(certPEM)[:64] // Use first 64 bytes as key
	if cached := s.getCachedValidation(cacheKey); cached != nil {
		if cached.valid {
			return nil
		}
		return fmt.Errorf("cached validation failure: %s", cached.reason)
	}

	// Parse certificates
	certs, err := s.parseCertificateChain(certPEM)
	if err != nil {
		s.cacheValidation(cacheKey, false, err.Error())
		return err
	}

	if len(certs) == 0 {
		s.cacheValidation(cacheKey, false, "no certificates in chain")
		return fmt.Errorf("no certificates in chain")
	}

	leafCert := certs[0]

	// Check certificate validity period
	now := time.Now()
	if now.Before(leafCert.NotBefore) {
		reason := "certificate not yet valid"
		s.cacheValidation(cacheKey, false, reason)
		return fmt.Errorf(reason)
	}

	if now.After(leafCert.NotAfter) && !s.config.AllowExpiredCerts {
		reason := "certificate has expired"
		s.cacheValidation(cacheKey, false, reason)
		return fmt.Errorf(reason)
	}

	// Verify certificate chain
	if err := s.verifyCertificateChain(certs); err != nil {
		s.cacheValidation(cacheKey, false, err.Error())
		return err
	}

	// Check OCSP status if enabled
	if s.config.EnableOCSP && s.config.OCSPResponderURL != "" {
		if err := s.checkOCSPStatus(ctx, leafCert); err != nil {
			s.log.Warn("OCSP check failed, continuing without OCSP validation",
				zap.Error(err),
			)
			// Don't fail on OCSP errors - it might be a network issue
		}
	}

	// Cache successful validation
	s.cacheValidation(cacheKey, true, "")

	return nil
}

// GetChargingContract retrieves the charging contract for a vehicle
func (s *ISO15118Service) GetChargingContract(ctx context.Context, emaid string) (*domain.ChargingContract, error) {
	cert, err := s.repo.GetCertificateByEMAID(ctx, emaid)
	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}

	contract := &domain.ChargingContract{
		ContractID:          cert.ContractID,
		EMAID:               cert.EMAID,
		ProviderID:          cert.ProviderID,
		ValidFrom:           cert.ValidFrom,
		ValidTo:             cert.ValidTo,
		MaxChargePowerKW:    cert.MaxChargePowerKW,
		MaxDischargePowerKW: cert.MaxDischargePowerKW,
		V2GEnabled:          cert.V2GCapable,
	}

	return contract, nil
}

// RevokeCertificate revokes an ISO 15118 certificate
func (s *ISO15118Service) RevokeCertificate(ctx context.Context, emaid, reason string) error {
	cert, err := s.repo.GetCertificateByEMAID(ctx, emaid)
	if err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}

	now := time.Now()
	cert.Revoked = true
	cert.RevokedAt = &now
	cert.RevocationReason = reason
	cert.UpdatedAt = now

	if err := s.repo.UpdateCertificate(ctx, cert); err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	// Invalidate cache
	s.invalidateCacheForEMAID(emaid)

	s.log.Info("Certificate revoked",
		zap.String("emaid", emaid),
		zap.String("reason", reason),
	)

	return nil
}

// InstallCertificate installs a new contract certificate for a vehicle
func (s *ISO15118Service) InstallCertificate(ctx context.Context, req *InstallCertificateRequest) error {
	// Parse and validate the certificate
	certs, err := s.parseCertificateChain([]byte(req.CertificatePEM))
	if err != nil {
		return fmt.Errorf("invalid certificate: %w", err)
	}

	if len(certs) == 0 {
		return fmt.Errorf("no certificate in PEM data")
	}

	leafCert := certs[0]

	// Extract identity from certificate
	identity := s.extractVehicleIdentity(leafCert)

	cert := &ISO15118Certificate{
		EMAID:               identity.EMAID,
		ContractID:          identity.ContractID,
		VehicleVIN:          req.VehicleVIN,
		CertificatePEM:      req.CertificatePEM,
		CertificateChain:    req.CertificateChain,
		PrivateKeyEncrypted: req.PrivateKeyEncrypted,
		V2GCapable:          req.V2GCapable,
		ValidFrom:           leafCert.NotBefore,
		ValidTo:             leafCert.NotAfter,
		Revoked:             false,
		ProviderID:          req.ProviderID,
		MaxChargePowerKW:    req.MaxChargePowerKW,
		MaxDischargePowerKW: req.MaxDischargePowerKW,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := s.repo.StoreCertificate(ctx, cert); err != nil {
		return fmt.Errorf("failed to store certificate: %w", err)
	}

	s.log.Info("Certificate installed",
		zap.String("emaid", cert.EMAID),
		zap.String("contractID", cert.ContractID),
		zap.Bool("v2gCapable", cert.V2GCapable),
	)

	return nil
}

// InstallCertificateRequest represents a certificate installation request
type InstallCertificateRequest struct {
	CertificatePEM      string  `json:"certificate_pem"`
	CertificateChain    string  `json:"certificate_chain,omitempty"`
	PrivateKeyEncrypted string  `json:"private_key_encrypted,omitempty"`
	VehicleVIN          string  `json:"vehicle_vin,omitempty"`
	ProviderID          string  `json:"provider_id,omitempty"`
	V2GCapable          bool    `json:"v2g_capable"`
	MaxChargePowerKW    float64 `json:"max_charge_power_kw,omitempty"`
	MaxDischargePowerKW float64 `json:"max_discharge_power_kw,omitempty"`
}

// GetCertificateStatus gets the status of a certificate
func (s *ISO15118Service) GetCertificateStatus(ctx context.Context, emaid string) (*CertificateStatus, error) {
	cert, err := s.repo.GetCertificateByEMAID(ctx, emaid)
	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}

	now := time.Now()
	status := &CertificateStatus{
		EMAID:          cert.EMAID,
		ContractID:     cert.ContractID,
		Valid:          !cert.Revoked && now.After(cert.ValidFrom) && now.Before(cert.ValidTo),
		Revoked:        cert.Revoked,
		RevokedAt:      cert.RevokedAt,
		RevocationReason: cert.RevocationReason,
		ValidFrom:      cert.ValidFrom,
		ValidTo:        cert.ValidTo,
		DaysUntilExpiry: int(cert.ValidTo.Sub(now).Hours() / 24),
		V2GCapable:     cert.V2GCapable,
	}

	if status.DaysUntilExpiry < 0 {
		status.Expired = true
		status.DaysUntilExpiry = 0
	}

	return status, nil
}

// CertificateStatus represents the status of an ISO 15118 certificate
type CertificateStatus struct {
	EMAID            string     `json:"emaid"`
	ContractID       string     `json:"contract_id"`
	Valid            bool       `json:"valid"`
	Expired          bool       `json:"expired"`
	Revoked          bool       `json:"revoked"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason string     `json:"revocation_reason,omitempty"`
	ValidFrom        time.Time  `json:"valid_from"`
	ValidTo          time.Time  `json:"valid_to"`
	DaysUntilExpiry  int        `json:"days_until_expiry"`
	V2GCapable       bool       `json:"v2g_capable"`
}

// --- Helper methods ---

// parseCertificateChain parses a PEM-encoded certificate chain
func (s *ISO15118Service) parseCertificateChain(pemData []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}

		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate: %w", err)
			}
			certs = append(certs, cert)
		}

		pemData = rest
	}

	return certs, nil
}

// extractVehicleIdentity extracts vehicle identity from a certificate
func (s *ISO15118Service) extractVehicleIdentity(cert *x509.Certificate) *domain.ISO15118VehicleIdentity {
	identity := &domain.ISO15118VehicleIdentity{}

	// Extract EMAID from Subject CN or extension
	if cert.Subject.CommonName != "" {
		identity.EMAID = cert.Subject.CommonName
	}

	// Try to extract from Subject Organization
	if len(cert.Subject.Organization) > 0 {
		identity.ContractID = cert.Subject.Organization[0]
	}

	// Extract VIN from Subject SerialNumber if present
	if cert.Subject.SerialNumber != "" {
		identity.VehicleVIN = cert.Subject.SerialNumber
	}

	// Check for V2G capability in extensions or key usage
	// V2G certificates typically have specific key usages or extensions
	for _, ext := range cert.Extensions {
		// ISO 15118-20 OID for V2G: 1.0.15118.1
		if ext.Id.String() == "1.0.15118.1" || ext.Id.String() == "1.0.15118.2" {
			identity.V2GCapable = true
			break
		}
	}

	// Check key usage for digital signature and key agreement (common in V2G)
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 &&
		cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
		identity.V2GCapable = true
	}

	return identity
}

// verifyCertificateChain verifies the certificate chain
func (s *ISO15118Service) verifyCertificateChain(certs []*x509.Certificate) error {
	if len(certs) < 1 {
		return fmt.Errorf("empty certificate chain")
	}

	// For a proper implementation, you would:
	// 1. Load the V2G Root CA
	// 2. Build the intermediate chain
	// 3. Verify each certificate is signed by its parent
	// 4. Check that the chain ends at a trusted root

	// Simplified verification: just check that each cert is signed by the next
	for i := 0; i < len(certs)-1; i++ {
		child := certs[i]
		parent := certs[i+1]

		if err := child.CheckSignatureFrom(parent); err != nil {
			return fmt.Errorf("certificate %d not signed by certificate %d: %w", i, i+1, err)
		}
	}

	return nil
}

// checkOCSPStatus checks the OCSP status of a certificate
func (s *ISO15118Service) checkOCSPStatus(ctx context.Context, cert *x509.Certificate) error {
	// OCSP checking would require:
	// 1. Building an OCSP request
	// 2. Sending it to the OCSP responder
	// 3. Verifying the response signature
	// 4. Checking the certificate status

	// For now, this is a placeholder
	s.log.Debug("OCSP check skipped - not fully implemented",
		zap.String("subject", cert.Subject.String()),
	)

	return nil
}

// getCachedValidation gets a cached validation result
func (s *ISO15118Service) getCachedValidation(key string) *certValidation {
	s.certCache.mu.RLock()
	defer s.certCache.mu.RUnlock()

	cached, ok := s.certCache.validations[key]
	if !ok {
		return nil
	}

	// Check if cache has expired
	if time.Now().After(cached.expiresAt) {
		return nil
	}

	return cached
}

// cacheValidation caches a validation result
func (s *ISO15118Service) cacheValidation(key string, valid bool, reason string) {
	s.certCache.mu.Lock()
	defer s.certCache.mu.Unlock()

	s.certCache.validations[key] = &certValidation{
		valid:     valid,
		reason:    reason,
		validAt:   time.Now(),
		expiresAt: time.Now().Add(s.config.CertCacheDuration),
	}
}

// invalidateCacheForEMAID invalidates all cached validations for an EMAID
func (s *ISO15118Service) invalidateCacheForEMAID(emaid string) {
	// In a real implementation, you would track which cache keys
	// belong to which EMAID. For simplicity, we clear the whole cache.
	s.certCache.mu.Lock()
	defer s.certCache.mu.Unlock()

	s.certCache.validations = make(map[string]*certValidation)
}

// CleanupExpiredCertificates removes expired certificates from cache
func (s *ISO15118Service) CleanupExpiredCertificates() {
	s.certCache.mu.Lock()
	defer s.certCache.mu.Unlock()

	now := time.Now()
	for key, val := range s.certCache.validations {
		if now.After(val.expiresAt) {
			delete(s.certCache.validations, key)
		}
	}
}
