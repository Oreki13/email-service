package cache

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCacheService adalah implementasi caching menggunakan Redis
type RedisCacheService struct {
	client *redis.Client
	logger telemetry.Logger
	ttl    time.Duration
}

// RedisCacheConfig merupakan konfigurasi untuk Redis cache
type RedisCacheConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration
}

// NewRedisCacheService membuat instance baru dari RedisCacheService
func NewRedisCacheService(config RedisCacheConfig, logger telemetry.Logger) (*RedisCacheService, error) {
	// Buat klien Redis
	client := redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + string(rune(config.Port)),
		Password: config.Password,
		DB:       config.DB,
	})

	// Uji koneksi
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		logger.Error(ctx, "Gagal terhubung ke Redis", telemetry.Fields{
			"error": err.Error(),
			"host":  config.Host,
			"port":  config.Port,
		})
		return nil, err
	}

	// Default TTL jika tidak diatur
	ttl := config.TTL
	if ttl == 0 {
		ttl = 1 * time.Hour
	}

	return &RedisCacheService{
		client: client,
		logger: logger,
		ttl:    ttl,
	}, nil
}

// SetTemplate menyimpan template ke cache
func (r *RedisCacheService) SetTemplate(ctx context.Context, template *domain.Template) error {
	// Buat key untuk template
	key := "template:" + template.ID

	// Marshal template ke JSON
	jsonData, err := json.Marshal(template)
	if err != nil {
		r.logger.Error(ctx, "Gagal marshal template ke JSON", telemetry.Fields{
			"error":       err.Error(),
			"template_id": template.ID,
		})
		return err
	}

	// Simpan ke Redis dengan TTL
	if err := r.client.Set(ctx, key, jsonData, r.ttl).Err(); err != nil {
		r.logger.Error(ctx, "Gagal menyimpan template ke Redis", telemetry.Fields{
			"error":       err.Error(),
			"template_id": template.ID,
		})
		return err
	}

	r.logger.Info(ctx, "Template berhasil disimpan ke cache", telemetry.Fields{
		"template_id": template.ID,
		"ttl":         r.ttl.String(),
	})

	return nil
}

// GetTemplate mendapatkan template dari cache
func (r *RedisCacheService) GetTemplate(ctx context.Context, id string) (*domain.Template, error) {
	// Buat key untuk template
	key := "template:" + id

	// Ambil dari Redis
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Cache miss
			r.logger.Info(ctx, "Template tidak ditemukan di cache", telemetry.Fields{
				"template_id": id,
			})
			return nil, nil
		}
		// Error lain
		r.logger.Error(ctx, "Gagal mendapatkan template dari Redis", telemetry.Fields{
			"error":       err.Error(),
			"template_id": id,
		})
		return nil, err
	}

	// Unmarshal JSON ke struct template
	var template domain.Template
	if err := json.Unmarshal([]byte(val), &template); err != nil {
		r.logger.Error(ctx, "Gagal unmarshal JSON template", telemetry.Fields{
			"error":       err.Error(),
			"template_id": id,
		})
		return nil, err
	}

	r.logger.Info(ctx, "Template berhasil diambil dari cache", telemetry.Fields{
		"template_id": id,
	})

	return &template, nil
}

// DeleteTemplate menghapus template dari cache
func (r *RedisCacheService) DeleteTemplate(ctx context.Context, id string) error {
	// Buat key untuk template
	key := "template:" + id

	// Hapus dari Redis
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.Error(ctx, "Gagal menghapus template dari Redis", telemetry.Fields{
			"error":       err.Error(),
			"template_id": id,
		})
		return err
	}

	r.logger.Info(ctx, "Template berhasil dihapus dari cache", telemetry.Fields{
		"template_id": id,
	})

	return nil
}

// IncrementCounter menambah counter untuk rate limiting
func (r *RedisCacheService) IncrementCounter(ctx context.Context, key string, expiry time.Duration) (int64, error) {
	// Tambah counter
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		r.logger.Error(ctx, "Gagal menambah counter di Redis", telemetry.Fields{
			"error": err.Error(),
			"key":   key,
		})
		return 0, err
	}

	// Set expiry jika counter baru
	if val == 1 {
		if err := r.client.Expire(ctx, key, expiry).Err(); err != nil {
			r.logger.Error(ctx, "Gagal mengatur expiry counter di Redis", telemetry.Fields{
				"error": err.Error(),
				"key":   key,
			})
			// Tidak mengembalikan error karena counter sudah berhasil dibuat
		}
	}

	return val, nil
}

// SetWithExpiry menyimpan nilai dengan expiry
func (r *RedisCacheService) SetWithExpiry(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	// Marshal value ke JSON jika bukan string
	var data string
	switch v := value.(type) {
	case string:
		data = v
	default:
		jsonData, err := json.Marshal(v)
		if err != nil {
			r.logger.Error(ctx, "Gagal marshal data ke JSON", telemetry.Fields{
				"error": err.Error(),
				"key":   key,
			})
			return err
		}
		data = string(jsonData)
	}

	// Simpan ke Redis dengan expiry
	if err := r.client.Set(ctx, key, data, expiry).Err(); err != nil {
		r.logger.Error(ctx, "Gagal menyimpan data ke Redis", telemetry.Fields{
			"error": err.Error(),
			"key":   key,
		})
		return err
	}

	return nil
}

// Get mendapatkan nilai dari cache
func (r *RedisCacheService) Get(ctx context.Context, key string) (string, error) {
	// Ambil dari Redis
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Cache miss
			return "", nil
		}
		// Error lain
		r.logger.Error(ctx, "Gagal mendapatkan data dari Redis", telemetry.Fields{
			"error": err.Error(),
			"key":   key,
		})
		return "", err
	}

	return val, nil
}

// Close menutup koneksi Redis
func (r *RedisCacheService) Close() error {
	return r.client.Close()
}
