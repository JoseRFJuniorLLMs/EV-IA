package v201

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

// CommandResponse represents the response from a charge point to a CSMS command
type CommandResponse struct {
	Success bool
	Payload json.RawMessage
	Error   *OCPPError
}

// OCPPError represents an OCPP error response
type OCPPError struct {
	Code        string
	Description string
	Details     interface{}
}

// PendingRequest represents a request waiting for a response from a charge point
type PendingRequest struct {
	MessageID     string
	Action        string
	ChargePointID string
	Payload       interface{}
	ResponseChan  chan *CommandResponse
	Timeout       time.Time
	CreatedAt     time.Time
}

// Server configuration constants
const (
	DefaultCommandTimeout = 30 * time.Second
	RequestCleanupInterval = 60 * time.Second
)

type Server struct {
	deviceService   ports.DeviceService
	txService       ports.TransactionService
	log             *zap.Logger
	clients         map[string]*websocket.Conn
	clientRequests  map[string]*http.Request // Track request for unregister
	pendingRequests map[string]*PendingRequest // Track pending CSMS → CP requests
	mu              sync.RWMutex
	pendingMu       sync.RWMutex // Separate mutex for pending requests
	upgrader        websocket.Upgrader
	securityManager *SecurityManager
	stopCleanup     chan struct{}
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
		pendingRequests: make(map[string]*PendingRequest),
		securityManager: sm,
		stopCleanup:     make(chan struct{}),
	}

	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     sm.CheckOrigin,
		Subprotocols:    []string{"ocpp2.0.1", "ocpp2.0"},
	}

	// Start background cleanup of expired pending requests
	go s.cleanupExpiredRequests()

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
	// Stop the cleanup goroutine
	close(s.stopCleanup)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.clients {
		conn.Close()
	}

	// Cancel all pending requests
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	for _, req := range s.pendingRequests {
		if req.ResponseChan != nil {
			req.ResponseChan <- &CommandResponse{
				Success: false,
				Error: &OCPPError{
					Code:        "ServerShutdown",
					Description: "Server is shutting down",
				},
			}
			close(req.ResponseChan)
		}
	}
	s.pendingRequests = make(map[string]*PendingRequest)
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

	var msgID string
	if err := json.Unmarshal(raw[1], &msgID); err != nil {
		s.log.Error("Failed to parse message ID", zap.Error(err))
		return
	}

	switch msgType {
	case Call:
		var action string
		if err := json.Unmarshal(raw[2], &action); err != nil {
			s.log.Error("Failed to parse action", zap.Error(err))
			return
		}
		// Payload is raw[3]
		s.handleAction(chargePointID, msgID, action, []byte(raw[3]))

	case CallResult:
		// Handle response to our request (CSMS → CP)
		if len(raw) >= 3 {
			s.handleCallResult(chargePointID, msgID, raw[2])
		}

	case CallError:
		// Handle error response to our request
		if len(raw) >= 5 {
			var errorCode, errorDesc string
			json.Unmarshal(raw[2], &errorCode)
			json.Unmarshal(raw[3], &errorDesc)
			s.handleCallError(chargePointID, msgID, errorCode, errorDesc, raw[4])
		}
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

// --- Command Sending Methods (CSMS → Charge Point) ---

// SendCommand sends a command to a charge point and waits for response
func (s *Server) SendCommand(ctx context.Context, chargePointID, action string, payload interface{}) (*CommandResponse, error) {
	return s.SendCommandWithTimeout(ctx, chargePointID, action, payload, DefaultCommandTimeout)
}

// SendCommandWithTimeout sends a command with custom timeout
func (s *Server) SendCommandWithTimeout(ctx context.Context, chargePointID, action string, payload interface{}, timeout time.Duration) (*CommandResponse, error) {
	messageID := uuid.New().String()

	// Create pending request
	responseChan := make(chan *CommandResponse, 1)
	pendingReq := &PendingRequest{
		MessageID:     messageID,
		Action:        action,
		ChargePointID: chargePointID,
		Payload:       payload,
		ResponseChan:  responseChan,
		Timeout:       time.Now().Add(timeout),
		CreatedAt:     time.Now(),
	}

	// Register pending request
	s.pendingMu.Lock()
	s.pendingRequests[messageID] = pendingReq
	s.pendingMu.Unlock()

	// Send the call message
	callMsg := []interface{}{Call, messageID, action, payload}
	data, err := json.Marshal(callMsg)
	if err != nil {
		s.removePendingRequest(messageID)
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := s.Send(chargePointID, data); err != nil {
		s.removePendingRequest(messageID)
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	s.log.Info("Sent OCPP command",
		zap.String("action", action),
		zap.String("chargePointID", chargePointID),
		zap.String("messageID", messageID),
	)

	// Wait for response or timeout
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		s.removePendingRequest(messageID)
		return nil, errors.New("command timeout")
	case <-ctx.Done():
		s.removePendingRequest(messageID)
		return nil, ctx.Err()
	}
}

// SendCommandAsync sends a command without waiting for response
func (s *Server) SendCommandAsync(chargePointID, action string, payload interface{}) (string, error) {
	messageID := uuid.New().String()

	// Create pending request (without response channel for async)
	pendingReq := &PendingRequest{
		MessageID:     messageID,
		Action:        action,
		ChargePointID: chargePointID,
		Payload:       payload,
		Timeout:       time.Now().Add(DefaultCommandTimeout),
		CreatedAt:     time.Now(),
	}

	s.pendingMu.Lock()
	s.pendingRequests[messageID] = pendingReq
	s.pendingMu.Unlock()

	// Send the call message
	callMsg := []interface{}{Call, messageID, action, payload}
	data, err := json.Marshal(callMsg)
	if err != nil {
		s.removePendingRequest(messageID)
		return "", fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := s.Send(chargePointID, data); err != nil {
		s.removePendingRequest(messageID)
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	s.log.Info("Sent async OCPP command",
		zap.String("action", action),
		zap.String("chargePointID", chargePointID),
		zap.String("messageID", messageID),
	)

	return messageID, nil
}

// handleCallResult processes a CallResult response from charge point
func (s *Server) handleCallResult(chargePointID, messageID string, payload json.RawMessage) {
	s.pendingMu.Lock()
	pendingReq, ok := s.pendingRequests[messageID]
	if ok {
		delete(s.pendingRequests, messageID)
	}
	s.pendingMu.Unlock()

	if !ok {
		s.log.Warn("Received CallResult for unknown message",
			zap.String("messageID", messageID),
			zap.String("chargePointID", chargePointID),
		)
		return
	}

	s.log.Info("Received CallResult",
		zap.String("action", pendingReq.Action),
		zap.String("chargePointID", chargePointID),
		zap.String("messageID", messageID),
	)

	if pendingReq.ResponseChan != nil {
		pendingReq.ResponseChan <- &CommandResponse{
			Success: true,
			Payload: payload,
		}
		close(pendingReq.ResponseChan)
	}
}

// handleCallError processes a CallError response from charge point
func (s *Server) handleCallError(chargePointID, messageID, errorCode, errorDesc string, details json.RawMessage) {
	s.pendingMu.Lock()
	pendingReq, ok := s.pendingRequests[messageID]
	if ok {
		delete(s.pendingRequests, messageID)
	}
	s.pendingMu.Unlock()

	if !ok {
		s.log.Warn("Received CallError for unknown message",
			zap.String("messageID", messageID),
			zap.String("chargePointID", chargePointID),
		)
		return
	}

	s.log.Warn("Received CallError",
		zap.String("action", pendingReq.Action),
		zap.String("chargePointID", chargePointID),
		zap.String("messageID", messageID),
		zap.String("errorCode", errorCode),
		zap.String("errorDesc", errorDesc),
	)

	if pendingReq.ResponseChan != nil {
		pendingReq.ResponseChan <- &CommandResponse{
			Success: false,
			Error: &OCPPError{
				Code:        errorCode,
				Description: errorDesc,
				Details:     details,
			},
		}
		close(pendingReq.ResponseChan)
	}
}

// removePendingRequest removes a pending request by message ID
func (s *Server) removePendingRequest(messageID string) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if req, ok := s.pendingRequests[messageID]; ok {
		if req.ResponseChan != nil {
			close(req.ResponseChan)
		}
		delete(s.pendingRequests, messageID)
	}
}

// cleanupExpiredRequests periodically removes expired pending requests
func (s *Server) cleanupExpiredRequests() {
	ticker := time.NewTicker(RequestCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCleanup:
			return
		case <-ticker.C:
			s.pendingMu.Lock()
			now := time.Now()
			for msgID, req := range s.pendingRequests {
				if now.After(req.Timeout) {
					s.log.Warn("Cleaning up expired pending request",
						zap.String("messageID", msgID),
						zap.String("action", req.Action),
						zap.String("chargePointID", req.ChargePointID),
					)
					if req.ResponseChan != nil {
						req.ResponseChan <- &CommandResponse{
							Success: false,
							Error: &OCPPError{
								Code:        "Timeout",
								Description: "Request timed out",
							},
						}
						close(req.ResponseChan)
					}
					delete(s.pendingRequests, msgID)
				}
			}
			s.pendingMu.Unlock()
		}
	}
}

// GetPendingRequestCount returns the number of pending requests
func (s *Server) GetPendingRequestCount() int {
	s.pendingMu.RLock()
	defer s.pendingMu.RUnlock()
	return len(s.pendingRequests)
}
