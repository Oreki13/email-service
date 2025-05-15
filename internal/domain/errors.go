package domain

import (
	"fmt"
)

// ErrorType adalah enum untuk tipe error
type ErrorType string

const (
	// ErrorTypeValidation untuk error validasi
	ErrorTypeValidation ErrorType = "VALIDATION_ERROR"
	// ErrorTypeNotFound untuk error ketika resource tidak ditemukan
	ErrorTypeNotFound ErrorType = "NOT_FOUND"
	// ErrorTypeDatabase untuk error yang terjadi pada database
	ErrorTypeDatabase ErrorType = "DATABASE_ERROR"
	// ErrorTypeExternal untuk error dari layanan eksternal
	ErrorTypeExternal ErrorType = "EXTERNAL_SERVICE_ERROR"
	// ErrorTypeInternal untuk error internal
	ErrorTypeInternal ErrorType = "INTERNAL_ERROR"
	// ErrorTypeUnauthorized untuk error otorisasi
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	// ErrorTypeForbidden untuk error permission
	ErrorTypeForbidden ErrorType = "FORBIDDEN"
	// ErrorTypeRateLimit untuk error rate limiting
	ErrorTypeRateLimit ErrorType = "RATE_LIMIT_EXCEEDED"
)

// AppError merepresentasikan error yang terjadi dalam aplikasi
type AppError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Err     error     `json:"-"`
	Details any       `json:"details,omitempty"`
}

// Error mengimplementasikan interface error
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap mengimplementasikan interface unwrap untuk error chaining
func (e *AppError) Unwrap() error {
	return e.Err
}

// ValidationError membuat error validasi baru
func ValidationError(message string, details any) *AppError {
	return &AppError{
		Type:    ErrorTypeValidation,
		Message: message,
		Details: details,
	}
}

// NotFoundError membuat error not found baru
func NotFoundError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeNotFound,
		Message: message,
	}
}

// WithError menambahkan underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// DatabaseError membuat error database baru
func DatabaseError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeDatabase,
		Message: message,
		Err:     err,
	}
}

// ExternalServiceError membuat error layanan eksternal baru
func ExternalServiceError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeExternal,
		Message: message,
		Err:     err,
	}
}

// InternalError membuat error internal baru
func InternalError(message string, err error) *AppError {
	return &AppError{
		Type:    ErrorTypeInternal,
		Message: message,
		Err:     err,
	}
}

// UnauthorizedError membuat error unauthorized baru
func UnauthorizedError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeUnauthorized,
		Message: message,
	}
}

// ForbiddenError membuat error forbidden baru
func ForbiddenError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeForbidden,
		Message: message,
	}
}

// RateLimitError membuat error rate limit baru
func RateLimitError(message string) *AppError {
	return &AppError{
		Type:    ErrorTypeRateLimit,
		Message: message,
	}
}
