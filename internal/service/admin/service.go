package admin

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// Service implements AdminService
type Service struct {
	userRepo        ports.UserRepository
	deviceRepo      ports.ChargePointRepository
	txRepo          ports.TransactionRepository
	paymentRepo     ports.PaymentRepository
	reservationRepo ports.ReservationRepository
	alertRepo       ports.AlertRepository
	log             *zap.Logger
}

// NewService creates a new admin service
func NewService(
	userRepo ports.UserRepository,
	deviceRepo ports.ChargePointRepository,
	txRepo ports.TransactionRepository,
	paymentRepo ports.PaymentRepository,
	reservationRepo ports.ReservationRepository,
	alertRepo ports.AlertRepository,
	log *zap.Logger,
) *Service {
	return &Service{
		userRepo:        userRepo,
		deviceRepo:      deviceRepo,
		txRepo:          txRepo,
		paymentRepo:     paymentRepo,
		reservationRepo: reservationRepo,
		alertRepo:       alertRepo,
		log:             log,
	}
}

// GetDashboardStats returns dashboard statistics
func (s *Service) GetDashboardStats(ctx context.Context) (*ports.DashboardStats, error) {
	stats := &ports.DashboardStats{}

	// TODO: UserRepository does not expose FindAll/Count methods.
	// To get TotalUsers/ActiveUsers, add a CountAll method to ports.UserRepository.
	stats.TotalUsers = 0
	stats.ActiveUsers = 0

	// Get station counts
	stations, err := s.deviceRepo.FindAll(ctx, nil)
	if err == nil {
		stats.TotalStations = len(stations)
		for _, station := range stations {
			if station.Status == domain.ChargePointStatusAvailable ||
				station.Status == domain.ChargePointStatusOccupied {
				stats.OnlineStations++
			}
		}
	}

	// Get today's transactions from the database
	today := time.Now().Truncate(24 * time.Hour)
	todayTxs, err := s.txRepo.FindByDate(ctx, today)
	if err != nil {
		s.log.Warn("Failed to fetch today's transactions", zap.Error(err))
	} else {
		stats.TodayTransactions = len(todayTxs)
		for _, tx := range todayTxs {
			stats.TodayRevenue += tx.Cost
			stats.TodayEnergyKWh += float64(tx.MeterStop-tx.MeterStart) / 1000.0
			if tx.Status == domain.TransactionStatusStarted {
				stats.ActiveTransactions++
			}
		}
	}

	// Get pending reservations count
	if s.reservationRepo != nil {
		count, err := s.reservationRepo.CountByUserAndStatus(ctx, "", []domain.ReservationStatus{domain.ReservationStatusPending})
		if err != nil {
			s.log.Warn("Failed to count pending reservations", zap.Error(err))
		} else {
			stats.PendingReservations = count
		}
	}

	// Get active alerts count
	if s.alertRepo != nil {
		count, err := s.alertRepo.CountUnacknowledged(ctx)
		if err == nil {
			stats.ActiveAlerts = count
		}
	}

	return stats, nil
}

// GetRevenueStats returns revenue statistics
func (s *Service) GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (*ports.RevenueStats, error) {
	stats := &ports.RevenueStats{
		RevenueByDay:    make(map[string]float64),
		RevenueByMethod: make(map[string]float64),
	}

	var totalTxCount int

	// Iterate each day in the range and aggregate revenue
	for d := startDate.Truncate(24 * time.Hour); !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayTxs, err := s.txRepo.FindByDate(ctx, d)
		if err != nil {
			s.log.Warn("Failed to fetch transactions for date", zap.Time("date", d), zap.Error(err))
			continue
		}

		dayKey := d.Format("2006-01-02")
		var dayRevenue float64
		for _, tx := range dayTxs {
			dayRevenue += tx.Cost
			// Aggregate by currency as a proxy for payment method
			if tx.Currency != "" {
				stats.RevenueByMethod[tx.Currency] += tx.Cost
			}
		}
		stats.RevenueByDay[dayKey] = dayRevenue
		stats.TotalRevenue += dayRevenue
		totalTxCount += len(dayTxs)
	}

	if totalTxCount > 0 {
		stats.AveragePerTx = stats.TotalRevenue / float64(totalTxCount)
	}

	return stats, nil
}

// GetUsageStats returns usage statistics
func (s *Service) GetUsageStats(ctx context.Context, startDate, endDate time.Time) (*ports.UsageStats, error) {
	stats := &ports.UsageStats{
		SessionsByDay: make(map[string]int),
		EnergyByDay:   make(map[string]float64),
		TopStations:   make([]ports.StationUsage, 0),
	}

	hourCounts := make(map[int]int)                // hour -> session count for peak hour calc
	stationMap := make(map[string]*ports.StationUsage) // stationID -> aggregated usage
	var totalDurationMin float64

	// Iterate each day in the range
	for d := startDate.Truncate(24 * time.Hour); !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayTxs, err := s.txRepo.FindByDate(ctx, d)
		if err != nil {
			s.log.Warn("Failed to fetch transactions for date", zap.Time("date", d), zap.Error(err))
			continue
		}

		dayKey := d.Format("2006-01-02")
		var dayEnergy float64

		for _, tx := range dayTxs {
			energyKWh := float64(tx.MeterStop-tx.MeterStart) / 1000.0
			dayEnergy += energyKWh

			// Track peak hour from transaction start times
			hourCounts[tx.StartTime.Hour()]++

			// Calculate session duration
			if tx.EndTime != nil {
				totalDurationMin += tx.EndTime.Sub(tx.StartTime).Minutes()
			}

			// Aggregate per station
			su, ok := stationMap[tx.ChargePointID]
			if !ok {
				su = &ports.StationUsage{StationID: tx.ChargePointID}
				stationMap[tx.ChargePointID] = su
			}
			su.Sessions++
			su.EnergyKWh += energyKWh
			su.Revenue += tx.Cost
		}

		stats.SessionsByDay[dayKey] = len(dayTxs)
		stats.EnergyByDay[dayKey] = dayEnergy
		stats.TotalSessions += len(dayTxs)
		stats.TotalEnergyKWh += dayEnergy
	}

	// Calculate average session duration
	if stats.TotalSessions > 0 {
		stats.AverageSessionMin = totalDurationMin / float64(stats.TotalSessions)
	}

	// Determine peak hour
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxCount = count
			stats.PeakHour = hour
		}
	}

	// Build top stations list (sorted by sessions descending)
	for _, su := range stationMap {
		stats.TopStations = append(stats.TopStations, *su)
	}
	// Simple insertion sort for top stations by sessions descending
	for i := 1; i < len(stats.TopStations); i++ {
		for j := i; j > 0 && stats.TopStations[j].Sessions > stats.TopStations[j-1].Sessions; j-- {
			stats.TopStations[j], stats.TopStations[j-1] = stats.TopStations[j-1], stats.TopStations[j]
		}
	}
	// Limit to top 10
	if len(stats.TopStations) > 10 {
		stats.TopStations = stats.TopStations[:10]
	}

	return stats, nil
}

// GetUsers returns paginated users
func (s *Service) GetUsers(ctx context.Context, filter ports.UserFilter, limit, offset int) ([]domain.User, int, error) {
	// TODO: UserRepository only exposes FindByID and FindByEmail.
	// To support listing/filtering users, add a FindAll(ctx, filter, limit, offset) method
	// to ports.UserRepository and implement it in the PostgreSQL adapter.
	// Until then, this endpoint cannot return user lists.
	s.log.Warn("GetUsers called but UserRepository.FindAll is not implemented")
	return []domain.User{}, 0, nil
}

// GetUserDetails returns detailed user information
func (s *Service) GetUserDetails(ctx context.Context, userID string) (*ports.UserDetails, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	details := &ports.UserDetails{
		User:              user,
		TotalTransactions: 0,
		TotalSpent:        0,
		TotalEnergyKWh:    0,
	}

	// Get transaction history
	history, err := s.txRepo.FindHistoryByUserID(ctx, userID)
	if err == nil {
		details.TotalTransactions = len(history)
		for _, tx := range history {
			details.TotalEnergyKWh += float64(tx.MeterStop-tx.MeterStart) / 1000.0
			details.TotalSpent += tx.Cost
		}

		// Get recent transactions (last 5)
		if len(history) > 5 {
			details.RecentTransactions = history[:5]
		} else {
			details.RecentTransactions = history
		}

		// Get last activity
		if len(history) > 0 {
			if history[0].EndTime != nil {
				details.LastActivity = history[0].EndTime
			} else {
				details.LastActivity = &history[0].StartTime
			}
		}
	}

	return details, nil
}

// UpdateUserStatus updates a user's status
func (s *Service) UpdateUserStatus(ctx context.Context, userID string, status string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	user.Status = status
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Save(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.log.Info("User status updated",
		zap.String("user_id", userID),
		zap.String("status", status),
	)

	return nil
}

// UpdateUserRole updates a user's role
func (s *Service) UpdateUserRole(ctx context.Context, userID string, role domain.UserRole) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	user.Role = role
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Save(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.log.Info("User role updated",
		zap.String("user_id", userID),
		zap.String("role", string(role)),
	)

	return nil
}

// GetStations returns paginated stations
func (s *Service) GetStations(ctx context.Context, filter ports.StationFilter, limit, offset int) ([]domain.ChargePoint, int, error) {
	filterMap := make(map[string]interface{})
	if filter.Status != "" {
		filterMap["status"] = filter.Status
	}
	if filter.Vendor != "" {
		filterMap["vendor"] = filter.Vendor
	}

	stations, err := s.deviceRepo.FindAll(ctx, filterMap)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get stations: %w", err)
	}

	total := len(stations)

	// Apply pagination
	if offset >= len(stations) {
		return []domain.ChargePoint{}, total, nil
	}

	end := offset + limit
	if end > len(stations) {
		end = len(stations)
	}

	return stations[offset:end], total, nil
}

// GetStationDetails returns detailed station information
func (s *Service) GetStationDetails(ctx context.Context, stationID string) (*ports.StationDetails, error) {
	station, err := s.deviceRepo.FindByID(ctx, stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find station: %w", err)
	}
	if station == nil {
		return nil, fmt.Errorf("station not found")
	}

	lastSeen := station.LastHeartbeat
	details := &ports.StationDetails{
		Station:       station,
		Connectors:    station.Connectors,
		LastHeartbeat: &lastSeen,
	}

	// Get today's transactions for this station
	today := time.Now().Truncate(24 * time.Hour)
	todayTxs, err := s.txRepo.FindByDate(ctx, today)
	if err == nil {
		for _, tx := range todayTxs {
			if tx.ChargePointID == stationID {
				details.TodayTransactions++
				details.TodayRevenue += tx.Cost
				details.TodayEnergyKWh += float64(tx.MeterStop-tx.MeterStart) / 1000.0
			}
		}
	}

	// Uptime: if station was seen within last 5 min, consider it up
	if time.Since(station.LastHeartbeat) < 5*time.Minute {
		details.Uptime = 100.0
	} else if station.Status == domain.ChargePointStatusAvailable || station.Status == domain.ChargePointStatusOccupied {
		details.Uptime = 95.0
	}

	return details, nil
}

// UpdateStationStatus updates a station's status
func (s *Service) UpdateStationStatus(ctx context.Context, stationID string, status domain.ChargePointStatus) error {
	if err := s.deviceRepo.UpdateStatus(ctx, stationID, status); err != nil {
		return fmt.Errorf("failed to update station status: %w", err)
	}

	s.log.Info("Station status updated",
		zap.String("station_id", stationID),
		zap.String("status", string(status)),
	)

	return nil
}

// GetTransactions returns paginated transactions
func (s *Service) GetTransactions(ctx context.Context, filter ports.TransactionFilter, limit, offset int) ([]domain.Transaction, int, error) {
	var transactions []domain.Transaction

	if filter.UserID != "" {
		txs, err := s.txRepo.FindHistoryByUserID(ctx, filter.UserID)
		if err != nil {
			return nil, 0, err
		}
		transactions = txs
	} else if !filter.StartDate.IsZero() {
		// Iterate each day in date range
		end := filter.EndDate
		if end.IsZero() {
			end = time.Now()
		}
		for d := filter.StartDate.Truncate(24 * time.Hour); !d.After(end); d = d.AddDate(0, 0, 1) {
			dayTxs, err := s.txRepo.FindByDate(ctx, d)
			if err != nil {
				continue
			}
			transactions = append(transactions, dayTxs...)
		}
	}

	// Filter by status if specified
	if filter.Status != "" {
		filtered := make([]domain.Transaction, 0, len(transactions))
		for _, tx := range transactions {
			if string(tx.Status) == filter.Status {
				filtered = append(filtered, tx)
			}
		}
		transactions = filtered
	}

	// Filter by charge point if specified
	if filter.ChargePointID != "" {
		filtered := make([]domain.Transaction, 0, len(transactions))
		for _, tx := range transactions {
			if tx.ChargePointID == filter.ChargePointID {
				filtered = append(filtered, tx)
			}
		}
		transactions = filtered
	}

	total := len(transactions)

	// Apply pagination
	if offset >= len(transactions) {
		return []domain.Transaction{}, total, nil
	}

	end := offset + limit
	if end > len(transactions) {
		end = len(transactions)
	}

	return transactions[offset:end], total, nil
}

// GetTransactionDetails returns detailed transaction information
func (s *Service) GetTransactionDetails(ctx context.Context, txID string) (*ports.TransactionDetails, error) {
	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}
	if tx == nil {
		return nil, fmt.Errorf("transaction not found")
	}

	details := &ports.TransactionDetails{
		Transaction: tx,
	}

	// Get user
	if tx.UserID != "" {
		user, _ := s.userRepo.FindByID(ctx, tx.UserID)
		details.User = user
	}

	// Get station
	if tx.ChargePointID != "" {
		station, _ := s.deviceRepo.FindByID(ctx, tx.ChargePointID)
		details.Station = station
	}

	return details, nil
}

// GetAlerts returns paginated alerts
func (s *Service) GetAlerts(ctx context.Context, limit, offset int) ([]ports.Alert, error) {
	if s.alertRepo == nil {
		return []ports.Alert{}, nil
	}

	alerts, err := s.alertRepo.GetAll(ctx, false, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}

	result := make([]ports.Alert, len(alerts))
	for i, a := range alerts {
		result[i] = ports.Alert{
			ID:           a.ID,
			Type:         a.Type,
			Severity:     a.Severity,
			Title:        a.Title,
			Message:      a.Message,
			Source:       a.Source,
			SourceID:     a.SourceID,
			Acknowledged: a.Acknowledged,
			CreatedAt:    a.CreatedAt,
		}
	}

	return result, nil
}

// AcknowledgeAlert acknowledges an alert
func (s *Service) AcknowledgeAlert(ctx context.Context, alertID string) error {
	if s.alertRepo == nil {
		return fmt.Errorf("alert repository not configured")
	}

	if err := s.alertRepo.Acknowledge(ctx, alertID); err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	s.log.Info("Alert acknowledged", zap.String("alert_id", alertID))

	return nil
}

// GenerateReport generates a CSV report
func (s *Service) GenerateReport(ctx context.Context, reportType string, startDate, endDate time.Time) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	switch reportType {
	case "revenue":
		w.Write([]string{"Date", "Transactions", "Revenue", "Energy_kWh"})
		for d := startDate.Truncate(24 * time.Hour); !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dayTxs, err := s.txRepo.FindByDate(ctx, d)
			if err != nil {
				continue
			}
			var revenue, energy float64
			for _, tx := range dayTxs {
				revenue += tx.Cost
				energy += float64(tx.MeterStop-tx.MeterStart) / 1000.0
			}
			w.Write([]string{
				d.Format("2006-01-02"),
				strconv.Itoa(len(dayTxs)),
				strconv.FormatFloat(revenue, 'f', 2, 64),
				strconv.FormatFloat(energy, 'f', 2, 64),
			})
		}

	case "usage":
		w.Write([]string{"Date", "Sessions", "Energy_kWh", "Avg_Duration_min"})
		for d := startDate.Truncate(24 * time.Hour); !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dayTxs, err := s.txRepo.FindByDate(ctx, d)
			if err != nil {
				continue
			}
			var energy, totalDur float64
			for _, tx := range dayTxs {
				energy += float64(tx.MeterStop-tx.MeterStart) / 1000.0
				if tx.EndTime != nil {
					totalDur += tx.EndTime.Sub(tx.StartTime).Minutes()
				}
			}
			avgDur := 0.0
			if len(dayTxs) > 0 {
				avgDur = totalDur / float64(len(dayTxs))
			}
			w.Write([]string{
				d.Format("2006-01-02"),
				strconv.Itoa(len(dayTxs)),
				strconv.FormatFloat(energy, 'f', 2, 64),
				strconv.FormatFloat(avgDur, 'f', 1, 64),
			})
		}

	case "stations":
		w.Write([]string{"StationID", "Vendor", "Model", "Status", "Location"})
		stations, err := s.deviceRepo.FindAll(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get stations: %w", err)
		}
		for _, st := range stations {
			addr := ""
			if st.Location != nil {
				addr = st.Location.Address
			}
			w.Write([]string{st.ID, st.Vendor, st.Model, string(st.Status), addr})
		}

	default:
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to write CSV: %w", err)
	}

	return buf.Bytes(), nil
}
