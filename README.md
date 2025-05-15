# Email Microservice

Microservice internal untuk pengiriman email dengan menggunakan Golang. Service ini bertanggung jawab untuk menangani permintaan pengiriman email dari service lain dalam ekosistem aplikasi. Service menerima request, memvalidasi data, menyiapkan template email, dan mengirimkan email menggunakan provider yang dikonfigurasi.

## Fitur

- Pengiriman email melalui REST API
- Mendukung email dengan template (HTML dan text)
- Antrian email menggunakan RabbitMQ
- Rate limiting untuk mencegah abuse
- Retry mechanism untuk email yang gagal terkirim
- Tracking status pengiriman email
- Support untuk attachment
- Multiple email provider (SMTP, AWS SES)
- Penyimpanan attachment di lokal, S3, atau Firebase Storage

## Arsitektur

Proyek ini dibangun menggunakan pendekatan **Clean Architecture** dengan pemisahan layer:

- **Domain Layer**: Entitas bisnis dan business rules
- **Use Case Layer**: Implementasi service untuk pengiriman email
- **Interface Adapter Layer**: Controllers, repositories, adapters
- **Framework Layer**: Framework dan library eksternal

## Teknologi

- **Bahasa**: Go 1.23+
- **Framework**: Fiber untuk REST API
- **Message Queue**: RabbitMQ untuk antrian
- **Database**: PostgreSQL/MySQL untuk menyimpan template dan history
- **Caching**: Redis untuk rate limiting dan caching template
- **Logging**: AWS CloudWatch / OpenTelemetry
- **Config**: Viper untuk konfigurasi
- **Email Library**: Gomail / AWS SES SDK

## Persyaratan

Untuk menjalankan aplikasi ini, Anda membutuhkan:

1. **Go 1.23+** - [Download Go](https://golang.org/dl/)
2. **Docker & Docker Compose** (opsional, untuk lingkungan development)
3. **Redis** - Untuk caching dan rate limiting
4. **RabbitMQ** - Untuk message queue
5. **MariaDB/MySQL/PostgreSQL** - Untuk penyimpanan data
6. **AWS Account** (opsional, jika menggunakan SES dan S3)
7. **Firebase Account** (opsional, jika menggunakan Firebase Storage)

## Instalasi

### Metode 1: Manual Setup

1. Clone repository ini

   ```bash
   git clone https://github.com/username/email-service.git
   cd email-service
   ```

2. Inisialisasi project (setup .env, dependencies, dan migrasi)
   ```bash
   make init
   ```
3. Build aplikasi
   ```bash
   make build
   ```

### Metode 2: Docker Setup

1. Clone repository

   ```bash
   git clone https://github.com/username/email-service.git
   cd email-service
   ```

2. Setup Docker environment

   ```bash
   make docker-setup
   ```

3. Jalankan service dependency dengan Docker Compose
   ```bash
   docker-compose up -d
   ```

## Konfigurasi

Semua konfigurasi berada dalam file `.env` di root project. File ini akan otomatis dibuat saat menjalankan `make init` atau `make setup-env`. Berikut beberapa konfigurasi penting:

```
# Server
EMAILSVC_SERVER_PORT=8080
EMAILSVC_ENV=development

# Database
EMAILSVC_DB_HOST=localhost
EMAILSVC_DB_PORT=3306
EMAILSVC_DB_USER=emailsvc
EMAILSVC_DB_PASSWORD=password
EMAILSVC_DB_NAME=emailsvc

# Redis
EMAILSVC_REDIS_HOST=localhost
EMAILSVC_REDIS_PORT=6379

# RabbitMQ
EMAILSVC_RABBITMQ_HOST=localhost
EMAILSVC_RABBITMQ_PORT=5672
EMAILSVC_RABBITMQ_USER=guest
EMAILSVC_RABBITMQ_PASSWORD=guest

# SMTP
EMAILSVC_SMTP_HOST=smtp.example.com
EMAILSVC_SMTP_PORT=587
EMAILSVC_SMTP_USERNAME=user@example.com
EMAILSVC_SMTP_PASSWORD=password
EMAILSVC_SMTP_FROM=noreply@example.com

# AWS SES
EMAILSVC_AWS_REGION=ap-southeast-1
EMAILSVC_AWS_ACCESS_KEY=
EMAILSVC_AWS_SECRET_KEY=

# Storage
EMAILSVC_STORAGE_TYPE=local  # local, s3, firebase
EMAILSVC_STORAGE_PATH=./storage/attachments
```

## Menjalankan Aplikasi

### Development Mode dengan Hot Reload

```bash
make dev
```

### Production Mode

```bash
make run
```

atau jalankan binary setelah build:

```bash
./build/email-service server
```

## API Endpoints

Dokumentasi lengkap API dapat dilihat pada `/docs` setelah menjalankan server.

Contoh endpoint utama:

- `POST /api/v1/emails` - Mengirim email
- `GET /api/v1/emails/:id` - Mendapatkan detail email
- `GET /api/v1/emails/:id/tracking` - Mendapatkan status tracking email
- `POST /api/v1/templates` - Membuat template email
- `GET /api/v1/templates` - Mendapatkan daftar template

## Perintah Makefile

Semua operasi project dapat dilakukan melalui Makefile:

- `make init` - Inisialisasi project setelah clone (setup env, deps, migrasi)
- `make setup-env` - Setup file environment
- `make setup-deps` - Download dan install dependencies
- `make migrate` - Jalankan migrasi database
- `make migrate-down` - Rollback migrasi database
- `make build` - Build aplikasi
- `make run` - Jalankan aplikasi
- `make dev` - Jalankan dalam mode development dengan hot reload
- `make dev-setup` - Setup untuk local development dengan hot reload
- `make docker-setup` - Setup Docker untuk development environment
- `make test` - Jalankan unit tests
- `make test-coverage` - Jalankan tests dengan coverage
- `make clean` - Bersihkan build artifacts
- `make mock` - Generate mock untuk testing
- `make help` - Tampilkan bantuan

## Migrasi Database

Migrasi database menggunakan perintah CLI:

```bash
# Menjalankan semua migrasi
go run main.go migrate up

# Rollback semua migrasi
go run main.go migrate down

# Migrasi ke versi tertentu
go run main.go migrate goto [versi]
```

## Testing

### Unit Testing

```bash
make test
```

### Test dengan Coverage

```bash
make test-coverage
```

## Logging

Service menggunakan AWS CloudWatch atau OpenTelemetry untuk logging terpusat. Semua request dan response di-log dengan trace ID untuk memudahkan debugging.

## Lisensi

[MIT License](LICENSE)
