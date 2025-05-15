// Package telemetry menyediakan implementasi logging dan telemetri untuk aplikasi
package telemetry

import (
	"context"
)

// LogLevel mendefinisikan tingkat severity log
type LogLevel string

const (
	// DebugLevel untuk log debugging
	DebugLevel LogLevel = "debug"
	// InfoLevel untuk log informasi umum
	InfoLevel LogLevel = "info"
	// WarnLevel untuk log peringatan
	WarnLevel LogLevel = "warn"
	// ErrorLevel untuk log error
	ErrorLevel LogLevel = "error"
	// FatalLevel untuk log fatal error (biasanya menyebabkan aplikasi berhenti)
	FatalLevel LogLevel = "fatal"
)

// Fields adalah tipe untuk menyimpan data tambahan dalam log
type Fields map[string]interface{}

// Logger mendefinisikan interface untuk semua implementasi logger
type Logger interface {
	// Debug logs dengan level Debug
	Debug(ctx context.Context, msg string, fields ...Fields)
	// Info logs dengan level Info
	Info(ctx context.Context, msg string, fields ...Fields)
	// Warn logs dengan level Warn
	Warn(ctx context.Context, msg string, fields ...Fields)
	// Error logs dengan level Error
	Error(ctx context.Context, msg string, fields ...Fields)
	// Fatal logs dengan level Fatal dan kemudian menghentikan aplikasi
	Fatal(ctx context.Context, msg string, fields ...Fields)
	// WithField menambahkan field ke logger instance
	WithField(key string, value interface{}) Logger
	// WithFields menambahkan multiple fields ke logger instance
	WithFields(fields Fields) Logger
	// WithError menambahkan error sebagai field ke logger instance
	WithError(err error) Logger
	// WithContext menambahkan context ke logger instance
	WithContext(ctx context.Context) Logger
	// Flush memaksa semua log yang masih dalam buffer untuk dikirim
	Flush()
}

// LoggerMiddleware provides functionality to log HTTP and gRPC requests
type LoggerMiddleware interface {
	// HTTPMiddleware returns a middleware for HTTP requests
	HTTPMiddleware() interface{}
	// GRPCMiddleware returns an interceptor for gRPC requests
	GRPCMiddleware() interface{}
}

// BaseLogger implementasi dasar untuk Logger interface
type BaseLogger struct {
	fields Fields
}

// NewBaseLogger membuat instance baru dari BaseLogger
func NewBaseLogger() *BaseLogger {
	return &BaseLogger{
		fields: make(Fields),
	}
}

// Debug logs dengan level Debug
func (l *BaseLogger) Debug(ctx context.Context, msg string, fields ...Fields) {
	// Implementasi dasar - override di implementasi konkret
}

// Info logs dengan level Info
func (l *BaseLogger) Info(ctx context.Context, msg string, fields ...Fields) {
	// Implementasi dasar - override di implementasi konkret
}

// Warn logs dengan level Warn
func (l *BaseLogger) Warn(ctx context.Context, msg string, fields ...Fields) {
	// Implementasi dasar - override di implementasi konkret
}

// Error logs dengan level Error
func (l *BaseLogger) Error(ctx context.Context, msg string, fields ...Fields) {
	// Implementasi dasar - override di implementasi konkret
}

// Fatal logs dengan level Fatal dan kemudian menghentikan aplikasi
func (l *BaseLogger) Fatal(ctx context.Context, msg string, fields ...Fields) {
	// Implementasi dasar - override di implementasi konkret
}

// Flush memaksa semua log yang masih dalam buffer untuk dikirim
func (l *BaseLogger) Flush() {
	// Implementasi dasar - override di implementasi konkret
}

// WithField menambahkan field ke logger instance
func (l *BaseLogger) WithField(key string, value interface{}) Logger {
	newLogger := &BaseLogger{
		fields: make(Fields),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields menambahkan multiple fields ke logger instance
func (l *BaseLogger) WithFields(fields Fields) Logger {
	newLogger := &BaseLogger{
		fields: make(Fields),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithError menambahkan error sebagai field ke logger instance
func (l *BaseLogger) WithError(err error) Logger {
	return l.WithField("error", err.Error())
}

// WithContext menambahkan context ke logger instance
func (l *BaseLogger) WithContext(ctx context.Context) Logger {
	// Implementasi dasar, bisa diperluas dengan ekstraksi trace ID dari context
	return l
}

// mergeFields menggabungkan fields yang ada dengan fields yang diberikan
func (l *BaseLogger) mergeFields(fields ...Fields) Fields {
	result := make(Fields)

	// Copy base fields
	for k, v := range l.fields {
		result[k] = v
	}

	// Add all provided fields
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}

	return result
}

// GetFields mengembalikan semua fields yang ada di logger
func (l *BaseLogger) GetFields() Fields {
	result := make(Fields)
	for k, v := range l.fields {
		result[k] = v
	}
	return result
}

// SetField menetapkan nilai field tertentu
func (l *BaseLogger) SetField(key string, value interface{}) {
	l.fields[key] = value
}

// MergeFields menggabungkan fields yang ada dengan fields yang diberikan
func (l *BaseLogger) MergeFields(fields ...Fields) Fields {
	return l.mergeFields(fields...)
}
