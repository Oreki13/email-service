package handler

import (
	"email-service/internal/config"
	"email-service/internal/domain"
	"email-service/internal/dto"
	"email-service/internal/middleware"
	"email-service/internal/service"
	"email-service/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

// EmailHandler menangani request terkait email
type EmailHandler struct {
	emailService domain.EmailService
}

// NewEmailHandler membuat instance baru dari EmailHandler
func NewEmailHandler(emailService domain.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
	}
}

// SendEmail menangani request untuk mengirim email
func (h *EmailHandler) SendEmail(c *fiber.Ctx) error {
	traceID := c.Locals(middleware.TraceIDKey).(string)

	// Parse request body
	var req domain.EmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to parse request body",
			Data:    nil,
		})
	}

	// Kirim email
	emailID, err := h.emailService.SendEmail(c.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			switch appErr.Type {
			case domain.ErrorTypeValidation:
				return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
					Status:  dto.StatusError,
					TraceID: traceID,
					Message: appErr.Message,
					Data:    appErr.Details,
				})
			case domain.ErrorTypeInternal:
				return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
					Status:  dto.StatusError,
					TraceID: traceID,
					Message: "Internal server error",
					Data:    nil,
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
					Status:  dto.StatusError,
					TraceID: traceID,
					Message: appErr.Message,
					Data:    nil,
				})
			}
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to send email",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "Email queued for delivery",
		Data:    fiber.Map{"email_id": emailID},
	})
}

// GetEmailStatus menangani request untuk mendapatkan status email
func (h *EmailHandler) GetEmailStatus(c *fiber.Ctx) error {
	traceID := c.Locals(middleware.TraceIDKey).(string)
	emailID := c.Params("id")

	if emailID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Email ID is required",
			Data:    nil,
		})
	}

	status, err := h.emailService.GetStatus(c.Context(), emailID)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			switch appErr.Type {
			case domain.ErrorTypeNotFound:
				return c.Status(fiber.StatusNotFound).JSON(dto.APIResponse{
					Status:  dto.StatusError,
					TraceID: traceID,
					Message: appErr.Message,
					Data:    nil,
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
					Status:  dto.StatusError,
					TraceID: traceID,
					Message: appErr.Message,
					Data:    nil,
				})
			}
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Failed to get email status",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "Success",
		Data:    status,
	})
}

// RegisterRoutes mendaftarkan semua routes untuk email service
func RegisterRoutes(router fiber.Router, cfg *config.Config, emailService domain.EmailService, apiKeyRepo domain.APIKeyRepository, logger telemetry.Logger) {
	// Buat handler untuk email dengan service yang sudah diinjeksi
	emailHandler := NewEmailHandler(emailService)

	// Daftarkan routes dengan prefix /emails
	emailRouter := router.Group("/emails")

	// Tambahkan middleware untuk request ID dan logging
	emailRouter.Use(middleware.RequestID())

	// Tambahkan middleware autentikasi API key
	emailRouter.Use(middleware.AuthMiddleware(apiKeyRepo, cfg.Auth.APIKeyHeaderName, logger))

	// Route untuk mengirim email
	emailRouter.Post("/manual", emailHandler.SendEmail)

	// Route untuk mendapatkan status email
	emailRouter.Get("/:id/status", emailHandler.GetEmailStatus)

	// Inisialisasi API key service
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, logger)

	// Daftarkan rute untuk manajemen API key
	RegisterAPIKeyRoutes(router, apiKeyService, logger, cfg)
}

// ErrorHandler menangani error dari Fiber
func ErrorHandler(ctx *fiber.Ctx, err error) error {
	traceID := ctx.Locals(middleware.TraceIDKey)
	if traceID == nil {
		traceID = "unknown"
	}

	code := fiber.StatusInternalServerError

	// Cek error dari Fiber
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Log error (seharusnya menggunakan CloudWatch)
	// TODO: Log error ke CloudWatch

	return ctx.Status(code).JSON(dto.APIResponse{
		Status:  dto.StatusError,
		TraceID: traceID.(string),
		Message: err.Error(),
		Data:    nil,
	})
}
