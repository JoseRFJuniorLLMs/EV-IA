package v2g

import (
	"context"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// GridPriceService implements dynamic grid pricing for V2G
type GridPriceService struct {
	log            *zap.Logger
	config         *GridPriceConfig
	priceCache     []domain.GridPricePoint
	lastFetch      time.Time
	mu             sync.RWMutex
}

// GridPriceConfig holds configuration for the grid price service
type GridPriceConfig struct {
	// Base prices (R$/kWh)
	BasePriceOffPeak float64 // Off-peak base price
	BasePricePeak    float64 // Peak base price
	BasePriceSuperPeak float64 // Super-peak base price

	// Peak hours definition
	PeakStartHour      int // Start of peak hours (e.g., 17)
	PeakEndHour        int // End of peak hours (e.g., 21)
	SuperPeakStartHour int // Start of super-peak (e.g., 18)
	SuperPeakEndHour   int // End of super-peak (e.g., 20)

	// Weekend adjustment
	WeekendMultiplier float64 // Multiplier for weekend prices (usually lower)

	// Dynamic adjustments
	HighDemandMultiplier float64 // Multiplier when demand is high
	LowDemandMultiplier  float64 // Multiplier when demand is low

	// Source (for future API integration)
	PriceSource string // "simulated", "ccee", "custom"

	// Cache duration
	CacheDuration time.Duration
}

// DefaultGridPriceConfig returns default Brazilian grid pricing configuration
func DefaultGridPriceConfig() *GridPriceConfig {
	return &GridPriceConfig{
		BasePriceOffPeak:     0.55,  // R$ 0.55/kWh off-peak
		BasePricePeak:        0.85,  // R$ 0.85/kWh peak
		BasePriceSuperPeak:   1.20,  // R$ 1.20/kWh super-peak

		PeakStartHour:        17,    // 5 PM
		PeakEndHour:          21,    // 9 PM
		SuperPeakStartHour:   18,    // 6 PM
		SuperPeakEndHour:     20,    // 8 PM

		WeekendMultiplier:    0.8,   // 20% lower on weekends

		HighDemandMultiplier: 1.5,
		LowDemandMultiplier:  0.7,

		PriceSource:          "simulated",
		CacheDuration:        5 * time.Minute,
	}
}

// NewGridPriceService creates a new grid price service
func NewGridPriceService(log *zap.Logger, config *GridPriceConfig) *GridPriceService {
	if config == nil {
		config = DefaultGridPriceConfig()
	}

	return &GridPriceService{
		log:    log,
		config: config,
	}
}

// GetCurrentPrice returns the current grid price in R$/kWh
func (s *GridPriceService) GetCurrentPrice(ctx context.Context) (float64, error) {
	now := time.Now()
	return s.getPriceAtTime(now), nil
}

// GetPriceForecast returns price forecast for the next N hours
func (s *GridPriceService) GetPriceForecast(ctx context.Context, hours int) ([]domain.GridPricePoint, error) {
	s.mu.RLock()
	if len(s.priceCache) > 0 && time.Since(s.lastFetch) < s.config.CacheDuration {
		s.mu.RUnlock()
		return s.priceCache, nil
	}
	s.mu.RUnlock()

	// Generate forecast
	forecast := make([]domain.GridPricePoint, hours)
	now := time.Now().Truncate(time.Hour)

	for i := 0; i < hours; i++ {
		t := now.Add(time.Duration(i) * time.Hour)
		price := s.getPriceAtTime(t)
		isPeak := s.isPeakHour(t)

		forecast[i] = domain.GridPricePoint{
			Timestamp: t,
			Price:     price,
			IsPeak:    isPeak,
			Source:    s.config.PriceSource,
		}
	}

	// Cache the forecast
	s.mu.Lock()
	s.priceCache = forecast
	s.lastFetch = time.Now()
	s.mu.Unlock()

	return forecast, nil
}

// IsPeakHour checks if the current time is during peak hours
func (s *GridPriceService) IsPeakHour(ctx context.Context) (bool, error) {
	return s.isPeakHour(time.Now()), nil
}

// IsSuperPeakHour checks if current time is during super-peak hours
func (s *GridPriceService) IsSuperPeakHour(ctx context.Context) (bool, error) {
	return s.isSuperPeakHour(time.Now()), nil
}

// CalculateV2GCompensation calculates the compensation for V2G energy discharge
func (s *GridPriceService) CalculateV2GCompensation(ctx context.Context, energyKWh float64, startTime, endTime time.Time) (float64, error) {
	// Calculate average price over the discharge period
	totalPrice := 0.0
	hours := 0

	for t := startTime; t.Before(endTime); t = t.Add(time.Hour) {
		totalPrice += s.getPriceAtTime(t)
		hours++
	}

	if hours == 0 {
		hours = 1
		totalPrice = s.getPriceAtTime(startTime)
	}

	avgPrice := totalPrice / float64(hours)

	// Apply operator margin (90% to user)
	compensation := energyKWh * avgPrice * 0.9

	return compensation, nil
}

// GetOptimalDischargeHours returns the hours with highest prices for V2G
func (s *GridPriceService) GetOptimalDischargeHours(ctx context.Context, hoursNeeded int) ([]time.Time, error) {
	forecast, err := s.GetPriceForecast(ctx, 24) // Next 24 hours
	if err != nil {
		return nil, err
	}

	// Sort by price (descending)
	type priceHour struct {
		time  time.Time
		price float64
	}

	priceHours := make([]priceHour, len(forecast))
	for i, pp := range forecast {
		priceHours[i] = priceHour{time: pp.Timestamp, price: pp.Price}
	}

	// Simple bubble sort for small array
	for i := 0; i < len(priceHours)-1; i++ {
		for j := 0; j < len(priceHours)-i-1; j++ {
			if priceHours[j].price < priceHours[j+1].price {
				priceHours[j], priceHours[j+1] = priceHours[j+1], priceHours[j]
			}
		}
	}

	// Return top N hours
	result := make([]time.Time, 0, hoursNeeded)
	for i := 0; i < hoursNeeded && i < len(priceHours); i++ {
		result = append(result, priceHours[i].time)
	}

	return result, nil
}

// GetDailyStats returns pricing statistics for a day
func (s *GridPriceService) GetDailyStats(ctx context.Context, date time.Time) (*GridPriceStats, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	var minPrice, maxPrice, totalPrice float64
	minPrice = math.MaxFloat64
	peakHours := 0

	for hour := 0; hour < 24; hour++ {
		t := startOfDay.Add(time.Duration(hour) * time.Hour)
		price := s.getPriceAtTime(t)

		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
		totalPrice += price

		if s.isPeakHour(t) {
			peakHours++
		}
	}

	return &GridPriceStats{
		Date:         date,
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		AveragePrice: totalPrice / 24,
		PeakHours:    peakHours,
	}, nil
}

// GridPriceStats holds daily price statistics
type GridPriceStats struct {
	Date         time.Time
	MinPrice     float64
	MaxPrice     float64
	AveragePrice float64
	PeakHours    int
}

// Internal methods

func (s *GridPriceService) getPriceAtTime(t time.Time) float64 {
	hour := t.Hour()
	weekday := t.Weekday()

	var basePrice float64

	// Determine base price based on time
	if s.isSuperPeakHour(t) {
		basePrice = s.config.BasePriceSuperPeak
	} else if s.isPeakHour(t) {
		basePrice = s.config.BasePricePeak
	} else {
		basePrice = s.config.BasePriceOffPeak
	}

	// Apply weekend discount
	if weekday == time.Saturday || weekday == time.Sunday {
		basePrice *= s.config.WeekendMultiplier
	}

	// Add some variation based on hour for realism
	// Morning ramp-up (6-9 AM)
	if hour >= 6 && hour < 9 {
		basePrice *= 1.0 + float64(hour-6)*0.05
	}

	// Late night discount (11 PM - 5 AM)
	if hour >= 23 || hour < 5 {
		basePrice *= 0.85
	}

	// Add small random-ish variation based on minute
	// This creates slight price movement without true randomness
	minute := t.Minute()
	variation := 1.0 + float64(minute%10-5)/100.0 // ±5% variation
	basePrice *= variation

	// Round to 2 decimal places
	return math.Round(basePrice*100) / 100
}

func (s *GridPriceService) isPeakHour(t time.Time) bool {
	hour := t.Hour()
	weekday := t.Weekday()

	// No peak hours on weekends
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	return hour >= s.config.PeakStartHour && hour < s.config.PeakEndHour
}

func (s *GridPriceService) isSuperPeakHour(t time.Time) bool {
	hour := t.Hour()
	weekday := t.Weekday()

	// No super-peak on weekends
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	return hour >= s.config.SuperPeakStartHour && hour < s.config.SuperPeakEndHour
}

// --- Future: CCEE Integration ---

// CCEEPrice represents a price from CCEE (Câmara de Comercialização de Energia Elétrica)
type CCEEPrice struct {
	Timestamp    time.Time
	PLD          float64 // Preço de Liquidação das Diferenças (R$/MWh)
	Region       string  // SE/CO, S, NE, N
	LoadLevel    string  // pesada, média, leve
}

// FetchCCEEPrices fetches prices from CCEE API (future implementation)
func (s *GridPriceService) FetchCCEEPrices(ctx context.Context) ([]CCEEPrice, error) {
	// TODO: Implement CCEE API integration
	// API docs: https://www.ccee.org.br/
	return nil, nil
}

// ConvertCCEEToRetail converts CCEE wholesale price to retail price
func (s *GridPriceService) ConvertCCEEToRetail(cceePrice float64) float64 {
	// CCEE prices are in R$/MWh, convert to R$/kWh
	retailPrice := cceePrice / 1000.0

	// Add distribution and taxes (approximately 40% on top)
	retailPrice *= 1.40

	return retailPrice
}
