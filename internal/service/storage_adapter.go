package service

import (
	"context"
	"email-service/internal/config"
	"email-service/pkg/telemetry"
	"fmt"
)

// StorageAdapter adalah adapter untuk mengakses provider penyimpanan
type StorageAdapter struct {
	providers      map[string]StorageProvider
	activeProvider StorageProvider
	logger         telemetry.Logger
}

// StorageConfig berisi konfigurasi untuk storage adapter
type StorageConfig struct {
	DefaultProvider string
	LocalPath       string
	S3Config        S3Config
	FirebaseConfig  FirebaseConfig
}

// S3Config berisi konfigurasi untuk S3 storage
type S3Config struct {
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	BaseURL   string
}

// FirebaseConfig berisi konfigurasi untuk Firebase storage
type FirebaseConfig struct {
	ProjectID   string
	Bucket      string
	CredFile    string
	StoragePath string
}

// NewStorageAdapter membuat instance baru storage adapter
func NewStorageAdapter(cfg *config.Config, logger telemetry.Logger) (*StorageAdapter, error) {
	adapter := &StorageAdapter{
		providers: make(map[string]StorageProvider),
		logger:    logger,
	}

	// Berdasarkan konfigurasi, inisialisasi provider yang sesuai
	storageType := cfg.Storage.Provider

	switch storageType {
	case "local":
		localProvider, err := NewLocalStorageProvider(cfg.Storage.Local.Path, logger)
		if err != nil {
			return nil, fmt.Errorf("gagal menginisialisasi local storage provider: %w", err)
		}
		adapter.providers["local"] = localProvider
		adapter.activeProvider = localProvider

	case "s3":
		s3Provider, err := NewS3StorageProvider(S3Config{
			Region:    cfg.Storage.S3.Region,
			Bucket:    cfg.Storage.S3.Bucket,
			AccessKey: cfg.Storage.S3.AccessKey,
			SecretKey: cfg.Storage.S3.SecretKey,
			BaseURL:   cfg.Storage.S3.BaseURL,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("gagal menginisialisasi S3 storage provider: %w", err)
		}
		adapter.providers["s3"] = s3Provider
		adapter.activeProvider = s3Provider

	case "firebase":
		firebaseProvider, err := NewFirebaseStorageProvider(FirebaseConfig{
			ProjectID:   cfg.Storage.Firebase.ProjectID,
			Bucket:      cfg.Storage.Firebase.Bucket,
			CredFile:    cfg.Storage.Firebase.CredFile,
			StoragePath: cfg.Storage.Firebase.StoragePath,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("gagal menginisialisasi Firebase storage provider: %w", err)
		}
		adapter.providers["firebase"] = firebaseProvider
		adapter.activeProvider = firebaseProvider

	default:
		return nil, fmt.Errorf("provider penyimpanan tidak valid: %s", storageType)
	}

	logger.Info(context.Background(), "Storage adapter berhasil diinisialisasi", telemetry.Fields{
		"provider": storageType,
	})

	return adapter, nil
}

// GetProvider mendapatkan provider berdasarkan nama
func (a *StorageAdapter) GetProvider(name string) (StorageProvider, error) {
	provider, ok := a.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider tidak ditemukan: %s", name)
	}
	return provider, nil
}

// GetActiveProvider mendapatkan provider yang aktif
func (a *StorageAdapter) GetActiveProvider() StorageProvider {
	return a.activeProvider
}

// SaveFile menyimpan file menggunakan provider aktif
func (a *StorageAdapter) SaveFile(ctx context.Context, data []byte, filename string, contentType string) (string, error) {
	return a.activeProvider.SaveFile(ctx, data, filename, contentType)
}

// Name mengembalikan nama provider aktif
func (a *StorageAdapter) Name() string {
	return a.activeProvider.Name()
}
