package domain

import (
	"context"
	"time"
)

// DeliveryStatus menggambarkan status pengiriman email
type DeliveryStatus string

const (
	// StatusPending menunjukkan email belum diproses
	StatusPending DeliveryStatus = "pending"
	// StatusSending menunjukkan email sedang dalam proses pengiriman
	StatusSending DeliveryStatus = "sending"
	// StatusSent menunjukkan email berhasil dikirim
	StatusSent DeliveryStatus = "sent"
	// StatusFailed menunjukkan pengiriman email gagal
	StatusFailed DeliveryStatus = "failed"
	// StatusBounced menunjukkan email dipantulkan oleh server tujuan
	StatusBounced DeliveryStatus = "bounced"
	// StatusDelivered menunjukkan email berhasil diterima oleh server tujuan
	StatusDelivered DeliveryStatus = "delivered"
	// StatusOpened menunjukkan email dibuka oleh penerima
	StatusOpened DeliveryStatus = "opened"
)

// Priority menggambarkan prioritas pengiriman email
type Priority string

const (
	// PriorityHigh untuk email yang perlu dikirim segera
	PriorityHigh Priority = "high"
	// PriorityNormal untuk email dengan prioritas normal
	PriorityNormal Priority = "normal"
	// PriorityLow untuk email yang dapat ditunda pengirimannya
	PriorityLow Priority = "low"
)

// Provider menggambarkan jenis provider email
type Provider string

const (
	// ProviderSMTP untuk pengiriman via SMTP
	ProviderSMTP Provider = "smtp"
	// ProviderSES untuk pengiriman via AWS SES
	ProviderSES Provider = "ses"
)

// Email merepresentasikan entitas email yang akan dikirim
type Email struct {
	ID           string            `json:"id"`
	From         string            `json:"from"`
	To           []string          `json:"to"`
	Cc           []string          `json:"cc"`
	Bcc          []string          `json:"bcc"`
	Subject      string            `json:"subject"`
	PlainBody    string            `json:"plainBody"`
	HTMLBody     string            `json:"htmlBody"`
	TemplateID   string            `json:"templateId,omitempty"`
	TemplateData interface{}       `json:"templateData,omitempty"`
	Attachments  []*Attachment     `json:"attachments,omitempty"`
	Status       DeliveryStatus    `json:"status"`
	Priority     Priority          `json:"priority"`
	Provider     Provider          `json:"provider"`
	SentAt       *time.Time        `json:"sentAt,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	RetryCount   int               `json:"retryCount"`
	MaxRetries   int               `json:"maxRetries"`
	Error        string            `json:"error,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Attachment merepresentasikan file yang dilampirkan ke email
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Data        []byte `json:"-"` // Data tidak di-serialisasi ke JSON
	Path        string `json:"path,omitempty"`
	URL         string `json:"url,omitempty"` // URL untuk mengakses file dari storage
}

// EmailRequest merepresentasikan request untuk mengirim email
type EmailRequest struct {
	To           []string               `json:"to" validate:"required,dive,email"`
	Cc           []string               `json:"cc,omitempty" validate:"omitempty,dive,email"`
	Bcc          []string               `json:"bcc,omitempty" validate:"omitempty,dive,email"`
	Subject      string                 `json:"subject" validate:"required"`
	PlainBody    string                 `json:"plainBody,omitempty"`
	HTMLBody     string                 `json:"htmlBody,omitempty"`
	TemplateID   string                 `json:"templateId,omitempty"`
	TemplateName string                 `json:"templateName,omitempty"` // Nama template sebagai alternatif untuk templateId
	TemplateData map[string]interface{} `json:"templateData,omitempty"`
	Attachments  []AttachmentRequest    `json:"attachments,omitempty"`
	Priority     Priority               `json:"priority,omitempty"`
	Provider     Provider               `json:"provider,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

// AttachmentRequest merepresentasikan request untuk melampirkan file
type AttachmentRequest struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"contentType" validate:"required"`
	Base64Data  string `json:"base64Data,omitempty" validate:"omitempty"`
	URL         string `json:"url,omitempty" validate:"omitempty,url"`
}

// EmailStatus merepresentasikan status pengiriman email
type EmailStatus struct {
	ID        string         `json:"id"`
	Status    DeliveryStatus `json:"status"`
	SentAt    *time.Time     `json:"sentAt,omitempty"`
	Error     string         `json:"error,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// EmailRepository mendefinisikan interface untuk operasi email repository
type EmailRepository interface {
	Save(ctx context.Context, email *Email) error
	FindByID(ctx context.Context, id string) (*Email, error)
	UpdateStatus(ctx context.Context, id string, status DeliveryStatus, errorMsg string) error
	FindPendingEmails(ctx context.Context, limit int) ([]*Email, error)
	FindByStatus(ctx context.Context, status DeliveryStatus, limit, offset int) ([]*Email, error)
	IncrementRetryCount(ctx context.Context, id string) error
	UpdateSentTime(ctx context.Context, id string, sentAt *time.Time) error
}

// EmailService mendefinisikan interface untuk operasi email service
type EmailService interface {
	SendEmail(ctx context.Context, request *EmailRequest) (string, error)
	GetStatus(ctx context.Context, id string) (*EmailStatus, error)
	ProcessPendingEmails(ctx context.Context) error

	// TrackEmailOpen melacak pembukaan email
	TrackEmailOpen(ctx context.Context, emailID string, userAgent, ipAddress string) error

	// TrackEmailClick melacak klik pada link dalam email
	TrackEmailClick(ctx context.Context, emailID string, url, userAgent, ipAddress string) error

	// GetEmailTrackingData mendapatkan data tracking untuk email tertentu
	GetEmailTrackingData(ctx context.Context, emailID string) ([]*EmailTracking, error)
}

// EmailDelivery mendefinisikan interface untuk mengirim email
type EmailDelivery interface {
	Send(ctx context.Context, email *Email) error
	Name() string
}

// TemplateRepository mendefinisikan interface untuk operasi template repository
type TemplateRepository interface {
	FindByID(ctx context.Context, id string) (*Template, error)
	FindByName(ctx context.Context, name string) (*Template, error)
	Save(ctx context.Context, template *Template) error
	Delete(ctx context.Context, id string) error
	FindAll(ctx context.Context) ([]*Template, error)
}

// Template merepresentasikan template email
type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Subject     string    `json:"subject"`
	PlainBody   string    `json:"plainBody"`
	HTMLBody    string    `json:"htmlBody"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TemplateRequest merepresentasikan request untuk membuat template
type TemplateRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Subject     string `json:"subject" validate:"required"`
	PlainBody   string `json:"plainBody"`
	HTMLBody    string `json:"htmlBody" validate:"required"`
}

// APIKey merepresentasikan API key untuk akses ke service
type APIKey struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ServiceName string     `json:"serviceName"`
	IsActive    bool       `json:"isActive"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

// APIKeyCreateRequest merepresentasikan request untuk membuat API key baru
type APIKeyCreateRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	ServiceName string `json:"serviceName" validate:"required"`
	ExpiryDays  int    `json:"expiryDays"` // 0 berarti tidak ada expiry
}

// APIKeyUpdateRequest merepresentasikan request untuk memperbarui API key
type APIKeyUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	ServiceName *string `json:"serviceName,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
	ExpiryDays  *int    `json:"expiryDays,omitempty"` // 0 berarti tidak ada expiry
}

// APIKeyRepository mendefinisikan interface untuk operasi API key repository
type APIKeyRepository interface {
	FindByKey(ctx context.Context, key string) (*APIKey, error)
	Save(ctx context.Context, apiKey *APIKey) error
	Update(ctx context.Context, apiKey *APIKey) error
	Delete(ctx context.Context, id string) error
	FindAll(ctx context.Context) ([]*APIKey, error)
	FindByID(ctx context.Context, id string) (*APIKey, error)
	UpdateLastUsed(ctx context.Context, id string) error
	FindWithPagination(ctx context.Context, offset, limit int) ([]*APIKey, int, error)
}
