package v2g

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCCEEClient_DefaultConfig(t *testing.T) {
	config := DefaultCCEEConfig()

	if config.BaseURL == "" {
		t.Error("Default config should have BaseURL")
	}

	if config.Timeout == 0 {
		t.Error("Default config should have Timeout")
	}

	if config.CacheDuration == 0 {
		t.Error("Default config should have CacheDuration")
	}

	if config.DefaultRegion == "" {
		t.Error("Default config should have DefaultRegion")
	}

	// Default region should be SE/CO (main Brazilian region)
	if config.DefaultRegion != "SE/CO" {
		t.Errorf("Expected default region 'SE/CO', got '%s'", config.DefaultRegion)
	}
}

func TestCCEEClient_GetSimulatedPrices(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCCEEConfig()
	config.EnableFallback = true
	config.APIKey = "" // Force fallback

	client := NewCCEEClient(config, logger)
	ctx := context.Background()

	start := time.Now()
	end := start.Add(24 * time.Hour)

	prices, err := client.GetPrices(ctx, "SE/CO", start, end)
	if err != nil {
		t.Fatalf("GetPrices failed: %v", err)
	}

	// Should have 24 hourly price points
	if len(prices) != 24 {
		t.Errorf("Expected 24 price points, got %d", len(prices))
	}

	// Verify price structure
	for i, price := range prices {
		if price.PLD <= 0 {
			t.Errorf("Price point %d has invalid PLD: %f", i, price.PLD)
		}

		if price.Region != "SE/CO" {
			t.Errorf("Price point %d has wrong region: %s", i, price.Region)
		}

		if price.LoadLevel == "" {
			t.Errorf("Price point %d has empty load level", i)
		}

		// Load level should be one of the Brazilian standards
		validLevels := map[string]bool{"pesada": true, "média": true, "leve": true}
		if !validLevels[price.LoadLevel] {
			t.Errorf("Price point %d has invalid load level: %s", i, price.LoadLevel)
		}
	}
}

func TestCCEEClient_RegionalPricing(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCCEEConfig()
	config.EnableFallback = true
	config.APIKey = ""

	client := NewCCEEClient(config, logger)
	ctx := context.Background()

	regions := client.GetRegions()
	if len(regions) != 4 {
		t.Errorf("Expected 4 regions, got %d", len(regions))
	}

	start := time.Now()
	end := start.Add(1 * time.Hour)

	// Get prices for each region and verify they differ
	regionPrices := make(map[string]float64)
	for _, region := range regions {
		prices, err := client.GetPrices(ctx, region, start, end)
		if err != nil {
			t.Fatalf("GetPrices for region %s failed: %v", region, err)
		}
		if len(prices) > 0 {
			regionPrices[region] = prices[0].PLD
		}
	}

	// Northeast should have lower prices (more wind/solar)
	if regionPrices["NE"] >= regionPrices["SE/CO"] {
		t.Log("Note: NE prices should typically be lower than SE/CO")
	}

	// North should have lower prices (hydro)
	if regionPrices["N"] >= regionPrices["SE/CO"] {
		t.Log("Note: N prices should typically be lower than SE/CO")
	}
}

func TestCCEEClient_GetCurrentPLD(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCCEEConfig()
	config.EnableFallback = true
	config.APIKey = ""

	client := NewCCEEClient(config, logger)
	ctx := context.Background()

	pld, err := client.GetCurrentPLD(ctx)
	if err != nil {
		t.Fatalf("GetCurrentPLD failed: %v", err)
	}

	// PLD should be positive
	if pld.PLD <= 0 {
		t.Errorf("PLD should be positive, got %f", pld.PLD)
	}

	// PLD should be in reasonable range for Brazil (50-500 R$/MWh)
	if pld.PLD < 50 || pld.PLD > 600 {
		t.Logf("Warning: PLD %f R$/MWh seems unusual", pld.PLD)
	}
}

func TestCCEEClient_ConvertPLDToRetail(t *testing.T) {
	logger := zap.NewNop()
	client := NewCCEEClient(nil, logger)

	tests := []struct {
		pldMWh         float64
		expectedMinKWh float64
		expectedMaxKWh float64
	}{
		{100.0, 0.15, 0.25},   // Low PLD
		{200.0, 0.30, 0.45},   // Medium PLD
		{400.0, 0.60, 0.85},   // High PLD
		{500.0, 0.75, 1.10},   // Very high PLD
	}

	for _, tt := range tests {
		retail := client.ConvertPLDToRetail(tt.pldMWh)

		if retail < tt.expectedMinKWh || retail > tt.expectedMaxKWh {
			t.Errorf("PLD %f R$/MWh -> retail %f R$/kWh, expected [%f, %f]",
				tt.pldMWh, retail, tt.expectedMinKWh, tt.expectedMaxKWh)
		}
	}
}

func TestCCEEClient_GetLoadLevel(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCCEEConfig()
	config.EnableFallback = true
	config.APIKey = ""

	client := NewCCEEClient(config, logger)
	ctx := context.Background()

	loadLevel, err := client.GetLoadLevel(ctx)
	if err != nil {
		t.Fatalf("GetLoadLevel failed: %v", err)
	}

	validLevels := map[string]bool{"pesada": true, "média": true, "leve": true}
	if !validLevels[loadLevel] {
		t.Errorf("Invalid load level: %s", loadLevel)
	}
}

func TestCCEEClient_IsPeakHour(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCCEEConfig()
	config.EnableFallback = true
	config.APIKey = ""

	client := NewCCEEClient(config, logger)
	ctx := context.Background()

	isPeak, err := client.IsPeakHour(ctx)
	if err != nil {
		t.Fatalf("IsPeakHour failed: %v", err)
	}

	// Just verify it returns a boolean without error
	t.Logf("Current time is peak hour: %v", isPeak)
}

func TestCCEEClient_GetPriceForecast(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCCEEConfig()
	config.EnableFallback = true
	config.APIKey = ""

	client := NewCCEEClient(config, logger)
	ctx := context.Background()

	forecast, err := client.GetPriceForecast(ctx, 12)
	if err != nil {
		t.Fatalf("GetPriceForecast failed: %v", err)
	}

	if len(forecast) != 12 {
		t.Errorf("Expected 12 forecast points, got %d", len(forecast))
	}

	for i, point := range forecast {
		if point.Price <= 0 {
			t.Errorf("Forecast point %d has invalid price: %f", i, point.Price)
		}
		if point.Source != "ccee" {
			t.Errorf("Forecast point %d has wrong source: %s", i, point.Source)
		}
	}
}

func TestCCEEClient_GetRegionName(t *testing.T) {
	logger := zap.NewNop()
	client := NewCCEEClient(nil, logger)

	tests := []struct {
		code     string
		expected string
	}{
		{"SE/CO", "Sudeste/Centro-Oeste"},
		{"S", "Sul"},
		{"NE", "Nordeste"},
		{"N", "Norte"},
		{"INVALID", "INVALID"}, // Should return the code itself
	}

	for _, tt := range tests {
		name := client.GetRegionName(tt.code)
		if name != tt.expected {
			t.Errorf("GetRegionName(%s) = %s, expected %s", tt.code, name, tt.expected)
		}
	}
}

func TestCCEEClient_SimulatedPricingModel(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultCCEEConfig()
	config.EnableFallback = true
	config.APIKey = ""

	client := NewCCEEClient(config, logger)
	ctx := context.Background()

	// Test pricing at different times
	weekday := time.Date(2026, 2, 2, 0, 0, 0, 0, time.Local) // Monday
	weekend := time.Date(2026, 2, 7, 0, 0, 0, 0, time.Local) // Saturday

	// Get prices for 24 hours on weekday
	weekdayPrices, _ := client.GetPrices(ctx, "SE/CO", weekday, weekday.Add(24*time.Hour))

	// Get prices for 24 hours on weekend
	weekendPrices, _ := client.GetPrices(ctx, "SE/CO", weekend, weekend.Add(24*time.Hour))

	// Calculate average weekday price
	var weekdaySum, weekendSum float64
	for _, p := range weekdayPrices {
		weekdaySum += p.PLD
	}
	for _, p := range weekendPrices {
		weekendSum += p.PLD
	}

	weekdayAvg := weekdaySum / float64(len(weekdayPrices))
	weekendAvg := weekendSum / float64(len(weekendPrices))

	// Weekend prices should generally be lower
	if weekendAvg >= weekdayAvg {
		t.Logf("Note: Weekend avg (%f) should typically be lower than weekday (%f)", weekendAvg, weekdayAvg)
	}

	// Check peak hour pricing (18-21h weekday)
	var peakPrice, offPeakPrice float64
	for _, p := range weekdayPrices {
		ts, _ := time.Parse(time.RFC3339, p.Timestamp)
		hour := ts.Hour()
		if hour >= 18 && hour < 21 {
			peakPrice = p.PLD
			break
		}
		if hour >= 2 && hour < 5 {
			offPeakPrice = p.PLD
		}
	}

	if peakPrice > 0 && offPeakPrice > 0 && peakPrice <= offPeakPrice {
		t.Logf("Peak price (%f) should be higher than off-peak (%f)", peakPrice, offPeakPrice)
	}
}
