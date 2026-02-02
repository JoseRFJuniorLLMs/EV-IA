package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Errors
var (
	ErrCircuitOpen    = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// Settings configures the circuit breaker
type Settings struct {
	// Name identifies the circuit breaker
	Name string

	// MaxRequests is the maximum number of requests allowed to pass through
	// when the circuit breaker is half-open
	MaxRequests uint32

	// Interval is the cyclic period of the closed state
	// for the circuit breaker to clear the internal counts
	Interval time.Duration

	// Timeout is the period of the open state
	// after which the state becomes half-open
	Timeout time.Duration

	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold uint32

	// SuccessThreshold is the number of successes needed to close the circuit
	// from half-open state
	SuccessThreshold uint32

	// ReadyToTrip is a callback to determine if the circuit should trip
	ReadyToTrip func(counts Counts) bool

	// OnStateChange is called when the circuit breaker changes state
	OnStateChange func(name string, from State, to State)

	// IsSuccessful is a callback to determine if an error should be counted
	// as a failure. If nil, all non-nil errors are considered failures.
	IsSuccessful func(err error) bool
}

// Counts holds the numbers of requests and their successes/failures
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	failureThreshold uint32
	successThreshold uint32
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)
	isSuccessful  func(err error) bool

	mu          sync.Mutex
	state       State
	generation  uint64
	counts      Counts
	expiry      time.Time
	log         *zap.Logger
}

// New creates a new circuit breaker with the given settings
func New(settings Settings, log *zap.Logger) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             settings.Name,
		maxRequests:      settings.MaxRequests,
		interval:         settings.Interval,
		timeout:          settings.Timeout,
		failureThreshold: settings.FailureThreshold,
		successThreshold: settings.SuccessThreshold,
		readyToTrip:      settings.ReadyToTrip,
		onStateChange:    settings.OnStateChange,
		isSuccessful:     settings.IsSuccessful,
		log:              log,
	}

	// Set defaults
	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}
	if cb.interval == 0 {
		cb.interval = 60 * time.Second
	}
	if cb.timeout == 0 {
		cb.timeout = 30 * time.Second
	}
	if cb.failureThreshold == 0 {
		cb.failureThreshold = 5
	}
	if cb.successThreshold == 0 {
		cb.successThreshold = 1
	}
	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures >= cb.failureThreshold
		}
	}
	if cb.isSuccessful == nil {
		cb.isSuccessful = func(err error) bool {
			return err == nil
		}
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// Execute runs the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := fn()
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

// ExecuteCtx runs the given function with context if the circuit breaker allows it
func (cb *CircuitBreaker) ExecuteCtx(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := fn(ctx)
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns a copy of the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.counts
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	switch state {
	case StateOpen:
		return generation, ErrCircuitOpen
	case StateHalfOpen:
		if cb.counts.Requests >= cb.maxRequests {
			return generation, ErrTooManyRequests
		}
	}

	cb.counts.Requests++
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
	case StateHalfOpen:
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
		if cb.counts.ConsecutiveSuccesses >= cb.successThreshold {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}

	cb.log.Info("Circuit breaker state changed",
		zap.String("name", cb.name),
		zap.String("from", prev.String()),
		zap.String("to", state.String()),
	)
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	log      *zap.Logger
}

// NewManager creates a new circuit breaker manager
func NewManager(log *zap.Logger) *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
		log:      log,
	}
}

// Get returns a circuit breaker by name, creating it if it doesn't exist
func (m *Manager) Get(name string, settings Settings) *CircuitBreaker {
	m.mu.RLock()
	cb, exists := m.breakers[name]
	m.mu.RUnlock()

	if exists {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = m.breakers[name]; exists {
		return cb
	}

	settings.Name = name
	cb = New(settings, m.log)
	m.breakers[name] = cb

	return cb
}

// GetAll returns all circuit breakers
func (m *Manager) GetAll() map[string]*CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*CircuitBreaker, len(m.breakers))
	for k, v := range m.breakers {
		result[k] = v
	}
	return result
}

// Status returns the status of all circuit breakers
func (m *Manager) Status() map[string]BreakerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]BreakerStatus, len(m.breakers))
	for name, cb := range m.breakers {
		counts := cb.Counts()
		status[name] = BreakerStatus{
			Name:   name,
			State:  cb.State().String(),
			Counts: counts,
		}
	}
	return status
}

// BreakerStatus represents the status of a circuit breaker
type BreakerStatus struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Counts Counts `json:"counts"`
}

// DefaultSettings returns default circuit breaker settings
func DefaultSettings() Settings {
	return Settings{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 2,
	}
}

// Execute is a convenience function to execute with circuit breaker
func Execute(cb *CircuitBreaker, fn func() error) error {
	_, err := cb.Execute(func() (interface{}, error) {
		return nil, fn()
	})
	return err
}

// ExecuteWithResult is a convenience function to execute with circuit breaker and return a result
func ExecuteWithResult[T any](cb *CircuitBreaker, fn func() (T, error)) (T, error) {
	result, err := cb.Execute(func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return result.(T), nil
}

// Wrap wraps a function with circuit breaker protection
func Wrap(cb *CircuitBreaker, fn func() error) func() error {
	return func() error {
		return Execute(cb, fn)
	}
}

// WrapWithResult wraps a function with circuit breaker protection
func WrapWithResult[T any](cb *CircuitBreaker, fn func() (T, error)) func() (T, error) {
	return func() (T, error) {
		return ExecuteWithResult(cb, fn)
	}
}

// Error wraps an error with circuit breaker context
type Error struct {
	Name  string
	State State
	Err   error
}

func (e *Error) Error() string {
	return fmt.Sprintf("circuit breaker %s (%s): %v", e.Name, e.State, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// IsCircuitOpen checks if the error is due to an open circuit
func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}

// IsTooManyRequests checks if the error is due to too many requests
func IsTooManyRequests(err error) bool {
	return errors.Is(err, ErrTooManyRequests)
}
