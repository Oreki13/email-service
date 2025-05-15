package service

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"fmt"
)

// trackingService implementasi dari domain.TrackingService
type trackingService struct {
	trackingRepo domain.EmailTrackingRepository
	emailRepo    domain.EmailRepository
	logger       telemetry.Logger
}

// TrackingService membuat instance baru dari trackingService
func TrackingService(
	trackingRepo domain.EmailTrackingRepository,
	emailRepo domain.EmailRepository,
	logger telemetry.Logger,
) *trackingService {
	return &trackingService{
		trackingRepo: trackingRepo,
		emailRepo:    emailRepo,
		logger:       logger,
	}
}

// TrackEmailOpen melacak pembukaan email
func (s *trackingService) TrackEmailOpen(ctx context.Context, emailID string, userAgent, ipAddress string) error {
	// Log aktivitas tracking
	s.logger.Info(ctx, "Tracking email open", telemetry.Fields{
		"email_id":   emailID,
		"user_agent": userAgent,
		"ip_address": ipAddress,
	})

	// Cek apakah email ada
	_, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		s.logger.Error(ctx, "Failed to track email open: email not found", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return fmt.Errorf("email not found: %w", err)
	}

	// Increment open count
	err = s.trackingRepo.IncrementOpenCount(ctx, emailID, userAgent, ipAddress)
	if err != nil {
		s.logger.Error(ctx, "Failed to increment email open count", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return err
	}

	// Update email status ke opened jika belum
	err = s.emailRepo.UpdateStatus(ctx, emailID, domain.StatusOpened, "")
	if err != nil {
		s.logger.Error(ctx, "Failed to update email status to opened", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		// Tidak return error, karena increment sudah berhasil
	}

	return nil
}

// TrackEmailClick melacak klik pada link dalam email
func (s *trackingService) TrackEmailClick(ctx context.Context, emailID string, url, userAgent, ipAddress string) error {
	// Log aktivitas tracking
	s.logger.Info(ctx, "Tracking email click", telemetry.Fields{
		"email_id":   emailID,
		"url":        url,
		"user_agent": userAgent,
		"ip_address": ipAddress,
	})

	// Cek apakah email ada
	_, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		s.logger.Error(ctx, "Failed to track email click: email not found", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return fmt.Errorf("email not found: %w", err)
	}

	// Increment click count
	err = s.trackingRepo.IncrementClickCount(ctx, emailID, url, userAgent, ipAddress)
	if err != nil {
		s.logger.Error(ctx, "Failed to increment email click count", telemetry.Fields{
			"email_id": emailID,
			"url":      url,
			"error":    err.Error(),
		})
		return err
	}

	return nil
}

// GetEmailTrackingData mendapatkan data tracking untuk email tertentu
func (s *trackingService) GetEmailTrackingData(ctx context.Context, emailID string) ([]*domain.EmailTracking, error) {
	// Cek apakah email ada
	_, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get tracking data: email not found", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("email not found: %w", err)
	}

	// Dapatkan data tracking
	trackingData, err := s.trackingRepo.GetEmailTrackingData(ctx, emailID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get email tracking data", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return nil, err
	}

	return trackingData, nil
}

// Mengimplementasikan metode tambahan dari EmailService untuk tracking
// TrackEmailOpen melacak pembukaan email
func (s *emailService) TrackEmailOpen(ctx context.Context, emailID string, userAgent, ipAddress string) error {
	// Log aktivitas tracking
	s.logger.Info(ctx, "Tracking email open", telemetry.Fields{
		"email_id":   emailID,
		"user_agent": userAgent,
		"ip_address": ipAddress,
	})

	// Cek apakah email ada
	_, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		s.logger.Error(ctx, "Failed to track email open: email not found", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return fmt.Errorf("email not found: %w", err)
	}

	// Update email status ke opened
	err = s.emailRepo.UpdateStatus(ctx, emailID, domain.StatusOpened, "")
	if err != nil {
		s.logger.Error(ctx, "Failed to update email status to opened", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return err
	}

	// Implementasi tracking akan ditambahkan ketika repository tracking tersedia
	if s.trackingRepo != nil {
		err = s.trackingRepo.IncrementOpenCount(ctx, emailID, userAgent, ipAddress)
		if err != nil {
			s.logger.Error(ctx, "Failed to increment email open count", telemetry.Fields{
				"email_id": emailID,
				"error":    err.Error(),
			})
			// Tidak return error, karena update status sudah berhasil
		}
	} else {
		s.logger.Warn(ctx, "Tracking repository not available, only updating email status", telemetry.Fields{
			"email_id": emailID,
		})
	}

	return nil
}

// TrackEmailClick melacak klik pada link dalam email
func (s *emailService) TrackEmailClick(ctx context.Context, emailID string, url, userAgent, ipAddress string) error {
	// Log aktivitas tracking
	s.logger.Info(ctx, "Tracking email click", telemetry.Fields{
		"email_id":   emailID,
		"url":        url,
		"user_agent": userAgent,
		"ip_address": ipAddress,
	})

	// Cek apakah email ada
	_, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		s.logger.Error(ctx, "Failed to track email click: email not found", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return fmt.Errorf("email not found: %w", err)
	}

	// Implementasi tracking akan ditambahkan ketika repository tracking tersedia
	if s.trackingRepo != nil {
		err = s.trackingRepo.IncrementClickCount(ctx, emailID, url, userAgent, ipAddress)
		if err != nil {
			s.logger.Error(ctx, "Failed to increment email click count", telemetry.Fields{
				"email_id": emailID,
				"url":      url,
				"error":    err.Error(),
			})
			return err
		}
	} else {
		s.logger.Warn(ctx, "Tracking repository not available, skipping click tracking", telemetry.Fields{
			"email_id": emailID,
			"url":      url,
		})
	}

	return nil
}

// GetEmailTrackingData mendapatkan data tracking untuk email tertentu
func (s *emailService) GetEmailTrackingData(ctx context.Context, emailID string) ([]*domain.EmailTracking, error) {
	// Cek apakah email ada
	_, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get tracking data: email not found", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("email not found: %w", err)
	}

	// Implementasi tracking akan ditambahkan ketika repository tracking tersedia
	if s.trackingRepo != nil {
		// Dapatkan data tracking
		trackingData, err := s.trackingRepo.GetEmailTrackingData(ctx, emailID)
		if err != nil {
			s.logger.Error(ctx, "Failed to get email tracking data", telemetry.Fields{
				"email_id": emailID,
				"error":    err.Error(),
			})
			return nil, err
		}
		return trackingData, nil
	}

	s.logger.Warn(ctx, "Tracking repository not available, returning empty tracking data", telemetry.Fields{
		"email_id": emailID,
	})
	return []*domain.EmailTracking{}, nil
}
