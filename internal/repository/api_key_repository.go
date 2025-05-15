package repository

import (
	"context"
	"database/sql"
	"email-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

// SQLAPIKeyRepository adalah implementasi APIKeyRepository menggunakan SQL database
type SQLAPIKeyRepository struct {
	db *sql.DB
}

// NewSQLAPIKeyRepository membuat instance baru dari SQLAPIKeyRepository
func NewSQLAPIKeyRepository(db *sql.DB) domain.APIKeyRepository {
	return &SQLAPIKeyRepository{
		db: db,
	}
}

// FindByKey mendapatkan API key berdasarkan nilai key-nya
func (r *SQLAPIKeyRepository) FindByKey(ctx context.Context, key string) (*domain.APIKey, error) {
	query := `
	SELECT id, api_key, name, description, service, is_active, expires_at, created_at, updated_at, last_used_at
	FROM api_keys
	WHERE api_key = $1
	`

	var apiKey domain.APIKey
	var expiresAtSQL, lastUsedAtSQL sql.NullTime

	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&apiKey.ID, &apiKey.Key, &apiKey.Name, &apiKey.Description, &apiKey.ServiceName,
		&apiKey.IsActive, &expiresAtSQL, &apiKey.CreatedAt, &apiKey.UpdatedAt, &lastUsedAtSQL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, domain.DatabaseError("Failed to find API key", err)
	}

	// Convert nullable fields
	if expiresAtSQL.Valid {
		apiKey.ExpiresAt = &expiresAtSQL.Time
	}

	if lastUsedAtSQL.Valid {
		apiKey.LastUsedAt = &lastUsedAtSQL.Time
	}

	return &apiKey, nil
}

// Save menyimpan API key baru ke database
func (r *SQLAPIKeyRepository) Save(ctx context.Context, apiKey *domain.APIKey) error {
	// Generate ID dan key jika belum ada
	if apiKey.ID == "" {
		apiKey.ID = uuid.New().String()
	}

	if apiKey.Key == "" {
		// Generate random API key dengan UUID
		apiKey.Key = uuid.New().String()
	}

	// Set waktu created dan updated
	now := time.Now()
	apiKey.CreatedAt = now
	apiKey.UpdatedAt = now

	query := `
	INSERT INTO api_keys (
		id, api_key, name, description, service, is_active, expires_at, created_at, updated_at, last_used_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var expiresAt interface{} = nil
	if apiKey.ExpiresAt != nil {
		expiresAt = apiKey.ExpiresAt
	}

	var lastUsedAt interface{} = nil
	if apiKey.LastUsedAt != nil {
		lastUsedAt = apiKey.LastUsedAt
	}

	_, err := r.db.ExecContext(
		ctx, query,
		apiKey.ID, apiKey.Key, apiKey.Name, apiKey.Description, apiKey.ServiceName,
		apiKey.IsActive, expiresAt, apiKey.CreatedAt, apiKey.UpdatedAt, lastUsedAt,
	)

	if err != nil {
		return domain.DatabaseError("Failed to save API key", err)
	}

	return nil
}

// Update memperbarui API key yang sudah ada
func (r *SQLAPIKeyRepository) Update(ctx context.Context, apiKey *domain.APIKey) error {
	apiKey.UpdatedAt = time.Now()

	query := `
	UPDATE api_keys
	SET name = $1, description = $2, service = $3, is_active = $4, expires_at = $5, updated_at = $6
	WHERE id = $7
	`

	var expiresAt interface{} = nil
	if apiKey.ExpiresAt != nil {
		expiresAt = apiKey.ExpiresAt
	}

	_, err := r.db.ExecContext(
		ctx, query,
		apiKey.Name, apiKey.Description, apiKey.ServiceName, apiKey.IsActive,
		expiresAt, apiKey.UpdatedAt, apiKey.ID,
	)

	if err != nil {
		return domain.DatabaseError("Failed to update API key", err)
	}

	return nil
}

// Delete menghapus API key berdasarkan ID
func (r *SQLAPIKeyRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)

	if err != nil {
		return domain.DatabaseError("Failed to delete API key", err)
	}

	return nil
}

// FindAll mendapatkan semua API key
func (r *SQLAPIKeyRepository) FindAll(ctx context.Context) ([]*domain.APIKey, error) {
	query := `
	SELECT id, api_key, name, description, service, is_active, expires_at, created_at, updated_at, last_used_at
	FROM api_keys
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, domain.DatabaseError("Failed to find API keys", err)
	}
	defer rows.Close()

	var apiKeys []*domain.APIKey

	for rows.Next() {
		var apiKey domain.APIKey
		var expiresAtSQL, lastUsedAtSQL sql.NullTime

		err := rows.Scan(
			&apiKey.ID, &apiKey.Key, &apiKey.Name, &apiKey.Description, &apiKey.ServiceName,
			&apiKey.IsActive, &expiresAtSQL, &apiKey.CreatedAt, &apiKey.UpdatedAt, &lastUsedAtSQL,
		)

		if err != nil {
			return nil, domain.DatabaseError("Failed to scan API key", err)
		}

		// Convert nullable fields
		if expiresAtSQL.Valid {
			apiKey.ExpiresAt = &expiresAtSQL.Time
		}

		if lastUsedAtSQL.Valid {
			apiKey.LastUsedAt = &lastUsedAtSQL.Time
		}

		apiKeys = append(apiKeys, &apiKey)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.DatabaseError("Error iterating API key rows", err)
	}

	return apiKeys, nil
}

// FindByID mendapatkan API key berdasarkan ID
func (r *SQLAPIKeyRepository) FindByID(ctx context.Context, id string) (*domain.APIKey, error) {
	query := `
	SELECT id, api_key, name, description, service, is_active, expires_at, created_at, updated_at, last_used_at
	FROM api_keys
	WHERE id = $1
	`

	var apiKey domain.APIKey
	var expiresAtSQL, lastUsedAtSQL sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&apiKey.ID, &apiKey.Key, &apiKey.Name, &apiKey.Description, &apiKey.ServiceName,
		&apiKey.IsActive, &expiresAtSQL, &apiKey.CreatedAt, &apiKey.UpdatedAt, &lastUsedAtSQL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, domain.DatabaseError("Failed to find API key", err)
	}

	// Convert nullable fields
	if expiresAtSQL.Valid {
		apiKey.ExpiresAt = &expiresAtSQL.Time
	}

	if lastUsedAtSQL.Valid {
		apiKey.LastUsedAt = &lastUsedAtSQL.Time
	}

	return &apiKey, nil
}

// UpdateLastUsed memperbarui waktu terakhir API key digunakan
func (r *SQLAPIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `
	UPDATE api_keys
	SET last_used_at = $1
	WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return domain.DatabaseError("Failed to update last used timestamp", err)
	}

	return nil
}

// FindWithPagination mendapatkan API key dengan pagination
func (r *SQLAPIKeyRepository) FindWithPagination(ctx context.Context, offset, limit int) ([]*domain.APIKey, int, error) {
	// Query untuk mendapatkan total data
	countQuery := `SELECT COUNT(*) FROM api_keys`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, domain.DatabaseError("Failed to count API keys", err)
	}

	// Query untuk mendapatkan data dengan pagination
	query := `
	SELECT id, api_key, name, description, service, is_active, expires_at, created_at, updated_at, last_used_at
	FROM api_keys
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, domain.DatabaseError("Failed to find API keys with pagination", err)
	}
	defer rows.Close()

	var apiKeys []*domain.APIKey

	for rows.Next() {
		var apiKey domain.APIKey
		var expiresAtSQL, lastUsedAtSQL sql.NullTime

		err := rows.Scan(
			&apiKey.ID, &apiKey.Key, &apiKey.Name, &apiKey.Description, &apiKey.ServiceName,
			&apiKey.IsActive, &expiresAtSQL, &apiKey.CreatedAt, &apiKey.UpdatedAt, &lastUsedAtSQL,
		)

		if err != nil {
			return nil, 0, domain.DatabaseError("Failed to scan API key", err)
		}

		// Convert nullable fields
		if expiresAtSQL.Valid {
			apiKey.ExpiresAt = &expiresAtSQL.Time
		}

		if lastUsedAtSQL.Valid {
			apiKey.LastUsedAt = &lastUsedAtSQL.Time
		}

		apiKeys = append(apiKeys, &apiKey)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, domain.DatabaseError("Error iterating API key rows", err)
	}

	return apiKeys, total, nil
}
