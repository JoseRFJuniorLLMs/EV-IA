package analytics

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

// TrendData represents demand trend information
type TrendData struct {
	DailyAverageKWh     float64            `json:"daily_average_kwh"`
	DailyAverageRevenue float64            `json:"daily_average_revenue"`
	GrowthRate          float64            `json:"growth_rate_percent"`
	PeakHour            int                `json:"peak_hour"`
	DailyTrends         map[string]float64 `json:"daily_trends"`
}

// PredictionService provides demand forecasting using historical data
type PredictionService struct {
	txRepo ports.TransactionRepository
	log    *zap.Logger
}

// NewPredictionService creates a new prediction service
func NewPredictionService(txRepo ports.TransactionRepository, log *zap.Logger) *PredictionService {
	return &PredictionService{txRepo: txRepo, log: log}
}

// PredictDemand predicts energy demand for a given location and time
func (s *PredictionService) PredictDemand(ctx context.Context, location string, timestamp time.Time) (float64, error) {
	ea := &EnergyAnalytics{repo: s.txRepo}
	return ea.PredictDemand(ctx, location, timestamp)
}

// PredictRevenue predicts revenue for a location over a given period
func (s *PredictionService) PredictRevenue(ctx context.Context, location string, days int) (float64, error) {
	var totalPredicted float64

	for d := 0; d < days; d++ {
		date := time.Now().AddDate(0, 0, d)
		for hour := 0; hour < 24; hour++ {
			ts := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, date.Location())
			demand, err := s.PredictDemand(ctx, location, ts)
			if err != nil {
				continue
			}
			totalPredicted += demand * 0.75 // R$ 0.75/kWh average
		}
	}

	return totalPredicted, nil
}

// GetTrends analyzes historical trends for a location
func (s *PredictionService) GetTrends(ctx context.Context, location string, days int) (*TrendData, error) {
	trends := &TrendData{
		DailyTrends: make(map[string]float64),
	}

	var totalEnergy, totalRevenue float64
	hourCounts := make(map[int]float64)

	for d := 0; d < days; d++ {
		date := time.Now().AddDate(0, 0, -d)
		txs, err := s.txRepo.FindByDate(ctx, date)
		if err != nil {
			continue
		}

		var dailyEnergy float64
		for _, tx := range txs {
			energy := float64(tx.TotalEnergy) / 1000.0
			dailyEnergy += energy
			totalRevenue += tx.Cost
			hourCounts[tx.StartTime.Hour()] += energy
		}

		totalEnergy += dailyEnergy
		trends.DailyTrends[date.Format("2006-01-02")] = dailyEnergy
	}

	if days > 0 {
		trends.DailyAverageKWh = totalEnergy / float64(days)
		trends.DailyAverageRevenue = totalRevenue / float64(days)
	}

	// Find peak hour
	var maxEnergy float64
	for hour, energy := range hourCounts {
		if energy > maxEnergy {
			maxEnergy = energy
			trends.PeakHour = hour
		}
	}

	// Growth rate: compare first half vs second half of the period
	if days >= 2 {
		var firstHalf, secondHalf float64
		mid := days / 2
		for d := 0; d < days; d++ {
			date := time.Now().AddDate(0, 0, -d)
			dateStr := date.Format("2006-01-02")
			if d < mid {
				secondHalf += trends.DailyTrends[dateStr]
			} else {
				firstHalf += trends.DailyTrends[dateStr]
			}
		}
		if firstHalf > 0 {
			trends.GrowthRate = ((secondHalf - firstHalf) / firstHalf) * 100
		}
	}

	return trends, nil
}
