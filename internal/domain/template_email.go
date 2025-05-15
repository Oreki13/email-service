package domain

import "context"

// WelcomeEmailRequest merepresentasikan request untuk mengirim email selamat datang
type WelcomeEmailRequest struct {
	To          []string            `json:"to" validate:"required,dive,email"`
	Cc          []string            `json:"cc,omitempty" validate:"omitempty,dive,email"`
	Bcc         []string            `json:"bcc,omitempty" validate:"omitempty,dive,email"`
	Name        string              `json:"name" validate:"required"`
	AppName     string              `json:"appName" validate:"required"`
	Attachments []AttachmentRequest `json:"attachments,omitempty"`
	Priority    Priority            `json:"priority,omitempty"`
	Provider    Provider            `json:"provider,omitempty"`
	Metadata    map[string]string   `json:"metadata,omitempty"`
}

// PasswordResetEmailRequest merepresentasikan request untuk mengirim email reset password
type PasswordResetEmailRequest struct {
	To        []string          `json:"to" validate:"required,dive,email"`
	Cc        []string          `json:"cc,omitempty" validate:"omitempty,dive,email"`
	Name      string            `json:"name" validate:"required"`
	AppName   string            `json:"appName" validate:"required"`
	ResetURL  string            `json:"resetUrl" validate:"required,url"`
	ExpiresIn int               `json:"expiresIn" validate:"required"`
	Priority  Priority          `json:"priority,omitempty"`
	Provider  Provider          `json:"provider,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NotificationEmailRequest merepresentasikan request untuk mengirim email notifikasi
type NotificationEmailRequest struct {
	To       []string          `json:"to" validate:"required,dive,email"`
	Cc       []string          `json:"cc,omitempty" validate:"omitempty,dive,email"`
	Bcc      []string          `json:"bcc,omitempty" validate:"omitempty,dive,email"`
	Subject  string            `json:"subject" validate:"required"`
	Message  string            `json:"message" validate:"required"`
	AppName  string            `json:"appName" validate:"required"`
	Priority Priority          `json:"priority,omitempty"`
	Provider Provider          `json:"provider,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TemplatedEmailService mendefinisikan interface untuk mengirim email dengan template spesifik
type TemplatedEmailService interface {
	// SendWelcomeEmail mengirim email selamat datang dengan template yang sudah ditentukan
	SendWelcomeEmail(ctx context.Context, request *WelcomeEmailRequest) (string, error)

	// SendPasswordResetEmail mengirim email reset password dengan template yang sudah ditentukan
	SendPasswordResetEmail(ctx context.Context, request *PasswordResetEmailRequest) (string, error)

	// SendNotificationEmail mengirim email notifikasi dengan template yang sudah ditentukan
	SendNotificationEmail(ctx context.Context, request *NotificationEmailRequest) (string, error)

	// GetTemplateInfo mendapatkan informasi tentang template yang tersedia
	GetTemplateInfo(ctx context.Context) (map[string]TemplateInfo, error)
}

// TemplateInfo berisi informasi tentang template email
type TemplateInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
}
