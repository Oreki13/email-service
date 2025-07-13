# Web UI untuk Email Service

Web UI ini menyediakan interface untuk mengelola template email dalam microservice email dengan \*\*dark theme## Tailwind Configuration

Konfigurasi Tailwind dengan custom dark color palette:

```javascript
tailwind.config = {
  theme: {
    extend: {
      colors: {
        primary: {
          50: "#1e293b",
          400: "#60a5fa",
          500: "#3b82f6",
          600: "#2563eb",
          700: "#1d4ed8",
        },
        dark: {
          50: "#f8fafc",
          100: "#f1f5f9",
          200: "#e2e8f0",
          300: "#cbd5e1",
          400: "#94a3b8",
          500: "#64748b",
          600: "#475569",
          700: "#334155",
          800: "#1e293b",
          900: "#0f172a",
        },
      },
    },
  },
};
```

## Dark Theme Implementation

### Background Colors

- `bg-dark-900`: Background utama halaman
- `bg-dark-800`: Background card dan panel
- `bg-dark-700`: Background input dan dropdown

### Text Colors

- `text-white`: Text utama
- `text-gray-300`: Text label dan secondary
- `text-gray-400`: Text placeholder dan muted
- `text-primary-400`: Link dan accent text

### Border Colors

- `border-dark-700`: Border utama
- `border-dark-600`: Border input dan form elements

### Hover States

- `hover:bg-dark-700`: Hover background untuk buttons
- `hover:bg-dark-600`: Hover background untuk inputs
- `hover:text-white`: Hover text untuk navigation elegan.

## Fitur

- **Dashboard**: Menampilkan statistik dan quick actions dengan dark theme
- **Template Management**: CRUD operations untuk email template dengan styling gelap
- **Authentication**: Login dengan username/password dari environment
- **Dark Theme**: Interface gelap yang nyaman untuk mata
- **Responsive Design**: UI responsif menggunakan Tailwind CSS dengan dark colors

## Teknologi yang Digunakan

- **Frontend**: HTML templates dengan Tailwind CSS (dark theme)
- **Backend**: Go dengan Fiber framework
- **Template Engine**: Go HTML templates
- **Authentication**: Session-based auth dengan Fiber middleware
- **Styling**: Tailwind CSS dengan custom dark color palette

## Dark Theme Colors

- **Background**: dark-900 (#0f172a) - Background utama
- **Cards/Panels**: dark-800 (#1e293b) - Panel dan card
- **Borders**: dark-700 (#334155) - Border dan divider
- **Text Primary**: white - Text utama
- **Text Secondary**: gray-300/400 - Text sekunder
- **Primary Color**: blue-500 (#3b82f6) - Accent color
- **Success**: green-400/500 - Status success
- **Error**: red-400/500 - Error states
- **Warning**: yellow-400/500 - Warning states

## Struktur File

```
web/
├── templates/           # HTML templates dengan dark theme
│   ├── login.html       # Halaman login dark
│   ├── dashboard.html   # Dashboard dengan sidebar gelap
│   ├── template_list.html    # Daftar template dengan table gelap
│   ├── template_create.html  # Form create dengan input gelap
│   ├── template_edit.html    # Form edit dengan styling gelap
│   └── template_detail.html  # Detail template dengan dark styling
└── README.md           # Dokumentasi ini
```

## Setup dan Konfigurasi

### 1. Environment Variables

Tambahkan variabel berikut ke file `.env`:

```bash
# Web UI Configuration
WEB_UI_ENABLED=true
WEB_UI_PORT=8081
WEB_UI_USERNAME=admin
WEB_UI_PASSWORD=your_secure_password
WEB_UI_SESSION_SECRET=your_session_secret_key
```

### 2. Template Engine

Template engine sudah dikonfigurasi di `cmd/server/server.go`:

```go
// Template engine
engine := html.New("./web/templates", ".html")
engine.Reload(true) // Reload templates in development
app.Views(engine)

// Session middleware
app.Use(session.New(session.Config{
    KeyLookup: "cookie:session_id",
    CookieSecure: false, // Set to true in production
    CookieHTTPOnly: true,
    CookieSameSite: "Lax",
    Expiration: 24 * time.Hour,
}))
```

## Penggunaan

### 1. Akses Web UI

Buka browser dan navigasi ke:

```
http://localhost:8081
```

### 2. Login

Gunakan credentials yang telah dikonfigurasi di environment variables:

- Username: `admin` (atau sesuai WEB_UI_USERNAME)
- Password: (sesuai WEB_UI_PASSWORD)

### 3. Navigasi

- **Dashboard**: Overview dan quick actions
- **Templates**: Kelola email templates
  - View list dengan pagination dan search
  - Create new template
  - Edit existing template
  - View template details
  - Delete template

## Development

### Local Development

1. Jalankan server:

```bash
go run main.go server
```

2. Akses web UI di `http://localhost:8081`

### Template Editing

Template menggunakan Go HTML template syntax. Perubahan template akan otomatis reload dalam development mode.

### Styling dengan Tailwind CSS

UI menggunakan Tailwind CSS yang dimuat via CDN. Komponen utama:

- **Layout**: Flexbox dan Grid untuk responsive design
- **Navigation**: Fixed navbar dengan dropdown menu
- **Sidebar**: Navigasi vertikal dengan active states
- **Forms**: Styled input dan textarea dengan validation states
- **Tables**: Responsive table dengan pagination
- **Modals**: Overlay modals untuk konfirmasi dan preview
- **Buttons**: Berbagai varian button dengan hover states

### Custom Tailwind Configuration

Konfigurasi Tailwind yang digunakan:

```javascript
tailwind.config = {
  theme: {
    extend: {
      colors: {
        primary: {
          50: "#eff6ff",
          500: "#3b82f6",
          600: "#2563eb",
          700: "#1d4ed8",
        },
      },
    },
  },
};
```

## API Integration

Web UI terintegrasi dengan API backend melalui form submission dan akan berkomunikasi dengan:

- **Template Service**: Untuk CRUD operations template
- **Email Service**: Untuk send test email
- **Statistics**: Untuk dashboard metrics

## Security

- Session-based authentication
- CSRF protection (perlu diimplementasi)
- Input validation dan sanitization
- Secure cookie configuration untuk production

## Production Deployment

Untuk production:

1. Set environment variables yang sesuai
2. Aktifkan HTTPS
3. Set `CookieSecure: true` untuk session
4. Gunakan session secret yang kuat
5. Pertimbangkan menggunakan reverse proxy (nginx)

## Troubleshooting

### Template tidak reload

Pastikan `engine.Reload(true)` diset dalam development.

### Session issues

Periksa session secret dan cookie configuration.

### Styling issues

Pastikan Tailwind CSS dimuat dengan benar via CDN.

## Kontribusi

Untuk menambah fitur atau memperbaiki bug:

1. Ikuti struktur template yang sudah ada
2. Gunakan Tailwind CSS untuk styling
3. Pastikan responsive design
4. Test di berbagai browser
5. Update dokumentasi jika diperlukan
