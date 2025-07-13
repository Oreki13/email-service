package domain

import "context"

// DashboardStats merepresentasikan statistik dashboard
type DashboardStats struct {
	TotalTemplates    int64 `json:"total_templates"`
	ActiveTemplates   int64 `json:"active_templates"`
	InactiveTemplates int64 `json:"inactive_templates"`
	EmailsSent        int64 `json:"emails_sent"`
	EmailsQueued      int64 `json:"emails_queued"`
	EmailsFailed      int64 `json:"emails_failed"`
	EmailsPending     int64 `json:"emails_pending"`
	TotalEmails       int64 `json:"total_emails"`
	EmailsToday       int64 `json:"emails_today"`
	EmailsThisWeek    int64 `json:"emails_this_week"`
	EmailsThisMonth   int64 `json:"emails_this_month"`
}

// DashboardRepository mendefinisikan interface untuk mengambil data dashboard
type DashboardRepository interface {
	GetStats(ctx context.Context) (*DashboardStats, error)
}

// DashboardService mendefinisikan interface untuk service dashboard
type DashboardService interface {
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)
}
