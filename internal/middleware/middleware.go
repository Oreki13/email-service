package middleware

import (
	"context"
	"email-service/internal/domain"
	"email-service/internal/dto"
	"email-service/pkg/telemetry"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ContextKey adalah tipe kustom untuk key context untuk menghindari collision
type ContextKey string

const (
	// XRequestIDHeader adalah nama header untuk request ID
	XRequestIDHeader = "X-Request-ID"
	// TraceIDKey adalah kunci untuk menyimpan trace ID di konteks
	TraceIDKey ContextKey = "traceID"
	// APIKeyContextKey adalah kunci untuk menyimpan API key di konteks
	APIKeyContextKey ContextKey = "apiKey"
)

// RateLimiterConfig adalah konfigurasi untuk middleware rate limiter
type RateLimiterConfig struct {
	// Redis client untuk menyimpan data rate limiting
	RedisClient *redis.Client
	// Max adalah maksimal request dalam durasi tertentu (default: 100)
	Max int
	// Duration adalah durasi untuk menghitung rate limit (default: 1 menit)
	Duration time.Duration
	// Key adalah fungsi untuk menghasilkan kunci rate limit (default: berdasarkan IP)
	Key func(*fiber.Ctx) string
	// Logger untuk mencatat pelanggaran rate limit
	Logger telemetry.Logger
	// SkipSuccessLog untuk melewati logging ketika request berhasil melewati rate limiter
	SkipSuccessLog bool
}

// NewRateLimiterConfig membuat konfigurasi default untuk rate limiter
func NewRateLimiterConfig(redisClient *redis.Client, logger telemetry.Logger) RateLimiterConfig {
	return RateLimiterConfig{
		RedisClient:    redisClient,
		Max:            100,
		Duration:       time.Minute,
		Logger:         logger,
		SkipSuccessLog: true,
		Key: func(c *fiber.Ctx) string {
			// Default menggunakan IP sebagai kunci
			return "ratelimit:" + c.IP()
		},
	}
}

// RequestID menambahkan request ID ke setiap request jika belum ada
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Cek apakah request ID sudah ada
		requestID := c.Get(XRequestIDHeader)
		if requestID == "" {
			// Jika tidak ada, buat request ID baru
			requestID = GenerateTraceID()
			c.Set(XRequestIDHeader, requestID)
		}

		// Simpan request ID ke konteks
		c.Locals(TraceIDKey, requestID)

		// Lanjutkan ke handler berikutnya
		return c.Next()
	}
}

// GenerateTraceID menghasilkan trace ID baru menggunakan UUID
func GenerateTraceID() string {
	return uuid.New().String()
}

// AuthMiddleware melakukan validasi API key
func AuthMiddleware(apiKeyRepo domain.APIKeyRepository, apiKeyHeader string, logger telemetry.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		traceID := c.Locals(TraceIDKey).(string)
		ctxWithTrace := context.WithValue(ctx, TraceIDKey, traceID)

		// Cek API key
		apiKey := c.Get(apiKeyHeader)
		if apiKey == "" {
			logger.WithField("traceID", traceID).Info(ctxWithTrace, "Missing API key in request")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   domain.UnauthorizedError("Missing API key").Error(),
				"traceID": traceID,
			})
		}

		// Validasi API key dari database
		foundKey, err := apiKeyRepo.FindByKey(ctxWithTrace, apiKey)
		if err != nil {
			logger.WithField("traceID", traceID).WithError(err).Error(ctxWithTrace, "Error validating API key")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   domain.InternalError("Failed to validate API key", err).Error(),
				"traceID": traceID,
			})
		}

		// Jika API key tidak ditemukan
		if foundKey == nil {
			logger.WithField("traceID", traceID).Info(ctxWithTrace, "Invalid API key provided")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   domain.UnauthorizedError("Invalid API key").Error(),
				"traceID": traceID,
			})
		}

		// Cek apakah API key aktif
		if !foundKey.IsActive {
			logger.WithField("traceID", traceID).WithField("apiKeyID", foundKey.ID).Info(ctxWithTrace, "Inactive API key used")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   domain.UnauthorizedError("API key is inactive").Error(),
				"traceID": traceID,
			})
		}

		// Cek apakah API key sudah expired
		if foundKey.ExpiresAt != nil && time.Now().After(*foundKey.ExpiresAt) {
			logger.WithField("traceID", traceID).WithField("apiKeyID", foundKey.ID).Info(ctxWithTrace, "Expired API key used")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   domain.UnauthorizedError("API key has expired").Error(),
				"traceID": traceID,
			})
		}

		// Update last used timestamp (secara asynchronous)
		go func(keyID string) {
			// Gunakan context dengan timeout untuk mencegah hanging
			ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Error diabaikan karena ini operasi asynchronous
			_ = apiKeyRepo.UpdateLastUsed(ctxWithTimeout, keyID)
		}(foundKey.ID)

		// Simpan API key di konteks untuk digunakan oleh handler
		c.Locals(APIKeyContextKey, foundKey)
		logger.WithField("traceID", traceID).WithField("apiKeyID", foundKey.ID).
			WithField("service", foundKey.ServiceName).
			Info(ctxWithTrace, "Authenticated request with API key")

		return c.Next()
	}
}

// RateLimiter middleware untuk membatasi jumlah request dari client
func RateLimiter(config RateLimiterConfig) fiber.Handler {
	// Validasi konfigurasi
	if config.RedisClient == nil {
		panic("Redis client is required for rate limiter")
	}
	if config.Max <= 0 {
		config.Max = 100
	}
	if config.Duration <= 0 {
		config.Duration = time.Minute
	}
	if config.Key == nil {
		config.Key = func(c *fiber.Ctx) string {
			return "ratelimit:" + c.IP()
		}
	}
	if config.Logger == nil {
		panic("Logger is required for rate limiter")
	}

	// Durasi dalam detik
	seconds := int(config.Duration.Seconds())

	// Return middleware handler
	return func(c *fiber.Ctx) error {
		// Ambil context dari request
		ctx := c.Context()

		// Dapatkan trace ID dari konteks jika ada
		var traceID string
		if tID, ok := c.Locals(TraceIDKey).(string); ok {
			traceID = tID
		} else {
			traceID = GenerateTraceID()
		}

		// Buat context dengan trace ID
		ctxWithTrace := context.WithValue(ctx, TraceIDKey, traceID)

		// Hitung key untuk rate limiting berdasarkan fungsi yang diberikan
		key := config.Key(c)

		// Cek jika API key tersedia, gunakan itu sebagai bagian dari key
		if apiKey, ok := c.Locals(APIKeyContextKey).(*domain.APIKey); ok && apiKey != nil {
			key = fmt.Sprintf("%s:%s", key, apiKey.ID)
		}

		// Script Lua untuk atomic rate limiting (sliding window)
		script := redis.NewScript(`
			local key = KEYS[1]
			local max = tonumber(ARGV[1])
			local window = tonumber(ARGV[2])
			local now = tonumber(ARGV[3])

			-- Hapus semua timestamp yang lebih lama dari window
			redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

			-- Dapatkan jumlah request dalam window
			local count = redis.call('ZCARD', key)

			-- Jika masih di bawah limit, tambahkan timestamp dan return count
			if count < max then
				redis.call('ZADD', key, now, now .. ':' .. math.random())
				redis.call('EXPIRE', key, window)
				return {count + 1, max - (count + 1), max}
			end

			-- Jika melebihi limit, return count dan 0 remaining
			return {count, 0, max}
		`)

		// Eksekusi script Lua
		now := time.Now().Unix()
		result, err := script.Run(
			ctxWithTrace,
			config.RedisClient,
			[]string{key},
			config.Max,
			seconds,
			now,
		).Int64Slice()

		// Handle error eksekusi Redis
		if err != nil {
			config.Logger.WithField("traceID", traceID).WithError(err).Error(ctxWithTrace, "Rate limiter Redis error")
			// Jika Redis gagal, izinkan request (fail open) tetapi log error
			return c.Next()
		}

		// Parse hasil dari script Lua
		current := result[0]
		remaining := result[1]
		limit := result[2]

		// Set header rate limit
		c.Set("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
		c.Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(now+int64(seconds), 10))

		// Jika melebihi limit, return 429 Too Many Requests
		if current > limit {
			logFields := telemetry.Fields{
				"traceID":   traceID,
				"ip":        c.IP(),
				"path":      c.Path(),
				"method":    c.Method(),
				"limit":     limit,
				"remaining": 0,
				"key":       key,
			}

			// Log pelanggaran rate limit
			config.Logger.WithFields(logFields).Warn(ctxWithTrace, "Rate limit exceeded")

			// Return error 429 dengan format response global
			return c.Status(fiber.StatusTooManyRequests).JSON(dto.APIResponse{
				Status:  dto.StatusError,
				TraceID: traceID,
				Message: domain.RateLimitError("Rate limit exceeded. Try again later.").Error(),
				Data: fiber.Map{
					"limit":     limit,
					"remaining": 0,
					"reset":     now + int64(seconds),
				},
			})
		}

		// Log informasi rate limit jika tidak dilewati
		if !config.SkipSuccessLog {
			logFields := telemetry.Fields{
				"traceID":   traceID,
				"ip":        c.IP(),
				"path":      c.Path(),
				"method":    c.Method(),
				"limit":     limit,
				"current":   current,
				"remaining": remaining,
				"key":       key,
			}
			config.Logger.WithFields(logFields).Debug(ctxWithTrace, "Rate limit check passed")
		}

		// Lanjutkan ke handler berikutnya
		return c.Next()
	}
}

// Logger middleware untuk logging request dan response
func Logger() fiber.Handler {
	// TODO: Implementasi logging dengan AWS CloudWatch
	return func(c *fiber.Ctx) error {
		// Implementasi logger akan dibuat nanti
		return c.Next()
	}
}

// RequestLogger middleware untuk mencatat request dan response
func RequestLogger(logger telemetry.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Dapatkan trace ID dari konteks
		traceID := c.Locals(TraceIDKey).(string)

		// Catat waktu mulai request
		start := time.Now()

		// Buat context dengan trace ID
		ctx := context.WithValue(c.Context(), TraceIDKey, traceID)

		// Catat detail request
		logger.WithFields(telemetry.Fields{
			"traceID":     traceID,
			"method":      c.Method(),
			"path":        c.Path(),
			"ip":          c.IP(),
			"userAgent":   c.Get("User-Agent"),
			"requestTime": start.Format(time.RFC3339),
		}).Info(ctx, "Incoming request")

		// Lanjutkan ke handler berikutnya
		err := c.Next()

		// Hitung durasi request
		duration := time.Since(start)

		// Catat detail response
		statusCode := c.Response().StatusCode()
		logFields := telemetry.Fields{
			"traceID":     traceID,
			"method":      c.Method(),
			"path":        c.Path(),
			"statusCode":  statusCode,
			"duration_ms": duration.Milliseconds(),
		}

		// Tentukan level log berdasarkan status code
		if statusCode >= 500 {
			logger.WithFields(logFields).Error(ctx, "Request error")
		} else if statusCode >= 400 {
			logger.WithFields(logFields).Warn(ctx, "Request warning")
		} else {
			logger.WithFields(logFields).Info(ctx, "Request completed")
		}

		return err
	}
}
