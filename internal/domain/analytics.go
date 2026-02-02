package domain

import "time"

// DailyReport represents a daily analytics report
type DailyReport struct {
	Date               time.Time          `json:"date"`
	TotalEnergy        float64            `json:"total_energy"`        // kWh
	TotalRevenue       float64            `json:"total_revenue"`       // BRL
	TotalTransactions  int                `json:"total_transactions"`
	AverageSessionTime float64            `json:"average_session_time"` // minutes
	PeakHour           int                `json:"peak_hour"`            // 0-23
	DeviceUtilization  map[string]float64 `json:"device_utilization"`   // device_id -> percentage
}

// WeeklyReport represents a weekly analytics report
type WeeklyReport struct {
	StartDate       time.Time      `json:"start_date"`
	EndDate         time.Time      `json:"end_date"`
	DailyReports    []DailyReport  `json:"daily_reports"`
	TotalEnergy     float64        `json:"total_energy"`
	TotalRevenue    float64        `json:"total_revenue"`
	GrowthRate      float64        `json:"growth_rate"` // percentage vs previous week
}

// MonthlyReport represents a monthly analytics report
type MonthlyReport struct {
	Year            int            `json:"year"`
	Month           int            `json:"month"`
	WeeklyReports   []WeeklyReport `json:"weekly_reports"`
	TotalEnergy     float64        `json:"total_energy"`
	TotalRevenue    float64        `json:"total_revenue"`
	TopDevices      []string       `json:"top_devices"`
	PeakDays        []time.Time    `json:"peak_days"`
}
