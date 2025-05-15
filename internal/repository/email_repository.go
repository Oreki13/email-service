package repository

import (
	"context"
	"database/sql"
	"email-service/internal/domain"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SQLEmailRepository adalah implementasi EmailRepository menggunakan SQL database
type SQLEmailRepository struct {
	db *sql.DB
}

// NewSQLEmailRepository membuat instance baru dari SQLEmailRepository
func NewSQLEmailRepository(db *sql.DB) domain.EmailRepository {
	return &SQLEmailRepository{
		db: db,
	}
}

// Save menyimpan email baru ke database
func (r *SQLEmailRepository) Save(ctx context.Context, email *domain.Email) error {
	// Marshal arrays ke JSON
	toJSON, err := json.Marshal(email.To)
	if err != nil {
		return domain.DatabaseError("Failed to marshal To field", err)
	}

	ccJSON, err := json.Marshal(email.Cc)
	if err != nil {
		return domain.DatabaseError("Failed to marshal Cc field", err)
	}

	bccJSON, err := json.Marshal(email.Bcc)
	if err != nil {
		return domain.DatabaseError("Failed to marshal Bcc field", err)
	}

	// Marshal template data ke JSON jika ada
	var templateDataJSON []byte
	if email.TemplateData != nil {
		// Pendekatan yang lebih aman:
		// 1. Marshal ke JSON string terlebih dahulu
		// 2. Unmarshal kembali ke map[string]interface{} untuk memastikan struktur yang valid
		// 3. Sanitasi data untuk memastikan semua nilai kompatibel dengan JSON
		// 4. Marshal kembali ke JSON untuk disimpan

		// Langkah 1: Marshal ke JSON dulu
		tempJSON, err := json.Marshal(email.TemplateData)
		if err != nil {
			return domain.DatabaseError("Failed to marshal TemplateData (step 1)", err)
		}

		// Langkah 2: Unmarshal ke map
		var tempMap map[string]interface{}
		if err := json.Unmarshal(tempJSON, &tempMap); err != nil {
			// Jika gagal dikonversi ke map, gunakan cara alternatif
			// Buat map baru dan simpan string JSON sebagai satu nilai
			tempMap = map[string]interface{}{
				"data": string(tempJSON),
			}
		}

		// Langkah 3: Sanitasi data
		sanitizedData := make(map[string]interface{})
		for key, value := range tempMap {
			switch v := value.(type) {
			case nil:
				sanitizedData[key] = nil
			case string, bool, float64, int, int64, float32:
				sanitizedData[key] = v // Tipe primitif JSON
			default:
				// Untuk tipe kompleks, konversi ke string
				strValue, err := json.Marshal(v)
				if err != nil {
					sanitizedData[key] = fmt.Sprintf("%v", v)
				} else {
					sanitizedData[key] = string(strValue)
				}
			}
		}

		// Langkah 4: Marshal kembali ke JSON
		templateDataJSON, err = json.Marshal(sanitizedData)
		if err != nil {
			// Jika masih gagal, gunakan objek JSON kosong sebagai fallback
			templateDataJSON = []byte("{}")
			// Log error tapi jangan gagalkan seluruh operasi
			fmt.Printf("Warning: Failed to marshal template data: %v. Using empty object instead.\n", err)
		}
	} else {
		// Jika template data kosong, gunakan objek JSON kosong
		templateDataJSON = []byte("{}")
	}

	// Marshal metadata ke JSON jika ada
	var metadataJSON []byte
	if email.Metadata != nil {
		metadataJSON, err = json.Marshal(email.Metadata)
		if err != nil {
			return domain.DatabaseError("Failed to marshal Metadata", err)
		}
	} else {
		// Jika metadata kosong, gunakan objek JSON kosong
		metadataJSON = []byte("{}")
	}

	// Siapkan query untuk insert
	query := `
	INSERT INTO emails (
		id, from_email, to_emails, cc_emails, bcc_emails, subject,
		plain_body, html_body, template_id, template_data, status,
		priority, provider, created_at, updated_at, retry_count,
		max_retries, metadata
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	// Execute query
	_, err = r.db.ExecContext(
		ctx, query,
		email.ID, email.From, toJSON, ccJSON, bccJSON, email.Subject,
		email.PlainBody, email.HTMLBody, email.TemplateID, templateDataJSON, email.Status,
		email.Priority, email.Provider, email.CreatedAt, email.UpdatedAt, email.RetryCount,
		email.MaxRetries, metadataJSON,
	)

	if err != nil {
		return domain.DatabaseError(fmt.Sprintf("Failed to save email: %v", err), err)
	}

	// Jika email memiliki attachment, simpan ke tabel email_attachments
	if len(email.Attachments) > 0 {
		for _, attachment := range email.Attachments {
			attachmentID := uuid.New().String()
			query := `
			INSERT INTO email_attachments (
				id, email_id, filename, content_type, size, storage_path, content, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`

			size := len(attachment.Data)
			_, err = r.db.ExecContext(
				ctx, query,
				attachmentID, email.ID, attachment.Filename, attachment.ContentType,
				size, attachment.Path, attachment.Data, time.Now(),
			)

			if err != nil {
				// Log error tapi jangan menghentikan proses
				// Dalam situasi nyata, mungkin ingin melakukan rollback atau mencoba lagi
				fmt.Printf("Failed to save attachment %s: %v\n", attachment.Filename, err)
			}
		}
	}

	return nil
}

// FindByID mendapatkan email berdasarkan ID
func (r *SQLEmailRepository) FindByID(ctx context.Context, id string) (*domain.Email, error) {
	query := `
	SELECT 
		id, from_email, to_emails, cc_emails, bcc_emails, subject,
		plain_body, html_body, template_id, template_data, status,
		priority, provider, sent_at, created_at, updated_at, retry_count,
		max_retries, error_message, metadata
	FROM emails
	WHERE id = $1
	`

	var email domain.Email
	var toJSON, ccJSON, bccJSON, templateDataJSON, metadataJSON []byte
	var sentAtSQL sql.NullTime
	var errorMessageSQL sql.NullString // Menggunakan sql.NullString untuk handle NULL

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&email.ID, &email.From, &toJSON, &ccJSON, &bccJSON, &email.Subject,
		&email.PlainBody, &email.HTMLBody, &email.TemplateID, &templateDataJSON, &email.Status,
		&email.Priority, &email.Provider, &sentAtSQL, &email.CreatedAt, &email.UpdatedAt, &email.RetryCount,
		&email.MaxRetries, &errorMessageSQL, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, domain.DatabaseError("Failed to find email", err)
	}

	// Konversi sql.NullString ke string jika valid
	if errorMessageSQL.Valid {
		email.Error = errorMessageSQL.String
	}

	// Unmarshal JSON fields
	if err = json.Unmarshal(toJSON, &email.To); err != nil {
		return nil, domain.DatabaseError("Failed to unmarshal To field", err)
	}

	if len(ccJSON) > 0 {
		if err = json.Unmarshal(ccJSON, &email.Cc); err != nil {
			return nil, domain.DatabaseError("Failed to unmarshal Cc field", err)
		}
	}

	if len(bccJSON) > 0 {
		if err = json.Unmarshal(bccJSON, &email.Bcc); err != nil {
			return nil, domain.DatabaseError("Failed to unmarshal Bcc field", err)
		}
	}

	if len(templateDataJSON) > 0 {
		var templateData map[string]interface{}
		if err = json.Unmarshal(templateDataJSON, &templateData); err != nil {
			return nil, domain.DatabaseError("Failed to unmarshal TemplateData", err)
		}
		email.TemplateData = templateData
	}

	if len(metadataJSON) > 0 {
		if err = json.Unmarshal(metadataJSON, &email.Metadata); err != nil {
			return nil, domain.DatabaseError("Failed to unmarshal Metadata", err)
		}
	}

	// Convert nullable fields
	if sentAtSQL.Valid {
		email.SentAt = &sentAtSQL.Time
	}

	// Get attachments
	email.Attachments, err = r.getAttachments(ctx, email.ID)
	if err != nil {
		return nil, err
	}

	return &email, nil
}

// getAttachments mendapatkan semua lampiran untuk email tertentu
func (r *SQLEmailRepository) getAttachments(ctx context.Context, emailID string) ([]*domain.Attachment, error) {
	query := `
	SELECT id, filename, content_type, storage_path, content
	FROM email_attachments
	WHERE email_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, emailID)
	if err != nil {
		return nil, domain.DatabaseError("Failed to query attachments", err)
	}
	defer rows.Close()

	var attachments []*domain.Attachment
	for rows.Next() {
		var attachment domain.Attachment
		var id string
		err := rows.Scan(&id, &attachment.Filename, &attachment.ContentType, &attachment.Path, &attachment.Data)
		if err != nil {
			return nil, domain.DatabaseError("Failed to scan attachment", err)
		}
		attachments = append(attachments, &attachment)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.DatabaseError("Error iterating attachment rows", err)
	}

	return attachments, nil
}

// UpdateStatus memperbarui status email
func (r *SQLEmailRepository) UpdateStatus(ctx context.Context, id string, status domain.DeliveryStatus, errorMsg string) error {
	query := `
	UPDATE emails
	SET status = $1, updated_at = $2, error_message = $3
	WHERE id = $4
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, status, now, errorMsg, id)
	if err != nil {
		return domain.DatabaseError("Failed to update email status", err)
	}

	// Jika status adalah sent, update sent_at
	if status == domain.StatusSent {
		query = `
		UPDATE emails
		SET sent_at = $1
		WHERE id = $2
		`
		_, err := r.db.ExecContext(ctx, query, now, id)
		if err != nil {
			return domain.DatabaseError("Failed to update sent_at", err)
		}
	}

	// Catat event untuk perubahan status
	return r.logEmailEvent(ctx, id, string(status), nil)
}

// logEmailEvent mencatat event email ke tabel email_events
func (r *SQLEmailRepository) logEmailEvent(ctx context.Context, emailID, eventType string, data interface{}) error {
	var dataJSON []byte
	var err error

	if data != nil {
		dataJSON, err = json.Marshal(data)
		if err != nil {
			return domain.DatabaseError("Failed to marshal event data", err)
		}
	} else {
		// Jika data nil, gunakan JSON objek kosong sebagai default
		dataJSON = []byte("{}")
	}

	query := `
	INSERT INTO email_events (id, email_id, event_type, timestamp, data)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err = r.db.ExecContext(
		ctx, query,
		uuid.New().String(), emailID, eventType, time.Now(), dataJSON,
	)

	if err != nil {
		return domain.DatabaseError("Failed to log email event", err)
	}

	return nil
}

// FindPendingEmails mendapatkan daftar email yang berstatus pending
func (r *SQLEmailRepository) FindPendingEmails(ctx context.Context, limit int) ([]*domain.Email, error) {
	query := `
	SELECT 
		id, from_email, to_emails, cc_emails, bcc_emails, subject,
		plain_body, html_body, template_id, template_data, status,
		priority, provider, sent_at, created_at, updated_at, retry_count,
		max_retries, error_message, metadata
	FROM emails
	WHERE status = $1
	ORDER BY priority DESC, created_at ASC
	LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, domain.StatusPending, limit)
	if err != nil {
		return nil, domain.DatabaseError("Failed to query pending emails", err)
	}
	defer rows.Close()

	var emails []*domain.Email
	for rows.Next() {
		var email domain.Email
		var toJSON, ccJSON, bccJSON, templateDataJSON, metadataJSON []byte
		var sentAtSQL sql.NullTime
		var errorMessageSQL sql.NullString // Menggunakan sql.NullString untuk handle NULL

		err := rows.Scan(
			&email.ID, &email.From, &toJSON, &ccJSON, &bccJSON, &email.Subject,
			&email.PlainBody, &email.HTMLBody, &email.TemplateID, &templateDataJSON, &email.Status,
			&email.Priority, &email.Provider, &sentAtSQL, &email.CreatedAt, &email.UpdatedAt, &email.RetryCount,
			&email.MaxRetries, &errorMessageSQL, &metadataJSON,
		)

		if err != nil {
			return nil, domain.DatabaseError("Failed to scan email", err)
		}

		// Konversi sql.NullString ke string jika valid
		if errorMessageSQL.Valid {
			email.Error = errorMessageSQL.String
		}

		// Unmarshal JSON fields
		if err = json.Unmarshal(toJSON, &email.To); err != nil {
			return nil, domain.DatabaseError("Failed to unmarshal To field", err)
		}

		if len(ccJSON) > 0 {
			if err = json.Unmarshal(ccJSON, &email.Cc); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal Cc field", err)
			}
		}

		if len(bccJSON) > 0 {
			if err = json.Unmarshal(bccJSON, &email.Bcc); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal Bcc field", err)
			}
		}

		if len(templateDataJSON) > 0 {
			var templateData map[string]interface{}
			if err = json.Unmarshal(templateDataJSON, &templateData); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal TemplateData", err)
			}
			email.TemplateData = templateData
		}

		if len(metadataJSON) > 0 {
			if err = json.Unmarshal(metadataJSON, &email.Metadata); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal Metadata", err)
			}
		}

		// Convert nullable fields
		if sentAtSQL.Valid {
			email.SentAt = &sentAtSQL.Time
		}

		// Get attachments (optional, bisa skip untuk optimasi)
		email.Attachments, err = r.getAttachments(ctx, email.ID)
		if err != nil {
			return nil, err
		}

		emails = append(emails, &email)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.DatabaseError("Error iterating email rows", err)
	}

	return emails, nil
}

// FindByStatus mendapatkan daftar email berdasarkan status dengan pagination
func (r *SQLEmailRepository) FindByStatus(ctx context.Context, status domain.DeliveryStatus, limit, offset int) ([]*domain.Email, error) {
	query := `
	SELECT 
		id, from_email, to_emails, cc_emails, bcc_emails, subject,
		plain_body, html_body, template_id, template_data, status,
		priority, provider, sent_at, created_at, updated_at, retry_count,
		max_retries, error_message, metadata
	FROM emails
	WHERE status = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, domain.DatabaseError("Failed to query emails by status", err)
	}
	defer rows.Close()

	var emails []*domain.Email
	for rows.Next() {
		var email domain.Email
		var toJSON, ccJSON, bccJSON, templateDataJSON, metadataJSON []byte
		var sentAtSQL sql.NullTime
		var errorMessageSQL sql.NullString // Menggunakan sql.NullString untuk handle NULL

		err := rows.Scan(
			&email.ID, &email.From, &toJSON, &ccJSON, &bccJSON, &email.Subject,
			&email.PlainBody, &email.HTMLBody, &email.TemplateID, &templateDataJSON, &email.Status,
			&email.Priority, &email.Provider, &sentAtSQL, &email.CreatedAt, &email.UpdatedAt, &email.RetryCount,
			&email.MaxRetries, &errorMessageSQL, &metadataJSON,
		)

		if err != nil {
			return nil, domain.DatabaseError("Failed to scan email", err)
		}

		// Konversi sql.NullString ke string jika valid
		if errorMessageSQL.Valid {
			email.Error = errorMessageSQL.String
		}

		// Unmarshal JSON fields
		if err = json.Unmarshal(toJSON, &email.To); err != nil {
			return nil, domain.DatabaseError("Failed to unmarshal To field", err)
		}

		if len(ccJSON) > 0 {
			if err = json.Unmarshal(ccJSON, &email.Cc); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal Cc field", err)
			}
		}

		if len(bccJSON) > 0 {
			if err = json.Unmarshal(bccJSON, &email.Bcc); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal Bcc field", err)
			}
		}

		if len(templateDataJSON) > 0 {
			var templateData map[string]interface{}
			if err = json.Unmarshal(templateDataJSON, &templateData); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal TemplateData", err)
			}
			email.TemplateData = templateData
		}

		if len(metadataJSON) > 0 {
			if err = json.Unmarshal(metadataJSON, &email.Metadata); err != nil {
				return nil, domain.DatabaseError("Failed to unmarshal Metadata", err)
			}
		}

		// Convert nullable fields
		if sentAtSQL.Valid {
			email.SentAt = &sentAtSQL.Time
		}

		// Get attachments
		email.Attachments, err = r.getAttachments(ctx, email.ID)
		if err != nil {
			return nil, err
		}

		emails = append(emails, &email)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.DatabaseError("Error iterating email rows", err)
	}

	return emails, nil
}

// IncrementRetryCount meningkatkan jumlah retry untuk email
func (r *SQLEmailRepository) IncrementRetryCount(ctx context.Context, id string) error {
	query := `
	UPDATE emails
	SET retry_count = retry_count + 1, updated_at = $1
	WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return domain.DatabaseError("Failed to increment retry count", err)
	}

	return nil
}

// UpdateSentTime memperbarui waktu pengiriman email
func (r *SQLEmailRepository) UpdateSentTime(ctx context.Context, id string, sentAt *time.Time) error {
	query := `
	UPDATE emails
	SET sent_at = $1, updated_at = $2
	WHERE id = $3
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, sentAt, now, id)
	if err != nil {
		return domain.DatabaseError("Failed to update sent time", err)
	}

	return nil
}

// CountByStatus menghitung jumlah email berdasarkan status
func (r *SQLEmailRepository) CountByStatus(ctx context.Context, status domain.DeliveryStatus) (int, error) {
	query := `
	SELECT COUNT(*)
	FROM emails
	WHERE status = $1
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, domain.DatabaseError("Failed to count emails by status", err)
	}

	return count, nil
}
