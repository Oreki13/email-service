package repository

import (
	"context"
	"database/sql"
	"email-service/internal/domain"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SQLEmailTrackingRepository adalah implementasi EmailTrackingRepository dengan database SQL
type SQLEmailTrackingRepository struct {
	db *sql.DB
}

// NewSQLEmailTrackingRepository membuat instance baru SQLEmailTrackingRepository
func NewSQLEmailTrackingRepository(db *sql.DB) domain.EmailTrackingRepository {
	return &SQLEmailTrackingRepository{
		db: db,
	}
}

// SaveTracking menyimpan event tracking baru
func (r *SQLEmailTrackingRepository) SaveTracking(ctx context.Context, tracking *domain.EmailTracking) error {
	query := `
		INSERT INTO email_tracking (id, email_id, type, timestamp, user_agent, ip_address, url, count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	if tracking.ID == "" {
		tracking.ID = uuid.New().String()
	}

	now := time.Now()
	if tracking.CreatedAt.IsZero() {
		tracking.CreatedAt = now
	}
	if tracking.UpdatedAt.IsZero() {
		tracking.UpdatedAt = now
	}
	if tracking.Timestamp.IsZero() {
		tracking.Timestamp = now
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		tracking.ID,
		tracking.EmailID,
		tracking.Type,
		tracking.Timestamp,
		tracking.UserAgent,
		tracking.IPAddress,
		tracking.URL,
		tracking.Count,
		tracking.CreatedAt,
		tracking.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save email tracking: %w", err)
	}

	return nil
}

// IncrementOpenCount menambah jumlah pembukaan email
func (r *SQLEmailTrackingRepository) IncrementOpenCount(ctx context.Context, emailID string, userAgent, ipAddress string) error {
	// Pertama cek apakah sudah ada record untuk email ini
	var count int
	var trackingID string

	query := `SELECT id, count FROM email_tracking WHERE email_id = $1 AND type = $2`
	err := r.db.QueryRowContext(ctx, query, emailID, domain.TrackingTypeOpen).Scan(&trackingID, &count)

	now := time.Now()

	if err == sql.ErrNoRows {
		// Belum ada record, buat baru
		tracking := &domain.EmailTracking{
			ID:        uuid.New().String(),
			EmailID:   emailID,
			Type:      domain.TrackingTypeOpen,
			Timestamp: now,
			UserAgent: userAgent,
			IPAddress: ipAddress,
			Count:     1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return r.SaveTracking(ctx, tracking)
	} else if err != nil {
		return fmt.Errorf("failed to check email tracking: %w", err)
	}

	// Update existing record
	updateQuery := `
		UPDATE email_tracking 
		SET count = count + 1, updated_at = $1, timestamp = $1, user_agent = $2, ip_address = $3
		WHERE id = $4
	`
	_, err = r.db.ExecContext(ctx, updateQuery, now, userAgent, ipAddress, trackingID)
	if err != nil {
		return fmt.Errorf("failed to increment open count: %w", err)
	}

	// Update status email menjadi opened jika belum
	updateEmailQuery := `
		UPDATE emails 
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status != $1
	`
	_, err = r.db.ExecContext(ctx, updateEmailQuery, domain.StatusOpened, now, emailID)
	if err != nil {
		return fmt.Errorf("failed to update email status to opened: %w", err)
	}

	return nil
}

// IncrementClickCount menambah jumlah klik pada link dalam email
func (r *SQLEmailTrackingRepository) IncrementClickCount(ctx context.Context, emailID string, url, userAgent, ipAddress string) error {
	// Pertama cek apakah sudah ada record untuk email ini dan URL ini
	var count int
	var trackingID string

	query := `SELECT id, count FROM email_tracking WHERE email_id = $1 AND type = $2 AND url = $3`
	err := r.db.QueryRowContext(ctx, query, emailID, domain.TrackingTypeClick, url).Scan(&trackingID, &count)

	now := time.Now()

	if err == sql.ErrNoRows {
		// Belum ada record, buat baru
		tracking := &domain.EmailTracking{
			ID:        uuid.New().String(),
			EmailID:   emailID,
			Type:      domain.TrackingTypeClick,
			Timestamp: now,
			UserAgent: userAgent,
			IPAddress: ipAddress,
			URL:       url,
			Count:     1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return r.SaveTracking(ctx, tracking)
	} else if err != nil {
		return fmt.Errorf("failed to check email tracking: %w", err)
	}

	// Update existing record
	updateQuery := `
		UPDATE email_tracking 
		SET count = count + 1, updated_at = $1, timestamp = $1, user_agent = $2, ip_address = $3
		WHERE id = $4
	`
	_, err = r.db.ExecContext(ctx, updateQuery, now, userAgent, ipAddress, trackingID)
	if err != nil {
		return fmt.Errorf("failed to increment click count: %w", err)
	}

	return nil
}

// GetEmailTrackingData mendapatkan data tracking untuk email tertentu
func (r *SQLEmailTrackingRepository) GetEmailTrackingData(ctx context.Context, emailID string) ([]*domain.EmailTracking, error) {
	query := `
		SELECT id, email_id, type, timestamp, user_agent, ip_address, url, count, created_at, updated_at 
		FROM email_tracking 
		WHERE email_id = $1
		ORDER BY timestamp DESC
	`

	rows, err := r.db.QueryContext(ctx, query, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email tracking data: %w", err)
	}
	defer rows.Close()

	var result []*domain.EmailTracking

	for rows.Next() {
		tracking := &domain.EmailTracking{}
		err := rows.Scan(
			&tracking.ID,
			&tracking.EmailID,
			&tracking.Type,
			&tracking.Timestamp,
			&tracking.UserAgent,
			&tracking.IPAddress,
			&tracking.URL,
			&tracking.Count,
			&tracking.CreatedAt,
			&tracking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email tracking: %w", err)
		}

		result = append(result, tracking)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tracking rows: %w", err)
	}

	return result, nil
}

// GetEmailOpenCount mendapatkan jumlah pembukaan email
func (r *SQLEmailTrackingRepository) GetEmailOpenCount(ctx context.Context, emailID string) (int, error) {
	query := `SELECT count FROM email_tracking WHERE email_id = $1 AND type = $2`

	var count int
	err := r.db.QueryRowContext(ctx, query, emailID, domain.TrackingTypeOpen).Scan(&count)

	if err == sql.ErrNoRows {
		// Belum pernah dibuka
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to get email open count: %w", err)
	}

	return count, nil
}
