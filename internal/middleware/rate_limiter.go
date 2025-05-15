package middleware

import (
	"context"
	"email-service/internal/dto"
	"fmt"
	"strconv"
	"strings"
	"time"

	"email-service/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// RateLimiterRedis implementasi rate limiter menggunakan Redis
type RateLimiterRedis struct {
	redisClient *redis.Client
	logger      telemetry.Logger
	keyPrefix   string
	maxRequests int64
	window      time.Duration
}

// NewRateLimiterRedis membuat instance baru dari RateLimiterRedis
func NewRateLimiterRedis(redisClient *redis.Client, logger telemetry.Logger, keyPrefix string, maxRequests int, window time.Duration) *RateLimiterRedis {
	// Default values
	if keyPrefix == "" {
		keyPrefix = "ratelimit:"
	}
	if maxRequests <= 0 {
		maxRequests = 100
	}
	if window <= 0 {
		window = time.Minute
	}

	return &RateLimiterRedis{
		redisClient: redisClient,
		logger:      logger,
		keyPrefix:   keyPrefix,
		maxRequests: int64(maxRequests),
		window:      window,
	}
}

// RateLimitByIP menerapkan rate limiting berdasarkan IP address
func (r *RateLimiterRedis) RateLimitByIP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Dapatkan IP address
		ip := c.IP()
		if ip == "" {
			ip = "unknown"
		}

		// Dapatkan API key dari header jika ada
		apiKey := c.Get("X-API-Key", "")
		var key string
		if apiKey != "" {
			// Gunakan API key sebagai identifier
			key = fmt.Sprintf("%s:api:%s", r.keyPrefix, apiKey)
		} else {
			// Gunakan IP address sebagai identifier
			key = fmt.Sprintf("%s:ip:%s", r.keyPrefix, ip)
		}

		return r.rateLimitByKey(c, key)
	}
}

// RateLimitByAPIKey menerapkan rate limiting berdasarkan API key
func (r *RateLimiterRedis) RateLimitByAPIKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Dapatkan API key dari header
		apiKey := c.Get("X-API-Key", "")
		if apiKey == "" {
			// Jika tidak ada API key, skip rate limiting
			return c.Next()
		}

		// Key untuk rate limiting
		key := fmt.Sprintf("%s:api:%s", r.keyPrefix, apiKey)

		return r.rateLimitByKey(c, key)
	}
}

// RateLimitByEndpoint menerapkan rate limiting berdasarkan endpoint
func (r *RateLimiterRedis) RateLimitByEndpoint() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Dapatkan IP address
		ip := c.IP()
		if ip == "" {
			ip = "unknown"
		}

		// Dapatkan endpoint path
		path := c.Path()

		// Key untuk rate limiting
		key := fmt.Sprintf("%s:endpoint:%s:%s", r.keyPrefix, path, ip)

		return r.rateLimitByKey(c, key)
	}
}

// rateLimitByKey fungsi pembantu untuk menerapkan rate limiting berdasarkan key
func (r *RateLimiterRedis) rateLimitByKey(c *fiber.Ctx, key string) error {
	ctx := context.Background()

	// Cek sisa kuota
	var currentCount int64
	currentCount, err := r.redisClient.Get(ctx, key).Int64()
	if err != nil && err != redis.Nil {
		// Error saat mengambil data dari Redis
		r.logger.Error(ctx, "Gagal mengambil data rate limit dari Redis", telemetry.Fields{
			"error": err.Error(),
			"key":   key,
		})
		// Lanjutkan request jika terjadi error
		return c.Next()
	}

	// Jika key tidak ada, buat key baru
	if err == redis.Nil {
		currentCount = 0
		// Set key dengan TTL sesuai window
		if err := r.redisClient.Set(ctx, key, "1", r.window).Err(); err != nil {
			r.logger.Error(ctx, "Gagal membuat key rate limit di Redis", telemetry.Fields{
				"error": err.Error(),
				"key":   key,
			})
			// Lanjutkan request jika terjadi error
			return c.Next()
		}
	} else {
		// Increment key
		currentCount, err = r.redisClient.Incr(ctx, key).Result()
		if err != nil {
			r.logger.Error(ctx, "Gagal increment key rate limit di Redis", telemetry.Fields{
				"error": err.Error(),
				"key":   key,
			})
			// Lanjutkan request jika terjadi error
			return c.Next()
		}
	}

	// Dapatkan TTL untuk menghitung reset time
	ttl, err := r.redisClient.TTL(ctx, key).Result()
	if err != nil {
		r.logger.Error(ctx, "Gagal mendapatkan TTL key rate limit dari Redis", telemetry.Fields{
			"error": err.Error(),
			"key":   key,
		})
		// Set default TTL
		ttl = r.window
	}

	// Set header rate limit
	c.Set("X-RateLimit-Limit", strconv.FormatInt(r.maxRequests, 10))
	c.Set("X-RateLimit-Remaining", strconv.FormatInt(r.maxRequests-currentCount, 10))
	c.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

	// Jika melebihi batas, kembalikan error 429
	if currentCount > r.maxRequests {
		message := fmt.Sprintf("Rate limit terlampaui. Coba lagi setelah %s", ttl.Round(time.Second).String())

		traceID := c.Locals(TraceIDKey).(string)

		r.logger.Warn(ctx, "Rate limit terlampaui", telemetry.Fields{
			"key":     key,
			"count":   currentCount,
			"limit":   r.maxRequests,
			"ttl":     ttl.String(),
			"traceID": traceID,
		})

		return c.Status(fiber.StatusTooManyRequests).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: message,
			Data:    nil,
		})
	}

	// Lanjutkan ke handler berikutnya
	return c.Next()
}

// CustomRateLimiter menerapkan rate limiting dengan konfigurasi kustom
func (r *RateLimiterRedis) CustomRateLimiter(keyFunc func(*fiber.Ctx) string, maxRequests int, window time.Duration) fiber.Handler {
	var maxRequestsInt64 int64
	if maxRequests <= 0 {
		maxRequestsInt64 = r.maxRequests
	} else {
		maxRequestsInt64 = int64(maxRequests)
	}
	if window <= 0 {
		window = r.window
	}

	return func(c *fiber.Ctx) error {
		// Dapatkan key untuk rate limiting
		key := keyFunc(c)
		if key == "" {
			// Skip rate limiting jika key kosong
			return c.Next()
		}

		// Pastikan key menggunakan prefix
		if !strings.HasPrefix(key, r.keyPrefix) {
			key = r.keyPrefix + key
		}

		ctx := context.Background()

		// Cek sisa kuota
		var currentCount int64
		currentCount, err := r.redisClient.Get(ctx, key).Int64()
		if err != nil && err != redis.Nil {
			// Error saat mengambil data dari Redis
			r.logger.Error(ctx, "Gagal mengambil data rate limit dari Redis", telemetry.Fields{
				"error": err.Error(),
				"key":   key,
			})
			// Lanjutkan request jika terjadi error
			return c.Next()
		}

		// Jika key tidak ada, buat key baru
		if err == redis.Nil {
			currentCount = 0
			// Set key dengan TTL sesuai window
			if err := r.redisClient.Set(ctx, key, "1", window).Err(); err != nil {
				r.logger.Error(ctx, "Gagal membuat key rate limit di Redis", telemetry.Fields{
					"error": err.Error(),
					"key":   key,
				})
				// Lanjutkan request jika terjadi error
				return c.Next()
			}
		} else {
			// Increment key
			currentCount, err = r.redisClient.Incr(ctx, key).Result()
			if err != nil {
				r.logger.Error(ctx, "Gagal increment key rate limit di Redis", telemetry.Fields{
					"error": err.Error(),
					"key":   key,
				})
				// Lanjutkan request jika terjadi error
				return c.Next()
			}
		}

		// Dapatkan TTL untuk menghitung reset time
		ttl, err := r.redisClient.TTL(ctx, key).Result()
		if err != nil {
			r.logger.Error(ctx, "Gagal mendapatkan TTL key rate limit dari Redis", telemetry.Fields{
				"error": err.Error(),
				"key":   key,
			})
			// Set default TTL
			ttl = window
		}

		// Set header rate limit
		c.Set("X-RateLimit-Limit", strconv.FormatInt(maxRequestsInt64, 10))
		c.Set("X-RateLimit-Remaining", strconv.FormatInt(maxRequestsInt64-currentCount, 10))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

		// Jika melebihi batas, kembalikan error 429
		if currentCount > maxRequestsInt64 {
			message := fmt.Sprintf("Rate limit terlampaui. Coba lagi setelah %s", ttl.Round(time.Second).String())

			traceID := c.Locals(TraceIDKey).(string)

			r.logger.Warn(ctx, "Rate limit terlampaui", telemetry.Fields{
				"key":     key,
				"count":   currentCount,
				"limit":   maxRequestsInt64,
				"ttl":     ttl.String(),
				"traceID": traceID,
			})

			return c.Status(fiber.StatusTooManyRequests).JSON(dto.APIResponse{
				Status:  dto.StatusError,
				TraceID: traceID,
				Message: message,
				Data:    nil,
			})
		}

		// Lanjutkan ke handler berikutnya
		return c.Next()
	}
}
