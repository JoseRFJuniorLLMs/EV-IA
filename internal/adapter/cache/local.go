package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// LocalCache implements the ports.Cache interface using an in-memory map.
// Used as a fallback when Redis is unavailable.
type LocalCache struct {
	data   map[string]cacheEntry
	mu     sync.RWMutex
	log    *zap.Logger
	stopCh chan struct{}
}

// NewLocalCache creates a new in-memory cache with periodic cleanup
func NewLocalCache(cleanupInterval time.Duration, log *zap.Logger) ports.Cache {
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute
	}

	c := &LocalCache{
		data:   make(map[string]cacheEntry),
		log:    log,
		stopCh: make(chan struct{}),
	}

	go c.cleanupLoop(cleanupInterval)

	log.Info("Local in-memory cache initialized",
		zap.Duration("cleanup_interval", cleanupInterval),
	)
	return c
}

func (c *LocalCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}

	if !entry.expiresAt.IsZero() && entry.expiresAt.Before(time.Now()) {
		return "", fmt.Errorf("key expired: %s", key)
	}

	return entry.value, nil
}

func (c *LocalCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		strVal = string(data)
	}

	entry := cacheEntry{value: strVal}
	if expiration > 0 {
		entry.expiresAt = time.Now().Add(expiration)
	}

	c.data[key] = entry
	return nil
}

func (c *LocalCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *LocalCache) Ping() error {
	return nil
}

func (c *LocalCache) Close() error {
	close(c.stopCh)
	return nil
}

func (c *LocalCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

func (c *LocalCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := 0
	for key, entry := range c.data {
		if !entry.expiresAt.IsZero() && entry.expiresAt.Before(now) {
			delete(c.data, key)
			expired++
		}
	}

	if expired > 0 {
		c.log.Debug("Cache cleanup completed", zap.Int("expired_entries", expired))
	}
}
