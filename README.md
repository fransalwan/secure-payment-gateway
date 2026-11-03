# AI-Powered Omni-Channel FinTech OMS & Secure Payment Gateway

> **Production-Grade B2B Merchant Order Management System (OMS), Anti-Overselling Inventory Engine, Zero-Trust Payment Switch & AI Workflows**  
> Built with **Golang (Chi + pgx/v5)**, **RabbitMQ (AMQP 0-9-1)**, **Redis 7**, **PostgreSQL 16**, **Gemini 1.5 Multimodal AI**, **`rs/zerolog`**, and **Vue 3 + TypeScript + Pinia + `shadcn-vue` (`Reka UI` & `Vaul`)**.

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue.svg)](https://golang.org)
[![RabbitMQ](https://img.shields.io/badge/Message%20Broker-RabbitMQ%203.13-orange.svg)](https://www.rabbitmq.com/)
[![Frontend](https://img.shields.io/badge/Frontend-Vue%203%20%2B%20TS%20%2B%20shadcn--vue-42b883.svg)](https://www.shadcn-vue.com/)
[![CI Pipeline](https://github.com/fransalwan/secure-payment-gateway/actions/workflows/ci.yml/badge.svg)](https://github.com/fransalwan/secure-payment-gateway/actions/workflows/ci.yml)
[![Database](https://img.shields.io/badge/Database-PostgreSQL%2016-blue.svg)](https://www.postgresql.org/)
[![Queue & Lock](https://img.shields.io/badge/Queue%20%26%20Lock-Redis%207-red.svg)](https://redis.io/)
[![Observability](https://img.shields.io/badge/Observability-rs%2Fzerolog-brightgreen.svg)](https://github.com/rs/zerolog)
[![Security](https://img.shields.io/badge/Security-AES--256--GCM%20%7C%20HMAC--SHA256-green.svg)](#-security--idempotency-guarantees)

---

## 📌 Executive Summary

Platform B2B Merchant FinTech yang menyatukan pesanan dari **Web Checkout**, **Marketplace Webhook (Shopee / TikTok Shop)**, **Manual Bank Transfer**, dan **Social Commerce Chat (WhatsApp / Live Selling via AI)**, dilengkapi:
1. **Asynchronous Vision-AI Bank Slip Validator & pHash Anti-Fraud**: Validasi bukti transfer bank (BCA, Mandiri, BRI, BNI, QRIS) berbasis Gemini 1.5 Multimodal Vision AI dengan deteksi keaslian gambar visual (*tamper check*) dan **Perceptual Hash (dHash 64-bit)** untuk mendeteksi struk editan/crop duplikat (< 10ms). Transisi status atomik ACID (`PAID` vs `SUSPICIOUS_HOLD`) via fast ACK (`202 Accepted`).
2. **Transactional Outbox Pattern & RabbitMQ Worker Pipeline**: Eliminasi total *Dual-Write Problem* antara PostgreSQL dan broker event. Data bisnis dan event ditulis dalam transaksi ACID atomik di tabel `outbox_events`, lalu didispatch secara konkuren via `SELECT ... FOR UPDATE SKIP LOCKED` ke RabbitMQ Topic Exchange `oms.events`. Pemisahan eksekusi AI/LLM lambat (2-4s) via *Worker Goroutines Pool* (QoS prefetch 10), context timeout (3.5s), *Exponential Backoff Retry* (`2s, 4s, 8s`), dan *Dead-Letter Exchange (`oms.dlx`)*.
3. **Enterprise Semantic Metrics Layer & Natural Language Text-to-SQL**: Mesin pelaporan analitik bisnis (*Business Intelligence*) berbasis Gemini LLM dengan *Semantic Metric Layer* (rumus baku GMV, Net Revenue, Receivables, Quarantine Volume), AST Token Sanitizer anti-SQL Injection deterministik, batas eksekusi read-only PostgreSQL (`statement_timeout = 2500ms`), dan visualisasi grafik otomatis.
4. **AI FinOps & Observability Cockpit (`/api/v1/ai/telemetry`)**: Telemetri produksi real-time yang memantau konsumsi token (Vision & SQL), estimasi biaya model dalam USD ($), rata-rata latensi inferensi vs SLA Fast-ACK (< 20ms), rasio karantina penipuan (*fraud quarantine rate*), dan monitoring kesehatan status antrean *Transactional Outbox*.
5. **Multi-Channel 3-State Inventory & Anti-Overselling Engine**: Penguncian baris deterministik PostgreSQL (`SELECT ... FOR UPDATE` + `WHERE available_stock >= $1`) dengan 3 status stok (`available_stock`, `reserved_stock`, `sold_stock`) yang menjamin **nol stok minus (*zero overselling*)** saat *flash sale*.
6. **Secure Payment Switch & Idempotent Webhook**: Verifikasi tanda tangan *constant-time* `HMAC-SHA256`, proteksi *replay attack* (`<= 5 menit`), deduplikasi atomik `UNIQUE(provider, external_event_id)` di `webhook_events` (`< 50ms` fast ACK), dan pembukuan akuntansi berpasangan (*Double-Entry Ledger*).
7. **Structured Logging & Enterprise Tracing (`rs/zerolog`)**: Structured logging berkinerja tinggi berbasis Zero Allocation JSON, injeksi Trace ID terpadu lintas HTTP request dan RabbitMQ event, serta audit trail kepatuhan FinTech.
8. **Modern Mobile-First `shadcn-vue` SaaS UI & Recruiter Guided Sandbox**: Arsitektur antarmuka **Vue 3 + TypeScript + Pinia + `Reka UI` + `Vaul`** dengan *Recruiter Guided Cockpit* (4 pill interaktif 1-klik untuk live demo), *Command Palette (`Ctrl+K`)*, *Payment Slip AI Inspector*, *Natural Language Text-to-SQL Console*, **Microsecond Lock Waterfall Lab**, serta cetak resi **Thermal A6 (`100x150mm`)**.

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
+-----------------------------------------------------------------------------------------+
|             FRONTEND: Vue 3 + TypeScript + Pinia + shadcn-vue (Reka UI/Vaul)            |
|  [Recruiter Guided Cockpit] [AI FinOps Telemetry] [Payment Slip AI] [Text-to-SQL BI]    |
+--------------------+-----------------------------------+--------------------------------+
                     | REST API (JSON / Multipart)       | Simulasi Webhook (HMAC)
                     v                                   v
+-----------------------------------------------------------------------------------------+
|                      BACKEND: Golang (Chi Router + rs/zerolog)                          |
|                                                                                         |
|  1. Ingestion & pHash Pre-Filter   2. Safe BI Query Engine       3. AI FinOps Telemetry |
|     - 64-bit dHash (Hamming <= 5)     - Grounded Semantic Rules     - Token & Cost Track|
|     - Constant-Time HMAC-SHA256       - AST Token Security          - P99 vs SLA Metric |
|     - Fast ACK (< 20ms, 202/200)      - Read-Only Tx 2500ms         - GET /ai/telemetry |
+--------------------+--------------------------------------------------------------------+
                     | ACID Dual-Write Protection (Atomic Insert)
                     v
+------------------------------------+
|      DATABASE: PostgreSQL 16       |
|  - outbox_events (PENDING/DONE)    |--------+ (Worker Polling: SELECT FOR UPDATE SKIP LOCKED)
|  - payment_proofs (image_phash)    |        v
|  - ai_telemetry_events             |   +------------------------------------------------+
|  - products (3-state inventory)    |   |         MESSAGE BROKER: RabbitMQ 3.13          |
|  - orders & ledger_entries         |   |  - Exchange: oms.events (Topic)                |
+------------------------------------+   |  - Queues: order.parse.queue,                  |
                     ^                   |            payment.verify.proof                |
                     |                   |  - Dead-Letter Exchange: oms.dlx -> oms.dlq   |
                     |                   +-----------------------+------------------------+
                     | ACID Pessimistic                          | QoS Prefetch (10)
                     | Row Locking                               v
                     |                     +----------------------------------------------+
                     |                     |       BACKGROUND WORKER GOROUTINES POOL      |
                     +---------------------+  - Multimodal Gemini Vision Slip Validator   |
                                           |  - Tamper Detection & Confidence Scoring     |
                                           |  - Atomic Settle -> PAID / SUSPICIOUS_HOLD   |
                                           |  - Exponential Backoff (2s, 4s, 8s) & DLX    |
                                           +----------------------------------------------+
```

---

## 🎨 Modern Web & Mobile-First UI Architecture

* **Recruiter & Tech Lead Guided Sandbox Cockpit**: 4 pill interaktif 1-klik di bagian atas dashboard untuk demo instan dalam 60 detik:
  1. `Struk Valid (202 Fast-ACK)`: Verifikasi bukti bayar asli BCA dengan respon cepat `< 20ms`.
  2. `Struk Duplikat (pHash Test)`: Deteksi fraud instan struk duplikat via 64-bit dHash tanpa membuang biaya token LLM.
  3. `Text-to-SQL BI (GMV Matrix)`: Pertanyaan analitik bisnis ter-grounding dengan aturan semantik metrik dan proteksi AST.
  4. `AI FinOps Cockpit (Cost & Latency)`: Panel observabilitas metrik token, estimasi biaya FinOps ($), dan latensi SLA.
* **`shadcn-vue` (`Reka UI`) + `Vaul` Adaptive Overlays**: Komponen `ResponsiveDialog` menyajikan *accessible modal / right slide-over sheet* di Desktop (`>= 768px`) dan otomatis bertransformasi menjadi **Swipeable Bottom Sheet** ber-unit `dvh` (*Dynamic Viewport Height*) di Mobile (`< 768px`) sehingga tidak terpotong oleh *virtual keyboard*.
* **Payment Slip AI Inspector Modal**: Modal interaktif dengan *drag-drop file upload*, simulator instan 1-klik (Struk BCA Valid, Struk Editan/Manipulasi, Mismatch Nominal, dan Struk Duplikat pHash), deteksi *fraud* visual, serta sinkronisasi status otomatis.
* **Natural Language Text-to-SQL Reporting Panel**: Konsol pelaporan bisnis berbasis bahasa alami dengan preset query finansial, indikator validasi keamanan AST, format hasil tabel dinamis, dan rekomendasi chart.
* **AI FinOps & Observability Cockpit Panel**: Monitoring langsung konsumsi token per modalitas, estimasi biaya operasional USD ($), rasio karantina penipuan, dan kesehatan Transactional Outbox.
* **CSS `@container` Query Responsive Tables**: Tabel pesanan, inventori, dan buku besar otomatis berubah dari tabel multi-kolom ber-*sticky header* menjadi **Actionable Mobile Fulfillment Cards** saat lebar kontainer `< 640px`.
* **Native CSS `light-dark()` Dual Theme & SPA View Transitions**: Mendukung tema *System / Light / Dark* bebas FOUC melalui `<meta name="color-scheme">` serta transisi mulus menggunakan `document.startViewTransition` dan notifikasi `vue-sonner`.

---

## 🛡️ Security & Idempotency Guarantees

* **Constant-Time `HMAC-SHA256` Verification**: Endpoint `/api/v1/webhooks/payment` dan `/api/v1/transfers` memvalidasi hash menggunakan `hmac.Equal` untuk mencegah *timing side-channel attacks*, disertai jendela waktu `±300` detik anti-*replay attack*.
* **Database-Level Webhook Idempotency**: Menggunakan `INSERT INTO webhook_events ... ON CONFLICT (provider, external_event_id) DO NOTHING RETURNING id` sehingga badai pengiriman ulang webhook identik langsung direspons `200 OK` (`{"status": "duplicate_ignored"}`) tanpa memotong stok dua kali.
* **One-Way Order State Machine**: Transisi ketat satu arah (`PENDING_PAYMENT` &rarr; `PAID` &rarr; `READY_TO_SHIP` &rarr; `SHIPPED` &rarr; `COMPLETED`, `PENDING_PAYMENT` &rarr; `CANCELLED`, atau `PENDING_PAYMENT` &rarr; `SUSPICIOUS_HOLD` untuk review fraud).
* **AST SQL Security Guardrail & Read-Only Transactions**: Endpoint pelaporan AI memverifikasi token SQL secara deterministik (menolak titik-koma ganda, `DROP`, `DELETE`, `UPDATE`, `INSERT`, `ALTER`, `GRANT`, `EXEC`) dan menjalankannya dalam transaksi `SET TRANSACTION READ ONLY` dengan batas timeout ketat 2.500 ms.
* **Field-Level Encryption (`AES-256-GCM`) & Hash-Chained Audit Trail**: Data identitas akun dienkripsi secara autentik sebelum masuk ke PostgreSQL, dan setiap mutasi saldo diikat dalam rantai kriptografis `SHA-256`.

---

## 📡 Daftar REST API Endpoints Utama

| Method | Endpoint | Deskripsi & Logika Utama |
| :--- | :--- | :--- |
| `GET` | `/api/v1/dashboard/overview` | Mengambil state terpadu multi-merchant: daftar `orders`, stok 3-status `products`, `webhook_events`, metrik antrean Redis, dan `system_error_logs`. |
| `POST` | `/api/v1/payments/proof` | Mengunggah bukti transfer bank (multipart JPEG/PNG atau Base64), validasi kilat, respons `202 Accepted` (< 20ms), dan publish event ke RabbitMQ `payment.verify.proof`. |
| `GET` | `/api/v1/payments/proof/{id}` | Mengambil status verifikasi struk Vision AI (ekstraksi bank, nominal, nomor referensi, status tamper, dan skor keyakinan). |
| `GET` | `/api/v1/payments/proof` | Mengambil daftar riwayat bukti transfer bank dan status verifikasinya. |
| `POST` | `/api/v1/reports/ask-ai` | Menjalankan analitik pelaporan bisnis berbasis bahasa alami (Text-to-SQL) dengan AST security validation, batas timeout 2500ms, dan hasil matriks tabular. |
| `GET` | `/api/v1/ai/telemetry` | Mengambil metrik observabilitas AI, FinOps token usage, pHash duplicate detection, dan kesehatan Transactional Outbox. |
| `POST` | `/api/v1/orders/ai-parse` | Mengekstrak teks chat WA/Live Selling menjadi draft JSON order terpetakan ke SKU katalog beserta peringatan stok kurang (`< 3s`). |
| `POST` | `/api/v1/orders` | Membuat order baru dalam transaksi PostgreSQL (`SELECT ... FOR UPDATE`), memindahkan `available_stock` &rarr; `reserved_stock`, mengembalikan `201 Created` atau `409 Conflict`. |
| `POST` | `/api/v1/orders/{id}/cancel` | Membatalkan order `PENDING_PAYMENT` dan mengembalikan `reserved_stock` &rarr; `available_stock` secara atomik. |
| `POST` | `/api/v1/webhooks/payment` | Menerima webhook pembayaran (`X-Webhook-Id`, `X-Webhook-Timestamp`, `X-Webhook-Signature`), memvalidasi idempotency, mendorong job ke antrean (`< 50ms`). |
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
* **PostgreSQL 16**, **Redis 7**, dan **RabbitMQ 3.13** (tersedia *out-of-the-box* via `docker compose`)

### 2. Jalankan Database, Redis, dan RabbitMQ
```bash
docker compose up -d
```
*(RabbitMQ Management Console dapat diakses di `http://localhost:15672` dengan kredensial `guest`/`guest`).*

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

### 7. Production Multi-Container Deployment (Docker Compose + Auto-HTTPS Caddy)
Repository ini sudah dilengkapi [`Dockerfile`](Dockerfile) *multi-stage* (Node 22 + Go Alpine + *non-root runtime*), [`docker-compose.prod.yml`](docker-compose.prod.yml) dengan pemisahan container `oms-api` (`ENABLE_EMBEDDED_WORKER=false`) dan `oms-worker`, serta reverse proxy `Caddy` untuk HTTPS otomatis:

```bash
chmod +x scripts/deploy-vps.sh
./scripts/deploy-vps.sh oms.yourdomain.my.id
```

Lihat panduan lengkap provisioning VPS lokal (QRIS), custom domain, dan Cloudflare WAF di [`docs/DEPLOYMENT_PRODUCTION.md`](docs/DEPLOYMENT_PRODUCTION.md).

---

## 👨‍💻 Author & Engineering Contact

* **Engineer**: Frans Alwan Purba
* **Email**: [fransalwan55@gmail.com](mailto:fransalwan55@gmail.com)
* **GitHub**: [github.com/fransalwan](https://github.com/fransalwan)
* **Education**: Master of Artificial Intelligence @ Universitas Gadjah Mada (UGM)
