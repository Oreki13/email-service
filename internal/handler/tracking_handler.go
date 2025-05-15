package handler

import (
	"context"
	"email-service/internal/domain"
	"email-service/internal/dto"
	"email-service/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

// EmailTrackingHandler menangani permintaan HTTP terkait tracking email
type EmailTrackingHandler struct {
	emailService domain.EmailService
	logger       telemetry.Logger
}

// NewEmailTrackingHandler membuat instance baru EmailTrackingHandler
func NewEmailTrackingHandler(emailService domain.EmailService, logger telemetry.Logger) *EmailTrackingHandler {
	return &EmailTrackingHandler{
		emailService: emailService,
		logger:       logger,
	}
}

// RegisterRoutes mendaftarkan routes untuk tracking email
func (h *EmailTrackingHandler) RegisterRoutes(router fiber.Router) {
	// Tracking pixel untuk tracking pembukaan email (GET untuk kompatibilitas dengan email client)
	router.Get("/track/open/:emailID", h.HandleTrackEmailOpen)

	// Tracking klik pada link (GET untuk kompatibilitas dengan email client)
	router.Get("/track/click/:emailID", h.HandleTrackEmailClick)

	// Mendapatkan data tracking untuk email tertentu
	router.Get("/tracking/:emailID", h.HandleGetEmailTrackingData)
}

// HandleTrackEmailOpen menangani request untuk tracking pembukaan email
func (h *EmailTrackingHandler) HandleTrackEmailOpen(c *fiber.Ctx) error {
	ctx := context.Background()
	emailID := c.Params("emailID")

	h.logger.Info(ctx, "Menerima request tracking pembukaan email", telemetry.Fields{
		"email_id":   emailID,
		"ip":         c.IP(),
		"user_agent": c.Get("User-Agent"),
	})

	// Track pembukaan email
	err := h.emailService.TrackEmailOpen(ctx, emailID, c.Get("User-Agent"), c.IP())
	if err != nil {
		h.logger.Error(ctx, "Gagal tracking pembukaan email", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})

		// Untuk tracking pixel, kita tetap mengembalikan gambar meskipun terjadi error
		// agar email client tidak menampilkan error
	}

	// Kirim gambar 1x1 pixel transparan
	c.Set("Content-Type", "image/gif")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Set("Pragma", "no-cache")

	// GIF 1x1 pixel transparan (data statis)
	// Ini adalah 1x1 piksel GIF transparan standar
	transparentPixelGIF := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0xFF, 0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x21, 0xF9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2C, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3B,
	}

	return c.Send(transparentPixelGIF)
}

// HandleTrackEmailClick menangani request untuk tracking klik pada link dalam email
func (h *EmailTrackingHandler) HandleTrackEmailClick(c *fiber.Ctx) error {
	ctx := context.Background()
	emailID := c.Params("emailID")
	redirectURL := c.Query("url", "")

	h.logger.Info(ctx, "Menerima request tracking klik email", telemetry.Fields{
		"email_id":   emailID,
		"url":        redirectURL,
		"ip":         c.IP(),
		"user_agent": c.Get("User-Agent"),
	})

	// Track klik email
	err := h.emailService.TrackEmailClick(ctx, emailID, redirectURL, c.Get("User-Agent"), c.IP())
	if err != nil {
		h.logger.Error(ctx, "Gagal tracking klik email", telemetry.Fields{
			"email_id": emailID,
			"url":      redirectURL,
			"error":    err.Error(),
		})

		// Untuk redirect, kita tetap melakukan redirect meskipun terjadi error
		// agar user experience tidak terganggu
	}

	// Cek apakah URL redirect valid
	if redirectURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			Message: "URL redirect tidak valid",
		})
	}

	// Redirect ke URL asli
	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

// HandleGetEmailTrackingData menangani request untuk mendapatkan data tracking email
func (h *EmailTrackingHandler) HandleGetEmailTrackingData(c *fiber.Ctx) error {
	ctx := c.Context()
	emailID := c.Params("emailID")

	h.logger.Info(ctx, "Menerima request data tracking email", telemetry.Fields{
		"email_id": emailID,
	})
	// Dapatkan data tracking
	trackingData, err := h.emailService.GetEmailTrackingData(ctx, emailID)
	if err != nil {
		h.logger.Error(ctx, "Gagal mendapatkan data tracking email", telemetry.Fields{
			"email_id": emailID,
			"error":    err.Error(),
		})

		// Format pesan error yang sesuai
		errorMsg := "Gagal mendapatkan data tracking email"
		if domainErr, ok := err.(*domain.AppError); ok {
			errorMsg = domainErr.Message
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{
			Status:  dto.StatusError,
			Message: errorMsg,
		})
	}

	// Konversi ke response DTO
	responseData := dto.EmailTrackingStatsResponse(emailID, trackingData)

	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{
		Status:  dto.StatusSuccess,
		Message: "Data tracking email berhasil didapatkan",
		Data:    responseData,
	})
}
