# AI-Native Software Engineering Workflow (`AI_WORKFLOW.md`)

> **Engineering Standard — Omni-Channel E-Commerce OMS & Secure Payment Gateway**  
> Dokumen ini mendefinisikan metodologi kerja **PRD-to-Production** menggunakan AI Coding Agents secara disiplin, terukur, dan aman untuk sistem berisiko tinggi (*inventory locking, payment webhooks, distributed queues, dan double-entry ledger*).

---

## 1. Prinsip Utama (AI-Assisted, Engineer-Owned)

1. **Zero Blind Copy-Paste untuk Jalur Kritis**: Seluruh transaksi database (`BEGIN ... FOR UPDATE ... COMMIT`), verifikasi kriptografi (`HMAC-SHA256`, `AES-256-GCM`), dan *state machine* pembayaran wajib melalui spesifikasi eksplisit sebelum kode ditulis.
2. **Spec-First Decomposition**: Setiap fitur dari *Product Requirement Document (PRD)* dipecah menjadi task kecil yang memiliki *Input/Output Contract*, *Edge Cases*, dan *Automated Verification*.
3. **Deterministic Guardrails**: Fitur berbasis LLM (seperti *AI Social Commerce WhatsApp Order Parser*) wajib memiliki batas waktu ketat (`< 3s` timeout), skema JSON tervalidasi, validasi stok real-time ke PostgreSQL, serta *deterministic fallback* agar operasional merchant tidak pernah berhenti saat API pihak ketiga mengalami gangguan.

---

## 2. PRD-to-Engineering Task Decomposition

| ID Modul | Fitur Bisnis (PRD) | Dekomposisi Task Engineering | Risiko Utama yang Dimitigasi |
| :--- | :--- | :--- | :--- |
| **MOD-01** | AI Social Commerce Order Capture (WA/Live) | `POST /api/v1/orders/ai-parse` dengan Gemini JSON Schema + Hybrid Indonesian NLP Fallback + Live SKU Stock Check | Halusinasi SKU harga, API LLM lambat/timeout (`> 3s`), order produk stok habis |
| **MOD-02** | Multi-Channel Anti-Overselling Engine | 3-State Stock (`available`, `reserved`, `sold`), pengurutan deterministik UUID SKU (anti-deadlock), `SELECT ... FOR UPDATE` + `WHERE available_stock >= $1` | *Race condition* saat Flash Sale / Live Selling yang menyebabkan stok minus (*overselling*) |
| **MOD-03** | Secure Payment & Idempotent Webhook | Verifikasi `HMAC-SHA256` constant-time, anti-replay window `<= 300s`, `UNIQUE(provider, external_event_id)` di `webhook_events`, fast ACK `< 50ms` | *Double stock deduction*, *replay attack*, webhook badai retry dari payment provider |
| **MOD-04** | Async Logistics Waybill & Stock Sync Queue | Redis Queue (`logistics-waybill-queue`, `channel-stock-sync-queue`, `notification-queue`) dengan *Exponential Backoff* (`2s, 4s, 8s, 16s`) + *Dead-Letter Queue* (`system_error_logs`) | API ekspedisi (JNE/J&T/SiCepat) lambat atau terkena *Rate Limit (HTTP 429)* |
| **MOD-05** | Vue 3 + TypeScript + Pinia Operational UI | 4-Panel B2B Merchant Dashboard, Stress-Test Simulator (10 Concurrent Buyers & 5x Duplicate Webhook), Thermal A6 Waybill Print, Core Ledger View | Kurangnya visibilitas operasional atas stok tertahan, job gagal (DLQ), dan audit pembayaran |

---

## 3. Prompt Task Template Standar

Setiap instruksi implementasi ke AI Agent menggunakan struktur kontrak berikut:

```markdown
### [TASK-ID]: <Judul Komponen Spesifik>

**1. Business Context & Goal:**
<Jelaskan masalah bisnis yang diselesaikan dan SLA performa yang wajib dipenuhi, misal ACK < 50ms>

**2. Architectural & Database Constraints:**
- Tabel terkait: <nama_tabel & index>
- Isolation & Locking: <misal: PostgreSQL Transaction + SELECT FOR UPDATE sorted by product_id>
- State Machine Rules: <misal: PENDING_PAYMENT -> PAID -> READY_TO_SHIP satu arah>

**3. API / Interface Contract:**
- Endpoint / Signature: <Method + Path + Headers>
- Success Response: <HTTP code + JSON structure>
- Failure / Edge Cases:
  - Insufficient stock -> 409 Conflict
  - Invalid HMAC / Replay > 5m -> 401 Unauthorized
  - Duplicate external_event_id -> 200 OK {"status": "duplicate_ignored"}

**4. Required Verification Tests:**
- Unit / Integration test menggunakan `go test -v -race ./...`
- Skenario konkurensi paralel (`sync.WaitGroup` + barrier channel)
```

---

## 4. Pre-Merge Engineering Review Checklist

Sebelum kode digabungkan ke branch `main`, setiap perubahan **wajib** lulus daftar periksa berikut:

### A. Concurrency & Data Integrity
- [x] Apakah mutasi stok (`available_stock`, `reserved_stock`, `sold_stock`) berjalan di dalam satu transaksi ACID PostgreSQL (`BEGIN ... COMMIT`)?
- [x] Apakah multi-item order mengurutkan `product_id` terlebih dahulu sebelum `SELECT ... FOR UPDATE` untuk mencegah *deadlock*?
- [x] Apakah terdapat constraint `CHECK (available_stock >= 0)` di level skema PostgreSQL sebagai pertahanan lapis terakhir?
- [x] Apakah `go test -v -race ./...` lulus 100% tanpa peringatan *data race*?

### B. Security & Webhook Idempotency
- [x] Apakah perbandingan signature `HMAC-SHA256` menggunakan `hmac.Equal` (*constant-time comparison*) untuk mencegah *timing attack*?
- [x] Apakah timestamp header diverifikasi dalam jendela toleransi `<= 5 menit` untuk mencegah *replay attack*?
- [x] Apakah idempotency dijamin secara atomik di database menggunakan `INSERT INTO webhook_events ... ON CONFLICT (provider, external_event_id) DO NOTHING`?
- [x] Apakah order yang sudah `PAID` terlindungi dari pemotongan stok ganda (*one-way state machine*)?

### C. Resilience, Retries, & Observability
- [x] Apakah pekerjaan pihak ketiga (cetak resi logistik, sinkronisasi stok marketplace, notifikasi WA) diproses secara asinkron di belakang layar sehingga webhook merespons `< 50ms`?
- [x] Apakah kegagalan sementara (*HTTP 429 / 503*) memiliki mekanisme *Exponential Backoff* (`attempts: 4`, delay `2s, 4s, 8s, 16s`)?
- [x] Apakah job yang tetap gagal setelah batas percobaan masuk ke *Dead-Letter Queue (DLQ)*, tercatat di `system_error_logs`, dan dapat di-*retry* secara manual dari dashboard?

### D. Frontend UX & Operational Readiness
- [x] Apakah Vue 3 + TypeScript (`vue-tsc`) terkompilasi bersih tanpa *type error*?
- [x] Apakah peringatan stok kurang muncul secara visual (merah) pada modal AI Order Parser sebelum order disubmit?
- [x] Apakah fitur cetak label pengiriman (*Thermal Waybill A6*) rapi saat dicetak?

