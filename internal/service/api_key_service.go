package service

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"time"
)

// APIKeyService mendefinisikan service untuk mengelola API key
type APIKeyService struct {
	repo   domain.APIKeyRepository
	logger telemetry.Logger
}

// NewAPIKeyService membuat instance baru APIKeyService
func NewAPIKeyService(repo domain.APIKeyRepository, logger telemetry.Logger) *APIKeyService {
	return &APIKeyService{
		repo:   repo,
		logger: logger,
	}
}

// CreateAPIKey membuat API key baru
func (s *APIKeyService) CreateAPIKey(ctx context.Context, req *domain.APIKeyCreateRequest) (*domain.APIKey, error) {
	// Log operasi
	s.logger.Info(ctx, "Creating new API key", telemetry.Fields{
		"name":    req.Name,
		"service": req.ServiceName,
	})

	// Validasi request
	if req.Name == "" {
		return nil, domain.ValidationError("API key name cannot be empty", nil)
	}

	// Buat entity API key
	apiKey := &domain.APIKey{
		Name:        req.Name,
		Description: req.Description,
		ServiceName: req.ServiceName,
		IsActive:    true,
	}

	// Set expiry date jika ada
	if req.ExpiryDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiryDays)
		apiKey.ExpiresAt = &expiresAt
	}

	// Simpan ke database
	err := s.repo.Save(ctx, apiKey)
	if err != nil {
		s.logger.Error(ctx, "Failed to create API key", telemetry.Fields{
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "API key created successfully", telemetry.Fields{
		"id":      apiKey.ID,
		"service": apiKey.ServiceName,
	})

	return apiKey, nil
}

// GetAPIKey mendapatkan API key berdasarkan ID
func (s *APIKeyService) GetAPIKey(ctx context.Context, id string) (*domain.APIKey, error) {
	s.logger.Info(ctx, "Getting API key by ID", telemetry.Fields{
		"id": id,
	})

	apiKey, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error(ctx, "Failed to get API key", telemetry.Fields{
			"id":    id,
			"error": err.Error(),
		})
		return nil, err
	}

	if apiKey == nil {
		return nil, domain.NotFoundError("API key not found")
	}

	return apiKey, nil
}

// ListAPIKeys mendapatkan semua API key
func (s *APIKeyService) ListAPIKeys(ctx context.Context) ([]*domain.APIKey, error) {
	s.logger.Info(ctx, "Listing all API keys", nil)

	apiKeys, err := s.repo.FindAll(ctx)
	if err != nil {
		s.logger.Error(ctx, "Failed to list API keys", telemetry.Fields{
			"error": err.Error(),
		})
		return nil, err
	}

	return apiKeys, nil
}

// GetAPIKeys mendapatkan API key dengan pagination
func (s *APIKeyService) GetAPIKeys(ctx context.Context, page, limit int) ([]*domain.APIKey, int, error) {
	s.logger.Info(ctx, "Getting API keys with pagination", telemetry.Fields{
		"page":  page,
		"limit": limit,
	})

	// Calculate offset
	offset := (page - 1) * limit

	// Get API keys with pagination
	apiKeys, total, err := s.repo.FindWithPagination(ctx, offset, limit)
	if err != nil {
		s.logger.Error(ctx, "Failed to get API keys with pagination", telemetry.Fields{
			"page":  page,
			"limit": limit,
			"error": err.Error(),
		})
		return nil, 0, err
	}

	return apiKeys, total, nil
}

// UpdateAPIKey memperbarui API key
func (s *APIKeyService) UpdateAPIKey(ctx context.Context, id string, req *domain.APIKeyUpdateRequest) (*domain.APIKey, error) {
	s.logger.Info(ctx, "Updating API key", telemetry.Fields{
		"id": id,
	})

	// Cek API key exist
	apiKey, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error(ctx, "Failed to get API key for update", telemetry.Fields{
			"id":    id,
			"error": err.Error(),
		})
		return nil, err
	}

	if apiKey == nil {
		return nil, domain.NotFoundError("API key not found")
	}

	// Update field-field yang diberikan
	if req.Name != nil {
		apiKey.Name = *req.Name
	}

	if req.Description != nil {
		apiKey.Description = *req.Description
	}

	if req.ServiceName != nil {
		apiKey.ServiceName = *req.ServiceName
	}

	if req.IsActive != nil {
		apiKey.IsActive = *req.IsActive
	}

	// Update expiry date jika ada
	if req.ExpiryDays != nil {
		if *req.ExpiryDays > 0 {
			expiresAt := time.Now().AddDate(0, 0, *req.ExpiryDays)
			apiKey.ExpiresAt = &expiresAt
		} else {
			apiKey.ExpiresAt = nil // Hapus expiry date
		}
	}

	// Simpan perubahan
	err = s.repo.Update(ctx, apiKey)
	if err != nil {
		s.logger.Error(ctx, "Failed to update API key", telemetry.Fields{
			"id":    id,
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info(ctx, "API key updated successfully", telemetry.Fields{
		"id": id,
	})

	return apiKey, nil
}

// DeleteAPIKey menghapus API key
func (s *APIKeyService) DeleteAPIKey(ctx context.Context, id string) error {
	s.logger.Info(ctx, "Deleting API key", telemetry.Fields{
		"id": id,
	})

	// Cek API key exist
	apiKey, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error(ctx, "Failed to get API key for deletion", telemetry.Fields{
			"id":    id,
			"error": err.Error(),
		})
		return err
	}

	if apiKey == nil {
		return domain.NotFoundError("API key not found")
	}

	// Hapus API key
	err = s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error(ctx, "Failed to delete API key", telemetry.Fields{
			"id":    id,
			"error": err.Error(),
		})
		return err
	}

	s.logger.Info(ctx, "API key deleted successfully", telemetry.Fields{
		"id": id,
	})

	return nil
}

// VerifyAPIKey memvalidasi API key dan memperbarui last used timestamp
func (s *APIKeyService) VerifyAPIKey(ctx context.Context, key string) (*domain.APIKey, error) {
	apiKey, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if apiKey == nil {
		return nil, domain.UnauthorizedError("Invalid API key")
	}

	// Cek apakah API key aktif
	if !apiKey.IsActive {
		return nil, domain.UnauthorizedError("API key is inactive")
	}

	// Cek apakah API key expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, domain.UnauthorizedError("API key has expired")
	}

	// Update last used timestamp
	err = s.repo.UpdateLastUsed(ctx, apiKey.ID)
	if err != nil {
		s.logger.Error(ctx, "Failed to update last used timestamp", telemetry.Fields{
			"id":    apiKey.ID,
			"error": err.Error(),
		})
		// Tetap lanjut meskipun update timestamp gagal
	}

	return apiKey, nil
}
