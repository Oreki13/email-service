package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig berisi konfigurasi untuk koneksi Redis
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// RedisClient merupakan wrapper untuk klien Redis
type RedisClient struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisClient membuat instance baru dari RedisClient
func NewRedisClient(config RedisConfig) (*RedisClient, error) {
	// Buat Redis client options
	opt := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	}

	// Buat client Redis
	client := redis.NewClient(opt)

	// Tes koneksi
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping Redis untuk memastikan koneksi berhasil
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		config: config,
	}, nil
}

// GetClient mengembalikan instance Redis client
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Close menutup koneksi Redis
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping memeriksa koneksi Redis
func (r *RedisClient) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx).Result()
	return err
}

// Set menyimpan nilai ke Redis
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get mengambil nilai dari Redis
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Del menghapus nilai dari Redis
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Incr menambah nilai untuk key
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Decr mengurangi nilai untuk key
func (r *RedisClient) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

// Exists memeriksa apakah key ada
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	val, err := r.client.Exists(ctx, key).Result()
	return val > 0, err
}

// Expire menentukan waktu expired untuk key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL mendapatkan waktu expired untuk key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// FlushAll menghapus semua data di database (untuk testing)
func (r *RedisClient) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}
