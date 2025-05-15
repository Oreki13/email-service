package service

import (
	"context"
	"email-service/pkg/telemetry"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState menggambarkan status circuit breaker
type CircuitBreakerState string

const (
	// StateClosed menunjukkan circuit sedang tertutup (normal, meneruskan request)
	StateClosed CircuitBreakerState = "closed"
	// StateOpen menunjukkan circuit terbuka (semua request gagal cepat)
	StateOpen CircuitBreakerState = "open"
	// StateHalfOpen menunjukkan circuit setengah terbuka (mengizinkan beberapa request)
	StateHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreakerConfig merupakan konfigurasi untuk circuit breaker
type CircuitBreakerConfig struct {
	Threshold         int           // Jumlah error yang memicu circuit breaker
	Timeout           time.Duration // Waktu untuk mengubah dari open ke half-open
	ResetCountTimeout time.Duration // Waktu untuk me-reset counter error
	MaxHalfOpenCalls  int           // Jumlah maksimum panggilan saat half-open
}

// CircuitBreaker adalah implementasi dari pola Circuit Breaker
type CircuitBreaker struct {
	name          string
	state         CircuitBreakerState
	config        CircuitBreakerConfig
	failures      int
	lastAttempt   time.Time
	halfOpenCalls int
	mutex         sync.RWMutex
	logger        telemetry.Logger
}

// NewCircuitBreaker membuat instance baru circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig, logger telemetry.Logger) *CircuitBreaker {
	// Default values
	if config.Threshold <= 0 {
		config.Threshold = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.ResetCountTimeout <= 0 {
		config.ResetCountTimeout = 60 * time.Second
	}
	if config.MaxHalfOpenCalls <= 0 {
		config.MaxHalfOpenCalls = 1
	}

	logger.Info(context.Background(), "Membuat circuit breaker baru", telemetry.Fields{
		"name":       name,
		"threshold":  config.Threshold,
		"timeout":    config.Timeout.String(),
		"reset_time": config.ResetCountTimeout.String(),
	})

	return &CircuitBreaker{
		name:          name,
		state:         StateClosed,
		config:        config,
		failures:      0,
		lastAttempt:   time.Now(),
		halfOpenCalls: 0,
		logger:        logger,
	}
}

// Execute menjalankan fungsi dengan circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Pengecekan status circuit breaker
	if !cb.AllowRequest() {
		cb.logger.Warn(ctx, "Circuit breaker terbuka, menolak permintaan", telemetry.Fields{
			"name":  cb.name,
			"state": string(cb.state),
		})
		return fmt.Errorf("circuit breaker '%s' is open (state: %s), request rejected", cb.name, cb.state)
	}

	// Jalankan fungsi
	err := fn()

	// Update status circuit breaker
	if err != nil {
		cb.RecordFailure(ctx)
		return err
	}

	cb.RecordSuccess(ctx)
	return nil
}

// AllowRequest memeriksa apakah permintaan diizinkan
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		// Dalam state tertutup, semua permintaan diizinkan
		return true
	case StateOpen:
		// Cek apakah sudah waktunya untuk mencoba lagi
		if now.After(cb.lastAttempt.Add(cb.config.Timeout)) {
			// Transisi ke half-open
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			cb.state = StateHalfOpen
			cb.halfOpenCalls = 0
			cb.mutex.Unlock()
			cb.mutex.RLock()
			return true
		}
		// Masih dalam timeout, tolak permintaan
		return false
	case StateHalfOpen:
		// Izinkan beberapa permintaan untuk mencoba
		return cb.halfOpenCalls < cb.config.MaxHalfOpenCalls
	default:
		return true
	}
}

// RecordFailure mencatat kegagalan
func (cb *CircuitBreaker) RecordFailure(ctx context.Context) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	cb.lastAttempt = now

	// Reset counter jika sudah melewati waktu reset
	if cb.state == StateClosed && now.After(cb.lastAttempt.Add(cb.config.ResetCountTimeout)) {
		cb.failures = 0
	}

	// Menambah counter kegagalan
	cb.failures++

	// Update state berdasarkan kondisi
	switch cb.state {
	case StateClosed:
		// Jika kegagalan melebihi threshold, buka circuit
		if cb.failures >= cb.config.Threshold {
			cb.state = StateOpen
			cb.logger.Warn(ctx, "Circuit breaker berubah ke state OPEN", telemetry.Fields{
				"name":      cb.name,
				"failures":  cb.failures,
				"threshold": cb.config.Threshold,
			})
		}
	case StateHalfOpen:
		// Jika gagal dalam state half-open, kembali ke state open
		cb.state = StateOpen
		cb.logger.Warn(ctx, "Circuit breaker kembali ke state OPEN dari HALF-OPEN", telemetry.Fields{
			"name": cb.name,
		})
	}

	if cb.state == StateHalfOpen {
		cb.halfOpenCalls++
	}
}

// RecordSuccess mencatat keberhasilan
func (cb *CircuitBreaker) RecordSuccess(ctx context.Context) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	cb.lastAttempt = now

	switch cb.state {
	case StateHalfOpen:
		// Jika berhasil dalam state half-open, kembali ke state closed
		cb.state = StateClosed
		cb.failures = 0
		cb.logger.Info(ctx, "Circuit breaker berubah ke state CLOSED dari HALF-OPEN", telemetry.Fields{
			"name": cb.name,
		})
	case StateClosed:
		// Reset counter jika sudah melewati waktu reset
		if now.After(cb.lastAttempt.Add(cb.config.ResetCountTimeout)) {
			cb.failures = 0
		}
	}

	if cb.state == StateHalfOpen {
		cb.halfOpenCalls++
	}
}

// GetState mengembalikan status circuit breaker saat ini
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetMetrics mengembalikan metrik circuit breaker
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"name":            cb.name,
		"state":           string(cb.state),
		"failures":        cb.failures,
		"threshold":       cb.config.Threshold,
		"timeout":         cb.config.Timeout.String(),
		"reset_timeout":   cb.config.ResetCountTimeout.String(),
		"last_attempt_at": cb.lastAttempt.Format(time.RFC3339),
	}
}
