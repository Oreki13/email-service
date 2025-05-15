package handler

import (
	"context"
	"database/sql"
	"email-service/internal/dto"
	"email-service/pkg/telemetry"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// HealthStatus menggambarkan status kesehatan komponen
type HealthStatus string

const (
	// StatusUp menunjukkan komponen berjalan normal
	StatusUp HealthStatus = "UP"
	// StatusDown menunjukkan komponen tidak berjalan
	StatusDown HealthStatus = "DOWN"
	// StatusDegraded menunjukkan komponen berjalan tapi tidak optimal
	StatusDegraded HealthStatus = "DEGRADED"
)

// ComponentHealth merepresentasikan status kesehatan suatu komponen
type ComponentHealth struct {
	Status      HealthStatus   `json:"status"`
	Details     map[string]any `json:"details,omitempty"`
	LastChecked time.Time      `json:"lastChecked"`
}

// HealthHandler menangani endpoint health check dan metrics
type HealthHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	logger      telemetry.Logger
	startTime   time.Time
}

// NewHealthHandler membuat instance baru health handler
func NewHealthHandler(db *sql.DB, redisClient *redis.Client, logger telemetry.Logger) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		startTime:   time.Now(),
	}
}

// RegisterRoutes mendaftarkan route untuk health handler
func (h *HealthHandler) RegisterRoutes(router fiber.Router) {
	// Health check endpoint
	router.Get("/health", h.HealthCheck)
	router.Get("/health/live", h.LivenessCheck)
	router.Get("/health/ready", h.ReadinessCheck)

	// Metrics endpoint
	router.Get("/metrics", h.Metrics)
}

// HealthCheck menangani request untuk memeriksa kesehatan seluruh sistem
func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	ctx := c.Context()
	h.logger.Debug(ctx, "Menerima request health check", nil)

	// Cek semua komponen
	dbHealth := h.checkDatabaseHealth(ctx)
	redisHealth := h.checkRedisHealth(ctx)

	// Tentukan status keseluruhan
	overallStatus := StatusUp
	if dbHealth.Status == StatusDown || redisHealth.Status == StatusDown {
		overallStatus = StatusDown
	} else if dbHealth.Status == StatusDegraded || redisHealth.Status == StatusDegraded {
		overallStatus = StatusDegraded
	}

	// Buat response
	healthResponse := map[string]any{
		"status":    overallStatus,
		"timestamp": time.Now().Format(time.RFC3339),
		"components": map[string]ComponentHealth{
			"database": dbHealth,
			"redis":    redisHealth,
		},
		"version":   "1.0.0", // Hardcoded sementara, idealnya ambil dari env atau build info
		"uptime":    time.Since(h.startTime).String(),
		"goVersion": runtime.Version(),
	}

	return c.JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		Message: "Health check completed",
		Data:    healthResponse,
	})
}

// LivenessCheck untuk Kubernetes liveness probe
func (h *HealthHandler) LivenessCheck(c *fiber.Ctx) error {
	// Liveness hanya mengecek apakah aplikasi berjalan
	// Tidak perlu cek database atau redis
	return c.JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		Message: "Service is live",
		Data: map[string]any{
			"status":    "UP",
			"timestamp": time.Now().Format(time.RFC3339),
			"uptime":    time.Since(h.startTime).String(),
		},
	})
}

// ReadinessCheck untuk Kubernetes readiness probe
func (h *HealthHandler) ReadinessCheck(c *fiber.Ctx) error {
	ctx := c.Context()

	// Cek semua komponen
	dbHealth := h.checkDatabaseHealth(ctx)
	redisHealth := h.checkRedisHealth(ctx)

	// Service siap jika semua komponen UP
	if dbHealth.Status == StatusUp && redisHealth.Status == StatusUp {
		return c.JSON(dto.APIResponse{
			Status:  dto.StatusSuccess,
			Message: "Service is ready",
			Data: map[string]any{
				"status":    "UP",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
	}

	// Jika tidak siap, kirim 503 Service Unavailable
	return c.Status(fiber.StatusServiceUnavailable).JSON(dto.APIResponse{
		Status:  dto.StatusError,
		Message: "Service is not ready",
		Data: map[string]any{
			"status":    "DOWN",
			"timestamp": time.Now().Format(time.RFC3339),
			"reason":    "One or more components are down",
		},
	})
}

// Metrics menangani request untuk metrics
func (h *HealthHandler) Metrics(c *fiber.Ctx) error {
	ctx := c.Context()
	h.logger.Debug(ctx, "Menerima request metrics", nil)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(h.startTime).String(),
		"memory": map[string]any{
			"alloc":      m.Alloc,
			"totalAlloc": m.TotalAlloc,
			"sys":        m.Sys,
			"numGC":      m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
	}

	return c.JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		Message: "Metrics data",
		Data:    metrics,
	})
}

// checkDatabaseHealth memeriksa kesehatan database
func (h *HealthHandler) checkDatabaseHealth(ctx context.Context) ComponentHealth {
	startTime := time.Now()
	details := make(map[string]any)

	// Cek apakah DB null
	if h.db == nil {
		return ComponentHealth{
			Status:      StatusDown,
			Details:     map[string]any{"error": "database connection not initialized"},
			LastChecked: time.Now(),
		}
	}

	// Cek koneksi database dengan ping
	err := h.db.PingContext(ctx)
	pingTime := time.Since(startTime)

	details["responseTime"] = pingTime.String()

	if err != nil {
		details["error"] = err.Error()
		h.logger.Error(ctx, "Database health check failed", telemetry.Fields{
			"error": err.Error(),
		})
		return ComponentHealth{
			Status:      StatusDown,
			Details:     details,
			LastChecked: time.Now(),
		}
	}

	// Eksekusi query sederhana untuk validasi lebih lanjut
	_, err = h.db.ExecContext(ctx, "SELECT 1")
	if err != nil {
		details["error"] = err.Error()
		h.logger.Error(ctx, "Database query check failed", telemetry.Fields{
			"error": err.Error(),
		})
		return ComponentHealth{
			Status:      StatusDown,
			Details:     details,
			LastChecked: time.Now(),
		}
	}

	// Periksa apakah responsetime terlalu lama
	if pingTime > 1*time.Second {
		details["warning"] = "database response time is high"
		h.logger.Warn(ctx, "Database response time is high", telemetry.Fields{
			"responseTime": pingTime.String(),
		})
		return ComponentHealth{
			Status:      StatusDegraded,
			Details:     details,
			LastChecked: time.Now(),
		}
	}

	return ComponentHealth{
		Status:      StatusUp,
		Details:     details,
		LastChecked: time.Now(),
	}
}

// checkRedisHealth memeriksa kesehatan Redis
func (h *HealthHandler) checkRedisHealth(ctx context.Context) ComponentHealth {
	startTime := time.Now()
	details := make(map[string]any)

	// Cek apakah Redis null
	if h.redisClient == nil {
		return ComponentHealth{
			Status:      StatusDown,
			Details:     map[string]any{"error": "redis client not initialized"},
			LastChecked: time.Now(),
		}
	}

	// Cek koneksi Redis dengan ping
	ctxTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	_, err := h.redisClient.Ping(ctxTimeout).Result()
	pingTime := time.Since(startTime)

	details["responseTime"] = pingTime.String()

	if err != nil {
		details["error"] = err.Error()
		h.logger.Error(ctx, "Redis health check failed", telemetry.Fields{
			"error": err.Error(),
		})
		return ComponentHealth{
			Status:      StatusDown,
			Details:     details,
			LastChecked: time.Now(),
		}
	}

	// Periksa apakah responsetime terlalu lama
	if pingTime > 500*time.Millisecond {
		details["warning"] = "redis response time is high"
		h.logger.Warn(ctx, "Redis response time is high", telemetry.Fields{
			"responseTime": pingTime.String(),
		})
		return ComponentHealth{
			Status:      StatusDegraded,
			Details:     details,
			LastChecked: time.Now(),
		}
	}

	return ComponentHealth{
		Status:      StatusUp,
		Details:     details,
		LastChecked: time.Now(),
	}
}
