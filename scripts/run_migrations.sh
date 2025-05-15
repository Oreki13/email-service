#!/bin/bash

# Script untuk menjalankan migrasi database PostgreSQL
# Penggunaan: ./run_migrations.sh [up|down] [nomor-migrasi]
# Contoh: 
#   ./run_migrations.sh up      # Jalankan semua migrasi
#   ./run_migrations.sh up 1    # Jalankan migrasi 001
#   ./run_migrations.sh down 1  # Rollback migrasi 001

set -e

# Warna untuk output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Direktori migrasi
MIGRATIONS_DIR="$(dirname "$0")/../migrations"

# Variabel database dari environment atau default
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5432"}
DB_USER=${DB_USER:-"postgres"}
DB_PASS=${DB_PASS:-"1234"}
DB_NAME=${DB_NAME:-"email_service"}

# Periksa dependensi
command -v psql >/dev/null 2>&1 || { echo -e "${RED}Error: PostgreSQL client tidak ditemukan. Silakan install terlebih dahulu.${NC}" >&2; exit 1; }

# Fungsi untuk menampilkan penggunaan
function show_usage {
    echo -e "Penggunaan: $0 [up|down] [nomor-migrasi]"
    echo -e "  up   : Jalankan migrasi"
    echo -e "  down : Rollback migrasi"
    echo -e "  [nomor-migrasi] : Opsional, nomor migrasi spesifik yang akan dijalankan"
}

# Fungsi untuk menjalankan migrasi
function run_migration {
    local file=$1
    echo -e "${YELLOW}Menjalankan migrasi: ${file}${NC}"
    
    # Set PGPASSWORD environment variable
    export PGPASSWORD=$DB_PASS
    
    # Eksekusi file SQL menggunakan psql client
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $file
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Migrasi berhasil: ${file}${NC}"
        return 0
    else
        echo -e "${RED}✗ Migrasi gagal: ${file}${NC}"
        return 1
    fi
}

# Validasi parameter
if [ "$#" -lt 1 ]; then
    show_usage
    exit 1
fi

direction=$1
migration_number=$2

# Pastikan direction valid
if [[ "$direction" != "up" && "$direction" != "down" ]]; then
    echo -e "${RED}Error: Parameter pertama harus 'up' atau 'down'${NC}"
    show_usage
    exit 1
fi

# Jalankan migrasi
if [ "$direction" == "up" ]; then
    echo -e "${YELLOW}=== Menjalankan migrasi UP ===${NC}"
    
    # Jika nomor migrasi ditentukan
    if [ -n "$migration_number" ]; then
        # Format nomor dengan leading zeros
        formatted_number=$(printf "%03d" $migration_number)
        migration_file="${MIGRATIONS_DIR}/${formatted_number}_*.sql"
        
        # Cek jika file migrasi exist
        if ls $migration_file 1> /dev/null 2>&1; then
            # Jalankan migrasi
            for file in $migration_file; do
                # Skip file rollback
                if [[ $file != *"rollback"* ]]; then
                    run_migration $file
                fi
            done
        else
            echo -e "${RED}Error: Tidak ada file migrasi dengan nomor ${formatted_number}${NC}"
            exit 1
        fi
    else
        # Jalankan semua migrasi
        for file in $(ls ${MIGRATIONS_DIR}/*.sql | sort); do
            # Skip file rollback
            if [[ $file != *"rollback"* ]]; then
                run_migration $file
            fi
        done
    fi
else
    echo -e "${YELLOW}=== Menjalankan migrasi DOWN (rollback) ===${NC}"
    
    # Jika nomor migrasi ditentukan
    if [ -n "$migration_number" ]; then
        # Format nomor dengan leading zeros
        formatted_number=$(printf "%03d" $migration_number)
        rollback_file="${MIGRATIONS_DIR}/${formatted_number}_*_rollback.sql"
        
        # Cek jika file rollback exists
        if ls $rollback_file 1> /dev/null 2>&1; then
            # Jalankan rollback
            for file in $rollback_file; do
                run_migration $file
            done
        else
            echo -e "${RED}Error: Tidak ada file rollback dengan nomor ${formatted_number}${NC}"
            exit 1
        fi
    else
        # Rollback semua migrasi (terbalik)
        for file in $(ls ${MIGRATIONS_DIR}/*_rollback.sql | sort -r); do
            run_migration $file
        done
    fi
fi

echo -e "${YELLOW}=== Proses migrasi selesai ===${NC}"