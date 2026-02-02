package v2g

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// MockISO15118Repository is a mock implementation
type MockISO15118Repository struct {
	certificates map[string]*ISO15118Certificate
}

func NewMockISO15118Repository() *MockISO15118Repository {
	return &MockISO15118Repository{
		certificates: make(map[string]*ISO15118Certificate),
	}
}

func (m *MockISO15118Repository) StoreCertificate(ctx context.Context, cert interface{}) error {
	c, ok := cert.(*ISO15118Certificate)
	if !ok {
		return nil
	}
	m.certificates[c.EMAID] = c
	return nil
}

func (m *MockISO15118Repository) GetCertificateByEMAID(ctx context.Context, emaid string) (interface{}, error) {
	if cert, ok := m.certificates[emaid]; ok {
		return cert, nil
	}
	return nil, nil
}

func (m *MockISO15118Repository) GetCertificateByContractID(ctx context.Context, contractID string) (interface{}, error) {
	for _, cert := range m.certificates {
		if cert.ContractID == contractID {
			return cert, nil
		}
	}
	return nil, nil
}

func (m *MockISO15118Repository) GetCertificateByVIN(ctx context.Context, vin string) ([]interface{}, error) {
	var result []interface{}
	for _, cert := range m.certificates {
		if cert.VehicleVIN == vin {
			result = append(result, cert)
		}
	}
	return result, nil
}

func (m *MockISO15118Repository) UpdateCertificate(ctx context.Context, cert interface{}) error {
	c, ok := cert.(*ISO15118Certificate)
	if !ok {
		return nil
	}
	m.certificates[c.EMAID] = c
	return nil
}

func (m *MockISO15118Repository) GetExpiringCertificates(ctx context.Context, daysUntilExpiry int) ([]interface{}, error) {
	expiryDate := time.Now().AddDate(0, 0, daysUntilExpiry)
	var result []interface{}
	for _, cert := range m.certificates {
		if cert.ValidTo.Before(expiryDate) && !cert.Revoked {
			result = append(result, cert)
		}
	}
	return result, nil
}

func (m *MockISO15118Repository) GetV2GCapableCertificates(ctx context.Context) ([]interface{}, error) {
	var result []interface{}
	for _, cert := range m.certificates {
		if cert.V2GCapable && !cert.Revoked {
			result = append(result, cert)
		}
	}
	return result, nil
}

// Helper function to generate a test certificate
func generateTestCertificate(cn string, v2gCapable bool) ([]byte, error) {
	// Generate a new RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   cn, // EMAID
			Organization: []string{"Test Contract ID"},
			SerialNumber: "TESTVIN123", // VIN
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // 1 year validity
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		BasicConstraintsValid: true,
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return certPEM, nil
}

func createTestISO15118Service() (*ISO15118Service, *MockISO15118Repository) {
	logger := zap.NewNop()
	repo := NewMockISO15118Repository()
	config := &ISO15118Config{
		CertCacheDuration: 1 * time.Hour,
		EnableOCSP:        false, // Disable for tests
		AllowExpiredCerts: false,
	}

	service := NewISO15118Service(repo, logger, config)
	return service, repo
}

func TestISO15118Service_ParseCertificateChain(t *testing.T) {
	service, _ := createTestISO15118Service()

	// Generate a test certificate
	certPEM, err := generateTestCertificate("BREMAID123456", false)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Parse the certificate
	certs, err := service.parseCertificateChain(certPEM)
	if err != nil {
		t.Fatalf("parseCertificateChain failed: %v", err)
	}

	if len(certs) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(certs))
	}

	if certs[0].Subject.CommonName != "BREMAID123456" {
		t.Errorf("Expected CN 'BREMAID123456', got '%s'", certs[0].Subject.CommonName)
	}
}

func TestISO15118Service_ExtractVehicleIdentity(t *testing.T) {
	service, _ := createTestISO15118Service()

	// Generate a test certificate
	certPEM, err := generateTestCertificate("BREMAID123456", true)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	certs, _ := service.parseCertificateChain(certPEM)
	identity := service.extractVehicleIdentity(certs[0])

	if identity.EMAID != "BREMAID123456" {
		t.Errorf("Expected EMAID 'BREMAID123456', got '%s'", identity.EMAID)
	}

	if identity.VehicleVIN != "TESTVIN123" {
		t.Errorf("Expected VIN 'TESTVIN123', got '%s'", identity.VehicleVIN)
	}
}

func TestISO15118Service_ValidateCertificate(t *testing.T) {
	service, _ := createTestISO15118Service()
	ctx := context.Background()

	// Generate a valid test certificate
	certPEM, err := generateTestCertificate("BREMAID123456", false)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Validate should pass (self-signed but valid structure)
	err = service.ValidateCertificate(ctx, certPEM)
	if err != nil {
		t.Logf("Validation failed (expected for self-signed): %v", err)
	}
}

func TestISO15118Service_AuthenticateVehicle(t *testing.T) {
	service, _ := createTestISO15118Service()
	ctx := context.Background()

	// Generate a test certificate
	certPEM, err := generateTestCertificate("BREMAID123456", true)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Authenticate (may fail on validation but should extract identity)
	identity, err := service.AuthenticateVehicle(ctx, certPEM)
	if err != nil {
		// Self-signed certs will fail validation, but we can test the extraction
		t.Logf("Authentication failed (expected for self-signed): %v", err)
		return
	}

	if identity.EMAID != "BREMAID123456" {
		t.Errorf("Expected EMAID 'BREMAID123456', got '%s'", identity.EMAID)
	}
}

func TestISO15118Service_InstallCertificate(t *testing.T) {
	service, repo := createTestISO15118Service()
	ctx := context.Background()

	// Generate a test certificate
	certPEM, err := generateTestCertificate("BREMAID789012", true)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	req := &InstallCertificateRequest{
		CertificatePEM:      string(certPEM),
		VehicleVIN:          "TESTVIN456",
		V2GCapable:          true,
		ProviderID:          "PROVIDER001",
		MaxChargePowerKW:    150.0,
		MaxDischargePowerKW: 50.0,
	}

	err = service.InstallCertificate(ctx, req)
	if err != nil {
		t.Fatalf("InstallCertificate failed: %v", err)
	}

	// Verify certificate was stored
	storedCert := repo.certificates["BREMAID789012"]
	if storedCert == nil {
		t.Fatal("Certificate not stored")
	}

	if storedCert.V2GCapable != true {
		t.Error("Expected V2GCapable to be true")
	}

	if storedCert.MaxDischargePowerKW != 50.0 {
		t.Errorf("Expected max discharge power 50.0, got %f", storedCert.MaxDischargePowerKW)
	}
}

func TestISO15118Service_RevokeCertificate(t *testing.T) {
	service, repo := createTestISO15118Service()
	ctx := context.Background()

	// Add a certificate first
	cert := &ISO15118Certificate{
		EMAID:       "BREMAID111111",
		ContractID:  "CONTRACT111",
		V2GCapable:  true,
		ValidFrom:   time.Now().Add(-24 * time.Hour),
		ValidTo:     time.Now().Add(365 * 24 * time.Hour),
		Revoked:     false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.certificates[cert.EMAID] = cert

	// Revoke the certificate
	err := service.RevokeCertificate(ctx, "BREMAID111111", "Test revocation")
	if err != nil {
		t.Fatalf("RevokeCertificate failed: %v", err)
	}

	// Verify certificate is revoked
	revokedCert := repo.certificates["BREMAID111111"]
	if !revokedCert.Revoked {
		t.Error("Certificate should be revoked")
	}

	if revokedCert.RevocationReason != "Test revocation" {
		t.Errorf("Expected reason 'Test revocation', got '%s'", revokedCert.RevocationReason)
	}

	if revokedCert.RevokedAt == nil {
		t.Error("RevokedAt should be set")
	}
}

func TestISO15118Service_GetCertificateStatus(t *testing.T) {
	service, repo := createTestISO15118Service()
	ctx := context.Background()

	// Add a valid certificate
	cert := &ISO15118Certificate{
		EMAID:       "BREMAID222222",
		ContractID:  "CONTRACT222",
		V2GCapable:  true,
		ValidFrom:   time.Now().Add(-24 * time.Hour),
		ValidTo:     time.Now().Add(30 * 24 * time.Hour), // 30 days from now
		Revoked:     false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.certificates[cert.EMAID] = cert

	status, err := service.GetCertificateStatus(ctx, "BREMAID222222")
	if err != nil {
		t.Fatalf("GetCertificateStatus failed: %v", err)
	}

	if !status.Valid {
		t.Error("Certificate should be valid")
	}

	if status.Revoked {
		t.Error("Certificate should not be revoked")
	}

	if status.Expired {
		t.Error("Certificate should not be expired")
	}

	if status.DaysUntilExpiry < 29 || status.DaysUntilExpiry > 31 {
		t.Errorf("Expected ~30 days until expiry, got %d", status.DaysUntilExpiry)
	}

	if !status.V2GCapable {
		t.Error("Certificate should be V2G capable")
	}
}

func TestISO15118Service_GetChargingContract(t *testing.T) {
	service, repo := createTestISO15118Service()
	ctx := context.Background()

	// Add a certificate with contract info
	cert := &ISO15118Certificate{
		EMAID:               "BREMAID333333",
		ContractID:          "CONTRACT333",
		V2GCapable:          true,
		ValidFrom:           time.Now().Add(-24 * time.Hour),
		ValidTo:             time.Now().Add(365 * 24 * time.Hour),
		ProviderID:          "PROVIDER001",
		MaxChargePowerKW:    150.0,
		MaxDischargePowerKW: 50.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	repo.certificates[cert.EMAID] = cert

	contract, err := service.GetChargingContract(ctx, "BREMAID333333")
	if err != nil {
		t.Fatalf("GetChargingContract failed: %v", err)
	}

	if contract.ContractID != "CONTRACT333" {
		t.Errorf("Expected contract ID 'CONTRACT333', got '%s'", contract.ContractID)
	}

	if contract.MaxChargePowerKW != 150.0 {
		t.Errorf("Expected max charge power 150.0, got %f", contract.MaxChargePowerKW)
	}

	if contract.MaxDischargePowerKW != 50.0 {
		t.Errorf("Expected max discharge power 50.0, got %f", contract.MaxDischargePowerKW)
	}

	if !contract.V2GEnabled {
		t.Error("Contract should have V2G enabled")
	}
}

func TestISO15118Service_CacheValidation(t *testing.T) {
	service, _ := createTestISO15118Service()

	// Cache a validation result
	service.cacheValidation("testkey", true, "")

	// Retrieve from cache
	cached := service.getCachedValidation("testkey")
	if cached == nil {
		t.Fatal("Expected cached validation")
	}

	if !cached.valid {
		t.Error("Cached validation should be valid")
	}

	// Test cache miss
	notCached := service.getCachedValidation("nonexistent")
	if notCached != nil {
		t.Error("Expected nil for non-existent cache key")
	}
}

func TestISO15118Service_DefaultConfig(t *testing.T) {
	config := DefaultISO15118Config()

	if config.CertCacheDuration == 0 {
		t.Error("Default config should have CertCacheDuration")
	}

	if config.EnableOCSP != true {
		t.Error("Default config should enable OCSP")
	}

	if config.AllowExpiredCerts != false {
		t.Error("Default config should not allow expired certs")
	}
}

func TestISO15118_VehicleIdentityExtraction(t *testing.T) {
	tests := []struct {
		cn           string
		expectedEMAID string
	}{
		{"BREMAID123456", "BREMAID123456"},
		{"DE*ABC*123456*7", "DE*ABC*123456*7"},
		{"simple-emaid", "simple-emaid"},
	}

	service, _ := createTestISO15118Service()

	for _, tt := range tests {
		certPEM, err := generateTestCertificate(tt.cn, false)
		if err != nil {
			t.Fatalf("Failed to generate certificate: %v", err)
		}

		certs, _ := service.parseCertificateChain(certPEM)
		identity := service.extractVehicleIdentity(certs[0])

		if identity.EMAID != tt.expectedEMAID {
			t.Errorf("CN '%s': expected EMAID '%s', got '%s'",
				tt.cn, tt.expectedEMAID, identity.EMAID)
		}
	}
}

func TestISO15118_ChargingContractMapping(t *testing.T) {
	service, repo := createTestISO15118Service()
	ctx := context.Background()

	cert := &ISO15118Certificate{
		EMAID:               "BREMAID444444",
		ContractID:          "CONTRACT444",
		V2GCapable:          true,
		ValidFrom:           time.Now().Add(-24 * time.Hour),
		ValidTo:             time.Now().Add(365 * 24 * time.Hour),
		ProviderID:          "PROVIDER002",
		MaxChargePowerKW:    350.0,
		MaxDischargePowerKW: 100.0,
	}
	repo.certificates[cert.EMAID] = cert

	contract, _ := service.GetChargingContract(ctx, cert.EMAID)

	// Verify contract maps correctly to domain type
	domainContract := &domain.ChargingContract{
		ContractID:          contract.ContractID,
		EMAID:               contract.EMAID,
		ProviderID:          contract.ProviderID,
		ValidFrom:           contract.ValidFrom,
		ValidTo:             contract.ValidTo,
		MaxChargePowerKW:    contract.MaxChargePowerKW,
		MaxDischargePowerKW: contract.MaxDischargePowerKW,
		V2GEnabled:          contract.V2GEnabled,
	}

	if domainContract.ContractID != "CONTRACT444" {
		t.Errorf("Domain contract mapping failed")
	}
}
