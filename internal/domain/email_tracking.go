package domain

import (
	"context"
	"time"
)

// TrackingType mendefinisikan jenis tracking yang dilakukan
type TrackingType string

const (
	// TrackingTypeOpen melacak pembukaan email
	TrackingTypeOpen TrackingType = "open"

	// TrackingTypeClick melacak klik pada link
	TrackingTypeClick TrackingType = "click"
)

// EmailTracking merepresentasikan data tracking untuk email
type EmailTracking struct {
	ID        string       `json:"id"`
	EmailID   string       `json:"email_id"`
	Type      TrackingType `json:"type"`
	Timestamp time.Time    `json:"timestamp"`
	UserAgent string       `json:"user_agent,omitempty"`
	IPAddress string       `json:"ip_address,omitempty"`
	URL       string       `json:"url,omitempty"` // Untuk tracking klik
	Count     int          `json:"count"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// EmailTrackingRepository adalah interface untuk akses data tracking email
type EmailTrackingRepository interface {
	// SaveTracking menyimpan event tracking baru
	SaveTracking(ctx context.Context, tracking *EmailTracking) error

	// IncrementOpenCount menambah jumlah pembukaan email
	IncrementOpenCount(ctx context.Context, emailID string, userAgent, ipAddress string) error

	// IncrementClickCount menambah jumlah klik pada link dalam email
	IncrementClickCount(ctx context.Context, emailID string, url, userAgent, ipAddress string) error

	// GetEmailTrackingData mendapatkan data tracking untuk email tertentu
	GetEmailTrackingData(ctx context.Context, emailID string) ([]*EmailTracking, error)

	// GetEmailOpenCount mendapatkan jumlah pembukaan email
	GetEmailOpenCount(ctx context.Context, emailID string) (int, error)
}
