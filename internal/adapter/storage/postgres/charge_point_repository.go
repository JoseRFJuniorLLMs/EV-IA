package postgres

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

type ChargePointRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewChargePointRepository(db *gorm.DB, log *zap.Logger) ports.ChargePointRepository {
	return &ChargePointRepository{
		db:  db,
		log: log,
	}
}

func (r *ChargePointRepository) Save(ctx context.Context, cp *domain.ChargePoint) error {
	result := r.db.WithContext(ctx).Save(cp)
	if result.Error != nil {
		r.log.Error("Failed to save charge point", zap.Error(result.Error))
		return result.Error
	}
	return nil
}

func (r *ChargePointRepository) FindByID(ctx context.Context, id string) (*domain.ChargePoint, error) {
	var cp domain.ChargePoint
	result := r.db.WithContext(ctx).Preload("Connectors").Preload("Location").First(&cp, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Or custom ErrNotFound
		}
		return nil, result.Error
	}
	return &cp, nil
}

func (r *ChargePointRepository) FindAll(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
	var cps []domain.ChargePoint
	// Basic filtering implementation
	query := r.db.WithContext(ctx).Preload("Connectors").Preload("Location")
	if status, ok := filter["status"]; ok {
		query = query.Where("status = ?", status)
	}
	// Add other filters as needed

	result := query.Find(&cps)
	if result.Error != nil {
		return nil, result.Error
	}
	return cps, nil
}

func (r *ChargePointRepository) UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error {
	result := r.db.WithContext(ctx).Model(&domain.ChargePoint{}).Where("id = ?", id).Update("status", status)
	return result.Error
}

func (r *ChargePointRepository) FindNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error) {
	var cps []domain.ChargePoint

	// Haversine formula in SQL for calculating distance between two points
	// Returns distance in kilometers
	// Earth radius: 6371 km
	//
	// Formula:
	// distance = 2 * R * asin(sqrt(
	//   sin²((lat2-lat1)/2) + cos(lat1) * cos(lat2) * sin²((lon2-lon1)/2)
	// ))
	haversineSQL := `
		SELECT cp.* FROM charge_points cp
		INNER JOIN locations l ON l.id = cp.location_id
		WHERE l.latitude IS NOT NULL
		  AND l.longitude IS NOT NULL
		  AND (
			6371 * 2 * ASIN(SQRT(
				POWER(SIN(RADIANS(l.latitude - ?) / 2), 2) +
				COS(RADIANS(?)) * COS(RADIANS(l.latitude)) *
				POWER(SIN(RADIANS(l.longitude - ?) / 2), 2)
			))
		  ) <= ?
		ORDER BY (
			6371 * 2 * ASIN(SQRT(
				POWER(SIN(RADIANS(l.latitude - ?) / 2), 2) +
				COS(RADIANS(?)) * COS(RADIANS(l.latitude)) *
				POWER(SIN(RADIANS(l.longitude - ?) / 2), 2)
			))
		) ASC
		LIMIT 50
	`

	// Execute raw SQL with Haversine formula
	// Parameters: lat (3x for formula), lon (2x), radius, lat (2x for ORDER BY), lon
	result := r.db.WithContext(ctx).Raw(
		haversineSQL,
		lat, lat, lon, radius, // WHERE clause
		lat, lat, lon,         // ORDER BY clause
	).Scan(&cps)

	if result.Error != nil {
		r.log.Error("Failed to find nearby charge points",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Float64("radius_km", radius),
			zap.Error(result.Error),
		)
		return nil, result.Error
	}

	// Preload relationships for each charge point found
	if len(cps) > 0 {
		ids := make([]string, len(cps))
		for i, cp := range cps {
			ids[i] = cp.ID
		}

		// Reload with preloaded relationships
		var cpsWithRelations []domain.ChargePoint
		if err := r.db.WithContext(ctx).
			Preload("Connectors").
			Preload("Location").
			Where("id IN ?", ids).
			Find(&cpsWithRelations).Error; err != nil {
			r.log.Error("Failed to preload charge point relations", zap.Error(err))
			return cps, nil // Return without relations rather than failing
		}

		// Maintain distance-based ordering
		cpMap := make(map[string]domain.ChargePoint, len(cpsWithRelations))
		for _, cp := range cpsWithRelations {
			cpMap[cp.ID] = cp
		}

		orderedCps := make([]domain.ChargePoint, 0, len(cps))
		for _, cp := range cps {
			if fullCp, ok := cpMap[cp.ID]; ok {
				orderedCps = append(orderedCps, fullCp)
			}
		}
		cps = orderedCps
	}

	r.log.Debug("Found nearby charge points",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon),
		zap.Float64("radius_km", radius),
		zap.Int("count", len(cps)),
	)

	return cps, nil
}
