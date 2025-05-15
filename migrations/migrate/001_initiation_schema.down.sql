-- Mulai transaksi
BEGIN;

-- Hapus semua tabel dalam urutan reverse karena foreign key constraints
DROP TABLE IF EXISTS email_providers;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS email_tracking;
DROP TABLE IF EXISTS email_events;
DROP TABLE IF EXISTS email_templates;
DROP TABLE IF EXISTS email_attachments;
DROP TABLE IF EXISTS emails;

-- Commit transaksi
COMMIT;
