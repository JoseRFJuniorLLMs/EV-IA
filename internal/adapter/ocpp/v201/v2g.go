package v201

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// V2GManager handles V2G-specific OCPP operations
type V2GManager struct {
	server          *Server
	log             *zap.Logger
	evCapabilities  map[string]*EVCapability // key: chargePointID:connectorID
	activeSessions  map[string]*V2GSessionInfo // key: chargePointID:connectorID
	mu              sync.RWMutex
}

// EVCapability represents an EV's V2G capability detected via OCPP
type EVCapability struct {
	ChargePointID       string
	ConnectorID         int
	EvseID              int
	RequestedTransfer   string    // AC_BPT, DC_BPT = V2G capable
	MaxDischargePowerW  int
	MaxDischargeCurrent int
	StateOfCharge       int       // Current SOC %
	BatteryCapacityKWh  int
	DepartureTime       *time.Time
	DetectedAt          time.Time
}

// V2GSessionInfo tracks an active V2G session
type V2GSessionInfo struct {
	SessionID       string
	ChargePointID   string
	ConnectorID     int
	EvseID          int
	Direction       domain.V2GDirection
	TargetPowerKW   float64
	CurrentPowerKW  float64
	EnergyKWh       float64
	StartedAt       time.Time
	LastUpdate      time.Time
}

// NewV2GManager creates a new V2G manager
func NewV2GManager(server *Server, log *zap.Logger) *V2GManager {
	return &V2GManager{
		server:         server,
		log:            log,
		evCapabilities: make(map[string]*EVCapability),
		activeSessions: make(map[string]*V2GSessionInfo),
	}
}

// ProcessChargingNeeds processes NotifyEVChargingNeeds to detect V2G capability
func (m *V2GManager) ProcessChargingNeeds(chargePointID string, req *NotifyEVChargingNeedsRequest) *EVCapability {
	key := fmt.Sprintf("%s:%d", chargePointID, req.EvseId)

	cap := &EVCapability{
		ChargePointID:     chargePointID,
		ConnectorID:       1, // Default, can be derived from EVSE
		EvseID:            req.EvseId,
		RequestedTransfer: req.ChargingNeeds.RequestedEnergyTransfer,
		DetectedAt:        time.Now(),
	}

	// Check for bidirectional power transfer (V2G)
	if req.ChargingNeeds.RequestedEnergyTransfer == "AC_BPT" ||
		req.ChargingNeeds.RequestedEnergyTransfer == "DC_BPT" {

		if req.ChargingNeeds.DCChargingParameters != nil {
			dc := req.ChargingNeeds.DCChargingParameters
			cap.StateOfCharge = dc.StateOfCharge
			if dc.EVEnergyCapacity != nil {
				cap.BatteryCapacityKWh = *dc.EVEnergyCapacity
			}
			if dc.EVMaxDischargePower != nil {
				cap.MaxDischargePowerW = *dc.EVMaxDischargePower
			}
			if dc.EVMaxDischargeCurrent != nil {
				cap.MaxDischargeCurrent = *dc.EVMaxDischargeCurrent
			}
		}

		if req.ChargingNeeds.DepartureTime != nil {
			if t, err := time.Parse(time.RFC3339, *req.ChargingNeeds.DepartureTime); err == nil {
				cap.DepartureTime = &t
			}
		}
	}

	m.mu.Lock()
	m.evCapabilities[key] = cap
	m.mu.Unlock()

	m.log.Info("V2G capability detected",
		zap.String("chargePointID", chargePointID),
		zap.Int("evseId", req.EvseId),
		zap.String("transferType", req.ChargingNeeds.RequestedEnergyTransfer),
		zap.Int("maxDischargePowerW", cap.MaxDischargePowerW),
		zap.Int("stateOfCharge", cap.StateOfCharge),
	)

	return cap
}

// GetEVCapability returns the V2G capability for a charge point/connector
func (m *V2GManager) GetEVCapability(chargePointID string, evseID int) *EVCapability {
	key := fmt.Sprintf("%s:%d", chargePointID, evseID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.evCapabilities[key]
}

// IsV2GCapable checks if an EV at the specified location supports V2G
func (m *V2GManager) IsV2GCapable(chargePointID string, evseID int) bool {
	cap := m.GetEVCapability(chargePointID, evseID)
	if cap == nil {
		return false
	}
	return cap.RequestedTransfer == "AC_BPT" || cap.RequestedTransfer == "DC_BPT"
}

// StartV2GDischarge initiates a V2G discharge session
func (m *V2GManager) StartV2GDischarge(ctx context.Context, chargePointID string, evseID int, dischargePowerKW float64, durationSeconds int, minSOC int) (*V2GSessionInfo, error) {
	// Verify V2G capability
	cap := m.GetEVCapability(chargePointID, evseID)
	if cap == nil {
		return nil, fmt.Errorf("no V2G capability detected for %s EVSE %d", chargePointID, evseID)
	}

	if cap.RequestedTransfer != "AC_BPT" && cap.RequestedTransfer != "DC_BPT" {
		return nil, fmt.Errorf("EV does not support bidirectional power transfer")
	}

	// Check SOC constraint
	if cap.StateOfCharge <= minSOC {
		return nil, fmt.Errorf("current SOC (%d%%) is at or below minimum (%d%%)", cap.StateOfCharge, minSOC)
	}

	// Limit discharge power to EV capability
	maxDischargeKW := float64(cap.MaxDischargePowerW) / 1000.0
	if dischargePowerKW > maxDischargeKW {
		dischargePowerKW = maxDischargeKW
		m.log.Warn("Discharge power limited to EV capability",
			zap.Float64("requestedKW", dischargePowerKW),
			zap.Float64("maxKW", maxDischargeKW),
		)
	}

	// Set charging profile with negative limit for discharge
	resp, err := m.server.SetV2GChargingProfile(ctx, chargePointID, evseID, dischargePowerKW, durationSeconds, minSOC)
	if err != nil {
		return nil, fmt.Errorf("failed to set V2G charging profile: %w", err)
	}

	if resp.Status != "Accepted" {
		return nil, fmt.Errorf("V2G charging profile rejected: %s", resp.Status)
	}

	// Create session tracking
	key := fmt.Sprintf("%s:%d", chargePointID, evseID)
	session := &V2GSessionInfo{
		SessionID:      fmt.Sprintf("v2g-%s-%d", chargePointID, time.Now().Unix()),
		ChargePointID:  chargePointID,
		ConnectorID:    cap.ConnectorID,
		EvseID:         evseID,
		Direction:      domain.V2GDirectionDischarging,
		TargetPowerKW:  dischargePowerKW,
		CurrentPowerKW: 0,
		EnergyKWh:      0,
		StartedAt:      time.Now(),
		LastUpdate:     time.Now(),
	}

	m.mu.Lock()
	m.activeSessions[key] = session
	m.mu.Unlock()

	m.log.Info("V2G discharge session started",
		zap.String("sessionID", session.SessionID),
		zap.String("chargePointID", chargePointID),
		zap.Int("evseID", evseID),
		zap.Float64("targetPowerKW", dischargePowerKW),
	)

	return session, nil
}

// StopV2GDischarge stops an active V2G discharge session
func (m *V2GManager) StopV2GDischarge(ctx context.Context, chargePointID string, evseID int) error {
	key := fmt.Sprintf("%s:%d", chargePointID, evseID)

	m.mu.Lock()
	session, ok := m.activeSessions[key]
	if ok {
		delete(m.activeSessions, key)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("no active V2G session for %s EVSE %d", chargePointID, evseID)
	}

	// Clear the V2G charging profile
	_, err := m.server.CancelV2GDischarge(ctx, chargePointID, evseID)
	if err != nil {
		m.log.Error("Failed to clear V2G profile", zap.Error(err))
	}

	m.log.Info("V2G discharge session stopped",
		zap.String("sessionID", session.SessionID),
		zap.String("chargePointID", chargePointID),
		zap.Float64("totalEnergyKWh", session.EnergyKWh),
	)

	return nil
}

// GetActiveSession returns the active V2G session for a charge point/EVSE
func (m *V2GManager) GetActiveSession(chargePointID string, evseID int) *V2GSessionInfo {
	key := fmt.Sprintf("%s:%d", chargePointID, evseID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeSessions[key]
}

// UpdateSessionMetrics updates metrics for an active V2G session
func (m *V2GManager) UpdateSessionMetrics(chargePointID string, evseID int, powerKW, energyKWh float64) {
	key := fmt.Sprintf("%s:%d", chargePointID, evseID)
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.activeSessions[key]; ok {
		session.CurrentPowerKW = powerKW
		session.EnergyKWh = energyKWh
		session.LastUpdate = time.Now()
	}
}

// GetAllActiveSessions returns all active V2G sessions
func (m *V2GManager) GetAllActiveSessions() []*V2GSessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*V2GSessionInfo, 0, len(m.activeSessions))
	for _, session := range m.activeSessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetAllCapabilities returns all detected V2G capabilities
func (m *V2GManager) GetAllCapabilities() []*EVCapability {
	m.mu.RLock()
	defer m.mu.RUnlock()

	caps := make([]*EVCapability, 0, len(m.evCapabilities))
	for _, cap := range m.evCapabilities {
		caps = append(caps, cap)
	}
	return caps
}

// ClearCapability removes a stored EV capability (e.g., when EV disconnects)
func (m *V2GManager) ClearCapability(chargePointID string, evseID int) {
	key := fmt.Sprintf("%s:%d", chargePointID, evseID)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.evCapabilities, key)
}

// --- V2G Optimization Functions ---

// OptimalDischargeWindow calculates the optimal time window for V2G discharge
// based on grid prices and EV constraints
type DischargeWindow struct {
	StartTime       time.Time
	EndTime         time.Time
	AveragePrice    float64
	TotalEnergyKWh  float64
	EstimatedRevenue float64
}

// CalculateOptimalDischarge calculates optimal discharge parameters
func (m *V2GManager) CalculateOptimalDischarge(
	chargePointID string,
	evseID int,
	pricePoints []domain.GridPricePoint,
	minSOC int,
	maxDischargeKWh float64,
) (*DischargeWindow, error) {

	cap := m.GetEVCapability(chargePointID, evseID)
	if cap == nil {
		return nil, fmt.Errorf("no V2G capability detected")
	}

	if len(pricePoints) == 0 {
		return nil, fmt.Errorf("no price data available")
	}

	// Calculate available energy for discharge
	availableSOC := cap.StateOfCharge - minSOC
	if availableSOC <= 0 {
		return nil, fmt.Errorf("insufficient SOC for discharge")
	}

	availableEnergyKWh := float64(availableSOC) / 100.0 * float64(cap.BatteryCapacityKWh)
	if availableEnergyKWh > maxDischargeKWh {
		availableEnergyKWh = maxDischargeKWh
	}

	// Find highest price periods
	// Sort by price (descending) and pick top periods
	var bestWindow *DischargeWindow
	var bestRevenue float64

	for i, pp := range pricePoints {
		if !pp.IsPeak {
			continue // Only consider peak hours for V2G
		}

		// Calculate potential revenue for this window
		dischargePowerKW := float64(cap.MaxDischargePowerW) / 1000.0
		durationHours := 1.0 // Assume 1-hour windows

		energyKWh := dischargePowerKW * durationHours
		if energyKWh > availableEnergyKWh {
			energyKWh = availableEnergyKWh
		}

		revenue := energyKWh * pp.Price * 0.9 // 10% operator margin

		if revenue > bestRevenue {
			bestRevenue = revenue
			endTime := pp.Timestamp.Add(time.Hour)
			bestWindow = &DischargeWindow{
				StartTime:       pp.Timestamp,
				EndTime:         endTime,
				AveragePrice:    pp.Price,
				TotalEnergyKWh:  energyKWh,
				EstimatedRevenue: revenue,
			}
		}

		// Check next period for continuous discharge
		if i < len(pricePoints)-1 {
			nextPP := pricePoints[i+1]
			if nextPP.IsPeak && nextPP.Timestamp.Sub(pp.Timestamp) == time.Hour {
				// Can extend the window
				combinedEnergy := energyKWh + dischargePowerKW
				if combinedEnergy > availableEnergyKWh {
					combinedEnergy = availableEnergyKWh
				}
				avgPrice := (pp.Price + nextPP.Price) / 2
				combinedRevenue := combinedEnergy * avgPrice * 0.9

				if combinedRevenue > bestRevenue {
					bestRevenue = combinedRevenue
					bestWindow = &DischargeWindow{
						StartTime:       pp.Timestamp,
						EndTime:         nextPP.Timestamp.Add(time.Hour),
						AveragePrice:    avgPrice,
						TotalEnergyKWh:  combinedEnergy,
						EstimatedRevenue: combinedRevenue,
					}
				}
			}
		}
	}

	if bestWindow == nil {
		return nil, fmt.Errorf("no suitable discharge window found")
	}

	return bestWindow, nil
}
