package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/adapter/queue"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// PricingConfig holds the pricing configuration
type PricingConfig struct {
	BaseRatePerKWh     float64 // Base rate per kWh
	PeakRateMultiplier float64 // Multiplier for peak hours
	IdleFeePerMinute   float64 // Fee per minute when vehicle stays connected after charging
	Currency           string  // Currency code (e.g., "BRL")
	PeakHoursStart     int     // Peak hours start (e.g., 18 for 6 PM)
	PeakHoursEnd       int     // Peak hours end (e.g., 21 for 9 PM)
}

// DefaultPricingConfig returns the default pricing configuration
func DefaultPricingConfig() *PricingConfig {
	return &PricingConfig{
		BaseRatePerKWh:     0.75,  // R$ 0.75 per kWh
		PeakRateMultiplier: 1.5,   // 50% more during peak hours
		IdleFeePerMinute:   0.10,  // R$ 0.10 per minute idle
		Currency:           "BRL",
		PeakHoursStart:     18,    // 6 PM
		PeakHoursEnd:       21,    // 9 PM
	}
}

// BillingService handles billing and payment calculations
type BillingService struct {
	txRepo  ports.TransactionRepository
	mq      queue.MessageQueue
	pricing *PricingConfig
	log     *zap.Logger
}

// NewBillingService creates a new billing service
func NewBillingService(
	txRepo ports.TransactionRepository,
	mq queue.MessageQueue,
	pricing *PricingConfig,
	log *zap.Logger,
) *BillingService {
	if pricing == nil {
		pricing = DefaultPricingConfig()
	}
	return &BillingService{
		txRepo:  txRepo,
		mq:      mq,
		pricing: pricing,
		log:     log,
	}
}

// CalculateCost calculates the total cost of a transaction
func (s *BillingService) CalculateCost(ctx context.Context, tx *domain.Transaction) (float64, error) {
	if tx == nil {
		return 0, errors.New("transaction cannot be nil")
	}

	// Calculate energy cost
	energyKWh := float64(tx.TotalEnergy) / 1000.0 // Convert Wh to kWh
	rate := s.getRate(tx.StartTime)
	energyCost := energyKWh * rate

	// Calculate idle fee if applicable
	idleFee := s.calculateIdleFee(tx)

	totalCost := energyCost + idleFee

	s.log.Info("Calculated transaction cost",
		zap.String("tx_id", tx.ID),
		zap.Float64("energy_kwh", energyKWh),
		zap.Float64("rate", rate),
		zap.Float64("energy_cost", energyCost),
		zap.Float64("idle_fee", idleFee),
		zap.Float64("total_cost", totalCost),
	)

	return totalCost, nil
}

// getRate returns the rate based on time of day
func (s *BillingService) getRate(startTime time.Time) float64 {
	hour := startTime.Hour()
	if hour >= s.pricing.PeakHoursStart && hour < s.pricing.PeakHoursEnd {
		return s.pricing.BaseRatePerKWh * s.pricing.PeakRateMultiplier
	}
	return s.pricing.BaseRatePerKWh
}

// calculateIdleFee calculates the idle fee if the vehicle stayed connected after charging
func (s *BillingService) calculateIdleFee(tx *domain.Transaction) float64 {
	if tx.EndTime == nil {
		return 0
	}

	// Estimate charging duration based on energy and assumed power
	// In a real implementation, this would come from meter values
	estimatedChargingMinutes := float64(tx.TotalEnergy) / 1000.0 / 7.0 * 60 // Assume 7kW average
	actualDuration := tx.EndTime.Sub(tx.StartTime).Minutes()

	idleMinutes := actualDuration - estimatedChargingMinutes
	if idleMinutes <= 5 { // Grace period of 5 minutes
		return 0
	}

	return (idleMinutes - 5) * s.pricing.IdleFeePerMinute
}

// ProcessPayment processes the payment for a completed transaction
func (s *BillingService) ProcessPayment(ctx context.Context, tx *domain.Transaction) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if tx.Status != domain.TransactionStatusStopped && tx.Status != domain.TransactionStatusCompleted {
		return errors.New("can only process payment for stopped or completed transactions")
	}

	// Calculate final cost
	cost, err := s.CalculateCost(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to calculate cost: %w", err)
	}

	// Update transaction with cost
	tx.Cost = cost
	tx.Currency = s.pricing.Currency
	tx.Status = domain.TransactionStatusCompleted
	tx.UpdatedAt = time.Now()

	if err := s.txRepo.Update(ctx, tx); err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	// Publish payment event for external processing (e.g., Stripe)
	paymentEvent := map[string]interface{}{
		"event_type":     "payment.required",
		"transaction_id": tx.ID,
		"user_id":        tx.UserID,
		"amount":         cost,
		"currency":       s.pricing.Currency,
		"energy_kwh":     float64(tx.TotalEnergy) / 1000.0,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	if data, err := json.Marshal(paymentEvent); err == nil {
		if err := s.mq.Publish("billing.payment.required", data); err != nil {
			s.log.Warn("Failed to publish payment event", zap.Error(err))
		}
	}

	s.log.Info("Payment processed",
		zap.String("tx_id", tx.ID),
		zap.String("user_id", tx.UserID),
		zap.Float64("amount", cost),
		zap.String("currency", s.pricing.Currency),
	)

	return nil
}

// GetPricePerKWh returns the current price per kWh
func (s *BillingService) GetPricePerKWh(ctx context.Context) float64 {
	return s.getRate(time.Now())
}

// GenerateInvoice generates an invoice for a transaction
func (s *BillingService) GenerateInvoice(ctx context.Context, tx *domain.Transaction) (*Invoice, error) {
	if tx == nil {
		return nil, errors.New("transaction cannot be nil")
	}

	energyKWh := float64(tx.TotalEnergy) / 1000.0
	rate := s.getRate(tx.StartTime)
	idleFee := s.calculateIdleFee(tx)

	var duration time.Duration
	if tx.EndTime != nil {
		duration = tx.EndTime.Sub(tx.StartTime)
	}

	invoice := &Invoice{
		InvoiceID:       fmt.Sprintf("INV-%s", tx.ID[:8]),
		TransactionID:   tx.ID,
		UserID:          tx.UserID,
		ChargePointID:   tx.ChargePointID,
		StartTime:       tx.StartTime,
		EndTime:         tx.EndTime,
		Duration:        duration,
		EnergyKWh:       energyKWh,
		RatePerKWh:      rate,
		EnergyCost:      energyKWh * rate,
		IdleFee:         idleFee,
		TotalAmount:     tx.Cost,
		Currency:        tx.Currency,
		GeneratedAt:     time.Now(),
	}

	return invoice, nil
}

// Invoice represents a billing invoice
type Invoice struct {
	InvoiceID       string        `json:"invoice_id"`
	TransactionID   string        `json:"transaction_id"`
	UserID          string        `json:"user_id"`
	ChargePointID   string        `json:"charge_point_id"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         *time.Time    `json:"end_time,omitempty"`
	Duration        time.Duration `json:"duration"`
	EnergyKWh       float64       `json:"energy_kwh"`
	RatePerKWh      float64       `json:"rate_per_kwh"`
	EnergyCost      float64       `json:"energy_cost"`
	IdleFee         float64       `json:"idle_fee"`
	TotalAmount     float64       `json:"total_amount"`
	Currency        string        `json:"currency"`
	GeneratedAt     time.Time     `json:"generated_at"`
}
