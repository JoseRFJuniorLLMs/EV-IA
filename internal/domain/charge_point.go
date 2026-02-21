package domain

import (
	"time"
)

type ChargePointStatus string

const (
	ChargePointStatusAvailable   ChargePointStatus = "Available"
	ChargePointStatusOccupied    ChargePointStatus = "Occupied"
	ChargePointStatusFaulted     ChargePointStatus = "Faulted"
	ChargePointStatusUnavailable ChargePointStatus = "Unavailable"
	ChargePointStatusCharging    ChargePointStatus = "Charging"
)

type ChargePoint struct {
	ID              string            `json:"id" gorm:"primaryKey"`
	Vendor          string            `json:"vendor"`
	Model           string            `json:"model"`
	SerialNumber    string            `json:"serial_number"`
	FirmwareVersion string            `json:"firmware_version"`
	Status          ChargePointStatus `json:"status"`
	LocationID      string            `json:"location_id"`
	Location        *Location         `json:"location,omitempty" gorm:"foreignKey:LocationID"`
	Connectors      []Connector       `json:"connectors" gorm:"foreignKey:ChargePointID"`
	LastSeen        time.Time         `json:"last_seen"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type Connector struct {
	ID            uint              `json:"id" gorm:"primaryKey"`
	ChargePointID string            `json:"charge_point_id" gorm:"index"` // Foreign key
	ConnectorID   int               `json:"connector_id"`                 // The standard 1-based connector ID
	Type          string            `json:"type"`                         // e.g., CCS, CHAdeMO, Type2
	Status        ChargePointStatus `json:"status"`
	MaxPowerKW    float64           `json:"max_power_kw"`
}

type Location struct {
	ID        string  `json:"id" gorm:"primaryKey"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Country   string  `json:"country"`
}
