-- +migrate Up
-- +migrate StatementBegin

-- Tabel untuk template email
CREATE TABLE IF NOT EXISTS email_templates (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    subject TEXT NOT NULL,
    plain_body TEXT,
    html_body TEXT NOT NULL,
    variables JSON,
    is_active BOOLEAN DEFAULT TRUE,
    version INT DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_email_templates_name ON email_templates(name);
CREATE INDEX idx_email_templates_is_active ON email_templates(is_active);

-- Tabel untuk email
CREATE TABLE IF NOT EXISTS emails (
    id VARCHAR(36) PRIMARY KEY,
    from_email VARCHAR(255) NOT NULL,
    to_emails JSON NOT NULL,
    cc_emails JSON,
    bcc_emails JSON,
    subject TEXT NOT NULL,
    plain_body TEXT,
    html_body TEXT,
    template_id VARCHAR(36),
    template_data JSON,
    status VARCHAR(20) NOT NULL,
    priority VARCHAR(10) NOT NULL,
    provider VARCHAR(20) NOT NULL,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    error TEXT,
    metadata JSON,
    sent_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES email_templates(id) ON DELETE SET NULL
);

CREATE INDEX idx_emails_status ON emails(status);
CREATE INDEX idx_emails_priority ON emails(priority);
CREATE INDEX idx_emails_provider ON emails(provider);
CREATE INDEX idx_emails_created_at ON emails(created_at);
CREATE INDEX idx_emails_sent_at ON emails(sent_at);

-- Tabel untuk attachment
CREATE TABLE IF NOT EXISTS email_attachments (
    id VARCHAR(36) PRIMARY KEY,
    email_id VARCHAR(36) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size INT NOT NULL,
    storage_path TEXT,
    storage_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

CREATE INDEX idx_email_attachments_email_id ON email_attachments(email_id);

-- Tabel untuk tracking email
CREATE TABLE IF NOT EXISTS email_tracking (
    id VARCHAR(36) PRIMARY KEY,
    email_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(20) NOT NULL, -- open, click, bounce, etc.
    ip_address VARCHAR(45),
    user_agent TEXT,
    url TEXT, -- untuk click tracking
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

CREATE INDEX idx_email_tracking_email_id ON email_tracking(email_id);
CREATE INDEX idx_email_tracking_event_type ON email_tracking(event_type);
CREATE INDEX idx_email_tracking_created_at ON email_tracking(created_at);

-- Tabel untuk API key
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(36) PRIMARY KEY,
    key VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    service_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_keys_key ON api_keys(key);
CREATE INDEX idx_api_keys_service_name ON api_keys(service_name);
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin

DROP TABLE IF EXISTS email_tracking;
DROP TABLE IF EXISTS email_attachments;
DROP TABLE IF EXISTS emails;
DROP TABLE IF EXISTS email_templates;
DROP TABLE IF EXISTS api_keys;

-- +migrate StatementEnd
