// Package telemetry menyediakan implementasi logging dan telemetri untuk aplikasi
package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelLogger adalah implementasi Logger menggunakan OpenTelemetry
type OTelLogger struct {
	*BaseLogger
	logger     *SimpleLogger
	config     Config
	attributes []attribute.KeyValue
	resource   *resource.Resource
}

// SimpleLogger adalah implementasi logger yang sederhana
type SimpleLogger struct {
	logger *log.Logger
}

// NewSimpleLogger membuat instance baru dari SimpleLogger
func NewSimpleLogger(w io.Writer) *SimpleLogger {
	return &SimpleLogger{
		logger: log.New(w, "", log.LstdFlags),
	}
}

// Log mencatat log dengan format yang ditentukan
func (l *SimpleLogger) Log(level string, msg string, attrs map[string]interface{}) {
	// Buat data log dalam format JSON
	logData := map[string]interface{}{
		"level":     level,
		"message":   msg,
		"timestamp": time.Now().Format(time.RFC3339Nano),
	}

	// Tambahkan atribut lainnya
	for k, v := range attrs {
		logData[k] = v
	}

	// Marshal ke JSON
	jsonData, err := json.Marshal(logData)
	if err != nil {
		l.logger.Printf("ERROR marshalling log: %v", err)
		return
	}

	// Cetak log
	l.logger.Println(string(jsonData))
}

// NewOTelLogger membuat instance baru dari OTelLogger dengan konfigurasi tertentu
func NewOTelLogger(cfg Config) (*OTelLogger, error) {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			attribute.String("service.version", "1.0.0"),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create log processor and exporter
	var writers []io.Writer

	// Setup provider untuk setiap tipe yang dikonfigurasi
	for _, providerCfg := range cfg.Providers {
		switch providerCfg.Type {
		case "stdout":
			pretty, _ := providerCfg.Config["pretty"].(bool)
			if pretty {
				writers = append(writers, os.Stdout)
			}

		case "file":
			// Implementasi file logger akan ditambahkan di sini
			// TODO: Implementasi file logger

		case "cloudwatch":
			// CloudWatch akan diimplementasikan dalam file terpisah
			// untuk modul cloudwatch di pkg/cloudwatch/logger.go
			// TODO: Implementasi CloudWatch logger

		case "otlp":
			// Implementasi OTLP provider akan ditambahkan di sini
			// TODO: Implementasi OTLP logger
		}
	}

	// Jika tidak ada provider yang dikonfigurasi, gunakan stdout
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	// Buat multi-writer jika ada lebih dari satu writer
	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else if len(writers) > 1 {
		writer = io.MultiWriter(writers...)
	} else {
		writer = os.Stdout
	}

	// Buat logger sederhana
	simpleLogger := NewSimpleLogger(writer)

	// Buat instance OTelLogger
	otelLogger := &OTelLogger{
		BaseLogger: NewBaseLogger(),
		logger:     simpleLogger,
		config:     cfg,
		attributes: []attribute.KeyValue{},
		resource:   res,
	}

	return otelLogger, nil
}

// Debug logs dengan level Debug
func (l *OTelLogger) Debug(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, DebugLevel, msg, fields...)
}

// Info logs dengan level Info
func (l *OTelLogger) Info(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, InfoLevel, msg, fields...)
}

// Warn logs dengan level Warn
func (l *OTelLogger) Warn(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, WarnLevel, msg, fields...)
}

// Error logs dengan level Error
func (l *OTelLogger) Error(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, ErrorLevel, msg, fields...)
}

// Fatal logs dengan level Fatal dan kemudian menghentikan aplikasi
func (l *OTelLogger) Fatal(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, FatalLevel, msg, fields...)
	os.Exit(1)
}

// Flush memaksa semua log yang masih dalam buffer untuk dikirim
func (l *OTelLogger) Flush() {
	// Di implementasi sederhana ini, tidak perlu melakukan apapun
}

// WithContext menambahkan context ke logger instance
func (l *OTelLogger) WithContext(ctx context.Context) Logger {
	newLogger := &OTelLogger{
		BaseLogger: &BaseLogger{
			fields: make(Fields),
		},
		logger:     l.logger,
		config:     l.config,
		attributes: l.attributes,
		resource:   l.resource,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Extract trace information from context if available
	if span := trace.SpanFromContext(ctx); span != nil {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		newLogger.fields["trace_id"] = traceID
		newLogger.fields["span_id"] = spanID

		newLogger.attributes = append(
			newLogger.attributes,
			attribute.String("trace_id", traceID),
			attribute.String("span_id", spanID),
		)
	}

	return newLogger
}

// log adalah implementasi internal untuk semua metode log
func (l *OTelLogger) log(ctx context.Context, level LogLevel, msg string, fields ...Fields) {
	// Merge semua fields
	mergedFields := l.mergeFields(fields...)

	// Convert to attributes map
	attrs := make(map[string]interface{})

	// Add resource attributes
	if l.resource != nil {
		for _, attr := range l.resource.Attributes() {
			attrs[string(attr.Key)] = attr.Value.AsInterface()
		}
	}

	// Add logger attributes
	for _, attr := range l.attributes {
		attrs[string(attr.Key)] = attr.Value.AsInterface()
	}

	// Add fields
	for k, v := range mergedFields {
		attrs[k] = v
	}

	// Ekstrak trace context jika ada
	if span := trace.SpanFromContext(ctx); span != nil {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		// Tambahkan trace dan span ID ke log jika belum ada
		if _, ok := attrs["trace_id"]; !ok {
			attrs["trace_id"] = traceID
		}
		if _, ok := attrs["span_id"]; !ok {
			attrs["span_id"] = spanID
		}
	}

	// Tambahkan request ID dari context jika ada
	if reqID := ctx.Value("request_id"); reqID != nil {
		attrs["request_id"] = reqID
	}

	// Log to SimpleLogger
	l.logger.Log(string(level), msg, attrs)
}

// FromConfig membuat Logger baru berdasarkan konfigurasi
func FromConfig(cfg Config) (Logger, error) {
	return NewOTelLogger(cfg)
}

// NewDefaultLogger membuat Logger dengan konfigurasi default
func NewDefaultLogger() (Logger, error) {
	return FromConfig(DefaultConfig())
}

// ConfigureFromAppConfig membuat Logger dari konfigurasi aplikasi
func ConfigureFromAppConfig(appCfg interface{}) (Logger, error) {
	telemetryCfg := FromAppConfig(appCfg)
	return FromConfig(telemetryCfg)
}
