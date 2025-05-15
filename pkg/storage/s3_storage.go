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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Storage adalah implementasi Storage menggunakan AWS S3
type S3Storage struct {
	session      *session.Session
	uploader     *s3manager.Uploader
	s3Service    *s3.S3
	bucket       string
	prefix       string
	baseURL      string
	maxSize      int64
	allowedTypes []string
	logger       telemetry.Logger
}

// NewS3Storage membuat instance baru dari S3Storage
func NewS3Storage(cfg *config.Config, logger telemetry.Logger) (*S3Storage, error) {
	// Buat konfigurasi AWS
	awsConfig := &aws.Config{
		Region: aws.String(cfg.Storage.S3.Region),
	}

	// Gunakan kredensial jika disediakan
	if cfg.Storage.S3.AccessKey != "" && cfg.Storage.S3.SecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			cfg.Storage.S3.AccessKey,
			cfg.Storage.S3.SecretKey,
			"", // token, tidak digunakan dalam kredensial statis
		)
	}

	// Buat session AWS
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session AWS: %w", err)
	}

	// Buat uploader dan service S3
	uploader := s3manager.NewUploader(sess)
	s3Service := s3.New(sess)

	return &S3Storage{
		session:      sess,
		uploader:     uploader,
		s3Service:    s3Service,
		bucket:       cfg.Storage.S3.Bucket,
		prefix:       cfg.Storage.S3.Prefix,
		baseURL:      cfg.Storage.BaseURL,
		maxSize:      cfg.Storage.MaxSize,
		allowedTypes: cfg.Storage.AllowedTypes,
		logger:       logger,
	}, nil
}

// Upload mengupload file ke S3
func (s *S3Storage) Upload(ctx context.Context, file io.Reader, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
	logFields := telemetry.Fields{
		"filename":     filename,
		"content_type": contentType,
		"bucket":       s.bucket,
	}

	s.logger.Debug(ctx, "Memulai upload file ke S3", logFields)

	// Validate content type
	if !ValidateFileType(contentType, s.allowedTypes) {
		s.logger.Warn(ctx, "Tipe file tidak diizinkan", logFields)
		return nil, ErrInvalidFileType
	}

	// Read the file into memory to get its size and validate
	// This is not ideal for large files but necessary for validation
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
	s3Key := GeneratePathWithPrefix(s.prefix, filename)

	// Prepare AWS metadata
	s3Metadata := make(map[string]*string)
	for k, v := range metadata {
		// S3 metadata keys must be lowercase
		s3Metadata["x-amz-meta-"+strings.ToLower(k)] = aws.String(v)
	}

	// Upload to S3
	uploadInput := &s3manager.UploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		Metadata:    s3Metadata,
		ACL:         aws.String("private"), // Default to private
	}

	result, err := s.uploader.UploadWithContext(ctx, uploadInput)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengupload file ke S3", telemetry.Fields{
			"bucket":   s.bucket,
			"key":      s3Key,
			"filename": filename,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("gagal mengupload file ke S3: %w", err)
	}

	// Create result object
	storageResult := &StorageResult{
		Filename:    filename,
		Path:        s3Key,
		Size:        fileSize,
		ContentType: contentType,
		UploadedAt:  time.Now(),
		Metadata:    metadata,
	}

	// Add URL if available
	if result.Location != "" {
		storageResult.URL = result.Location
	} else if s.baseURL != "" {
		storageResult.URL = strings.TrimSuffix(s.baseURL, "/") + "/" + strings.TrimPrefix(s3Key, "/")
	}

	s.logger.Info(ctx, "Berhasil mengupload file ke S3", telemetry.Fields{
		"bucket":   s.bucket,
		"key":      s3Key,
		"filename": filename,
		"size":     fileSize,
	})

	return storageResult, nil
}

// UploadFromURL mengunduh file dari URL dan menguploadnya ke S3
func (s *S3Storage) UploadFromURL(ctx context.Context, url, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
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

// UploadFromBase64 mengupload file dari data base64 ke S3
func (s *S3Storage) UploadFromBase64(ctx context.Context, base64Data, filename, contentType string, metadata map[string]string) (*StorageResult, error) {
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

// Get mengambil file dari S3
func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	logFields := telemetry.Fields{
		"bucket": s.bucket,
		"key":    path,
	}

	s.logger.Debug(ctx, "Mengambil file dari S3", logFields)

	// Create get object input
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	// Get the object
	result, err := s.s3Service.GetObjectWithContext(ctx, input)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengambil file dari S3", telemetry.Fields{
			"bucket": s.bucket,
			"key":    path,
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("gagal mengambil file dari S3: %w", err)
	}

	return result.Body, nil
}

// GetURL mendapatkan URL untuk mengakses file di S3
func (s *S3Storage) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	logFields := telemetry.Fields{
		"bucket": s.bucket,
		"key":    path,
		"expiry": expiry,
	}

	s.logger.Debug(ctx, "Membuat pre-signed URL", logFields)

	// Jika tidak ada expiry, gunakan default 1 jam
	if expiry == 0 {
		expiry = time.Hour
	}

	// Create request
	req, _ := s.s3Service.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	// Create presigned URL
	url, err := req.Presign(expiry)
	if err != nil {
		s.logger.Error(ctx, "Gagal membuat pre-signed URL", telemetry.Fields{
			"bucket": s.bucket,
			"key":    path,
			"error":  err.Error(),
		})
		return "", fmt.Errorf("gagal membuat pre-signed URL: %w", err)
	}

	s.logger.Debug(ctx, "Berhasil membuat pre-signed URL", telemetry.Fields{
		"bucket": s.bucket,
		"key":    path,
		"url":    url,
	})

	return url, nil
}

// Delete menghapus file dari S3
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	logFields := telemetry.Fields{
		"bucket": s.bucket,
		"key":    path,
	}

	s.logger.Debug(ctx, "Menghapus file dari S3", logFields)

	// Create delete object input
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	// Delete the object
	_, err := s.s3Service.DeleteObjectWithContext(ctx, input)
	if err != nil {
		s.logger.Error(ctx, "Gagal menghapus file dari S3", telemetry.Fields{
			"bucket": s.bucket,
			"key":    path,
			"error":  err.Error(),
		})
		return fmt.Errorf("gagal menghapus file dari S3: %w", err)
	}

	s.logger.Info(ctx, "Berhasil menghapus file dari S3", logFields)
	return nil
}

// Close menutup koneksi
func (s *S3Storage) Close() error {
	// Nothing to close for S3
	return nil
}
