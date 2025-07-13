package service

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
)

// TemplatedEmailServiceImpl implementasi dari domain.TemplatedEmailService
type TemplatedEmailServiceImpl struct {
	emailService domain.EmailService
	templateRepo domain.TemplateRepository
	templates    map[string]string // Map nama template ke ID template
	logger       telemetry.Logger
}

// NewTemplatedEmailService membuat instance baru template email service
func NewTemplatedEmailService(
	emailService domain.EmailService,
	templateRepo domain.TemplateRepository,
	logger telemetry.Logger,
) domain.TemplatedEmailService {
	templates := map[string]string{
		"welcome":        "welcome_template",
		"reset-password": "password_reset",
		"notification":   "notification_template",
	}

	return &TemplatedEmailServiceImpl{
		emailService: emailService,
		templateRepo: templateRepo,
		templates:    templates,
		logger:       logger,
	}
}

// SendWelcomeEmail mengirim email selamat datang
func (s *TemplatedEmailServiceImpl) SendWelcomeEmail(ctx context.Context, request *domain.WelcomeEmailRequest) (string, error) {
	s.logger.Info(ctx, "Mengirim email selamat datang", telemetry.Fields{
		"to":       request.To,
		"name":     request.Name,
		"app_name": request.AppName,
	})

	// Konversi ke EmailRequest general
	emailReq := &domain.EmailRequest{
		To:           request.To,
		Cc:           request.Cc,
		Bcc:          request.Bcc,
		TemplateName: s.templates["welcome"],
		TemplateData: map[string]interface{}{
			"name":     request.Name,
			"app_name": request.AppName,
		},
		Attachments: request.Attachments,
		Priority:    request.Priority,
		Provider:    request.Provider,
		Metadata:    request.Metadata,
	}

	// Gunakan email service yang ada untuk mengirim
	emailID, err := s.emailService.SendEmail(ctx, emailReq)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengirim email selamat datang", telemetry.Fields{
			"to":    request.To,
			"name":  request.Name,
			"error": err.Error(),
		})
		return "", err
	}

	s.logger.Info(ctx, "Email selamat datang berhasil dikirim", telemetry.Fields{
		"email_id": emailID,
		"to":       request.To,
	})

	return emailID, nil
}

// SendPasswordResetEmail mengirim email reset password
func (s *TemplatedEmailServiceImpl) SendPasswordResetEmail(ctx context.Context, request *domain.PasswordResetEmailRequest) (string, error) {
	s.logger.Info(ctx, "Mengirim email reset password", telemetry.Fields{
		"to":        request.To,
		"name":      request.Name,
		"reset_url": request.ResetURL,
	})

	// Konversi ke EmailRequest general
	emailReq := &domain.EmailRequest{
		To:           request.To,
		Cc:           request.Cc,
		TemplateName: s.templates["reset-password"],
		TemplateData: map[string]interface{}{
			"name":       request.Name,
			"app_name":   request.AppName,
			"reset_url":  request.ResetURL,
			"expires_in": request.ExpiresIn,
		},
		Priority: request.Priority,
		Provider: request.Provider,
		Metadata: request.Metadata,
	}

	// Gunakan email service yang ada untuk mengirim
	emailID, err := s.emailService.SendEmail(ctx, emailReq)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengirim email reset password", telemetry.Fields{
			"to":    request.To,
			"name":  request.Name,
			"error": err.Error(),
		})
		return "", err
	}

	s.logger.Info(ctx, "Email reset password berhasil dikirim", telemetry.Fields{
		"email_id": emailID,
		"to":       request.To,
	})

	return emailID, nil
}

// SendNotificationEmail mengirim email notifikasi
func (s *TemplatedEmailServiceImpl) SendNotificationEmail(ctx context.Context, request *domain.NotificationEmailRequest) (string, error) {
	s.logger.Info(ctx, "Mengirim email notifikasi", telemetry.Fields{
		"to":      request.To,
		"subject": request.Subject,
	})

	// Konversi ke EmailRequest general
	emailReq := &domain.EmailRequest{
		To:           request.To,
		Cc:           request.Cc,
		Bcc:          request.Bcc,
		Subject:      request.Subject, // Menggunakan subject dari request
		TemplateName: s.templates["notification"],
		TemplateData: map[string]interface{}{
			"subject":  request.Subject,
			"message":  request.Message,
			"app_name": request.AppName,
		},
		Priority: request.Priority,
		Provider: request.Provider,
		Metadata: request.Metadata,
	}

	// Gunakan email service yang ada untuk mengirim
	emailID, err := s.emailService.SendEmail(ctx, emailReq)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengirim email notifikasi", telemetry.Fields{
			"to":      request.To,
			"subject": request.Subject,
			"error":   err.Error(),
		})
		return "", err
	}

	s.logger.Info(ctx, "Email notifikasi berhasil dikirim", telemetry.Fields{
		"email_id": emailID,
		"to":       request.To,
		"subject":  request.Subject,
	})

	return emailID, nil
}

// GetTemplateInfo mendapatkan informasi tentang template yang tersedia
func (s *TemplatedEmailServiceImpl) GetTemplateInfo(ctx context.Context) (map[string]domain.TemplateInfo, error) {
	s.logger.Info(ctx, "Mendapatkan informasi template", nil)

	// Ambil informasi template dari repository
	templateInfos := make(map[string]domain.TemplateInfo)

	// Template Welcome Email
	templateInfos["welcome"] = domain.TemplateInfo{
		Name:        "Welcome Email",
		Description: "Email untuk menyambut pengguna baru",
		Parameters: []string{
			"name",
			"app_name",
		},
	}

	// Template Reset Password
	templateInfos["reset-password"] = domain.TemplateInfo{
		Name:        "Reset Password",
		Description: "Email untuk reset password",
		Parameters: []string{
			"name",
			"app_name",
			"reset_url",
			"expires_in",
		},
	}

	// Template Notification
	templateInfos["notification"] = domain.TemplateInfo{
		Name:        "Notification",
		Description: "Email notifikasi umum",
		Parameters: []string{
			"subject",
			"message",
			"app_name",
		},
	}

	return templateInfos, nil
}

// InitializeTemplates inisialisasi template jika belum ada di database
// func (s *TemplatedEmailServiceImpl) InitializeTemplates(ctx context.Context) error {
// 	s.logger.Info(ctx, "Menginisialisasi template email default", nil)

// 	// Template Welcome Email
// 	welcomeTemplate := &domain.Template{
// 		ID:          s.templates["welcome"],
// 		Name:        "Welcome Email",
// 		Description: "Email untuk menyambut pengguna baru",
// 		Subject:     "Selamat Datang di {{app_name}}",
// 		HTMLBody: `
// 		<!DOCTYPE html>
// 		<html>
// 		<head>
// 			<meta charset="utf-8">
// 			<title>Selamat Datang</title>
// 		</head>
// 		<body>
// 			<h1>Halo {{name}},</h1>
// 			<p>Selamat datang di {{app_name}}! Kami senang Anda bergabung dengan kami.</p>
// 			<p>Silakan gunakan platform kami untuk kebutuhan Anda.</p>
// 			<p>Terima kasih,<br>Tim {{app_name}}</p>
// 		</body>
// 		</html>
// 		`,
// 		PlainBody: `
// 		Halo {{name}},

// 		Selamat datang di {{app_name}}! Kami senang Anda bergabung dengan kami.

// 		Silakan gunakan platform kami untuk kebutuhan Anda.

// 		Terima kasih,
// 		Tim {{app_name}}
// 		`,
// 	}

// 	// Template Reset Password
// 	resetPasswordTemplate := &domain.Template{
// 		ID:          s.templates["reset-password"],
// 		Name:        "Reset Password",
// 		Description: "Email untuk reset password",
// 		Subject:     "Reset Password untuk {{app_name}}",
// 		HTMLBody: `
// 		<!DOCTYPE html>
// 		<html>
// 		<head>
// 			<meta charset="utf-8">
// 			<title>Reset Password</title>
// 		</head>
// 		<body>
// 			<h1>Halo {{name}},</h1>
// 			<p>Kami menerima permintaan untuk mereset password Anda di {{app_name}}.</p>
// 			<p>Silakan klik link berikut untuk melanjutkan proses reset password:</p>
// 			<p><a href="{{reset_url}}">Reset Password</a></p>
// 			<p>Link ini akan kedaluwarsa dalam {{expires_in}} jam.</p>
// 			<p>Jika Anda tidak meminta reset password, abaikan email ini.</p>
// 			<p>Terima kasih,<br>Tim {{app_name}}</p>
// 		</body>
// 		</html>
// 		`,
// 		PlainBody: `
// 		Halo {{name}},

// 		Kami menerima permintaan untuk mereset password Anda di {{app_name}}.

// 		Silakan kunjungi link berikut untuk melanjutkan proses reset password:
// 		{{reset_url}}

// 		Link ini akan kedaluwarsa dalam {{expires_in}} jam.

// 		Jika Anda tidak meminta reset password, abaikan email ini.

// 		Terima kasih,
// 		Tim {{app_name}}
// 		`,
// 	}

// 	// Template Notification
// 	notificationTemplate := &domain.Template{
// 		ID:          s.templates["notification"],
// 		Name:        "Notification",
// 		Description: "Email notifikasi umum",
// 		Subject:     "{{subject}}",
// 		HTMLBody: `
// 		<!DOCTYPE html>
// 		<html>
// 		<head>
// 			<meta charset="utf-8">
// 			<title>{{subject}}</title>
// 		</head>
// 		<body>
// 			<p>{{message}}</p>
// 			<p>Terima kasih,<br>Tim {{app_name}}</p>
// 		</body>
// 		</html>
// 		`,
// 		PlainBody: `
// 		{{message}}

// 		Terima kasih,
// 		Tim {{app_name}}
// 		`,
// 	}

// 	// Simpan template ke repository jika belum ada
// 	templates := []*domain.Template{welcomeTemplate, resetPasswordTemplate, notificationTemplate}

// 	for _, template := range templates {
// 		existingTemplate, err := s.templateRepo.FindByID(ctx, template.ID)
// 		if err != nil {
// 			s.logger.Error(ctx, "Gagal memeriksa template", telemetry.Fields{
// 				"template_id": template.ID,
// 				"error":       err.Error(),
// 			})
// 			return err
// 		}

// 		// Jika template belum ada, simpan template baru
// 		if existingTemplate == nil {
// 			if err := s.templateRepo.Save(ctx, template); err != nil {
// 				s.logger.Error(ctx, "Gagal menyimpan template", telemetry.Fields{
// 					"template_id":   template.ID,
// 					"template_name": template.Name,
// 					"error":         err.Error(),
// 				})
// 				return err
// 			}
// 			s.logger.Info(ctx, "Template berhasil dibuat", telemetry.Fields{
// 				"template_id":   template.ID,
// 				"template_name": template.Name,
// 			})
// 		}
// 	}

// 	s.logger.Info(ctx, "Inisialisasi template selesai", nil)
// 	return nil
// }
