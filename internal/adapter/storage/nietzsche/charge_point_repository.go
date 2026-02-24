// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package nietzsche

import (
	"context"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
	"go.uber.org/zap"
)

type ChargePointRepository struct {
	db  *DB
	log *zap.Logger
}

func NewChargePointRepository(db *DB, log *zap.Logger) ports.ChargePointRepository {
	return &ChargePointRepository{db: db, log: log}
}

func (r *ChargePointRepository) Save(ctx context.Context, cp *domain.ChargePoint) error {
	m, err := ToMap(cp)
	if err != nil {
		return err
	}
	_, err = r.db.Insert(ctx, "charge_points", m)
	return err
}

func (r *ChargePointRepository) FindByID(ctx context.Context, id string) (*domain.ChargePoint, error) {
	m, err := r.db.QueryFirst(ctx, "charge_points", " AND n.id = $id", map[string]interface{}{"id": id})
	if err != nil || m == nil {
		return nil, err
	}
	cp := &domain.ChargePoint{}
	if err := FromMap(m, cp); err != nil {
		return nil, err
	}
	// Load connectors
	connRows, err := r.db.QueryByLabel(ctx, "connectors", " AND n.charge_point_id = $cpid", map[string]interface{}{"cpid": id})
	if err == nil {
		for _, cr := range connRows {
			var c domain.Connector
			if err := FromMap(cr, &c); err == nil {
				cp.Connectors = append(cp.Connectors, c)
			}
		}
	}
	// Load location
	if cp.LocationID != "" {
		locM, err := r.db.QueryFirst(ctx, "locations", " AND n.id = $lid", map[string]interface{}{"lid": cp.LocationID})
		if err == nil && locM != nil {
			loc := &domain.Location{}
			if err := FromMap(locM, loc); err == nil {
				cp.Location = loc
			}
		}
	}
	return cp, nil
}

func (r *ChargePointRepository) FindAll(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
	rows, err := r.db.QueryByLabel(ctx, "charge_points", "", nil)
	if err != nil {
		return nil, err
	}
	var result []domain.ChargePoint
	for _, m := range rows {
		// Apply filters
		match := true
		for k, v := range filter {
			if mv, ok := m[k]; !ok || mv != v {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		var cp domain.ChargePoint
		if err := FromMap(m, &cp); err == nil {
			result = append(result, cp)
		}
	}
	return result, nil
}

func (r *ChargePointRepository) UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error {
	return r.db.UpdateFields(ctx, "charge_points", id, map[string]interface{}{
		"status": string(status),
	})
}

func (r *ChargePointRepository) FindNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error) {
	// Load all locations, compute Haversine distance, filter by radius
	locRows, err := r.db.QueryByLabel(ctx, "locations", "", nil)
	if err != nil {
		return nil, err
	}
	nearbyLocIDs := map[string]bool{}
	for _, lm := range locRows {
		locLat := GetFloat64(lm, "latitude")
		locLon := GetFloat64(lm, "longitude")
		dist := Haversine(lat, lon, locLat, locLon)
		if dist <= radius {
			nearbyLocIDs[GetString(lm, "id")] = true
		}
	}
	if len(nearbyLocIDs) == 0 {
		return nil, nil
	}

	// Load charge points at nearby locations
	cpRows, err := r.db.QueryByLabel(ctx, "charge_points", "", nil)
	if err != nil {
		return nil, err
	}
	var result []domain.ChargePoint
	for _, m := range cpRows {
		locID := GetString(m, "location_id")
		if !nearbyLocIDs[locID] {
			continue
		}
		var cp domain.ChargePoint
		if err := FromMap(m, &cp); err == nil {
			result = append(result, cp)
		}
	}
	return result, nil
}
