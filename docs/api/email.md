# Email API

API ini memungkinkan pengiriman email kustom melalui layanan Email Microservice. Anda dapat mengirim email dengan konten yang sepenuhnya kustomisasi.

## Endpoints

### Mengirim Email

Endpoint ini digunakan untuk mengirim email kustom.

**URL**: `/api/v1/emails/manual`

**Metode**: `POST`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**Struktur Request**:

```json
{
  "to": ["recipient1@example.com", "recipient2@example.com"],
  "cc": ["cc1@example.com"],
  "bcc": ["bcc1@example.com"],
  "subject": "Subjek Email",
  "body": {
    "text": "Ini adalah email dalam format teks",
    "html": "<p>Ini adalah email dalam format <strong>HTML</strong></p>"
  },
  "attachments": [
    {
      "filename": "dokumen.pdf",
      "content": "base64-encoded-content",
      "contentType": "application/pdf"
    }
  ],
  "priority": "normal",
  "headers": {
    "X-Custom-Header": "nilai-kustom"
  },
  "metadata": {
    "campaignId": "campaign123",
    "userId": "user456"
  }
}
```

**Fields**:

- `to` (wajib): Array email penerima
- `cc` (opsional): Array email carbon copy
- `bcc` (opsional): Array email blind carbon copy
- `subject` (wajib): Subjek email
- `body` (wajib): Isi email
  - `text`: Versi teks biasa (plain text)
  - `html`: Versi HTML (opsional, tapi direkomendasikan)
- `attachments` (opsional): Array lampiran
  - `filename`: Nama file
  - `content`: Konten file dalam format base64
  - `contentType`: MIME type file
- `priority` (opsional): Prioritas email ('high', 'normal', 'low')
- `headers` (opsional): Header email kustom
- `metadata` (opsional): Metadata untuk keperluan tracking

**Response Sukses**: `202 Accepted`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Email queued for delivery",
  "data": {
    "email_id": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983"
  }
}
```

**Response Error**: `400 Bad Request`

```json
{
  "status": "error",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Validation error",
  "data": {
    "errors": [
      {
        "field": "to",
        "message": "Email recipient is required"
      }
    ]
  }
}
```

### Mendapatkan Status Email

Endpoint ini digunakan untuk memeriksa status pengiriman email.

**URL**: `/api/v1/emails/:id/status`

**Metode**: `GET`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**URL Parameters**:

- `id`: ID email yang ingin diperiksa statusnya

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Success",
  "data": {
    "id": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
    "status": "delivered",
    "sentAt": "2025-05-13T10:30:45Z",
    "deliveredAt": "2025-05-13T10:30:50Z",
    "failureReason": null,
    "attempts": 1,
    "provider": "aws-ses",
    "messageId": "aws-message-id-123456"
  }
}
```

**Response Error**: `404 Not Found`

```json
{
  "status": "error",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Email not found",
  "data": null
}
```

## Status Email

Email dapat memiliki status berikut:

- `pending`: Email sedang menunggu dalam antrian untuk dikirim
- `sending`: Email sedang dalam proses pengiriman
- `delivered`: Email berhasil dikirim ke server penerima
- `failed`: Email gagal dikirim setelah beberapa percobaan
- `bounced`: Email ditolak oleh server penerima
- `rejected`: Email ditolak oleh sistem (misalnya, karena alamat tidak valid)
- `complained`: Penerima melaporkan email sebagai spam

## Pertimbangan

1. Pengiriman email dilakukan secara asinkron (queue-based)
2. Email ID yang dikembalikan digunakan untuk melacak status email
3. Semua file lampiran (attachment) harus diencode dalam base64
4. File lampiran memiliki batasan ukuran maksimal 10MB (total)
