# GitHub Actions untuk Email Microservice

## Pull Request Workflow

Workflow ini akan berjalan setiap kali ada pull request ke branch `main`. Workflow ini melakukan:

1. **Unit Testing**

   - Menjalankan semua unit test di direktori `internal/`
   - Menggunakan layanan Docker untuk PostgreSQL, Redis, dan RabbitMQ
   - Menghasilkan laporan code coverage

2. **Vulnerability Scanning**

   - Menggunakan `gosec` untuk menganalisis kerentanan kode
   - Menggunakan `govulncheck` untuk memeriksa kerentanan pustaka Go
   - Menggunakan `nancy` untuk memeriksa kerentanan dependencies

3. **Code Quality Check**
   - Menggunakan `golangci-lint` dengan berbagai linter

## Persyaratan

Untuk memanfaatkan GitHub Actions ini, pastikan:

- Kode telah di-push ke GitHub
- Branch `main` telah dikonfigurasi dengan perlindungan (protected)
- Repository memiliki akses untuk menjalankan GitHub Actions

## Pengaturan Branch Protection

Disarankan untuk mengatur branch protection untuk branch `main`:

1. Buka repository di GitHub
2. Buka tab "Settings"
3. Pilih "Branches"
4. Klik "Add rule" untuk branch `main`
5. Aktifkan "Require status checks to pass before merging"
6. Cari dan aktifkan status check "unit-test", "code-vuln-scan", dan "code-quality"
7. Simpan aturan

## Konfigurasi Tambahan

- File `.golangci.yml` mengonfigurasi linter yang digunakan dalam code quality check
- Jika perlu mengubah versi Go, ubah bagian `go-version` di file workflow

## Alur Kerja

1. Developer membuat branch feature
2. Developer men-push code dan membuat pull request ke `main`
3. GitHub Actions otomatis menjalankan unit test, vulnerability scanning, dan code quality check
4. Pull request hanya dapat di-merge jika semua status check berhasil
