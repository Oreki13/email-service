# Tracking API

API ini memungkinkan pelacakan dan analisis pengiriman email. Dengan Tracking API, Anda dapat memantau status email, mengetahui ketika email dibuka, tautan diklik, dan memperoleh statistik tentang kampanye email Anda.

## Endpoints

### Mendapatkan Status Email

Endpoint ini digunakan untuk mendapatkan status terkini dari sebuah email.

**URL**: `/api/v1/tracking/:email_id`

**Metode**: `GET`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**URL Parameters**:

- `email_id`: ID email yang ingin dilacak

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Success",
  "data": {
    "emailId": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
    "status": "delivered",
    "events": [
      {
        "type": "queued",
        "timestamp": "2025-05-13T10:30:40Z",
        "metadata": {}
      },
      {
        "type": "sent",
        "timestamp": "2025-05-13T10:30:45Z",
        "metadata": {
          "provider": "aws-ses",
          "messageId": "aws-message-id-123456"
        }
      },
      {
        "type": "delivered",
        "timestamp": "2025-05-13T10:30:50Z",
        "metadata": {
          "recipient": "recipient@example.com"
        }
      },
      {
        "type": "opened",
        "timestamp": "2025-05-13T10:35:20Z",
        "metadata": {
          "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
          "ipAddress": "192.168.1.1"
        }
      },
      {
        "type": "clicked",
        "timestamp": "2025-05-13T10:35:30Z",
        "metadata": {
          "url": "https://example.com/promo",
          "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
          "ipAddress": "192.168.1.1"
        }
      }
    ]
  }
}
```

### Mendapatkan Statistik Email

Endpoint ini digunakan untuk mendapatkan statistik agregat dari satu atau beberapa email.

**URL**: `/api/v1/tracking/stats`

**Metode**: `GET`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**Query Parameters**:

- `emailId` (opsional): ID email spesifik
- `campaignId` (opsional): ID kampanye
- `startDate` (opsional): Tanggal awal (format ISO8601)
- `endDate` (opsional): Tanggal akhir (format ISO8601)
- `templateId` (opsional): ID template

**Response Sukses**: `200 OK`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Success",
  "data": {
    "stats": {
      "total": 100,
      "delivered": 95,
      "failed": 5,
      "opened": 75,
      "clicked": 30,
      "bounced": 3,
      "complained": 2,
      "deliveryRate": 95.0,
      "openRate": 78.9,
      "clickRate": 40.0,
      "clickToOpenRate": 40.0
    },
    "topLinks": [
      {
        "url": "https://example.com/promo",
        "clicks": 20
      },
      {
        "url": "https://example.com/landing",
        "clicks": 10
      }
    ]
  }
}
```

### Melacak Pembukaan Email

Endpoint ini digunakan untuk melacak pembukaan email melalui pixel tracking. Endpoint ini biasanya dipanggil secara otomatis oleh pixel yang disematkan dalam email.

**URL**: `/api/v1/tracking/open/:tracking_id`

**Metode**: `GET`

**Auth diperlukan**: Tidak

**URL Parameters**:

- `tracking_id`: ID tracking unik yang disematkan dalam email

**Response**: Pixel GIF transparan 1x1

### Melacak Klik Tautan

Endpoint ini digunakan untuk melacak klik pada tautan dalam email. Endpoint ini akan menerima klik dan mengarahkan pengguna ke URL tujuan asli.

**URL**: `/api/v1/tracking/click/:tracking_id`

**Metode**: `GET`

**Auth diperlukan**: Tidak

**URL Parameters**:

- `tracking_id`: ID tracking unik yang disematkan dalam tautan email

**Query Parameters**:

- `url`: URL tujuan yang telah dienkripsi

**Response**: Redirect 302 ke URL tujuan

### Webhook untuk Event Tracking

Endpoint ini memungkinkan layanan lain untuk menerima notifikasi real-time tentang event tracking email.

**URL**: `/api/v1/tracking/webhooks`

**Metode**: `POST`

**Auth diperlukan**: Ya (API Key di header `X-API-Key`)

**Struktur Request**:

```json
{
  "url": "https://yourapplication.com/email-events",
  "events": ["delivered", "opened", "clicked", "bounced", "complained"],
  "metadata": {
    "description": "Production webhook",
    "environment": "production"
  },
  "headers": {
    "X-Custom-Auth": "your-secret-token"
  }
}
```

**Fields**:

- `url` (wajib): URL yang akan menerima notifikasi webhook
- `events` (wajib): Array jenis event yang ingin diterima
- `metadata` (opsional): Metadata untuk webhook
- `headers` (opsional): Header HTTP tambahan yang akan disertakan dalam request webhook

**Response Sukses**: `201 Created`

```json
{
  "status": "success",
  "traceID": "65d4e7a3-8c2f-49d1-9024-e41a8c4856f3",
  "message": "Webhook created successfully",
  "data": {
    "id": "webhook-id-123456",
    "secret": "wehook-verification-secret"
  }
}
```

## Jenis Event Tracking

Sistem tracking mendukung jenis event berikut:

- `queued`: Email masuk dalam antrian untuk dikirim
- `sent`: Email telah dikirim ke server email penerima
- `delivered`: Email telah berhasil diterima oleh server penerima
- `opened`: Email telah dibuka oleh penerima
- `clicked`: Tautan dalam email telah diklik
- `bounced`: Email dikembalikan oleh server penerima
- `complained`: Penerima menandai email sebagai spam
- `failed`: Pengiriman email gagal

## Format Data Webhook

Ketika event terjadi, sistem akan mengirim payload berikut ke URL webhook yang terdaftar:

```json
{
  "event": "opened",
  "timestamp": "2025-05-13T10:35:20Z",
  "emailId": "87f6bcde-f43a-4295-8e0e-3b8bc92eb983",
  "recipient": "recipient@example.com",
  "templateId": "template-id-123456",
  "campaignId": "campaign-id-123456",
  "metadata": {
    "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
    "ipAddress": "192.168.1.1"
  },
  "signature": "webhook-signature-for-verification"
}
```

## Verifikasi Webhook

Untuk memastikan keamanan webhook, signature disertakan dalam setiap request. Untuk memverifikasi signature:

1. Ambil header `X-Webhook-Signature` dari request
2. Gabungkan timestamp dan payload mentah, lalu buat HMAC menggunakan SHA-256 dan webhook secret
3. Verifikasi bahwa signature yang dihitung cocok dengan signature dalam header

Contoh kode verifikasi (pseudocode):

```
hmacSignature = HMAC-SHA256(webhookSecret, timestamp + rawPayload)
isValid = (hmacSignature === requestSignature)
```
