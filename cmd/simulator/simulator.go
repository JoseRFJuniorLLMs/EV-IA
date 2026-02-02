package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SimulatorConfig holds the simulator configuration
type SimulatorConfig struct {
	ServerURL           string
	ChargePointID       string
	Vendor              string
	Model               string
	SerialNumber        string
	FirmwareVersion     string
	V2GCapable          bool
	BatterySOC          int     // Current SOC %
	BatteryCapacityKWh  float64 // Battery capacity
	MaxChargePowerKW    float64
	MaxDischargePowerKW float64
	ConnectorCount      int
}

// ConnectorState represents a connector's state
type ConnectorState struct {
	ID        int
	Status    string // Available, Occupied, Reserved, Unavailable, Faulted
	MeterWh   int
	IsCharging bool
}

// Simulator simulates an OCPP 2.0.1 charge point
type Simulator struct {
	config     *SimulatorConfig
	conn       *websocket.Conn
	log        *zap.Logger
	connectors []ConnectorState

	// State
	currentTxID    string
	isCharging     bool
	isDischarging  bool // V2G
	heartbeatInterval int

	// Message handling
	messageID   int
	pendingMsgs map[string]chan []byte
	mu          sync.RWMutex

	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewSimulator creates a new charge point simulator
func NewSimulator(config *SimulatorConfig, log *zap.Logger) *Simulator {
	connectors := make([]ConnectorState, config.ConnectorCount)
	for i := 0; i < config.ConnectorCount; i++ {
		connectors[i] = ConnectorState{
			ID:     i + 1,
			Status: "Available",
		}
	}

	return &Simulator{
		config:      config,
		log:         log,
		connectors:  connectors,
		pendingMsgs: make(map[string]chan []byte),
		stopChan:    make(chan struct{}),
		heartbeatInterval: 300,
	}
}

// Connect connects to the OCPP server
func (s *Simulator) Connect() error {
	url := fmt.Sprintf("%s/%s", s.config.ServerURL, s.config.ChargePointID)

	dialer := websocket.Dialer{
		Subprotocols: []string{"ocpp2.0.1", "ocpp2.0"},
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	s.conn = conn
	s.log.Info("Connected to OCPP server",
		zap.String("url", url),
		zap.String("chargePointID", s.config.ChargePointID),
	)

	// Start message reader
	s.wg.Add(1)
	go s.readMessages()

	// Send BootNotification
	resp, err := s.sendBootNotification()
	if err != nil {
		s.log.Error("BootNotification failed", zap.Error(err))
	} else {
		s.log.Info("BootNotification response", zap.Any("response", resp))
		if interval, ok := resp["interval"].(float64); ok {
			s.heartbeatInterval = int(interval)
		}
	}

	// Start heartbeat goroutine
	s.wg.Add(1)
	go s.heartbeatLoop()

	return nil
}

// Stop stops the simulator
func (s *Simulator) Stop() {
	close(s.stopChan)
	if s.conn != nil {
		s.conn.Close()
	}
	s.wg.Wait()
}

// readMessages reads and processes incoming messages
func (s *Simulator) readMessages() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				s.log.Error("Read error", zap.Error(err))
				return
			}
			s.handleMessage(message)
		}
	}
}

// handleMessage processes an incoming OCPP message
func (s *Simulator) handleMessage(data []byte) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		s.log.Error("Invalid message", zap.Error(err))
		return
	}

	if len(raw) < 3 {
		return
	}

	var msgType int
	json.Unmarshal(raw[0], &msgType)

	var msgID string
	json.Unmarshal(raw[1], &msgID)

	switch msgType {
	case 2: // Call - request from server
		var action string
		json.Unmarshal(raw[2], &action)
		s.handleServerRequest(msgID, action, raw[3])

	case 3: // CallResult - response to our request
		s.mu.Lock()
		if ch, ok := s.pendingMsgs[msgID]; ok {
			ch <- raw[2]
			delete(s.pendingMsgs, msgID)
		}
		s.mu.Unlock()

	case 4: // CallError
		s.mu.Lock()
		if ch, ok := s.pendingMsgs[msgID]; ok {
			close(ch)
			delete(s.pendingMsgs, msgID)
		}
		s.mu.Unlock()
	}
}

// handleServerRequest handles requests from the CSMS
func (s *Simulator) handleServerRequest(msgID, action string, payload json.RawMessage) {
	s.log.Info("Received server request", zap.String("action", action))

	var response interface{}

	switch action {
	case "RequestStartTransaction":
		response = s.handleRemoteStart(payload)
	case "RequestStopTransaction":
		response = s.handleRemoteStop(payload)
	case "Reset":
		response = s.handleReset(payload)
	case "SetChargingProfile":
		response = s.handleSetChargingProfile(payload)
	case "ClearChargingProfile":
		response = s.handleClearChargingProfile(payload)
	case "TriggerMessage":
		response = s.handleTriggerMessage(payload)
	case "UpdateFirmware":
		response = s.handleUpdateFirmware(payload)
	case "GetVariables":
		response = s.handleGetVariables(payload)
	case "SetVariables":
		response = s.handleSetVariables(payload)
	case "UnlockConnector":
		response = s.handleUnlockConnector(payload)
	case "ChangeAvailability":
		response = s.handleChangeAvailability(payload)
	default:
		s.sendCallError(msgID, "NotImplemented", fmt.Sprintf("Action %s not implemented", action))
		return
	}

	s.sendCallResult(msgID, response)
}

// --- Request Handlers ---

func (s *Simulator) handleRemoteStart(payload json.RawMessage) map[string]interface{} {
	var req struct {
		IdToken struct {
			IdToken string `json:"idToken"`
		} `json:"idToken"`
		EvseId *int `json:"evseId"`
	}
	json.Unmarshal(payload, &req)

	connectorID := 1
	if req.EvseId != nil {
		connectorID = *req.EvseId
	}

	// Simulate starting a transaction
	s.currentTxID = fmt.Sprintf("TX-%d", time.Now().Unix())
	s.isCharging = true

	// Update connector status
	if connectorID <= len(s.connectors) {
		s.connectors[connectorID-1].Status = "Occupied"
		s.connectors[connectorID-1].IsCharging = true
	}

	s.log.Info("Remote start accepted",
		zap.String("transactionID", s.currentTxID),
		zap.Int("connectorID", connectorID),
	)

	// Send TransactionEvent Started
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.sendTransactionEvent("Started", s.currentTxID, connectorID, req.IdToken.IdToken)
	}()

	return map[string]interface{}{
		"status":        "Accepted",
		"transactionId": s.currentTxID,
	}
}

func (s *Simulator) handleRemoteStop(payload json.RawMessage) map[string]interface{} {
	var req struct {
		TransactionId string `json:"transactionId"`
	}
	json.Unmarshal(payload, &req)

	if !s.isCharging {
		return map[string]interface{}{
			"status": "Rejected",
		}
	}

	s.log.Info("Remote stop accepted", zap.String("transactionID", req.TransactionId))

	// Send TransactionEvent Ended
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.sendTransactionEvent("Ended", req.TransactionId, 1, "")
		s.isCharging = false
		if len(s.connectors) > 0 {
			s.connectors[0].Status = "Available"
			s.connectors[0].IsCharging = false
		}
	}()

	return map[string]interface{}{
		"status": "Accepted",
	}
}

func (s *Simulator) handleReset(payload json.RawMessage) map[string]interface{} {
	var req struct {
		Type   string `json:"type"`
		EvseId *int   `json:"evseId"`
	}
	json.Unmarshal(payload, &req)

	s.log.Info("Reset requested", zap.String("type", req.Type))

	// Simulate reset
	go func() {
		if req.Type == "Immediate" {
			time.Sleep(500 * time.Millisecond)
		} else {
			// Wait for charging to complete
			time.Sleep(2 * time.Second)
		}

		// Reset state
		s.isCharging = false
		s.isDischarging = false
		for i := range s.connectors {
			s.connectors[i].Status = "Available"
			s.connectors[i].IsCharging = false
		}

		// Reconnect and send BootNotification
		s.sendBootNotification()
	}()

	return map[string]interface{}{
		"status": "Accepted",
	}
}

func (s *Simulator) handleSetChargingProfile(payload json.RawMessage) map[string]interface{} {
	var req struct {
		EvseId          int `json:"evseId"`
		ChargingProfile struct {
			ChargingSchedule []struct {
				ChargingSchedulePeriod []struct {
					Limit float64 `json:"limit"`
				} `json:"chargingSchedulePeriod"`
			} `json:"chargingSchedule"`
		} `json:"chargingProfile"`
	}
	json.Unmarshal(payload, &req)

	s.log.Info("Charging profile set", zap.Int("evseId", req.EvseId))

	// Check for V2G (negative limit)
	if len(req.ChargingProfile.ChargingSchedule) > 0 {
		schedule := req.ChargingProfile.ChargingSchedule[0]
		if len(schedule.ChargingSchedulePeriod) > 0 {
			limit := schedule.ChargingSchedulePeriod[0].Limit
			if limit < 0 {
				s.isDischarging = true
				s.log.Info("V2G discharge started", zap.Float64("powerW", -limit))
			}
		}
	}

	return map[string]interface{}{
		"status": "Accepted",
	}
}

func (s *Simulator) handleClearChargingProfile(payload json.RawMessage) map[string]interface{} {
	s.isDischarging = false
	s.log.Info("Charging profile cleared")

	return map[string]interface{}{
		"status": "Accepted",
	}
}

func (s *Simulator) handleTriggerMessage(payload json.RawMessage) map[string]interface{} {
	var req struct {
		RequestedMessage string `json:"requestedMessage"`
	}
	json.Unmarshal(payload, &req)

	s.log.Info("Trigger message", zap.String("message", req.RequestedMessage))

	// Send the requested message
	go func() {
		time.Sleep(100 * time.Millisecond)
		switch req.RequestedMessage {
		case "BootNotification":
			s.sendBootNotification()
		case "Heartbeat":
			s.sendHeartbeat()
		case "StatusNotification":
			for _, conn := range s.connectors {
				s.sendStatusNotification(conn.ID, conn.Status)
			}
		case "MeterValues":
			if s.isCharging && len(s.connectors) > 0 {
				s.sendMeterValues(1, s.connectors[0].MeterWh)
			}
		}
	}()

	return map[string]interface{}{
		"status": "Accepted",
	}
}

func (s *Simulator) handleUpdateFirmware(payload json.RawMessage) map[string]interface{} {
	var req struct {
		RequestId int `json:"requestId"`
		Firmware  struct {
			Location string `json:"location"`
		} `json:"firmware"`
	}
	json.Unmarshal(payload, &req)

	s.log.Info("Firmware update requested",
		zap.Int("requestId", req.RequestId),
		zap.String("location", req.Firmware.Location),
	)

	// Simulate firmware update process
	go func() {
		statuses := []string{"Downloading", "Downloaded", "Installing", "Installed"}
		for _, status := range statuses {
			time.Sleep(1 * time.Second)
			s.sendFirmwareStatus(status, req.RequestId)
		}
	}()

	return map[string]interface{}{
		"status": "Accepted",
	}
}

func (s *Simulator) handleGetVariables(payload json.RawMessage) map[string]interface{} {
	var req struct {
		GetVariableData []struct {
			Component struct {
				Name string `json:"name"`
			} `json:"component"`
			Variable struct {
				Name string `json:"name"`
			} `json:"variable"`
		} `json:"getVariableData"`
	}
	json.Unmarshal(payload, &req)

	results := make([]map[string]interface{}, len(req.GetVariableData))
	for i, v := range req.GetVariableData {
		value := s.getVariableValue(v.Component.Name, v.Variable.Name)
		results[i] = map[string]interface{}{
			"attributeStatus": "Accepted",
			"attributeValue":  value,
			"component":       v.Component,
			"variable":        v.Variable,
		}
	}

	return map[string]interface{}{
		"getVariableResult": results,
	}
}

func (s *Simulator) handleSetVariables(payload json.RawMessage) map[string]interface{} {
	var req struct {
		SetVariableData []struct {
			AttributeValue string `json:"attributeValue"`
			Component      struct {
				Name string `json:"name"`
			} `json:"component"`
			Variable struct {
				Name string `json:"name"`
			} `json:"variable"`
		} `json:"setVariableData"`
	}
	json.Unmarshal(payload, &req)

	results := make([]map[string]interface{}, len(req.SetVariableData))
	for i, v := range req.SetVariableData {
		s.log.Info("Set variable",
			zap.String("component", v.Component.Name),
			zap.String("variable", v.Variable.Name),
			zap.String("value", v.AttributeValue),
		)
		results[i] = map[string]interface{}{
			"attributeStatus": "Accepted",
			"component":       v.Component,
			"variable":        v.Variable,
		}
	}

	return map[string]interface{}{
		"setVariableResult": results,
	}
}

func (s *Simulator) handleUnlockConnector(payload json.RawMessage) map[string]interface{} {
	var req struct {
		EvseId      int `json:"evseId"`
		ConnectorId int `json:"connectorId"`
	}
	json.Unmarshal(payload, &req)

	s.log.Info("Unlock connector", zap.Int("evseId", req.EvseId), zap.Int("connectorId", req.ConnectorId))

	return map[string]interface{}{
		"status": "Unlocked",
	}
}

func (s *Simulator) handleChangeAvailability(payload json.RawMessage) map[string]interface{} {
	var req struct {
		OperationalStatus string `json:"operationalStatus"`
		Evse              *struct {
			Id int `json:"id"`
		} `json:"evse"`
	}
	json.Unmarshal(payload, &req)

	s.log.Info("Change availability", zap.String("status", req.OperationalStatus))

	status := "Available"
	if req.OperationalStatus == "Inoperative" {
		status = "Unavailable"
	}

	if req.Evse != nil {
		if req.Evse.Id <= len(s.connectors) {
			s.connectors[req.Evse.Id-1].Status = status
		}
	} else {
		for i := range s.connectors {
			s.connectors[i].Status = status
		}
	}

	return map[string]interface{}{
		"status": "Accepted",
	}
}

func (s *Simulator) getVariableValue(component, variable string) string {
	switch component {
	case "ChargingStation":
		switch variable {
		case "Model":
			return s.config.Model
		case "VendorName":
			return s.config.Vendor
		case "SerialNumber":
			return s.config.SerialNumber
		case "FirmwareVersion":
			return s.config.FirmwareVersion
		}
	case "EVSE":
		switch variable {
		case "AvailabilityState":
			if len(s.connectors) > 0 {
				return s.connectors[0].Status
			}
		}
	}
	return ""
}

// --- Outgoing Messages ---

func (s *Simulator) sendCall(action string, payload interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	s.messageID++
	msgID := fmt.Sprintf("%d", s.messageID)
	responseChan := make(chan []byte, 1)
	s.pendingMsgs[msgID] = responseChan
	s.mu.Unlock()

	msg := []interface{}{2, msgID, action, payload}
	data, _ := json.Marshal(msg)

	if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return nil, err
	}

	select {
	case respData := <-responseChan:
		var result map[string]interface{}
		json.Unmarshal(respData, &result)
		return result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (s *Simulator) sendCallResult(msgID string, payload interface{}) {
	msg := []interface{}{3, msgID, payload}
	data, _ := json.Marshal(msg)
	s.conn.WriteMessage(websocket.TextMessage, data)
}

func (s *Simulator) sendCallError(msgID, code, desc string) {
	msg := []interface{}{4, msgID, code, desc, nil}
	data, _ := json.Marshal(msg)
	s.conn.WriteMessage(websocket.TextMessage, data)
}

func (s *Simulator) sendBootNotification() (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"chargingStation": map[string]interface{}{
			"model":           s.config.Model,
			"vendorName":      s.config.Vendor,
			"serialNumber":    s.config.SerialNumber,
			"firmwareVersion": s.config.FirmwareVersion,
		},
		"reason": "PowerUp",
	}
	return s.sendCall("BootNotification", payload)
}

func (s *Simulator) sendHeartbeat() {
	s.sendCall("Heartbeat", map[string]interface{}{})
}

func (s *Simulator) sendStatusNotification(connectorID int, status string) {
	payload := map[string]interface{}{
		"timestamp":       time.Now().Format(time.RFC3339),
		"connectorStatus": status,
		"evseId":          connectorID,
		"connectorId":     connectorID,
	}
	s.sendCall("StatusNotification", payload)
}

func (s *Simulator) sendTransactionEvent(eventType, txID string, connectorID int, idToken string) {
	payload := map[string]interface{}{
		"eventType":     eventType,
		"timestamp":     time.Now().Format(time.RFC3339),
		"triggerReason": "RemoteStart",
		"seqNo":         1,
		"transactionInfo": map[string]interface{}{
			"transactionId": txID,
		},
		"evse": map[string]interface{}{
			"id":          connectorID,
			"connectorId": connectorID,
		},
	}

	if idToken != "" {
		payload["idToken"] = map[string]interface{}{
			"idToken": idToken,
			"type":    "ISO14443",
		}
	}

	s.sendCall("TransactionEvent", payload)
}

func (s *Simulator) sendMeterValues(evseID, valueWh int) {
	payload := map[string]interface{}{
		"evseId": evseID,
		"meterValue": []map[string]interface{}{
			{
				"timestamp": time.Now().Format(time.RFC3339),
				"sampledValue": []map[string]interface{}{
					{
						"value":     fmt.Sprintf("%d", valueWh),
						"measurand": "Energy.Active.Import.Register",
						"unit":      "Wh",
					},
				},
			},
		},
	}
	s.sendCall("MeterValues", payload)
}

func (s *Simulator) sendFirmwareStatus(status string, requestID int) {
	payload := map[string]interface{}{
		"status":    status,
		"requestId": requestID,
	}
	s.sendCall("FirmwareStatusNotification", payload)
}

func (s *Simulator) sendNotifyEVChargingNeeds(evseID int) {
	energyTransfer := "DC"
	if s.config.V2GCapable {
		energyTransfer = "DC_BPT" // Bidirectional Power Transfer
	}

	payload := map[string]interface{}{
		"evseId": evseID,
		"chargingNeeds": map[string]interface{}{
			"requestedEnergyTransfer": energyTransfer,
			"dcChargingParameters": map[string]interface{}{
				"evMaxCurrent":     400,
				"evMaxVoltage":     500,
				"stateOfCharge":    s.config.BatterySOC,
				"evEnergyCapacity": int(s.config.BatteryCapacityKWh),
			},
		},
	}

	if s.config.V2GCapable {
		payload["chargingNeeds"].(map[string]interface{})["dcChargingParameters"].(map[string]interface{})["evMaxDischargePower"] = int(s.config.MaxDischargePowerKW * 1000)
		payload["chargingNeeds"].(map[string]interface{})["dcChargingParameters"].(map[string]interface{})["evMaxDischargeCurrent"] = int(s.config.MaxDischargePowerKW * 1000 / 400)
	}

	s.sendCall("NotifyEVChargingNeeds", payload)
}

func (s *Simulator) heartbeatLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Duration(s.heartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.sendHeartbeat()
		}
	}
}

// RunInteractive runs the simulator in interactive mode
func (s *Simulator) RunInteractive() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(line)

		if len(parts) == 0 {
			fmt.Print("> ")
			continue
		}

		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "start":
			connID := 1
			if len(args) > 0 {
				connID, _ = strconv.Atoi(args[0])
			}
			s.currentTxID = fmt.Sprintf("TX-%d", time.Now().Unix())
			s.isCharging = true
			s.sendTransactionEvent("Started", s.currentTxID, connID, "USER123")
			fmt.Printf("Started charging on connector %d, TX: %s\n", connID, s.currentTxID)

		case "stop":
			if s.isCharging {
				s.sendTransactionEvent("Ended", s.currentTxID, 1, "USER123")
				s.isCharging = false
				fmt.Println("Stopped charging")
			} else {
				fmt.Println("Not currently charging")
			}

		case "status":
			if len(args) < 1 {
				fmt.Println("Usage: status <connector> [status]")
			} else {
				connID, _ := strconv.Atoi(args[0])
				status := "Available"
				if len(args) > 1 {
					status = args[1]
				}
				s.sendStatusNotification(connID, status)
				fmt.Printf("Sent status %s for connector %d\n", status, connID)
			}

		case "meter":
			if len(args) < 1 {
				fmt.Println("Usage: meter <valueWh>")
			} else {
				value, _ := strconv.Atoi(args[0])
				s.sendMeterValues(1, value)
				fmt.Printf("Sent meter value: %d Wh\n", value)
			}

		case "heartbeat":
			s.sendHeartbeat()
			fmt.Println("Sent heartbeat")

		case "v2g":
			if len(args) < 1 {
				fmt.Println("Usage: v2g start|stop|soc <value>")
			} else {
				switch args[0] {
				case "start":
					if s.config.V2GCapable {
						s.sendNotifyEVChargingNeeds(1)
						fmt.Println("Sent V2G charging needs (bidirectional)")
					} else {
						fmt.Println("V2G not enabled (use --v2g flag)")
					}
				case "stop":
					s.isDischarging = false
					fmt.Println("V2G discharge stopped")
				case "soc":
					if len(args) > 1 {
						soc, _ := strconv.Atoi(args[1])
						s.config.BatterySOC = soc
						fmt.Printf("Battery SOC set to %d%%\n", soc)
					}
				}
			}

		case "fault":
			connID := 1
			if len(args) > 0 {
				connID, _ = strconv.Atoi(args[0])
			}
			s.sendStatusNotification(connID, "Faulted")
			fmt.Printf("Sent fault status for connector %d\n", connID)

		case "reset":
			fmt.Println("Simulating reset...")
			s.sendBootNotification()
			fmt.Println("Reset complete")

		case "firmware":
			if len(args) < 1 {
				fmt.Println("Usage: firmware accept|reject")
			} else {
				if args[0] == "accept" {
					s.sendFirmwareStatus("Installed", 1)
					fmt.Println("Firmware update accepted")
				} else {
					s.sendFirmwareStatus("InstallationFailed", 1)
					fmt.Println("Firmware update rejected")
				}
			}

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}

		fmt.Print("> ")
	}
}
