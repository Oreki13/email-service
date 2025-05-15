# Instruksi untuk GitHub Copilot

Gunakan Bahasa Indonesia

## Proyek: Email Microservice dengan Golang

Saya sedang mengembangkan microservice internal untuk pengiriman email dengan menggunakan Golang. Berikut adalah spesifikasi dan panduan untuk membantu dalam pembuatan kode.

### Deskripsi Proyek

Microservice ini bertanggung jawab untuk menangani permintaan pengiriman email dari service lain dalam ekosistem aplikasi kami. Service akan menerima request, memvalidasi data, menyiapkan template email, dan mengirimkan email menggunakan provider yang dikonfigurasi.

### Arsitektur

Menggunakan pendekatan **Clean Architecture** dengan pemisahan layer berikut:

- **Domain Layer**: Entitas bisnis dan business rules
- **Use Case Layer**: Implementasi service/use case untuk pengiriman email
- **Interface Adapter Layer**: Controllers, repositories, external services adapters
- **Framework Layer**: Frameworks eksternal, libraries, dan tools

### Struktur Folder

Struktur folder dapat berubah sesuai kebutuhan tetapi tetap mengikuti prinsip clean architecture. Berikut adalah struktur folder yang diusulkan:

```
/email-service
├── cmd
│   └── server
│       └── main.go              # Entry point aplikasi
├── internal
|   |-- dto                      # Data Transfer Objects
│   ├── config                   # Konfigurasi aplikasi
│   │   └── config.go
│   ├── domain                   # Model domain bisnis
│   │   ├── email.go             # Entitas email
│   │   └── errors.go            # Domain errors
│   ├── repository               # Data access layer
│   │   └── template_repository.go # Repository untuk template email
│   ├── handler                  # API handlers dengan Fiber
│   │   └── email_handler.go
│   ├── middleware               # Fiber middleware
│   │   ├── auth.go
│   │   └── logging.go
│   ├── service                  # Business logic
│   │   └── email_service.go
│   └── delivery                 # Email delivery adapters
│       ├── smtp_adapter.go
│       └── ses_adapter.go
├── pkg                          # Public libraries
│   ├── cloudwatch               # AWS CloudWatch logging
│   ├── validator                # Validation utilities
│   └── common                   # Shared utilities
├── migrations                   # MariaDB migrations
├── scripts                      # Build/deployment scripts
├── deployments                  # Docker, Kubernetes manifests
│   ├── Dockerfile
│   └── k8s
├── docs                         # API documentation
├── test                         # Integration tests
│   └── integration
├── go.mod                       # Go modules
├── go.sum                       # Dependencies checksum
└── Makefile                     # Build automation
```

### Principles yang Diimplementasikan

1. **SOLID Principles**:
   - **S**ingle Responsibility: Setiap komponen memiliki satu tanggung jawab
   - **O**pen/Closed: Entitas dapat diperluas tanpa modifikasi
   - **L**iskov Substitution: Subtype dapat menggantikan parent type
   - **I**nterface Segregation: Interface yang kecil dan spesifik
   - **D**ependency Inversion: Dependensi pada abstraksi bukan konkret implementation
2. **DRY (Don't Repeat Yourself)**: Hindari duplikasi kode dengan abstraksi yang tepat
3. **KISS (Keep It Simple, Stupid)**: Solusi sederhana lebih baik daripada yang kompleks
4. **Loose Coupling**: Komponen seminimal mungkin memiliki ketergantungan
5. **High Cohesion**: Fungsi terkait dikelompokkan bersama

### Fitur yang Dibutuhkan

1. Menerima request pengiriman email via REST API
2. Mendukung email dengan template (HTML dan text)
3. Antrian email menggunakan message broker (RabbitMQ/Kafka)
4. Rate limiting untuk mencegah abuse
5. Retry mechanism untuk email yang gagal terkirim
6. Tracking status pengiriman email
7. Support untuk attachment
8. Multiple email provider (SMTP, AWS SES, dll)

### Teknologi yang Digunakan

- **Framework**: Fiber untuk REST API
- **Message Queue**: RabbitMQ atau Kafka untuk antrian
- **Database**: MariaDB untuk menyimpan template dan history
- **Caching**: Redis untuk rate limiting dan caching template
- **Logging**: AWS CloudWatch untuk logging terpusat
- **Config**: Viper untuk konfigurasi
- **Email Library**: Gomail atau AWS SDK untuk SES

### Error Handling

- Gunakan custom error types dengan konteks yang jelas
- Log error dengan detail yang cukup untuk debugging
- Return response error yang konsisten dan informatif
- Kategorikan error (validation, business, system, external)

### Logging dengan AWS CloudWatch

- Log semua request dan response dengan trace ID ke CloudWatch
- Struktur log yang konsisten dengan JSON untuk memudahkan query
- Log level yang sesuai (INFO, ERROR, DEBUG)
- Log groups terorganisir berdasarkan environment dan service
- Retention policy yang sesuai dengan kebutuhan
- Capture metrics dalam log untuk:
  - Jumlah email terkirim/gagal
  - Latency pengiriman
  - Queue depth
  - Rate limiting stats

### Testing

- Unit test untuk setiap komponen/service
- Integration test untuk end-to-end flow
- Mocking dependencies dengan testify/mock atau gomock

### Security

- Autentikasi untuk API menggunakan JWT atau API key
- Validasi input dengan sanitasi
- Rate limiting per client
- Enkripsi data sensitif
- HTTPS untuk semua komunikasi

### Best Practices

1. **Graceful Shutdown**: Handle server shutdown dengan benar
2. **Health Check**: Endpoint untuk health check dan readiness
3. **Circuit Breaker**: Untuk external dependencies
4. **Configurability**: Semua konfigurasi dapat diubah via environment variables
5. **Backward Compatibility**: Versi API untuk perubahan breaking
6. **Documentation**: OpenAPI/Swagger untuk REST
7. **Containerization**: Docker multi-stage build untuk image yang kecil
8. **DRY Code**: Implementasi helper functions dan shared utilities untuk menghindari duplikasi
9. **KISS Design**: Implementasi solusi sesederhana mungkin yang menyelesaikan masalah
10. **Consistent Error Handling**: Pattern yang konsisten untuk error handling

### Contoh Implementasi Service Layer

```go
// Ini adalah contoh interface dan implementasi service layer
type EmailService interface {
    SendEmail(ctx context.Context, request *domain.EmailRequest) (string, error)
    GetStatus(ctx context.Context, id string) (*domain.EmailStatus, error)
}

type EmailRepository interface {
    SaveEmail(ctx context.Context, email *domain.Email) error
    UpdateStatus(ctx context.Context, id string, status domain.DeliveryStatus) error
    GetByID(ctx context.Context, id string) (*domain.Email, error)
}

type EmailDelivery interface {
    Send(ctx context.Context, email *domain.Email) error
}

type emailService struct {
    repo      EmailRepository
    delivery  EmailDelivery
    templates TemplateRepository
    logger    *cloudwatchlogs.CloudWatchLogs
}

func NewEmailService(repo EmailRepository, delivery EmailDelivery, templates TemplateRepository, logger *cloudwatchlogs.CloudWatchLogs) EmailService {
    return &emailService{
        repo:      repo,
        delivery:  delivery,
        templates: templates,
        logger:    logger,
    }
}
```

Saat Anda membantu dengan kode, prioritaskan:

1. Clean code dengan penamaan yang jelas
2. Error handling yang robust
3. Unit testing untuk fungsi yang dibuat
4. Dokumentasi untuk fungsi publik
5. Dependency injection untuk testability
6. Implementasi SOLID, DRY, dan KISS principles:
   - Gunakan interface untuk dependency inversion
   - Hindari duplikasi kode
   - Buat solusi sesederhana mungkin
   - Pastikan setiap fungsi memiliki single responsibility

Pastikan semua fungsi dan package mempertahankan separation of concerns yang baik sesuai dengan clean architecture.
