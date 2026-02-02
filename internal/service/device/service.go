package device

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/adapter/queue"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

const (
	cacheKeyPrefix = "device:"
	cacheTTL       = 30 * time.Second
)

type Service struct {
	repo  ports.ChargePointRepository
	cache ports.Cache
	mq    queue.MessageQueue
	log   *zap.Logger
}

func NewService(repo ports.ChargePointRepository, cache ports.Cache, mq queue.MessageQueue, log *zap.Logger) ports.DeviceService {
	return &Service{
		repo:  repo,
		cache: cache,
		mq:    mq,
		log:   log,
	}
}

func (s *Service) GetDevice(ctx context.Context, id string) (*domain.ChargePoint, error) {
	// Try cache first
	cacheKey := cacheKeyPrefix + id
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var cp domain.ChargePoint
		if err := json.Unmarshal([]byte(cached), &cp); err == nil {
			s.log.Debug("Cache hit for device", zap.String("id", id))
			return &cp, nil
		}
	}

	// Cache miss - fetch from repository
	cp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Set cache
	if cp != nil {
		if data, err := json.Marshal(cp); err == nil {
			if err := s.cache.Set(ctx, cacheKey, string(data), cacheTTL); err != nil {
				s.log.Warn("Failed to cache device", zap.String("id", id), zap.Error(err))
			}
		}
	}

	return cp, nil
}

func (s *Service) ListDevices(ctx context.Context, filter map[string]interface{}) ([]domain.ChargePoint, error) {
	return s.repo.FindAll(ctx, filter)
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status domain.ChargePointStatus) error {
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := cacheKeyPrefix + id
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.log.Warn("Failed to invalidate cache", zap.String("id", id), zap.Error(err))
	}

	// Publish event (if message queue available)
	if s.mq != nil {
		event := map[string]interface{}{
			"device_id": id,
			"status":    status,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		if data, err := json.Marshal(event); err == nil {
			if err := s.mq.Publish("device.status.changed", data); err != nil {
				s.log.Warn("Failed to publish status change event", zap.Error(err))
			}
		}
	}

	return nil
}

func (s *Service) GetNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error) {
	return s.repo.FindNearby(ctx, lat, lon, radius)
}

// ListAvailableDevices returns all devices with Available status (used by VoiceAssistant)
func (s *Service) ListAvailableDevices(ctx context.Context) ([]domain.ChargePoint, error) {
	filter := map[string]interface{}{
		"status": domain.ChargePointStatusAvailable,
	}

	devices, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.log.Error("Failed to list available devices", zap.Error(err))
		return nil, fmt.Errorf("failed to list available devices: %w", err)
	}

	return devices, nil
}
