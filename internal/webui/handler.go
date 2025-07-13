package webui

import (
	"context"
	"email-service/internal/config"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// WebUIHandler handles web UI requests untuk management template email
type WebUIHandler struct {
	config           *config.Config
	templateService  domain.TemplatedEmailService
	templateRepo     domain.TemplateRepository
	dashboardService domain.DashboardService
	logger           telemetry.Logger
	sessionStore     *session.Store
}

// NewWebUIHandler membuat instance baru WebUIHandler
func NewWebUIHandler(cfg *config.Config, templateService domain.TemplatedEmailService, templateRepo domain.TemplateRepository, dashboardService domain.DashboardService, logger telemetry.Logger) *WebUIHandler {
	// Setup session store untuk authentication
	store := session.New(session.Config{
		KeyLookup:  "cookie:session_id",
		CookiePath: "/",
		Expiration: time.Duration(cfg.WebUI.SessionDuration) * time.Minute,
		Storage:    nil, // default in-memory storage
	})

	return &WebUIHandler{
		config:           cfg,
		templateService:  templateService,
		templateRepo:     templateRepo,
		dashboardService: dashboardService,
		logger:           logger,
		sessionStore:     store,
	}
}

// RegisterRoutes mendaftarkan semua route untuk Web UI
func (h *WebUIHandler) RegisterRoutes(app *fiber.App) {
	// Skip jika WebUI tidak diaktifkan
	if !h.config.WebUI.Enabled {
		h.logger.Info(context.Background(), "WebUI is disabled, skipping route registration", nil)
		return
	}

	// Setup static files
	app.Static("/static", "./web/static")

	// Public routes (tidak perlu authentication)
	app.Get("/login", h.LoginPage)
	app.Post("/login", h.HandleLogin)
	app.Post("/logout", h.HandleLogout)

	// Protected routes (perlu authentication)
	protected := app.Group("/", h.AuthMiddleware)
	protected.Get("/", h.DashboardPage)
	protected.Get("/dashboard", h.DashboardPage)

	// Template management routes
	templates := protected.Group("/templates")
	templates.Get("/", h.TemplateListPage)
	templates.Get("/create", h.TemplateCreatePage)
	templates.Post("/create", h.HandleTemplateCreate)
	templates.Get("/:id", h.TemplateDetailPage)
	templates.Get("/:id/edit", h.TemplateEditPage)
	templates.Post("/:id/edit", h.HandleTemplateEdit)
	templates.Post("/:id/delete", h.HandleTemplateDelete)
	templates.Delete("/:id", h.HandleTemplateDelete)
	templates.Post("/:id/preview", h.HandleTemplatePreview)
	templates.Post("/:id/test", h.HandleTemplateSendTest)

	h.logger.Info(context.Background(), "WebUI routes registered successfully", telemetry.Fields{
		"enabled": h.config.WebUI.Enabled,
	})
}

// AuthMiddleware middleware untuk memverifikasi authentication
func (h *WebUIHandler) AuthMiddleware(c *fiber.Ctx) error {
	sess, err := h.sessionStore.Get(c)
	if err != nil {
		h.logger.Error(c.Context(), "Failed to get session", telemetry.Fields{
			"error": err.Error(),
		})
		return c.Redirect("/login")
	}

	// Check if user is authenticated
	authenticated := sess.Get("authenticated")
	if authenticated != true {
		return c.Redirect("/login")
	}

	return c.Next()
}

// Helper method untuk mengecek apakah user sudah authenticated
func (h *WebUIHandler) isAuthenticated(c *fiber.Ctx) bool {
	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return false
	}

	authenticated := sess.Get("authenticated")
	return authenticated == true
}
