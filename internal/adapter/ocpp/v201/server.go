package v201

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

type Server struct {
	deviceService   ports.DeviceService
	txService       ports.TransactionService
	log             *zap.Logger
	clients         map[string]*websocket.Conn
	clientRequests  map[string]*http.Request // Track request for unregister
	mu              sync.RWMutex
	upgrader        websocket.Upgrader
	securityManager *SecurityManager
}

// NewServer creates a new OCPP 2.0.1 server with default security (disabled)
func NewServer(deviceService ports.DeviceService, txService ports.TransactionService, log *zap.Logger) *Server {
	return NewServerWithSecurity(deviceService, txService, log, nil)
}

// NewServerWithSecurity creates a new OCPP 2.0.1 server with security configuration
func NewServerWithSecurity(deviceService ports.DeviceService, txService ports.TransactionService, log *zap.Logger, securityConfig *SecurityConfig) *Server {
	sm := NewSecurityManager(securityConfig, log)

	s := &Server{
		deviceService:   deviceService,
		txService:       txService,
		log:             log,
		clients:         make(map[string]*websocket.Conn),
		clientRequests:  make(map[string]*http.Request),
		securityManager: sm,
	}

	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     sm.CheckOrigin,
		Subprotocols:    []string{"ocpp2.0.1", "ocpp2.0"},
	}

	return s
}

func (s *Server) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ocpp/", s.handleConnection) // /ocpp/{chargePointId}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Configure TLS if enabled
	tlsConfig, err := s.securityManager.GetTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to configure TLS: %w", err)
	}

	if tlsConfig != nil {
		server.TLSConfig = tlsConfig
		s.log.Info("Starting OCPP server with TLS", zap.Int("port", port))
		return server.ListenAndServeTLS("", "") // Certs are in TLSConfig
	}

	s.log.Info("Starting OCPP server", zap.Int("port", port))
	return server.ListenAndServe()
}

// StartTLS starts the server with TLS using provided cert and key files
func (s *Server) StartTLS(port int, certFile, keyFile string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ocpp/", s.handleConnection)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	s.log.Info("Starting OCPP server with TLS", zap.Int("port", port))
	return server.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.clients {
		conn.Close()
	}
}

func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	chargePointID := r.URL.Path[len("/ocpp/"):]
	if chargePointID == "" {
		http.Error(w, "ChargePointID required", http.StatusBadRequest)
		return
	}

	// Security: Validate subprotocol
	if !s.securityManager.ValidateSubprotocol(r) {
		s.log.Warn("Subprotocol validation failed",
			zap.String("chargePointID", chargePointID),
			zap.String("remote_addr", r.RemoteAddr),
		)
		http.Error(w, "Invalid subprotocol", http.StatusBadRequest)
		return
	}

	// Security: Validate charge point ID
	if err := s.securityManager.ValidateChargePoint(chargePointID, r); err != nil {
		s.log.Warn("Charge point validation failed",
			zap.String("chargePointID", chargePointID),
			zap.Error(err),
		)
		http.Error(w, "Unauthorized charge point", http.StatusUnauthorized)
		return
	}

	// Security: Check rate limit
	if !s.securityManager.CheckRateLimit(r) {
		s.log.Warn("Rate limit exceeded",
			zap.String("chargePointID", chargePointID),
			zap.String("remote_addr", r.RemoteAddr),
		)
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}

	// Upgrade to WebSocket (CheckOrigin is handled by upgrader via SecurityManager)
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("Failed to upgrade websocket", zap.Error(err))
		return
	}

	// Register connection for rate limiting
	s.securityManager.RegisterConnection(r)

	s.registerClient(chargePointID, conn, r)
	defer s.unregisterClient(chargePointID)

	s.log.Info("New OCPP connection",
		zap.String("chargePointID", chargePointID),
		zap.String("remote_addr", r.RemoteAddr),
	)

	for {
		// Read message (Call, CallResult, CallError)
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.log.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		s.handleMessage(chargePointID, message)
	}
}

func (s *Server) registerClient(id string, conn *websocket.Conn, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[id] = conn
	s.clientRequests[id] = r
}

func (s *Server) unregisterClient(id string) {
	s.mu.Lock()
	r := s.clientRequests[id]
	if conn, ok := s.clients[id]; ok {
		conn.Close()
		delete(s.clients, id)
		delete(s.clientRequests, id)
	}
	s.mu.Unlock()

	// Unregister from rate limiter
	if r != nil {
		s.securityManager.UnregisterConnection(r)
	}
}

func (s *Server) handleMessage(chargePointID string, data []byte) {
	// OCPP messages are JSON arrays: [MessageTypeId, MessageId, Action, Payload]
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		s.log.Error("Invalid JSON message", zap.String("data", string(data)))
		return
	}

	if len(raw) < 3 {
		s.log.Error("Invalid OCPP message length")
		return
	}

	var msgType MessageType
	if err := json.Unmarshal(raw[0], &msgType); err != nil {
		return
	}

	if msgType == Call {
		var msgID string
		var action string
		if err := json.Unmarshal(raw[1], &msgID); err != nil {
			return
		}
		if err := json.Unmarshal(raw[2], &action); err != nil {
			return
		}

		// Payload is raw[3]
		s.handleAction(chargePointID, msgID, action, []byte(raw[3]))
	} else {
		// Handle CallResult or CallError if we acted as a Client (CSMS -> CP requests)
		// For now, only handling Server mode (CP -> CSMS)
	}
}

// Send sends an OCPP message to a charge point
func (s *Server) Send(chargePointID string, data []byte) error {
	s.mu.RLock()
	conn, ok := s.clients[chargePointID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("charge point not connected")
	}

	s.mu.Lock() // Write concurrency
	defer s.mu.Unlock()
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, data)
}

// GetSecurityManager returns the security manager for external configuration
func (s *Server) GetSecurityManager() *SecurityManager {
	return s.securityManager
}

// AddAllowedChargePoint adds a charge point to the allowed list
func (s *Server) AddAllowedChargePoint(chargePointID string) {
	s.securityManager.AddAllowedChargePoint(chargePointID)
}

// RemoveAllowedChargePoint removes a charge point from the allowed list
func (s *Server) RemoveAllowedChargePoint(chargePointID string) {
	s.securityManager.RemoveAllowedChargePoint(chargePointID)
}

// AddAllowedOrigin adds an origin to the allowed list
func (s *Server) AddAllowedOrigin(origin string) {
	s.securityManager.AddAllowedOrigin(origin)
}

// GetConnectedClients returns a list of connected charge point IDs
func (s *Server) GetConnectedClients() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]string, 0, len(s.clients))
	for id := range s.clients {
		clients = append(clients, id)
	}
	return clients
}

// IsConnected checks if a charge point is currently connected
func (s *Server) IsConnected(chargePointID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.clients[chargePointID]
	return ok
}
