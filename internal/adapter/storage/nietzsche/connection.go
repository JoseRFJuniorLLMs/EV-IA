// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package nietzsche

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	sdk "nietzsche-sdk"
	"go.uber.org/zap"
)

const DefaultCollection = "ev_charging"

// DB wraps the NietzscheDB gRPC client for EV-IA repositories.
type DB struct {
	Client     *sdk.NietzscheClient
	Collection string
	Log        *zap.Logger
}

// NewConnection connects to NietzscheDB and returns a DB wrapper.
func NewConnection(addr string, log *zap.Logger) (*DB, error) {
	client, err := sdk.ConnectInsecure(addr)
	if err != nil {
		return nil, fmt.Errorf("nietzsche connect %s: %w", addr, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.HealthCheck(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("nietzsche health check: %w", err)
	}
	log.Info("NietzscheDB connected", zap.String("addr", addr), zap.String("collection", DefaultCollection))
	return &DB{Client: client, Collection: DefaultCollection, Log: log}, nil
}

// Close closes the gRPC connection.
func (db *DB) Close() error {
	return db.Client.Close()
}

// ── Query helpers ────────────────────────────────────────────────────────

// QueryByLabel returns content maps for nodes matching node_label.
func (db *DB) QueryByLabel(ctx context.Context, label string, extraWhere string, params map[string]interface{}) ([]map[string]interface{}, error) {
	if params == nil {
		params = map[string]interface{}{}
	}
	params["_label"] = label
	nql := fmt.Sprintf("MATCH (n) WHERE n.node_label = $_label%s RETURN n", extraWhere)
	result, err := db.Client.Query(ctx, nql, params, db.Collection)
	if err != nil {
		db.Log.Error("NQL query failed", zap.String("nql", nql), zap.Error(err))
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(result.Nodes))
	for _, n := range result.Nodes {
		rows = append(rows, n.Content)
	}
	return rows, nil
}

// QueryFirst returns the first matching node or nil.
func (db *DB) QueryFirst(ctx context.Context, label string, extraWhere string, params map[string]interface{}) (map[string]interface{}, error) {
	if params == nil {
		params = map[string]interface{}{}
	}
	params["_label"] = label
	nql := fmt.Sprintf("MATCH (n) WHERE n.node_label = $_label%s RETURN n LIMIT 1", extraWhere)
	result, err := db.Client.Query(ctx, nql, params, db.Collection)
	if err != nil {
		return nil, err
	}
	if len(result.Nodes) == 0 {
		return nil, nil
	}
	return result.Nodes[0].Content, nil
}

// Insert creates a new node with the given label and content.
func (db *DB) Insert(ctx context.Context, label string, content map[string]interface{}) (string, error) {
	content["node_label"] = label
	if _, ok := content["created_at"]; !ok {
		content["created_at"] = time.Now().Format(time.RFC3339)
	}
	if _, ok := content["updated_at"]; !ok {
		content["updated_at"] = time.Now().Format(time.RFC3339)
	}
	result, err := db.Client.InsertNode(ctx, sdk.InsertNodeOpts{
		Coords:     []float64{},
		Content:    content,
		NodeType:   label,
		Collection: db.Collection,
	})
	if err != nil {
		db.Log.Error("Insert failed", zap.String("label", label), zap.Error(err))
		return "", err
	}
	return result.ID, nil
}

// Merge upserts a node by matchKeys.
func (db *DB) Merge(ctx context.Context, label string, matchKeys, onCreate, onMatch map[string]interface{}) (string, bool, error) {
	if onCreate == nil {
		onCreate = map[string]interface{}{}
	}
	onCreate["node_label"] = label
	if _, ok := onCreate["created_at"]; !ok {
		onCreate["created_at"] = time.Now().Format(time.RFC3339)
	}
	if onMatch == nil {
		onMatch = map[string]interface{}{}
	}
	onMatch["updated_at"] = time.Now().Format(time.RFC3339)

	result, err := db.Client.MergeNode(ctx, sdk.MergeNodeOpts{
		Collection:  db.Collection,
		NodeType:    label,
		MatchKeys:   matchKeys,
		OnCreateSet: onCreate,
		OnMatchSet:  onMatch,
	})
	if err != nil {
		db.Log.Error("Merge failed", zap.String("label", label), zap.Error(err))
		return "", false, err
	}
	return result.NodeID, result.Created, nil
}

// UpdateFields updates fields on a node identified by its id.
func (db *DB) UpdateFields(ctx context.Context, label string, id string, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	_, _, err := db.Merge(ctx, label, map[string]interface{}{"id": id, "node_label": label}, nil, fields)
	return err
}

// DeleteNode removes a node by its NietzscheDB node ID.
func (db *DB) DeleteNode(ctx context.Context, nodeID string) error {
	return db.Client.DeleteNode(ctx, nodeID, db.Collection)
}

// ── Serialization helpers ────────────────────────────────────────────────

// ToMap converts a struct to a map via JSON roundtrip.
func ToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// FromMap converts a content map to a struct via JSON roundtrip.
func FromMap(m map[string]interface{}, dst interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// GetString extracts a string field from a content map.
func GetString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// GetFloat64 extracts a float64 field from a content map.
func GetFloat64(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// GetInt extracts an int field from a content map.
func GetInt(m map[string]interface{}, key string) int {
	return int(GetFloat64(m, key))
}

// GetBool extracts a bool field from a content map.
func GetBool(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// GetTime parses a time string from a content map.
func GetTime(m map[string]interface{}, key string) time.Time {
	s := GetString(m, key)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339Nano, s)
	}
	return t
}

// GetTimePtr parses a time string, returning nil if empty.
func GetTimePtr(m map[string]interface{}, key string) *time.Time {
	s := GetString(m, key)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339Nano, s)
	}
	if t.IsZero() {
		return nil
	}
	return &t
}

// Haversine computes distance in km between two lat/lng points.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
