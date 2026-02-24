// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package nietzsche

import (
	"context"
	"sort"
	"time"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"go.uber.org/zap"
)

type V2GRepository struct {
	db  *DB
	log *zap.Logger
}

func NewV2GRepository(db *DB, log *zap.Logger) *V2GRepository {
	return &V2GRepository{db: db, log: log}
}

// ── Sessions ─────────────────────────────────────────────────────────────

func (r *V2GRepository) CreateSession(ctx context.Context, session *domain.V2GSession) error {
	m, err := ToMap(session)
	if err != nil {
		return err
	}
	_, err = r.db.Insert(ctx, "v2g_sessions", m)
	return err
}

func (r *V2GRepository) UpdateSession(ctx context.Context, session *domain.V2GSession) error {
	m, err := ToMap(session)
	if err != nil {
		return err
	}
	delete(m, "id")
	delete(m, "node_label")
	delete(m, "created_at")
	return r.db.UpdateFields(ctx, "v2g_sessions", session.ID, m)
}

func (r *V2GRepository) GetSession(ctx context.Context, sessionID string) (*domain.V2GSession, error) {
	m, err := r.db.QueryFirst(ctx, "v2g_sessions", " AND n.id = $id", map[string]interface{}{"id": sessionID})
	if err != nil || m == nil {
		return nil, err
	}
	s := &domain.V2GSession{}
	return s, FromMap(m, s)
}

func (r *V2GRepository) GetSessionsByChargePoint(ctx context.Context, chargePointID string, limit int) ([]domain.V2GSession, error) {
	rows, err := r.db.QueryByLabel(ctx, "v2g_sessions",
		" AND n.charge_point_id = $cpid",
		map[string]interface{}{"cpid": chargePointID})
	if err != nil {
		return nil, err
	}
	var sessions []domain.V2GSession
	for _, m := range rows {
		var s domain.V2GSession
		if err := FromMap(m, &s); err == nil {
			sessions = append(sessions, s)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, nil
}

func (r *V2GRepository) GetSessionsByUser(ctx context.Context, userID string, limit int) ([]domain.V2GSession, error) {
	rows, err := r.db.QueryByLabel(ctx, "v2g_sessions",
		" AND n.user_id = $uid",
		map[string]interface{}{"uid": userID})
	if err != nil {
		return nil, err
	}
	var sessions []domain.V2GSession
	for _, m := range rows {
		var s domain.V2GSession
		if err := FromMap(m, &s); err == nil {
			sessions = append(sessions, s)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, nil
}

func (r *V2GRepository) GetActiveSessions(ctx context.Context) ([]domain.V2GSession, error) {
	rows, err := r.db.QueryByLabel(ctx, "v2g_sessions",
		" AND n.status = $st",
		map[string]interface{}{"st": string(domain.V2GStatusActive)})
	if err != nil {
		return nil, err
	}
	var sessions []domain.V2GSession
	for _, m := range rows {
		var s domain.V2GSession
		if err := FromMap(m, &s); err == nil {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

// ── Preferences ─────────────────────────────────────────────────────────

func (r *V2GRepository) SavePreferences(ctx context.Context, prefs *domain.V2GPreferences) error {
	m, err := ToMap(prefs)
	if err != nil {
		return err
	}
	delete(m, "id")
	_, _, err = r.db.Merge(ctx, "v2g_preferences",
		map[string]interface{}{"user_id": prefs.UserID},
		m, m)
	return err
}

func (r *V2GRepository) GetPreferences(ctx context.Context, userID string) (*domain.V2GPreferences, error) {
	m, err := r.db.QueryFirst(ctx, "v2g_preferences",
		" AND n.user_id = $uid",
		map[string]interface{}{"uid": userID})
	if err != nil {
		return nil, err
	}
	if m == nil {
		// Return defaults
		return &domain.V2GPreferences{
			UserID:          userID,
			AutoDischarge:   false,
			MinGridPrice:    0.5,
			MaxDischargeKWh: 20,
			PreserveSOC:     30,
			NotifyOnStart:   true,
			NotifyOnEnd:     true,
		}, nil
	}
	p := &domain.V2GPreferences{}
	return p, FromMap(m, p)
}

// ── Events ──────────────────────────────────────────────────────────────

func (r *V2GRepository) CreateEvent(ctx context.Context, event *domain.V2GEvent) error {
	m, err := ToMap(event)
	if err != nil {
		return err
	}
	_, err = r.db.Insert(ctx, "v2g_events", m)
	return err
}

func (r *V2GRepository) GetEventsBySession(ctx context.Context, sessionID string) ([]domain.V2GEvent, error) {
	rows, err := r.db.QueryByLabel(ctx, "v2g_events",
		" AND n.session_id = $sid",
		map[string]interface{}{"sid": sessionID})
	if err != nil {
		return nil, err
	}
	var events []domain.V2GEvent
	for _, m := range rows {
		var e domain.V2GEvent
		if err := FromMap(m, &e); err == nil {
			events = append(events, e)
		}
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	return events, nil
}

// ── Statistics ───────────────────────────────────────────────────────────

func (r *V2GRepository) computeStats(sessions []domain.V2GSession, entityID, entityType string, startDate, endDate time.Time) *domain.V2GStats {
	stats := &domain.V2GStats{
		EntityID:   entityID,
		EntityType: entityType,
		StartDate:  startDate,
		EndDate:    endDate,
	}
	var totalDuration time.Duration
	var peakCount int
	for _, s := range sessions {
		if s.CreatedAt.Before(startDate) || s.CreatedAt.After(endDate) {
			continue
		}
		stats.TotalSessions++
		if s.Direction == domain.V2GDirectionDischarging {
			stats.TotalEnergyDischargedKWh += s.EnergyTransferred
		}
		stats.TotalCompensation += s.UserCompensation
		if s.EndTime != nil {
			totalDuration += s.EndTime.Sub(s.StartTime)
		}
		hour := s.StartTime.Hour()
		if hour >= 18 || hour < 6 {
			peakCount++
		}
	}
	if stats.TotalSessions > 0 {
		stats.AverageSessionDuration = totalDuration / time.Duration(stats.TotalSessions)
		stats.PeakHoursParticipation = float64(peakCount) / float64(stats.TotalSessions) * 100
	}
	return stats
}

func (r *V2GRepository) GetUserStats(ctx context.Context, userID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	sessions, err := r.GetSessionsByUser(ctx, userID, 0)
	if err != nil {
		return nil, err
	}
	return r.computeStats(sessions, userID, "user", startDate, endDate), nil
}

func (r *V2GRepository) GetChargePointStats(ctx context.Context, chargePointID string, startDate, endDate time.Time) (*domain.V2GStats, error) {
	sessions, err := r.GetSessionsByChargePoint(ctx, chargePointID, 0)
	if err != nil {
		return nil, err
	}
	return r.computeStats(sessions, chargePointID, "charge_point", startDate, endDate), nil
}

func (r *V2GRepository) GetGlobalStats(ctx context.Context, startDate, endDate time.Time) (*domain.V2GStats, error) {
	rows, err := r.db.QueryByLabel(ctx, "v2g_sessions", "", nil)
	if err != nil {
		return nil, err
	}
	var sessions []domain.V2GSession
	for _, m := range rows {
		var s domain.V2GSession
		if err := FromMap(m, &s); err == nil {
			sessions = append(sessions, s)
		}
	}
	return r.computeStats(sessions, "global", "global", startDate, endDate), nil
}

// ── Compensation ────────────────────────────────────────────────────────

func (r *V2GRepository) GetPendingCompensations(ctx context.Context) ([]domain.V2GSession, error) {
	rows, err := r.db.QueryByLabel(ctx, "v2g_sessions",
		" AND n.status = $st",
		map[string]interface{}{"st": string(domain.V2GStatusCompleted)})
	if err != nil {
		return nil, err
	}
	var pending []domain.V2GSession
	for _, m := range rows {
		// Check if compensation is pending (user_compensation > 0 and no payment_id)
		comp := GetFloat64(m, "user_compensation")
		paymentID := GetString(m, "compensation_payment_id")
		if comp > 0 && paymentID == "" {
			var s domain.V2GSession
			if err := FromMap(m, &s); err == nil {
				pending = append(pending, s)
			}
		}
	}
	return pending, nil
}

func (r *V2GRepository) MarkCompensationPaid(ctx context.Context, sessionID string, paymentID string) error {
	return r.db.UpdateFields(ctx, "v2g_sessions", sessionID, map[string]interface{}{
		"compensation_payment_id": paymentID,
		"compensation_paid_at":    time.Now().Format(time.RFC3339),
	})
}
