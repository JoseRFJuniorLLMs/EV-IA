package reservation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Service implements ReservationService
type Service struct {
	repo          ports.ReservationRepository
	deviceRepo    ports.ChargePointRepository
	walletSvc     ports.WalletService
	config        *domain.ReservationConfig
	log           *zap.Logger
}

// NewService creates a new reservation service
func NewService(
	repo ports.ReservationRepository,
	deviceRepo ports.ChargePointRepository,
	walletSvc ports.WalletService,
	config *domain.ReservationConfig,
	log *zap.Logger,
) *Service {
	if config == nil {
		config = domain.DefaultReservationConfig()
	}

	return &Service{
		repo:       repo,
		deviceRepo: deviceRepo,
		walletSvc:  walletSvc,
		config:     config,
		log:        log,
	}
}

// CreateReservation creates a new reservation
func (s *Service) CreateReservation(ctx context.Context, req *ports.ReservationRequest) (*domain.Reservation, error) {
	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	// Check station exists and is available
	station, err := s.deviceRepo.FindByID(ctx, req.ChargePointID)
	if err != nil {
		return nil, fmt.Errorf("failed to find station: %w", err)
	}
	if station == nil {
		return nil, fmt.Errorf("station not found: %s", req.ChargePointID)
	}

	// Check user's active reservations limit
	activeCount, err := s.repo.CountByUserAndStatus(ctx, req.UserID, []domain.ReservationStatus{
		domain.ReservationStatusPending,
		domain.ReservationStatusConfirmed,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check active reservations: %w", err)
	}
	if activeCount >= s.config.MaxActiveReservations {
		return nil, fmt.Errorf("maximum active reservations reached (%d)", s.config.MaxActiveReservations)
	}

	// Calculate end time
	endTime := req.StartTime.Add(time.Duration(req.Duration) * time.Minute)

	// Check availability
	available, err := s.CheckAvailability(ctx, req.ChargePointID, req.ConnectorID, req.StartTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}
	if !available {
		return nil, fmt.Errorf("time slot not available")
	}

	// Create reservation
	reservation := &domain.Reservation{
		ID:            uuid.New().String(),
		UserID:        req.UserID,
		ChargePointID: req.ChargePointID,
		ConnectorID:   req.ConnectorID,
		Status:        domain.ReservationStatusPending,
		StartTime:     req.StartTime,
		EndTime:       endTime,
		Duration:      req.Duration,
		Fee:           s.config.ReservationFee,
		Notes:         req.Notes,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Process payment if required
	if s.config.RequirePaymentUpfront && s.config.ReservationFee > 0 {
		if s.walletSvc != nil {
			hasFunds, err := s.walletSvc.HasSufficientBalance(ctx, req.UserID, s.config.ReservationFee)
			if err != nil {
				return nil, fmt.Errorf("failed to check balance: %w", err)
			}
			if !hasFunds {
				return nil, fmt.Errorf("insufficient balance for reservation fee")
			}

			err = s.walletSvc.DeductFunds(ctx, req.UserID, s.config.ReservationFee, "Reservation fee", reservation.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to process reservation fee: %w", err)
			}
			reservation.FeePaid = true
		}
	}

	// Save reservation
	if err := s.repo.Save(ctx, reservation); err != nil {
		// Refund if payment was made
		if reservation.FeePaid && s.walletSvc != nil {
			s.walletSvc.AddFunds(ctx, req.UserID, s.config.ReservationFee, "")
		}
		return nil, fmt.Errorf("failed to save reservation: %w", err)
	}

	s.log.Info("Reservation created",
		zap.String("reservation_id", reservation.ID),
		zap.String("user_id", req.UserID),
		zap.String("station_id", req.ChargePointID),
		zap.Time("start_time", req.StartTime),
	)

	return reservation, nil
}

// validateRequest validates a reservation request
func (s *Service) validateRequest(req *ports.ReservationRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if req.ChargePointID == "" {
		return fmt.Errorf("charge point ID is required")
	}
	if req.ConnectorID <= 0 {
		return fmt.Errorf("invalid connector ID")
	}

	// Check duration
	if req.Duration < s.config.MinDurationMinutes {
		return fmt.Errorf("minimum duration is %d minutes", s.config.MinDurationMinutes)
	}
	if req.Duration > s.config.MaxDurationMinutes {
		return fmt.Errorf("maximum duration is %d minutes", s.config.MaxDurationMinutes)
	}

	// Check start time is in the future
	if req.StartTime.Before(time.Now()) {
		return fmt.Errorf("start time must be in the future")
	}

	// Check max advance booking
	maxAdvance := time.Now().AddDate(0, 0, s.config.MaxAdvanceBookingDays)
	if req.StartTime.After(maxAdvance) {
		return fmt.Errorf("cannot book more than %d days in advance", s.config.MaxAdvanceBookingDays)
	}

	return nil
}

// GetReservation retrieves a reservation by ID
func (s *Service) GetReservation(ctx context.Context, id string) (*domain.Reservation, error) {
	return s.repo.GetByID(ctx, id)
}

// GetUserReservations retrieves all reservations for a user
func (s *Service) GetUserReservations(ctx context.Context, userID string, status string, limit, offset int) ([]domain.Reservation, error) {
	return s.repo.GetByUserID(ctx, userID, status, limit, offset)
}

// GetStationReservations retrieves all reservations for a station
func (s *Service) GetStationReservations(ctx context.Context, chargePointID string, date time.Time) ([]domain.Reservation, error) {
	return s.repo.GetByChargePointID(ctx, chargePointID, date)
}

// CancelReservation cancels a reservation
func (s *Service) CancelReservation(ctx context.Context, id string, userID string, reason string) error {
	reservation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get reservation: %w", err)
	}
	if reservation == nil {
		return fmt.Errorf("reservation not found")
	}

	// Verify ownership
	if reservation.UserID != userID {
		return fmt.Errorf("not authorized to cancel this reservation")
	}

	// Check if can be cancelled
	if !reservation.CanBeCancelled() {
		return fmt.Errorf("reservation cannot be cancelled in status: %s", reservation.Status)
	}

	// Check cancellation deadline for refund
	deadline := reservation.StartTime.Add(-time.Duration(s.config.CancellationDeadlineMinutes) * time.Minute)
	refundEligible := time.Now().Before(deadline)

	// Update status
	reservation.Status = domain.ReservationStatusCancelled
	reservation.CancellationReason = reason
	reservation.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, reservation); err != nil {
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	// Process refund if eligible and fee was paid
	if refundEligible && reservation.FeePaid && s.walletSvc != nil {
		if err := s.walletSvc.AddFunds(ctx, reservation.UserID, reservation.Fee, ""); err != nil {
			s.log.Error("Failed to refund reservation fee",
				zap.String("reservation_id", id),
				zap.Error(err),
			)
		}
	}

	s.log.Info("Reservation cancelled",
		zap.String("reservation_id", id),
		zap.String("reason", reason),
		zap.Bool("refunded", refundEligible && reservation.FeePaid),
	)

	return nil
}

// ConfirmReservation confirms a pending reservation
func (s *Service) ConfirmReservation(ctx context.Context, id string) error {
	reservation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get reservation: %w", err)
	}
	if reservation == nil {
		return fmt.Errorf("reservation not found")
	}

	if reservation.Status != domain.ReservationStatusPending {
		return fmt.Errorf("can only confirm pending reservations")
	}

	reservation.Status = domain.ReservationStatusConfirmed
	reservation.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, reservation); err != nil {
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	s.log.Info("Reservation confirmed", zap.String("reservation_id", id))

	return nil
}

// ActivateReservation marks user as arrived and starts charging
func (s *Service) ActivateReservation(ctx context.Context, id string, transactionID string) error {
	reservation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get reservation: %w", err)
	}
	if reservation == nil {
		return fmt.Errorf("reservation not found")
	}

	if reservation.Status != domain.ReservationStatusConfirmed {
		return fmt.Errorf("can only activate confirmed reservations")
	}

	now := time.Now()
	reservation.Status = domain.ReservationStatusActive
	reservation.ActualArrival = &now
	reservation.TransactionID = transactionID
	reservation.UpdatedAt = now

	if err := s.repo.Save(ctx, reservation); err != nil {
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	s.log.Info("Reservation activated",
		zap.String("reservation_id", id),
		zap.String("transaction_id", transactionID),
	)

	return nil
}

// CompleteReservation marks reservation as completed
func (s *Service) CompleteReservation(ctx context.Context, id string) error {
	reservation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get reservation: %w", err)
	}
	if reservation == nil {
		return fmt.Errorf("reservation not found")
	}

	reservation.Status = domain.ReservationStatusCompleted
	reservation.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, reservation); err != nil {
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	s.log.Info("Reservation completed", zap.String("reservation_id", id))

	return nil
}

// CheckAvailability checks if a time slot is available
func (s *Service) CheckAvailability(ctx context.Context, chargePointID string, connectorID int, startTime, endTime time.Time) (bool, error) {
	// Get existing reservations that overlap
	existing, err := s.repo.GetByTimeRange(ctx, chargePointID, connectorID, startTime, endTime)
	if err != nil {
		return false, fmt.Errorf("failed to check existing reservations: %w", err)
	}

	// Filter out cancelled/completed
	for _, r := range existing {
		if r.Status == domain.ReservationStatusPending ||
			r.Status == domain.ReservationStatusConfirmed ||
			r.Status == domain.ReservationStatusActive {
			return false, nil
		}
	}

	return true, nil
}

// GetAvailableSlots returns available time slots for a station
func (s *Service) GetAvailableSlots(ctx context.Context, chargePointID string, date time.Time) ([]domain.TimeSlot, error) {
	// Get all reservations for the day
	reservations, err := s.repo.GetByChargePointID(ctx, chargePointID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get reservations: %w", err)
	}

	// Build slots for the day (6 AM to 10 PM, 30-minute slots)
	slots := make([]domain.TimeSlot, 0)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 6, 0, 0, 0, date.Location())
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 22, 0, 0, 0, date.Location())

	slotDuration := 30 * time.Minute

	for current := startOfDay; current.Before(endOfDay); current = current.Add(slotDuration) {
		slotEnd := current.Add(slotDuration)

		// Check if slot is available
		available := true
		for _, r := range reservations {
			if r.Status != domain.ReservationStatusPending &&
				r.Status != domain.ReservationStatusConfirmed &&
				r.Status != domain.ReservationStatusActive {
				continue
			}

			// Check for overlap
			if (current.Before(r.EndTime) || current.Equal(r.EndTime)) &&
				(slotEnd.After(r.StartTime) || slotEnd.Equal(r.StartTime)) {
				if current.Before(r.EndTime) && slotEnd.After(r.StartTime) {
					available = false
					break
				}
			}
		}

		// Don't show past slots
		if current.Before(time.Now()) {
			available = false
		}

		slots = append(slots, domain.TimeSlot{
			StartTime: current,
			EndTime:   slotEnd,
			Available: available,
		})
	}

	return slots, nil
}

// ProcessExpiredReservations processes reservations that have expired
func (s *Service) ProcessExpiredReservations(ctx context.Context) error {
	gracePeriod := time.Duration(s.config.GracePeriodMinutes) * time.Minute

	expired, err := s.repo.GetExpired(ctx, gracePeriod)
	if err != nil {
		return fmt.Errorf("failed to get expired reservations: %w", err)
	}

	for _, r := range expired {
		r.Status = domain.ReservationStatusNoShow
		r.UpdatedAt = time.Now()

		if err := s.repo.Save(ctx, &r); err != nil {
			s.log.Error("Failed to mark reservation as no-show",
				zap.String("reservation_id", r.ID),
				zap.Error(err),
			)
			continue
		}

		// Apply no-show penalty
		if s.config.NoShowPenalty > 0 && s.walletSvc != nil {
			if err := s.walletSvc.DeductFunds(ctx, r.UserID, s.config.NoShowPenalty, "No-show penalty", r.ID); err != nil {
				s.log.Error("Failed to apply no-show penalty",
					zap.String("reservation_id", r.ID),
					zap.Error(err),
				)
			}
		}

		s.log.Info("Reservation marked as no-show",
			zap.String("reservation_id", r.ID),
			zap.String("user_id", r.UserID),
		)
	}

	return nil
}

// GetReservationSummary returns reservation statistics
func (s *Service) GetReservationSummary(ctx context.Context, chargePointID string, startDate, endDate time.Time) (*domain.ReservationSummary, error) {
	// This would typically be a database aggregation query
	// For now, return a placeholder
	return &domain.ReservationSummary{
		TotalReservations:     0,
		PendingReservations:   0,
		CompletedReservations: 0,
		CancelledReservations: 0,
		NoShowCount:           0,
		TotalRevenue:          0,
		AverageDuration:       0,
	}, nil
}
