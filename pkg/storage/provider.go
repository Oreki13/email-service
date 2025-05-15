// Package storage menyediakan abstraksi untuk menyimpan file attachment
package storage

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"email-service/internal/config"
	"email-service/pkg/telemetry"
)

// Errors
var (
	ErrInvalidProvider = errors.New("provider penyimpanan tidak valid")
	ErrInvalidFile     = errors.New("file tidak valid")
	ErrFileTooLarge    = errors.New("ukuran file terlalu besar")
	ErrInvalidFileType = errors.New("tipe file tidak diizinkan")
	ErrFileNotFound    = errors.New("file tidak ditemukan")
	ErrUploadFailed    = errors.New("gagal mengupload file")
	ErrDownloadFailed  = errors.New("gagal mengunduh file")
	ErrBase64Decode    = errors.New("gagal mendekode data base64")
	ErrNotImplemented  = errors.New("fitur belum diimplementasikan")
)

// StorageProvider adalah pabrik untuk membuat instance Storage
func NewStorageProvider(cfg *config.Config, logger telemetry.Logger) (Storage, error) {
	provider := Provider(cfg.Storage.Provider)
	switch provider {
	case ProviderLocal:
		return NewLocalStorage(cfg, logger)
	case ProviderS3:
		return NewS3Storage(cfg, logger)
	case ProviderFirebase:
		return NewFirebaseStorage(cfg, logger)
	default:
		return nil, ErrInvalidProvider
	}
}

// Utility functions

// ValidateFileType memeriksa apakah tipe file diizinkan
func ValidateFileType(contentType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		return true
	}

	for _, allowed := range allowedTypes {
		if contentType == allowed {
			return true
		}
		// Mendukung wildcard seperti "image/*"
		if strings.HasSuffix(allowed, "/*") {
			prefix := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(contentType, prefix+"/") {
				return true
			}
		}
	}

	return false
}

// DetectContentType mendeteksi tipe konten dari data
func DetectContentType(data []byte) string {
	return http.DetectContentType(data)
}

// GeneratePathWithPrefix menghasilkan path file dengan prefix
func GeneratePathWithPrefix(prefix, filename string) string {
	// Buat subfolder berdasarkan tanggal untuk menghindari terlalu banyak file dalam satu folder
	now := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", now.Year(), now.Month(), now.Day())

	// Bersihkan filename agar aman untuk sistem file
	safeFilename := filepath.Base(filepath.Clean(filename))

	// Buat path akhir
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/") + "/"
	}

	return fmt.Sprintf("%s%s/%s", prefix, datePath, safeFilename)
}
