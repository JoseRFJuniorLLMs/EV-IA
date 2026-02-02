package domain

import (
	"time"
)

type TransactionStatus string

const (
	TransactionStatusStarted   TransactionStatus = "Started"
	TransactionStatusStopped   TransactionStatus = "Stopped"
	TransactionStatusFaulted   TransactionStatus = "Faulted"
	TransactionStatusCompleted TransactionStatus = "Completed"
)

type Transaction struct {
	ID            string            `json:"id" gorm:"primaryKey"`
	ChargePointID string            `json:"charge_point_id" gorm:"index"`
	ConnectorID   int               `json:"connector_id"`
	UserID        string            `json:"user_id" gorm:"index"`
	IdTag         string            `json:"id_tag"` // RFID or other auth token
	StartTime     time.Time         `json:"start_time"`
	EndTime       *time.Time        `json:"end_time,omitempty"`
	MeterStart    int               `json:"meter_start"`  // Wh
	MeterStop     int               `json:"meter_stop"`   // Wh
	TotalEnergy   int               `json:"total_energy"` // Wh
	Status        TransactionStatus `json:"status"`
	Cost          float64           `json:"cost"`
	Currency      string            `json:"currency"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}
