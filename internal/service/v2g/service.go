package v2g

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Service implements the V2G service
type Service struct {
	v2gRepo         ports.V2GRepository
	deviceService   ports.DeviceService
	txService       ports.TransactionService
	gridPriceService ports.GridPriceService
	ocppServer      ports.OCPPCommandService
	mq              ports.MessageQueue
	log             *zap.Logger

	// In-memory tracking
	activeSessions  map[string]*domain.V2GSession
	capabilities    map[string]*domain.V2GCapability
	mu              sync.RWMutex

	config          *Config
}

// Config holds V2G service configuration
type Config struct {
	DefaultMinSOC           int     // Default minimum SOC to preserve (%)
	DefaultMaxDischargeKWh  float64 // Default max discharge per session
	OperatorMargin          float64 // Operator margin on V2G compensation (0.10 = 10%)
	MinGridPriceForV2G      float64 // Minimum grid price to consider V2G worthwhile (R$/kWh)
	CompensationCurrency    string  // Currency for compensation (BRL)
}

// DefaultConfig returns default V2G configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultMinSOC:          20,    // 20%
		DefaultMaxDischargeKWh: 50.0,  // 50 kWh
		OperatorMargin:         0.10,  // 10%
		MinGridPriceForV2G:     0.80,  // R$ 0.80/kWh
		CompensationCurrency:   "BRL",
	}
}

// NewService creates a new V2G service
func NewService(
	v2gRepo ports.V2GRepository,
	deviceService ports.DeviceService,
	txService ports.TransactionService,
	gridPriceService ports.GridPriceService,
	ocppServer ports.OCPPCommandService,
	mq ports.MessageQueue,
	log *zap.Logger,
	config *Config,
) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		v2gRepo:          v2gRepo,
		deviceService:    deviceService,
		txService:        txService,
		gridPriceService: gridPriceService,
		ocppServer:       ocppServer,
		mq:               mq,
		log:              log,
		activeSessions:   make(map[string]*domain.V2GSession),
		capabilities:     make(map[string]*domain.V2GCapability),
		config:           config,
	}
}

// DischargeRequest represents a request to start V2G discharge
type DischargeRequest struct {
	ChargePointID string
	ConnectorID   int
	UserID        string
	MaxPowerKW    float64
	MaxEnergyKWh  float64
	MinBatterySOC int
	EndTime       *time.Time
}

// StartDischarge initiates a V2G discharge session
func (s *Service) StartDischarge(ctx context.Context, req *DischargeRequest) (*domain.V2GSession, error) {
	// Validate charge point exists and is connected
	device, err := s.deviceService.GetDevice(ctx, req.ChargePointID)
	if err != nil {
		return nil, fmt.Errorf("charge point not found: %w", err)
	}

	if device.Status != domain.ChargePointStatusOccupied {
		return nil, errors.New("charge point must be occupied (EV connected) for V2G")
	}

	// Check V2G capability
	cap, err := s.CheckV2GCapability(ctx, req.ChargePointID)
	if err != nil {
		return nil, fmt.Errorf("failed to check V2G capability: %w", err)
	}

	if !cap.Supported {
		return nil, errors.New("connected vehicle does not support V2G")
	}

	// Apply defaults
	minSOC := req.MinBatterySOC
	if minSOC == 0 {
		minSOC = s.config.DefaultMinSOC
	}

	maxEnergy := req.MaxEnergyKWh
	if maxEnergy == 0 {
		maxEnergy = s.config.DefaultMaxDischargeKWh
	}

	maxPower := req.MaxPowerKW
	if maxPower == 0 || maxPower > cap.MaxDischargePowerKW {
		maxPower = cap.MaxDischargePowerKW
	}

	// Check SOC constraint
	if cap.CurrentSOC <= minSOC {
		return nil, fmt.Errorf("current SOC (%d%%) is at or below minimum (%d%%)", cap.CurrentSOC, minSOC)
	}

	// Get current grid price
	gridPrice, err := s.gridPriceService.GetCurrentPrice(ctx)
	if err != nil {
		s.log.Warn("Failed to get grid price, using default", zap.Error(err))
		gridPrice = 0.75 // Default price
	}

	// Create V2G session
	session := &domain.V2GSession{
		ID:               uuid.New().String(),
		ChargePointID:    req.ChargePointID,
		ConnectorID:      req.ConnectorID,
		UserID:           req.UserID,
		Direction:        domain.V2GDirectionDischarging,
		RequestedPowerKW: maxPower,
		MinBatterySOC:    minSOC,
		CurrentSOC:       cap.CurrentSOC,
		GridPriceAtStart: gridPrice,
		CurrentGridPrice: gridPrice,
		Status:           domain.V2GStatusPending,
		StartTime:        time.Now(),
	}

	// Calculate discharge duration
	durationSeconds := 3600 // Default 1 hour
	if req.EndTime != nil {
		durationSeconds = int(req.EndTime.Sub(time.Now()).Seconds())
		if durationSeconds <= 0 {
			return nil, errors.New("end time must be in the future")
		}
	}

	// Send V2G charging profile to charge point via OCPP
	err = s.ocppServer.SetV2GChargingProfile(ctx, req.ChargePointID, req.ConnectorID, maxPower, durationSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to set V2G profile on charge point: %w", err)
	}

	session.Status = domain.V2GStatusActive

	// Store session
	s.mu.Lock()
	s.activeSessions[session.ID] = session
	s.mu.Unlock()

	// Persist to database
	if s.v2gRepo != nil {
		if err := s.v2gRepo.CreateSession(ctx, session); err != nil {
			s.log.Error("Failed to persist V2G session", zap.Error(err))
		}
	}

	// Publish event
	if s.mq != nil {
		s.mq.Publish("v2g.session.started", map[string]interface{}{
			"session_id":      session.ID,
			"charge_point_id": session.ChargePointID,
			"user_id":         session.UserID,
			"power_kw":        session.RequestedPowerKW,
			"grid_price":      gridPrice,
		})
	}

	s.log.Info("V2G discharge session started",
		zap.String("sessionID", session.ID),
		zap.String("chargePointID", req.ChargePointID),
		zap.String("userID", req.UserID),
		zap.Float64("powerKW", maxPower),
		zap.Float64("gridPrice", gridPrice),
	)

	return session, nil
}

// StopDischarge stops an active V2G discharge session
func (s *Service) StopDischarge(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	session, ok := s.activeSessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	delete(s.activeSessions, sessionID)
	s.mu.Unlock()

	// Clear V2G profile from charge point
	err := s.ocppServer.ClearV2GChargingProfile(ctx, session.ChargePointID, session.ConnectorID)
	if err != nil {
		s.log.Error("Failed to clear V2G profile", zap.Error(err))
	}

	// Update session
	now := time.Now()
	session.EndTime = &now
	session.Status = domain.V2GStatusCompleted

	// Calculate compensation
	compensation, err := s.CalculateCompensation(ctx, session)
	if err != nil {
		s.log.Error("Failed to calculate compensation", zap.Error(err))
	} else {
		session.UserCompensation = compensation.NetAmount
	}

	// Persist update
	if s.v2gRepo != nil {
		if err := s.v2gRepo.UpdateSession(ctx, session); err != nil {
			s.log.Error("Failed to update V2G session", zap.Error(err))
		}
	}

	// Publish event
	if s.mq != nil {
		s.mq.Publish("v2g.session.completed", map[string]interface{}{
			"session_id":        session.ID,
			"charge_point_id":   session.ChargePointID,
			"user_id":           session.UserID,
			"energy_kwh":        session.EnergyTransferred,
			"compensation":      session.UserCompensation,
			"duration_seconds":  now.Sub(session.StartTime).Seconds(),
		})
	}

	s.log.Info("V2G discharge session stopped",
		zap.String("sessionID", sessionID),
		zap.Float64("energyKWh", session.EnergyTransferred),
		zap.Float64("compensation", session.UserCompensation),
	)

	return nil
}

// GetActiveSession returns the active V2G session for a charge point
func (s *Service) GetActiveSession(ctx context.Context, chargePointID string) (*domain.V2GSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.activeSessions {
		if session.ChargePointID == chargePointID && session.Status == domain.V2GStatusActive {
			return session, nil
		}
	}

	return nil, nil
}

// GetSession returns a V2G session by ID
func (s *Service) GetSession(ctx context.Context, sessionID string) (*domain.V2GSession, error) {
	s.mu.RLock()
	session, ok := s.activeSessions[sessionID]
	s.mu.RUnlock()

	if ok {
		return session, nil
	}

	// Try database
	if s.v2gRepo != nil {
		return s.v2gRepo.GetSession(ctx, sessionID)
	}

	return nil, fmt.Errorf("session %s not found", sessionID)
}

// CalculateCompensation calculates the compensation for a V2G session
func (s *Service) CalculateCompensation(ctx context.Context, session *domain.V2GSession) (*domain.V2GCompensation, error) {
	if session.EnergyTransferred >= 0 {
		// No discharge occurred
		return &domain.V2GCompensation{
			SessionID:           session.ID,
			UserID:              session.UserID,
			EnergyDischargedKWh: 0,
			Currency:            s.config.CompensationCurrency,
			CalculatedAt:        time.Now(),
		}, nil
	}

	// Energy transferred is negative for discharge
	energyDischarged := -session.EnergyTransferred

	// Use average of start and current price
	avgPrice := (session.GridPriceAtStart + session.CurrentGridPrice) / 2

	// Calculate gross amount (what the energy is worth)
	grossAmount := energyDischarged * avgPrice

	// Apply operator margin
	netAmount := grossAmount * (1 - s.config.OperatorMargin)

	return &domain.V2GCompensation{
		SessionID:           session.ID,
		UserID:              session.UserID,
		EnergyDischargedKWh: energyDischarged,
		AverageGridPrice:    avgPrice,
		OperatorMargin:      s.config.OperatorMargin,
		GrossAmount:         grossAmount,
		NetAmount:           netAmount,
		Currency:            s.config.CompensationCurrency,
		CalculatedAt:        time.Now(),
	}, nil
}

// CheckV2GCapability checks if a vehicle at a charge point supports V2G
func (s *Service) CheckV2GCapability(ctx context.Context, chargePointID string) (*domain.V2GCapability, error) {
	key := chargePointID

	// Check cache first
	s.mu.RLock()
	cap, ok := s.capabilities[key]
	s.mu.RUnlock()

	if ok && time.Since(cap.LastUpdated) < 5*time.Minute {
		return cap, nil
	}

	// Get from OCPP server's V2G manager
	ocppCap, err := s.ocppServer.GetV2GCapability(ctx, chargePointID)
	if err != nil {
		// Return cached even if stale
		if cap != nil {
			return cap, nil
		}
		return nil, err
	}

	// Update cache
	s.mu.Lock()
	s.capabilities[key] = ocppCap
	s.mu.Unlock()

	return ocppCap, nil
}

// SetUserPreferences sets V2G preferences for a user
func (s *Service) SetUserPreferences(ctx context.Context, userID string, prefs *domain.V2GPreferences) error {
	prefs.UserID = userID
	if s.v2gRepo != nil {
		return s.v2gRepo.SavePreferences(ctx, prefs)
	}
	return nil
}

// GetUserPreferences gets V2G preferences for a user
func (s *Service) GetUserPreferences(ctx context.Context, userID string) (*domain.V2GPreferences, error) {
	if s.v2gRepo != nil {
		return s.v2gRepo.GetPreferences(ctx, userID)
	}

	// Return defaults
	return &domain.V2GPreferences{
		UserID:          userID,
		AutoDischarge:   false,
		MinGridPrice:    s.config.MinGridPriceForV2G,
		MaxDischargeKWh: s.config.DefaultMaxDischargeKWh,
		PreserveSOC:     s.config.DefaultMinSOC,
		NotifyOnStart:   true,
		NotifyOnEnd:     true,
	}, nil
}

// OptimizeV2G automatically optimizes V2G based on preferences and grid prices
func (s *Service) OptimizeV2G(ctx context.Context, chargePointID string, userID string) error {
	// Get user preferences
	prefs, err := s.GetUserPreferences(ctx, userID)
	if err != nil {
		return err
	}

	if !prefs.AutoDischarge {
		return nil // Auto-discharge not enabled
	}

	// Get current grid price
	currentPrice, err := s.gridPriceService.GetCurrentPrice(ctx)
	if err != nil {
		return err
	}

	// Check if price meets threshold
	if currentPrice < prefs.MinGridPrice {
		return nil // Price not high enough
	}

	// Check V2G capability
	cap, err := s.CheckV2GCapability(ctx, chargePointID)
	if err != nil || !cap.Supported {
		return nil
	}

	// Check if already in a session
	existingSession, _ := s.GetActiveSession(ctx, chargePointID)
	if existingSession != nil {
		return nil // Already discharging
	}

	// Start discharge
	req := &DischargeRequest{
		ChargePointID: chargePointID,
		ConnectorID:   cap.ConnectorID,
		UserID:        userID,
		MaxEnergyKWh:  prefs.MaxDischargeKWh,
		MinBatterySOC: prefs.PreserveSOC,
	}

	_, err = s.StartDischarge(ctx, req)
	if err != nil {
		return err
	}

	s.log.Info("Auto V2G discharge started",
		zap.String("chargePointID", chargePointID),
		zap.String("userID", userID),
		zap.Float64("gridPrice", currentPrice),
	)

	return nil
}

// UpdateSessionMetrics updates energy and power metrics for an active session
func (s *Service) UpdateSessionMetrics(ctx context.Context, sessionID string, powerKW, energyKWh float64, currentSOC int) error {
	s.mu.Lock()
	session, ok := s.activeSessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.ActualPowerKW = powerKW
	session.EnergyTransferred = energyKWh // Will be negative for discharge
	session.CurrentSOC = currentSOC

	// Get current grid price
	if s.gridPriceService != nil {
		if price, err := s.gridPriceService.GetCurrentPrice(ctx); err == nil {
			session.CurrentGridPrice = price
		}
	}
	s.mu.Unlock()

	// Persist update
	if s.v2gRepo != nil {
		if err := s.v2gRepo.UpdateSession(ctx, session); err != nil {
			s.log.Error("Failed to update V2G session", zap.Error(err))
		}
	}

	// Publish update event
	if s.mq != nil {
		s.mq.Publish("v2g.session.updated", map[string]interface{}{
			"session_id": sessionID,
			"power_kw":   powerKW,
			"energy_kwh": energyKWh,
			"soc":        currentSOC,
		})
	}

	// Check if SOC reached minimum
	if currentSOC <= session.MinBatterySOC {
		s.log.Info("V2G session stopping due to min SOC reached",
			zap.String("sessionID", sessionID),
			zap.Int("currentSOC", currentSOC),
			zap.Int("minSOC", session.MinBatterySOC),
		)
		return s.StopDischarge(ctx, sessionID)
	}

	return nil
}

// GetUserStats returns V2G statistics for a user
func (s *Service) GetUserStats(ctx context.Context, userID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	if s.v2gRepo != nil {
		return s.v2gRepo.GetUserStats(ctx, userID, startDate, endDate)
	}

	return &domain.V2GStats{
		EntityID:   userID,
		EntityType: "user",
		StartDate:  startDate,
		EndDate:    endDate,
	}, nil
}

// GetChargePointStats returns V2G statistics for a charge point
func (s *Service) GetChargePointStats(ctx context.Context, chargePointID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	if s.v2gRepo != nil {
		return s.v2gRepo.GetChargePointStats(ctx, chargePointID, startDate, endDate)
	}

	return &domain.V2GStats{
		EntityID:   chargePointID,
		EntityType: "charge_point",
		StartDate:  startDate,
		EndDate:    endDate,
	}, nil
}
