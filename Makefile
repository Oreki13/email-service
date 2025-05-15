# Makefile untuk Email Microservice

# Variabel
APP_NAME := email-service
BUILD_DIR := build
CONFIG_FILE := .env
MIGRATIONS_DIR := migrations
MAIN_FILE := main.go

# Deteksi OS untuk perintah yang berbeda di Linux/macOS dan Windows
ifeq ($(OS),Windows_NT)
	RM := del /Q
	MKDIR := mkdir
	COPY := copy
else
	RM := rm -f
	MKDIR := mkdir -p
	COPY := cp
endif

# Target utama
.PHONY: all
all: init build

# Inisialisasi project setelah clone
.PHONY: init
init: setup-env setup-deps migrate

# Setup environment
.PHONY: setup-env
setup-env:
	@echo "Mengatur environment untuk project..."
	@test -f $(CONFIG_FILE) || (echo "Membuat file .env dari template..." && \
		echo "# Konfigurasi Email Service\n\
		\n\
		# Server\n\
		EMAILSVC_SERVER_PORT=8080\n\
		EMAILSVC_ENV=development\n\
		\n\
		# Database\n\
		EMAILSVC_DB_HOST=localhost\n\
		EMAILSVC_DB_PORT=3306\n\
		EMAILSVC_DB_USER=emailsvc\n\
		EMAILSVC_DB_PASSWORD=password\n\
		EMAILSVC_DB_NAME=emailsvc\n\
		\n\
		# Redis\n\
		EMAILSVC_REDIS_HOST=localhost\n\
		EMAILSVC_REDIS_PORT=6379\n\
		EMAILSVC_REDIS_PASSWORD=\n\
		\n\
		# RabbitMQ\n\
		EMAILSVC_RABBITMQ_HOST=localhost\n\
		EMAILSVC_RABBITMQ_PORT=5672\n\
		EMAILSVC_RABBITMQ_USER=guest\n\
		EMAILSVC_RABBITMQ_PASSWORD=guest\n\
		\n\
		# SMTP\n\
		EMAILSVC_SMTP_HOST=smtp.example.com\n\
		EMAILSVC_SMTP_PORT=587\n\
		EMAILSVC_SMTP_USERNAME=user@example.com\n\
		EMAILSVC_SMTP_PASSWORD=password\n\
		EMAILSVC_SMTP_FROM=noreply@example.com\n\
		\n\
		# AWS SES\n\
		EMAILSVC_AWS_REGION=ap-southeast-1\n\
		EMAILSVC_AWS_ACCESS_KEY=\n\
		EMAILSVC_AWS_SECRET_KEY=\n\
		\n\
		# Storage\n\
		EMAILSVC_STORAGE_TYPE=local\n\
		EMAILSVC_STORAGE_PATH=./storage/attachments\n\
		\n\
		# AWS S3\n\
		EMAILSVC_S3_BUCKET=\n\
		\n\
		# Firebase\n\
		EMAILSVC_FIREBASE_PROJECT_ID=\n\
		EMAILSVC_FIREBASE_BUCKET=\n\
		EMAILSVC_FIREBASE_CREDENTIALS=\n\
		\n\
		# Logging\n\
		EMAILSVC_LOG_LEVEL=info\n\
		\n\
		# API\n\
		EMAILSVC_API_SECRET=rahasia123\n\
		" > $(CONFIG_FILE))
	@echo "File .env telah dibuat."
	@$(MKDIR) storage/attachments 2>/dev/null || true
	@echo "Folder storage untuk attachment telah dibuat."

# Setup dependencies
.PHONY: setup-deps
setup-deps:
	@echo "Menginstall dependencies dari go.mod..."
	@go mod download
	@go mod tidy
	@echo "Dependencies telah diinstal."
	@echo "Menginstall development tools..."
	@go install github.com/golang/mock/mockgen@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development tools telah diinstal."

# Jalankan migrasi database
.PHONY: migrate
migrate:
	@echo "Memulai migrasi database..."
	@go run main.go migrate up
	@echo "Migrasi database selesai."

# Jalankan rollback migrasi database
.PHONY: migrate-down
migrate-down:
	@echo "Memulai rollback migrasi database..."
	@go run main.go migrate down
	@echo "Rollback migrasi database selesai."

# Build aplikasi
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@$(MKDIR) $(BUILD_DIR) 2>/dev/null || true
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build selesai. Binary tersedia di $(BUILD_DIR)/$(APP_NAME)"

# Run aplikasi
.PHONY: run
run:
	@echo "Menjalankan $(APP_NAME)..."
	@go run $(MAIN_FILE) server

# Jalankan dalam mode development dengan hot reload
.PHONY: dev
dev:
	@echo "Mengecek air (hot reload)..."
	@which air > /dev/null || (echo "Installing air for hot reload..." && go install github.com/air-verse/air@latest)
	@echo "Menjalankan dengan hot reload..."
	@air -c .air.toml

# Setup untuk local development
.PHONY: dev-setup
dev-setup: setup-env setup-deps
	@echo "Membuat file konfigurasi air untuk hot reload..."
	@test -f .air.toml || (echo 'Creating .air.toml for hot reload...' && \
		echo '# .air.toml\n\
		root = "."\n\
		tmp_dir = "tmp"\n\
		\n\
		[build]\n\
		cmd = "go build -o ./tmp/main ."\n\
		bin = "tmp/main server"\n\
		include_ext = ["go", "tpl", "tmpl", "html"]\n\
		exclude_dir = ["assets", "tmp", "vendor", "build", "storage"]\n\
		delay = 1000\n\
		\n\
		[log]\n\
		time = true\n\
		\n\
		[misc]\n\
		clean_on_exit = true\n\
		\n\
		[screen]\n\
		clear_on_rebuild = true\n\
		keep_scroll = true\n\
		' > .air.toml)
	@echo "Menginstal Air untuk hot reload..."
	@go install github.com/air-verse/air@latest
	@echo "Air untuk hot reload telah diinstal. Gunakan 'make dev' untuk menjalankan dengan hot reload."

# Setup Docker development environment
.PHONY: docker-setup
docker-setup:
	@echo "Menyiapkan lingkungan Docker untuk development..."
	@test -f docker-compose.yml || (echo "Membuat file docker-compose.yml..." && \
		echo "version: '3.8'\n\
		\n\
		services:\n\
		  mariadb:\n\
		    image: mariadb:10.6\n\
		    container_name: emailsvc-mariadb\n\
		    ports:\n\
		      - \"3306:3306\"\n\
		    environment:\n\
		      MYSQL_ROOT_PASSWORD: rootpassword\n\
		      MYSQL_DATABASE: emailsvc\n\
		      MYSQL_USER: emailsvc\n\
		      MYSQL_PASSWORD: password\n\
		    volumes:\n\
		      - emailsvc-mariadb-data:/var/lib/mysql\n\
		    networks:\n\
		      - emailsvc-network\n\
		    healthcheck:\n\
		      test: [\"CMD\", \"mysqladmin\", \"ping\", \"-h\", \"localhost\", \"-u\", \"root\", \"-prootpassword\"]\n\
		      interval: 5s\n\
		      timeout: 5s\n\
		      retries: 5\n\
		\n\
		  redis:\n\
		    image: redis:6.2-alpine\n\
		    container_name: emailsvc-redis\n\
		    ports:\n\
		      - \"6379:6379\"\n\
		    volumes:\n\
		      - emailsvc-redis-data:/data\n\
		    networks:\n\
		      - emailsvc-network\n\
		    healthcheck:\n\
		      test: [\"CMD\", \"redis-cli\", \"ping\"]\n\
		      interval: 5s\n\
		      timeout: 5s\n\
		      retries: 5\n\
		\n\
		  rabbitmq:\n\
		    image: rabbitmq:3.9-management-alpine\n\
		    container_name: emailsvc-rabbitmq\n\
		    ports:\n\
		      - \"5672:5672\"\n\
		      - \"15672:15672\"\n\
		    environment:\n\
		      RABBITMQ_DEFAULT_USER: guest\n\
		      RABBITMQ_DEFAULT_PASS: guest\n\
		    volumes:\n\
		      - emailsvc-rabbitmq-data:/var/lib/rabbitmq\n\
		    networks:\n\
		      - emailsvc-network\n\
		    healthcheck:\n\
		      test: [\"CMD\", \"rabbitmq-diagnostics\", \"-q\", \"ping\"]\n\
		      interval: 5s\n\
		      timeout: 5s\n\
		      retries: 5\n\
		\n\
		networks:\n\
		  emailsvc-network:\n\
		    driver: bridge\n\
		\n\
		volumes:\n\
		  emailsvc-mariadb-data:\n\
		  emailsvc-redis-data:\n\
		  emailsvc-rabbitmq-data:\n\
		" > docker-compose.yml)
	@echo "Docker compose telah dibuat. Jalankan dengan 'docker-compose up -d'"

# Run tests
.PHONY: test
test:
	@echo "Menjalankan unit tests..."
	@go test -v ./internal/...
	@echo "Unit tests selesai."

# Run tests dengan coverage
.PHONY: test-coverage
test-coverage:
	@echo "Menjalankan tests dengan coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./internal/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Laporan coverage dihasilkan di coverage.html"

# Clean project files
.PHONY: clean
clean:
	@echo "Membersihkan build artifacts..."
	@$(RM) $(BUILD_DIR)/$(APP_NAME)
	@$(RM) coverage.out coverage.html
	@echo "Pembersihan selesai."

# Generate mock untuk testing
.PHONY: mock
mock:
	@echo "Generating mocks..."
	@mockgen -source=internal/service/email_service.go -destination=internal/service/mock/mock_email_service.go -package=mock
	@mockgen -source=internal/repository/email_repository.go -destination=internal/repository/mock/mock_email_repository.go -package=mock
	@mockgen -source=internal/delivery/ses_adapter.go -destination=internal/delivery/mock/mock_ses_adapter.go -package=mock
	@mockgen -source=internal/delivery/smtp_adapter.go -destination=internal/delivery/mock/mock_smtp_adapter.go -package=mock
	@echo "Mocks generated."

# Help
.PHONY: help
help:
	@echo "Email Service Makefile"
	@echo ""
	@echo "Penggunaan:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  init              - Inisialisasi project setelah clone (setup env, deps, migrasi)"
	@echo "  setup-env         - Setup file environment"
	@echo "  setup-deps        - Download dan install dependencies"
	@echo "  migrate           - Jalankan migrasi database"
	@echo "  migrate-down      - Rollback migrasi database"
	@echo "  build             - Build aplikasi"
	@echo "  run               - Jalankan aplikasi"
	@echo "  dev               - Jalankan dalam mode development dengan hot reload"
	@echo "  dev-setup         - Setup untuk local development dengan hot reload"
	@echo "  docker-setup      - Setup Docker untuk development environment"
	@echo "  test              - Jalankan unit tests"
	@echo "  test-coverage     - Jalankan tests dengan coverage"
	@echo "  clean             - Bersihkan build artifacts"
	@echo "  mock              - Generate mock untuk testing"
	@echo "  help              - Tampilkan bantuan ini"
