package health

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string        `json:"name"`
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration_ms"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    Status                 `json:"status"`
	Version   string                 `json:"version,omitempty"`
	Uptime    string                 `json:"uptime,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// ReadyResponse represents the readiness response
type ReadyResponse struct {
	Ready     bool                   `json:"ready"`
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// Checker defines a health check function
type Checker func(ctx context.Context) CheckResult

// Service handles health checks
type Service struct {
	db        *sql.DB
	redis     *redis.Client
	natsURL   string
	startTime time.Time
	version   string
	checkers  map[string]Checker
	log       *zap.Logger
	mu        sync.RWMutex
}

// Config holds health service configuration
type Config struct {
	Version string
	DB      *sql.DB
	Redis   *redis.Client
	NatsURL string
}

// NewService creates a new health service
func NewService(config *Config, log *zap.Logger) *Service {
	s := &Service{
		db:        config.DB,
		redis:     config.Redis,
		natsURL:   config.NatsURL,
		startTime: time.Now(),
		version:   config.Version,
		checkers:  make(map[string]Checker),
		log:       log,
	}

	// Register default checkers
	if config.DB != nil {
		s.RegisterChecker("database", s.checkDatabase)
	}
	if config.Redis != nil {
		s.RegisterChecker("redis", s.checkRedis)
	}
	if config.NatsURL != "" {
		s.RegisterChecker("nats", s.checkNATS)
	}

	return s
}

// RegisterChecker registers a custom health checker
func (s *Service) RegisterChecker(name string, checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
	s.log.Info("Registered health checker", zap.String("name", name))
}

// Health performs a basic liveness check
func (s *Service) Health(ctx context.Context) *HealthResponse {
	return &HealthResponse{
		Status:    StatusHealthy,
		Version:   s.version,
		Uptime:    time.Since(s.startTime).String(),
		Timestamp: time.Now(),
	}
}

// Ready performs a comprehensive readiness check
func (s *Service) Ready(ctx context.Context) *ReadyResponse {
	s.mu.RLock()
	checkers := make(map[string]Checker, len(s.checkers))
	for k, v := range s.checkers {
		checkers[k] = v
	}
	s.mu.RUnlock()

	// Run all checks concurrently
	results := make(map[string]CheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, checker := range checkers {
		wg.Add(1)
		go func(name string, checker Checker) {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			result := checker(checkCtx)

			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(name, checker)
	}

	wg.Wait()

	// Determine overall status
	overallStatus := StatusHealthy
	allReady := true

	for _, result := range results {
		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			allReady = false
		} else if result.Status == StatusDegraded && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		}
	}

	return &ReadyResponse{
		Ready:     allReady,
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    results,
	}
}

// checkDatabase checks the database connection
func (s *Service) checkDatabase(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      "database",
		Timestamp: time.Now(),
	}

	if s.db == nil {
		result.Status = StatusUnhealthy
		result.Message = "database not configured"
		result.Duration = time.Since(start)
		return result
	}

	err := s.db.PingContext(ctx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("ping failed: %v", err)
		s.log.Warn("Database health check failed", zap.Error(err))
	} else {
		result.Status = StatusHealthy
		result.Message = "connection ok"
	}

	return result
}

// checkRedis checks the Redis connection
func (s *Service) checkRedis(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      "redis",
		Timestamp: time.Now(),
	}

	if s.redis == nil {
		result.Status = StatusUnhealthy
		result.Message = "redis not configured"
		result.Duration = time.Since(start)
		return result
	}

	err := s.redis.Ping(ctx).Err()
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("ping failed: %v", err)
		s.log.Warn("Redis health check failed", zap.Error(err))
	} else {
		result.Status = StatusHealthy
		result.Message = "connection ok"
	}

	return result
}

// checkNATS checks the NATS connection
func (s *Service) checkNATS(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      "nats",
		Timestamp: time.Now(),
	}

	if s.natsURL == "" {
		result.Status = StatusUnhealthy
		result.Message = "nats not configured"
		result.Duration = time.Since(start)
		return result
	}

	// For now, just check if URL is configured
	// In production, you'd check actual connection status
	result.Duration = time.Since(start)
	result.Status = StatusHealthy
	result.Message = "configured"

	return result
}

// LivenessHandler returns a simple liveness check handler
func (s *Service) LivenessHandler() func() *HealthResponse {
	return func() *HealthResponse {
		return s.Health(context.Background())
	}
}

// ReadinessHandler returns a readiness check handler
func (s *Service) ReadinessHandler() func() *ReadyResponse {
	return func() *ReadyResponse {
		return s.Ready(context.Background())
	}
}
