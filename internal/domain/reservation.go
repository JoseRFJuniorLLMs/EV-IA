package domain

import (
	"time"
)

// ReservationStatus represents the status of a reservation
type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusConfirmed ReservationStatus = "confirmed"
	ReservationStatusActive    ReservationStatus = "active"    // User arrived and is charging
	ReservationStatusCompleted ReservationStatus = "completed"
	ReservationStatusCancelled ReservationStatus = "cancelled"
	ReservationStatusExpired   ReservationStatus = "expired"   // User didn't arrive
	ReservationStatusNoShow    ReservationStatus = "no_show"
)

// Reservation represents a charging station reservation
type Reservation struct {
	ID              string            `json:"id" gorm:"primaryKey"`
	UserID          string            `json:"user_id" gorm:"index"`
	ChargePointID   string            `json:"charge_point_id" gorm:"index"`
	ConnectorID     int               `json:"connector_id"`
	Status          ReservationStatus `json:"status" gorm:"index"`
	StartTime       time.Time         `json:"start_time" gorm:"index"`
	EndTime         time.Time         `json:"end_time"`
	Duration        int               `json:"duration"` // Duration in minutes
	ActualArrival   *time.Time        `json:"actual_arrival,omitempty"`
	TransactionID   string            `json:"transaction_id,omitempty"` // Linked transaction when active
	Fee             float64           `json:"fee"`                      // Reservation fee
	FeePaid         bool              `json:"fee_paid"`
	Notes           string            `json:"notes,omitempty"`
	CancellationReason string         `json:"cancellation_reason,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`

	// Relations (for JSON responses)
	User        *User        `json:"user,omitempty" gorm:"foreignKey:UserID"`
	ChargePoint *ChargePoint `json:"charge_point,omitempty" gorm:"foreignKey:ChargePointID"`
}

// ReservationConfig holds reservation system configuration
type ReservationConfig struct {
	// MaxDurationMinutes is the maximum reservation duration
	MaxDurationMinutes int `json:"max_duration_minutes"`

	// MinDurationMinutes is the minimum reservation duration
	MinDurationMinutes int `json:"min_duration_minutes"`

	// MaxAdvanceBookingDays is how far in advance users can book
	MaxAdvanceBookingDays int `json:"max_advance_booking_days"`

	// GracePeriodMinutes is how long to wait before marking as no-show
	GracePeriodMinutes int `json:"grace_period_minutes"`

	// CancellationDeadlineMinutes is the deadline for free cancellation
	CancellationDeadlineMinutes int `json:"cancellation_deadline_minutes"`

	// ReservationFee is the fee for making a reservation
	ReservationFee float64 `json:"reservation_fee"`

	// NoShowPenalty is the penalty for not showing up
	NoShowPenalty float64 `json:"no_show_penalty"`

	// MaxActiveReservations is the max concurrent reservations per user
	MaxActiveReservations int `json:"max_active_reservations"`

	// RequirePaymentUpfront requires payment when making reservation
	RequirePaymentUpfront bool `json:"require_payment_upfront"`
}

// DefaultReservationConfig returns sensible defaults
func DefaultReservationConfig() *ReservationConfig {
	return &ReservationConfig{
		MaxDurationMinutes:          180, // 3 hours
		MinDurationMinutes:          30,  // 30 minutes
		MaxAdvanceBookingDays:       7,   // 1 week
		GracePeriodMinutes:          15,  // 15 minutes grace period
		CancellationDeadlineMinutes: 60,  // 1 hour before
		ReservationFee:              5.0, // R$ 5.00
		NoShowPenalty:               20.0, // R$ 20.00
		MaxActiveReservations:       2,
		RequirePaymentUpfront:       false,
	}
}

// IsActive returns true if the reservation is currently active
func (r *Reservation) IsActive() bool {
	return r.Status == ReservationStatusActive
}

// IsPending returns true if the reservation is pending or confirmed
func (r *Reservation) IsPending() bool {
	return r.Status == ReservationStatusPending || r.Status == ReservationStatusConfirmed
}

// CanBeCancelled returns true if the reservation can still be cancelled
func (r *Reservation) CanBeCancelled() bool {
	return r.Status == ReservationStatusPending || r.Status == ReservationStatusConfirmed
}

// IsExpired returns true if the reservation has expired
func (r *Reservation) IsExpired(gracePeriod time.Duration) bool {
	if r.Status != ReservationStatusConfirmed {
		return false
	}
	return time.Now().After(r.StartTime.Add(gracePeriod))
}

// TimeSlot represents an available time slot
type TimeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Available bool      `json:"available"`
}

// ReservationSummary provides a summary of reservations
type ReservationSummary struct {
	TotalReservations     int     `json:"total_reservations"`
	PendingReservations   int     `json:"pending_reservations"`
	CompletedReservations int     `json:"completed_reservations"`
	CancelledReservations int     `json:"cancelled_reservations"`
	NoShowCount           int     `json:"no_show_count"`
	TotalRevenue          float64 `json:"total_revenue"`
	AverageDuration       float64 `json:"average_duration_minutes"`
}
