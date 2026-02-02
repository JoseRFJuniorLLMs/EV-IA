package circuitbreaker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HTTPClient wraps an HTTP client with circuit breaker protection
type HTTPClient struct {
	client  *http.Client
	breaker *CircuitBreaker
	log     *zap.Logger
}

// NewHTTPClient creates a new HTTP client with circuit breaker
func NewHTTPClient(client *http.Client, breaker *CircuitBreaker, log *zap.Logger) *HTTPClient {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &HTTPClient{
		client:  client,
		breaker: breaker,
		log:     log,
	}
}

// Do executes an HTTP request with circuit breaker protection
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	result, err := c.breaker.ExecuteCtx(req.Context(), func(ctx context.Context) (interface{}, error) {
		req = req.WithContext(ctx)
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}

		// Consider 5xx errors as failures for circuit breaker
		if resp.StatusCode >= 500 {
			return resp, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return resp, nil
	})

	if err != nil {
		if IsCircuitOpen(err) {
			c.log.Warn("Circuit breaker open, request blocked",
				zap.String("url", req.URL.String()),
				zap.String("breaker", c.breaker.Name()),
			)
		}
		return nil, err
	}

	return result.(*http.Response), nil
}

// Get performs a GET request with circuit breaker protection
func (c *HTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post performs a POST request with circuit breaker protection
func (c *HTTPClient) Post(ctx context.Context, url string, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// HTTPClientSettings configures the HTTP client with circuit breaker
type HTTPClientSettings struct {
	// HTTP client settings
	Timeout time.Duration

	// Circuit breaker settings
	Name             string
	MaxRequests      uint32
	Interval         time.Duration
	BreakerTimeout   time.Duration
	FailureThreshold uint32
	SuccessThreshold uint32
}

// DefaultHTTPClientSettings returns default settings
func DefaultHTTPClientSettings(name string) HTTPClientSettings {
	return HTTPClientSettings{
		Name:             name,
		Timeout:          30 * time.Second,
		MaxRequests:      3,
		Interval:         60 * time.Second,
		BreakerTimeout:   30 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 2,
	}
}

// NewHTTPClientWithSettings creates a new HTTP client with the given settings
func NewHTTPClientWithSettings(settings HTTPClientSettings, log *zap.Logger) *HTTPClient {
	client := &http.Client{
		Timeout: settings.Timeout,
	}

	breaker := New(Settings{
		Name:             settings.Name,
		MaxRequests:      settings.MaxRequests,
		Interval:         settings.Interval,
		Timeout:          settings.BreakerTimeout,
		FailureThreshold: settings.FailureThreshold,
		SuccessThreshold: settings.SuccessThreshold,
		OnStateChange: func(name string, from State, to State) {
			log.Info("HTTP client circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}, log)

	return NewHTTPClient(client, breaker, log)
}

// ServiceClient provides circuit breaker protection for service calls
type ServiceClient struct {
	manager *Manager
	log     *zap.Logger
}

// NewServiceClient creates a new service client
func NewServiceClient(manager *Manager, log *zap.Logger) *ServiceClient {
	return &ServiceClient{
		manager: manager,
		log:     log,
	}
}

// Call executes a service call with circuit breaker protection
func (c *ServiceClient) Call(ctx context.Context, service string, fn func(context.Context) error) error {
	breaker := c.manager.Get(service, DefaultSettings())

	_, err := breaker.ExecuteCtx(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, fn(ctx)
	})

	return err
}

// CallWithResult executes a service call with circuit breaker protection and returns a result
func CallWithResult[T any](c *ServiceClient, ctx context.Context, service string, fn func(context.Context) (T, error)) (T, error) {
	breaker := c.manager.Get(service, DefaultSettings())

	result, err := breaker.ExecuteCtx(ctx, func(ctx context.Context) (interface{}, error) {
		return fn(ctx)
	})

	if err != nil {
		var zero T
		return zero, err
	}

	return result.(T), nil
}

// RetryWithBackoff executes a function with retry and exponential backoff
func RetryWithBackoff(ctx context.Context, maxRetries int, initialDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for i := 0; i <= maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry circuit breaker errors
		if IsCircuitOpen(err) || IsTooManyRequests(err) {
			return err
		}

		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2 // Exponential backoff
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
