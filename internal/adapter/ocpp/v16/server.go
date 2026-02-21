package v16

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// OCPP 1.6 message types
const (
	CallMessage       = 2
	CallResultMessage = 3
	CallErrorMessage  = 4
)

// Server handles OCPP 1.6 WebSocket connections (legacy support)
type Server struct {
	deviceService ports.DeviceService
	txService     ports.TransactionService
	clients       map[string]*websocket.Conn
	mu            sync.RWMutex
	handlers      *Handlers
	log           *zap.Logger
}

// NewServer creates a new OCPP 1.6 WebSocket server
func NewServer(deviceService ports.DeviceService, txService ports.TransactionService, log *zap.Logger) *Server {
	s := &Server{
		deviceService: deviceService,
		txService:     txService,
		clients:       make(map[string]*websocket.Conn),
		log:           log,
	}
	s.handlers = NewHandlers(deviceService, txService, log)
	return s
}

// Start starts the OCPP 1.6 WebSocket server on the given port
func (s *Server) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ocpp/1.6/", s.handleWebSocket)

	addr := fmt.Sprintf(":%d", port)
	s.log.Info("Starting OCPP 1.6 Legacy WebSocket Server", zap.String("addr", addr))
	return http.ListenAndServe(addr, mux)
}

// Stop gracefully closes all client connections
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, conn := range s.clients {
		conn.Close()
		delete(s.clients, id)
	}
	s.log.Info("OCPP 1.6 server stopped")
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract charge point ID from URL path: /ocpp/1.6/{chargePointID}
	chargePointID := r.URL.Path[len("/ocpp/1.6/"):]
	if chargePointID == "" {
		http.Error(w, "missing charge point ID", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	s.mu.Lock()
	s.clients[chargePointID] = conn
	s.mu.Unlock()

	s.log.Info("OCPP 1.6 charge point connected",
		zap.String("charge_point_id", chargePointID),
	)

	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.clients, chargePointID)
		s.mu.Unlock()
		s.log.Info("OCPP 1.6 charge point disconnected",
			zap.String("charge_point_id", chargePointID),
		)
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				s.log.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		response, err := s.processMessage(chargePointID, message)
		if err != nil {
			s.log.Error("Failed to process OCPP 1.6 message",
				zap.String("charge_point_id", chargePointID),
				zap.Error(err),
			)
			continue
		}

		if response != nil {
			if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
				s.log.Error("Failed to send response", zap.Error(err))
				break
			}
		}
	}
}

// processMessage parses and routes OCPP 1.6 JSON messages
// Format: [MessageTypeId, UniqueId, Action, Payload] for Call
// Format: [MessageTypeId, UniqueId, Payload] for CallResult
func (s *Server) processMessage(chargePointID string, raw []byte) ([]byte, error) {
	var msg []json.RawMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("invalid OCPP message format: %w", err)
	}

	if len(msg) < 3 {
		return nil, fmt.Errorf("OCPP message too short")
	}

	var msgType int
	if err := json.Unmarshal(msg[0], &msgType); err != nil {
		return nil, fmt.Errorf("invalid message type: %w", err)
	}

	var uniqueID string
	if err := json.Unmarshal(msg[1], &uniqueID); err != nil {
		return nil, fmt.Errorf("invalid unique ID: %w", err)
	}

	if msgType != CallMessage || len(msg) < 4 {
		return nil, nil // Only handle Call messages from charge points
	}

	var action string
	if err := json.Unmarshal(msg[2], &action); err != nil {
		return nil, fmt.Errorf("invalid action: %w", err)
	}

	s.log.Debug("Received OCPP 1.6 message",
		zap.String("charge_point_id", chargePointID),
		zap.String("action", action),
		zap.String("unique_id", uniqueID),
	)

	responsePayload, err := s.handlers.HandleMessage(chargePointID, action, msg[3])
	if err != nil {
		// Return CallError
		errorResp := []interface{}{CallErrorMessage, uniqueID, "InternalError", err.Error(), map[string]string{}}
		return json.Marshal(errorResp)
	}

	// Return CallResult
	result := []interface{}{CallResultMessage, uniqueID, responsePayload}
	return json.Marshal(result)
}
