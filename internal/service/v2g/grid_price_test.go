package v2g

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestGridPriceService_GetCurrentPrice(t *testing.T) {
	logger := zap.NewNop()
	config := &GridPriceConfig{
		BasePrice:       0.75,
		PeakMultiplier:  1.8,
		OffPeakDiscount: 0.7,
	}
	service := NewGridPriceService(nil, logger, config)

	ctx := context.Background()
	price, err := service.GetCurrentPrice(ctx)
	if err != nil {
		t.Fatalf("GetCurrentPrice failed: %v", err)
	}

	// Price should be positive
	if price <= 0 {
		t.Errorf("Expected positive price, got %f", price)
	}

	// Price should be reasonable (between 0.50 and 2.00 R$/kWh)
	if price < 0.50 || price > 2.00 {
		t.Errorf("Price %f seems unreasonable", price)
	}
}

func TestGridPriceService_IsPeakHour(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(nil, logger, nil)
	ctx := context.Background()

	// Test peak hour detection
	isPeak, err := service.IsPeakHour(ctx)
	if err != nil {
		t.Fatalf("IsPeakHour failed: %v", err)
	}

	// Result should be boolean (no error checking needed, just verify it doesn't panic)
	t.Logf("Current hour is peak: %v", isPeak)
}

func TestGridPriceService_GetPriceForecast(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(nil, logger, nil)
	ctx := context.Background()

	hours := 24
	forecast, err := service.GetPriceForecast(ctx, hours)
	if err != nil {
		t.Fatalf("GetPriceForecast failed: %v", err)
	}

	if len(forecast) != hours {
		t.Errorf("Expected %d forecast points, got %d", hours, len(forecast))
	}

	// Verify forecast structure
	for i, point := range forecast {
		if point.Price <= 0 {
			t.Errorf("Forecast point %d has invalid price: %f", i, point.Price)
		}
		if point.Timestamp.IsZero() {
			t.Errorf("Forecast point %d has zero timestamp", i)
		}
	}

	// Verify timestamps are sequential
	for i := 1; i < len(forecast); i++ {
		if !forecast[i].Timestamp.After(forecast[i-1].Timestamp) {
			t.Errorf("Forecast timestamps not sequential at index %d", i)
		}
	}
}

func TestGridPriceService_CalculateV2GCompensation(t *testing.T) {
	logger := zap.NewNop()
	config := &GridPriceConfig{
		BasePrice:         0.75,
		PeakMultiplier:    1.8,
		OperatorMargin:    0.10,
		V2GBonusMultiplier: 1.1,
	}
	service := NewGridPriceService(nil, logger, config)
	ctx := context.Background()

	energyKWh := 20.0
	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now()

	compensation, err := service.CalculateV2GCompensation(ctx, energyKWh, startTime, endTime)
	if err != nil {
		t.Fatalf("CalculateV2GCompensation failed: %v", err)
	}

	// Compensation should be positive
	if compensation <= 0 {
		t.Errorf("Expected positive compensation, got %f", compensation)
	}

	// Compensation should be less than raw energy value (due to operator margin)
	maxPossible := energyKWh * config.BasePrice * config.PeakMultiplier * 1.5 // generous upper bound
	if compensation > maxPossible {
		t.Errorf("Compensation %f exceeds maximum possible %f", compensation, maxPossible)
	}

	t.Logf("Compensation for %.2f kWh: R$ %.2f", energyKWh, compensation)
}

func TestGridPriceService_PeakHourPricing(t *testing.T) {
	logger := zap.NewNop()
	config := &GridPriceConfig{
		BasePrice:       0.75,
		PeakMultiplier:  1.8,
		OffPeakDiscount: 0.7,
	}
	service := NewGridPriceService(nil, logger, config)

	// Test price at different hours
	tests := []struct {
		hour     int
		weekday  time.Weekday
		minPrice float64
		maxPrice float64
		isPeak   bool
	}{
		{3, time.Monday, 0.40, 0.70, false},      // Off-peak night
		{10, time.Monday, 0.60, 0.90, false},     // Regular weekday
		{19, time.Monday, 1.00, 1.50, true},      // Peak hour
		{15, time.Saturday, 0.40, 0.80, false},   // Weekend
		{19, time.Sunday, 0.40, 0.80, false},     // Weekend evening (not peak)
	}

	for _, tt := range tests {
		// Create a specific time for testing
		testTime := time.Date(2026, 2, 2+int(tt.weekday), tt.hour, 0, 0, 0, time.Local)

		// Calculate expected price based on hour
		price := service.calculatePriceForTime(testTime)

		if price < tt.minPrice || price > tt.maxPrice {
			t.Errorf("Hour %d, %s: price %f not in expected range [%f, %f]",
				tt.hour, tt.weekday, price, tt.minPrice, tt.maxPrice)
		}

		isPeak := service.isPeakTime(testTime)
		if isPeak != tt.isPeak {
			t.Errorf("Hour %d, %s: isPeak=%v, expected %v",
				tt.hour, tt.weekday, isPeak, tt.isPeak)
		}
	}
}

func TestGridPriceService_BrazilianTariffStructure(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(nil, logger, nil)

	// Test Brazilian tariff periods
	// Ponta (peak): 18:00-21:00 weekdays
	// Fora-ponta (off-peak): other times

	// Peak hours should have higher prices
	peakTime := time.Date(2026, 2, 2, 19, 0, 0, 0, time.Local) // Monday 19:00
	offPeakTime := time.Date(2026, 2, 2, 3, 0, 0, 0, time.Local) // Monday 03:00

	peakPrice := service.calculatePriceForTime(peakTime)
	offPeakPrice := service.calculatePriceForTime(offPeakTime)

	if peakPrice <= offPeakPrice {
		t.Errorf("Peak price (%f) should be higher than off-peak (%f)", peakPrice, offPeakPrice)
	}

	// Peak should be significantly higher (at least 50% more)
	ratio := peakPrice / offPeakPrice
	if ratio < 1.5 {
		t.Errorf("Peak/off-peak ratio (%f) should be at least 1.5", ratio)
	}
}

func TestGridPriceService_ForecastConsistency(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(nil, logger, nil)
	ctx := context.Background()

	// Get two forecasts in sequence - should have some overlap
	forecast1, _ := service.GetPriceForecast(ctx, 24)
	forecast2, _ := service.GetPriceForecast(ctx, 24)

	// First point of both forecasts should be similar (both are "now")
	diff := forecast1[0].Price - forecast2[0].Price
	if diff < -0.01 || diff > 0.01 {
		t.Errorf("Sequential forecasts have inconsistent current price: %f vs %f",
			forecast1[0].Price, forecast2[0].Price)
	}
}
