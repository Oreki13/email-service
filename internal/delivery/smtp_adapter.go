package delivery

import (
	"bytes"
	"context"
	"crypto/tls"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jordan-wright/email"
)

// SMTPAdapter adalah adapter untuk mengirim email menggunakan SMTP
type SMTPAdapter struct {
	host     string
	port     int
	username string
	password string
	useSSL   bool
	logger   telemetry.Logger
}

// NewSMTPAdapter membuat instance baru SMTP adapter
func NewSMTPAdapter(config SMTPConfig, logger telemetry.Logger) domain.EmailDelivery {
	return &SMTPAdapter{
		host:     config.Host,
		port:     config.Port,
		username: config.Username,
		password: config.Password,
		useSSL:   config.UseSSL,
		logger:   logger,
	}
}

// Send mengirim email menggunakan SMTP
func (a *SMTPAdapter) Send(ctx context.Context, mail *domain.Email) error {
	// Buat email menggunakan library jordan-wright/email
	e := &email.Email{
		From:    mail.From,
		To:      mail.To,
		Cc:      mail.Cc,
		Bcc:     mail.Bcc,
		Subject: mail.Subject,
		Text:    []byte(mail.PlainBody),
		HTML:    []byte(mail.HTMLBody),
		Headers: make(map[string][]string),
	}

	// Tambahkan headers dari metadata
	for key, value := range mail.Metadata {
		e.Headers.Add(key, value)
	}

	// Tambahkan Message-ID
	e.Headers.Add("Message-ID", fmt.Sprintf("<%s@%s>", mail.ID, a.host))

	// Tambahkan attachments jika ada
	if len(mail.Attachments) > 0 {
		for _, attachment := range mail.Attachments {
			if len(attachment.Data) > 0 {
				// Attach dari data binary
				_, err := e.Attach(bytes.NewReader(attachment.Data), attachment.Filename, attachment.ContentType)
				if err != nil {
					return domain.ExternalServiceError("Failed to attach file", err)
				}
			} else if attachment.Path != "" {
				// Baca file dari disk jika path tersedia tetapi data tidak
				fileData, err := os.ReadFile(attachment.Path)
				if err != nil {
					a.logger.Error(ctx, "Gagal membaca file attachment", telemetry.Fields{
						"path":   attachment.Path,
						"error":  err.Error(),
						"method": "SMTP Adapter - Send",
					})
					return domain.ExternalServiceError("Failed to read attachment file", err)
				}

				// Attach menggunakan data yang dibaca dari file
				_, err = e.Attach(bytes.NewReader(fileData), attachment.Filename, attachment.ContentType)
				if err != nil {
					return domain.ExternalServiceError("Failed to attach file", err)
				}
			} else if attachment.URL != "" {
				// Jika hanya URL yang tersedia, coba download file
				a.logger.Info(ctx, "Mencoba mengunduh attachment dari URL", telemetry.Fields{
					"url":    attachment.URL,
					"method": "SMTP Adapter - Send",
				})

				// Lewati URL localhost yang tidak dapat diakses dari luar
				if strings.Contains(attachment.URL, "localhost") || strings.Contains(attachment.URL, "127.0.0.1") {
					// Extract path dari URL localhost dan coba baca dari disk
					urlPath := strings.TrimPrefix(attachment.URL, "http://localhost:8081")
					localPath := filepath.Join("./storage", urlPath)

					a.logger.Info(ctx, "URL localhost terdeteksi, mencoba membaca dari disk", telemetry.Fields{
						"url":       attachment.URL,
						"localPath": localPath,
					})

					// Baca file dari disk sebagai fallback
					fileData, err := os.ReadFile(localPath)
					if err != nil {
						a.logger.Error(ctx, "Gagal membaca file lokal attachment", telemetry.Fields{
							"path":   localPath,
							"error":  err.Error(),
							"method": "SMTP Adapter - Send",
						})
						continue // Skip attachment ini daripada gagal seluruh email
					}

					// Attach menggunakan data yang dibaca dari file
					_, err = e.Attach(bytes.NewReader(fileData), attachment.Filename, attachment.ContentType)
					if err != nil {
						a.logger.Error(ctx, "Gagal melampirkan file setelah membaca lokal", telemetry.Fields{
							"error": err.Error(),
						})
						continue // Skip attachment ini daripada gagal seluruh email
					}
				} else {
					// Download file dari URL yang bisa diakses publik
					resp, err := http.Get(attachment.URL)
					if err != nil {
						a.logger.Error(ctx, "Gagal mengunduh attachment dari URL", telemetry.Fields{
							"url":    attachment.URL,
							"error":  err.Error(),
							"method": "SMTP Adapter - Send",
						})
						continue // Skip attachment ini daripada gagal seluruh email
					}
					defer resp.Body.Close()

					// Baca response body
					fileData, err := io.ReadAll(resp.Body)
					if err != nil {
						a.logger.Error(ctx, "Gagal membaca data attachment dari URL", telemetry.Fields{
							"url":    attachment.URL,
							"error":  err.Error(),
							"method": "SMTP Adapter - Send",
						})
						continue // Skip attachment ini daripada gagal seluruh email
					}

					// Attach menggunakan data yang diunduh
					_, err = e.Attach(bytes.NewReader(fileData), attachment.Filename, attachment.ContentType)
					if err != nil {
						a.logger.Error(ctx, "Gagal melampirkan file setelah mengunduh", telemetry.Fields{
							"error": err.Error(),
						})
						continue // Skip attachment ini daripada gagal seluruh email
					}
				}
			}
		}
	}

	// Setup auth
	auth := smtp.PlainAuth("", a.username, a.password, a.host)

	// Alamat server SMTP
	serverAddr := fmt.Sprintf("%s:%d", a.host, a.port)

	// Buat child context dengan timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Channel untuk hasil pengiriman
	errCh := make(chan error, 1)

	// Kirim email dalam goroutine
	go func() {
		var err error
		if a.useSSL {
			// Implementasi pengiriman dengan TLS
			// Membuat konfigurasi TLS
			tlsConfig := &tls.Config{
				ServerName:         a.host,
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: false, // Jangan skip verifikasi sertifikat
			}

			// Untuk Gmail spesifik, kita perlu menambahkan ini:
			if a.host == "smtp.gmail.com" {
				// Pengiriman khusus untuk Gmail dengan TLS
				err = e.SendWithTLS(serverAddr, auth, tlsConfig)
				errCh <- err
			} else {
				// Untuk server SMTP lain
				err = e.SendWithTLS(serverAddr, auth, tlsConfig)
				errCh <- err
			}
		} else {
			err = e.Send(serverAddr, auth)
			errCh <- err
		}
	}()

	// Tunggu hasil pengiriman atau timeout
	select {
	case err := <-errCh:
		if err != nil {
			return domain.ExternalServiceError("Failed to send email via SMTP", err)
		}
		return nil
	case <-timeoutCtx.Done():
		return domain.ExternalServiceError("SMTP send operation timed out", timeoutCtx.Err())
	}
}

// Name mengembalikan nama provider
func (a *SMTPAdapter) Name() string {
	return string(domain.ProviderSMTP)
}
