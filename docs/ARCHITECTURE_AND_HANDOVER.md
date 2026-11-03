# Architecture, System Invariants & Developer Handover Guide

> **Engineering Maintenance & Onboarding Manual**  
> Dokumen ini ditulis sebagai panduan serah-terima teknis (*engineering handover*) agar developer selanjutnya dapat memahami arsitektur, batasan kritis (*system invariants*), alur data, serta prosedur pengembangan lanjutan tanpa merusak integritas stok maupun buku besar pembayaran.

---

## 1. Gambaran Umum & Bounded Contexts

Sistem ini menggabungkan tiga *bounded context* utama dalam satu *monolith modular* berbasis event-driven berbahasa **Go (`chi/v5`, `pgx/v5`, `amqp091-go`, `go-redis/v9`, `rs/zerolog`)** dengan antarmuka operasional **Vue 3 + TypeScript + Pinia**:

1. **Bounded Context A — Omni-Channel E-Commerce OMS (`oms_*`)**
   * Menangani *order capture* lintas kanal (`WEB`, `SHOPEE`, `TIKTOK_SHOP`, `WHATSAPP_LIVE`), ekstraksi chat sosial berbasis AI, manajemen inventori 3-status anti-*overselling*, penerima webhook pembayaran idempotent, serta *background worker* pencetakan nomor resi (*waybill*) dan sinkronisasi stok.
2. **Bounded Context B — Core Payment Switch & Financial Ledger (`account_*`, `transfer_*`, `reconciliation_*`)**
   * Menangani brankas akun dengan enkripsi identitas *field-level* (`AES-256-GCM`), transfer dana atomik berpasangan (*Double-Entry Bookkeeping*), rantai audit kriptografis (`SHA-256 Hash Chain`), *Transactional Outbox*, dan rekonsiliasi mutasi bank 3-arah (*3-Way CSV Reconciliation*).
3. **Bounded Context C — AI Multimodal Slip Verification & Text-to-SQL Reporting (`payment_proofs`, `ai_reporting_*`)**
   * Menangani validasi bukti transfer bank via **RabbitMQ** dan **Gemini 1.5 Flash Vision AI** dengan deteksi manipulasi gambar (*tamper check*), transisi pesanan ACID atomik ke `PAID` atau `SUSPICIOUS_HOLD`, serta antarmuka laporan analitik bisnis berbasis bahasa alami dengan *AST SQL Token Sanitizer* dan read-only transaction guardrail.

```mermaid
flowchart TB
    subgraph Client["Frontend Layer (Vue 3 + TypeScript + Pinia)"]
        UI_OMS["4-Panel Operational OMS UI"]
        UI_LEDGER["Core Payment Switch & Recon UI"]
        UI_SLIP["Vision AI Slip Verification Modal"]
        UI_BI["Natural Language Ask-AI Reporting Console"]
    end

    subgraph HTTP["Transport Layer (cmd/api & internal/handler/http)"]
        MW["CORS + RequestID + Zerolog Trace + HMAC Auth"]
        H_OMS["OMSHandler (Orders, Webhooks, Simulators)"]
        H_PAY["Account / Transfer / Reconciliation Handlers"]
        H_PROOF["PaymentProofHandler (POST /proof 202 Fast ACK)"]
        H_REPORT["ReportHandler (POST /reports/ask-ai)"]
    end

    subgraph Broker["Message Broker & Asynchronous Pipeline (RabbitMQ 3.13)"]
        EX_OMS["Topic Exchange: oms.events"]
        Q_PARSE["Queue: order.parse.queue"]
        Q_VERIFY["Queue: payment.verify.proof"]
        EX_DLX["DLX: oms.dlx -> Queue: oms.dlq"]
    end

    subgraph Domain["Service Layer & Worker Pool (internal/service)"]
        S_AI["AIOrderParserService (Gemini JSON + NLP Fallback)"]
        S_OMS["OMSService (Stock Lock, Webhook Dedup, Queue Worker)"]
        S_TX["TransferService & ReconciliationService"]
        S_VISION["AIVisionService (Gemini Multimodal Vision + Tamper Check)"]
        S_BI["ReportAIService (Gemini Text-to-SQL + AST Sanitizer)"]
    end

    subgraph Infra["Persistence Layer (PostgreSQL 16)"]
        PG[("PostgreSQL 16 (pgxpool)")]
        RD[("Redis 7 (Distributed Lock & Retry)")]
    end

    UI_OMS --> MW
    UI_LEDGER --> MW
    UI_SLIP --> MW
    UI_BI --> MW
    MW --> H_OMS
    MW --> H_PAY
    MW --> H_PROOF
    MW --> H_REPORT

    H_PROOF -->|Publish payment.verify| EX_OMS
    H_OMS -->|Publish order.parse| EX_OMS
    EX_OMS --> Q_PARSE
    EX_OMS --> Q_VERIFY
    Q_PARSE --> S_AI
    Q_VERIFY --> S_VISION
    S_VISION -->|Failure > 3x| EX_DLX

    H_REPORT --> S_BI
    S_BI -->|Read-Only Tx 2500ms| PG
    S_VISION -->|ACID SELECT FOR UPDATE| PG
    H_OMS --> S_OMS
    H_PAY --> S_TX
    S_OMS --> PG
    S_OMS --> RD
    S_TX --> PG
    S_TX --> RD
```

---

## 2. Peta Direktori & Kepemilikan Modul (Codebase Map)

| Path | Tanggung Jawab Utama | Catatan untuk Maintainer |
| :--- | :--- | :--- |
| `cmd/api/main.go` | Entrypoint HTTP server, bootstrap skema SQL idempotent, static file serving `web/dist`, RabbitMQ setup, dan *embedded queue worker*. | Mendukung *graceful shutdown* (`SIGTERM`/`SIGINT`) untuk menghentikan HTTP listener, RabbitMQ channel, dan worker goroutine secara bersih. |
| `cmd/worker/main.go` | Standalone daemon worker untuk *RabbitMQ AI Workers*, *Transactional Outbox*, dan *OMS Redis Queue*. | Gunakan di produksi ketika ingin memisahkan scaling proses API dari proses *background worker* (`ENABLE_EMBEDDED_WORKER=false` pada API). |
| `internal/config/config.go` | Pembacaan konfigurasi environment (`.env`) untuk PostgreSQL, Redis, RabbitMQ, dan validasi panjang kunci kriptografi. | `AES_SECRET_KEY` **wajib tepat 32 byte**. Server akan menolak start jika panjangnya tidak valid. |
| `internal/pkg/logger/` | Structured logger enterprise berbasis `github.com/rs/zerolog`. | Mendukung format JSON di production dan Pretty Console di local dev, dengan injeksi RequestID/TraceID ke context. |
| `internal/domain/` | Definisi entitas bisnis murni (`oms.go`, `ai_event.go`, `account.go`), konstanta *state machine*, dan DTO JSON. | Bebas dari ketergantungan driver database maupun framework HTTP. |
| `internal/repository/rabbitmq/` | Client AMQP 0-9-1, deklarasi exchange/queue/DLX, Producer (`PublishOrderParse`, `PublishPaymentProofVerify`), dan Consumer Pool ber-QoS prefetch (10) & Exponential Backoff. | Dilengkapi *in-memory channel fallback* untuk memastikan tes integrasi tetap berjalan mulus meskipun broker offline. |
| `internal/repository/postgres/` | Seluruh query SQL menggunakan `pgx/v5`, transaksi ACID, dan *row-level locking* (`payment_proof_repo.go`, `reporting_repo.go`, `oms_repo.go`). | Semua operasi tulis multi-tabel **wajib** dibungkus `pool.BeginTx(ctx, pgx.TxOptions{})` dengan `defer tx.Rollback(ctx)`. |
| `internal/service/` | Orkestrasi aturan bisnis, verifikasi slip Multimodal AI (`ai_vision_service.go`), Text-to-SQL Reporting (`report_ai_service.go`), simulator konkurensi, dan *integration tests*. | Lihat `ai_enterprise_workflows_test.go` dan `critical_workflows_test.go` sebagai spesifikasi hidup (*living specification*). |
| `internal/handler/http/` | HTTP handlers Chi: `payment_proof_handler.go` (Fast ACK `202 Accepted`), `report_handler.go` (Text-to-SQL BI), `oms_handler.go`. | Validasi payload, enkapsulasi status code, fast ACK, dan pendelegasian ke service layer. |
| `pkg/crypto/` | Primitif keamanan: `AES-256-GCM`, `HMAC-SHA256` (*constant-time*), `SHA-256` *hash chain*, dan `phash.go` (64-bit Difference Hash `dHash` untuk deteksi duplikasi struk bank transfer berjarak Hamming $\le 5$). | Jangan pernah mengganti `hmac.Equal` dengan operator `==` biasa untuk menghindari *timing side-channel attack*. |
| `migrations/` | Skrip DDL PostgreSQL berurutan (`000001` s/d `000005`) beserta skrip *rollback* (`.down.sql`). | Migrasi `000004` menambahkan `payment_proofs` & audit logs; migrasi `000005` menambahkan `outbox_events`, `ai_telemetry_events`, serta kolom `image_phash` & `is_duplicate_image`. |
| `frontend/` | Source code aplikasi SPA **Vue 3 + TypeScript + Pinia + Tailwind CSS v4 + `shadcn-vue` (`Reka UI` & `Vaul` Mobile Drawer)**. | Dilengkapi `PaymentSlipModal.vue` dan `AiReportingPanel.vue`. Hasil build (`npm run build`) dikompilasi ke `web/dist`. |

---

## 3. Invariant Kritis ("Do Not Break" Rules)

Bagian ini adalah aturan arsitektur paling penting. **Jangan melanggar kelima aturan ini** saat memodifikasi kode di masa depan:

### Rule 1: Deterministic Lock Ordering (Pencegahan Deadlock)
Saat sebuah transaksi mengunci lebih dari satu baris pada tabel yang sama (`products` pada multi-item order atau `accounts` pada transfer dana), **selalu urutkan UUID secara leksikografis** sebelum mengeksekusi `SELECT ... FOR UPDATE`:
* Pada `OMSRepository.CreateOrderAtomic` (`internal/repository/postgres/oms_repo.go`): `sortedIDs` diurutkan ascending sebelum loop `FOR UPDATE`.
* Pada `TransferRepository.ExecuteTransfer` (`internal/repository/postgres/transfer_repo.go`): `firstLockID` dan `secondLockID` dibandingkan dengan `strings.Compare` sebelum dikunci.
* **Alasan**: Jika Order A mengunci `[SKU-1, SKU-2]` sementara Order B mengunci `[SKU-2, SKU-1]` pada milidetik yang sama, PostgreSQL akan mengalami *circular wait deadlock*.

### Rule 2: Hukum Kekekalan Stok 3-Status & One-Way Order State Machine
Setiap produk memiliki tiga ember stok (`available_stock`, `reserved_stock`, `sold_stock`) dengan constraint `CHECK (>= 0)` di level database:
1. `CreateOrderAtomic` (`PENDING_PAYMENT`): `available_stock -= qty`, `reserved_stock += qty` (hanya jika `available_stock >= qty`).
2. `MarkOrderPaidAtomic` (`PENDING_PAYMENT` &rarr; `PAID`): `reserved_stock -= qty`, `sold_stock += qty`.
3. `CancelOrderAtomic` (`PENDING_PAYMENT` &rarr; `CANCELLED`): `reserved_stock -= qty`, `available_stock += qty`.

```mermaid
stateDiagram-v2
    [*] --> PENDING_PAYMENT : CreateOrderAtomic (avail -> reserved)
    PENDING_PAYMENT --> PAID : MarkOrderPaidAtomic (reserved -> sold)
    PENDING_PAYMENT --> CANCELLED : CancelOrderAtomic (reserved -> avail)
    PAID --> READY_TO_SHIP : AttachWaybillAndReadyToShip (AWB Generated)
    READY_TO_SHIP --> SHIPPED : Courier Handover
    SHIPPED --> COMPLETED : Buyer Confirmed
```

> **Penting**: Status `PAID`, `READY_TO_SHIP`, `SHIPPED`, dan `COMPLETED` **tidak boleh** kembali ke `PENDING_PAYMENT` atau `CANCELLED`. `MarkOrderPaidAtomic` mengunci baris `orders FOR UPDATE` dan langsung keluar dengan aman (`alreadyPaid = true`) apabila status sudah melewati `PENDING_PAYMENT`.

### Rule 3: Dual-Layer Idempotency
* **Webhook Pembayaran (`POST /api/v1/webhooks/payment`)**: Menggunakan constraint unik PostgreSQL `UNIQUE(provider, external_event_id)` pada tabel `webhook_events` dengan klausa `ON CONFLICT DO NOTHING RETURNING id`. Jika `pgx.ErrNoRows` kembali, itu berarti event duplikat—kembalikan `200 OK` (`{"status": "duplicate_ignored"}`) tanpa mendorong job baru ke antrean.
* **Transfer Dana (`POST /api/v1/transfers`)**: Menggunakan header `X-Idempotency-Key` yang dikunci secara atomik di `IdempotencyRepository` (`ACQUIRED` &rarr; `COMPLETED`).

### Rule 4: Fast Webhook ACK (< 50ms) & Asynchronous Retries
Endpoint penerima webhook **dilarang** memanggil API pihak ketiga secara sinkron (seperti ekspedisi kurir atau marketplace). Webhook hanya memvalidasi HMAC, mencatat `webhook_events`, mendorong job ke `QueueRepository`, dan langsung membalas `200 OK`. Pekerjaan lambat/rawan gagal dijalankan oleh worker dengan *Exponential Backoff* (`baseDelay * 2^(attempt-1)` = `2s, 4s, 8s, 16s`) hingga maksimum 4 percobaan sebelum dipindahkan ke *Dead-Letter Queue* (`system_error_logs`).

### Rule 5: Graceful Fallback untuk Ketergantungan Eksternal
* Jika `GEMINI_API_KEY` kosong atau API LLM mengalami *timeout* (`> 2.5s`), `AIOrderParserService` wajib jatuh otomatis ke parser deterministik (`parseDeterministicIndonesianChat`) tanpa menggagalkan alur kerja kasir/merchant.
* Jika `redisClient == nil`, `QueueRepository` dan `IdempotencyRepository` wajib tetap beroperasi menggunakan penyimpanan *in-memory* berpelindung `sync.Mutex`.
* Jika RabbitMQ broker offline, `RabbitMQClient` wajib jatuh otomatis ke saluran buffer *in-memory* Go channel agar testing dan fallback lokal tidak terblokir.

### Rule 6: AST Token Sanitization & Read-Only Tx untuk AI BI Reporting
Query SQL yang digenerasi oleh model Text-to-SQL wajib melalui dua lapis pertahanan keamanan sebelum dieksekusi:
1. **Application-Level AST Sanitizer**: Validasi token regex/tokenizer deterministik yang menolak query multi-pernyataan (karakter `;`) dan menolak seluruh kata kunci modifikasi/destruktif (`DROP`, `DELETE`, `UPDATE`, `INSERT`, `ALTER`, `TRUNCATE`, `GRANT`, `EXEC`, `CREATE`).
2. **Database-Level Read-Only Transaction**: Dijalankan dalam transaksi isolasi dengan `SET TRANSACTION READ ONLY` serta batas timeout eksekusi ketat `SET statement_timeout = '2500ms'` untuk mencegah resource exhaustion query agregasi berat.

### Rule 7: Zero-Trust Slip Validation & Pessimistic Order Lock
Ketika bukti transfer diverifikasi oleh worker via antrean RabbitMQ `payment.verify.proof`:
1. Jika struk terbukti asli (`tamper_detected == false`) dan nominal transfer `>= order.total_amount`, worker mengeksekusi transaksi ACID dengan `SELECT ... FOR UPDATE` pada tabel `orders` dan `order_items`, memindahkan `reserved_stock` &rarr; `sold_stock`, dan mengubah status order menjadi `PAID`.
2. Jika terdeteksi editan visual (*photoshop/font anomaly*) atau nominal kurang, sistem menandai bukti transfer sebagai `FRAUD_DETECTED / SUSPICIOUS`, mengalihkan status order menjadi `SUSPICIOUS_HOLD` untuk review manual operasional, dan **tidak** melepaskan stok reserved ke pasar bebas demi mencegah *double spending*.

### Rule 8: Transactional Outbox Pattern & pHash Duplicate Receipt Pre-Filter
1. **Zero Dual-Write Integrity**: Handler pembayaran dilarang mempublikasikan event secara langsung ke broker sebelum commit database. Transaksi PostgreSQL atomik menulis record bisnis dan antrean `outbox_events` secara bersamaan. Outbox Worker membaca baris `PENDING` dengan `SELECT ... FOR UPDATE SKIP LOCKED` lalu mendispatch ke RabbitMQ `oms.events`.
2. **Perceptual Hashing (pHash dHash 64-bit) Anti-Duplicate**: Setiap struk dihitung nilai 64-bit dHash sebelum pemanggilan AI. Gambar dengan jarak Hamming $\le 5$ dibandingkan riwayat struk sebelumnya langsung ditandai `is_duplicate_image = true` dan diisolasi ke `SUSPICIOUS_HOLD` dalam waktu $< 10$ms, mencegah pembakaran token LLM Vision yang sia-sia dan meniadakan penipuan upload struk berulang.

---

## 4. Relasi Skema Database & Strategi Indexing

```mermaid
erDiagram
    MERCHANTS ||--o{ PRODUCTS : owns
    MERCHANTS ||--o{ ORDERS : receives
    ORDERS ||--|{ ORDER_ITEMS : contains
    PRODUCTS ||--o{ ORDER_ITEMS : referenced_by
    ORDERS ||--o| SHIPMENTS : generates
    ACCOUNTS ||--o{ TRANSACTIONS : initiates
    ACCOUNTS ||--o{ LEDGER_ENTRIES : records
    TRANSACTIONS ||--|{ LEDGER_ENTRIES : balances
    RECONCILIATION_BATCHES ||--|{ RECONCILIATION_ITEMS : details

    PRODUCTS {
        uuid id PK
        uuid merchant_id FK
        varchar sku
        numeric price
        int available_stock
        int reserved_stock
        int sold_stock
        int version
    }
    ORDERS {
        uuid id PK
        uuid merchant_id FK
        varchar order_number UK
        varchar channel
        varchar status
        varchar waybill_number
        timestamptz created_at
    }
    WEBHOOK_EVENTS {
        uuid id PK
        varchar provider
        varchar external_event_id
        varchar status
        int processing_ms
    }
    SYSTEM_ERROR_LOGS {
        uuid id PK
        varchar queue_name
        varchar job_id
        varchar error_code
        boolean resolved
    }
```

### Strategi Optimasi Query yang Diterapkan:
1. **Multi-Tenant Composite Indexing**: Query inbox pesanan memfilter `merchant_id`, `status`/`channel`, dan mengurutkan berdasarkan `created_at DESC`. Indeks `idx_orders_merchant_status_created` dan `idx_orders_merchant_channel_created` menghindari *Sequential Scan* maupun *external sort* saat tabel `orders` tumbuh besar.
2. **Eliminasi N+1 Query**: `OMSRepository.ListOrders` mengambil daftar pesanan terlebih dahulu, lalu memuat seluruh item terkait dalam **satu batch query** `WHERE order_id = ANY($1)` (`getItemsForOrders`), bukan melakukan query di dalam *loop*.

---

## 5. Alur Sekuens End-to-End

### 5.1 Alur Webhook Pembayaran, Redis Queue, Exponential Backoff & DLQ

```mermaid
sequenceDiagram
    participant PGW as Payment Provider
    participant API as OMSHandler (Go)
    participant DB as PostgreSQL 16
    participant Q as QueueRepository (Redis)
    participant W as Background Worker
    participant EXP as 3rd-Party Courier API

    PGW->>API: POST /api/v1/webhooks/payment (HMAC Headers)
    API->>API: Verify Timestamp (<=300s) & Constant-Time HMAC
    API->>DB: INSERT webhook_events ON CONFLICT DO NOTHING
    alt Duplicate external_event_id
        DB-->>API: 0 rows (Conflict)
        API-->>PGW: 200 OK {"status": "duplicate_ignored"} (<5ms)
    else First-time Event
        DB-->>API: webhook_db_id
        API->>Q: Enqueue(payment-webhook-queue)
        API-->>PGW: 200 OK {"status": "accepted"} (<15ms)
        W->>Q: PopNextDueJob()
        W->>DB: MarkOrderPaidAtomic (reserved -> sold, status=PAID)
        W->>Q: Fan-out (logistics-waybill, stock-sync, notification)
        W->>EXP: Request Waybill AWB
        alt HTTP 429 / Timeout (Attempt 1..3)
            EXP-->>W: 429 Too Many Requests
            W->>Q: Schedule ZSET Delayed Retry (2s, 4s, 8s, 16s)
        else Attempt 4 Still Fails
            W->>Q: Move to FAILED (Dead-Letter Queue)
            W->>DB: INSERT system_error_logs
        else Courier 200 OK
            EXP-->>W: AWB Number (JNE-AWB-XXXX)
            W->>DB: AttachWaybillAndReadyToShip (status=READY_TO_SHIP)
        end
    end
```

---

## 6. Panduan Pengembangan Lanjutan (How-To Playbooks)

### Playbook A: Menambahkan Kanal Penjualan Baru (Misal: `LAZADA`)
1. Tambahkan konstanta `ChannelLazada OrderChannel = "LAZADA"` di `internal/domain/oms.go` dan tipe `OrderChannel` di `frontend/src/types/oms.ts`.
2. Buat file migrasi baru `migrations/000004_add_lazada_channel.up.sql` untuk memperbarui constraint `CHECK (channel IN (...))` pada tabel `orders`, lalu daftarkan file migrasi tersebut di `cmd/api/main.go` dan `.github/workflows/ci.yml`.
3. Tambahkan badge warna dan filter pill di `frontend/src/components/OrderInboxPanel.vue`, lalu jalankan `cd frontend && npm run build`.

### Playbook B: Mengganti Simulator Kurir dengan API Logistik Produksi (Biteship / RajaOngkir)
1. Buka `OMSService.executeQueueJob` di `internal/service/oms_service.go` pada blok `case domain.QueueLogisticsWaybill:`.
2. Gantikan pembangkitan string `awb` lokal dengan pemanggilan HTTP client ber-timeout eksplisit (misal `5 * time.Second`) ke endpoint pembuatan order kurir.
3. Jika respons HTTP berstatus `429` atau `5xx`, kembalikan `error` biasa—`ProcessNextQueueJob` secara otomatis akan menjadwalkan *Exponential Backoff* hingga 4 kali dan mencatat ke `system_error_logs` bila gagal total.

### Playbook C: Memodifikasi UI Frontend Vue 3
1. Jalankan backend Go di terminal pertama: `go run ./cmd/api` (port `8080`).
2. Jalankan Vite dev server dengan *hot-reload* di terminal kedua: `cd frontend && npm run dev` (port `5173`, otomatis mem-proxy `/api` ke `:8080`).
3. Setelah selesai, jalankan `cd frontend && npm run build` untuk memperbarui bundle produksi di `web/dist/` dan pastikan `vue-tsc` lulus tanpa error tipe data.

### Playbook D: Menambahkan Event Asinkron Baru ke RabbitMQ
1. Deklarasikan nama antrean dan routing key baru pada `internal/domain/ai_event.go` dan `internal/repository/rabbitmq/client.go`.
2. Daftarkan binding antrean ke Exchange `oms.events` di fungsi `DeclareTopology` (`internal/repository/rabbitmq/client.go`).
3. Tambahkan fungsi *publisher* di `internal/repository/rabbitmq/producer.go` dan fungsi *consumer* di `internal/repository/rabbitmq/consumer.go`.
4. Selalu sertakan `context.WithTimeout` (misal 3.5s - 5s) dan jadwalkan retry dengan exponential backoff (`requeueWithBackoff`) serta pengalihan ke Dead-Letter Exchange (`oms.dlx`) bila melebihi batas percobaan.

---

## 7. Runbook Operasional & Troubleshooting

* **Webhook Ditolak `401 Unauthorized`**:
  * Periksa selisih jam server pengirim vs jam UTC server penerima (`X-Webhook-Timestamp` maksimal berbeda `300` detik).
  * Pastikan string yang di-hash adalah `"{timestamp}.{raw_json_body}"` menggunakan secret `WEBHOOK_SECRET` tanpa mengubah urutan spasi JSON.
* **Job Berhenti di Status `FAILED / DLQ`**:
  * Periksa pesan error spesifik di tab **System Error Logs / DLQ** pada dashboard atau query `SELECT * FROM system_error_logs WHERE resolved = FALSE`.
  * Setelah gangguan layanan eksternal pulih (atau *Chaos Simulator* dikembalikan ke `Normal`), klik tombol **Manual Retry** pada dashboard atau panggil `POST /api/v1/queue/jobs/{id}/retry`.
* **Menjalankan `go test -race` di Windows**:
  * Flag `-race` di Go membutuhkan kompiler C (`CGO_ENABLED=1` + MinGW-w64 `gcc`). Jika Windows lokal tidak memiliki `gcc`, jalankan `go test -v ./...` secara lokal dan biarkan pipeline **GitHub Actions Ubuntu Runner** memvalidasi `go test -v -race ./...` secara penuh pada setiap push.
