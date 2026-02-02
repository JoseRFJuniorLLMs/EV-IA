package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/adapter/queue"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// ChargingProfile represents a smart charging schedule
type ChargingProfile struct {
	ProfileID       string           `json:"profile_id"`
	DeviceID        string           `json:"device_id"`
	ConnectorID     int              `json:"connector_id"`
	ProfilePurpose  string           `json:"profile_purpose"` // ChargePointMaxProfile, TxDefaultProfile, TxProfile
	StackLevel      int              `json:"stack_level"`
	ChargingSchedule *ChargingSchedule `json:"charging_schedule"`
	ValidFrom       *time.Time       `json:"valid_from,omitempty"`
	ValidTo         *time.Time       `json:"valid_to,omitempty"`
}

// ChargingSchedule represents a charging schedule within a profile
type ChargingSchedule struct {
	Duration            int                      `json:"duration,omitempty"` // in seconds
	StartSchedule       *time.Time               `json:"start_schedule,omitempty"`
	ChargingRateUnit    string                   `json:"charging_rate_unit"` // W or A
	MinChargingRate     float64                  `json:"min_charging_rate,omitempty"`
	ChargingSchedulePeriods []ChargingSchedulePeriod `json:"charging_schedule_period"`
}

// ChargingSchedulePeriod represents a period within a charging schedule
type ChargingSchedulePeriod struct {
	StartPeriod     int     `json:"start_period"` // Start in seconds from schedule start
	Limit           float64 `json:"limit"`        // Power limit in W or current limit in A
	NumberPhases    int     `json:"number_phases,omitempty"`
}

// SmartChargingConfig holds the smart charging configuration
type SmartChargingConfig struct {
	MaxSitePowerKW      float64 // Maximum power available at the site
	DefaultMaxPowerKW   float64 // Default max power per connector
	MinPowerKW          float64 // Minimum power to maintain charging
	LoadBalancingEnabled bool    // Enable load balancing between chargers
	PeakShavingEnabled  bool    // Enable peak shaving during high demand
	V2GEnabled          bool    // Enable Vehicle-to-Grid (future)
}

// DefaultSmartChargingConfig returns the default smart charging configuration
func DefaultSmartChargingConfig() *SmartChargingConfig {
	return &SmartChargingConfig{
		MaxSitePowerKW:      500.0, // 500 kW site capacity
		DefaultMaxPowerKW:   150.0, // 150 kW per connector
		MinPowerKW:          7.0,   // 7 kW minimum
		LoadBalancingEnabled: true,
		PeakShavingEnabled:  true,
		V2GEnabled:          false, // Not yet implemented
	}
}

// SmartChargingService handles intelligent charging optimization
type SmartChargingService struct {
	deviceRepo ports.ChargePointRepository
	txRepo     ports.TransactionRepository
	mq         queue.MessageQueue
	config     *SmartChargingConfig
	log        *zap.Logger
}

// NewSmartChargingService creates a new smart charging service
func NewSmartChargingService(
	deviceRepo ports.ChargePointRepository,
	txRepo ports.TransactionRepository,
	mq queue.MessageQueue,
	config *SmartChargingConfig,
	log *zap.Logger,
) *SmartChargingService {
	if config == nil {
		config = DefaultSmartChargingConfig()
	}
	return &SmartChargingService{
		deviceRepo: deviceRepo,
		txRepo:     txRepo,
		mq:         mq,
		config:     config,
		log:        log,
	}
}

// OptimizeCharging creates an optimized charging profile for a device
func (s *SmartChargingService) OptimizeCharging(
	ctx context.Context,
	deviceID string,
	connectorID int,
	targetEnergyKWh float64,
	departureTime *time.Time,
) (*ChargingProfile, error) {
	if deviceID == "" {
		return nil, errors.New("device ID is required")
	}

	// Get device information
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, errors.New("device not found")
	}

	// Get connector max power
	var maxPowerKW float64 = s.config.DefaultMaxPowerKW
	for _, conn := range device.Connectors {
		if conn.ConnectorID == connectorID {
			maxPowerKW = conn.MaxPowerKW
			break
		}
	}

	// Calculate available power based on site load
	availablePower := s.calculateAvailablePower(ctx)
	if availablePower < maxPowerKW {
		maxPowerKW = availablePower
	}

	// Ensure minimum power
	if maxPowerKW < s.config.MinPowerKW {
		maxPowerKW = s.config.MinPowerKW
	}

	// Create charging schedule
	now := time.Now()
	schedule := s.createOptimalSchedule(targetEnergyKWh, maxPowerKW, departureTime, now)

	profile := &ChargingProfile{
		ProfileID:      fmt.Sprintf("PROF-%s-%d-%d", deviceID[:8], connectorID, now.Unix()),
		DeviceID:       deviceID,
		ConnectorID:    connectorID,
		ProfilePurpose: "TxProfile",
		StackLevel:     1,
		ChargingSchedule: schedule,
		ValidFrom:      &now,
	}

	// Publish profile to OCPP (to be sent to the charger)
	if data, err := json.Marshal(profile); err == nil {
		if err := s.mq.Publish("ocpp.set_charging_profile", data); err != nil {
			s.log.Warn("Failed to publish charging profile", zap.Error(err))
		}
	}

	s.log.Info("Created optimized charging profile",
		zap.String("profile_id", profile.ProfileID),
		zap.String("device_id", deviceID),
		zap.Float64("max_power_kw", maxPowerKW),
		zap.Float64("target_energy_kwh", targetEnergyKWh),
	)

	return profile, nil
}

// calculateAvailablePower calculates available power based on site load
func (s *SmartChargingService) calculateAvailablePower(ctx context.Context) float64 {
	// In a real implementation, this would:
	// 1. Query all active charging sessions
	// 2. Sum current power consumption
	// 3. Calculate remaining capacity

	// Simplified: assume 80% of site capacity is available
	return s.config.MaxSitePowerKW * 0.8
}

// createOptimalSchedule creates an optimal charging schedule
func (s *SmartChargingService) createOptimalSchedule(
	targetEnergyKWh float64,
	maxPowerKW float64,
	departureTime *time.Time,
	startTime time.Time,
) *ChargingSchedule {
	periods := make([]ChargingSchedulePeriod, 0)

	// If no departure time, charge at max power
	if departureTime == nil || departureTime.Before(startTime) {
		periods = append(periods, ChargingSchedulePeriod{
			StartPeriod:  0,
			Limit:        maxPowerKW * 1000, // Convert to W
			NumberPhases: 3,
		})

		return &ChargingSchedule{
			ChargingRateUnit:        "W",
			MinChargingRate:         s.config.MinPowerKW * 1000,
			ChargingSchedulePeriods: periods,
		}
	}

	// Calculate time until departure
	availableTime := departureTime.Sub(startTime)
	availableHours := availableTime.Hours()

	// Calculate required charging rate
	requiredPowerKW := targetEnergyKWh / availableHours

	// Ensure within limits
	if requiredPowerKW > maxPowerKW {
		requiredPowerKW = maxPowerKW
	}
	if requiredPowerKW < s.config.MinPowerKW {
		requiredPowerKW = s.config.MinPowerKW
	}

	// Create schedule with peak shaving if enabled
	if s.config.PeakShavingEnabled {
		periods = s.createPeakShavingSchedule(requiredPowerKW, maxPowerKW, startTime, availableTime)
	} else {
		periods = append(periods, ChargingSchedulePeriod{
			StartPeriod:  0,
			Limit:        requiredPowerKW * 1000,
			NumberPhases: 3,
		})
	}

	duration := int(availableTime.Seconds())

	return &ChargingSchedule{
		Duration:                duration,
		StartSchedule:          &startTime,
		ChargingRateUnit:       "W",
		MinChargingRate:        s.config.MinPowerKW * 1000,
		ChargingSchedulePeriods: periods,
	}
}

// createPeakShavingSchedule creates a schedule that avoids peak hours
func (s *SmartChargingService) createPeakShavingSchedule(
	avgPowerKW float64,
	maxPowerKW float64,
	startTime time.Time,
	duration time.Duration,
) []ChargingSchedulePeriod {
	periods := make([]ChargingSchedulePeriod, 0)

	// Peak hours: 18:00 - 21:00
	peakStart := 18
	peakEnd := 21

	// Calculate periods
	currentTime := startTime
	endTime := startTime.Add(duration)
	periodStart := 0

	for currentTime.Before(endTime) {
		hour := currentTime.Hour()
		isPeak := hour >= peakStart && hour < peakEnd

		// Calculate power for this period
		var periodPower float64
		if isPeak {
			// Reduce power during peak hours
			periodPower = math.Min(avgPowerKW*0.5, maxPowerKW*0.5)
		} else {
			// Increase power during off-peak to compensate
			periodPower = math.Min(avgPowerKW*1.5, maxPowerKW)
		}

		// Find end of this period (either peak transition or end of charging)
		var periodEnd time.Time
		if isPeak {
			periodEnd = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), peakEnd, 0, 0, 0, currentTime.Location())
		} else if hour < peakStart {
			periodEnd = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), peakStart, 0, 0, 0, currentTime.Location())
		} else {
			// After peak, go to next day's peak
			nextDay := currentTime.Add(24 * time.Hour)
			periodEnd = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), peakStart, 0, 0, 0, nextDay.Location())
		}

		if periodEnd.After(endTime) {
			periodEnd = endTime
		}

		periods = append(periods, ChargingSchedulePeriod{
			StartPeriod:  periodStart,
			Limit:        periodPower * 1000,
			NumberPhases: 3,
		})

		periodStart += int(periodEnd.Sub(currentTime).Seconds())
		currentTime = periodEnd
	}

	return periods
}

// GetChargingProfile retrieves the current charging profile for a device
func (s *SmartChargingService) GetChargingProfile(ctx context.Context, deviceID string, connectorID int) (*ChargingProfile, error) {
	// In a real implementation, this would query the OCPP server or database
	// for the current active profile

	s.log.Info("Getting charging profile",
		zap.String("device_id", deviceID),
		zap.Int("connector_id", connectorID),
	)

	// Return nil if no profile is set (charger uses default)
	return nil, nil
}

// ClearChargingProfile removes a charging profile from a device
func (s *SmartChargingService) ClearChargingProfile(ctx context.Context, deviceID string, connectorID int) error {
	clearRequest := map[string]interface{}{
		"device_id":    deviceID,
		"connector_id": connectorID,
	}

	if data, err := json.Marshal(clearRequest); err == nil {
		if err := s.mq.Publish("ocpp.clear_charging_profile", data); err != nil {
			return fmt.Errorf("failed to publish clear profile request: %w", err)
		}
	}

	s.log.Info("Cleared charging profile",
		zap.String("device_id", deviceID),
		zap.Int("connector_id", connectorID),
	)

	return nil
}

// LoadBalance performs load balancing across all active charging sessions
func (s *SmartChargingService) LoadBalance(ctx context.Context) error {
	if !s.config.LoadBalancingEnabled {
		return nil
	}

	// Get all devices
	devices, err := s.deviceRepo.FindAll(ctx, map[string]interface{}{
		"status": domain.ChargePointStatusCharging,
	})
	if err != nil {
		return fmt.Errorf("failed to get charging devices: %w", err)
	}

	if len(devices) == 0 {
		return nil
	}

	// Calculate fair share of power
	fairShareKW := s.config.MaxSitePowerKW / float64(len(devices))

	// Apply limits to each device
	for _, device := range devices {
		for _, conn := range device.Connectors {
			if conn.Status == domain.ChargePointStatusCharging {
				limit := math.Min(fairShareKW, conn.MaxPowerKW)

				profile := &ChargingProfile{
					ProfileID:      fmt.Sprintf("LB-%s-%d-%d", device.ID[:8], conn.ConnectorID, time.Now().Unix()),
					DeviceID:       device.ID,
					ConnectorID:    conn.ConnectorID,
					ProfilePurpose: "ChargePointMaxProfile",
					StackLevel:     0, // Highest priority
					ChargingSchedule: &ChargingSchedule{
						ChargingRateUnit: "W",
						ChargingSchedulePeriods: []ChargingSchedulePeriod{
							{
								StartPeriod:  0,
								Limit:        limit * 1000,
								NumberPhases: 3,
							},
						},
					},
				}

				if data, err := json.Marshal(profile); err == nil {
					s.mq.Publish("ocpp.set_charging_profile", data)
				}
			}
		}
	}

	s.log.Info("Load balancing completed",
		zap.Int("device_count", len(devices)),
		zap.Float64("fair_share_kw", fairShareKW),
	)

	return nil
}
