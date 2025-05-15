package cmd

import (
	"email-service/cmd/migrate"
	"email-service/cmd/server"
	"email-service/internal/config"
	"fmt"
	"log"
	"os"
)

// Execute menjalankan perintah utama dari CLI
func Execute() {
	// Cek argumen untuk menentukan perintah yang akan dijalankan
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Perintah yang didukung
	command := os.Args[1]

	// Opsi konfigurasi dasar
	configOptions := config.ConfigOptions{
		ServiceOverride: command,
		ConfigType:      "env",               // Menggunakan .env sebagai tipe konfigurasi default
		ConfigPath:      []string{".", "./"}, // Memeriksa file .env di direktori saat ini dan folder config
	}

	// Load konfigurasi dengan viper
	cfg, err := config.LoadConfigWithOptions(configOptions)
	if err != nil {
		log.Fatalf("Gagal memuat konfigurasi: %v", err)
	}

	// Jalankan perintah yang sesuai
	switch command {
	case "http", "server":
		server.RunServer(cfg)
	case "migrate":
		if len(os.Args) < 3 {
			log.Fatalf("Perintah migrate membutuhkan subcommand: up, down, atau goto")
		}
		migrate.RunMigrate(cfg, os.Args[2:])
	case "version":
		printVersion()
	case "help":
		printUsage()
	default:
		log.Fatalf("Perintah tidak dikenal: %s", command)
	}
}

// printUsage menampilkan petunjuk penggunaan aplikasi
func printUsage() {
	fmt.Println("Email Service CLI")
	fmt.Println("Penggunaan:")
	fmt.Println("  go run main.go [perintah] [argumen...]")
	fmt.Println()
	fmt.Println("Perintah yang tersedia:")
	fmt.Println("  http, server         Menjalankan server HTTP")
	fmt.Println("  migrate up           Menjalankan migrasi database")
	fmt.Println("  migrate down         Rollback semua migrasi database")
	fmt.Println("  migrate goto [v]     Menjalankan migrasi ke versi tertentu")
	fmt.Println("  version              Menampilkan versi aplikasi")
	fmt.Println("  help                 Menampilkan bantuan ini")
}

// printVersion menampilkan versi aplikasi
func printVersion() {
	fmt.Println("App Name:" + os.Getenv("APP_SERVICE_NAME"))
	fmt.Println("Version: " + os.Getenv("APP_VERSION"))
	fmt.Println("Copyright (c) 2025")
}
