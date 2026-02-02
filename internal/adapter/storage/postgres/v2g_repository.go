package postgres

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

// V2GRepository implements V2G data persistence
type V2GRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewV2GRepository creates a new V2G repository
func NewV2GRepository(db *gorm.DB, log *zap.Logger) ports.V2GRepository {
	return &V2GRepository{
		db:  db,
		log: log,
	}
}

// --- Session Operations ---

// CreateSession creates a new V2G session
func (r *V2GRepository) CreateSession(ctx context.Context, session *domain.V2GSession) error {
	result := r.db.WithContext(ctx).Create(session)
	if result.Error != nil {
		r.log.Error("Failed to create V2G session",
			zap.String("sessionID", session.ID),
			zap.Error(result.Error),
		)
		return result.Error
	}

	r.log.Debug("Created V2G session",
		zap.String("sessionID", session.ID),
		zap.String("chargePointID", session.ChargePointID),
	)
	return nil
}

// UpdateSession updates an existing V2G session
func (r *V2GRepository) UpdateSession(ctx context.Context, session *domain.V2GSession) error {
	result := r.db.WithContext(ctx).Save(session)
	if result.Error != nil {
		r.log.Error("Failed to update V2G session",
			zap.String("sessionID", session.ID),
			zap.Error(result.Error),
		)
		return result.Error
	}
	return nil
}

// GetSession retrieves a V2G session by ID
func (r *V2GRepository) GetSession(ctx context.Context, sessionID string) (*domain.V2GSession, error) {
	var session domain.V2GSession
	result := r.db.WithContext(ctx).First(&session, "id = ?", sessionID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &session, nil
}

// GetSessionsByChargePoint retrieves V2G sessions for a charge point
func (r *V2GRepository) GetSessionsByChargePoint(ctx context.Context, chargePointID string, limit int) ([]domain.V2GSession, error) {
	var sessions []domain.V2GSession
	result := r.db.WithContext(ctx).
		Where("charge_point_id = ?", chargePointID).
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions)
	if result.Error != nil {
		return nil, result.Error
	}
	return sessions, nil
}

// GetSessionsByUser retrieves V2G sessions for a user
func (r *V2GRepository) GetSessionsByUser(ctx context.Context, userID string, limit int) ([]domain.V2GSession, error) {
	var sessions []domain.V2GSession
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions)
	if result.Error != nil {
		return nil, result.Error
	}
	return sessions, nil
}

// GetActiveSessions retrieves all active V2G sessions
func (r *V2GRepository) GetActiveSessions(ctx context.Context) ([]domain.V2GSession, error) {
	var sessions []domain.V2GSession
	result := r.db.WithContext(ctx).
		Where("status = ?", domain.V2GStatusActive).
		Find(&sessions)
	if result.Error != nil {
		return nil, result.Error
	}
	return sessions, nil
}

// --- Preferences Operations ---

// SavePreferences saves V2G preferences for a user
func (r *V2GRepository) SavePreferences(ctx context.Context, prefs *domain.V2GPreferences) error {
	// Use Upsert - create if not exists, update if exists
	result := r.db.WithContext(ctx).
		Where("user_id = ?", prefs.UserID).
		Assign(prefs).
		FirstOrCreate(prefs)
	if result.Error != nil {
		r.log.Error("Failed to save V2G preferences",
			zap.String("userID", prefs.UserID),
			zap.Error(result.Error),
		)
		return result.Error
	}
	return nil
}

// GetPreferences retrieves V2G preferences for a user
func (r *V2GRepository) GetPreferences(ctx context.Context, userID string) (*domain.V2GPreferences, error) {
	var prefs domain.V2GPreferences
	result := r.db.WithContext(ctx).First(&prefs, "user_id = ?", userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Return default preferences
			return &domain.V2GPreferences{
				UserID:          userID,
				AutoDischarge:   false,
				MinGridPrice:    0.80,
				MaxDischargeKWh: 50.0,
				PreserveSOC:     20,
				NotifyOnStart:   true,
				NotifyOnEnd:     true,
			}, nil
		}
		return nil, result.Error
	}
	return &prefs, nil
}

// --- Event Operations ---

// CreateEvent creates a V2G event
func (r *V2GRepository) CreateEvent(ctx context.Context, event *domain.V2GEvent) error {
	result := r.db.WithContext(ctx).Create(event)
	if result.Error != nil {
		r.log.Error("Failed to create V2G event",
			zap.String("eventID", event.ID),
			zap.Error(result.Error),
		)
		return result.Error
	}
	return nil
}

// GetEventsBySession retrieves events for a V2G session
func (r *V2GRepository) GetEventsBySession(ctx context.Context, sessionID string) ([]domain.V2GEvent, error) {
	var events []domain.V2GEvent
	result := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("timestamp ASC").
		Find(&events)
	if result.Error != nil {
		return nil, result.Error
	}
	return events, nil
}

// --- Statistics ---

// GetUserStats retrieves V2G statistics for a user
func (r *V2GRepository) GetUserStats(ctx context.Context, userID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	var stats domain.V2GStats
	stats.EntityID = userID
	stats.EntityType = "user"
	stats.StartDate = startDate
	stats.EndDate = endDate

	// Get total sessions
	var sessionCount int64
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startDate, endDate).
		Count(&sessionCount)
	stats.TotalSessions = int(sessionCount)

	// Get total energy discharged and compensation
	type AggregateResult struct {
		TotalEnergy       float64
		TotalCompensation float64
	}
	var agg AggregateResult
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Select("COALESCE(SUM(ABS(energy_transferred)), 0) as total_energy, COALESCE(SUM(user_compensation), 0) as total_compensation").
		Where("user_id = ? AND direction = ? AND created_at BETWEEN ? AND ?",
			userID, domain.V2GDirectionDischarging, startDate, endDate).
		Scan(&agg)

	stats.TotalEnergyDischargedKWh = agg.TotalEnergy
	stats.TotalCompensation = agg.TotalCompensation

	// Calculate average session duration
	type DurationResult struct {
		AvgDuration float64
	}
	var dur DurationResult
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(end_time, NOW()) - start_time))), 0) as avg_duration").
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startDate, endDate).
		Scan(&dur)
	stats.AverageSessionDuration = time.Duration(dur.AvgDuration) * time.Second

	// Calculate peak hours participation
	var peakSessions int64
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Where("user_id = ? AND created_at BETWEEN ? AND ? AND EXTRACT(HOUR FROM start_time) BETWEEN 17 AND 21",
			userID, startDate, endDate).
		Count(&peakSessions)
	if stats.TotalSessions > 0 {
		stats.PeakHoursParticipation = float64(peakSessions) / float64(stats.TotalSessions) * 100
	}

	return &stats, nil
}

// GetChargePointStats retrieves V2G statistics for a charge point
func (r *V2GRepository) GetChargePointStats(ctx context.Context, chargePointID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	var stats domain.V2GStats
	stats.EntityID = chargePointID
	stats.EntityType = "charge_point"
	stats.StartDate = startDate
	stats.EndDate = endDate

	// Get total sessions
	var sessionCount int64
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Where("charge_point_id = ? AND created_at BETWEEN ? AND ?", chargePointID, startDate, endDate).
		Count(&sessionCount)
	stats.TotalSessions = int(sessionCount)

	// Get total energy discharged and compensation
	type AggregateResult struct {
		TotalEnergy       float64
		TotalCompensation float64
	}
	var agg AggregateResult
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Select("COALESCE(SUM(ABS(energy_transferred)), 0) as total_energy, COALESCE(SUM(user_compensation), 0) as total_compensation").
		Where("charge_point_id = ? AND direction = ? AND created_at BETWEEN ? AND ?",
			chargePointID, domain.V2GDirectionDischarging, startDate, endDate).
		Scan(&agg)

	stats.TotalEnergyDischargedKWh = agg.TotalEnergy
	stats.TotalCompensation = agg.TotalCompensation

	// Calculate average session duration
	type DurationResult struct {
		AvgDuration float64
	}
	var dur DurationResult
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(end_time, NOW()) - start_time))), 0) as avg_duration").
		Where("charge_point_id = ? AND created_at BETWEEN ? AND ?", chargePointID, startDate, endDate).
		Scan(&dur)
	stats.AverageSessionDuration = time.Duration(dur.AvgDuration) * time.Second

	// Calculate peak hours participation
	var peakSessions int64
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Where("charge_point_id = ? AND created_at BETWEEN ? AND ? AND EXTRACT(HOUR FROM start_time) BETWEEN 17 AND 21",
			chargePointID, startDate, endDate).
		Count(&peakSessions)
	if stats.TotalSessions > 0 {
		stats.PeakHoursParticipation = float64(peakSessions) / float64(stats.TotalSessions) * 100
	}

	return &stats, nil
}

// GetGlobalStats retrieves global V2G statistics
func (r *V2GRepository) GetGlobalStats(ctx context.Context, startDate, endDate time.Time) (*domain.V2GStats, error) {
	var stats domain.V2GStats
	stats.EntityID = "global"
	stats.EntityType = "system"
	stats.StartDate = startDate
	stats.EndDate = endDate

	// Get total sessions
	var sessionCount int64
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(&sessionCount)
	stats.TotalSessions = int(sessionCount)

	// Get total energy and compensation
	type AggregateResult struct {
		TotalEnergy       float64
		TotalCompensation float64
	}
	var agg AggregateResult
	r.db.WithContext(ctx).Model(&domain.V2GSession{}).
		Select("COALESCE(SUM(ABS(energy_transferred)), 0) as total_energy, COALESCE(SUM(user_compensation), 0) as total_compensation").
		Where("direction = ? AND created_at BETWEEN ? AND ?",
			domain.V2GDirectionDischarging, startDate, endDate).
		Scan(&agg)

	stats.TotalEnergyDischargedKWh = agg.TotalEnergy
	stats.TotalCompensation = agg.TotalCompensation

	return &stats, nil
}

// --- Compensation Operations ---

// GetPendingCompensations retrieves sessions with pending compensation
func (r *V2GRepository) GetPendingCompensations(ctx context.Context) ([]domain.V2GSession, error) {
	var sessions []domain.V2GSession
	result := r.db.WithContext(ctx).
		Where("status = ? AND user_compensation > 0", domain.V2GStatusCompleted).
		Find(&sessions)
	if result.Error != nil {
		return nil, result.Error
	}
	return sessions, nil
}

// MarkCompensationPaid marks a session's compensation as paid
func (r *V2GRepository) MarkCompensationPaid(ctx context.Context, sessionID string, paymentID string) error {
	// This would typically update a paid_at field or create a payment record
	// For now, we just log it
	r.log.Info("Compensation marked as paid",
		zap.String("sessionID", sessionID),
		zap.String("paymentID", paymentID),
	)
	return nil
}
