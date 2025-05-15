package repository

import (
	"context"
	"database/sql"
	"email-service/internal/domain"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SQLTemplateRepository adalah implementasi TemplateRepository menggunakan SQL database
type SQLTemplateRepository struct {
	db *sql.DB
}

// NewSQLTemplateRepository membuat instance baru dari SQLTemplateRepository
func NewSQLTemplateRepository(db *sql.DB) domain.TemplateRepository {
	return &SQLTemplateRepository{
		db: db,
	}
}

// FindByID mendapatkan template berdasarkan ID
func (r *SQLTemplateRepository) FindByID(ctx context.Context, id string) (*domain.Template, error) {
	query := `
	SELECT 
		id, name, description, subject, plain_body, html_body, 
		variables, created_at, updated_at
	FROM email_templates
	WHERE id = $1 AND is_active = TRUE
	`

	var template domain.Template
	var variablesJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID, &template.Name, &template.Description, &template.Subject,
		&template.PlainBody, &template.HTMLBody, &variablesJSON, &template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, domain.DatabaseError("Failed to find template", err)
	}

	// Unmarshal variables jika ada
	if len(variablesJSON) > 0 {
		var variables []string
		if err = json.Unmarshal(variablesJSON, &variables); err != nil {
			return nil, domain.DatabaseError("Failed to unmarshal variables", err)
		}
		// Tambahkan ke domain.Template jika diperlukan
		// (Tidak ada dalam definisi template saat ini, mungkin perlu ditambahkan)
	}

	return &template, nil
}

// Save menyimpan template baru atau memperbarui yang sudah ada
func (r *SQLTemplateRepository) Save(ctx context.Context, template *domain.Template) error {
	var variables []string // Daftar variabel yang digunakan dalam template

	// Seharusnya mengekstrak variabel dari template menggunakan regex
	// Misalnya mencari pola {{variable}} dalam konten template
	// Ini hanya contoh sederhana
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return domain.DatabaseError("Failed to marshal variables", err)
	}

	// Cek apakah template sudah ada (update) atau baru (insert)
	var exists bool
	if template.ID != "" {
		err := r.db.QueryRowContext(ctx, "SELECT 1 FROM email_templates WHERE id = $1", template.ID).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			return domain.DatabaseError("Failed to check template existence", err)
		}
		exists = err != sql.ErrNoRows
	}

	if exists {
		// Update template yang sudah ada
		query := `
		UPDATE email_templates
		SET name = $1, description = $2, subject = $3, plain_body = $4, 
			html_body = $5, variables = $6, updated_at = $7, version = version + 1
		WHERE id = $8
		`

		_, err = r.db.ExecContext(
			ctx, query,
			template.Name, template.Description, template.Subject, template.PlainBody,
			template.HTMLBody, variablesJSON, time.Now(), template.ID,
		)
	} else {
		// Buat template baru dengan ID baru jika belum ada
		if template.ID == "" {
			template.ID = uuid.New().String()
		}

		// Set waktu created dan updated
		now := time.Now()
		template.CreatedAt = now
		template.UpdatedAt = now

		// Insert template baru
		query := `
		INSERT INTO email_templates (
			id, name, description, subject, plain_body, html_body,
			variables, created_at, updated_at, created_by, is_active, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, TRUE, 1)
		`

		_, err = r.db.ExecContext(
			ctx, query,
			template.ID, template.Name, template.Description, template.Subject,
			template.PlainBody, template.HTMLBody, variablesJSON, template.CreatedAt,
			template.UpdatedAt, nil, // created_by bisa null atau user ID
		)
	}

	if err != nil {
		return domain.DatabaseError("Failed to save template", err)
	}

	return nil
}

// Delete menghapus template (soft delete dengan mengubah is_active)
func (r *SQLTemplateRepository) Delete(ctx context.Context, id string) error {
	query := `
	UPDATE email_templates
	SET is_active = FALSE, updated_at = $1
	WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return domain.DatabaseError("Failed to delete template", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.DatabaseError("Failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return domain.NotFoundError("Template not found")
	}

	return nil
}

// FindAll mendapatkan semua template aktif
func (r *SQLTemplateRepository) FindAll(ctx context.Context) ([]*domain.Template, error) {
	query := `
	SELECT 
		id, name, description, subject, plain_body, html_body, 
		variables, created_at, updated_at
	FROM email_templates
	WHERE is_active = TRUE
	ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, domain.DatabaseError("Failed to query templates", err)
	}
	defer rows.Close()

	var templates []*domain.Template
	for rows.Next() {
		var template domain.Template
		var variablesJSON []byte

		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &template.Subject,
			&template.PlainBody, &template.HTMLBody, &variablesJSON, &template.CreatedAt,
			&template.UpdatedAt,
		)

		if err != nil {
			return nil, domain.DatabaseError("Failed to scan template", err)
		}

		templates = append(templates, &template)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.DatabaseError("Error iterating template rows", err)
	}

	return templates, nil
}

// FindByName mendapatkan template berdasarkan nama
func (r *SQLTemplateRepository) FindByName(ctx context.Context, name string) (*domain.Template, error) {
	query := `
	SELECT 
		id, name, description, subject, plain_body, html_body, 
		variables, created_at, updated_at
	FROM email_templates
	WHERE name = $1 AND is_active = TRUE
	`

	var template domain.Template
	var variablesJSON []byte

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&template.ID, &template.Name, &template.Description, &template.Subject,
		&template.PlainBody, &template.HTMLBody, &variablesJSON, &template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, domain.DatabaseError("Failed to find template by name", err)
	}

	return &template, nil
}

// Pagination mendapatkan template dengan pagination
func (r *SQLTemplateRepository) Pagination(ctx context.Context, limit, offset int) ([]*domain.Template, error) {
	query := `
	SELECT 
		id, name, description, subject, plain_body, html_body, 
		variables, created_at, updated_at
	FROM email_templates
	WHERE is_active = TRUE
	ORDER BY name
	LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, domain.DatabaseError("Failed to query templates with pagination", err)
	}
	defer rows.Close()

	var templates []*domain.Template
	for rows.Next() {
		var template domain.Template
		var variablesJSON []byte

		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &template.Subject,
			&template.PlainBody, &template.HTMLBody, &variablesJSON, &template.CreatedAt,
			&template.UpdatedAt,
		)

		if err != nil {
			return nil, domain.DatabaseError("Failed to scan template", err)
		}

		templates = append(templates, &template)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.DatabaseError("Error iterating template rows", err)
	}

	return templates, nil
}

// Count menghitung jumlah total template aktif
func (r *SQLTemplateRepository) Count(ctx context.Context) (int, error) {
	query := `
	SELECT COUNT(*)
	FROM email_templates
	WHERE is_active = TRUE
	`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, domain.DatabaseError("Failed to count templates", err)
	}

	return count, nil
}
