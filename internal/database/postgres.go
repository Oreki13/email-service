package database

import (
	"context"
	"database/sql"
	"email-service/internal/config"
	"email-service/internal/domain"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // File source driver
	_ "github.com/lib/pq"                                // PostgreSQL driver
)

// PostgresDB adalah wrapper untuk koneksi database SQL
type PostgresDB struct {
	DB *sql.DB
}

// NewPostgresDB membuat dan menginisialisasi koneksi database baru
func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	// Buat connection string untuk PostgreSQL
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Database,
	)

	// Buka koneksi ke database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, domain.DatabaseError("failed to connect to PostgreSQL", err)
	}

	// Konfigurasi koneksi pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Verifikasi koneksi dengan ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close() // tutup koneksi jika ping gagal
		return nil, domain.DatabaseError("failed to ping PostgreSQL", err)
	}

	return &PostgresDB{DB: db}, nil
}

// Close menutup koneksi database
func (p *PostgresDB) Close() error {
	if p.DB != nil {
		return p.DB.Close()
	}
	return nil
}

// GetDB mengembalikan objek database SQL
func (p *PostgresDB) GetDB() *sql.DB {
	return p.DB
}

// Migrate menjalankan migrasi database menggunakan golang-migrate
func (p *PostgresDB) Migrate(migrationPath string) error {
	log.Printf("Menjalankan migrasi dari path: %s", migrationPath)

	// Verifikasi path migrasi
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		return domain.DatabaseError(fmt.Sprintf("migration path does not exist: %s", migrationPath), err)
	}

	// Gunakan folder migrate di dalam path migrasi
	migratePath := filepath.Join(migrationPath, "migrate")
	if _, err := os.Stat(migratePath); os.IsNotExist(err) {
		log.Printf("Subfolder migrate tidak ditemukan di %s, menggunakan path asli", migrationPath)
		migratePath = migrationPath
	}

	// Buat driver instance untuk postgres
	driver, err := postgres.WithInstance(p.DB, &postgres.Config{})
	if err != nil {
		return domain.DatabaseError("failed to create migration driver instance", err)
	}

	// Konversi path ke format file URL
	migrationURL := fmt.Sprintf("file://%s", migratePath)
	log.Printf("Migration URL: %s", migrationURL)

	// Buat migrate instance dengan source dari file sistem
	m, err := migrate.NewWithDatabaseInstance(
		migrationURL,
		"postgres", driver)
	if err != nil {
		return domain.DatabaseError("failed to create migration instance", err)
	}

	// Jalankan migrasi up untuk menerapkan semua migrasi
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No migration needed, database is up to date")
			return nil
		}
		return domain.DatabaseError("failed to run migrations", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// MigrateDown menjalankan rollback semua migrasi
func (p *PostgresDB) MigrateDown() error {
	// Dapatkan working directory
	workDir, err := os.Getwd()
	if err != nil {
		return domain.DatabaseError("failed to get working directory", err)
	}

	// Path migrasi default
	migrationPath := filepath.Join(workDir, "migrations")

	// Verifikasi path migrasi
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		return domain.DatabaseError(fmt.Sprintf("migration path does not exist: %s", migrationPath), err)
	}

	// Gunakan folder migrate di dalam path migrasi
	migratePath := filepath.Join(migrationPath, "migrate")
	if _, err := os.Stat(migratePath); os.IsNotExist(err) {
		log.Printf("Subfolder migrate tidak ditemukan di %s, menggunakan path asli", migrationPath)
		migratePath = migrationPath
	}

	// Buat driver instance untuk postgres
	driver, err := postgres.WithInstance(p.DB, &postgres.Config{})
	if err != nil {
		return domain.DatabaseError("failed to create migration driver instance", err)
	}

	// Konversi path ke format file URL
	migrationURL := fmt.Sprintf("file://%s", migratePath)
	log.Printf("Migration URL: %s", migrationURL)

	// Buat migrate instance dengan source dari file sistem
	m, err := migrate.NewWithDatabaseInstance(
		migrationURL,
		"postgres", driver)
	if err != nil {
		return domain.DatabaseError("failed to create migration instance", err)
	}

	// Jalankan migrasi down untuk rollback semua migrasi
	if err := m.Down(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No migration to rollback")
			return nil
		}
		return domain.DatabaseError("failed to rollback migrations", err)
	}

	log.Println("Database rollback completed successfully")
	return nil
}

// MigrateTo menjalankan migrasi ke versi tertentu
func (p *PostgresDB) MigrateTo(version uint) error {
	// Dapatkan working directory
	workDir, err := os.Getwd()
	if err != nil {
		return domain.DatabaseError("failed to get working directory", err)
	}

	// Path migrasi default
	migrationPath := filepath.Join(workDir, "migrations")

	// Verifikasi path migrasi
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		return domain.DatabaseError(fmt.Sprintf("migration path does not exist: %s", migrationPath), err)
	}

	// Gunakan folder migrate di dalam path migrasi
	migratePath := filepath.Join(migrationPath, "migrate")
	if _, err := os.Stat(migratePath); os.IsNotExist(err) {
		log.Printf("Subfolder migrate tidak ditemukan di %s, menggunakan path asli", migrationPath)
		migratePath = migrationPath
	}

	// Buat driver instance untuk postgres
	driver, err := postgres.WithInstance(p.DB, &postgres.Config{})
	if err != nil {
		return domain.DatabaseError("failed to create migration driver instance", err)
	}

	// Konversi path ke format file URL
	migrationURL := fmt.Sprintf("file://%s", migratePath)
	log.Printf("Migration URL: %s", migrationURL)

	// Buat migrate instance dengan source dari file sistem
	m, err := migrate.NewWithDatabaseInstance(
		migrationURL,
		"postgres", driver)
	if err != nil {
		return domain.DatabaseError("failed to create migration instance", err)
	}

	// Jalankan migrasi ke versi tertentu
	if err := m.Migrate(version); err != nil {
		if err == migrate.ErrNoChange {
			log.Printf("No migration needed, database is at version %d\n", version)
			return nil
		}
		return domain.DatabaseError(fmt.Sprintf("failed to migrate to version %d", version), err)
	}

	log.Printf("Database migration to version %d completed successfully\n", version)
	return nil
}

// ForceVersion memaksa versi migrasi ke versi tertentu dan membersihkan status dirty
func (p *PostgresDB) ForceVersion(version uint) error {
	// Dapatkan working directory
	workDir, err := os.Getwd()
	if err != nil {
		return domain.DatabaseError("failed to get working directory", err)
	}

	// Path migrasi default
	migrationPath := filepath.Join(workDir, "migrations")

	// Verifikasi path migrasi
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		return domain.DatabaseError(fmt.Sprintf("migration path does not exist: %s", migrationPath), err)
	}

	// Gunakan folder migrate di dalam path migrasi
	migratePath := filepath.Join(migrationPath, "migrate")
	if _, err := os.Stat(migratePath); os.IsNotExist(err) {
		log.Printf("Subfolder migrate tidak ditemukan di %s, menggunakan path asli", migrationPath)
		migratePath = migrationPath
	}

	// Buat driver instance untuk postgres
	driver, err := postgres.WithInstance(p.DB, &postgres.Config{})
	if err != nil {
		return domain.DatabaseError("failed to create migration driver instance", err)
	}

	// Konversi path ke format file URL
	migrationURL := fmt.Sprintf("file://%s", migratePath)
	log.Printf("Migration URL: %s", migrationURL)

	// Buat migrate instance dengan source dari file sistem
	m, err := migrate.NewWithDatabaseInstance(
		migrationURL,
		"postgres", driver)
	if err != nil {
		return domain.DatabaseError("failed to create migration instance", err)
	}

	// Force version untuk membersihkan status dirty
	if err := m.Force(int(version)); err != nil {
		return domain.DatabaseError(fmt.Sprintf("failed to force version to %d", version), err)
	}

	log.Printf("Database version has been forced to %d and dirty flag cleared\n", version)
	return nil
}

// RunInTransaction menjalankan fungsi dalam transaksi database
func (p *PostgresDB) RunInTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.DatabaseError("failed to begin transaction", err)
	}

	// Panggil fungsi dalam konteks transaksi
	err = fn(tx)
	if err != nil {
		// Rollback transaksi jika terjadi error
		if rbErr := tx.Rollback(); rbErr != nil {
			return domain.DatabaseError(
				fmt.Sprintf("error on rollback: %v, original error: %v", rbErr, err),
				err,
			)
		}
		return err
	}

	// Commit transaksi jika tidak ada error
	if err := tx.Commit(); err != nil {
		return domain.DatabaseError("failed to commit transaction", err)
	}

	return nil
}
