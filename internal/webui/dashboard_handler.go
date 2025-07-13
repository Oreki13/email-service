package webui

import (
	"email-service/internal/domain"
	"email-service/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

// DashboardPage menampilkan halaman dashboard utama
func (h *WebUIHandler) DashboardPage(c *fiber.Ctx) error {
	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	username := sess.Get("username")

	// Ambil statistik dari dashboard service
	dashboardStats, err := h.dashboardService.GetDashboardStats(c.Context())
	if err != nil {
		h.logger.Error(c.Context(), "Failed to get dashboard statistics", telemetry.Fields{
			"error":    err.Error(),
			"username": username,
			"ip":       c.IP(),
		})

		// Fallback ke data dummy jika gagal mengambil data dari database
		dashboardStats = &domain.DashboardStats{
			TotalTemplates:  0,
			ActiveTemplates: 0,
			EmailsSent:      0,
			EmailsQueued:    0,
		}
	}

	// Convert domain stats ke format untuk template
	stats := fiber.Map{
		"TotalTemplates":    dashboardStats.TotalTemplates,
		"ActiveTemplates":   dashboardStats.ActiveTemplates,
		"InactiveTemplates": dashboardStats.InactiveTemplates,
		"EmailsSent":        dashboardStats.EmailsSent,
		"EmailsQueued":      dashboardStats.EmailsQueued,
		"EmailsFailed":      dashboardStats.EmailsFailed,
		"EmailsPending":     dashboardStats.EmailsPending,
		"TotalEmails":       dashboardStats.TotalEmails,
		"EmailsToday":       dashboardStats.EmailsToday,
		"EmailsThisWeek":    dashboardStats.EmailsThisWeek,
		"EmailsThisMonth":   dashboardStats.EmailsThisMonth,
	}

	h.logger.Info(c.Context(), "Dashboard accessed", telemetry.Fields{
		"username":         username,
		"ip":               c.IP(),
		"total_templates":  dashboardStats.TotalTemplates,
		"active_templates": dashboardStats.ActiveTemplates,
		"emails_sent":      dashboardStats.EmailsSent,
		"emails_queued":    dashboardStats.EmailsQueued,
	})

	return c.Render("dashboard", fiber.Map{
		"Title":    "Dashboard - Email Service Management",
		"Username": username,
		"Stats":    stats,
	})
}
