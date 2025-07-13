package service

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
)

// dashboardService implementasi dari DashboardService
type dashboardService struct {
	dashboardRepo domain.DashboardRepository
	logger        telemetry.Logger
}

// NewDashboardService membuat instance baru dari dashboard service
func NewDashboardService(
	dashboardRepo domain.DashboardRepository,
	logger telemetry.Logger,
) domain.DashboardService {
	return &dashboardService{
		dashboardRepo: dashboardRepo,
		logger:        logger,
	}
}

// GetDashboardStats mengambil statistik untuk dashboard
func (s *dashboardService) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	s.logger.Info(ctx, "Getting dashboard statistics")

	stats, err := s.dashboardRepo.GetStats(ctx)
	if err != nil {
		s.logger.Error(ctx, "Failed to get dashboard statistics", telemetry.Fields{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "Successfully retrieved dashboard statistics", telemetry.Fields{
		"total_templates":  stats.TotalTemplates,
		"active_templates": stats.ActiveTemplates,
		"emails_sent":      stats.EmailsSent,
		"emails_queued":    stats.EmailsQueued,
		"total_emails":     stats.TotalEmails,
	})

	return stats, nil
}
