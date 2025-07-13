package repository

import (
	"context"
	"database/sql"
	"email-service/internal/domain"
	"time"
)

// SQLDashboardRepository adalah implementasi DashboardRepository menggunakan SQL database
type SQLDashboardRepository struct {
	db *sql.DB
}

// NewSQLDashboardRepository membuat instance baru dari SQLDashboardRepository
func NewSQLDashboardRepository(db *sql.DB) domain.DashboardRepository {
	return &SQLDashboardRepository{
		db: db,
	}
}

// GetStats mengambil statistik dashboard dari database
func (r *SQLDashboardRepository) GetStats(ctx context.Context) (*domain.DashboardStats, error) {
	stats := &domain.DashboardStats{}

	// Query untuk template statistics
	err := r.getTemplateStats(ctx, stats)
	if err != nil {
		return nil, err
	}

	// Query untuk email statistics
	err = r.getEmailStats(ctx, stats)
	if err != nil {
		return nil, err
	}

	// Query untuk email stats berdasarkan waktu
	err = r.getTimeBasedEmailStats(ctx, stats)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// getTemplateStats mengambil statistik template
func (r *SQLDashboardRepository) getTemplateStats(ctx context.Context, stats *domain.DashboardStats) error {
	query := `
		SELECT 
			COUNT(*) as total_templates,
			SUM(CASE WHEN is_active = true THEN 1 ELSE 0 END) as active_templates,
			SUM(CASE WHEN is_active = false THEN 1 ELSE 0 END) as inactive_templates
		FROM email_templates
	`

	row := r.db.QueryRowContext(ctx, query)
	err := row.Scan(&stats.TotalTemplates, &stats.ActiveTemplates, &stats.InactiveTemplates)
	if err != nil {
		return err
	}

	return nil
}

// getEmailStats mengambil statistik email berdasarkan status
func (r *SQLDashboardRepository) getEmailStats(ctx context.Context, stats *domain.DashboardStats) error {
	query := `
		SELECT 
			COUNT(*) as total_emails,
			SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as emails_sent,
			SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END) as emails_queued,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as emails_failed,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as emails_pending
		FROM emails
	`

	row := r.db.QueryRowContext(ctx, query)
	err := row.Scan(
		&stats.TotalEmails,
		&stats.EmailsSent,
		&stats.EmailsQueued,
		&stats.EmailsFailed,
		&stats.EmailsPending,
	)
	if err != nil {
		return err
	}

	return nil
}

// getTimeBasedEmailStats mengambil statistik email berdasarkan periode waktu
func (r *SQLDashboardRepository) getTimeBasedEmailStats(ctx context.Context, stats *domain.DashboardStats) error {
	now := time.Now()

	// Start of today
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Start of this week (Monday)
	weekday := today.Weekday()
	if weekday == 0 { // Sunday
		weekday = 7
	}
	thisWeek := today.AddDate(0, 0, -int(weekday-1))

	// Start of this month
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Query untuk emails hari ini
	query := `SELECT COUNT(*) FROM emails WHERE created_at >= $1`
	row := r.db.QueryRowContext(ctx, query, today)
	err := row.Scan(&stats.EmailsToday)
	if err != nil {
		return err
	}

	// Query untuk emails minggu ini
	row = r.db.QueryRowContext(ctx, query, thisWeek)
	err = row.Scan(&stats.EmailsThisWeek)
	if err != nil {
		return err
	}

	// Query untuk emails bulan ini
	row = r.db.QueryRowContext(ctx, query, thisMonth)
	err = row.Scan(&stats.EmailsThisMonth)
	if err != nil {
		return err
	}

	return nil
}
