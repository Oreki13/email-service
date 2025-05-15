package handler

import (
	"context"
	"email-service/internal/config"
	"email-service/internal/domain"
	"email-service/internal/middleware"
	"email-service/pkg/telemetry"
	"math"
	"strconv"

	"email-service/internal/dto"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Inisialisasi validator untuk validasi struct
var validate = validator.New()

// APIKeyServicer mendefinisikan interface untuk service API Key
type APIKeyServicer interface {
	CreateAPIKey(ctx context.Context, req *domain.APIKeyCreateRequest) (*domain.APIKey, error)
	GetAPIKeys(ctx context.Context, page, limit int) ([]*domain.APIKey, int, error)
	GetAPIKey(ctx context.Context, id string) (*domain.APIKey, error)
	UpdateAPIKey(ctx context.Context, id string, req *domain.APIKeyUpdateRequest) (*domain.APIKey, error)
	DeleteAPIKey(ctx context.Context, id string) error
	VerifyAPIKey(ctx context.Context, key string) (*domain.APIKey, error)
	// Metode yang tidak digunakan lagi (deprecated)
	// ListAPIKeys(ctx context.Context) ([]*domain.APIKey, error)
}

// APIKeyHandler menangani HTTP request terkait API key
type APIKeyHandler struct {
	apiKeyService APIKeyServicer
	logger        telemetry.Logger
}

// NewAPIKeyHandler membuat instance baru APIKeyHandler
func NewAPIKeyHandler(apiKeyService APIKeyServicer, logger telemetry.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
		logger:        logger,
	}
}

// RegisterAPIKeyRoutes mendaftarkan endpoint API untuk pengelolaan API key
func RegisterAPIKeyRoutes(router fiber.Router, apiKeyService APIKeyServicer, logger telemetry.Logger, cfg *config.Config) {
	handler := NewAPIKeyHandler(apiKeyService, logger)

	apiKeyRoutes := router.Group("/api-keys")

	// Mendaftarkan endpoint dengan authorisasi admin
	apiKeyRoutes.Use(AdminAuthMiddleware(cfg, logger))
	apiKeyRoutes.Post("/", handler.CreateAPIKey)
	apiKeyRoutes.Get("/", handler.GetAPIKeys) // Menggunakan GetAPIKeys yang mendukung pagination
	apiKeyRoutes.Get("/:id", handler.GetAPIKey)
	apiKeyRoutes.Put("/:id", handler.UpdateAPIKey)
	apiKeyRoutes.Delete("/:id", handler.DeleteAPIKey)
}

// AdminAuthMiddleware adalah middleware untuk memvalidasi bahwa request berasal dari admin
func AdminAuthMiddleware(cfg *config.Config, logger telemetry.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Dapatkan trace ID dari konteks
		traceID := c.Locals(middleware.TraceIDKey).(string)
		ctx := context.Background()
		ctx = context.WithValue(ctx, middleware.TraceIDKey, traceID)

		// Ambil token dari header atau cookie
		token := c.Get("X-Admin-Token")
		if token == "" {
			// Cek juga dari cookie
			token = c.Cookies("admin_token")
		}

		// Validasi token admin menggunakan konfigurasi
		expectedToken := cfg.Auth.AdminAPIToken
		if token == "" || token != expectedToken {
			logger.Info(ctx, "Unauthorized admin access attempt", telemetry.Fields{
				"traceID": traceID,
				"path":    c.Path(),
			})

			return domain.UnauthorizedError("Unauthorized access")
		}

		logger.Info(ctx, "Admin authentication successful", telemetry.Fields{
			"traceID": traceID,
			"path":    c.Path(),
		})

		return c.Next()
	}
}

// CreateAPIKey menangani request pembuatan API key baru
// @Summary Membuat API key baru
// @Description Membuat API key baru untuk akses ke service
// @Tags API Keys
// @Accept json
// @Produce json
// @Param request body domain.APIKeyCreateRequest true "Data API key baru"
// @Success 201 {object} domain.APIKey
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/api-keys [post]
func (h *APIKeyHandler) CreateAPIKey(c *fiber.Ctx) error {
	traceID := c.Locals(middleware.TraceIDKey).(string)
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.TraceIDKey, traceID)

	var req domain.APIKeyCreateRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error(ctx, "Failed to parse request body", telemetry.Fields{
			"handler": "CreateAPIKey",
			"error":   err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validasi request
	if err := validate.Struct(req); err != nil {
		h.logger.Error(ctx, "Request validation failed", telemetry.Fields{
			"handler": "CreateAPIKey",
			"error":   err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Invalid request data",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "Creating new API key", telemetry.Fields{
		"handler":     "CreateAPIKey",
		"serviceName": req.ServiceName,
	})

	apiKey, err := h.apiKeyService.CreateAPIKey(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to create API key", telemetry.Fields{
			"handler": "CreateAPIKey",
			"error":   err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to create API key",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "API key created successfully", telemetry.Fields{
		"handler":  "CreateAPIKey",
		"apiKeyID": apiKey.ID,
		"service":  apiKey.ServiceName,
	})

	return c.Status(fiber.StatusCreated).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "API key created successfully",
		Data:    apiKey,
	})
}

// GetAPIKey menangani request untuk mendapatkan API key berdasarkan ID
// @Summary Mendapatkan detail API key
// @Description Mendapatkan detail API key berdasarkan ID
// @Tags API Keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} domain.APIKey
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/api-keys/{id} [get]
func (h *APIKeyHandler) GetAPIKey(c *fiber.Ctx) error {
	id := c.Params("id")
	traceID := c.Locals(middleware.TraceIDKey).(string)
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.TraceIDKey, traceID)

	if id == "" {
		h.logger.Error(ctx, "Missing API key ID parameter", telemetry.Fields{
			"handler": "GetAPIKey",
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "API key ID is required",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "Fetching API key", telemetry.Fields{
		"handler":  "GetAPIKey",
		"apiKeyID": id,
	})

	apiKey, err := h.apiKeyService.GetAPIKey(ctx, id)
	if err != nil {
		h.logger.Error(ctx, "Failed to fetch API key", telemetry.Fields{
			"handler":  "GetAPIKey",
			"apiKeyID": id,
			"error":    err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to fetch API key",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "API key fetched successfully", telemetry.Fields{
		"handler":     "GetAPIKey",
		"apiKeyID":    id,
		"serviceName": apiKey.ServiceName,
	})

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "API key fetched successfully",
		Data:    apiKey,
	})
}

// UpdateAPIKey menangani request untuk memperbarui API key
// @Summary Memperbarui API key
// @Description Memperbarui API key berdasarkan ID
// @Tags API Keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Param request body domain.APIKeyUpdateRequest true "Data API key yang diperbarui"
// @Success 200 {object} domain.APIKey
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/api-keys/{id} [put]
func (h *APIKeyHandler) UpdateAPIKey(c *fiber.Ctx) error {
	id := c.Params("id")
	traceID := c.Locals(middleware.TraceIDKey).(string)
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.TraceIDKey, traceID)

	if id == "" {
		h.logger.Error(ctx, "Missing API key ID parameter", telemetry.Fields{
			"handler": "UpdateAPIKey",
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "API key ID is required",
			Data:    nil,
		})
	}

	var req domain.APIKeyUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error(ctx, "Failed to parse request body", telemetry.Fields{
			"handler":  "UpdateAPIKey",
			"apiKeyID": id,
			"error":    err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Invalid request body",
			Data:    nil,
		})
	}

	// Validasi: minimal satu field tidak nil
	if req.Name == nil && req.Description == nil && req.ServiceName == nil && req.IsActive == nil && req.ExpiryDays == nil {
		h.logger.Error(ctx, "Request validation failed: no fields to update", telemetry.Fields{
			"handler":  "UpdateAPIKey",
			"apiKeyID": id,
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Tidak ada field yang diupdate",
			Data:    nil,
		})
	}

	// Validasi request dengan validator
	if err := validate.Struct(req); err != nil {
		h.logger.Error(ctx, "Request validation failed", telemetry.Fields{
			"handler":  "UpdateAPIKey",
			"apiKeyID": id,
			"error":    err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Invalid request data",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "Updating API key", telemetry.Fields{
		"handler":     "UpdateAPIKey",
		"apiKeyID":    id,
		"serviceName": req.ServiceName,
		"description": req.Description,
		"isActive":    req.IsActive,
	})

	apiKey, err := h.apiKeyService.UpdateAPIKey(ctx, id, &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to update API key", telemetry.Fields{
			"handler":  "UpdateAPIKey",
			"apiKeyID": id,
			"error":    err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to update API key",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "API key updated successfully", telemetry.Fields{
		"handler":     "UpdateAPIKey",
		"apiKeyID":    id,
		"serviceName": apiKey.ServiceName,
	})

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "API key updated successfully",
		Data:    apiKey,
	})
}

// DeleteAPIKey menangani request untuk menghapus API key
// @Summary Menghapus API key
// @Description Menghapus API key berdasarkan ID
// @Tags API Keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/api-keys/{id} [delete]
func (h *APIKeyHandler) DeleteAPIKey(c *fiber.Ctx) error {
	traceID := c.Locals(middleware.TraceIDKey).(string)
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.TraceIDKey, traceID)

	id := c.Params("id")
	if id == "" {
		h.logger.Warn(ctx, "Invalid API key ID", telemetry.Fields{
			"handler": "DeleteAPIKey",
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "ID tidak boleh kosong",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "Deleting API key", telemetry.Fields{
		"handler": "DeleteAPIKey",
		"id":      id,
	})

	err := h.apiKeyService.DeleteAPIKey(ctx, id)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete API key", telemetry.Fields{
			"handler": "DeleteAPIKey",
			"id":      id,
			"error":   err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to delete API key",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "API key deleted successfully", telemetry.Fields{
		"handler": "DeleteAPIKey",
		"id":      id,
	})

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "API key deleted successfully",
		Data:    nil,
	})
}

// GetAPIKeys menangani request untuk mendapatkan daftar API key
// @Summary Mendapatkan daftar API key
// @Description Mendapatkan daftar semua API key dengan pagination
// @Tags API Keys
// @Accept json
// @Produce json
// @Param page query int false "Nomor halaman" default(1)
// @Param limit query int false "Jumlah data per halaman" default(10)
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/api-keys [get]
func (h *APIKeyHandler) GetAPIKeys(c *fiber.Ctx) error {
	traceID := c.Locals(middleware.TraceIDKey).(string)
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.TraceIDKey, traceID)

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	h.logger.Info(ctx, "Fetching API keys", telemetry.Fields{
		"handler": "GetAPIKeys",
		"page":    page,
		"limit":   limit,
	})

	apiKeys, total, err := h.apiKeyService.GetAPIKeys(ctx, page, limit)
	if err != nil {
		h.logger.Error(ctx, "Failed to fetch API keys", telemetry.Fields{
			"handler": "GetAPIKeys",
			"error":   err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Gagal mengambil data API keys",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "API keys fetched successfully", telemetry.Fields{
		"handler":    "GetAPIKeys",
		"total_keys": len(apiKeys),
	})

	pagination := dto.PaginationInfo{
		Page:      page,
		Limit:     limit,
		Total:     total,
		TotalPage: int(math.Ceil(float64(total) / float64(limit))),
	}

	responseData := fiber.Map{
		"items":      apiKeys,
		"pagination": pagination,
	}

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "API keys fetched successfully",
		Data:    responseData,
	})
}

// ListAPIKeys menangani request untuk mendapatkan semua API key (deprecated)
// @Summary Mendapatkan daftar semua API key
// @Description Mendapatkan daftar semua API key tanpa pagination (deprecated, gunakan GetAPIKeys)
// @Tags API Keys
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/api-keys/list [get]
func (h *APIKeyHandler) ListAPIKeys(c *fiber.Ctx) error {
	traceID := c.Locals(middleware.TraceIDKey).(string)
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.TraceIDKey, traceID)

	h.logger.Warn(ctx, "Using deprecated ListAPIKeys method", telemetry.Fields{
		"handler": "ListAPIKeys",
		"info":    "This method is deprecated, use GetAPIKeys with pagination instead",
	})

	apiKeys, total, err := h.apiKeyService.GetAPIKeys(ctx, 1, 1000)
	if err != nil {
		h.logger.Error(ctx, "Failed to fetch API keys", telemetry.Fields{
			"handler": "ListAPIKeys",
			"error":   err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to fetch API keys",
			Data:    nil,
		})
	}

	h.logger.Info(ctx, "API keys fetched successfully (using deprecated method)", telemetry.Fields{
		"handler":    "ListAPIKeys",
		"total_keys": len(apiKeys),
		"total":      total,
	})

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "API keys fetched successfully",
		Data:    apiKeys,
	})
}
