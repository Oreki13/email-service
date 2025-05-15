package handler

import (
	"context"
	"email-service/pkg/telemetry"

	"github.com/stretchr/testify/mock"
)

// MockLogger adalah mock untuk interface Logger
// Dapat digunakan di seluruh unit test handler
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...telemetry.Fields) {
	m.Called(ctx, msg, fields)
}
func (m *MockLogger) Info(ctx context.Context, msg string, fields ...telemetry.Fields) {
	m.Called(ctx, msg, fields)
}
func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...telemetry.Fields) {
	m.Called(ctx, msg, fields)
}
func (m *MockLogger) Error(ctx context.Context, msg string, fields ...telemetry.Fields) {
	m.Called(ctx, msg, fields)
}
func (m *MockLogger) Fatal(ctx context.Context, msg string, fields ...telemetry.Fields) {
	m.Called(ctx, msg, fields)
}
func (m *MockLogger) WithField(key string, value interface{}) telemetry.Logger {
	return m
}
func (m *MockLogger) WithFields(fields telemetry.Fields) telemetry.Logger {
	return m
}
func (m *MockLogger) WithError(err error) telemetry.Logger {
	return m
}
func (m *MockLogger) WithContext(ctx context.Context) telemetry.Logger {
	return m
}
func (m *MockLogger) Flush() {}
