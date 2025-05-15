// Package cloudwatch menyediakan implementasi adapter untuk AWS CloudWatch
package cloudwatch

import (
	"context"
	"fmt"
	"os"

	"email-service/pkg/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Logger adalah implementasi dari telemetry.Logger yang menggunakan CloudWatch
type Logger struct {
	*telemetry.BaseLogger
	cwLogger *CloudWatchLogger
}

// NewCloudWatchTelemetryLogger membuat instance baru Logger yang menggunakan CloudWatch
func NewCloudWatchTelemetryLogger(cfg CloudWatchConfig) (telemetry.Logger, error) {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("email-service"),
			attribute.String("service.version", "1.0.0"),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create CloudWatch logger
	cwLogger, err := NewCloudWatchLogger(cfg, res)
	if err != nil {
		return nil, err
	}

	// Create telemetry logger
	logger := &Logger{
		BaseLogger: telemetry.NewBaseLogger(),
		cwLogger:   cwLogger,
	}

	return logger, nil
}

// NewFromAppConfig membuat instance Logger dari konfigurasi aplikasi
func NewFromAppConfig(appConfig interface{}) (telemetry.Logger, error) {
	// Ekstrak konfigurasi AWS dari appConfig
	// Ini perlu disesuaikan dengan struktur konfigurasi aplikasi Anda

	// Contoh untuk config.Config dari internal/config/config.go
	if cfg, ok := appConfig.(*struct {
		AWS struct {
			Region    string
			AccessKey string
			SecretKey string
			LogGroup  string
		}
		App struct {
			Environment string
			ServiceName string
		}
	}); ok {
		// Create CloudWatch config
		cwConfig := CloudWatchConfig{
			Region:        cfg.AWS.Region,
			AccessKey:     cfg.AWS.AccessKey,
			SecretKey:     cfg.AWS.SecretKey,
			LogGroup:      cfg.AWS.LogGroup,
			LogStream:     fmt.Sprintf("%s-%s-%d", cfg.App.ServiceName, cfg.App.Environment, os.Getpid()),
			RetentionDays: 30,
			Environment:   cfg.App.Environment,
		}

		return NewCloudWatchTelemetryLogger(cwConfig)
	}

	return nil, fmt.Errorf("unsupported config type: %T", appConfig)
}

// Debug logs dengan level Debug
func (l *Logger) Debug(ctx context.Context, msg string, fields ...telemetry.Fields) {
	l.log(ctx, telemetry.DebugLevel, msg, fields...)
}

// Info logs dengan level Info
func (l *Logger) Info(ctx context.Context, msg string, fields ...telemetry.Fields) {
	l.log(ctx, telemetry.InfoLevel, msg, fields...)
}

// Warn logs dengan level Warn
func (l *Logger) Warn(ctx context.Context, msg string, fields ...telemetry.Fields) {
	l.log(ctx, telemetry.WarnLevel, msg, fields...)
}

// Error logs dengan level Error
func (l *Logger) Error(ctx context.Context, msg string, fields ...telemetry.Fields) {
	l.log(ctx, telemetry.ErrorLevel, msg, fields...)
}

// Fatal logs dengan level Fatal dan kemudian menghentikan aplikasi
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...telemetry.Fields) {
	l.log(ctx, telemetry.FatalLevel, msg, fields...)
	os.Exit(1)
}

// WithField menambahkan field ke logger instance
func (l *Logger) WithField(key string, value interface{}) telemetry.Logger {
	return l.WithFields(telemetry.Fields{key: value})
}

// WithFields menambahkan multiple fields ke logger instance
func (l *Logger) WithFields(fields telemetry.Fields) telemetry.Logger {
	newLogger := &Logger{
		BaseLogger: &telemetry.BaseLogger{},
		cwLogger:   l.cwLogger,
	}

	// Copy fields dari base logger
	for k, v := range l.BaseLogger.GetFields() {
		newLogger.BaseLogger.SetField(k, v)
	}

	// Add new fields
	for k, v := range fields {
		newLogger.BaseLogger.SetField(k, v)
	}

	return newLogger
}

// WithError menambahkan error sebagai field ke logger instance
func (l *Logger) WithError(err error) telemetry.Logger {
	return l.WithField("error", err.Error())
}

// WithContext menambahkan context ke logger instance
func (l *Logger) WithContext(ctx context.Context) telemetry.Logger {
	// Extract trace information from context if available
	// Implementasi ini sama dengan OTelLogger

	newLogger := &Logger{
		BaseLogger: &telemetry.BaseLogger{},
		cwLogger:   l.cwLogger,
	}

	// Copy existing fields
	for k, v := range l.BaseLogger.GetFields() {
		newLogger.BaseLogger.SetField(k, v)
	}

	return newLogger
}

// Flush memaksa semua log yang masih dalam buffer untuk dikirim
func (l *Logger) Flush() {
	if err := l.cwLogger.ForceFlush(context.Background()); err != nil {
		fmt.Printf("Failed to flush logs: %v\n", err)
	}
}

// log adalah implementasi internal untuk semua metode log
func (l *Logger) log(ctx context.Context, level telemetry.LogLevel, msg string, fields ...telemetry.Fields) {
	// Merge fields
	mergedFields := l.BaseLogger.MergeFields(fields...)

	// Convert to attributes map for CloudWatch
	attrs := make(map[string]interface{})
	for k, v := range mergedFields {
		attrs[k] = v
	}

	// Add severity level
	levelStr := "INFO"
	switch level {
	case telemetry.DebugLevel:
		levelStr = "DEBUG"
	case telemetry.InfoLevel:
		levelStr = "INFO"
	case telemetry.WarnLevel:
		levelStr = "WARN"
	case telemetry.ErrorLevel:
		levelStr = "ERROR"
	case telemetry.FatalLevel:
		levelStr = "FATAL"
	}

	// Ekstrak trace context dari context jika ada
	traceIDKey := "trace_id"
	spanIDKey := "span_id"
	requestIDKey := "request_id"

	// Cek apakah context memiliki trace information
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceID := spanContext.TraceID().String()
		spanID := spanContext.SpanID().String()

		// Tambahkan trace dan span ID ke log jika belum ada
		if _, ok := attrs[traceIDKey]; !ok {
			attrs[traceIDKey] = traceID
		}
		if _, ok := attrs[spanIDKey]; !ok {
			attrs[spanIDKey] = spanID
		}
	}

	// Tambahkan request ID dari context jika ada
	if reqID := ctx.Value(requestIDKey); reqID != nil {
		if _, ok := attrs[requestIDKey]; !ok {
			attrs[requestIDKey] = reqID
		}
	}

	// Send log to CloudWatch dengan context
	// Context bisa digunakan untuk handling cancellation atau timeout
	// jika operasi logging berjalan terlalu lama
	l.cwLogger.Log(levelStr, msg, attrs)
}

// Shutdown menutup logger dengan cara yang bersih
func (l *Logger) Shutdown(ctx context.Context) error {
	return l.cwLogger.Shutdown(ctx)
}
