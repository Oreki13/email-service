// Package storage menyediakan abstraksi untuk menyimpan file attachment
package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"email-service/internal/config"
	"email-service/pkg/telemetry"
)

// LocalStorage adalah implementasi Storage menggunakan sistem file lokal
type LocalStorage struct {
	basePath     string
	baseURL      string
	maxSize      int64
	allowedTypes []string
	logger       telemetry.Logger
}

// NewLocalStorage membuat instance baru dari LocalStorage
func NewLocalStorage(cfg *config.Config, logger telemetry.Logger) (*LocalStorage, error) {
	basePath := cfg.Storage.Local.Path

	// Buat direktori jika belum ada
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori penyimpanan: %w", err)
	}

	return &LocalStorage{
		basePath:     basePath,
		baseURL:      cfg.Storage.BaseURL,
		maxSize:      cfg.Storage.MaxSize,
		allowedTypes: cfg.Storage.AllowedTypes,
		logger:       logger,
	}, nil
}

// Upload mengupload file ke sistem file lokal
func (s *LocalStorage) Upload(ctx context.Context, file io.Reader, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
	logFields := telemetry.Fields{
		"filename":     filename,
		"content_type": contentType,
	}

	s.logger.Debug(ctx, "Memulai upload file ke penyimpanan lokal", logFields)

	// Validate content type
	if !ValidateFileType(contentType, s.allowedTypes) {
		s.logger.Warn(ctx, "Tipe file tidak diizinkan", logFields)
		return nil, ErrInvalidFileType
	}

	// Read the file into memory to get its size and for validation
	// Note: This approach works for small files but should be changed for large files
	data, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error(ctx, "Gagal membaca file", telemetry.Fields{
			"filename": filename,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("gagal membaca file: %w", err)
	}

	// Check file size
	fileSize := int64(len(data))
	if s.maxSize > 0 && fileSize > s.maxSize {
		s.logger.Warn(ctx, "Ukuran file melebihi batas maksimum", telemetry.Fields{
			"filename": filename,
			"size":     fileSize,
			"max_size": s.maxSize,
		})
		return nil, ErrFileTooLarge
	}

	// Generate storage path
	relPath := GeneratePathWithPrefix("", filename)
	absPath := filepath.Join(s.basePath, relPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		s.logger.Error(ctx, "Gagal membuat direktori", telemetry.Fields{
			"directory": filepath.Dir(absPath),
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("gagal membuat direktori: %w", err)
	}

	// Write file to disk
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		s.logger.Error(ctx, "Gagal menulis file", telemetry.Fields{
			"path":  absPath,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("gagal menulis file: %w", err)
	}

	// Create result
	result := &StorageResult{
		Filename:    filename,
		Path:        relPath,
		Size:        fileSize,
		ContentType: contentType,
		UploadedAt:  time.Now(),
		Metadata:    metadata,
	}

	// Add URL if baseURL is defined
	if s.baseURL != "" {
		result.URL = strings.TrimSuffix(s.baseURL, "/") + "/" + strings.TrimPrefix(relPath, "/")
	}

	s.logger.Info(ctx, "Berhasil mengupload file ke penyimpanan lokal", telemetry.Fields{
		"filename": filename,
		"path":     relPath,
		"size":     fileSize,
	})

	return result, nil
}

// UploadFromURL mengunduh file dari URL dan menyimpannya ke penyimpanan lokal
func (s *LocalStorage) UploadFromURL(ctx context.Context, url, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
	logFields := telemetry.Fields{
		"url":      url,
		"filename": filename,
	}

	s.logger.Debug(ctx, "Mengunduh file dari URL", logFields)

	// Create HTTP client with context
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		s.logger.Error(ctx, "Gagal membuat request HTTP", telemetry.Fields{
			"url":   url,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("gagal membuat request HTTP: %w", err)
	}

	// Get the file
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengunduh file", telemetry.Fields{
			"url":   url,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("gagal mengunduh file: %w", err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		s.logger.Error(ctx, "Server mengembalikan status error", telemetry.Fields{
			"url":    url,
			"status": resp.StatusCode,
		})
		return nil, fmt.Errorf("server mengembalikan status %d", resp.StatusCode)
	}

	// If content type is not provided, use the one from response
	if contentType == "" {
		contentType = resp.Header.Get("Content-Type")
	}

	// If filename is not provided, try to get it from Content-Disposition or URL
	if filename == "" {
		// Try Content-Disposition header
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			if _, params, err := mime.ParseMediaType(cd); err == nil {
				if fn, ok := params["filename"]; ok {
					filename = fn
				}
			}
		}

		// If still no filename, extract from URL
		if filename == "" {
			urlPath := strings.Split(url, "/")
			filename = urlPath[len(urlPath)-1]
			filename = strings.Split(filename, "?")[0] // Remove query string
		}
	}

	// Upload the file using the standard Upload method
	return s.Upload(ctx, resp.Body, filename, contentType, metadata)
}

// UploadFromBase64 mengupload file dari data base64 ke penyimpanan lokal
func (s *LocalStorage) UploadFromBase64(ctx context.Context, base64Data, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
	logFields := telemetry.Fields{
		"filename": filename,
	}

	s.logger.Debug(ctx, "Memproses data base64", logFields)

	// Decode base64 data
	// Handle common base64 formats with prefix like "data:image/jpeg;base64,"
	base64Data = strings.TrimSpace(base64Data)
	if idx := strings.Index(base64Data, ";base64,"); idx >= 0 {
		// Extract content type if not provided
		if contentType == "" {
			prefix := base64Data[:idx]
			if strings.HasPrefix(prefix, "data:") {
				contentType = prefix[5:]
			}
		}
		base64Data = base64Data[idx+8:] // 8 is len(";base64,")
	}

	// Decode base64 string
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		s.logger.Error(ctx, "Gagal mendekode data base64", telemetry.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("%w: %v", ErrBase64Decode, err)
	}

	// If content type not provided, try to detect it
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	// Create reader from data
	reader := bytes.NewReader(data)

	// Upload using standard method
	return s.Upload(ctx, reader, filename, contentType, metadata)
}

// Get mengambil file dari penyimpanan lokal
func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	absPath := filepath.Join(s.basePath, path)

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		s.logger.Warn(ctx, "File tidak ditemukan", telemetry.Fields{"path": path})
		return nil, ErrFileNotFound
	}

	// Open file
	file, err := os.Open(absPath)
	if err != nil {
		s.logger.Error(ctx, "Gagal membuka file", telemetry.Fields{
			"path":  path,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("gagal membuka file: %w", err)
	}

	return file, nil
}

// GetURL mendapatkan URL untuk mengakses file
func (s *LocalStorage) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	// For local storage, we just return baseURL + path if baseURL is defined
	if s.baseURL == "" {
		s.logger.Warn(ctx, "Base URL tidak dikonfigurasi", telemetry.Fields{"path": path})
		return "", fmt.Errorf("base URL tidak dikonfigurasi")
	}

	return strings.TrimSuffix(s.baseURL, "/") + "/" + strings.TrimPrefix(path, "/"), nil
}

// Delete menghapus file dari penyimpanan lokal
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	absPath := filepath.Join(s.basePath, path)

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		s.logger.Warn(ctx, "File tidak ditemukan untuk dihapus", telemetry.Fields{"path": path})
		return ErrFileNotFound
	}

	// Delete file
	if err := os.Remove(absPath); err != nil {
		s.logger.Error(ctx, "Gagal menghapus file", telemetry.Fields{
			"path":  path,
			"error": err.Error(),
		})
		return fmt.Errorf("gagal menghapus file: %w", err)
	}

	s.logger.Info(ctx, "Berhasil menghapus file", telemetry.Fields{"path": path})
	return nil
}

// Close menutup koneksi
func (s *LocalStorage) Close() error {
	// No connections to close for local storage
	return nil
}
