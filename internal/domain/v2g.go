package domain

import "time"

// V2GDirection represents the direction of energy flow
type V2GDirection string

const (
	V2GDirectionCharging    V2GDirection = "Charging"
	V2GDirectionDischarging V2GDirection = "Discharging"
	V2GDirectionIdle        V2GDirection = "Idle"
)

// V2GStatus represents the status of a V2G session
type V2GStatus string

const (
	V2GStatusPending   V2GStatus = "Pending"
	V2GStatusActive    V2GStatus = "Active"
	V2GStatusCompleted V2GStatus = "Completed"
	V2GStatusFailed    V2GStatus = "Failed"
	V2GStatusCancelled V2GStatus = "Cancelled"
)

// V2GSession represents a Vehicle-to-Grid session
type V2GSession struct {
	ID                string       `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TransactionID     string       `json:"transaction_id" gorm:"index"`
	ChargePointID     string       `json:"charge_point_id" gorm:"index"`
	ConnectorID       int          `json:"connector_id"`
	UserID            string       `json:"user_id" gorm:"index"`
	VehicleID         string       `json:"vehicle_id,omitempty"`
	Direction         V2GDirection `json:"direction"`
	RequestedPowerKW  float64      `json:"requested_power_kw"`
	ActualPowerKW     float64      `json:"actual_power_kw"`
	EnergyTransferred float64      `json:"energy_transferred"` // kWh (positive = charging, negative = discharging)
	GridPriceAtStart  float64      `json:"grid_price_at_start"` // R$/kWh at session start
	CurrentGridPrice  float64      `json:"current_grid_price"`  // Current R$/kWh
	UserCompensation  float64      `json:"user_compensation"`   // Total compensation to pay user
	MinBatterySOC     int          `json:"min_battery_soc"`     // Minimum SOC to maintain
	CurrentSOC        int          `json:"current_soc"`         // Current battery SOC
	StartTime         time.Time    `json:"start_time"`
	EndTime           *time.Time   `json:"end_time,omitempty"`
	Status            V2GStatus    `json:"status"`
	CreatedAt         time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

// V2GCapability represents a vehicle's V2G capabilities
type V2GCapability struct {
	ChargePointID         string  `json:"charge_point_id"`
	ConnectorID           int     `json:"connector_id"`
	Supported             bool    `json:"supported"`
	MaxDischargePowerKW   float64 `json:"max_discharge_power_kw"`
	MaxDischargeCurrent   float64 `json:"max_discharge_current"`
	BidirectionalCharging bool    `json:"bidirectional_charging"`
	ISO15118Support       bool    `json:"iso15118_support"`
	CurrentSOC            int     `json:"current_soc"`
	BatteryCapacityKWh    float64 `json:"battery_capacity_kwh"`
	LastUpdated           time.Time `json:"last_updated"`
}

// V2GPreferences represents user preferences for V2G
type V2GPreferences struct {
	ID              string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID          string    `json:"user_id" gorm:"uniqueIndex"`
	AutoDischarge   bool      `json:"auto_discharge"`   // Automatically discharge when grid price is high
	MinGridPrice    float64   `json:"min_grid_price"`   // Minimum R$/kWh to accept discharge
	MaxDischargeKWh float64   `json:"max_discharge_kwh"` // Maximum kWh to discharge per day
	PreserveSOC     int       `json:"preserve_soc"`     // Minimum battery SOC to maintain (%)
	NotifyOnStart   bool      `json:"notify_on_start"`  // Notify when V2G session starts
	NotifyOnEnd     bool      `json:"notify_on_end"`    // Notify when V2G session ends
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// V2GEvent represents a V2G event for analytics/audit
type V2GEvent struct {
	ID            string       `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID     string       `json:"session_id" gorm:"index"`
	ChargePointID string       `json:"charge_point_id"`
	EventType     string       `json:"event_type"` // started, updated, completed, failed, compensated
	Direction     V2GDirection `json:"direction"`
	PowerKW       float64      `json:"power_kw"`
	EnergyKWh     float64      `json:"energy_kwh"`
	GridPrice     float64      `json:"grid_price"`
	Details       string       `json:"details,omitempty"` // JSON details
	Timestamp     time.Time    `json:"timestamp"`
}

// GridPricePoint represents a price point for grid electricity
type GridPricePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"` // R$/kWh
	IsPeak    bool      `json:"is_peak"`
	Source    string    `json:"source"` // CCEE, simulated, custom
}

// V2GCompensation represents compensation calculation for V2G
type V2GCompensation struct {
	SessionID        string    `json:"session_id"`
	UserID           string    `json:"user_id"`
	EnergyDischargedKWh float64 `json:"energy_discharged_kwh"`
	AverageGridPrice float64   `json:"average_grid_price"`
	OperatorMargin   float64   `json:"operator_margin"` // Percentage (e.g., 0.10 for 10%)
	GrossAmount      float64   `json:"gross_amount"`    // Before operator margin
	NetAmount        float64   `json:"net_amount"`      // Amount to pay user
	Currency         string    `json:"currency"`        // BRL
	CalculatedAt     time.Time `json:"calculated_at"`
}

// V2GStats represents V2G statistics for a user or charge point
type V2GStats struct {
	EntityID             string    `json:"entity_id"` // UserID or ChargePointID
	EntityType           string    `json:"entity_type"` // "user" or "charge_point"
	TotalSessions        int       `json:"total_sessions"`
	TotalEnergyDischargedKWh float64 `json:"total_energy_discharged_kwh"`
	TotalCompensation    float64   `json:"total_compensation"`
	AverageSessionDuration time.Duration `json:"average_session_duration"`
	PeakHoursParticipation float64 `json:"peak_hours_participation"` // Percentage of sessions during peak
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
}

// ISO15118VehicleIdentity represents vehicle identity from ISO 15118 certificate
type ISO15118VehicleIdentity struct {
	EMAID      string `json:"emaid"`       // E-Mobility Account Identifier
	ContractID string `json:"contract_id"`
	VehicleVIN string `json:"vehicle_vin,omitempty"`
	V2GCapable bool   `json:"v2g_capable"`
	ValidFrom  time.Time `json:"valid_from"`
	ValidTo    time.Time `json:"valid_to"`
}

// ChargingContract represents an ISO 15118 charging contract
type ChargingContract struct {
	ID                  string    `json:"id"`
	ContractID          string    `json:"contract_id"`
	EMAID               string    `json:"emaid"`
	ProviderID          string    `json:"provider_id"`
	ContractType        string    `json:"contract_type"` // standard, v2g
	V2GEnabled          bool      `json:"v2g_enabled"`
	MaxChargePowerKW    float64   `json:"max_charge_power_kw"`
	MaxDischargePowerKW float64   `json:"max_discharge_power_kw,omitempty"`
	ValidFrom           time.Time `json:"valid_from"`
	ValidTo             time.Time `json:"valid_to"`
}

// ISO15118Certificate represents a stored ISO 15118 certificate
type ISO15118Certificate struct {
	ID                  string     `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	EMAID               string     `json:"emaid" gorm:"type:varchar(100);uniqueIndex;not null"`
	ContractID          string     `json:"contract_id" gorm:"type:varchar(100);uniqueIndex;not null"`
	VehicleVIN          string     `json:"vehicle_vin,omitempty" gorm:"type:varchar(50);index"`
	CertificatePEM      string     `json:"certificate_pem" gorm:"type:text;not null"`
	CertificateChain    string     `json:"certificate_chain,omitempty" gorm:"type:text"`
	PrivateKeyEncrypted string     `json:"private_key_encrypted,omitempty" gorm:"type:text"`
	V2GCapable          bool       `json:"v2g_capable" gorm:"default:false"`
	ValidFrom           time.Time  `json:"valid_from" gorm:"not null"`
	ValidTo             time.Time  `json:"valid_to" gorm:"not null;index"`
	Revoked             bool       `json:"revoked" gorm:"default:false"`
	RevokedAt           *time.Time `json:"revoked_at,omitempty"`
	RevocationReason    string     `json:"revocation_reason,omitempty" gorm:"type:varchar(200)"`
	ProviderID          string     `json:"provider_id,omitempty" gorm:"type:varchar(50)"`
	MaxChargePowerKW    float64    `json:"max_charge_power_kw,omitempty" gorm:"type:decimal(10,2)"`
	MaxDischargePowerKW float64    `json:"max_discharge_power_kw,omitempty" gorm:"type:decimal(10,2)"`
	CreatedAt           time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"not null;default:now()"`
}

// TableName returns the table name for GORM
func (ISO15118Certificate) TableName() string {
	return "iso15118_certificates"
}
