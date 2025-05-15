# Template API

API ini memungkinkan pengelolaan dan penggunaan template email. Dengan Template API, Anda dapat menyimpan template HTML untuk email, menggunakannya dengan parameter dinamis, dan melacak penggunaannya.

## Endpoints

### Membuat Template Email

Endpoint ini digunakan untuk membuat template email baru.

**URL**: `/api/v1/templates`

**Metode**: `POST`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**Struktur Request**:

```json
{
  "name": "welcome-email",
  "subject": "Selamat Datang di {{company_name}}",
  "description": "Template untuk email selamat datang",
  "htmlContent": "<html><body><h1>Halo {{name}}!</h1><p>Selamat datang di {{company_name}}.</p></body></html>",
  "textContent": "Halo {{name}}! Selamat datang di {{company_name}}.",
  "category": "onboarding",
  "isActive": true,
  "metadata": {
    "creator": "admin",
    "version": "1.0"
  }
}
```

**Fields**:

- `name` (wajib): Nama unik untuk template
- `subject` (wajib): Subjek email dengan placeholder
- `description` (opsional): Deskripsi template
- `htmlContent` (wajib): Konten HTML dengan placeholder
- `textContent` (opsional): Versi teks biasa
- `category` (opsional): Kategori untuk pengelompokan
- `isActive` (opsional): Status template, default `true`
- `metadata` (opsional): Metadata tambahan

**Response Sukses**: `201 Created`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Template created successfully",
  "data": {
    "id": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
    "name": "welcome-email",
    "subject": "Selamat Datang di {{company_name}}",
    "createdAt": "2025-05-13T10:30:45Z",
    "updatedAt": "2025-05-13T10:30:45Z"
  }
}
```

### Mendapatkan Daftar Template

Endpoint ini digunakan untuk mendapatkan daftar template yang tersedia.

**URL**: `/api/v1/templates`

**Metode**: `GET`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**Query Parameters**:

- `page` (opsional): Nomor halaman, default 1
- `limit` (opsional): Jumlah item per halaman, default 20
- `category` (opsional): Filter berdasarkan kategori
- `status` (opsional): Filter berdasarkan status (`active`, `inactive`)

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Success",
  "data": {
    "templates": [
      {
        "id": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
        "name": "welcome-email",
        "subject": "Selamat Datang di {{company_name}}",
        "description": "Template untuk email selamat datang",
        "category": "onboarding",
        "isActive": true,
        "createdAt": "2025-05-13T10:30:45Z",
        "updatedAt": "2025-05-13T10:30:45Z"
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

### Mendapatkan Detail Template

Endpoint ini digunakan untuk mendapatkan detail lengkap template.

**URL**: `/api/v1/templates/:id`

**Metode**: `GET`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**URL Parameters**:

- `id`: ID template yang ingin dilihat

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Success",
  "data": {
    "id": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
    "name": "welcome-email",
    "subject": "Selamat Datang di {{company_name}}",
    "description": "Template untuk email selamat datang",
    "htmlContent": "<html><body><h1>Halo {{name}}!</h1><p>Selamat datang di {{company_name}}.</p></body></html>",
    "textContent": "Halo {{name}}! Selamat datang di {{company_name}}.",
    "category": "onboarding",
    "isActive": true,
    "metadata": {
      "creator": "admin",
      "version": "1.0"
    },
    "createdAt": "2025-05-13T10:30:45Z",
    "updatedAt": "2025-05-13T10:30:45Z"
  }
}
```

### Memperbarui Template

Endpoint ini digunakan untuk memperbarui template yang sudah ada.

**URL**: `/api/v1/templates/:id`

**Metode**: `PUT`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**URL Parameters**:

- `id`: ID template yang ingin diperbarui

**Struktur Request**: Sama dengan endpoint pembuatan template

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Template updated successfully",
  "data": {
    "id": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
    "updatedAt": "2025-05-13T11:30:45Z"
  }
}
```

### Menghapus Template

Endpoint ini digunakan untuk menghapus template.

**URL**: `/api/v1/templates/:id`

**Metode**: `DELETE`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**URL Parameters**:

- `id`: ID template yang ingin dihapus

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Template deleted successfully",
  "data": null
}
```

### Mengirim Email dengan Template

Endpoint ini digunakan untuk mengirim email menggunakan template yang sudah ada.

**URL**: `/api/v1/emails/template`

**Metode**: `POST`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**Struktur Request**:

```json
{
  "templateId": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
  "to": ["recipient@example.com"],
  "cc": ["cc@example.com"],
  "bcc": ["bcc@example.com"],
  "variables": {
    "name": "John Doe",
    "company_name": "Acme Corp"
  },
  "attachments": [
    {
      "filename": "dokumen.pdf",
      "content": "base64-encoded-content",
      "contentType": "application/pdf"
    }
  ],
  "metadata": {
    "campaignId": "campaign123",
    "userId": "user456"
  }
}
```

**Fields**:

- `templateId` (wajib): ID template yang akan digunakan
- `to` (wajib): Array email penerima
- `cc` (opsional): Array email carbon copy
- `bcc` (opsional): Array email blind carbon copy
- `variables` (wajib): Objek berisi nilai untuk placeholder dalam template
- `attachments` (opsional): Array lampiran
- `metadata` (opsional): Metadata tambahan

**Response Sukses**: `202 Accepted`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Email queued for delivery",
  "data": {
    "email_id": "97a6fcde-e43a-4295-8e0e-3b8bc92eb456"
  }
}
```

## Placeholder dalam Template

Template email mendukung penggunaan placeholder dengan format `{{variable_name}}`. Placeholder ini akan digantikan dengan nilai yang diberikan dalam parameter `variables` saat mengirim email.

Contoh:

Template:

```html
<h1>Halo {{name}}!</h1>
<p>Selamat datang di {{company_name}}.</p>
```

Variables:

```json
{
  "name": "John Doe",
  "company_name": "Acme Corp"
}
```

Hasil:

```html
<h1>Halo John Doe!</h1>
<p>Selamat datang di Acme Corp.</p>
```

## Versioning Template

Saat memperbarui template, versi lama template tidak akan dihapus. Setiap perubahan akan menciptakan versi baru dan mempertahankan versi lama untuk audit trail. Untuk mengakses versi tertentu, gunakan endpoint `/api/v1/templates/:id/versions/:version`.
