package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedis_BasicOperations tests basic Redis operations
func TestRedis_BasicOperations(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	// Set and Get
	t.Run("SetGet", func(t *testing.T) {
		err := env.Redis.Set(ctx, "test:key", "test-value", time.Minute).Err()
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}

		val, err := env.Redis.Get(ctx, "test:key").Result()
		if err != nil {
			t.Fatalf("Failed to get key: %v", err)
		}

		if val != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", val)
		}
	})

	// Set with expiration
	t.Run("SetWithExpiration", func(t *testing.T) {
		err := env.Redis.Set(ctx, "test:expiring", "value", 100*time.Millisecond).Err()
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}

		// Verify it exists
		_, err = env.Redis.Get(ctx, "test:expiring").Result()
		if err != nil {
			t.Fatalf("Key should exist: %v", err)
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Verify it's gone
		_, err = env.Redis.Get(ctx, "test:expiring").Result()
		if err != redis.Nil {
			t.Error("Key should have expired")
		}
	})

	// Delete
	t.Run("Delete", func(t *testing.T) {
		env.Redis.Set(ctx, "test:delete", "value", time.Minute)

		err := env.Redis.Del(ctx, "test:delete").Err()
		if err != nil {
			t.Fatalf("Failed to delete key: %v", err)
		}

		_, err = env.Redis.Get(ctx, "test:delete").Result()
		if err != redis.Nil {
			t.Error("Key should have been deleted")
		}
	})

	// Exists
	t.Run("Exists", func(t *testing.T) {
		env.Redis.Set(ctx, "test:exists", "value", time.Minute)

		exists, err := env.Redis.Exists(ctx, "test:exists").Result()
		if err != nil {
			t.Fatalf("Failed to check exists: %v", err)
		}

		if exists != 1 {
			t.Error("Key should exist")
		}

		exists, err = env.Redis.Exists(ctx, "test:nonexistent").Result()
		if err != nil {
			t.Fatalf("Failed to check exists: %v", err)
		}

		if exists != 0 {
			t.Error("Key should not exist")
		}
	})
}

// TestRedis_JSONOperations tests storing and retrieving JSON
func TestRedis_JSONOperations(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	type Device struct {
		ID     string `json:"id"`
		Vendor string `json:"vendor"`
		Model  string `json:"model"`
		Status string `json:"status"`
	}

	// Store JSON
	t.Run("StoreJSON", func(t *testing.T) {
		device := Device{
			ID:     "CP001",
			Vendor: "ABB",
			Model:  "Terra 184",
			Status: "Available",
		}

		data, err := json.Marshal(device)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		err = env.Redis.Set(ctx, "device:CP001", data, time.Minute).Err()
		if err != nil {
			t.Fatalf("Failed to store JSON: %v", err)
		}
	})

	// Retrieve JSON
	t.Run("RetrieveJSON", func(t *testing.T) {
		data, err := env.Redis.Get(ctx, "device:CP001").Bytes()
		if err != nil {
			t.Fatalf("Failed to get JSON: %v", err)
		}

		var device Device
		if err := json.Unmarshal(data, &device); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if device.Vendor != "ABB" {
			t.Errorf("Expected vendor 'ABB', got '%s'", device.Vendor)
		}
	})
}

// TestRedis_HashOperations tests Redis hash operations
func TestRedis_HashOperations(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	// HSet
	t.Run("HSet", func(t *testing.T) {
		err := env.Redis.HSet(ctx, "user:123", map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
			"role":  "user",
		}).Err()

		if err != nil {
			t.Fatalf("Failed to HSet: %v", err)
		}
	})

	// HGet
	t.Run("HGet", func(t *testing.T) {
		name, err := env.Redis.HGet(ctx, "user:123", "name").Result()
		if err != nil {
			t.Fatalf("Failed to HGet: %v", err)
		}

		if name != "John Doe" {
			t.Errorf("Expected 'John Doe', got '%s'", name)
		}
	})

	// HGetAll
	t.Run("HGetAll", func(t *testing.T) {
		data, err := env.Redis.HGetAll(ctx, "user:123").Result()
		if err != nil {
			t.Fatalf("Failed to HGetAll: %v", err)
		}

		if len(data) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(data))
		}

		if data["email"] != "john@example.com" {
			t.Errorf("Expected email 'john@example.com', got '%s'", data["email"])
		}
	})

	// HIncrBy
	t.Run("HIncrBy", func(t *testing.T) {
		env.Redis.HSet(ctx, "stats:daily", "requests", 0)

		newVal, err := env.Redis.HIncrBy(ctx, "stats:daily", "requests", 1).Result()
		if err != nil {
			t.Fatalf("Failed to HIncrBy: %v", err)
		}

		if newVal != 1 {
			t.Errorf("Expected 1, got %d", newVal)
		}

		newVal, err = env.Redis.HIncrBy(ctx, "stats:daily", "requests", 5).Result()
		if err != nil {
			t.Fatalf("Failed to HIncrBy: %v", err)
		}

		if newVal != 6 {
			t.Errorf("Expected 6, got %d", newVal)
		}
	})
}

// TestRedis_ListOperations tests Redis list operations
func TestRedis_ListOperations(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	// LPush
	t.Run("LPush", func(t *testing.T) {
		err := env.Redis.LPush(ctx, "queue:events", "event1", "event2", "event3").Err()
		if err != nil {
			t.Fatalf("Failed to LPush: %v", err)
		}
	})

	// LLen
	t.Run("LLen", func(t *testing.T) {
		length, err := env.Redis.LLen(ctx, "queue:events").Result()
		if err != nil {
			t.Fatalf("Failed to LLen: %v", err)
		}

		if length != 3 {
			t.Errorf("Expected length 3, got %d", length)
		}
	})

	// RPop
	t.Run("RPop", func(t *testing.T) {
		val, err := env.Redis.RPop(ctx, "queue:events").Result()
		if err != nil {
			t.Fatalf("Failed to RPop: %v", err)
		}

		if val != "event1" {
			t.Errorf("Expected 'event1', got '%s'", val)
		}
	})

	// LRange
	t.Run("LRange", func(t *testing.T) {
		vals, err := env.Redis.LRange(ctx, "queue:events", 0, -1).Result()
		if err != nil {
			t.Fatalf("Failed to LRange: %v", err)
		}

		if len(vals) != 2 {
			t.Errorf("Expected 2 elements, got %d", len(vals))
		}
	})
}

// TestRedis_SetOperations tests Redis set operations
func TestRedis_SetOperations(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	// SAdd
	t.Run("SAdd", func(t *testing.T) {
		err := env.Redis.SAdd(ctx, "online:users", "user1", "user2", "user3").Err()
		if err != nil {
			t.Fatalf("Failed to SAdd: %v", err)
		}
	})

	// SMembers
	t.Run("SMembers", func(t *testing.T) {
		members, err := env.Redis.SMembers(ctx, "online:users").Result()
		if err != nil {
			t.Fatalf("Failed to SMembers: %v", err)
		}

		if len(members) != 3 {
			t.Errorf("Expected 3 members, got %d", len(members))
		}
	})

	// SIsMember
	t.Run("SIsMember", func(t *testing.T) {
		isMember, err := env.Redis.SIsMember(ctx, "online:users", "user1").Result()
		if err != nil {
			t.Fatalf("Failed to SIsMember: %v", err)
		}

		if !isMember {
			t.Error("user1 should be a member")
		}

		isMember, err = env.Redis.SIsMember(ctx, "online:users", "user999").Result()
		if err != nil {
			t.Fatalf("Failed to SIsMember: %v", err)
		}

		if isMember {
			t.Error("user999 should not be a member")
		}
	})

	// SRem
	t.Run("SRem", func(t *testing.T) {
		err := env.Redis.SRem(ctx, "online:users", "user2").Err()
		if err != nil {
			t.Fatalf("Failed to SRem: %v", err)
		}

		isMember, _ := env.Redis.SIsMember(ctx, "online:users", "user2").Result()
		if isMember {
			t.Error("user2 should have been removed")
		}
	})
}

// TestRedis_PubSub tests Redis pub/sub
func TestRedis_PubSub(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	// Subscribe and publish
	t.Run("PubSub", func(t *testing.T) {
		pubsub := env.Redis.Subscribe(ctx, "test:channel")
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(ctx)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Publish in goroutine
		go func() {
			time.Sleep(100 * time.Millisecond)
			env.Redis.Publish(ctx, "test:channel", "test-message")
		}()

		// Receive message with timeout
		ch := pubsub.Channel()
		select {
		case msg := <-ch:
			if msg.Payload != "test-message" {
				t.Errorf("Expected 'test-message', got '%s'", msg.Payload)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for message")
		}
	})
}

// TestRedis_Caching tests caching patterns
func TestRedis_Caching(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	// Cache-aside pattern
	t.Run("CacheAside", func(t *testing.T) {
		key := "cache:device:CP001"

		// Cache miss
		_, err := env.Redis.Get(ctx, key).Result()
		if err != redis.Nil {
			t.Error("Expected cache miss")
		}

		// Simulate fetching from DB and caching
		data := `{"id":"CP001","vendor":"ABB"}`
		err = env.Redis.Set(ctx, key, data, 5*time.Minute).Err()
		if err != nil {
			t.Fatalf("Failed to cache: %v", err)
		}

		// Cache hit
		cached, err := env.Redis.Get(ctx, key).Result()
		if err != nil {
			t.Fatalf("Cache hit failed: %v", err)
		}

		if cached != data {
			t.Errorf("Cached data mismatch")
		}
	})

	// Write-through pattern
	t.Run("WriteThrough", func(t *testing.T) {
		key := "cache:user:123:balance"

		// Update cache and DB together (simulated)
		newBalance := "150.00"
		err := env.Redis.Set(ctx, key, newBalance, 5*time.Minute).Err()
		if err != nil {
			t.Fatalf("Failed to update cache: %v", err)
		}

		// Verify cache is updated
		cached, _ := env.Redis.Get(ctx, key).Result()
		if cached != newBalance {
			t.Errorf("Expected '%s', got '%s'", newBalance, cached)
		}
	})
}

// TestRedis_RateLimiting tests rate limiting pattern
func TestRedis_RateLimiting(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.Redis == nil {
		t.Skip("Redis not available")
	}

	FlushRedis(t, env.Redis)
	ctx := context.Background()

	// Sliding window rate limiter
	t.Run("RateLimiter", func(t *testing.T) {
		key := "ratelimit:user:123"
		limit := int64(5)
		window := time.Minute

		// Simulate requests
		for i := 0; i < 7; i++ {
			count, err := env.Redis.Incr(ctx, key).Result()
			if err != nil {
				t.Fatalf("Failed to increment: %v", err)
			}

			// Set expiration on first request
			if count == 1 {
				env.Redis.Expire(ctx, key, window)
			}

			if count <= limit {
				// Request allowed
				t.Logf("Request %d allowed", i+1)
			} else {
				// Request denied
				t.Logf("Request %d denied (rate limited)", i+1)
			}
		}

		// Verify count
		count, _ := env.Redis.Get(ctx, key).Int64()
		if count != 7 {
			t.Errorf("Expected count 7, got %d", count)
		}
	})
}
