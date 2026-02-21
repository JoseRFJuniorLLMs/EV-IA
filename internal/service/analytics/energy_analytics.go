package analytics

import (
	"context"
	"time"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

type EnergyAnalytics struct {
	repo ports.TransactionRepository
}

func (ea *EnergyAnalytics) GenerateDailyReport(ctx context.Context, date time.Time) (*domain.DailyReport, error) {
	transactions, err := ea.repo.FindByDate(ctx, date)
	if err != nil {
		return nil, err
	}

	report := &domain.DailyReport{
		Date:               date,
		TotalEnergy:        0,
		TotalRevenue:       0,
		AverageSessionTime: 0,
		PeakHour:           0,
		DeviceUtilization:  make(map[string]float64),
	}

	for _, tx := range transactions {
		report.TotalEnergy += float64(tx.TotalEnergy) / 1000.0 // Convert Wh to kWh
		report.TotalRevenue += tx.Cost
		report.TotalTransactions++
	}

	return report, nil
}

// PredictDemand predicts energy demand using historical transaction data
// Uses a weighted moving average based on the same day-of-week and hour
func (ea *EnergyAnalytics) PredictDemand(ctx context.Context, location string, timestamp time.Time) (float64, error) {
	targetHour := timestamp.Hour()
	targetWeekday := timestamp.Weekday()

	var totalDemand float64
	var weightSum float64

	// Analyze last 30 days of data
	for daysBack := 1; daysBack <= 30; daysBack++ {
		date := timestamp.AddDate(0, 0, -daysBack)
		txs, err := ea.repo.FindByDate(ctx, date)
		if err != nil {
			continue
		}

		var hourlyEnergy float64
		for _, tx := range txs {
			if tx.StartTime.Hour() == targetHour {
				hourlyEnergy += float64(tx.TotalEnergy) / 1000.0 // Wh to kWh
			}
		}

		// Weight: same day-of-week gets 3x, recent days get higher weight
		weight := 1.0 / float64(daysBack) // recency weight
		if date.Weekday() == targetWeekday {
			weight *= 3.0
		}

		totalDemand += hourlyEnergy * weight
		weightSum += weight
	}

	if weightSum == 0 {
		return 0, nil
	}

	predictedKWh := totalDemand / weightSum
	return predictedKWh, nil
}
