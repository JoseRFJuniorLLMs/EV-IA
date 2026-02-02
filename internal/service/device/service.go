package device

import (
	"context"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/adapter/queue"
	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
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
	// cacheKey := "device:" + id
	// val, err := s.cache.Get(ctx, cacheKey)
	// if err == nil {
	// 	// unmarshal and return
	// }

	cp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Set cache
	// s.cache.Set(...)

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
	// s.cache.Delete(...)

	// Publish event
	// s.mq.Publish("device.status", ...)

	return nil
}

func (s *Service) GetNearby(ctx context.Context, lat, lon, radius float64) ([]domain.ChargePoint, error) {
	return s.repo.FindNearby(ctx, lat, lon, radius)
}
