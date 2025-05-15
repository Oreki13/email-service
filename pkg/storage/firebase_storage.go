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
	"strings"
	"time"

	"email-service/internal/config"
	"email-service/pkg/telemetry"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/storage"
	"google.golang.org/api/option"
)

// FirebaseStorage adalah implementasi Storage menggunakan Firebase Storage
type FirebaseStorage struct {
	app          *firebase.App
	client       *storage.Client
	bucket       string
	baseURL      string
	maxSize      int64
	allowedTypes []string
	logger       telemetry.Logger
}

// NewFirebaseStorage membuat instance baru dari FirebaseStorage
func NewFirebaseStorage(cfg *config.Config, logger telemetry.Logger) (*FirebaseStorage, error) {
	var (
		app    *firebase.App
		err    error
		opts   []option.ClientOption
		ctx    = context.Background()
		bucket = cfg.Storage.Firebase.Bucket
	)

	// Jika ServiceAccountJSON disediakan, gunakan itu
	if cfg.Storage.Firebase.ServiceAccountJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.Storage.Firebase.ServiceAccountJSON)))
	} else if cfg.Storage.Firebase.CredentialsFile != "" {
		// Jika file kredensial disediakan, gunakan itu
		opts = append(opts, option.WithCredentialsFile(cfg.Storage.Firebase.CredentialsFile))
	}

	// Konfigurasi Firebase
	firebaseConfig := &firebase.Config{
		StorageBucket: bucket,
		ProjectID:     cfg.Storage.Firebase.ProjectID,
	}

	// Inisialisasi aplikasi Firebase
	app, err = firebase.NewApp(ctx, firebaseConfig, opts...)
	if err != nil {
		return nil, fmt.Errorf("gagal menginisialisasi aplikasi Firebase: %w", err)
	}

	// Buat client storage
	client, err := app.Storage(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat client Firebase Storage: %w", err)
	}

	return &FirebaseStorage{
		app:          app,
		client:       client,
		bucket:       bucket,
		baseURL:      cfg.Storage.BaseURL,
		maxSize:      cfg.Storage.MaxSize,
		allowedTypes: cfg.Storage.AllowedTypes,
		logger:       logger,
	}, nil
}

// Upload mengupload file ke Firebase Storage
func (s *FirebaseStorage) Upload(ctx context.Context, file io.Reader, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
	logFields := telemetry.Fields{
		"filename":     filename,
		"content_type": contentType,
		"bucket":       s.bucket,
	}

	s.logger.Debug(ctx, "Memulai upload file ke Firebase Storage", logFields)

	// Validate content type
	if !ValidateFileType(contentType, s.allowedTypes) {
		s.logger.Warn(ctx, "Tipe file tidak diizinkan", logFields)
		return nil, ErrInvalidFileType
	}

	// Read the file into memory to get its size and validate
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
	storagePath := GeneratePathWithPrefix("attachments", filename)

	// Get default bucket
	bucket, err := s.client.DefaultBucket()
	if err != nil {
		s.logger.Error(ctx, "Gagal mendapatkan bucket default", telemetry.Fields{
			"bucket": s.bucket,
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("gagal mendapatkan bucket default: %w", err)
	}

	// Jika bucket dikonfigurasi, gunakan itu
	if s.bucket != "" {
		bucket, err = s.client.Bucket(s.bucket)
		if err != nil {
			s.logger.Error(ctx, "Gagal mendapatkan bucket yang dikonfigurasi", telemetry.Fields{
				"bucket": s.bucket,
				"error":  err.Error(),
			})
			return nil, fmt.Errorf("gagal mendapatkan bucket %s: %w", s.bucket, err)
		}
	}

	// Create object handle
	obj := bucket.Object(storagePath)

	// Create writer
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType
	writer.Metadata = metadata

	// Write the file data
	if _, err := writer.Write(data); err != nil {
		s.logger.Error(ctx, "Gagal menulis data ke Firebase Storage", telemetry.Fields{
			"path":  storagePath,
			"error": err.Error(),
		})
		writer.Close()
		return nil, fmt.Errorf("gagal menulis data ke Firebase Storage: %w", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		s.logger.Error(ctx, "Gagal menutup writer Firebase Storage", telemetry.Fields{
			"path":  storagePath,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("gagal menutup writer Firebase Storage: %w", err)
	}

	// Get the URL
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		s.logger.Warn(ctx, "Gagal mendapatkan atribut objek", telemetry.Fields{
			"path":  storagePath,
			"error": err.Error(),
		})
		// Continue without URL
	}

	// Create result
	result := &StorageResult{
		Filename:    filename,
		Path:        storagePath,
		Size:        fileSize,
		ContentType: contentType,
		UploadedAt:  time.Now(),
		Metadata:    metadata,
	}

	// Set URL if available
	if attrs != nil && attrs.MediaLink != "" {
		result.URL = attrs.MediaLink
	} else if s.baseURL != "" {
		result.URL = strings.TrimSuffix(s.baseURL, "/") + "/" + strings.TrimPrefix(storagePath, "/")
	}

	s.logger.Info(ctx, "Berhasil mengupload file ke Firebase Storage", telemetry.Fields{
		"path": storagePath,
		"size": fileSize,
	})

	return result, nil
}

// UploadFromURL mengunduh file dari URL dan menguploadnya ke Firebase Storage
func (s *FirebaseStorage) UploadFromURL(ctx context.Context, url, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
	logFields := telemetry.Fields{
		"url":      url,
		"filename": filename,
		"bucket":   s.bucket,
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

// UploadFromBase64 mengupload file dari data base64 ke Firebase Storage
func (s *FirebaseStorage) UploadFromBase64(ctx context.Context, base64Data, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
	logFields := telemetry.Fields{
		"filename": filename,
		"bucket":   s.bucket,
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

// Get mengambil file dari Firebase Storage
func (s *FirebaseStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	logFields := telemetry.Fields{
		"path":   path,
		"bucket": s.bucket,
	}

	s.logger.Debug(ctx, "Mengambil file dari Firebase Storage", logFields)

	// Get default bucket or specific bucket
	bucket, err := s.client.DefaultBucket()
	if err != nil {
		s.logger.Error(ctx, "Gagal mendapatkan bucket default", telemetry.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("gagal mendapatkan bucket default: %w", err)
	}

	if s.bucket != "" {
		bucket, err = s.client.Bucket(s.bucket)
		if err != nil {
			s.logger.Error(ctx, "Gagal mendapatkan bucket", telemetry.Fields{
				"bucket": s.bucket,
				"error":  err.Error(),
			})
			return nil, fmt.Errorf("gagal mendapatkan bucket %s: %w", s.bucket, err)
		}
	}

	// Get the object
	obj := bucket.Object(path)

	// Check if object exists
	if _, err := obj.Attrs(ctx); err != nil {
		s.logger.Warn(ctx, "Objek tidak ditemukan", telemetry.Fields{
			"path":  path,
			"error": err.Error(),
		})
		return nil, ErrFileNotFound
	}

	// Open the object for reading
	reader, err := obj.NewReader(ctx)
	if err != nil {
		s.logger.Error(ctx, "Gagal membuka file untuk dibaca", telemetry.Fields{
			"path":  path,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("gagal membuka file untuk dibaca: %w", err)
	}

	return reader, nil
}

// GetURL mendapatkan URL untuk mengakses file di Firebase Storage
func (s *FirebaseStorage) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	logFields := telemetry.Fields{
		"path":   path,
		"bucket": s.bucket,
		"expiry": expiry,
	}

	s.logger.Debug(ctx, "Mendapatkan URL file", logFields)

	// Get bucket
	bucket, err := s.client.DefaultBucket()
	if err != nil {
		s.logger.Error(ctx, "Gagal mendapatkan bucket default", telemetry.Fields{
			"error": err.Error(),
		})
		return "", fmt.Errorf("gagal mendapatkan bucket default: %w", err)
	}

	if s.bucket != "" {
		bucket, err = s.client.Bucket(s.bucket)
		if err != nil {
			s.logger.Error(ctx, "Gagal mendapatkan bucket", telemetry.Fields{
				"bucket": s.bucket,
				"error":  err.Error(),
			})
			return "", fmt.Errorf("gagal mendapatkan bucket %s: %w", s.bucket, err)
		}
	}

	// Get the object
	obj := bucket.Object(path)

	// Check if object exists
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		s.logger.Warn(ctx, "Objek tidak ditemukan", telemetry.Fields{
			"path":  path,
			"error": err.Error(),
		})
		return "", ErrFileNotFound
	}

	// Jika menggunakan firebase-admin/Go SDK, gunakan metode alternatif karena SignedURL tidak tersedia
	// dengan cara yang kita harapkan di Firebase Admin Go SDK

	// Opsi 1: Gunakan URL publik jika tersedia
	if attrs.MediaLink != "" {
		return attrs.MediaLink, nil
	}

	// Opsi 2: Gunakan base URL + path jika dikonfigurasi
	if s.baseURL != "" {
		return strings.TrimSuffix(s.baseURL, "/") + "/" + strings.TrimPrefix(path, "/"), nil
	}

	// Fallback: Gunakan format URL Cloud Storage standar
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucket, path), nil
}

// Delete menghapus file dari Firebase Storage
func (s *FirebaseStorage) Delete(ctx context.Context, path string) error {
	logFields := telemetry.Fields{
		"path":   path,
		"bucket": s.bucket,
	}

	s.logger.Debug(ctx, "Menghapus file dari Firebase Storage", logFields)

	// Get bucket
	bucket, err := s.client.DefaultBucket()
	if err != nil {
		s.logger.Error(ctx, "Gagal mendapatkan bucket default", telemetry.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("gagal mendapatkan bucket default: %w", err)
	}

	if s.bucket != "" {
		bucket, err = s.client.Bucket(s.bucket)
		if err != nil {
			s.logger.Error(ctx, "Gagal mendapatkan bucket", telemetry.Fields{
				"bucket": s.bucket,
				"error":  err.Error(),
			})
			return fmt.Errorf("gagal mendapatkan bucket %s: %w", s.bucket, err)
		}
	}

	// Get the object
	obj := bucket.Object(path)

	// Delete the object
	if err := obj.Delete(ctx); err != nil {
		s.logger.Error(ctx, "Gagal menghapus file", telemetry.Fields{
			"path":  path,
			"error": err.Error(),
		})
		return fmt.Errorf("gagal menghapus file: %w", err)
	}

	s.logger.Info(ctx, "Berhasil menghapus file", logFields)
	return nil
}

// Close menutup koneksi
func (s *FirebaseStorage) Close() error {
	// Nothing to close for Firebase Storage
	return nil
}
