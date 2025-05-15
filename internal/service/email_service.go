package service

import (
	"bytes"
	"context"
	"email-service/internal/config"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	textTemplate "text/template"
	"time"

	"github.com/google/uuid"
)

// StorageProvider mendefinisikan interface untuk penyimpanan file
type StorageProvider interface {
	// SaveFile menyimpan file dari data binary
	SaveFile(ctx context.Context, data []byte, filename string, contentType string) (string, error)
	// Name mengembalikan nama provider
	Name() string
}

// emailService implementasi dari domain.EmailService
type emailService struct {
	emailRepo       domain.EmailRepository
	templateRepo    domain.TemplateRepository
	deliveries      map[domain.Provider]domain.EmailDelivery
	defaultFrom     string
	defaultProvider domain.Provider
	logger          telemetry.Logger
	queueAdapter    EmailQueueAdapter              // Tambahkan queue adapter
	storageProvider StorageProvider                // Tambahkan storage provider
	trackingRepo    domain.EmailTrackingRepository // Tambahkan tracking repository
	emailEnhancer   *EmailEnhancer                 // Tambahkan email enhancer
}

// EmailQueueAdapter adalah interface untuk mengirim email ke queue
type EmailQueueAdapter interface {
	PublishEmail(ctx context.Context, emailID string, priority domain.Priority) error
	Close() error
}

// NewEmailService membuat instance baru email service
func NewEmailService(
	emailRepo domain.EmailRepository,
	templateRepo domain.TemplateRepository,
	emailDelivery domain.EmailDelivery,
	queueAdapter EmailQueueAdapter,
	storageProvider StorageProvider,
	cfg *config.Config,
	logger telemetry.Logger,
	trackingRepo domain.EmailTrackingRepository,
) domain.EmailService {
	// Siapkan map deliveries dengan satu delivery default
	deliveries := make(map[domain.Provider]domain.EmailDelivery)
	deliveries[domain.Provider(cfg.Email.DefaultProvider)] = emailDelivery

	// Siapkan base URL untuk tracking
	baseURL := fmt.Sprintf("https://%s", cfg.Server.Domain)
	if cfg.Server.Domain == "" || cfg.Server.Domain == "localhost" {
		baseURL = fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.HTTPPort)
	}

	// Buat email enhancer
	emailEnhancer := NewEmailEnhancer(baseURL, logger)

	return &emailService{
		emailRepo:       emailRepo,
		templateRepo:    templateRepo,
		deliveries:      deliveries,
		defaultFrom:     cfg.Email.From,
		defaultProvider: domain.Provider(cfg.Email.DefaultProvider),
		logger:          logger,
		queueAdapter:    queueAdapter,
		storageProvider: storageProvider,
		trackingRepo:    trackingRepo,
		emailEnhancer:   emailEnhancer,
	}
}

// SendEmail mengirim email berdasarkan request
func (s *emailService) SendEmail(ctx context.Context, request *domain.EmailRequest) (string, error) {
	// Log permintaan pengiriman email
	s.logger.Info(ctx, "Menerima permintaan pengiriman email", telemetry.Fields{
		"to":          request.To,
		"subject":     request.Subject,
		"template_id": request.TemplateID,
		"provider":    request.Provider,
		"priority":    request.Priority,
	})

	// Validasi request
	if err := s.validateEmailRequest(ctx, request); err != nil {
		s.logger.Error(ctx, "Validasi email gagal", telemetry.Fields{
			"to":      request.To,
			"subject": request.Subject,
			"error":   err.Error(),
		})
		return "", err
	}

	// Siapkan model email dari request
	email, err := s.prepareEmail(ctx, request)
	if err != nil {
		s.logger.Error(ctx, "Persiapan email gagal", telemetry.Fields{
			"to":      request.To,
			"subject": request.Subject,
			"error":   err.Error(),
		})
		return "", err
	}

	// Tambahkan fitur tracking jika ada HTML body
	if email.HTMLBody != "" && s.emailEnhancer != nil {
		s.emailEnhancer.EnhanceEmail(ctx, email)
		s.logger.Debug(ctx, "Email berhasil ditambahkan fitur tracking", telemetry.Fields{
			"email_id": email.ID,
		})
	}

	// Simpan email ke database
	if err := s.emailRepo.Save(ctx, email); err != nil {
		s.logger.Error(ctx, "Gagal menyimpan email ke database", telemetry.Fields{
			"email_id": email.ID,
			"to":       email.To,
			"subject":  email.Subject,
			"error":    err.Error(),
		})
		return "", domain.DatabaseError("Failed to save email", err)
	}

	s.logger.Info(ctx, "Email berhasil disimpan ke database", telemetry.Fields{
		"email_id": email.ID,
		"status":   email.Status,
	})

	// Kirim email ID ke antrian untuk diproses secara asynchronous
	if s.queueAdapter != nil {
		if err := s.queueAdapter.PublishEmail(ctx, email.ID, email.Priority); err != nil {
			s.logger.Error(ctx, "Gagal mengirim email ke antrian", telemetry.Fields{
				"email_id": email.ID,
				"priority": email.Priority,
				"error":    err.Error(),
			})
			// Kita tetap mengembalikan ID email meskipun gagal dikirim ke antrian
			// Email akan tetap tersimpan dengan status pending dan bisa diproses lewat job scheduler
			return email.ID, domain.InternalError("Email saved but failed to queue for delivery", err)
		}

		s.logger.Info(ctx, "Email berhasil dikirim ke antrian", telemetry.Fields{
			"email_id": email.ID,
			"priority": email.Priority,
		})
	} else {
		s.logger.Warn(ctx, "Queue adapter tidak tersedia, email tidak dikirim ke antrian", telemetry.Fields{
			"email_id": email.ID,
		})
		// Jika tidak ada queue adapter, proses langsung (untuk kasus testing atau sederhana)
		go func(emailID string) {
			// Gunakan context baru karena context asli mungkin sudah dibatalkan
			// saat goroutine ini berjalan
			bgCtx := context.Background()
			if err := s.ProcessPendingEmails(bgCtx); err != nil {
				s.logger.Error(bgCtx, "Gagal memproses email secara langsung", telemetry.Fields{
					"email_id": emailID,
					"error":    err.Error(),
				})
			}
		}(email.ID)
	}

	return email.ID, nil
}

// GetStatus mendapatkan status pengiriman email
func (s *emailService) GetStatus(ctx context.Context, id string) (*domain.EmailStatus, error) {
	s.logger.Info(ctx, "Mendapatkan status email", telemetry.Fields{
		"email_id": id,
	})

	email, err := s.emailRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error(ctx, "Gagal mendapatkan email dari database", telemetry.Fields{
			"email_id": id,
			"error":    err.Error(),
		})
		return nil, domain.DatabaseError("Failed to find email", err)
	}

	if email == nil {
		s.logger.Warn(ctx, "Email tidak ditemukan", telemetry.Fields{
			"email_id": id,
		})
		return nil, domain.NotFoundError("Email not found")
	}

	status := &domain.EmailStatus{
		ID:        email.ID,
		Status:    email.Status,
		SentAt:    email.SentAt,
		Error:     email.Error,
		UpdatedAt: email.UpdatedAt,
	}

	s.logger.Info(ctx, "Status email berhasil didapatkan", telemetry.Fields{
		"email_id": id,
		"status":   email.Status,
		"sent_at":  email.SentAt,
	})

	return status, nil
}

// ProcessPendingEmails memproses semua email yang pending
func (s *emailService) ProcessPendingEmails(ctx context.Context) error {
	// Ambil email yang pending dari database
	emails, err := s.emailRepo.FindPendingEmails(ctx, 10)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengambil email pending dari database", telemetry.Fields{
			"error": err.Error(),
		})
		return domain.DatabaseError("Failed to fetch pending emails", err)
	}

	if len(emails) == 0 {
		s.logger.Info(ctx, "Tidak ada email pending yang perlu diproses", nil)
		return nil
	}

	// Proses setiap email
	for _, email := range emails {
		// Tentukan provider yang digunakan
		provider := email.Provider
		if provider == "" {
			provider = s.defaultProvider
		}

		// Update status menjadi sending
		if err := s.emailRepo.UpdateStatus(ctx, email.ID, domain.StatusSending, ""); err != nil {
			s.logger.Error(ctx, "Gagal mengupdate status email menjadi sending", telemetry.Fields{
				"email_id": email.ID,
				"error":    err.Error(),
			})
			continue
		}

		// Cari adaptor yang sesuai dengan provider
		delivery, ok := s.deliveries[provider]
		if !ok {
			errMsg := fmt.Sprintf("Provider %s tidak tersedia", provider)
			s.logger.Error(ctx, errMsg, telemetry.Fields{
				"email_id": email.ID,
				"provider": provider,
			})
			// Update status menjadi failed
			s.emailRepo.UpdateStatus(ctx, email.ID, domain.StatusFailed, errMsg)
			continue
		}

		// Jika menggunakan template, aplikasikan template
		if email.TemplateID != "" && (email.HTMLBody == "" || email.PlainBody == "") {
			if err := s.applyTemplate(ctx, email); err != nil {
				errMsg := fmt.Sprintf("Gagal mengaplikasikan template: %v", err)
				s.logger.Error(ctx, errMsg, telemetry.Fields{
					"email_id":    email.ID,
					"template_id": email.TemplateID,
					"error":       err.Error(),
				})
				// Update status menjadi failed
				s.emailRepo.UpdateStatus(ctx, email.ID, domain.StatusFailed, errMsg)
				continue
			}
		}

		// Tambahkan tracking pixel dan rewrite links jika email memiliki HTML body
		if email.HTMLBody != "" && s.emailEnhancer != nil {
			s.emailEnhancer.EnhanceEmail(ctx, email)
			s.logger.Debug(ctx, "Email ditingkatkan dengan fitur tracking", telemetry.Fields{
				"email_id": email.ID,
			})
		}

		// Kirim email melalui adapter yang dipilih
		err := delivery.Send(ctx, email)
		if err != nil {
			errMsg := fmt.Sprintf("Gagal mengirim email: %v", err)
			s.logger.Error(ctx, errMsg, telemetry.Fields{
				"email_id": email.ID,
				"provider": provider,
				"error":    err.Error(),
			})

			// Cek jika perlu retry
			if email.RetryCount < email.MaxRetries {
				// Increment retry count dan jadikan pending lagi
				email.RetryCount++
				s.emailRepo.IncrementRetryCount(ctx, email.ID)
				s.logger.Warn(ctx, "Email akan di-retry", telemetry.Fields{
					"email_id":    email.ID,
					"retry_count": email.RetryCount,
					"max_retries": email.MaxRetries,
				})
			} else {
				// Sudah mencapai max retries, tandai sebagai failed
				s.logger.Warn(ctx, "Email gagal setelah max retries", telemetry.Fields{
					"email_id":    email.ID,
					"retry_count": email.RetryCount,
					"max_retries": email.MaxRetries,
				})
				s.emailRepo.UpdateStatus(ctx, email.ID, domain.StatusFailed, errMsg)
			}
			continue
		}

		// Email berhasil dikirim, update status
		now := time.Now()
		if err := s.emailRepo.UpdateStatus(ctx, email.ID, domain.StatusSent, ""); err != nil {
			s.logger.Error(ctx, "Gagal mengupdate status email menjadi sent", telemetry.Fields{
				"email_id": email.ID,
				"error":    err.Error(),
			})
		}
		if err := s.emailRepo.UpdateSentTime(ctx, email.ID, &now); err != nil {
			s.logger.Error(ctx, "Gagal mengupdate waktu pengiriman email", telemetry.Fields{
				"email_id": email.ID,
				"error":    err.Error(),
			})
		}

		s.logger.Info(ctx, "Email berhasil dikirim", telemetry.Fields{
			"email_id": email.ID,
			"to":       email.To,
			"subject":  email.Subject,
		})
	}

	return nil
}

// validateEmailRequest memvalidasi request email
func (s *emailService) validateEmailRequest(ctx context.Context, request *domain.EmailRequest) error {
	logFields := telemetry.Fields{
		"to":            request.To,
		"subject":       request.Subject,
		"template_id":   request.TemplateID,
		"template_name": request.TemplateName,
	}

	s.logger.Debug(ctx, "Memvalidasi permintaan email", logFields)

	// Validasi To, minimal harus ada satu penerima
	if len(request.To) == 0 {
		s.logger.Warn(ctx, "Validasi gagal: tidak ada penerima", logFields)
		return domain.ValidationError("At least one recipient is required", nil)
	}

	// Validasi Subject
	if request.Subject == "" && request.TemplateID == "" && request.TemplateName == "" {
		s.logger.Warn(ctx, "Validasi gagal: subject kosong tanpa template", logFields)
		return domain.ValidationError("Subject is required when not using a template", nil)
	}

	// Validasi body atau template
	if request.PlainBody == "" && request.HTMLBody == "" && request.TemplateID == "" && request.TemplateName == "" {
		s.logger.Warn(ctx, "Validasi gagal: tidak ada konten email", logFields)
		return domain.ValidationError("Either plain body, HTML body, or template is required", nil)
	}

	// Validasi template jika disediakan (via ID)
	if request.TemplateID != "" {
		s.logger.Debug(ctx, "Memeriksa template berdasarkan ID", telemetry.Fields{
			"template_id": request.TemplateID,
		})

		template, err := s.templateRepo.FindByID(ctx, request.TemplateID)
		if err != nil {
			s.logger.Error(ctx, "Gagal mengambil template dari database", telemetry.Fields{
				"template_id": request.TemplateID,
				"error":       err.Error(),
			})
			return domain.DatabaseError("Failed to find template", err)
		}
		if template == nil {
			s.logger.Warn(ctx, "Template tidak ditemukan", telemetry.Fields{
				"template_id": request.TemplateID,
			})
			return domain.NotFoundError("Template not found")
		}

		s.logger.Debug(ctx, "Template ditemukan", telemetry.Fields{
			"template_id":   request.TemplateID,
			"template_name": template.Name,
		})
	}

	// Validasi template jika disediakan (via Nama)
	if request.TemplateName != "" {
		s.logger.Debug(ctx, "Memeriksa template berdasarkan nama", telemetry.Fields{
			"template_name": request.TemplateName,
		})

		template, err := s.templateRepo.FindByName(ctx, request.TemplateName)
		if err != nil {
			s.logger.Error(ctx, "Gagal mengambil template dari database", telemetry.Fields{
				"template_name": request.TemplateName,
				"error":         err.Error(),
			})
			return domain.DatabaseError("Failed to find template by name", err)
		}
		if template == nil {
			s.logger.Warn(ctx, "Template tidak ditemukan berdasarkan nama", telemetry.Fields{
				"template_name": request.TemplateName,
			})
			return domain.NotFoundError(fmt.Sprintf("Template with name '%s' not found", request.TemplateName))
		}

		// Set template ID dari template yang ditemukan berdasarkan nama
		request.TemplateID = template.ID

		s.logger.Debug(ctx, "Template ditemukan berdasarkan nama", telemetry.Fields{
			"template_name": request.TemplateName,
			"template_id":   template.ID,
		})
	}

	s.logger.Debug(ctx, "Validasi email berhasil", logFields)
	return nil
}

// prepareEmail mempersiapkan model email dari request
func (s *emailService) prepareEmail(ctx context.Context, request *domain.EmailRequest) (*domain.Email, error) {
	// Buat model email
	emailID := uuid.New().String()

	s.logger.Debug(ctx, "Mempersiapkan email", telemetry.Fields{
		"email_id": emailID,
		"to":       request.To,
		"subject":  request.Subject,
	})

	email := &domain.Email{
		ID:           emailID,
		From:         s.defaultFrom,
		To:           request.To,
		Cc:           request.Cc,
		Bcc:          request.Bcc,
		Subject:      request.Subject,
		PlainBody:    request.PlainBody,
		HTMLBody:     request.HTMLBody,
		TemplateID:   request.TemplateID,
		TemplateData: request.TemplateData,
		Status:       domain.StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		MaxRetries:   3, // Default max retries
		Metadata:     request.Metadata,
	}

	// Set provider
	if request.Provider != "" {
		email.Provider = request.Provider
	} else {
		email.Provider = s.defaultProvider
		s.logger.Debug(ctx, "Menggunakan provider default", telemetry.Fields{
			"email_id": emailID,
			"provider": s.defaultProvider,
		})
	}

	// Set priority
	if request.Priority != "" {
		email.Priority = request.Priority
	} else {
		email.Priority = domain.PriorityNormal
		s.logger.Debug(ctx, "Menggunakan prioritas normal", telemetry.Fields{
			"email_id": emailID,
			"priority": domain.PriorityNormal,
		})
	}

	// Proses attachments jika ada
	if len(request.Attachments) > 0 {
		s.logger.Debug(ctx, "Memproses attachment", telemetry.Fields{
			"email_id":         emailID,
			"attachment_count": len(request.Attachments),
		})

		email.Attachments = make([]*domain.Attachment, 0, len(request.Attachments))

		// Proses attachment (download dari URL, decode base64, etc)
		for _, attachReq := range request.Attachments {
			attachment := &domain.Attachment{
				Filename:    attachReq.Filename,
				ContentType: attachReq.ContentType,
			}

			// Proses attachment dari Base64
			if attachReq.Base64Data != "" {
				data, err := base64.StdEncoding.DecodeString(attachReq.Base64Data)
				if err != nil {
					s.logger.Error(ctx, "Gagal mendecode Base64 attachment", telemetry.Fields{
						"email_id": emailID,
						"filename": attachReq.Filename,
						"error":    err.Error(),
					})
					return nil, domain.ValidationError("Invalid Base64 attachment data", err)
				}

				// Simpan file ke storage
				fileURL, err := s.storageProvider.SaveFile(ctx, data, attachReq.Filename, attachReq.ContentType)
				if err != nil {
					s.logger.Error(ctx, "Gagal menyimpan attachment ke storage", telemetry.Fields{
						"email_id": emailID,
						"filename": attachReq.Filename,
						"error":    err.Error(),
					})
					return nil, domain.InternalError("Failed to save attachment to storage", err)
				}

				attachment.URL = fileURL
			}

			// Proses attachment dari URL
			if attachReq.URL != "" {
				resp, err := http.Get(attachReq.URL)
				if err != nil {
					s.logger.Error(ctx, "Gagal mendownload attachment dari URL", telemetry.Fields{
						"email_id": emailID,
						"url":      attachReq.URL,
						"error":    err.Error(),
					})
					return nil, domain.InternalError("Failed to download attachment from URL", err)
				}
				defer resp.Body.Close()

				data, err := io.ReadAll(resp.Body)
				if err != nil {
					s.logger.Error(ctx, "Gagal membaca data attachment dari URL", telemetry.Fields{
						"email_id": emailID,
						"url":      attachReq.URL,
						"error":    err.Error(),
					})
					return nil, domain.InternalError("Failed to read attachment data from URL", err)
				}

				// Simpan file ke storage
				filename := attachment.Filename
				fileURL, err := s.storageProvider.SaveFile(ctx, data, filename, resp.Header.Get("Content-Type"))
				if err != nil {
					s.logger.Error(ctx, "Gagal menyimpan attachment ke storage", telemetry.Fields{
						"email_id": emailID,
						"filename": filename,
						"error":    err.Error(),
					})
					return nil, domain.InternalError("Failed to save attachment to storage", err)
				}

				attachment.URL = fileURL
			}

			email.Attachments = append(email.Attachments, attachment)
		}
	}

	// Proses template jika menggunakan template
	if email.TemplateID != "" {
		s.logger.Debug(ctx, "Mengaplikasikan template", telemetry.Fields{
			"email_id":    emailID,
			"template_id": email.TemplateID,
		})

		if err := s.applyTemplate(ctx, email); err != nil {
			s.logger.Error(ctx, "Gagal mengaplikasikan template", telemetry.Fields{
				"email_id":    emailID,
				"template_id": email.TemplateID,
				"error":       err.Error(),
			})
			return nil, err
		}
	}

	s.logger.Debug(ctx, "Email berhasil dipersiapkan", telemetry.Fields{
		"email_id": emailID,
		"provider": email.Provider,
		"priority": email.Priority,
	})

	return email, nil
}

// applyTemplate mengaplikasikan template ke email
func (s *emailService) applyTemplate(ctx context.Context, email *domain.Email) error {
	logFields := telemetry.Fields{
		"email_id":    email.ID,
		"template_id": email.TemplateID,
	}

	s.logger.Debug(ctx, "Mengambil template", logFields)

	template, err := s.templateRepo.FindByID(ctx, email.TemplateID)
	if err != nil {
		s.logger.Error(ctx, "Gagal mengambil template dari database", telemetry.Fields{
			"email_id":    email.ID,
			"template_id": email.TemplateID,
			"error":       err.Error(),
		})
		return domain.DatabaseError("Failed to find template", err)
	}

	if template == nil {
		s.logger.Warn(ctx, "Template tidak ditemukan", telemetry.Fields{
			"email_id":    email.ID,
			"template_id": email.TemplateID,
		})
		return domain.NotFoundError("Template not found")
	}

	// Set subject jika belum diset
	if email.Subject == "" {
		// Render subject dengan template data
		renderedSubject, err := s.renderTemplate(template.Subject, email.TemplateData)
		if err != nil {
			s.logger.Error(ctx, "Gagal merender subject template", telemetry.Fields{
				"email_id":    email.ID,
				"template_id": email.TemplateID,
				"error":       err.Error(),
			})
			return domain.InternalError("Failed to render email subject template", err)
		}

		email.Subject = renderedSubject
		s.logger.Debug(ctx, "Menggunakan subject dari template", telemetry.Fields{
			"email_id": email.ID,
			"subject":  email.Subject,
		})
	}

	// Set HTML body dari template jika belum diset
	if email.HTMLBody == "" {
		// Render HTML body dengan template data
		renderedHTMLBody, err := s.renderTemplate(template.HTMLBody, email.TemplateData)
		if err != nil {
			s.logger.Error(ctx, "Gagal merender HTML body template", telemetry.Fields{
				"email_id":    email.ID,
				"template_id": email.TemplateID,
				"error":       err.Error(),
			})
			return domain.InternalError("Failed to render email HTML body template", err)
		}

		email.HTMLBody = renderedHTMLBody
		s.logger.Debug(ctx, "Menggunakan HTML body dari template", logFields)
	}

	// Set Plain body dari template jika belum diset
	if email.PlainBody == "" {
		// Render Plain body dengan template data
		renderedPlainBody, err := s.renderTemplate(template.PlainBody, email.TemplateData)
		if err != nil {
			s.logger.Error(ctx, "Gagal merender Plain body template", telemetry.Fields{
				"email_id":    email.ID,
				"template_id": email.TemplateID,
				"error":       err.Error(),
			})
			return domain.InternalError("Failed to render email Plain body template", err)
		}

		email.PlainBody = renderedPlainBody
		s.logger.Debug(ctx, "Menggunakan Plain body dari template", logFields)
	}

	s.logger.Debug(ctx, "Template berhasil diaplikasikan", telemetry.Fields{
		"email_id":      email.ID,
		"template_id":   email.TemplateID,
		"template_name": template.Name,
	})

	return nil
}

// renderTemplate merender template dengan data yang diberikan
func (s *emailService) renderTemplate(templateText string, data interface{}) (string, error) {
	// Jika tidak ada data, kembalikan template asli
	if data == nil {
		return templateText, nil
	}

	// Pre-process template untuk mengkonversi {{var}} menjadi {{.var}}
	// Ini untuk backward compatibility dengan format template yang sudah ada
	processedTemplate := s.preprocessTemplate(templateText)

	// Gunakan text/template dari Go standard library
	tmpl, err := textTemplate.New("email").Funcs(textTemplate.FuncMap{
		// Tambahkan fungsi kustom jika diperlukan
		"formatDate": func(t time.Time) string {
			return t.Format("02 Jan 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("02 Jan 2006 15:04")
		},
		"formatCurrency": func(amount float64) string {
			return fmt.Sprintf("Rp %.2f", amount)
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}).Parse(processedTemplate)

	if err != nil {
		return "", fmt.Errorf("gagal parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("gagal mengeksekusi template: %w", err)
	}

	return buf.String(), nil
}

// preprocessTemplate mengubah format template {{var}} menjadi {{.var}}
// untuk kompatibilitas dengan template yang sudah disimpan
func (s *emailService) preprocessTemplate(templateText string) string {
	// Menggunakan regex untuk mencari pola {{variable}} dan menggantinya dengan {{.variable}}
	re := strings.NewReplacer(
		"{{app_name}}", "{{.app_name}}",
		"{{name}}", "{{.name}}",
		"{{dashboard_url}}", "{{.dashboard_url}}",
		"{{reset_url}}", "{{.reset_url}}",
		"{{subject}}", "{{.subject}}",
		"{{message}}", "{{.message}}",
	)

	return re.Replace(templateText)
}
