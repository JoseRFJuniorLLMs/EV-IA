package v2g

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// V2GPaymentService handles V2G compensation payments
type V2GPaymentService struct {
	v2gRepo       ports.V2GRepository
	walletService ports.WalletService
	mq            ports.MessageQueue
	log           *zap.Logger
	config        *PaymentConfig
}

// PaymentConfig holds payment service configuration
type PaymentConfig struct {
	OperatorMargin      float64 // Operator margin (0.10 = 10%)
	MinPayoutAmount     float64 // Minimum amount to process payout (R$)
	PayoutCurrency      string  // Currency (BRL)
	AutoPayoutEnabled   bool    // Automatically process payouts
	PayoutSchedule      string  // cron expression for batch payouts
}

// DefaultPaymentConfig returns default payment configuration
func DefaultPaymentConfig() *PaymentConfig {
	return &PaymentConfig{
		OperatorMargin:    0.10,   // 10%
		MinPayoutAmount:   5.00,   // R$ 5.00 minimum
		PayoutCurrency:    "BRL",
		AutoPayoutEnabled: true,
		PayoutSchedule:    "0 0 * * *", // Daily at midnight
	}
}

// NewV2GPaymentService creates a new V2G payment service
func NewV2GPaymentService(
	v2gRepo ports.V2GRepository,
	walletService ports.WalletService,
	mq ports.MessageQueue,
	log *zap.Logger,
	config *PaymentConfig,
) *V2GPaymentService {
	if config == nil {
		config = DefaultPaymentConfig()
	}

	return &V2GPaymentService{
		v2gRepo:       v2gRepo,
		walletService: walletService,
		mq:            mq,
		log:           log,
		config:        config,
	}
}

// V2GCompensationRecord represents a compensation record
type V2GCompensationRecord struct {
	ID                  string    `json:"id"`
	SessionID           string    `json:"session_id"`
	UserID              string    `json:"user_id"`
	EnergyDischargedKWh float64   `json:"energy_discharged_kwh"`
	AverageGridPrice    float64   `json:"average_grid_price"`
	OperatorMargin      float64   `json:"operator_margin"`
	GrossAmount         float64   `json:"gross_amount"`
	NetAmount           float64   `json:"net_amount"`
	Currency            string    `json:"currency"`
	Status              string    `json:"status"` // pending, processed, paid, failed
	PaymentID           string    `json:"payment_id,omitempty"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	CalculatedAt        time.Time `json:"calculated_at"`
	CreatedAt           time.Time `json:"created_at"`
}

// CalculateAndRecordCompensation calculates and records compensation for a V2G session
func (s *V2GPaymentService) CalculateAndRecordCompensation(ctx context.Context, session *domain.V2GSession) (*V2GCompensationRecord, error) {
	// Only process completed discharge sessions
	if session.Status != domain.V2GStatusCompleted || session.Direction != domain.V2GDirectionDischarging {
		return nil, fmt.Errorf("session not eligible for compensation")
	}

	// Energy transferred is negative for discharge, so we use absolute value
	energyDischarged := session.EnergyTransferred
	if energyDischarged > 0 {
		energyDischarged = -energyDischarged
	}
	energyDischarged = -energyDischarged // Make positive

	if energyDischarged <= 0 {
		return nil, fmt.Errorf("no energy discharged in session")
	}

	// Calculate average price
	avgPrice := (session.GridPriceAtStart + session.CurrentGridPrice) / 2

	// Calculate amounts
	grossAmount := energyDischarged * avgPrice
	netAmount := grossAmount * (1 - s.config.OperatorMargin)

	record := &V2GCompensationRecord{
		ID:                  uuid.New().String(),
		SessionID:           session.ID,
		UserID:              session.UserID,
		EnergyDischargedKWh: energyDischarged,
		AverageGridPrice:    avgPrice,
		OperatorMargin:      s.config.OperatorMargin,
		GrossAmount:         grossAmount,
		NetAmount:           netAmount,
		Currency:            s.config.PayoutCurrency,
		Status:              "pending",
		CalculatedAt:        time.Now(),
		CreatedAt:           time.Now(),
	}

	s.log.Info("V2G compensation calculated",
		zap.String("sessionID", session.ID),
		zap.String("userID", session.UserID),
		zap.Float64("energyKWh", energyDischarged),
		zap.Float64("grossAmount", grossAmount),
		zap.Float64("netAmount", netAmount),
	)

	// Publish event
	if s.mq != nil {
		s.mq.Publish("v2g.compensation.calculated", map[string]interface{}{
			"record_id":   record.ID,
			"session_id":  record.SessionID,
			"user_id":     record.UserID,
			"net_amount":  record.NetAmount,
			"currency":    record.Currency,
		})
	}

	return record, nil
}

// ProcessPayout processes a compensation payout to user's wallet
func (s *V2GPaymentService) ProcessPayout(ctx context.Context, record *V2GCompensationRecord) error {
	if record.Status != "pending" {
		return fmt.Errorf("compensation already processed: %s", record.Status)
	}

	if record.NetAmount < s.config.MinPayoutAmount {
		s.log.Debug("Compensation below minimum payout amount",
			zap.String("recordID", record.ID),
			zap.Float64("amount", record.NetAmount),
			zap.Float64("minimum", s.config.MinPayoutAmount),
		)
		return nil // Will be batched later
	}

	// Add funds to user's wallet
	err := s.walletService.AddFunds(
		ctx,
		record.UserID,
		record.NetAmount,
		fmt.Sprintf("v2g-compensation-%s", record.ID),
	)
	if err != nil {
		record.Status = "failed"
		s.log.Error("Failed to process V2G payout",
			zap.String("recordID", record.ID),
			zap.Error(err),
		)

		if s.mq != nil {
			s.mq.Publish("v2g.compensation.failed", map[string]interface{}{
				"record_id": record.ID,
				"user_id":   record.UserID,
				"error":     err.Error(),
			})
		}

		return fmt.Errorf("failed to add funds to wallet: %w", err)
	}

	// Update record
	now := time.Now()
	record.Status = "paid"
	record.PaidAt = &now
	record.PaymentID = fmt.Sprintf("WALLET-%s", uuid.New().String()[:8])

	s.log.Info("V2G compensation paid",
		zap.String("recordID", record.ID),
		zap.String("userID", record.UserID),
		zap.Float64("amount", record.NetAmount),
		zap.String("paymentID", record.PaymentID),
	)

	// Publish success event
	if s.mq != nil {
		s.mq.Publish("v2g.compensation.paid", map[string]interface{}{
			"record_id":  record.ID,
			"user_id":    record.UserID,
			"amount":     record.NetAmount,
			"payment_id": record.PaymentID,
		})
	}

	return nil
}

// ProcessSessionCompensation processes compensation for a completed V2G session (end-to-end)
func (s *V2GPaymentService) ProcessSessionCompensation(ctx context.Context, session *domain.V2GSession) error {
	// Calculate compensation
	record, err := s.CalculateAndRecordCompensation(ctx, session)
	if err != nil {
		return err
	}

	// Process payout if auto-payout is enabled
	if s.config.AutoPayoutEnabled && record.NetAmount >= s.config.MinPayoutAmount {
		return s.ProcessPayout(ctx, record)
	}

	return nil
}

// GetPendingCompensations retrieves all pending compensations for a user
func (s *V2GPaymentService) GetPendingCompensations(ctx context.Context, userID string) ([]V2GCompensationRecord, error) {
	// This would query the database
	// For now, return empty slice
	return []V2GCompensationRecord{}, nil
}

// GetCompensationHistory retrieves compensation history for a user
func (s *V2GPaymentService) GetCompensationHistory(ctx context.Context, userID string, limit, offset int) ([]V2GCompensationRecord, error) {
	// This would query the database
	return []V2GCompensationRecord{}, nil
}

// GetTotalCompensation returns total compensation paid to a user
func (s *V2GPaymentService) GetTotalCompensation(ctx context.Context, userID string) (float64, error) {
	// This would query the database
	return 0, nil
}

// BatchProcessPendingPayouts processes all pending payouts in batch
func (s *V2GPaymentService) BatchProcessPendingPayouts(ctx context.Context) (int, error) {
	// Get all pending compensations
	// This would be called by a scheduled job

	processed := 0
	// Implement batch processing logic here

	s.log.Info("Batch payout processing completed",
		zap.Int("processed", processed),
	)

	return processed, nil
}

// GenerateCompensationReport generates a compensation report for a period
func (s *V2GPaymentService) GenerateCompensationReport(ctx context.Context, startDate, endDate time.Time) (*CompensationReport, error) {
	report := &CompensationReport{
		StartDate: startDate,
		EndDate:   endDate,
		GeneratedAt: time.Now(),
	}

	// This would aggregate data from the database

	return report, nil
}

// CompensationReport represents a V2G compensation report
type CompensationReport struct {
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
	TotalSessions       int       `json:"total_sessions"`
	TotalEnergyKWh      float64   `json:"total_energy_kwh"`
	TotalGrossAmount    float64   `json:"total_gross_amount"`
	TotalNetAmount      float64   `json:"total_net_amount"`
	TotalOperatorRevenue float64  `json:"total_operator_revenue"`
	UniqueUsers         int       `json:"unique_users"`
	AveragePerSession   float64   `json:"average_per_session"`
	Currency            string    `json:"currency"`
	GeneratedAt         time.Time `json:"generated_at"`
}
