package migrate

import (
	"email-service/internal/config"
	"email-service/internal/database"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RunMigrate menjalankan perintah migrasi database
func RunMigrate(cfg *config.Config, args []string) {
	if len(args) < 1 {
		log.Fatalf("Perintah migrate membutuhkan subcommand")
	}

	// Dapatkan subperintah migrasi
	action := args[0]

	// Buat koneksi database
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer db.Close()

	// Tentukan path absolut untuk folder migrasi
	// Gunakan path relatif terhadap root project
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Gagal mendapatkan working directory: %v", err)
	}

	// Path migrasi default - relatif terhadap root project
	migrationPath := filepath.Join(workDir, "migrations")

	// Cek apakah folder migrations ada
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		log.Fatalf("Folder migrasi tidak ditemukan di path: %s", migrationPath)
	}

	// Cari path migrasi custom dari argumen (jika ada)
	for i, arg := range args {
		if arg == "--path" && i+1 < len(args) {
			migrationPath = args[i+1]
			break
		}
		if strings.HasPrefix(arg, "--path=") {
			migrationPath = strings.TrimPrefix(arg, "--path=")
			break
		}
	}

	// Pastikan path migrasi adalah path absolut
	if !filepath.IsAbs(migrationPath) {
		migrationPath = filepath.Join(workDir, migrationPath)
	}

	log.Printf("Menggunakan folder migrasi: %s", migrationPath)

	// Jalankan perintah migrasi sesuai dengan subperintah
	switch action {
	case "up":
		log.Println("Menjalankan migrasi database...")
		if err := db.Migrate(migrationPath); err != nil {
			log.Fatalf("Gagal menjalankan migrasi: %v", err)
		}
		log.Println("Migrasi database berhasil!")

	case "down":
		log.Println("Melakukan rollback migrasi database...")
		if err := db.MigrateDown(); err != nil {
			log.Fatalf("Gagal melakukan rollback migrasi: %v", err)
		}
		log.Println("Rollback migrasi database berhasil!")

	case "goto":
		if len(args) < 2 {
			log.Fatalf("Perintah 'goto' membutuhkan nomor versi")
		}

		// Parse versi
		versionStr := args[1]
		version, err := strconv.ParseUint(versionStr, 10, 32)
		if err != nil {
			log.Fatalf("Nomor versi harus berupa angka: %v", err)
		}

		log.Printf("Menjalankan migrasi database ke versi %d...\n", version)
		if err := db.MigrateTo(uint(version)); err != nil {
			log.Fatalf("Gagal migrasi ke versi %d: %v", version, err)
		}
		log.Printf("Migrasi database ke versi %d berhasil!\n", version)

	case "force":
		if len(args) < 2 {
			log.Fatalf("Perintah 'force' membutuhkan nomor versi")
		}

		// Parse versi
		versionStr := args[1]
		version, err := strconv.ParseUint(versionStr, 10, 32)
		if err != nil {
			log.Fatalf("Nomor versi harus berupa angka: %v", err)
		}

		log.Printf("Memaksa versi migrasi database ke %d...\n", version)
		if err := db.ForceVersion(uint(version)); err != nil {
			log.Fatalf("Gagal memaksa migrasi ke versi %d: %v", version, err)
		}
		log.Printf("Database berhasil diatur ke versi %d dan status dirty dibersihkan!\n", version)

	default:
		fmt.Fprintf(os.Stderr, "Perintah migrasi tidak dikenal: %s\n", action)
		fmt.Fprintf(os.Stderr, "Perintah yang tersedia: up, down, goto, force\n")
		os.Exit(1)
	}
}
