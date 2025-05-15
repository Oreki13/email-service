package delivery

// SMTPConfig berisi konfigurasi koneksi SMTP
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	UseSSL   bool
}

// SESConfig berisi konfigurasi koneksi AWS SES
type SESConfig struct {
	Region    string
	AccessKey string
	SecretKey string
}
