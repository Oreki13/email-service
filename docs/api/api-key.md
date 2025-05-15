# API Key Management

API ini digunakan untuk mengelola API key yang diperlukan untuk mengakses Email Microservice. Berbeda dengan endpoint lainnya, endpoint manajemen API key memerlukan autentikasi admin.

## Endpoints

### Membuat API Key

Endpoint ini digunakan untuk membuat API key baru.

**URL**: `/api/v1/api-keys`

**Metode**: `POST`

**Auth diperlukan**: Ya (Admin Token di header `X-Admin-Token`)

**Struktur Request**:

```json
{
  "name": "Service A - Production",
  "description": "API Key untuk penggunaan production Service A",
  "permissions": ["send_email", "manage_templates"],
  "rateLimits": {
    "maxRequestsPerMinute": 1000,
    "maxEmailsPerDay": 10000
  },
  "expiry": "2026-12-31T23:59:59Z",
  "metadata": {
    "ownerTeam": "Team A",
    "environment": "production",
    "costCenter": "CC123"
  }
}
```

**Fields**:

- `name` (wajib): Nama untuk identifikasi API key
- `description` (opsional): Deskripsi penggunaan API key
- `permissions` (wajib): Array permission yang diberikan, nilai yang mungkin:
  - `send_email`: Mengizinkan pengiriman email
  - `manage_templates`: Mengizinkan manajemen template
  - `view_tracking`: Mengizinkan akses ke data tracking
- `rateLimits` (opsional): Batasan penggunaan
  - `maxRequestsPerMinute`: Jumlah maksimum request API per menit
  - `maxEmailsPerDay`: Jumlah maksimum email yang dapat dikirim per hari
- `expiry` (opsional): Tanggal kedaluwarsa API key (format ISO8601)
- `metadata` (opsional): Informasi tambahan

**Response Sukses**: `201 Created`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "API Key created successfully",
  "data": {
    "id": "api-key-id-123456",
    "key": "svc.abcdefghijklmnopqrstuvwxyz123456",
    "name": "Service A - Production",
    "permissions": ["send_email", "manage_templates"],
    "createdAt": "2025-05-13T10:30:45Z",
    "expiresAt": "2026-12-31T23:59:59Z"
  }
}
```

**Catatan**: Nilai `key` dalam response hanya akan ditampilkan sekali saat pembuatan. Simpan nilai ini dengan aman karena tidak akan dapat diambil kembali.

### Mendapatkan Daftar API Key

Endpoint ini digunakan untuk mendapatkan daftar API key yang ada.

**URL**: `/api/v1/api-keys`

**Metode**: `GET`

**Auth diperlukan**: Ya (Admin Token di header `X-Admin-Token`)

**Query Parameters**:

- `page` (opsional): Nomor halaman, default 1
- `limit` (opsional): Jumlah item per halaman, default 20
- `active` (opsional): Filter berdasarkan status aktif, nilai: `true` atau `false`

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Success",
  "data": {
    "apiKeys": [
      {
        "id": "api-key-id-123456",
        "name": "Service A - Production",
        "description": "API Key untuk penggunaan production Service A",
        "permissions": ["send_email", "manage_templates"],
        "active": true,
        "createdAt": "2025-05-13T10:30:45Z",
        "expiresAt": "2026-12-31T23:59:59Z",
        "lastUsedAt": "2025-05-13T15:45:22Z"
      }
    ],
    "pagination": {
      "currentPage": 1,
      "totalPages": 1,
      "totalItems": 1,
      "itemsPerPage": 20
    }
  }
}
```

### Mendapatkan Detail API Key

Endpoint ini digunakan untuk mendapatkan detail API key.

**URL**: `/api/v1/api-keys/:id`

**Metode**: `GET`

**Auth diperlukan**: Ya (Admin Token di header `X-Admin-Token`)

**URL Parameters**:

- `id`: ID API key yang ingin dilihat

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Success",
  "data": {
    "id": "api-key-id-123456",
    "name": "Service A - Production",
    "description": "API Key untuk penggunaan production Service A",
    "permissions": ["send_email", "manage_templates"],
    "rateLimits": {
      "maxRequestsPerMinute": 1000,
      "maxEmailsPerDay": 10000,
      "currentUsage": {
        "requestsLastMinute": 150,
        "emailsSentToday": 2500
      }
    },
    "active": true,
    "createdAt": "2025-05-13T10:30:45Z",
    "expiresAt": "2026-12-31T23:59:59Z",
    "lastUsedAt": "2025-05-13T15:45:22Z",
    "usageStats": {
      "totalRequests": 12500,
      "totalEmailsSent": 75000,
      "lastActivities": [
        {
          "action": "send_email",
          "timestamp": "2025-05-13T15:45:22Z",
          "ipAddress": "192.168.1.1"
        }
      ]
    },
    "metadata": {
      "ownerTeam": "Team A",
      "environment": "production",
      "costCenter": "CC123"
    }
  }
}
```

### Memperbarui API Key

Endpoint ini digunakan untuk memperbarui properti API key.

**URL**: `/api/v1/api-keys/:id`

**Metode**: `PUT`

**Auth diperlukan**: Ya (Admin Token di header `X-Admin-Token`)

**URL Parameters**:

- `id`: ID API key yang ingin diperbarui

**Struktur Request**:

```json
{
  "name": "Service A - Production Updated",
  "description": "API Key untuk penggunaan production Service A (updated)",
  "permissions": ["send_email", "manage_templates", "view_tracking"],
  "rateLimits": {
    "maxRequestsPerMinute": 2000,
    "maxEmailsPerDay": 20000
  },
  "expiry": "2027-12-31T23:59:59Z",
  "metadata": {
    "ownerTeam": "Team A",
    "environment": "production",
    "costCenter": "CC456"
  }
}
```

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "API Key updated successfully",
  "data": {
    "id": "api-key-id-123456",
    "updatedAt": "2025-05-13T16:30:45Z"
  }
}
```

### Menonaktifkan API Key

Endpoint ini digunakan untuk menonaktifkan API key.

**URL**: `/api/v1/api-keys/:id/deactivate`

**Metode**: `POST`

**Auth diperlukan**: Ya (Admin Token di header `X-Admin-Token`)

**URL Parameters**:

- `id`: ID API key yang ingin dinonaktifkan

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "API Key deactivated successfully",
  "data": {
    "id": "api-key-id-123456",
    "deactivatedAt": "2025-05-13T16:35:20Z"
  }
}
```

### Mengaktifkan Kembali API Key

Endpoint ini digunakan untuk mengaktifkan kembali API key yang telah dinonaktifkan.

**URL**: `/api/v1/api-keys/:id/activate`

**Metode**: `POST`

**Auth diperlukan**: Ya (Admin Token di header `X-Admin-Token`)

**URL Parameters**:

- `id`: ID API key yang ingin diaktifkan kembali

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "API Key activated successfully",
  "data": {
    "id": "api-key-id-123456",
    "activatedAt": "2025-05-13T16:40:15Z"
  }
}
```

### Menghapus API Key

Endpoint ini digunakan untuk menghapus API key secara permanen.

**URL**: `/api/v1/api-keys/:id`

**Metode**: `DELETE`

**Auth diperlukan**: Ya (Admin Token di header `X-Admin-Token`)

**URL Parameters**:

- `id`: ID API key yang ingin dihapus

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "API Key deleted successfully",
  "data": null
}
```

## Format API Key

API key menggunakan format berikut:

```
svc.abcdefghijklmnopqrstuvwxyz123456
```

- Awalan `svc.` menunjukkan bahwa ini adalah service API key
- Bagian selanjutnya adalah string acak yang digunakan untuk autentikasi

## Rate Limiting

API key dapat dikonfigurasi dengan batasan penggunaan untuk mencegah penyalahgunaan:

- `maxRequestsPerMinute`: Membatasi jumlah request API per menit
- `maxEmailsPerDay`: Membatasi jumlah email yang dapat dikirim per hari

Jika batas ini terlampaui, API akan mengembalikan respons error dengan status code `429 Too Many Requests`.

## Praktik Keamanan Terbaik

1. **Simpan API key dengan aman**: Jangan simpan API key dalam kode yang di-commit ke repositori publik
2. **Gunakan permission minimal**: Berikan hanya permission yang diperlukan
3. **Rotasi key secara berkala**: Buat API key baru dan hapus yang lama secara berkala
4. **Gunakan environment berbeda**: Gunakan API key terpisah untuk development dan production
5. **Pantau penggunaan**: Periksa statistik penggunaan untuk mendeteksi aktivitas mencurigakan
6. **Atur batas penggunaan**: Konfigurasikan rate limiting untuk mencegah penyalahgunaan
