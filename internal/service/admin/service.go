package admin

import (
	"context"
	"fmt"
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

	// Get user counts (simplified - in production use aggregation queries)
	// These would be actual database queries
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

	// Get today's data
	today := time.Now().Truncate(24 * time.Hour)

	// Count active transactions
	// This would be a database query in production
	stats.ActiveTransactions = 0
	stats.TodayTransactions = 0
	stats.TodayRevenue = 0
	stats.TodayEnergyKWh = 0

	// Get pending reservations count
	stats.PendingReservations = 0

	// Get active alerts count
	if s.alertRepo != nil {
		count, err := s.alertRepo.CountUnacknowledged(ctx)
		if err == nil {
			stats.ActiveAlerts = count
		}
	}

	_ = today // Used in production queries

	return stats, nil
}

// GetRevenueStats returns revenue statistics
func (s *Service) GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (*ports.RevenueStats, error) {
	stats := &ports.RevenueStats{
		RevenueByDay:    make(map[string]float64),
		RevenueByMethod: make(map[string]float64),
	}

	// In production, these would be database aggregation queries
	// For now, return empty stats

	return stats, nil
}

// GetUsageStats returns usage statistics
func (s *Service) GetUsageStats(ctx context.Context, startDate, endDate time.Time) (*ports.UsageStats, error) {
	stats := &ports.UsageStats{
		SessionsByDay: make(map[string]int),
		EnergyByDay:   make(map[string]float64),
		TopStations:   make([]ports.StationUsage, 0),
	}

	// In production, these would be database aggregation queries

	return stats, nil
}

// GetUsers returns paginated users
func (s *Service) GetUsers(ctx context.Context, filter ports.UserFilter, limit, offset int) ([]domain.User, int, error) {
	// In production, this would use the filter and pagination
	// For now, return empty list
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
			details.TotalEnergyKWh += tx.MeterStop - tx.MeterStart
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

	details := &ports.StationDetails{
		Station:           station,
		Connectors:        station.Connectors,
		TodayTransactions: 0,
		TodayRevenue:      0,
		TodayEnergyKWh:    0,
		Uptime:            99.9, // Placeholder
		LastHeartbeat:     station.LastHeartbeat,
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
	// In production, this would use the filter and pagination
	var transactions []domain.Transaction

	if filter.UserID != "" {
		txs, err := s.txRepo.FindHistoryByUserID(ctx, filter.UserID)
		if err != nil {
			return nil, 0, err
		}
		transactions = txs
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

// GenerateReport generates a report
func (s *Service) GenerateReport(ctx context.Context, reportType string, startDate, endDate time.Time) ([]byte, error) {
	// In production, this would generate actual reports (PDF, CSV, etc.)
	// For now, return a placeholder

	switch reportType {
	case "revenue":
		return []byte("Revenue report placeholder"), nil
	case "usage":
		return []byte("Usage report placeholder"), nil
	case "users":
		return []byte("Users report placeholder"), nil
	case "stations":
		return []byte("Stations report placeholder"), nil
	default:
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}
}
