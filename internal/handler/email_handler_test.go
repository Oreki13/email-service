package handler

import (
	"bytes"
	"context"
	"email-service/internal/domain"
	"email-service/internal/middleware"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmailService adalah mock untuk domain.EmailService
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendEmail(ctx context.Context, request *domain.EmailRequest) (string, error) {
	args := m.Called(ctx, request)
	return args.String(0), args.Error(1)
}

func (m *MockEmailService) GetStatus(ctx context.Context, id string) (*domain.EmailStatus, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EmailStatus), args.Error(1)
}

func (m *MockEmailService) ProcessPendingEmails(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailService) TrackEmailOpen(ctx context.Context, emailID string, userAgent, ipAddress string) error {
	args := m.Called(ctx, emailID, userAgent, ipAddress)
	return args.Error(0)
}

func (m *MockEmailService) TrackEmailClick(ctx context.Context, emailID string, url, userAgent, ipAddress string) error {
	args := m.Called(ctx, emailID, url, userAgent, ipAddress)
	return args.Error(0)
}

func (m *MockEmailService) GetEmailTrackingData(ctx context.Context, emailID string) ([]*domain.EmailTracking, error) {
	args := m.Called(ctx, emailID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.EmailTracking), args.Error(1)
}

// setupApp membuat instance baru dari aplikasi Fiber dengan middleware yang diperlukan
func setupApp() (*fiber.App, *MockEmailService) {
	app := fiber.New()
	mockEmailService := new(MockEmailService)

	// Middleware untuk menambahkan trace ID ke request
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.TraceIDKey, "test-trace-id")
		return c.Next()
	})

	return app, mockEmailService
}

// TestSendEmail menguji fungsi SendEmail
func TestSendEmail(t *testing.T) {
	// Persiapkan beberapa skenario uji
	tests := []struct {
		name           string
		requestBody    domain.EmailRequest
		mockReturn     string
		mockError      error
		expectedStatus int
		expectedAPIRes map[string]interface{}
	}{
		{
			name: "Success",
			requestBody: domain.EmailRequest{
				To:        []string{"test@example.com"},
				Subject:   "Test Email",
				PlainBody: "This is a test email",
			},
			mockReturn:     "test-email-id",
			mockError:      nil,
			expectedStatus: fiber.StatusAccepted,
			expectedAPIRes: map[string]interface{}{
				"status":   "SUCCESS",
				"trace_id": "test-trace-id",
				"message":  "Email queued for delivery",
				"data":     map[string]interface{}{"email_id": "test-email-id"},
			},
		},
		{
			name: "ValidationError",
			requestBody: domain.EmailRequest{
				To:      []string{"test@example.com"},
				Subject: "Test Email",
				// Tanpa body
			},
			mockReturn:     "",
			mockError:      domain.ValidationError("Either plain body, HTML body, or template is required", nil),
			expectedStatus: fiber.StatusBadRequest,
			expectedAPIRes: map[string]interface{}{
				"status":   "ERROR",
				"trace_id": "test-trace-id",
				"message":  "Either plain body, HTML body, or template is required",
				"data":     nil,
			},
		},
		{
			name: "InternalError",
			requestBody: domain.EmailRequest{
				To:        []string{"test@example.com"},
				Subject:   "Test Email",
				PlainBody: "This is a test email",
			},
			mockReturn:     "",
			mockError:      domain.InternalError("Failed to send email", errors.New("internal error")),
			expectedStatus: fiber.StatusInternalServerError,
			expectedAPIRes: map[string]interface{}{
				"status":   "ERROR",
				"trace_id": "test-trace-id",
				"message":  "Internal server error",
				"data":     nil,
			},
		},
		{
			name: "GenericError",
			requestBody: domain.EmailRequest{
				To:        []string{"test@example.com"},
				Subject:   "Test Email",
				PlainBody: "This is a test email",
			},
			mockReturn:     "",
			mockError:      errors.New("unexpected error"),
			expectedStatus: fiber.StatusInternalServerError,
			expectedAPIRes: map[string]interface{}{
				"status":   "ERROR",
				"trace_id": "test-trace-id",
				"message":  "Failed to send email",
				"data":     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app, mockEmailService := setupApp()
			handler := NewEmailHandler(mockEmailService)

			// Setup route
			app.Post("/emails/manual", handler.SendEmail)

			// Setup expectations
			mockEmailService.On("SendEmail", mock.Anything, mock.MatchedBy(func(req *domain.EmailRequest) bool {
				return req.To[0] == tt.requestBody.To[0] && req.Subject == tt.requestBody.Subject
			})).Return(tt.mockReturn, tt.mockError)

			// Siapkan request body
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/emails/manual", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Jalankan request
			resp, _ := app.Test(req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response body
			var respBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&respBody)

			// Verifikasi response body sesuai APIResponse
			for key, expectedValue := range tt.expectedAPIRes {
				if key == "data" && expectedValue != nil {
					assert.Equal(t, expectedValue, respBody[key])
				} else {
					assert.Equal(t, expectedValue, respBody[key])
				}
			}

			// Verifikasi mock dipanggil
			mockEmailService.AssertExpectations(t)
		})
	}
}

// TestGetEmailStatus menguji fungsi GetEmailStatus
func TestGetEmailStatus(t *testing.T) {
	// Setup waktu untuk test
	now := time.Now()

	// Persiapkan beberapa skenario uji
	tests := []struct {
		name           string
		emailID        string
		mockReturn     *domain.EmailStatus
		mockError      error
		expectedStatus int
		expectedAPIRes map[string]interface{}
	}{
		{
			name:    "Success",
			emailID: "test-email-id",
			mockReturn: &domain.EmailStatus{
				ID:        "test-email-id",
				Status:    domain.StatusSent,
				SentAt:    &now,
				UpdatedAt: now,
			},
			mockError:      nil,
			expectedStatus: fiber.StatusOK,
			expectedAPIRes: map[string]interface{}{
				"status":   "SUCCESS",
				"trace_id": "test-trace-id",
				"message":  "Success",
				"data": map[string]interface{}{
					"id":     "test-email-id",
					"status": string(domain.StatusSent),
					// sentAt dan updatedAt tidak dicek detail
				},
			},
		},
		{
			name:           "EmailNotFound",
			emailID:        "non-existent-id",
			mockReturn:     nil,
			mockError:      domain.NotFoundError("Email not found"),
			expectedStatus: fiber.StatusNotFound,
			expectedAPIRes: map[string]interface{}{
				"status":   "ERROR",
				"trace_id": "test-trace-id",
				"message":  "Email not found",
				"data":     nil,
			},
		},
		{
			name:           "InternalError",
			emailID:        "test-email-id",
			mockReturn:     nil,
			mockError:      domain.InternalError("Database error", errors.New("connection lost")),
			expectedStatus: fiber.StatusInternalServerError,
			expectedAPIRes: map[string]interface{}{
				"status":   "ERROR",
				"trace_id": "test-trace-id",
				"message":  "Database error",
				"data":     nil,
			},
		},
		{
			name:           "GenericError",
			emailID:        "test-email-id",
			mockReturn:     nil,
			mockError:      errors.New("unexpected error"),
			expectedStatus: fiber.StatusInternalServerError,
			expectedAPIRes: map[string]interface{}{
				"status":   "ERROR",
				"trace_id": "test-trace-id",
				"message":  "Failed to get email status",
				"data":     nil,
			},
		},
		{
			name:           "MissingID",
			emailID:        "",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: fiber.StatusBadRequest,
			expectedAPIRes: map[string]interface{}{
				"status":   "ERROR",
				"trace_id": "test-trace-id",
				"message":  "Email ID is required",
				"data":     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app, mockEmailService := setupApp()
			handler := NewEmailHandler(mockEmailService)

			// Setup route
			app.Get("/emails/:id/status", handler.GetEmailStatus)

			// Setup mock expectations (kecuali untuk kasus MissingID)
			if tt.emailID != "" {
				mockEmailService.On("GetStatus", mock.Anything, tt.emailID).Return(tt.mockReturn, tt.mockError)
			}

			// Buat request
			var url string
			if tt.name == "MissingID" {
				// Untuk kasus MissingID, kita perlu mensimulasikan permintaan ke handler langsung
				// karena Fiber router akan menolak URL dengan parameter path kosong
				app = fiber.New()
				app.Use(func(c *fiber.Ctx) error {
					c.Locals(middleware.TraceIDKey, "test-trace-id")
					return c.Next()
				})

				// Setup handler langsung
				handler := NewEmailHandler(mockEmailService)
				app.Get("/test-empty-id", func(c *fiber.Ctx) error {
					// Mengatur parameter ID secara manual ke string kosong
					c.Params("id", "")
					return handler.GetEmailStatus(c)
				})

				url = "/test-empty-id"
			} else {
				url = "/emails/" + tt.emailID + "/status"
			}

			req := httptest.NewRequest("GET", url, nil)

			// Jalankan request
			resp, _ := app.Test(req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response body
			var respBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&respBody)

			// Verifikasi expected body sesuai APIResponse
			for key, expectedValue := range tt.expectedAPIRes {
				if key == "data" && expectedValue != nil {
					// Untuk Success, data adalah map status
					dataMap, ok := respBody[key].(map[string]interface{})
					if ok {
						for k, v := range expectedValue.(map[string]interface{}) {
							assert.Equal(t, v, dataMap[k])
						}
					} else {
						assert.Equal(t, expectedValue, respBody[key])
					}
				} else {
					assert.Equal(t, expectedValue, respBody[key])
				}
			}

			// Verifikasi mock dipanggil (kecuali untuk kasus MissingID)
			if tt.emailID != "" && tt.name != "MissingID" {
				mockEmailService.AssertExpectations(t)
			}
		})
	}
}

// TestNewEmailHandler menguji pembuatan instance EmailHandler
func TestNewEmailHandler(t *testing.T) {
	// Setup
	mockEmailService := new(MockEmailService)

	// Act
	handler := NewEmailHandler(mockEmailService)

	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, mockEmailService, handler.emailService)
}
