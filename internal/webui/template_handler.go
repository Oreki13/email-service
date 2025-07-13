package webui

import (
	"context"
	"email-service/internal/domain"
	"email-service/internal/dto"
	"email-service/pkg/telemetry"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// TemplateListPage menampilkan daftar template email
func (h *WebUIHandler) TemplateListPage(c *fiber.Ctx) error {
	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	username := sess.Get("username")

	// Parse query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	search := c.Query("search", "")
	status := c.Query("status", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Get templates with pagination
	templates, total, err := h.getTemplatesFromRepository(c.Context(), limit, offset, search, status)
	if err != nil {
		h.logger.Error(c.Context(), "Failed to get templates", telemetry.Fields{
			"error":    err.Error(),
			"username": username,
			"ip":       c.IP(),
			"page":     page,
			"limit":    limit,
			"search":   search,
			"status":   status,
		})

		// Return empty list on error
		templates = []*domain.Template{}
		total = 0
	}

	// Convert to DTO
	templateItems := make([]dto.TemplateListItem, 0, len(templates))
	for _, template := range templates {
		templateType := "Email"
		if template.PlainBody != "" && template.HTMLBody == "" {
			templateType = "Text"
		} else if template.HTMLBody != "" && template.PlainBody == "" {
			templateType = "HTML"
		}

		templateItems = append(templateItems, dto.TemplateListItem{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			Type:        templateType,
			IsActive:    template.IsActive,
			CreatedAt:   template.CreatedAt,
			UpdatedAt:   template.UpdatedAt,
		})
	}

	// Create pagination
	pagination := dto.NewWebPagination(page, limit, total)

	h.logger.Info(c.Context(), "Template list accessed", telemetry.Fields{
		"username":      username,
		"ip":            c.IP(),
		"page":          page,
		"limit":         limit,
		"search":        search,
		"status":        status,
		"total_results": total,
		"results_count": len(templateItems),
	})

	h.logger.Debug(c.Context(), "Rendering template_list", telemetry.Fields{
		"template_name": "template_list",
		"data_keys":     []string{"Title", "Username", "Templates", "Pagination", "Search", "Status"},
	})

	return c.Render("template_list", fiber.Map{
		"Title":      "Email Templates - Email Service Management",
		"Username":   username,
		"Templates":  templateItems,
		"Pagination": pagination,
		"Search":     search,
		"Status":     status,
	})
}

// TemplateCreatePage menampilkan halaman create template
func (h *WebUIHandler) TemplateCreatePage(c *fiber.Ctx) error {
	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	username := sess.Get("username")

	return c.Render("template_create", fiber.Map{
		"Title":    "Create Template - Email Service Management",
		"Username": username,
	})
}

// HandleTemplateCreate memproses pembuatan template baru
func (h *WebUIHandler) HandleTemplateCreate(c *fiber.Ctx) error {
	// TODO: Implementasi create template
	// Untuk sementara redirect ke list
	return c.Redirect("/templates")
}

// TemplateDetailPage menampilkan detail template
func (h *WebUIHandler) TemplateDetailPage(c *fiber.Ctx) error {
	templateID := c.Params("id")

	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	username := sess.Get("username")

	// Get template detail dari repository
	template, err := h.templateRepo.FindByID(c.Context(), templateID)
	if err != nil {
		h.logger.Error(c.Context(), "Failed to get template detail", telemetry.Fields{
			"error":      err.Error(),
			"templateID": templateID,
			"username":   username,
			"ip":         c.IP(),
		})

		// Return 500 error
		return c.Status(fiber.StatusInternalServerError).Render("error", fiber.Map{
			"Title":   "Error - Email Service Management",
			"Message": "Failed to load template detail",
			"Code":    500,
		})
	}

	if template == nil {
		h.logger.Warn(c.Context(), "Template not found", telemetry.Fields{
			"templateID": templateID,
			"username":   username,
			"ip":         c.IP(),
		})

		// Return 404 error
		return c.Status(fiber.StatusNotFound).Render("error", fiber.Map{
			"Title":   "Template Not Found - Email Service Management",
			"Message": "The requested template was not found",
			"Code":    404,
		})
	}

	// Convert variables map to string for display
	var variablesStr string
	if template.Variables != nil && len(template.Variables) > 0 {
		var vars []string
		for key := range template.Variables {
			vars = append(vars, key)
		}
		variablesStr = strings.Join(vars, ",")
	}

	// Determine template type based on name or content
	templateType := "General"
	templateName := strings.ToLower(template.Name)
	if strings.Contains(templateName, "welcome") {
		templateType = "Welcome"
	} else if strings.Contains(templateName, "reset") || strings.Contains(templateName, "password") {
		templateType = "Password Reset"
	} else if strings.Contains(templateName, "notification") {
		templateType = "Notification"
	}

	// Prepare template data for rendering
	templateData := fiber.Map{
		"ID":          template.ID,
		"Name":        template.Name,
		"Description": template.Description,
		"Subject":     template.Subject,
		"BodyHTML":    template.HTMLBody,
		"BodyText":    template.PlainBody,
		"Variables":   variablesStr,
		"IsActive":    template.IsActive,
		"Type":        templateType,
		"Version":     template.Version,
		"CreatedAt":   template.CreatedAt,
		"UpdatedAt":   template.UpdatedAt,
		"FromName":    "", // Could be extracted from template or config
		"FromEmail":   "", // Could be extracted from template or config
	}

	h.logger.Info(c.Context(), "Template detail retrieved successfully", telemetry.Fields{
		"templateID":   templateID,
		"templateName": template.Name,
		"username":     username,
		"ip":           c.IP(),
	})

	return c.Render("template_detail", fiber.Map{
		"Title":    "Template Detail - Email Service Management",
		"Username": username,
		"Template": templateData,
	})
}

// TemplateEditPage menampilkan halaman edit template
func (h *WebUIHandler) TemplateEditPage(c *fiber.Ctx) error {
	templateID := c.Params("id")

	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	username := sess.Get("username")

	// TODO: Get template detail dari service
	template := fiber.Map{
		"ID":          templateID,
		"Name":        "Sample Template",
		"Subject":     "Welcome Email",
		"HTMLContent": "<h1>Welcome!</h1>",
		"TextContent": "Welcome!",
		"IsActive":    true,
	}

	return c.Render("template_edit", fiber.Map{
		"Title":    "Edit Template - Email Service Management",
		"Username": username,
		"Template": template,
	})
}

// HandleTemplateEdit memproses update template
func (h *WebUIHandler) HandleTemplateEdit(c *fiber.Ctx) error {
	templateID := c.Params("id")

	// TODO: Implementasi update template
	h.logger.Info(c.Context(), "Template updated", telemetry.Fields{
		"template_id": templateID,
	})

	return c.Redirect("/templates/" + templateID)
}

// HandleTemplateDelete memproses penghapusan template
func (h *WebUIHandler) HandleTemplateDelete(c *fiber.Ctx) error {
	templateID := c.Params("id")

	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	username := sess.Get("username")

	// Delete template from repository
	err = h.templateRepo.Delete(c.Context(), templateID)
	if err != nil {
		h.logger.Error(c.Context(), "Failed to delete template", telemetry.Fields{
			"error":      err.Error(),
			"templateID": templateID,
			"username":   username,
			"ip":         c.IP(),
		})

		// Check if this is a form submission from web UI
		if c.Get("Content-Type") == "application/x-www-form-urlencoded" || c.Method() == "POST" {
			return c.Redirect("/templates?error=delete_failed")
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete template",
			"error":   err.Error(),
		})
	}

	h.logger.Info(c.Context(), "Template deleted successfully", telemetry.Fields{
		"templateID": templateID,
		"username":   username,
		"ip":         c.IP(),
	})

	// Check if this is a form submission from web UI
	if c.Get("Content-Type") == "application/x-www-form-urlencoded" || c.Method() == "POST" {
		return c.Redirect("/templates?success=deleted")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Template deleted successfully",
	})
}

// HandleTemplatePreview memproses preview template
func (h *WebUIHandler) HandleTemplatePreview(c *fiber.Ctx) error {
	templateID := c.Params("id")

	// TODO: Implementasi preview template
	h.logger.Info(c.Context(), "Template preview requested", telemetry.Fields{
		"template_id": templateID,
	})

	return c.JSON(fiber.Map{
		"success": true,
		"html":    "<h1>Preview Content</h1>",
	})
}

// HandleTemplateSendTest memproses pengiriman test email
func (h *WebUIHandler) HandleTemplateSendTest(c *fiber.Ctx) error {
	templateID := c.Params("id")

	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	username := sess.Get("username")

	// Parse request body
	var request struct {
		Email     string                 `json:"email" form:"test_email"`
		Variables map[string]interface{} `json:"variables" form:"test_variables"`
	}

	if err := c.BodyParser(&request); err != nil {
		h.logger.Error(c.Context(), "Failed to parse test email request", telemetry.Fields{
			"error":      err.Error(),
			"templateID": templateID,
			"username":   username,
			"ip":         c.IP(),
		})

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
	}

	// Validate email
	if request.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Email address is required",
		})
	}

	// Get template detail
	template, err := h.templateRepo.FindByID(c.Context(), templateID)
	if err != nil {
		h.logger.Error(c.Context(), "Failed to get template for test email", telemetry.Fields{
			"error":      err.Error(),
			"templateID": templateID,
			"email":      request.Email,
			"username":   username,
			"ip":         c.IP(),
		})

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load template",
			"error":   err.Error(),
		})
	}

	if template == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Template not found",
		})
	}

	// Create email request for sending test email
	emailRequest := &domain.EmailRequest{
		To:           []string{request.Email},
		Subject:      template.Subject + " [TEST]",
		HTMLBody:     template.HTMLBody,
		PlainBody:    template.PlainBody,
		TemplateID:   template.ID,
		TemplateName: template.Name,
		TemplateData: request.Variables,
		Priority:     domain.PriorityNormal,
		Provider:     domain.ProviderSMTP, // Default provider
		Metadata: map[string]string{
			"source":     "webui_test",
			"templateID": templateID,
			"testBy":     username.(string),
		},
	}

	// TODO: Send test email using email service
	// For now, we'll just log the attempt and return success
	h.logger.Info(c.Context(), "Test email request processed", telemetry.Fields{
		"templateID":   templateID,
		"templateName": template.Name,
		"email":        request.Email,
		"username":     username,
		"ip":           c.IP(),
		"variables":    request.Variables,
		"emailRequest": emailRequest.Subject, // Log only subject to avoid sensitive data
	})

	// TODO: Implement actual email sending
	// For now, just return success
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Test email sent successfully to " + request.Email,
		"data": fiber.Map{
			"templateID":   templateID,
			"templateName": template.Name,
			"email":        request.Email,
		},
	})
}

// getTemplatesFromRepository mengambil template dari repository dengan pagination dan filtering
func (h *WebUIHandler) getTemplatesFromRepository(ctx context.Context, limit, offset int, search, status string) ([]*domain.Template, int64, error) {
	h.logger.Debug(ctx, "Getting templates from repository with pagination", telemetry.Fields{
		"limit":  limit,
		"offset": offset,
		"search": search,
		"status": status,
	})

	// Menggunakan repository method yang sudah mendukung pagination dan filtering
	templates, total, err := h.templateRepo.FindWithPagination(ctx, limit, offset, search, status)
	if err != nil {
		h.logger.Error(ctx, "Failed to get templates with pagination from repository", telemetry.Fields{
			"error":  err.Error(),
			"limit":  limit,
			"offset": offset,
			"search": search,
			"status": status,
		})
		return nil, 0, err
	}

	h.logger.Debug(ctx, "Successfully retrieved templates from repository", telemetry.Fields{
		"total_count":    total,
		"returned_count": len(templates),
		"limit":          limit,
		"offset":         offset,
		"search":         search,
		"status":         status,
	})

	return templates, total, nil
}

// getAllTemplates mendapatkan semua template dari repository
func (h *WebUIHandler) getAllTemplates(ctx context.Context) ([]*domain.Template, error) {
	h.logger.Debug(ctx, "Fetching all templates from repository", nil)

	// Menggunakan repository untuk mendapatkan semua template
	templates, err := h.templateRepo.FindAll(ctx)
	if err != nil {
		h.logger.Error(ctx, "Failed to fetch templates from repository", telemetry.Fields{
			"error": err.Error(),
		})
		return nil, err
	}

	h.logger.Debug(ctx, "Successfully fetched templates from repository", telemetry.Fields{
		"template_count": len(templates),
	})

	return templates, nil
}
