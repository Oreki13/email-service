package service

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"fmt"
	"regexp"
	"strings"
)

// EmailEnhancer untuk menambahkan fitur tracking pada email
type EmailEnhancer struct {
	baseURL string
	logger  telemetry.Logger
}

// NewEmailEnhancer membuat instance baru EmailEnhancer
func NewEmailEnhancer(baseURL string, logger telemetry.Logger) *EmailEnhancer {
	// Pastikan baseURL tidak diakhiri dengan slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &EmailEnhancer{
		baseURL: baseURL,
		logger:  logger,
	}
}

// AddTrackingPixel menambahkan tracking pixel ke dalam HTML email
func (e *EmailEnhancer) AddTrackingPixel(ctx context.Context, emailID string, htmlBody string) string {
	if htmlBody == "" {
		return htmlBody
	}

	// Tracking pixel URL
	trackingPixelURL := fmt.Sprintf("%s/api/v1/track/open/%s", e.baseURL, emailID)

	// Tracking pixel HTML
	trackingPixel := fmt.Sprintf(`<img src="%s" alt="" width="1" height="1" style="display:none;width:1px;height:1px;"/>`, trackingPixelURL)

	// Cek apakah HTML memiliki tag penutup </body>
	if strings.Contains(htmlBody, "</body>") {
		// Tambahkan tracking pixel sebelum tag penutup </body>
		return strings.Replace(htmlBody, "</body>", trackingPixel+"</body>", 1)
	} else if strings.Contains(htmlBody, "</html>") {
		// Jika tidak ada </body>, coba tambahkan sebelum </html>
		return strings.Replace(htmlBody, "</html>", trackingPixel+"</html>", 1)
	} else {
		// Jika tidak ada keduanya, tambahkan di akhir HTML
		return htmlBody + trackingPixel
	}
}

// RewriteLinks mengubah semua link dalam HTML menjadi tracked links
func (e *EmailEnhancer) RewriteLinks(ctx context.Context, emailID string, htmlBody string) string {
	if htmlBody == "" {
		return htmlBody
	}

	// Regex untuk mencocokkan tag <a href="...">
	re := regexp.MustCompile(`<a\s+(?:[^>]*?\s+)?href=(['"])(.*?)\\1`)

	e.logger.Debug(ctx, "Rewriting links in email", telemetry.Fields{
		"email_id": emailID,
	})

	// Fungsi pengganti untuk mengubah setiap link yang ditemukan
	rewrittenHTML := re.ReplaceAllStringFunc(htmlBody, func(match string) string {
		// Ekstrak URL dari match
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		originalURL := submatches[2]
		urlQuote := submatches[1] // Tanda kutip yang digunakan (''' atau '"')

		// Skip jika URL adalah anchor atau mailto
		if strings.HasPrefix(originalURL, "#") ||
			strings.HasPrefix(originalURL, "mailto:") ||
			strings.HasPrefix(originalURL, "tel:") {
			return match
		}

		// Buat tracking URL
		trackingURL := fmt.Sprintf("%s/api/v1/track/click/%s?url=%s",
			e.baseURL,
			emailID,
			originalURL)

		// Ganti URL dalam tag a href
		return strings.Replace(match, fmt.Sprintf("href=%s%s%s", urlQuote, originalURL, urlQuote),
			fmt.Sprintf("href=%s%s%s", urlQuote, trackingURL, urlQuote), 1)
	})

	return rewrittenHTML
}

// EnhanceEmail menambahkan tracking pixel dan rewrite links pada email
func (e *EmailEnhancer) EnhanceEmail(ctx context.Context, email *domain.Email) {
	if email.HTMLBody == "" {
		return
	}

	e.logger.Debug(ctx, "Enhancing email with tracking features", telemetry.Fields{
		"email_id": email.ID,
	})

	// Tambahkan tracking pixel
	enhancedHTML := e.AddTrackingPixel(ctx, email.ID, email.HTMLBody)

	// Rewrite links
	enhancedHTML = e.RewriteLinks(ctx, email.ID, enhancedHTML)

	// Update HTML body email
	email.HTMLBody = enhancedHTML
}
