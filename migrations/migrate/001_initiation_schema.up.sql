-- Mulai transaksi
BEGIN;

-- Buat tabel emails
CREATE TABLE IF NOT EXISTS emails (
    id VARCHAR(36) PRIMARY KEY,
    from_email VARCHAR(255) NOT NULL,
    to_emails TEXT NOT NULL,                   -- JSON array of emails
    cc_emails TEXT,                            -- JSON array of emails
    bcc_emails TEXT,                           -- JSON array of emails
    subject VARCHAR(255) NOT NULL,
    plain_body TEXT,
    html_body TEXT,
    template_id VARCHAR(36),
    template_data JSONB,                       -- JSON data for template
    status VARCHAR(20) NOT NULL,               -- pending, sending, sent, failed, bounced, delivered, opened
    priority VARCHAR(10) NOT NULL,             -- high, normal, low
    provider VARCHAR(10) NOT NULL,             -- smtp, ses
    sent_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    error_message TEXT,
    metadata JSONB                             -- JSON for additional metadata
);

-- Buat indeks
CREATE INDEX idx_emails_status ON emails(status);
CREATE INDEX idx_emails_created_at ON emails(created_at);
CREATE INDEX idx_emails_template_id ON emails(template_id);

-- Buat tabel email_attachments
CREATE TABLE IF NOT EXISTS email_attachments (
    id VARCHAR(36) PRIMARY KEY,
    email_id VARCHAR(36) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size INT NOT NULL,
    storage_path VARCHAR(255),
    content BYTEA,                             -- Binary content untuk attachment kecil
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_email_attachments_email FOREIGN KEY (email_id)
        REFERENCES emails(id) ON DELETE CASCADE
);

-- Buat indeks
CREATE INDEX idx_email_attachments_email_id ON email_attachments(email_id);

-- Buat tabel email_templates
CREATE TABLE IF NOT EXISTS email_templates (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    subject VARCHAR(255) NOT NULL,
    plain_body TEXT,
    html_body TEXT NOT NULL,
    variables JSONB,                           -- JSON array of variable names used in template
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(36),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    version INT NOT NULL DEFAULT 1
);

-- Buat indeks
CREATE UNIQUE INDEX idx_email_templates_name ON email_templates(name);
CREATE INDEX idx_email_templates_is_active ON email_templates(is_active);

-- Tambahkan constraint unique untuk kombinasi name dan is_active
ALTER TABLE email_templates ADD CONSTRAINT uq_email_templates_name UNIQUE (name, is_active);

-- Buat tabel email_events
CREATE TABLE IF NOT EXISTS email_events (
    id VARCHAR(36) PRIMARY KEY,
    email_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(20) NOT NULL,           -- queued, processed, sent, delivered, opened, clicked, bounced, complained, rejected
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data JSONB,                                -- JSON data specific to the event
    ip_address VARCHAR(45),                    -- IPv4 or IPv6
    user_agent TEXT,
    CONSTRAINT fk_email_events_email FOREIGN KEY (email_id)
        REFERENCES emails(id) ON DELETE CASCADE
);

-- Buat indeks
CREATE INDEX idx_email_events_email_id ON email_events(email_id);
CREATE INDEX idx_email_events_event_type ON email_events(event_type);
CREATE INDEX idx_email_events_timestamp ON email_events(timestamp);

-- Buat tabel email_tracking
CREATE TABLE IF NOT EXISTS email_tracking (
    id VARCHAR(36) PRIMARY KEY,
    email_id VARCHAR(36) NOT NULL,
    type VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    user_agent TEXT,
    ip_address VARCHAR(45),
    url TEXT,
    count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

-- Buat indeks
CREATE INDEX idx_email_tracking_email_id ON email_tracking(email_id);
CREATE INDEX idx_email_tracking_type ON email_tracking(type);
CREATE INDEX idx_email_tracking_timestamp ON email_tracking(timestamp);

-- Menambahkan komentar ke tabel
COMMENT ON TABLE email_tracking IS 'Tabel untuk menyimpan data tracking pembukaan dan klik email';
COMMENT ON COLUMN email_tracking.id IS 'Primary key';
COMMENT ON COLUMN email_tracking.email_id IS 'ID email yang di-track';
COMMENT ON COLUMN email_tracking.type IS 'Jenis tracking, misalnya open atau click';
COMMENT ON COLUMN email_tracking.timestamp IS 'Waktu terjadinya event tracking';
COMMENT ON COLUMN email_tracking.user_agent IS 'User agent pengguna';
COMMENT ON COLUMN email_tracking.ip_address IS 'Alamat IP pengguna';
COMMENT ON COLUMN email_tracking.url IS 'URL yang diklik (untuk tracking klik)';
COMMENT ON COLUMN email_tracking.count IS 'Jumlah kejadian tracking';
COMMENT ON COLUMN email_tracking.created_at IS 'Waktu pembuatan record';
COMMENT ON COLUMN email_tracking.updated_at IS 'Waktu terakhir pembaruan record';

-- Buat tabel api_keys dengan struktur yang diperbarui
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    service VARCHAR(100) NOT NULL,
    rate_limit INT NOT NULL DEFAULT 100,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE
);

-- Buat indeks
CREATE UNIQUE INDEX idx_api_keys_api_key ON api_keys(api_key);
CREATE INDEX idx_api_keys_service ON api_keys(service);
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);

-- Buat tabel email_providers
CREATE TABLE IF NOT EXISTS email_providers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    type VARCHAR(10) NOT NULL,                 -- smtp, ses
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    configuration JSONB NOT NULL,              -- JSON configuration
    max_rate INT NOT NULL DEFAULT 0,           -- Max emails per second (0 = unlimited)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Buat indeks
CREATE UNIQUE INDEX idx_email_providers_name ON email_providers(name);
CREATE INDEX idx_email_providers_is_active ON email_providers(is_active);
CREATE INDEX idx_email_providers_is_default ON email_providers(is_default);

-- Sample data untuk pengembangan dan testing

-- Sample email templates
INSERT INTO email_templates (id, name, description, subject, plain_body, html_body, variables, created_at, updated_at, is_active, version) 
VALUES 
(
    'e4a7709c-53d5-4b7a-b9f0-12c9a3e35e8e', 
    'welcome_template', 
    'Template untuk email selamat datang kepada pengguna baru', 
    'Selamat Datang di {{app_name}}', 
    'Halo {{name}},\n\nSelamat datang di {{app_name}}! Kami senang Anda bergabung bersama kami.\n\nKami di sini untuk membantu Anda memulai. Silakan jelajahi platform kami dan hubungi kami jika Anda memiliki pertanyaan.\n\nSalam,\nTim {{app_name}}', 
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Selamat Datang</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 10px; text-align: center; }
        .footer { font-size: 12px; color: #777; text-align: center; margin-top: 30px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Selamat Datang di {{app_name}}</h1>
        </div>
        <div class="content">
            <p>Halo <strong>{{name}}</strong>,</p>
            <p>Selamat datang di <strong>{{app_name}}</strong>! Kami senang Anda bergabung bersama kami.</p>
            <p>Kami di sini untuk membantu Anda memulai. Silakan jelajahi platform kami dan hubungi kami jika Anda memiliki pertanyaan.</p>
            <p><a href="{{dashboard_url}}" class="button">Masuk ke Dashboard</a></p>
        </div>
        <div class="footer">
            <p>Email ini dikirim oleh {{app_name}}. &copy; 2025 Semua hak dilindungi undang-undang.</p>
        </div>
    </div>
</body>
</html>', 
    '["name", "app_name", "dashboard_url"]', 
    NOW(), 
    NOW(), 
    TRUE, 
    1
),
(
    'f8b3d8c2-a74d-48b9-b66c-30e841c32a19', 
    'password_reset', 
    'Template untuk email reset password', 
    'Reset Password untuk {{app_name}}', 
    'Halo {{name}},\n\nKami menerima permintaan untuk mengatur ulang password akun {{app_name}} Anda. Klik tautan di bawah untuk mengatur ulang password Anda:\n\n{{reset_url}}\n\nTautan ini akan kedaluwarsa dalam 1 jam.\n\nJika Anda tidak meminta pengaturan ulang password, Anda dapat mengabaikan email ini.\n\nSalam,\nTim {{app_name}}', 
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Reset Password</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #2196F3; color: white; padding: 10px; text-align: center; }
        .footer { font-size: 12px; color: #777; text-align: center; margin-top: 30px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #2196F3; color: white; text-decoration: none; border-radius: 5px; }
        .warning { color: #FF5722; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Reset Password</h1>
        </div>
        <div class="content">
            <p>Halo <strong>{{name}}</strong>,</p>
            <p>Kami menerima permintaan untuk mengatur ulang password akun <strong>{{app_name}}</strong> Anda.</p>
            <p>Klik tombol di bawah untuk mengatur ulang password Anda:</p>
            <p><a href="{{reset_url}}" class="button">Reset Password</a></p>
            <p class="warning">Tautan ini akan kedaluwarsa dalam 1 jam.</p>
            <p>Jika Anda tidak meminta pengaturan ulang password, Anda dapat mengabaikan email ini.</p>
        </div>
        <div class="footer">
            <p>Email ini dikirim oleh {{app_name}}. &copy; 2025 Semua hak dilindungi undang-undang.</p>
        </div>
    </div>
</body>
</html>', 
    '["name", "app_name", "reset_url"]', 
    NOW(), 
    NOW(), 
    TRUE, 
    1
),
(
    'a1c2e3b4-c5d6-47e8-9f10-11121314a151', 
    'notification_template', 
    'Template umum untuk notifikasi', 
    '{{subject}}', 
    '{{message}}\n\nSalam,\nTim {{app_name}}', 
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Notifikasi</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #673AB7; color: white; padding: 10px; text-align: center; }
        .footer { font-size: 12px; color: #777; text-align: center; margin-top: 30px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{subject}}</h1>
        </div>
        <div class="content">
            <p>{{message}}</p>
        </div>
        <div class="footer">
            <p>Email ini dikirim oleh {{app_name}}. &copy; 2025 Semua hak dilindungi undang-undang.</p>
        </div>
    </div>
</body>
</html>', 
    '["subject", "message", "app_name"]', 
    NOW(), 
    NOW(), 
    TRUE, 
    1
);

-- Sample email providers
INSERT INTO email_providers (id, name, type, is_default, configuration, max_rate, is_active, created_at, updated_at) 
VALUES 
(
    'a9d8c7b6-e5f4-43a2-b1c0-d9e8f7a6b5c4', 
    'Default SMTP', 
    'smtp', 
    TRUE, 
    '{"host": "smtp.example.com", "port": 587, "username": "smtp_user", "password": "smtp_password", "encryption": "tls"}', 
    10, 
    TRUE, 
    NOW(), 
    NOW()
),
(
    'b8c7d6e5-f4a3-42b1-a0c9-e8d7f6a5b4c3', 
    'AWS SES', 
    'ses', 
    FALSE, 
    '{"region": "us-west-2", "access_key": "aws_access_key", "secret_key": "aws_secret_key", "configuration_set": "email_service"}', 
    50, 
    TRUE, 
    NOW(), 
    NOW()
);

-- Menambahkan API key default untuk testing
INSERT INTO api_keys (
    id, api_key, name, description, service, 
    rate_limit, is_active, created_at, updated_at
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'test-api-key',
    'Test API Key',
    'API key untuk testing',
    'test-service',
    100,
    TRUE,
    NOW(),
    NOW()
);

-- Commit transaksi
COMMIT;
