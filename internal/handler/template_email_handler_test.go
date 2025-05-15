package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"email-service/internal/domain"
	"email-service/internal/dto"
)

// Mock untuk TemplatedEmailService
type MockTemplatedEmailService struct {
	mock.Mock
}

func (m *MockTemplatedEmailService) GetTemplateInfo(ctx context.Context) (map[string]domain.TemplateInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]domain.TemplateInfo), args.Error(1)
}

func (m *MockTemplatedEmailService) SendWelcomeEmail(ctx context.Context, req *domain.WelcomeEmailRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *MockTemplatedEmailService) SendPasswordResetEmail(ctx context.Context, req *domain.PasswordResetEmailRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *MockTemplatedEmailService) SendNotificationEmail(ctx context.Context, req *domain.NotificationEmailRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func TestGetTemplateInfo_Success(t *testing.T) {
	app := fiber.New()
	mockService := new(MockTemplatedEmailService)
	mockLogger := new(MockLogger)
	h := NewTemplateEmailHandler(mockService, mockLogger)

	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockService.On("GetTemplateInfo", mock.Anything).Return(map[string]domain.TemplateInfo{
		"welcome": {Name: "welcome"},
	}, nil)

	app.Get("/templates", h.GetTemplateInfo)
	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusSuccess, apiResp.Status)
	assert.NotNil(t, apiResp.Data)
}

func TestGetTemplateInfo_Error(t *testing.T) {
	app := fiber.New()
	mockService := new(MockTemplatedEmailService)
	mockLogger := new(MockLogger)
	h := NewTemplateEmailHandler(mockService, mockLogger)

	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()
	mockService.On("GetTemplateInfo", mock.Anything).Return(map[string]domain.TemplateInfo{}, errors.New("db error"))

	app.Get("/templates", h.GetTemplateInfo)
	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusError, apiResp.Status)
}

func TestSendWelcomeEmail_ValidationError(t *testing.T) {
	app := fiber.New()
	mockService := new(MockTemplatedEmailService)
	mockLogger := new(MockLogger)
	h := NewTemplateEmailHandler(mockService, mockLogger)

	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()

	app.Post("/templates/welcome", h.SendWelcomeEmail)
	// Kirim body yang tidak valid
	req := httptest.NewRequest(http.MethodPost, "/templates/welcome", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSendWelcomeEmail_Success(t *testing.T) {
	app := fiber.New()
	mockService := new(MockTemplatedEmailService)
	mockLogger := new(MockLogger)
	h := NewTemplateEmailHandler(mockService, mockLogger)

	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockService.On("SendWelcomeEmail", mock.Anything, mock.Anything).Return("email-id-123", nil)

	app.Post("/templates/welcome", h.SendWelcomeEmail)
	// Perbaiki assignment To menjadi slice of string
	body := domain.WelcomeEmailRequest{To: []string{"test@example.com"}}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/templates/welcome", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusSuccess, apiResp.Status)
	assert.NotNil(t, apiResp.Data)
}

func TestSendWelcomeEmail_AppErrorValidation(t *testing.T) {
	app := fiber.New()
	mockService := new(MockTemplatedEmailService)
	mockLogger := new(MockLogger)
	h := NewTemplateEmailHandler(mockService, mockLogger)

	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()
	appErr := &domain.AppError{Type: domain.ErrorTypeValidation, Message: "validation failed", Details: map[string]string{"to": "required"}}
	mockService.On("SendWelcomeEmail", mock.Anything, mock.Anything).Return("", appErr)

	app.Post("/templates/welcome", h.SendWelcomeEmail)
	// Perbaiki assignment To menjadi slice of string
	body := domain.WelcomeEmailRequest{To: []string{""}}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/templates/welcome", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusError, apiResp.Status)
	assert.Equal(t, "validation failed", apiResp.Message)
}

// Test serupa dapat dibuat untuk SendPasswordResetEmail dan SendNotificationEmail
type dummyPasswordResetReq struct{ To string }
type dummyNotificationReq struct{ To string }
