package v201

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// SecurityConfig holds OCPP security configuration
type SecurityConfig struct {
	// Enabled enables security checks
	Enabled bool

	// AllowedOrigins is a list of allowed WebSocket origins
	// Use "*" to allow all origins (not recommended for production)
	AllowedOrigins []string

	// AllowedChargePointIDs is a list of pre-registered charge point IDs
	// If empty, all charge point IDs are allowed (discovery mode)
	AllowedChargePointIDs []string

	// RequireSubprotocol requires the OCPP subprotocol header
	RequireSubprotocol bool

	// TLS configuration
	TLSEnabled        bool
	TLSCertFile       string
	TLSKeyFile        string
	TLSClientCA       string // For mTLS
	RequireClientCert bool   // Enable mTLS

	// Rate limiting
	MaxConnectionsPerIP  int
	MaxMessagesPerMinute int
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Enabled:              true,
		AllowedOrigins:       []string{}, // Empty means validate against allowed list
		AllowedChargePointIDs: []string{}, // Empty means allow all (discovery mode)
		RequireSubprotocol:   true,
		TLSEnabled:           false,
		RequireClientCert:    false,
		MaxConnectionsPerIP:  10,
		MaxMessagesPerMinute: 1000,
	}
}

// SecurityManager handles OCPP security
type SecurityManager struct {
	config              *SecurityConfig
	log                 *zap.Logger
	allowedOrigins      map[string]bool
	allowedChargePoints map[string]bool
	connectionCount     map[string]int
	mu                  sync.RWMutex
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig, log *zap.Logger) *SecurityManager {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	sm := &SecurityManager{
		config:              config,
		log:                 log,
		allowedOrigins:      make(map[string]bool),
		allowedChargePoints: make(map[string]bool),
		connectionCount:     make(map[string]int),
	}

	// Pre-populate allowed origins map for fast lookup
	for _, origin := range config.AllowedOrigins {
		sm.allowedOrigins[strings.ToLower(origin)] = true
	}

	// Pre-populate allowed charge points map
	for _, cpID := range config.AllowedChargePointIDs {
		sm.allowedChargePoints[cpID] = true
	}

	return sm
}

// CheckOrigin validates the WebSocket origin header
func (sm *SecurityManager) CheckOrigin(r *http.Request) bool {
	if !sm.config.Enabled {
		return true
	}

	origin := r.Header.Get("Origin")

	// If no origin header, check if it's a direct connection (no browser)
	if origin == "" {
		// Allow connections without Origin header (non-browser clients like charge points)
		sm.log.Debug("No Origin header, allowing direct connection",
			zap.String("remote_addr", r.RemoteAddr),
		)
		return true
	}

	// If no allowed origins configured, reject all browser connections
	if len(sm.config.AllowedOrigins) == 0 {
		sm.log.Warn("Origin rejected: no allowed origins configured",
			zap.String("origin", origin),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}

	// Check if origin is in allowed list
	originLower := strings.ToLower(origin)

	// Check for wildcard
	if sm.allowedOrigins["*"] {
		return true
	}

	// Extract host from origin (remove protocol)
	originHost := originLower
	if idx := strings.Index(originLower, "://"); idx != -1 {
		originHost = originLower[idx+3:]
	}

	// Check exact match
	if sm.allowedOrigins[originLower] || sm.allowedOrigins[originHost] {
		return true
	}

	// Check for subdomain wildcard (e.g., *.example.com)
	for allowed := range sm.allowedOrigins {
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:] // Remove *.
			if strings.HasSuffix(originHost, domain) {
				return true
			}
		}
	}

	sm.log.Warn("Origin rejected",
		zap.String("origin", origin),
		zap.String("remote_addr", r.RemoteAddr),
	)
	return false
}

// ValidateChargePoint validates if a charge point ID is allowed to connect
func (sm *SecurityManager) ValidateChargePoint(chargePointID string, r *http.Request) error {
	if !sm.config.Enabled {
		return nil
	}

	// If no allowed list, allow all (discovery mode)
	if len(sm.config.AllowedChargePointIDs) == 0 {
		sm.log.Debug("Allowing charge point in discovery mode",
			zap.String("charge_point_id", chargePointID),
		)
		return nil
	}

	// Check if charge point is in allowed list
	if !sm.allowedChargePoints[chargePointID] {
		sm.log.Warn("Charge point rejected: not in allowed list",
			zap.String("charge_point_id", chargePointID),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return fmt.Errorf("charge point not authorized: %s", chargePointID)
	}

	return nil
}

// ValidateSubprotocol validates the WebSocket subprotocol
func (sm *SecurityManager) ValidateSubprotocol(r *http.Request) bool {
	if !sm.config.Enabled || !sm.config.RequireSubprotocol {
		return true
	}

	protocols := r.Header.Get("Sec-WebSocket-Protocol")
	if protocols == "" {
		sm.log.Warn("Subprotocol validation failed: no protocol header",
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}

	// Check if ocpp2.0.1 is in the requested protocols
	for _, proto := range strings.Split(protocols, ",") {
		proto = strings.TrimSpace(proto)
		if proto == "ocpp2.0.1" || proto == "ocpp2.0" {
			return true
		}
	}

	sm.log.Warn("Subprotocol validation failed: OCPP protocol not found",
		zap.String("protocols", protocols),
		zap.String("remote_addr", r.RemoteAddr),
	)
	return false
}

// CheckRateLimit checks if the IP has exceeded connection limits
func (sm *SecurityManager) CheckRateLimit(r *http.Request) bool {
	if !sm.config.Enabled || sm.config.MaxConnectionsPerIP <= 0 {
		return true
	}

	ip := getClientIP(r)

	sm.mu.RLock()
	count := sm.connectionCount[ip]
	sm.mu.RUnlock()

	if count >= sm.config.MaxConnectionsPerIP {
		sm.log.Warn("Rate limit exceeded",
			zap.String("ip", ip),
			zap.Int("connections", count),
			zap.Int("limit", sm.config.MaxConnectionsPerIP),
		)
		return false
	}

	return true
}

// RegisterConnection increments the connection count for an IP
func (sm *SecurityManager) RegisterConnection(r *http.Request) {
	ip := getClientIP(r)

	sm.mu.Lock()
	sm.connectionCount[ip]++
	sm.mu.Unlock()
}

// UnregisterConnection decrements the connection count for an IP
func (sm *SecurityManager) UnregisterConnection(r *http.Request) {
	ip := getClientIP(r)

	sm.mu.Lock()
	if sm.connectionCount[ip] > 0 {
		sm.connectionCount[ip]--
	}
	sm.mu.Unlock()
}

// AddAllowedChargePoint dynamically adds a charge point to the allowed list
func (sm *SecurityManager) AddAllowedChargePoint(chargePointID string) {
	sm.mu.Lock()
	sm.allowedChargePoints[chargePointID] = true
	sm.mu.Unlock()

	sm.log.Info("Added allowed charge point", zap.String("charge_point_id", chargePointID))
}

// RemoveAllowedChargePoint removes a charge point from the allowed list
func (sm *SecurityManager) RemoveAllowedChargePoint(chargePointID string) {
	sm.mu.Lock()
	delete(sm.allowedChargePoints, chargePointID)
	sm.mu.Unlock()

	sm.log.Info("Removed allowed charge point", zap.String("charge_point_id", chargePointID))
}

// AddAllowedOrigin dynamically adds an origin to the allowed list
func (sm *SecurityManager) AddAllowedOrigin(origin string) {
	sm.mu.Lock()
	sm.allowedOrigins[strings.ToLower(origin)] = true
	sm.mu.Unlock()

	sm.log.Info("Added allowed origin", zap.String("origin", origin))
}

// GetTLSConfig returns a TLS configuration for the server
func (sm *SecurityManager) GetTLSConfig() (*tls.Config, error) {
	if !sm.config.TLSEnabled {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(sm.config.TLSCertFile, sm.config.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}

	// Configure mTLS if required
	if sm.config.RequireClientCert && sm.config.TLSClientCA != "" {
		caCert, err := os.ReadFile(sm.config.TLSClientCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read client CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse client CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for reverse proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
