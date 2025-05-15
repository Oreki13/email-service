package service

import (
	"context"
	"email-service/pkg/telemetry"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LocalStorageProvider adalah implementasi StorageProvider untuk penyimpanan lokal
type LocalStorageProvider struct {
	basePath string
	baseURL  string
	logger   telemetry.Logger
}

// NewLocalStorageProvider membuat instance baru LocalStorageProvider
func NewLocalStorageProvider(basePath string, logger telemetry.Logger) (*LocalStorageProvider, error) {
	// Pastikan direktori ada dan bisa diakses
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori penyimpanan: %w", err)
	}

	logger.Info(context.Background(), "Local storage provider diinisialisasi", telemetry.Fields{
		"base_path": basePath,
	})

	return &LocalStorageProvider{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// WithBaseURL menambahkan baseURL ke provider
func (p *LocalStorageProvider) WithBaseURL(baseURL string) *LocalStorageProvider {
	p.baseURL = baseURL
	return p
}

// SaveFile menyimpan file ke penyimpanan lokal
func (p *LocalStorageProvider) SaveFile(ctx context.Context, data []byte, filename string, contentType string) (string, error) {
	// Buat struktur folder berdasarkan tanggal
	currentTime := time.Now()
	datePath := fmt.Sprintf("%04d/%02d/%02d", currentTime.Year(), currentTime.Month(), currentTime.Day())
	dirPath := filepath.Join(p.basePath, datePath)

	// Pastikan folder ada
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori untuk file: %w", err)
	}

	// Buat nama file unik dengan UUID
	fileExt := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, fileExt)
	uniqueID := uuid.New().String()
	uniqueFilename := fmt.Sprintf("%s-%s%s", baseName, uniqueID, fileExt)

	// Path lengkap untuk file
	filePath := filepath.Join(dirPath, uniqueFilename)
	relativePath := filepath.Join(datePath, uniqueFilename)

	// Simpan file
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("gagal menyimpan file: %w", err)
	}

	p.logger.Info(ctx, "File berhasil disimpan ke local storage", telemetry.Fields{
		"filename":     filename,
		"path":         filePath,
		"size":         len(data),
		"content_type": contentType,
	})

	// Kembalikan URL atau path relatif
	if p.baseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(p.baseURL, "/"), relativePath), nil
	}

	// Jika tidak ada baseURL, kembalikan path relatif
	return relativePath, nil
}

// Name mengembalikan nama provider
func (p *LocalStorageProvider) Name() string {
	return "local"
}
