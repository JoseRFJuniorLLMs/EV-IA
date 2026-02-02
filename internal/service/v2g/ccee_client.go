package v2g

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
)

// CCEEClient is a client for the CCEE (Câmara de Comercialização de Energia Elétrica) API
type CCEEClient struct {
	httpClient  *http.Client
	baseURL     string
	apiKey      string
	log         *zap.Logger
	cache       *priceCache
	config      *CCEEConfig
}

// CCEEConfig holds CCEE client configuration
type CCEEConfig struct {
	BaseURL         string
	APIKey          string
	Timeout         time.Duration
	CacheDuration   time.Duration
	DefaultRegion   string // SE/CO, S, NE, N
	EnableFallback  bool   // Use simulated prices if API fails
}

// DefaultCCEEConfig returns default CCEE configuration
func DefaultCCEEConfig() *CCEEConfig {
	return &CCEEConfig{
		BaseURL:        "https://api.ccee.org.br/v1",
		Timeout:        30 * time.Second,
		CacheDuration:  5 * time.Minute,
		DefaultRegion:  "SE/CO", // Southeast/Center-West (main region)
		EnableFallback: true,
	}
}

// priceCache caches CCEE prices
type priceCache struct {
	prices    []domain.GridPricePoint
	fetchedAt time.Time
	mu        sync.RWMutex
}

// CCEEPriceResponse represents CCEE API response
type CCEEPriceResponse struct {
	Data []CCEEPriceData `json:"data"`
}

// CCEEPriceData represents a single price point from CCEE
type CCEEPriceData struct {
	Timestamp string  `json:"timestamp"`      // ISO 8601
	PLD       float64 `json:"pld"`            // Preço de Liquidação das Diferenças (R$/MWh)
	Region    string  `json:"submercado"`     // SE/CO, S, NE, N
	LoadLevel string  `json:"patamar"`        // pesada, média, leve
}

// NewCCEEClient creates a new CCEE API client
func NewCCEEClient(config *CCEEConfig, log *zap.Logger) *CCEEClient {
	if config == nil {
		config = DefaultCCEEConfig()
	}

	return &CCEEClient{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		log:     log,
		cache:   &priceCache{},
		config:  config,
	}
}

// GetCurrentPLD gets the current PLD (Preço de Liquidação das Diferenças)
func (c *CCEEClient) GetCurrentPLD(ctx context.Context) (*CCEEPriceData, error) {
	prices, err := c.GetPrices(ctx, c.config.DefaultRegion, time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		return nil, err
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no price data available")
	}

	// Return most recent price
	return &prices[len(prices)-1], nil
}

// GetPrices gets PLD prices for a region and time period
func (c *CCEEClient) GetPrices(ctx context.Context, region string, start, end time.Time) ([]CCEEPriceData, error) {
	// Check cache first
	c.cache.mu.RLock()
	if c.cache.prices != nil && time.Since(c.cache.fetchedAt) < c.config.CacheDuration {
		c.cache.mu.RUnlock()
		// Convert cached domain prices to CCEE format
		result := make([]CCEEPriceData, len(c.cache.prices))
		for i, p := range c.cache.prices {
			result[i] = CCEEPriceData{
				Timestamp: p.Timestamp.Format(time.RFC3339),
				PLD:       p.Price * 1000, // Convert R$/kWh to R$/MWh
				Region:    region,
			}
		}
		return result, nil
	}
	c.cache.mu.RUnlock()

	// If no API key, use fallback
	if c.apiKey == "" {
		if c.config.EnableFallback {
			return c.getSimulatedPrices(region, start, end)
		}
		return nil, fmt.Errorf("CCEE API key not configured")
	}

	// Build request URL
	url := fmt.Sprintf("%s/pld?submercado=%s&data_inicio=%s&data_fim=%s",
		c.baseURL,
		region,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Warn("CCEE API request failed, using fallback",
			zap.Error(err),
		)
		if c.config.EnableFallback {
			return c.getSimulatedPrices(region, start, end)
		}
		return nil, fmt.Errorf("CCEE API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("CCEE API returned error, using fallback",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		if c.config.EnableFallback {
			return c.getSimulatedPrices(region, start, end)
		}
		return nil, fmt.Errorf("CCEE API error: status %d", resp.StatusCode)
	}

	var response CCEEPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Update cache
	c.updateCache(response.Data)

	return response.Data, nil
}

// getSimulatedPrices generates simulated prices based on Brazilian tariff structure
func (c *CCEEClient) getSimulatedPrices(region string, start, end time.Time) ([]CCEEPriceData, error) {
	var prices []CCEEPriceData

	current := start.Truncate(time.Hour)
	for current.Before(end) {
		hour := current.Hour()
		weekday := current.Weekday()

		// Base PLD in R$/MWh
		var pld float64
		var loadLevel string

		// Weekend - generally lower prices
		if weekday == time.Saturday || weekday == time.Sunday {
			pld = 150.0 // R$ 150/MWh base
			loadLevel = "leve"
		} else {
			// Weekday pricing
			switch {
			case hour >= 18 && hour < 21: // Peak hours (ponta)
				pld = 450.0 // R$ 450/MWh
				loadLevel = "pesada"
			case hour >= 17 && hour < 22: // Intermediate (intermediário)
				pld = 300.0 // R$ 300/MWh
				loadLevel = "média"
			case hour >= 0 && hour < 6: // Off-peak night
				pld = 100.0 // R$ 100/MWh
				loadLevel = "leve"
			default: // Regular hours
				pld = 200.0 // R$ 200/MWh
				loadLevel = "média"
			}
		}

		// Add some variation based on minute for realism
		variation := 1.0 + float64(current.Minute()%10-5)/100.0
		pld *= variation

		// Regional adjustments
		switch region {
		case "NE": // Northeast - usually lower due to wind/solar
			pld *= 0.85
		case "N": // North - usually lower due to hydro
			pld *= 0.80
		case "S": // South - moderate
			pld *= 0.95
		}

		prices = append(prices, CCEEPriceData{
			Timestamp: current.Format(time.RFC3339),
			PLD:       pld,
			Region:    region,
			LoadLevel: loadLevel,
		})

		current = current.Add(time.Hour)
	}

	c.log.Debug("Generated simulated CCEE prices",
		zap.String("region", region),
		zap.Int("count", len(prices)),
	)

	return prices, nil
}

// updateCache updates the price cache
func (c *CCEEClient) updateCache(data []CCEEPriceData) {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	prices := make([]domain.GridPricePoint, len(data))
	for i, d := range data {
		t, _ := time.Parse(time.RFC3339, d.Timestamp)
		prices[i] = domain.GridPricePoint{
			Timestamp: t,
			Price:     d.PLD / 1000, // Convert R$/MWh to R$/kWh
			IsPeak:    d.LoadLevel == "pesada",
			Source:    "ccee",
		}
	}

	c.cache.prices = prices
	c.cache.fetchedAt = time.Now()
}

// ConvertPLDToRetail converts CCEE wholesale PLD to retail price
func (c *CCEEClient) ConvertPLDToRetail(pldMWh float64) float64 {
	// Convert MWh to kWh
	priceKWh := pldMWh / 1000.0

	// Add typical distribution and tax components
	// ICMS: ~25-30%
	// PIS/COFINS: ~9%
	// Distribution: ~40% of total
	// Transmission: ~10%

	// Simplified: multiply by 1.8 to account for all components
	retailPrice := priceKWh * 1.8

	return retailPrice
}

// GetLoadLevel returns the current load level
func (c *CCEEClient) GetLoadLevel(ctx context.Context) (string, error) {
	price, err := c.GetCurrentPLD(ctx)
	if err != nil {
		return "", err
	}
	return price.LoadLevel, nil
}

// IsPeakHour checks if current time is during peak hours based on CCEE
func (c *CCEEClient) IsPeakHour(ctx context.Context) (bool, error) {
	loadLevel, err := c.GetLoadLevel(ctx)
	if err != nil {
		// Fallback to time-based check
		hour := time.Now().Hour()
		return hour >= 18 && hour < 21, nil
	}
	return loadLevel == "pesada", nil
}

// GetPriceForecast returns price forecast for next hours
func (c *CCEEClient) GetPriceForecast(ctx context.Context, hours int) ([]domain.GridPricePoint, error) {
	now := time.Now()
	end := now.Add(time.Duration(hours) * time.Hour)

	cceeData, err := c.GetPrices(ctx, c.config.DefaultRegion, now, end)
	if err != nil {
		return nil, err
	}

	forecast := make([]domain.GridPricePoint, len(cceeData))
	for i, d := range cceeData {
		t, _ := time.Parse(time.RFC3339, d.Timestamp)
		forecast[i] = domain.GridPricePoint{
			Timestamp: t,
			Price:     c.ConvertPLDToRetail(d.PLD), // Convert to retail price
			IsPeak:    d.LoadLevel == "pesada",
			Source:    "ccee",
		}
	}

	return forecast, nil
}

// GetRegions returns available CCEE regions
func (c *CCEEClient) GetRegions() []string {
	return []string{"SE/CO", "S", "NE", "N"}
}

// GetRegionName returns the full name of a region
func (c *CCEEClient) GetRegionName(code string) string {
	names := map[string]string{
		"SE/CO": "Sudeste/Centro-Oeste",
		"S":     "Sul",
		"NE":    "Nordeste",
		"N":     "Norte",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}
