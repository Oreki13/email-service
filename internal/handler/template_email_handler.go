package handler

import (
	"email-service/internal/domain"
	"email-service/internal/dto"
	"email-service/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

// TemplateEmailHandler menangani request terkait email dengan template spesifik
type TemplateEmailHandler struct {
	templatedEmailService domain.TemplatedEmailService
	logger                telemetry.Logger
}

// NewTemplateEmailHandler membuat instance baru template email handler
func NewTemplateEmailHandler(
	templatedEmailService domain.TemplatedEmailService,
	logger telemetry.Logger,
) *TemplateEmailHandler {
	return &TemplateEmailHandler{
		templatedEmailService: templatedEmailService,
		logger:                logger,
	}
}

// RegisterTemplateRoutes mendaftarkan route untuk template email
func (h *TemplateEmailHandler) RegisterRoutes(router fiber.Router) {
	// Grup untuk template email
	templateGroup := router.Group("/templates")

	// Endpoint untuk mendapatkan informasi template
	templateGroup.Get("/", h.GetTemplateInfo)

	// Endpoint untuk template spesifik
	templateGroup.Post("/welcome", h.SendWelcomeEmail)
	templateGroup.Post("/reset-password", h.SendPasswordResetEmail)
	templateGroup.Post("/notification", h.SendNotificationEmail)
}

// GetTemplateInfo menangani request untuk mendapatkan informasi template
func (h *TemplateEmailHandler) GetTemplateInfo(c *fiber.Ctx) error {
	ctx := c.Context()
	h.logger.Info(ctx, "Menerima permintaan informasi template", nil)

	templateInfos, err := h.templatedEmailService.GetTemplateInfo(ctx)
	traceID := c.GetRespHeader("X-Request-Id", "-")
	if err != nil {
		h.logger.Error(ctx, "Gagal mendapatkan informasi template", telemetry.Fields{
			"error": err.Error(),
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Gagal mendapatkan informasi template",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "Berhasil mendapatkan informasi template",
		Data:    templateInfos,
	})
}

// SendWelcomeEmail menangani request untuk mengirim email selamat datang
func (h *TemplateEmailHandler) SendWelcomeEmail(c *fiber.Ctx) error {
	ctx := c.Context()
	h.logger.Info(ctx, "Menerima permintaan pengiriman email selamat datang", nil)

	var request domain.WelcomeEmailRequest
	traceID := c.GetRespHeader("X-Request-Id", "-")
	if err := c.BodyParser(&request); err != nil {
		h.logger.Error(ctx, "Format request tidak valid", telemetry.Fields{
			"error": err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Format request tidak valid",
			Data:    nil,
		})
	}

	emailID, err := h.templatedEmailService.SendWelcomeEmail(ctx, &request)
	if err != nil {
		h.logger.Error(ctx, "Gagal mengirim email selamat datang", telemetry.Fields{
			"error": err.Error(),
			"to":    request.To,
		})
		if appErr, ok := err.(*domain.AppError); ok && appErr.Type == domain.ErrorTypeValidation {
			return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
				Status:  dto.StatusError,
				TraceID: traceID,
				Message: appErr.Message,
				Data:    appErr.Details,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Gagal mengirim email selamat datang",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "Email selamat datang sedang diproses",
		Data:    fiber.Map{"email_id": emailID},
	})
}

// SendPasswordResetEmail menangani request untuk mengirim email reset password
func (h *TemplateEmailHandler) SendPasswordResetEmail(c *fiber.Ctx) error {
	ctx := c.Context()
	h.logger.Info(ctx, "Menerima permintaan pengiriman email reset password", nil)

	var request domain.PasswordResetEmailRequest
	traceID := c.GetRespHeader("X-Request-Id", "-")
	if err := c.BodyParser(&request); err != nil {
		h.logger.Error(ctx, "Format request tidak valid", telemetry.Fields{
			"error": err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Format request tidak valid",
			Data:    nil,
		})
	}

	emailID, err := h.templatedEmailService.SendPasswordResetEmail(ctx, &request)
	if err != nil {
		h.logger.Error(ctx, "Gagal mengirim email reset password", telemetry.Fields{
			"error": err.Error(),
			"to":    request.To,
		})
		if appErr, ok := err.(*domain.AppError); ok && appErr.Type == domain.ErrorTypeValidation {
			return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
				Status:  dto.StatusError,
				TraceID: traceID,
				Message: appErr.Message,
				Data:    appErr.Details,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Gagal mengirim email reset password",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "Email reset password sedang diproses",
		Data:    fiber.Map{"email_id": emailID},
	})
}

// SendNotificationEmail menangani request untuk mengirim email notifikasi
func (h *TemplateEmailHandler) SendNotificationEmail(c *fiber.Ctx) error {
	ctx := c.Context()
	h.logger.Info(ctx, "Menerima permintaan pengiriman email notifikasi", nil)

	var request domain.NotificationEmailRequest
	traceID := c.GetRespHeader("X-Request-Id", "-")
	if err := c.BodyParser(&request); err != nil {
		h.logger.Error(ctx, "Format request tidak valid", telemetry.Fields{
			"error": err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Format request tidak valid",
			Data:    nil,
		})
	}

	emailID, err := h.templatedEmailService.SendNotificationEmail(ctx, &request)
	if err != nil {
		h.logger.Error(ctx, "Gagal mengirim email notifikasi", telemetry.Fields{
			"error": err.Error(),
			"to":    request.To,
		})
		if appErr, ok := err.(*domain.AppError); ok && appErr.Type == domain.ErrorTypeValidation {
			return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
				Status:  dto.StatusError,
				TraceID: traceID,
				Message: appErr.Message,
				Data:    appErr.Details,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			TraceID: traceID,
			Message: "Gagal mengirim email notifikasi",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		TraceID: traceID,
		Message: "Email notifikasi sedang diproses",
		Data:    fiber.Map{"email_id": emailID},
	})
}
