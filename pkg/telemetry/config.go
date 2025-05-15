// Package telemetry menyediakan implementasi logging dan telemetri untuk aplikasi
package telemetry

import (
	"time"
)

// Config berisi konfigurasi untuk sistem telemetri dan logging
type Config struct {
	// ServiceName adalah nama layanan yang akan ditampilkan dalam logs
	ServiceName string
	// Environment adalah lingkungan tempat aplikasi berjalan (development, staging, production)
	Environment string
	// LogLevel adalah level minimum log yang akan dihasilkan
	LogLevel LogLevel
	// Providers adalah daftar provider logging yang akan digunakan
	Providers []ProviderConfig
	// TraceConfig berisi konfigurasi untuk tracing
	TraceConfig TraceConfig
	// BatchSize menentukan berapa banyak log yang dikumpulkan sebelum dikirim ke provider
	BatchSize int
	// BatchInterval adalah interval maksimum untuk mengirim batch log
	BatchInterval time.Duration
}

// ProviderConfig berisi konfigurasi untuk provider logging tertentu
type ProviderConfig struct {
	// Type menentukan jenis provider (cloudwatch, stdout, file, etc)
	Type string
	// Config berisi konfigurasi spesifik untuk provider
	Config map[string]interface{}
}

// TraceConfig berisi konfigurasi untuk sistem distributed tracing
type TraceConfig struct {
	// Enabled menentukan apakah tracing diaktifkan
	Enabled bool
	// SamplingRate menentukan persentase request yang akan dicatat (0.0-1.0)
	SamplingRate float64
	// ExporterType menentukan jenis exporter untuk trace data (otlp, zipkin, jaeger)
	ExporterType string
	// ExporterConfig berisi konfigurasi spesifik untuk exporter
	ExporterConfig map[string]interface{}
}

// DefaultConfig mengembalikan konfigurasi default untuk telemetri
func DefaultConfig() Config {
	return Config{
		ServiceName:   "email-service",
		Environment:   "development",
		LogLevel:      InfoLevel,
		BatchSize:     100,
		BatchInterval: 5 * time.Second,
		Providers: []ProviderConfig{
			{
				Type: "stdout",
				Config: map[string]interface{}{
					"pretty": true,
				},
			},
		},
		TraceConfig: TraceConfig{
			Enabled:      true,
			SamplingRate: 1.0, // Trace semua request di default config
			ExporterType: "stdout",
			ExporterConfig: map[string]interface{}{
				"pretty": true,
			},
		},
	}
}

// CloudWatchConfig mengembalikan konfigurasi untuk CloudWatch provider
func CloudWatchConfig(region, accessKey, secretKey, logGroup, logStream string) ProviderConfig {
	return ProviderConfig{
		Type: "cloudwatch",
		Config: map[string]interface{}{
			"region":        region,
			"accessKey":     accessKey,
			"secretKey":     secretKey,
			"logGroup":      logGroup,
			"logStream":     logStream,
			"retentionDays": 30,
		},
	}
}

// OTLPExporterConfig mengembalikan konfigurasi untuk OTLP exporter (OpenTelemetry Protocol)
func OTLPExporterConfig(endpoint string, insecure bool, headers map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"endpoint": endpoint,
		"insecure": insecure,
		"headers":  headers,
	}
}

// FileProviderConfig mengembalikan konfigurasi untuk file logging provider
func FileProviderConfig(path string, maxSize, maxBackups, maxAge int, compress bool) ProviderConfig {
	return ProviderConfig{
		Type: "file",
		Config: map[string]interface{}{
			"path":       path,
			"maxSize":    maxSize,    // ukuran maksimum file dalam MB
			"maxBackups": maxBackups, // jumlah maksimum backup yang disimpan
			"maxAge":     maxAge,     // usia maksimum file backup dalam hari
			"compress":   compress,   // kompres file backup
		},
	}
}

// StdoutProviderConfig mengembalikan konfigurasi untuk output ke stdout
func StdoutProviderConfig(pretty bool) ProviderConfig {
	return ProviderConfig{
		Type: "stdout",
		Config: map[string]interface{}{
			"pretty": pretty,
		},
	}
}

// WithServiceName mengembalikan konfigurasi baru dengan nama service yang ditentukan
func (c Config) WithServiceName(name string) Config {
	c.ServiceName = name
	return c
}

// WithEnvironment mengembalikan konfigurasi baru dengan environment yang ditentukan
func (c Config) WithEnvironment(env string) Config {
	c.Environment = env
	return c
}

// WithLogLevel mengembalikan konfigurasi baru dengan log level yang ditentukan
func (c Config) WithLogLevel(level LogLevel) Config {
	c.LogLevel = level
	return c
}

// AddProvider menambahkan provider ke konfigurasi
func (c Config) AddProvider(provider ProviderConfig) Config {
	c.Providers = append(c.Providers, provider)
	return c
}

// WithTracing mengatur konfigurasi tracing
func (c Config) WithTracing(enabled bool, samplingRate float64, exporterType string, exporterConfig map[string]interface{}) Config {
	c.TraceConfig = TraceConfig{
		Enabled:        enabled,
		SamplingRate:   samplingRate,
		ExporterType:   exporterType,
		ExporterConfig: exporterConfig,
	}
	return c
}

// FromAppConfig membuat konfigurasi telemetri dari konfigurasi aplikasi
func FromAppConfig(appConfig interface{}) Config {
	// Implementasi ini akan tergantung pada struktur konfigurasi aplikasi
	// Dalam contoh ini, kita hanya mengembalikan konfigurasi default
	return DefaultConfig()
}
