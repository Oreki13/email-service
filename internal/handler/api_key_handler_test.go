package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"email-service/internal/domain"
	"email-service/internal/dto"
	"email-service/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIKeyService adalah mock untuk APIKeyServicer
// Menggunakan testify/mock agar mudah diatur ekspektasi dan hasilnya
type MockAPIKeyService struct {
	mock.Mock
}

func (m *MockAPIKeyService) CreateAPIKey(ctx context.Context, req *domain.APIKeyCreateRequest) (*domain.APIKey, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}
func (m *MockAPIKeyService) GetAPIKeys(ctx context.Context, page, limit int) ([]*domain.APIKey, int, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.APIKey), args.Int(1), args.Error(2)
}
func (m *MockAPIKeyService) GetAPIKey(ctx context.Context, id string) (*domain.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}
func (m *MockAPIKeyService) UpdateAPIKey(ctx context.Context, id string, req *domain.APIKeyUpdateRequest) (*domain.APIKey, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}
func (m *MockAPIKeyService) DeleteAPIKey(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockAPIKeyService) VerifyAPIKey(ctx context.Context, key string) (*domain.APIKey, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}

// Hapus deklarasi MockLogger di sini, gunakan import dari mock_logger.go

// setupFiberAppAPIKey untuk testing handler APIKey
func setupFiberAppAPIKey(service *MockAPIKeyService, logger *MockLogger) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.TraceIDKey, "test-trace-id")
		return c.Next()
	})
	h := NewAPIKeyHandler(service, logger)
	app.Post("/api-keys", h.CreateAPIKey)
	app.Get("/api-keys", h.GetAPIKeys)
	app.Get("/api-keys/:id", h.GetAPIKey)
	app.Put("/api-keys/:id", h.UpdateAPIKey)
	app.Delete("/api-keys/:id", h.DeleteAPIKey)
	return app
}

// Helper untuk setup default mock logger agar tidak panic
func setupMockLoggerDefault(logger *MockLogger) {
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Warn", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Fatal", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("WithField", mock.Anything, mock.Anything).Return(logger)
	logger.On("WithFields", mock.Anything).Return(logger)
	logger.On("WithError", mock.Anything).Return(logger)
	logger.On("WithContext", mock.Anything).Return(logger)
	logger.On("Flush").Return()
}

func TestCreateAPIKey_Success(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	reqBody := domain.APIKeyCreateRequest{
		Name:        "Test Key",
		Description: "desc",
		ServiceName: "svc",
		ExpiryDays:  30,
	}
	apiKey := &domain.APIKey{ID: "1", Name: "Test Key", ServiceName: "svc", IsActive: true}
	service.On("CreateAPIKey", mock.Anything, &reqBody).Return(apiKey, nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusSuccess, apiResp.Status)
	assert.Equal(t, "API key created successfully", apiResp.Message)
	assert.NotNil(t, apiResp.Data)
}

func TestCreateAPIKey_ValidationError(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	// Name kosong (invalid)
	reqBody := domain.APIKeyCreateRequest{
		Name:        "",
		ServiceName: "svc",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetAPIKey_Success(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	apiKey := &domain.APIKey{ID: "1", Name: "Test Key", ServiceName: "svc", IsActive: true}
	service.On("GetAPIKey", mock.Anything, "1").Return(apiKey, nil)

	req := httptest.NewRequest("GET", "/api-keys/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusSuccess, apiResp.Status)
	assert.NotNil(t, apiResp.Data)
}

func TestGetAPIKey_NotFound(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	service.On("GetAPIKey", mock.Anything, "notfound").Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/api-keys/notfound", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestUpdateAPIKey_Success(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	updateReq := domain.APIKeyUpdateRequest{
		Description: ptrStr("desc updated"),
		IsActive:    ptrBool(true),
	}
	apiKey := &domain.APIKey{ID: "1", Name: "Test Key", ServiceName: "svc", IsActive: true, Description: "desc updated"}
	service.On("UpdateAPIKey", mock.Anything, "1", &updateReq).Return(apiKey, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/api-keys/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusSuccess, apiResp.Status)
	assert.NotNil(t, apiResp.Data)
}

func TestUpdateAPIKey_ValidationError(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	// Tidak ada field yang diupdate (semua nil)
	updateReq := domain.APIKeyUpdateRequest{}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/api-keys/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeleteAPIKey_Success(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	service.On("DeleteAPIKey", mock.Anything, "1").Return(nil)

	req := httptest.NewRequest("DELETE", "/api-keys/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusSuccess, apiResp.Status)
}

func TestDeleteAPIKey_Error(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	service.On("DeleteAPIKey", mock.Anything, "notfound").Return(errors.New("not found"))

	req := httptest.NewRequest("DELETE", "/api-keys/notfound", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestGetAPIKeys_Success(t *testing.T) {
	service := new(MockAPIKeyService)
	logger := &MockLogger{}
	setupMockLoggerDefault(logger)
	app := setupFiberAppAPIKey(service, logger)

	apiKeys := []*domain.APIKey{{ID: "1", Name: "Test Key", ServiceName: "svc", IsActive: true}}
	total := 1
	service.On("GetAPIKeys", mock.Anything, 1, 10).Return(apiKeys, total, nil)

	req := httptest.NewRequest("GET", "/api-keys?page=1&limit=10", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var apiResp dto.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, dto.StatusSuccess, apiResp.Status)
	assert.NotNil(t, apiResp.Data)
}

func ptrStr(s string) *string { return &s }
func ptrBool(b bool) *bool    { return &b }
