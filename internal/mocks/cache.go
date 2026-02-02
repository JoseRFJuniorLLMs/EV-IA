package mocks

import (
	"context"
	"time"
)

// MockCache is a mock implementation of Cache interface
type MockCache struct {
	data     map[string]string
	GetFunc  func(ctx context.Context, key string) (string, error)
	SetFunc  func(ctx context.Context, key string, value string, expiration time.Duration) error
	DeleteFunc func(ctx context.Context, key string) error
	PingFunc func() error
	CloseFunc func() error
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]string),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (m *MockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, expiration)
	}
	m.data[key] = value
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}
	delete(m.data, key)
	return nil
}

func (m *MockCache) Ping() error {
	if m.PingFunc != nil {
		return m.PingFunc()
	}
	return nil
}

func (m *MockCache) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
