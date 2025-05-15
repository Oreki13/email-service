package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config merupakan struktur utama untuk konfigurasi aplikasi
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Email    EmailConfig    `mapstructure:"email"`
	Auth     AuthConfig     `mapstructure:"auth"`
	AWS      AWSConfig      `mapstructure:"aws"`
	App      AppConfig      `mapstructure:"app"`
	Storage  StorageConfig  `mapstructure:"storage"`
}

// ServerConfig berisi konfigurasi server HTTP
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	HTTPPort        int           `mapstructure:"http_port"`
	Domain          string        `mapstructure:"domain"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig berisi konfigurasi koneksi database
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"name"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig berisi konfigurasi Redis untuk caching dan rate limiting
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// RabbitMQConfig berisi konfigurasi RabbitMQ untuk message queue
type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

// EmailConfig berisi konfigurasi untuk pengiriman email
type EmailConfig struct {
	DefaultProvider string       `mapstructure:"default_provider"`
	From            string       `mapstructure:"from"`
	ReplyTo         string       `mapstructure:"reply_to"`
	SMTP            SMTPConfig   `mapstructure:"smtp"`
	AWS             AWSSESConfig `mapstructure:"aws"`
}

// SMTPConfig berisi konfigurasi SMTP untuk pengiriman email
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	UseSSL   bool   `mapstructure:"use_ssl"`
}

// AWSSESConfig berisi konfigurasi AWS SES untuk pengiriman email
type AWSSESConfig struct {
	Region    string `mapstructure:"region"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// AuthConfig berisi konfigurasi untuk autentikasi
type AuthConfig struct {
	APIKeyHeaderName string `mapstructure:"api_key_header_name"`
	AdminAPIToken    string `mapstructure:"admin_api_token"`
}

// AWSConfig berisi konfigurasi AWS untuk CloudWatch
type AWSConfig struct {
	Region    string `mapstructure:"region"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	LogGroup  string `mapstructure:"log_group"`
}

// AppConfig berisi konfigurasi umum aplikasi
type AppConfig struct {
	Environment string `mapstructure:"environment"`
	LogLevel    string `mapstructure:"log_level"`
	ServiceName string `mapstructure:"service_name"`
}

// StorageConfig berisi konfigurasi untuk penyimpanan attachment
type StorageConfig struct {
	Provider     string          `mapstructure:"provider"` // local, s3, firebase
	BaseURL      string          `mapstructure:"base_url"`
	MaxSize      int64           `mapstructure:"max_size"`
	AllowedTypes []string        `mapstructure:"allowed_types"`
	Local        LocalStorage    `mapstructure:"local"`
	S3           S3Storage       `mapstructure:"s3"`
	Firebase     FirebaseStorage `mapstructure:"firebase"`
}

// LocalStorage berisi konfigurasi untuk penyimpanan lokal
type LocalStorage struct {
	Path    string `mapstructure:"path"`
	BaseURL string `mapstructure:"base_url"`
}

// S3Storage berisi konfigurasi untuk penyimpanan di S3
type S3Storage struct {
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	BaseURL   string `mapstructure:"base_url"`
	Prefix    string `mapstructure:"prefix"`
}

// FirebaseStorage berisi konfigurasi untuk penyimpanan di Firebase
type FirebaseStorage struct {
	ProjectID          string `mapstructure:"project_id"`
	Bucket             string `mapstructure:"bucket"`
	CredFile           string `mapstructure:"cred_file"`
	ServiceAccountJSON string `mapstructure:"service_account_json"`
	CredentialsFile    string `mapstructure:"credentials_file"`
	StoragePath        string `mapstructure:"storage_path"`
}

// LoadConfig membaca konfigurasi menggunakan viper dari file .env dan environment variables
func LoadConfig() (*Config, error) {
	// Secara default gunakan file .env
	return LoadConfigWithOptions(ConfigOptions{
		ConfigType: "env",
		ConfigFile: ".env", // Secara eksplisit tentukan bahwa kita menggunakan file .env
		ConfigPath: []string{"."},
	})
}

// ConfigOptions berisi opsi-opsi untuk memuat konfigurasi
type ConfigOptions struct {
	ConfigFile      string
	ConfigType      string
	ConfigName      string
	ConfigPath      []string
	EnvPrefix       string
	ServiceOverride string
}

// LoadConfigWithOptions membaca konfigurasi dari file .env dan environment variables dengan opsi khusus
func LoadConfigWithOptions(options ConfigOptions) (*Config, error) {
	// Set default values
	if options.ConfigFile == "" {
		options.ConfigFile = ".env"
	}

	// Baca file .env dengan godotenv terlebih dahulu
	err := godotenv.Load(options.ConfigFile)
	if err != nil {
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
		// Kita tetap melanjutkan, karena bisa saja menggunakan environment variables
	} else {
		fmt.Printf("Sukses memuat file .env: %s\n", options.ConfigFile)
	}

	// Inisialisasi Viper
	v := viper.New()

	// Set environment variable support
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaultValues(v)

	// Ambil konfigurasi dari environment variables yang sudah dimuat
	// Untuk database
	v.Set("database.driver", getEnvOrDefault("DATABASE_DRIVER", v.GetString("database.driver")))
	v.Set("database.host", getEnvOrDefault("DATABASE_HOST", v.GetString("database.host")))
	v.Set("database.port", getEnvIntOrDefault("DATABASE_PORT", v.GetInt("database.port")))
	v.Set("database.username", getEnvOrDefault("DATABASE_USERNAME", v.GetString("database.username")))
	v.Set("database.password", getEnvOrDefault("DATABASE_PASSWORD", v.GetString("database.password")))
	v.Set("database.name", getEnvOrDefault("DATABASE_NAME", v.GetString("database.name")))
	v.Set("database.max_open_conns", getEnvIntOrDefault("DATABASE_MAX_OPEN_CONNS", v.GetInt("database.max_open_conns")))
	v.Set("database.max_idle_conns", getEnvIntOrDefault("DATABASE_MAX_IDLE_CONNS", v.GetInt("database.max_idle_conns")))
	v.Set("database.conn_max_lifetime", getEnvOrDefault("DATABASE_CONN_MAX_LIFETIME", v.GetString("database.conn_max_lifetime")))

	// Untuk server
	v.Set("server.host", getEnvOrDefault("SERVER_HOST", v.GetString("server.host")))
	v.Set("server.http_port", getEnvIntOrDefault("SERVER_HTTP_PORT", v.GetInt("server.http_port")))
	v.Set("server.grpc_port", getEnvIntOrDefault("SERVER_GRPC_PORT", v.GetInt("server.grpc_port")))
	// Parse durasi untuk timeout
	readTimeout := getEnvDurationOrDefault("SERVER_READ_TIMEOUT", v.GetDuration("server.read_timeout"))
	writeTimeout := getEnvDurationOrDefault("SERVER_WRITE_TIMEOUT", v.GetDuration("server.write_timeout"))
	shutdownTimeout := getEnvDurationOrDefault("SERVER_SHUTDOWN_TIMEOUT", v.GetDuration("server.shutdown_timeout"))
	v.Set("server.read_timeout", readTimeout)
	v.Set("server.write_timeout", writeTimeout)
	v.Set("server.shutdown_timeout", shutdownTimeout)

	// Untuk Email
	v.Set("email.default_provider", getEnvOrDefault("EMAIL_DEFAULT_PROVIDER", v.GetString("email.default_provider")))
	v.Set("email.from", getEnvOrDefault("EMAIL_FROM", v.GetString("email.from")))
	v.Set("email.reply_to", getEnvOrDefault("EMAIL_REPLY_TO", v.GetString("email.reply_to")))
	v.Set("email.smtp.host", getEnvOrDefault("EMAIL_SMTP_HOST", v.GetString("email.smtp.host")))
	v.Set("email.smtp.port", getEnvIntOrDefault("EMAIL_SMTP_PORT", v.GetInt("email.smtp.port")))
	v.Set("email.smtp.username", getEnvOrDefault("EMAIL_SMTP_USERNAME", v.GetString("email.smtp.username")))
	v.Set("email.smtp.password", getEnvOrDefault("EMAIL_SMTP_PASSWORD", v.GetString("email.smtp.password")))
	v.Set("email.smtp.use_ssl", getEnvBoolOrDefault("EMAIL_SMTP_USE_SSL", v.GetBool("email.smtp.use_ssl")))

	// Untuk Auth
	v.Set("auth.api_key_header_name", getEnvOrDefault("AUTH_API_KEY_HEADER_NAME", v.GetString("auth.api_key_header_name")))
	v.Set("auth.admin_api_token", getEnvOrDefault("AUTH_ADMIN_API_TOKEN", v.GetString("auth.admin_api_token")))

	// Untuk App
	v.Set("app.environment", getEnvOrDefault("APP_ENVIRONMENT", v.GetString("app.environment")))
	v.Set("app.log_level", getEnvOrDefault("APP_LOG_LEVEL", v.GetString("app.log_level")))
	v.Set("app.service_name", getEnvOrDefault("APP_SERVICE_NAME", v.GetString("app.service_name")))

	// Untuk Storage
	v.Set("storage.provider", getEnvOrDefault("STORAGE_PROVIDER", v.GetString("storage.provider")))
	v.Set("storage.base_url", getEnvOrDefault("STORAGE_BASE_URL", v.GetString("storage.base_url")))
	v.Set("storage.max_size", getEnvInt64OrDefault("STORAGE_MAX_SIZE", v.GetInt64("storage.max_size")))
	v.Set("storage.allowed_types", getEnvStringSliceOrDefault("STORAGE_ALLOWED_TYPES", v.GetStringSlice("storage.allowed_types")))

	// Local storage
	v.Set("storage.local.path", getEnvOrDefault("STORAGE_LOCAL_PATH", v.GetString("storage.local.path")))
	v.Set("storage.local.base_url", getEnvOrDefault("STORAGE_LOCAL_BASE_URL", v.GetString("storage.local.base_url")))

	// S3 storage
	v.Set("storage.s3.region", getEnvOrDefault("STORAGE_S3_REGION", v.GetString("storage.s3.region")))
	v.Set("storage.s3.bucket", getEnvOrDefault("STORAGE_S3_BUCKET", v.GetString("storage.s3.bucket")))
	v.Set("storage.s3.access_key", getEnvOrDefault("STORAGE_S3_ACCESS_KEY", v.GetString("storage.s3.access_key")))
	v.Set("storage.s3.secret_key", getEnvOrDefault("STORAGE_S3_SECRET_KEY", v.GetString("storage.s3.secret_key")))
	v.Set("storage.s3.base_url", getEnvOrDefault("STORAGE_S3_BASE_URL", v.GetString("storage.s3.base_url")))
	v.Set("storage.s3.prefix", getEnvOrDefault("STORAGE_S3_PREFIX", v.GetString("storage.s3.prefix")))

	// Firebase storage
	v.Set("storage.firebase.project_id", getEnvOrDefault("STORAGE_FIREBASE_PROJECT_ID", v.GetString("storage.firebase.project_id")))
	v.Set("storage.firebase.bucket", getEnvOrDefault("STORAGE_FIREBASE_BUCKET", v.GetString("storage.firebase.bucket")))
	v.Set("storage.firebase.cred_file", getEnvOrDefault("STORAGE_FIREBASE_CRED_FILE", v.GetString("storage.firebase.cred_file")))
	v.Set("storage.firebase.service_account_json", getEnvOrDefault("STORAGE_FIREBASE_SERVICE_ACCOUNT_JSON", v.GetString("storage.firebase.service_account_json")))
	v.Set("storage.firebase.credentials_file", getEnvOrDefault("STORAGE_FIREBASE_CREDENTIALS_FILE", v.GetString("storage.firebase.credentials_file")))
	v.Set("storage.firebase.storage_path", getEnvOrDefault("STORAGE_FIREBASE_STORAGE_PATH", v.GetString("storage.firebase.storage_path")))

	// Override service name if provided
	if options.ServiceOverride != "" {
		v.Set("app.service_name", options.ServiceOverride)
	}

	// Map the viper config to our struct
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Debug config untuk kredensial database
	fmt.Printf("Database config: %s:%s@%s:%d/%s\n",
		config.Database.Username,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.Database)

	// Validate the config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// setDefaultValues menetapkan nilai default untuk konfigurasi
func setDefaultValues(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.http_port", 8080)
	v.SetDefault("server.grpc_port", 9090)
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)

	// Database defaults
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.username", "")
	v.SetDefault("database.password", "")
	v.SetDefault("database.name", "email_service")
	v.SetDefault("database.max_open_conns", 10)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "1h")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// RabbitMQ defaults
	v.SetDefault("rabbitmq.host", "localhost")
	v.SetDefault("rabbitmq.port", 5672)
	v.SetDefault("rabbitmq.username", "guest")
	v.SetDefault("rabbitmq.password", "guest")
	v.SetDefault("rabbitmq.vhost", "/")

	// Email defaults
	v.SetDefault("email.default_provider", "smtp")
	v.SetDefault("email.from", "noreply@example.com")
	v.SetDefault("email.reply_to", "")

	// SMTP defaults
	v.SetDefault("email.smtp.host", "")
	v.SetDefault("email.smtp.port", 587)
	v.SetDefault("email.smtp.username", "")
	v.SetDefault("email.smtp.password", "")
	v.SetDefault("email.smtp.use_ssl", false)

	// AWS SES defaults
	v.SetDefault("email.aws.region", "")
	v.SetDefault("email.aws.access_key", "")
	v.SetDefault("email.aws.secret_key", "")

	// Auth defaults
	v.SetDefault("auth.api_key_header_name", "X-API-Key")
	v.SetDefault("auth.admin_api_token", "admin_secret_token") // Default token yang aman harus diganti di production

	// AWS defaults
	v.SetDefault("aws.region", "")
	v.SetDefault("aws.access_key", "")
	v.SetDefault("aws.secret_key", "")
	v.SetDefault("aws.log_group", "/email-service/production")

	// App defaults
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("app.service_name", "email-service")

	// Storage defaults
	v.SetDefault("storage.provider", "local")
	v.SetDefault("storage.base_url", "")
	v.SetDefault("storage.max_size", int64(10485760)) // 10 MB
	v.SetDefault("storage.allowed_types", []string{"image/jpeg", "image/png", "application/pdf"})

	// Local storage defaults
	v.SetDefault("storage.local.path", "./storage/attachments")
	v.SetDefault("storage.local.base_url", "")

	// S3 storage defaults
	v.SetDefault("storage.s3.region", "")
	v.SetDefault("storage.s3.bucket", "")
	v.SetDefault("storage.s3.access_key", "")
	v.SetDefault("storage.s3.secret_key", "")
	v.SetDefault("storage.s3.base_url", "")
	v.SetDefault("storage.s3.prefix", "")

	// Firebase storage defaults
	v.SetDefault("storage.firebase.project_id", "")
	v.SetDefault("storage.firebase.bucket", "")
	v.SetDefault("storage.firebase.cred_file", "")
	v.SetDefault("storage.firebase.service_account_json", "")
	v.SetDefault("storage.firebase.credentials_file", "")
	v.SetDefault("storage.firebase.storage_path", "email-attachments")
}

// validateConfig memvalidasi konfigurasi
func validateConfig(cfg *Config) error {
	// Validasi konfigurasi database
	if cfg.Database.Username == "" || cfg.Database.Password == "" {
		return fmt.Errorf("database credentials are not configured")
	}

	// Validasi konfigurasi email provider
	if cfg.Email.DefaultProvider == "smtp" {
		if cfg.Email.SMTP.Host == "" || cfg.Email.SMTP.Username == "" || cfg.Email.SMTP.Password == "" {
			return fmt.Errorf("SMTP configuration is incomplete")
		}
	} else if cfg.Email.DefaultProvider == "ses" {
		if cfg.Email.AWS.Region == "" || cfg.Email.AWS.AccessKey == "" || cfg.Email.AWS.SecretKey == "" {
			return fmt.Errorf("AWS SES configuration is incomplete")
		}
	}

	// Validasi konfigurasi storage berdasarkan provider
	switch cfg.Storage.Provider {
	case "s3":
		if cfg.Storage.S3.Region == "" || cfg.Storage.S3.Bucket == "" {
			return fmt.Errorf("S3 storage configuration is incomplete")
		}
	case "firebase":
		if cfg.Storage.Firebase.ProjectID == "" || cfg.Storage.Firebase.Bucket == "" {
			return fmt.Errorf("Firebase storage configuration is incomplete")
		}
	case "local":
		// Untuk local, kita hanya perlu memastikan path ada dan bisa ditulis
		// Ini akan ditangani saat inisialisasi provider
	default:
		return fmt.Errorf("unsupported storage provider: %s", cfg.Storage.Provider)
	}

	return nil
}

// LoadConfigFromEnv loads configuration from .env files for backward compatibility
func LoadConfigFromEnv() (*Config, error) {
	return LoadConfigWithOptions(ConfigOptions{
		ConfigType: "env",
		ConfigPath: []string{"."},
	})
}

// GetViperInstance returns a new viper instance with the email service configuration
func GetViperInstance(options ConfigOptions) (*viper.Viper, error) {
	// Set default values
	if options.ConfigFile == "" {
		options.ConfigFile = ".env"
	}

	// Inisialisasi Viper
	v := viper.New()

	// Konfigurasi untuk membaca .env
	v.SetConfigFile(options.ConfigFile)
	v.SetConfigType("env")

	// Set environment variable support
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaultValues(v)

	// Baca konfigurasi dari file .env
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return v, nil
}

// Helper functions untuk membaca dari environment variables
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvStringSliceOrDefault(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}

// getEnvDurationOrDefault mengambil nilai durasi dari environment variable
// dan mengembalikan nilai default jika tidak ditemukan
func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
