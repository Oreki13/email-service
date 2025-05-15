# Dokumentasi API Email Microservice

Dokumentasi ini berisi informasi tentang API yang tersedia dalam Email Microservice. Service ini menyediakan berbagai endpoint untuk mengirim email, mengelola template, melacak aktivitas email, dan mengelola API key.

## Daftar Isi

1. [Autentikasi](#autentikasi)
2. [Format Response](#format-response)
3. [API Endpoints](#api-endpoints)
   - [Email API](./api/email.md)
   - [Template API](./api/template.md)
   - [Tracking API](./api/tracking.md)
   - [API Key Management](./api/api-key.md)
4. [Contoh Penggunaan](./examples/)
5. [OpenAPI Specification](./openapi.json)

## Autentikasi

Semua endpoint API memerlukan autentikasi menggunakan API key. API key harus disertakan dalam header HTTP `X-API-Key`.

Contoh:

```
X-API-Key: your-api-key-here
```

Endpoint untuk manajemen API key memerlukan autentikasi admin menggunakan token admin yang dikirimkan dalam header `X-Admin-Token`.

## Format Response

Semua endpoint API mengembalikan response dalam format JSON yang konsisten dengan struktur berikut:

```json
{
  "status": "success|error",
  "traceID": "unique-trace-id",
  "message": "Pesan deskriptif tentang hasil operasi",
  "data": {
    // Data response yang berbeda-beda untuk setiap endpoint
  }
}
```

### Status Code

Service menggunakan HTTP status code standar:

- `200 OK` - Request berhasil
- `202 Accepted` - Request diterima dan sedang diproses
- `400 Bad Request` - Parameter tidak valid atau kurang
- `401 Unauthorized` - API key tidak valid
- `403 Forbidden` - API key valid tetapi tidak memiliki izin
- `404 Not Found` - Resource tidak ditemukan
- `500 Internal Server Error` - Terjadi error di server

## Dokumentasi Endpoint

Untuk detail lengkap setiap API, silakan lihat dokumentasi khusus:

- [Email API](./api/email.md) - Mengirim email custom
- [Template API](./api/template.md) - Mengirim email berdasarkan template
- [Tracking API](./api/tracking.md) - Melacak email yang dikirim
- [API Key Management](./api/api-key.md) - Mengelola API key

## Contoh Penggunaan

Kunjungi [folder examples](./examples/) untuk contoh penggunaan API dalam berbagai bahasa pemrograman.
