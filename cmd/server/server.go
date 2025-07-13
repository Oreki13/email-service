package server

import (
	"context"
	"email-service/internal/config"
	"email-service/internal/database"
	"email-service/internal/delivery"
	"email-service/internal/domain"
	"email-service/internal/handler"
	"email-service/internal/middleware"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"email-service/internal/service"
	"email-service/internal/webui"
	"email-service/pkg/cloudwatch"
	"email-service/pkg/telemetry"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/redis/go-redis/v9"
)

// RunServer menjalankan HTTP server
func RunServer(cfg *config.Config) {
	// Initialize database connection
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis connection
	redisClient, err := database.NewRedisClient(database.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Printf("WARNING: Failed to initialize Redis: %v. Rate limiting will be disabled.", err)
	} else {
		defer redisClient.Close()
		log.Println("Redis connected successfully. Rate limiting enabled.")
	}

	// Initialize dependencies
	emailService, mqAdapter, apiKeyRepo, templatedEmailService, dashboardService, templateRepo, logger, err := initializeServices(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Initialize Fiber app
	app := initializeFiberApp(cfg, emailService, apiKeyRepo, templatedEmailService, dashboardService, templateRepo, logger, redisClient, db)

	// Initialize email consumer jika RabbitMQ tersedia
	var consumer *queue.EmailConsumer
	if mqAdapter != nil {
		// Konfigurasi RabbitMQ
		rmqConfig := queue.RabbitMQConfig{
			Host:     cfg.RabbitMQ.Host,
			Port:     cfg.RabbitMQ.Port,
			Username: cfg.RabbitMQ.Username,
			Password: cfg.RabbitMQ.Password,
			VHost:    cfg.RabbitMQ.VHost,
		}

		// Buat consumer
		consumer, err = queue.NewEmailConsumer(rmqConfig, logger, emailService)
		if err != nil {
			logger.Error(context.Background(), "Failed to initialize email consumer", telemetry.Fields{
				"error": err.Error(),
			})
			logger.Warn(context.Background(), "Emails will only be processed by scheduled tasks", nil)
		} else {
			logger.Info(context.Background(), "Email consumer initialized successfully", nil)
		}
	}

	// Start servers
	startServers(app, consumer, cfg)
}

// initializeServices menginialisasi semua service yang diperlukan
func initializeServices(cfg *config.Config, db *database.PostgresDB) (domain.EmailService, service.EmailQueueAdapter, domain.APIKeyRepository, domain.TemplatedEmailService, domain.DashboardService, domain.TemplateRepository, telemetry.Logger, error) {
	// Initialize repositories
	emailRepo := repository.NewSQLEmailRepository(db.GetDB())
	templateRepo := repository.NewSQLTemplateRepository(db.GetDB())
	apiKeyRepo := repository.NewSQLAPIKeyRepository(db.GetDB())
	trackingRepo := repository.NewSQLEmailTrackingRepository(db.GetDB())
	dashboardRepo := repository.NewSQLDashboardRepository(db.GetDB())

	// Initialize telemetry/logger
	var logger telemetry.Logger
	var err error

	// Gunakan logger CloudWatch jika konfigurasi AWS tersedia
	if cfg.AWS.Region != "" && cfg.AWS.LogGroup != "" {
		cwConfig := cloudwatch.CloudWatchConfig{
			Region:        cfg.AWS.Region,
			AccessKey:     cfg.AWS.AccessKey,
			SecretKey:     cfg.AWS.SecretKey,
			LogGroup:      cfg.AWS.LogGroup,
			LogStream:     fmt.Sprintf("%s-%s-%d", cfg.App.ServiceName, cfg.App.Environment, os.Getpid()),
			RetentionDays: 30,
			Environment:   cfg.App.Environment,
		}

		logger, err = cloudwatch.NewCloudWatchTelemetryLogger(cwConfig)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, err
		}
	} else {
		// Fallback ke logger default jika tidak ada konfigurasi AWS
		var err error
		logger, err = telemetry.NewDefaultLogger()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to initialize default logger: %w", err)
		}
	}

	// Initialize email delivery adapters
	var emailDelivery domain.EmailDelivery
	switch cfg.Email.DefaultProvider {
	case string(domain.ProviderSMTP):
		smtpConfig := delivery.SMTPConfig{
			Host:     cfg.Email.SMTP.Host,
			Port:     cfg.Email.SMTP.Port,
			Username: cfg.Email.SMTP.Username,
			Password: cfg.Email.SMTP.Password,
			UseSSL:   cfg.Email.SMTP.UseSSL,
		}
		emailDelivery = delivery.NewSMTPAdapter(smtpConfig, logger)
	case string(domain.ProviderSES):
		sesConfig := delivery.SESConfig{
			Region:    cfg.Email.AWS.Region,
			AccessKey: cfg.Email.AWS.AccessKey,
			SecretKey: cfg.Email.AWS.SecretKey,
		}
		emailDelivery = delivery.NewSESAdapter(sesConfig, logger)
	default:
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("unsupported email provider: %s", cfg.Email.DefaultProvider)
	}

	// Initialize email queue adapter
	queueConfig := queue.RabbitMQConfig{
		Host:     cfg.RabbitMQ.Host,
		Port:     cfg.RabbitMQ.Port,
		Username: cfg.RabbitMQ.Username,
		Password: cfg.RabbitMQ.Password,
		VHost:    cfg.RabbitMQ.VHost,
	}
	queueAdapter, err := queue.NewRabbitMQAdapter(queueConfig, logger)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	// Initialize storage adapter untuk attachment
	storageAdapter, err := service.NewStorageAdapter(cfg, logger)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize storage adapter", telemetry.Fields{
			"error": err.Error(),
		})
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to initialize storage adapter: %w", err)
	}

	logger.Info(context.Background(), "Storage adapter initialized successfully", telemetry.Fields{
		"provider": cfg.Storage.Provider,
	})

	// Initialize email service with storage adapter
	emailSvc := service.NewEmailService(
		emailRepo,
		templateRepo,
		emailDelivery,
		queueAdapter,
		storageAdapter,
		cfg,
		logger,
		trackingRepo,
	)

	// Initialize templated email service
	templatedEmailSvc := service.NewTemplatedEmailService(
		emailSvc,
		templateRepo,
		logger,
	)

	// Initialize dashboard service
	dashboardSvc := service.NewDashboardService(
		dashboardRepo,
		logger,
	)

	// Inisialisasi template default jika diperlukan
	// go func() {
	// 	// Gunakan context background karena ini dilakukan di background
	// 	ctx := context.Background()
	// 	if serviceImpl, ok := templatedEmailSvc.(*service.TemplatedEmailServiceImpl); ok {
	// 		if err := serviceImpl.InitializeTemplates(ctx); err != nil {
	// 			logger.Error(ctx, "Gagal menginisialisasi template default", telemetry.Fields{
	// 				"error": err.Error(),
	// 			})
	// 		} else {
	// 			logger.Info(ctx, "Template default berhasil diinisialisasi", nil)
	// 		}
	// 	}
	// }()

	return emailSvc, queueAdapter, apiKeyRepo, templatedEmailSvc, dashboardSvc, templateRepo, logger, nil
}

// initializeFiberApp menginialisasi aplikasi Fiber
func initializeFiberApp(cfg *config.Config, emailService domain.EmailService, apiKeyRepo domain.APIKeyRepository, templatedEmailService domain.TemplatedEmailService, dashboardService domain.DashboardService, templateRepo domain.TemplateRepository, logger telemetry.Logger, redisClient *database.RedisClient, db *database.PostgresDB) *fiber.App { // Initialize template engine
	engine := html.New("./web/templates", ".html")

	// Enable template reloading and debug mode for development environment
	if cfg.App.Environment == "development" {
		engine.Reload(true)
		engine.Debug(true)
	}

	// Add custom template functions
	engine.AddFunc("timeFormat", func(t time.Time, format string) string {
		return t.Format(format)
	})

	// Add split function for template use
	engine.AddFunc("split", func(s, sep string) []string {
		if s == "" {
			return []string{}
		}
		return strings.Split(s, sep)
	})

	// Add other useful template functions
	engine.AddFunc("join", func(elems []string, sep string) string {
		return strings.Join(elems, sep)
	})

	engine.AddFunc("contains", func(s, substr string) bool {
		return strings.Contains(s, substr)
	})

	engine.AddFunc("toLower", func(s string) string {
		return strings.ToLower(s)
	})

	engine.AddFunc("toUpper", func(s string) string {
		return strings.ToUpper(s)
	})

	// Initialize Fiber app with timeouts and template engine
	app := fiber.New(fiber.Config{
		AppName:               "Email Microservice",
		ErrorHandler:          handler.ErrorHandler,
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		DisableStartupMessage: false,
		IdleTimeout:           120 * time.Second, // Waktu maksimum koneksi idle
		ReadBufferSize:        4096,              // Ukuran buffer baca
		WriteBufferSize:       4096,              // Ukuran buffer tulis
		Views:                 engine,            // Set template engine
	})

	// Use middleware
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLogger(logger))

	// Setup rate limiter jika Redis tersedia
	if redisClient != nil {
		// Konfigurasi global rate limiter
		rateLimiterConfig := middleware.NewRateLimiterConfig(redisClient.GetClient(), logger)
		rateLimiterConfig.Max = 1000             // 1000 request per menit secara global
		rateLimiterConfig.Duration = time.Minute // Window 1 menit

		// Gunakan rate limiter untuk semua route
		app.Use(middleware.RateLimiter(rateLimiterConfig))
	}

	// Setup API routes
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Setup authentication middleware untuk API routes
	v1.Use(middleware.AuthMiddleware(apiKeyRepo, cfg.Auth.APIKeyHeaderName, logger))

	// Setup rate limiter khusus untuk endpoint email jika Redis tersedia
	if redisClient != nil {
		// Rate limiter yang lebih ketat untuk endpoint pengiriman email
		emailRateLimiterConfig := middleware.NewRateLimiterConfig(redisClient.GetClient(), logger)
		emailRateLimiterConfig.Max = 100              // 100 email per menit
		emailRateLimiterConfig.Duration = time.Minute // Window 1 menit
		emailRateLimiterConfig.Key = func(c *fiber.Ctx) string {
			// Gunakan kombinasi IP dan path untuk rate limiting
			return fmt.Sprintf("ratelimit:email:%s", c.IP())
		}

		// Terapkan rate limiter khusus untuk endpoint email
		v1.Use("/emails", middleware.RateLimiter(emailRateLimiterConfig))
	}

	// Log konfigurasi server timeout
	logger.Info(context.Background(), "Server timeout configuration", telemetry.Fields{
		"read_timeout":     cfg.Server.ReadTimeout.String(),
		"write_timeout":    cfg.Server.WriteTimeout.String(),
		"shutdown_timeout": cfg.Server.ShutdownTimeout.String(),
		"idle_timeout":     "120s",
	})

	// Register API routes dengan emailService dan apiKeyRepo
	handler.RegisterRoutes(v1, cfg, emailService, apiKeyRepo, logger)

	// Register Template-specific email routes
	templateEmailHandler := handler.NewTemplateEmailHandler(templatedEmailService, logger)
	templateEmailHandler.RegisterRoutes(v1.Group("/emails"))

	// Register Email Tracking routes
	trackingHandler := handler.NewEmailTrackingHandler(emailService, logger)
	trackingHandler.RegisterRoutes(v1)

	// Health check endpoints
	var redisClientInstance *redis.Client
	if redisClient != nil {
		redisClientInstance = redisClient.GetClient()
	}
	healthHandler := handler.NewHealthHandler(db.GetDB(), redisClientInstance, logger)
	healthHandler.RegisterRoutes(app)

	// Register Web UI routes
	webuiHandler := webui.NewWebUIHandler(cfg, templatedEmailService, templateRepo, dashboardService, logger)
	webuiHandler.RegisterRoutes(app)

	return app
}

// startServers menjalankan server HTTP
func startServers(app *fiber.App, consumer *queue.EmailConsumer, cfg *config.Config) {
	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	// Start HTTP server in a goroutine
	go func() {
		serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort)
		log.Printf("HTTP server starting on %s with read timeout: %s, write timeout: %s, shutdown timeout: %s",
			serverAddr,
			cfg.Server.ReadTimeout,
			cfg.Server.WriteTimeout,
			cfg.Server.ShutdownTimeout)
		if err := app.Listen(serverAddr); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Start email consumer jika tersedia
	if consumer != nil {
		go func() {
			log.Println("Starting email consumer...")
			if err := consumer.Start(); err != nil {
				log.Printf("Email consumer stopped with error: %v", err)
			}
		}()
	}

	// Tunggu sinyal shutdown
	<-quit
	log.Println("Shutting down servers...")

	// Hentikan consumer jika ada
	if consumer != nil {
		log.Println("Stopping email consumer...")
		if err := consumer.Stop(); err != nil {
			log.Printf("Error stopping consumer: %v", err)
		}
	}

	// Shutdown gracefully
	shutdownTimeout := 30 * time.Second
	if cfg.Server.ShutdownTimeout > 0 {
		shutdownTimeout = cfg.Server.ShutdownTimeout
	}

	// Buat context dengan timeout untuk shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	log.Println("Stopping HTTP server...")
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Servers gracefully stopped")
}
