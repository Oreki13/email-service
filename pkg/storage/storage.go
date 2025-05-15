// Package storage menyediakan abstraksi untuk menyimpan file attachment
package storage

import (
	"context"
	"io"
	"time"
)

// Provider adalah tipe untuk provider penyimpanan file
type Provider string

const (
	// ProviderLocal menggunakan sistem file lokal untuk penyimpanan
	ProviderLocal Provider = "local"
	// ProviderS3 menggunakan AWS S3 untuk penyimpanan
	ProviderS3 Provider = "s3"
	// ProviderFirebase menggunakan Firebase Storage untuk penyimpanan
	ProviderFirebase Provider = "firebase"
)

// StorageResult berisi informasi hasil penyimpanan file
type StorageResult struct {
	// Filename adalah nama file yang disimpan
	Filename string
	// Path adalah path relatif di storage (tanpa base URL)
	Path string
	// URL adalah URL lengkap untuk mengakses file (opsional)
	URL string
	// Size adalah ukuran file dalam bytes
	Size int64
	// ContentType adalah MIME type dari file
	ContentType string
	// Metadata adalah data tambahan tentang file
	Metadata map[string]string
	// UploadedAt adalah waktu saat file diupload
	UploadedAt time.Time
}

// Storage adalah interface untuk mengakses storage provider
type Storage interface {
	// Upload mengupload file ke storage
	Upload(ctx context.Context, file io.Reader, filename, contentType string, metadata map[string]string) (*StorageResult, error)

	// UploadFromURL mengunduh file dari URL dan menguploadnya ke storage
	UploadFromURL(ctx context.Context, url, filename, contentType string, metadata map[string]string) (*StorageResult, error)

	// UploadFromBase64 mengupload file dari data base64 ke storage
	UploadFromBase64(ctx context.Context, base64Data, filename, contentType string, metadata map[string]string) (*StorageResult, error)

	// Get mengambil file dari storage
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// GetURL mendapatkan URL untuk mengakses file
	GetURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// Delete menghapus file dari storage
	Delete(ctx context.Context, path string) error

	// Close menutup koneksi ke storage provider
	Close() error
}
