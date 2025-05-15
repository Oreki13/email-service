package service

import (
	"context"
	"email-service/pkg/telemetry"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// FirebaseStorageProvider adalah implementasi StorageProvider untuk Firebase Cloud Storage
type FirebaseStorageProvider struct {
	bucket      *storage.BucketHandle
	bucketName  string
	storagePath string
	projectID   string
	logger      telemetry.Logger
}

// NewFirebaseStorageProvider membuat instance baru FirebaseStorageProvider
func NewFirebaseStorageProvider(config FirebaseConfig, logger telemetry.Logger) (*FirebaseStorageProvider, error) {
	ctx := context.Background()
	var app *firebase.App
	var err error

	// Inisialisasi Firebase app
	if config.CredFile != "" {
		// Gunakan file kredensial yang ditentukan
		sa := option.WithCredentialsFile(config.CredFile)
		app, err = firebase.NewApp(ctx, &firebase.Config{
			ProjectID:     config.ProjectID,
			StorageBucket: config.Bucket,
		}, sa)
	} else {
		// Gunakan autentikasi default (misalnya dari variabel lingkungan GOOGLE_APPLICATION_CREDENTIALS)
		app, err = firebase.NewApp(ctx, &firebase.Config{
			ProjectID:     config.ProjectID,
			StorageBucket: config.Bucket,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("gagal menginisialisasi Firebase app: %w", err)
	}

	// Dapatkan client Storage
	storageClient, err := app.Storage(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan Storage client: %w", err)
	}

	// Dapatkan handle bucket
	bucket, err := storageClient.DefaultBucket()
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan bucket default: %w", err)
	}

	logger.Info(ctx, "Firebase storage provider diinisialisasi", telemetry.Fields{
		"project_id": config.ProjectID,
		"bucket":     config.Bucket,
	})

	return &FirebaseStorageProvider{
		bucket:      bucket,
		bucketName:  config.Bucket,
		storagePath: config.StoragePath,
		projectID:   config.ProjectID,
		logger:      logger,
	}, nil
}

// SaveFile menyimpan file ke Firebase Cloud Storage
func (p *FirebaseStorageProvider) SaveFile(ctx context.Context, data []byte, filename string, contentType string) (string, error) {
	// Buat struktur folder berdasarkan tanggal
	currentTime := time.Now()
	prefix := fmt.Sprintf("%s/%04d/%02d/%02d",
		strings.TrimPrefix(p.storagePath, "/"),
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day())

	// Buat nama file unik dengan UUID
	fileExt := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, fileExt)
	uniqueID := uuid.New().String()
	uniqueFilename := fmt.Sprintf("%s-%s%s", baseName, uniqueID, fileExt)

	// Path lengkap untuk file di Firebase Storage
	objectPath := fmt.Sprintf("%s/%s", prefix, uniqueFilename)

	// Jika contentType kosong, coba tebak dari ekstensi file
	if contentType == "" {
		contentType = detectContentType(fileExt)
	}

	// Buat file writer
	obj := p.bucket.Object(objectPath)
	wc := obj.NewWriter(ctx)
	wc.ContentType = contentType
	wc.Metadata = map[string]string{
		"originalname": filename,
		"uploadedat":   currentTime.Format(time.RFC3339),
	}

	// Tulis data ke storage
	if _, err := wc.Write(data); err != nil {
		wc.Close()
		return "", fmt.Errorf("gagal menulis data ke Firebase Storage: %w", err)
	}

	// Tutup writer
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("gagal menutup writer Firebase Storage: %w", err)
	}

	// Ubah ACL untuk publik jika memungkinkan
	// Ini adalah setting opsional, dan memerlukan konfigurasi tambahan di Firebase
	// Untuk keamanan lebih baik, gunakan signed URL daripada membuat file publik
	attr := storage.ObjectAttrsToUpdate{
		PredefinedACL: "publicRead",
	}
	if _, err := obj.Update(ctx, attr); err != nil {
		p.logger.Warn(ctx, "Gagal mengatur file sebagai publik, akan menggunakan URL biasa", telemetry.Fields{
			"error": err.Error(),
		})
	}

	p.logger.Info(ctx, "File berhasil disimpan ke Firebase Storage", telemetry.Fields{
		"filename":     filename,
		"path":         objectPath,
		"size":         len(data),
		"content_type": contentType,
	})

	// Kembalikan URL publik
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", p.bucketName, objectPath)
	return publicURL, nil
}

// Name mengembalikan nama provider
func (p *FirebaseStorageProvider) Name() string {
	return "firebase"
}
