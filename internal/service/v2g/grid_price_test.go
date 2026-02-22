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
		BasePriceOffPeak:   0.55,
		BasePricePeak:      0.85,
		BasePriceSuperPeak: 1.20,
		PeakStartHour:      17,
		PeakEndHour:        21,
		SuperPeakStartHour: 18,
		SuperPeakEndHour:   20,
		WeekendMultiplier:  0.8,
	}
	service := NewGridPriceService(logger, config)

	ctx := context.Background()
	price, err := service.GetCurrentPrice(ctx)
	if err != nil {
		t.Fatalf("GetCurrentPrice failed: %v", err)
	}

	if price <= 0 {
		t.Errorf("Expected positive price, got %f", price)
	}

	if price < 0.30 || price > 2.00 {
		t.Errorf("Price %f seems unreasonable", price)
	}
}

func TestGridPriceService_IsPeakHour(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(logger, nil)
	ctx := context.Background()

	isPeak, err := service.IsPeakHour(ctx)
	if err != nil {
		t.Fatalf("IsPeakHour failed: %v", err)
	}

	t.Logf("Current hour is peak: %v", isPeak)
}

func TestGridPriceService_GetPriceForecast(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(logger, nil)
	ctx := context.Background()

	hours := 24
	forecast, err := service.GetPriceForecast(ctx, hours)
	if err != nil {
		t.Fatalf("GetPriceForecast failed: %v", err)
	}

	if len(forecast) != hours {
		t.Errorf("Expected %d forecast points, got %d", hours, len(forecast))
	}

	for i, point := range forecast {
		if point.Price <= 0 {
			t.Errorf("Forecast point %d has invalid price: %f", i, point.Price)
		}
		if point.Timestamp.IsZero() {
			t.Errorf("Forecast point %d has zero timestamp", i)
		}
	}

	for i := 1; i < len(forecast); i++ {
		if !forecast[i].Timestamp.After(forecast[i-1].Timestamp) {
			t.Errorf("Forecast timestamps not sequential at index %d", i)
		}
	}
}

func TestGridPriceService_CalculateV2GCompensation(t *testing.T) {
	logger := zap.NewNop()
	config := &GridPriceConfig{
		BasePriceOffPeak:   0.55,
		BasePricePeak:      0.85,
		BasePriceSuperPeak: 1.20,
		PeakStartHour:      17,
		PeakEndHour:        21,
		SuperPeakStartHour: 18,
		SuperPeakEndHour:   20,
		WeekendMultiplier:  0.8,
	}
	service := NewGridPriceService(logger, config)
	ctx := context.Background()

	energyKWh := 20.0
	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now()

	compensation, err := service.CalculateV2GCompensation(ctx, energyKWh, startTime, endTime)
	if err != nil {
		t.Fatalf("CalculateV2GCompensation failed: %v", err)
	}

	if compensation <= 0 {
		t.Errorf("Expected positive compensation, got %f", compensation)
	}

	t.Logf("Compensation for %.2f kWh: R$ %.2f", energyKWh, compensation)
}

func TestGridPriceService_PeakHourPricing(t *testing.T) {
	logger := zap.NewNop()
	config := &GridPriceConfig{
		BasePriceOffPeak:   0.55,
		BasePricePeak:      0.85,
		BasePriceSuperPeak: 1.20,
		PeakStartHour:      17,
		PeakEndHour:        21,
		SuperPeakStartHour: 18,
		SuperPeakEndHour:   20,
		WeekendMultiplier:  0.8,
	}
	service := NewGridPriceService(logger, config)

	// 2026-02-02 is a Monday, so we offset from there
	// Monday=0 offset, Tuesday=+1, ..., Saturday=+5, Sunday=+6
	monday := time.Date(2026, 2, 2, 0, 0, 0, 0, time.Local)

	tests := []struct {
		dayOffset int
		hour      int
		isPeak    bool
		desc      string
	}{
		{0, 3, false, "Monday 3AM (off-peak)"},
		{0, 10, false, "Monday 10AM (regular)"},
		{0, 19, true, "Monday 19PM (peak)"},
		{5, 15, false, "Saturday 15PM (weekend)"},
		{6, 19, false, "Sunday 19PM (weekend, not peak)"},
	}

	for _, tt := range tests {
		testTime := monday.AddDate(0, 0, tt.dayOffset)
		testTime = time.Date(testTime.Year(), testTime.Month(), testTime.Day(), tt.hour, 0, 0, 0, time.Local)

		isPeak := service.isPeakHour(testTime)
		if isPeak != tt.isPeak {
			t.Errorf("%s: isPeak=%v, expected %v (weekday=%s)",
				tt.desc, isPeak, tt.isPeak, testTime.Weekday())
		}
	}
}

func TestGridPriceService_BrazilianTariffStructure(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(logger, nil)

	peakTime := time.Date(2026, 2, 2, 19, 0, 0, 0, time.Local)
	offPeakTime := time.Date(2026, 2, 2, 3, 0, 0, 0, time.Local)

	peakPrice := service.getPriceAtTime(peakTime)
	offPeakPrice := service.getPriceAtTime(offPeakTime)

	if peakPrice <= offPeakPrice {
		t.Errorf("Peak price (%f) should be higher than off-peak (%f)", peakPrice, offPeakPrice)
	}

	ratio := peakPrice / offPeakPrice
	if ratio < 1.5 {
		t.Errorf("Peak/off-peak ratio (%f) should be at least 1.5", ratio)
	}
}

func TestGridPriceService_ForecastConsistency(t *testing.T) {
	logger := zap.NewNop()
	service := NewGridPriceService(logger, nil)
	ctx := context.Background()

	forecast1, _ := service.GetPriceForecast(ctx, 24)
	forecast2, _ := service.GetPriceForecast(ctx, 24)

	diff := forecast1[0].Price - forecast2[0].Price
	if diff < -0.01 || diff > 0.01 {
		t.Errorf("Sequential forecasts have inconsistent current price: %f vs %f",
			forecast1[0].Price, forecast2[0].Price)
	}
}
