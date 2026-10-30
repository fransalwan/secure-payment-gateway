# Omni-Channel Mini E-Commerce OMS & Secure Payment Gateway

> **Production-Grade B2B Merchant Order Management System (OMS), Anti-Overselling Inventory Engine & Zero-Trust Payment Gateway**  
> Built with **Vue 3 + TypeScript + Pinia + `shadcn-vue` (`Reka UI` & `Vaul`)**, **Golang (Chi + pgx/v5)**, **PostgreSQL 16**, and **Redis 7**.

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue.svg)](https://golang.org)
[![Frontend](https://img.shields.io/badge/Frontend-Vue%203%20%2B%20TS%20%2B%20shadcn--vue-42b883.svg)](https://www.shadcn-vue.com/)
[![CI Pipeline](https://github.com/fransalwan/secure-payment-gateway/actions/workflows/ci.yml/badge.svg)](https://github.com/fransalwan/secure-payment-gateway/actions/workflows/ci.yml)
[![Database](https://img.shields.io/badge/Database-PostgreSQL%2016-blue.svg)](https://www.postgresql.org/)
[![Queue & Lock](https://img.shields.io/badge/Queue%20%26%20Lock-Redis%207-red.svg)](https://redis.io/)
[![Security](https://img.shields.io/badge/Security-AES--256--GCM%20%7C%20HMAC--SHA256-green.svg)](#-security--idempotency-guarantees)

---

## 📌 Executive Summary

Platform B2B Merchant yang menyatukan pesanan dari **Web Checkout**, **Marketplace Webhook (Shopee / TikTok Shop)**, dan **Social Commerce Chat (WhatsApp / Live Selling via AI)**, dilengkapi:
1. **AI Social Commerce Order Capture**: Ekstraksi chat WhatsApp/Live Selling tidak terstruktur menjadi draft order terpetakan ke SKU PostgreSQL (`< 3s` SLA + validasi stok real-time + *hybrid NLP fallback*).
2. **Multi-Channel 3-State Inventory & Anti-Overselling Engine**: Penguncian baris deterministik PostgreSQL (`SELECT ... FOR UPDATE` + `WHERE available_stock >= $1`) dengan 3 status stok (`available_stock`, `reserved_stock`, `sold_stock`) yang menjamin **nol stok minus (*zero overselling*)** saat *flash sale*.
3. **Secure Payment Gateway & Idempotent Webhook**: Verifikasi tanda tangan *constant-time* `HMAC-SHA256`, proteksi *replay attack* (`<= 5 menit`), deduplikasi atomik `UNIQUE(provider, external_event_id)` di `webhook_events` (`< 50ms` fast ACK), dan pembukuan akuntansi berpasangan (*Double-Entry Ledger*).
4. **Asynchronous Redis Queue, Exponential Backoff & DLQ**: Pemrosesan *Logistics Waybill Generator* (`JNE`, `J&T`, `SiCepat`), sinkronisasi stok marketplace, dan notifikasi WA dengan *Exponential Backoff* (`2s, 4s, 8s, 16s`) serta *Dead-Letter Queue* (`system_error_logs`).
5. **Modern Mobile-First `shadcn-vue` SaaS UI**: Arsitektur antarmuka **Vue 3 + TypeScript + Pinia + `Reka UI` + `Vaul`** dengan *Adaptive App Shell* (Desktop Sidebar & Mobile Bottom Bar), *Container-Query Responsive Table-to-Card*, *Command Palette (`Ctrl+K`)*, *Native CSS `light-dark()` Dual Theme*, **Microsecond Lock Waterfall Lab**, serta cetak resi **Thermal A6 (`100x150mm`)**.

---

## 📚 Engineering & Developer Handover Documentation

Repositori ini dilengkapi dokumentasi teknis terstruktur untuk mempermudah *onboarding* dan pengembangan lanjutan oleh *maintainer* berikutnya:

| Dokumen | Fokus & Isi Utama |
| :--- | :--- |
| **[`docs/ARCHITECTURE_AND_HANDOVER.md`](docs/ARCHITECTURE_AND_HANDOVER.md)** | Peta modul (*codebase map*), 5 *System Invariants ("Do Not Break" Rules)*, diagram ER & sekuens, *playbooks* penambahan fitur baru, serta *troubleshooting runbook*. |
| **[`docs/API_REFERENCE.md`](docs/API_REFERENCE.md)** | Spesifikasi kontrak REST API lengkap (request/response JSON, header keamanan HMAC, kode status HTTP `200`/`201`/`401`/`409`). |
| **[`AI_WORKFLOW.md`](AI_WORKFLOW.md)** | Standar kerja *Spec-First AI Engineering*, dekomposisi task, *prompt template*, dan *Pre-Merge Review Checklist*. |

---

## 🏗️ Arsitektur Sistem End-to-End

```
+-------------------------------------------------------------------------+
|     FRONTEND: Vue 3 + TypeScript + Pinia + shadcn-vue (Reka UI/Vaul)    |
|  [Mobile-First Shell] [Ctrl+K Command] [3-State Stock] [Waterfall Lab]  |
+-------------------+---------------------------------+-------------------+
                    | REST API (JSON)                 | Simulasi Webhook
                    v                                 v (HMAC-SHA256)
+-------------------------------------------------------------------------+
|                      BACKEND: Golang (Chi Router)                       |
|                                                                         |
|  1. Order, Inventory & AI Service   2. Webhook & Core Payment Switch    |
|     - Gemini JSON + NLP Fallback       - Verify Constant-Time HMAC      |
|     - Deterministic Lock Ordering      - Atomic `webhook_events` Guard  |
|     - PostgreSQL `FOR UPDATE` Tx       - Fast ACK (200 OK < 50ms)       |
+-------------------+---------------------------------+-------------------+
                    |                                 | Enqueue Job
       ACID Tx      v                                 v
+-----------------------------------+   +---------------------------------+
|      DATABASE: PostgreSQL 16      |   |       QUEUE: Redis 7            |
|  - merchants (Multi-Tenant)       |   |  - payment-webhook-queue        |
|  - products (avail/reserved/sold) |   |  - logistics-waybill-queue      |
|  - orders & order_items           |   |  - channel-stock-sync-queue     |
|  - webhook_events (Idempotency)   |   |  - notification-queue           |
|  - shipments (Waybill AWB)        |   +-----------------+---------------+
|  - system_error_logs (DLQ Audit)  |                     | Retry Backoff
|  - accounts & ledger_entries      |                     v (2s,4s,8s,16s)
+-----------------------------------+   +---------------------------------+
                                        |     BACKGROUND QUEUE WORKER     |
                                        |  - Settle Order -> PAID         |
                                        |  - Generate Courier Waybill AWB |
                                        |  - Sync Marketplace Stock       |
                                        +---------------------------------+
```

---

## 🎨 Modern Web & Mobile-First UI Architecture

* **`shadcn-vue` (`Reka UI`) + `Vaul` Adaptive Overlays**: Komponen `ResponsiveDialog` menyajikan *accessible modal / right slide-over sheet* di Desktop (`>= 768px`) dan otomatis bertransformasi menjadi **Swipeable Bottom Sheet** ber-unit `dvh` (*Dynamic Viewport Height*) di Mobile (`< 768px`) sehingga tidak terpotong oleh *virtual keyboard*.
* **CSS `@container` Query Responsive Tables**: Tabel pesanan, inventori, dan buku besar otomatis berubah dari tabel multi-kolom ber-*sticky header* menjadi **Actionable Mobile Fulfillment Cards** saat lebar kontainer `< 640px`.
* **Native CSS `light-dark()` Dual Theme & SPA View Transitions**: Mendukung tema *System / Light / Dark* bebas FOUC melalui `<meta name="color-scheme">` serta transisi mulus menggunakan `document.startViewTransition` dan notifikasi `vue-sonner`.

---

## 🛡️ Security & Idempotency Guarantees

* **Constant-Time `HMAC-SHA256` Verification**: Endpoint `/api/v1/webhooks/payment` dan `/api/v1/transfers` memvalidasi hash menggunakan `hmac.Equal` untuk mencegah *timing side-channel attacks*, disertai jendela waktu `±300` detik anti-*replay attack*.
* **Database-Level Webhook Idempotency**: Menggunakan `INSERT INTO webhook_events ... ON CONFLICT (provider, external_event_id) DO NOTHING RETURNING id` sehingga badai pengiriman ulang webhook identik langsung direspons `200 OK` (`{"status": "duplicate_ignored"}`) tanpa memotong stok dua kali.
* **One-Way Order State Machine**: Transisi ketat satu arah (`PENDING_PAYMENT` &rarr; `PAID` &rarr; `READY_TO_SHIP` &rarr; `SHIPPED` &rarr; `COMPLETED` atau `PENDING_PAYMENT` &rarr; `CANCELLED`).
* **Field-Level Encryption (`AES-256-GCM`) & Hash-Chained Audit Trail**: Data identitas akun dienkripsi secara autentik sebelum masuk ke PostgreSQL, dan setiap mutasi saldo diikat dalam rantai kriptografis `SHA-256`.

---

## 📡 Daftar REST API Endpoints Utama

| Method | Endpoint | Deskripsi & Logika Utama |
| :--- | :--- | :--- |
| `GET` | `/api/v1/dashboard/overview` | Mengambil state terpadu multi-merchant: daftar `orders`, stok 3-status `products`, `webhook_events`, metrik antrean Redis, dan `system_error_logs`. |
| `POST` | `/api/v1/orders/ai-parse` | Mengekstrak teks chat WA/Live Selling menjadi draft JSON order terpetakan ke SKU katalog beserta peringatan stok kurang (`< 3s`). |
| `POST` | `/api/v1/orders` | Membuat order baru dalam transaksi PostgreSQL (`SELECT ... FOR UPDATE`), memindahkan `available_stock` &rarr; `reserved_stock`, mengembalikan `201 Created` atau `409 Conflict`. |
| `POST` | `/api/v1/orders/{id}/cancel` | Membatalkan order `PENDING_PAYMENT` dan mengembalikan `reserved_stock` &rarr; `available_stock` secara atomik. |
| `POST` | `/api/v1/webhooks/payment` | Menerima webhook pembayaran (`X-Webhook-Id`, `X-Webhook-Timestamp`, `X-Webhook-Signature`), memvalidasi idempotency, mendorong job ke Redis Queue (`< 50ms`). |
| `POST` | `/api/v1/simulator/race-condition` | Menjalankan simulasi `N` pembeli konkuren yang memperebutkan 1 stok terakhir secara simultan. |
| `POST` | `/api/v1/simulator/duplicate-webhook` | Menembakkan 5 webhook identik beruntun untuk membuktikan deduplikasi idempotent. |
| `POST` | `/api/v1/simulator/chaos-mode` | Mengatur simulasi kegagalan API logistik pihak ketiga (`OFF`, `TRANSIENT_TWICE`, `ALWAYS_429`) untuk menguji *Exponential Backoff* & DLQ. |
| `POST` | `/api/v1/queue/jobs/{id}/retry` | Melakukan *Manual Retry* atas job yang berada di status `FAILED / DLQ`. |
| `POST` | `/api/v1/transfers` | Transfer dana atomik *Double-Entry Ledger* berproteksi `HMAC-SHA256` & Redis Idempotency Lock. |
| `POST` | `/api/v1/reconciliation/upload` | Pencocokan mutasi bank *3-Way Settlement Reconciliation* berbasis CSV stream. |

---

## 🚀 Panduan Menjalankan & Pengujian

### 1. Prasyarat
* **Go 1.22+**
* **Node.js 20+** (untuk kompilasi source Vue 3 + TypeScript di `frontend/`)
* **PostgreSQL 16** & **Redis 7** (via Docker Compose atau instalasi lokal)

### 2. Jalankan Database & Redis
```bash
docker compose up -d
```

### 3. Build Frontend Vue 3 (Opsional — bundle produksi sudah tersedia di `web/dist`)
```bash
cd frontend
npm install
npm run build
```

### 4. Jalankan Automated Critical Workflow & Race Detector Tests
```bash
go test -v -race ./...
```

### 5. Jalankan Server Aplikasi
```bash
go run ./cmd/api
```
Akses **Operational Dashboard & Simulator UI** di `http://localhost:8080`.

### 6. Live Remote Demo via Public Tunnel
```bash
make tunnel
# Atau di PowerShell:
.\scripts\tunnel.ps1
```

---

## 👨‍💻 Author & Engineering Contact

* **Engineer**: Frans Alwan Purba
* **Email**: [fransalwan55@gmail.com](mailto:fransalwan55@gmail.com)
* **GitHub**: [github.com/fransalwan](https://github.com/fransalwan)
* **Education**: Master of Artificial Intelligence @ Universitas Gadjah Mada (UGM)
