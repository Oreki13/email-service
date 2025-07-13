package webui

import (
	"email-service/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

// LoginPage menampilkan halaman login
func (h *WebUIHandler) LoginPage(c *fiber.Ctx) error {
	// Jika sudah authenticated, redirect ke dashboard
	if h.isAuthenticated(c) {
		return c.Redirect("/dashboard")
	}

	// Ambil error message jika ada
	errorMsg := c.Query("error", "")

	return c.Render("login", fiber.Map{
		"Title": "Login - Email Service Management",
		"Error": errorMsg,
	})
}

// HandleLogin memproses login request
func (h *WebUIHandler) HandleLogin(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Validasi credentials
	if username != h.config.WebUI.Username || password != h.config.WebUI.Password {
		h.logger.Warn(c.Context(), "Failed login attempt", telemetry.Fields{
			"username": username,
			"ip":       c.IP(),
		})
		return c.Redirect("/login?error=Invalid username or password")
	}

	// Create session
	sess, err := h.sessionStore.Get(c)
	if err != nil {
		h.logger.Error(c.Context(), "Failed to get session", telemetry.Fields{
			"error": err.Error(),
		})
		return c.Redirect("/login?error=Session error")
	}

	// Set session data
	sess.Set("authenticated", true)
	sess.Set("username", username)
	err = sess.Save()
	if err != nil {
		h.logger.Error(c.Context(), "Failed to save session", telemetry.Fields{
			"error": err.Error(),
		})
		return c.Redirect("/login?error=Session error")
	}

	h.logger.Info(c.Context(), "User logged in successfully", telemetry.Fields{
		"username": username,
		"ip":       c.IP(),
	})

	return c.Redirect("/dashboard")
}

// HandleLogout memproses logout request
func (h *WebUIHandler) HandleLogout(c *fiber.Ctx) error {
	sess, err := h.sessionStore.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	// Log user activity
	username := sess.Get("username")
	h.logger.Info(c.Context(), "User logged out", telemetry.Fields{
		"username": username,
		"ip":       c.IP(),
	})

	// Destroy session
	err = sess.Destroy()
	if err != nil {
		h.logger.Error(c.Context(), "Failed to destroy session", telemetry.Fields{
			"error": err.Error(),
		})
	}

	return c.Redirect("/login")
}
