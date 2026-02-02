package v201

import (
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
	deviceService ports.DeviceService
	txService     ports.TransactionService
	log           *zap.Logger
	clients       map[string]*websocket.Conn
	mu            sync.RWMutex
	upgrader      websocket.Upgrader
}

func NewServer(deviceService ports.DeviceService, txService ports.TransactionService, log *zap.Logger) *Server {
	return &Server{
		deviceService: deviceService,
		txService:     txService,
		log:           log,
		clients:       make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Security: Validate subprotocol/origin in prod
			},
			Subprotocols: []string{"ocpp2.0.1"},
		},
	}
}

func (s *Server) Start(port int) error {
	http.HandleFunc("/ocpp/", s.handleConnection) // /ocpp/{chargePointId}
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
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

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("Failed to upgrade websocket", zap.Error(err))
		return
	}

	s.registerClient(chargePointID, conn)
	defer s.unregisterClient(chargePointID)

	s.log.Info("New OCPP connection", zap.String("chargePointID", chargePointID))

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

func (s *Server) registerClient(id string, conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[id] = conn
}

func (s *Server) unregisterClient(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if conn, ok := s.clients[id]; ok {
		conn.Close()
		delete(s.clients, id)
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
