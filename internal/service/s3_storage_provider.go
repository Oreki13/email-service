package service

import (
	"bytes"
	"context"
	"email-service/pkg/telemetry"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

// S3StorageProvider adalah implementasi StorageProvider untuk Amazon S3
type S3StorageProvider struct {
	client    *s3.S3
	bucket    string
	baseURL   string
	region    string
	accessKey string
	secretKey string
	logger    telemetry.Logger
}

// NewS3StorageProvider membuat instance baru S3StorageProvider
func NewS3StorageProvider(config S3Config, logger telemetry.Logger) (*S3StorageProvider, error) {
	// Konfigurasi AWS Session
	awsConfig := &aws.Config{
		Region: aws.String(config.Region),
	}

	// Jika ada kredensi, gunakan
	if config.AccessKey != "" && config.SecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			config.AccessKey,
			config.SecretKey,
			"",
		)
	}

	// Buat session
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session AWS: %w", err)
	}

	// Buat client S3
	s3Client := s3.New(sess)

	// Pastikan bucket ada dan bisa diakses
	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(config.Bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal mengakses bucket S3: %w", err)
	}

	logger.Info(context.Background(), "S3 storage provider diinisialisasi", telemetry.Fields{
		"bucket": config.Bucket,
		"region": config.Region,
	})

	return &S3StorageProvider{
		client:    s3Client,
		bucket:    config.Bucket,
		baseURL:   config.BaseURL,
		region:    config.Region,
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		logger:    logger,
	}, nil
}

// SaveFile menyimpan file ke Amazon S3
func (p *S3StorageProvider) SaveFile(ctx context.Context, data []byte, filename string, contentType string) (string, error) {
	// Buat struktur folder berdasarkan tanggal
	currentTime := time.Now()
	prefix := fmt.Sprintf("attachments/%04d/%02d/%02d", currentTime.Year(), currentTime.Month(), currentTime.Day())

	// Buat nama file unik dengan UUID
	fileExt := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, fileExt)
	uniqueID := uuid.New().String()
	uniqueFilename := fmt.Sprintf("%s-%s%s", baseName, uniqueID, fileExt)

	// Path lengkap untuk file di S3
	s3Key := fmt.Sprintf("%s/%s", prefix, uniqueFilename)

	// Jika contentType kosong, coba tebak dari ekstensi file
	if contentType == "" {
		contentType = detectContentType(fileExt)
	}

	// Set parameter upload
	params := &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	// Upload file ke S3
	_, err := p.client.PutObject(params)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan file ke S3: %w", err)
	}

	p.logger.Info(ctx, "File berhasil disimpan ke S3", telemetry.Fields{
		"filename":     filename,
		"bucket":       p.bucket,
		"key":          s3Key,
		"size":         len(data),
		"content_type": contentType,
	})

	// Kembalikan URL atau path S3
	var fileURL string
	if p.baseURL != "" {
		// Jika ada base URL, gunakan itu
		fileURL = fmt.Sprintf("%s/%s", strings.TrimRight(p.baseURL, "/"), s3Key)
	} else {
		// Gunakan format URL S3 standar
		fileURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", p.bucket, p.region, s3Key)
	}

	return fileURL, nil
}

// Name mengembalikan nama provider
func (p *S3StorageProvider) Name() string {
	return "s3"
}

// detectContentType mencoba menebak content type berdasarkan ekstensi file
func detectContentType(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".zip":
		return "application/zip"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}
