# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.3.0] - 2026-11-14 (JMC IT Consultant Assessment & Enterprise Compliance Release)

### Added
- **Demo Account Gate & Role-Based Switcher (`frontend/src/components/DemoLoginModal.vue`, `authStore.ts`)**:
  - Modal dan dialog login kredensial pengujian resmi PT. Jiwa Muda Cipta Indonesia (JMC).
  - Dua kartu preset akun 1-klik:
    - `demo@merchant.id` (Pass: `DemoJMC2026!`) — Role: *Merchant Admin & Gudang (Operasional)*.
    - `finance@merchant.id` (Pass: `DemoJMC2026!`) — Role: *Finance, Bendahara & Auditor (ERP Accounting)*.
  - Form validasi input manual kredensial pengujian dengan persistensi sesi di `localStorage`.
- **Ekspor Laporan Finansial & Fulfillment ke CSV & PDF (`OrderInboxPanel.vue`, `AiReportingPanel.vue`)**:
  - Tombol **"📥 Export CSV"** pada tabel Order Inbox: mengunduh seluruh pesanan terfilter dalam format CSV UTF-8 BOM.
  - Tombol **"🖨️ Cetak PDF"** pada Order Inbox: membuka jendela pratinjau cetak resmi ber-kop *PT Toko Maju Omni-Retail* lengkap dengan rekapitulasi total nominal, ringkasan filter, dan tanda tangan digital.
  - Tombol **"📥 Export CSV Hasil BI"** pada konsol Text-to-SQL: memungkinkan ekspor instan matriks data analitik dinamis hasil inferensi AI ke file spreadsheet.
- **1-Click State Recovery & Demo Reset (`POST /api/v1/simulator/reset-demo`, `OMSRepository.ResetAndSeedDemoState`)**:
  - Mengembalikan stok produk (`SKU-1` kembali ke 1 pcs untuk pengujian Flash Sale), pesanan sampel, dan membersihkan log sistem dengan satu klik melalui header dashboard atau Command Palette (`Ctrl+K`).
- **Skrip Backup & Prosedur Disaster Recovery (`scripts/backup-db.sh`, `docs/DEPLOYMENT_PRODUCTION.md`)**:
  - Skrip shell otomatis pencadangan database PostgreSQL (`pg_dump -F c`) berpelindung hak akses `chmod 600`.
  - Kebijakan retensi otomatis pembersihan cadangan lama (7 hari) dan integrasi Linux `crontab`.
  - Panduan pemulihan bencana (*Disaster Recovery*) dan konsep *Point-in-Time Recovery (PITR)* berbasis *WAL Archiving*.

### Changed
- Header antarmuka web diperbarui dengan chip pengguna aktif (`Logged in as: demo@merchant.id`), tombol cepat *Reset Data*, dan penunjuk peran aktif.
- Banner panduan penguji diselaraskan dengan spesifikasi tes teknis JMC (Kode Lowongan `R260032`).

---

## [v1.2.0] - 2026-11-04 (Transactional Outbox, Vision AI Slip & FinOps Telemetry)

### Added
- **Transactional Outbox Pattern**:
  - Tabel `outbox_events` pada PostgreSQL dengan mekanisme pembacaan *high-concurrency* `SELECT ... FOR UPDATE SKIP LOCKED` untuk mengeliminasi *Dual-Write Problem* antara PostgreSQL dan RabbitMQ broker.
- **Perceptual Hash (64-bit dHash) Anti-Fraud Pre-Filter (`pkg/crypto/phash.go`)**:
  - Kalkulasi hash selisih gradien visual piksel $9 \times 8$ dan perbandingan jarak *Hamming Distance* $\le 5$ dalam waktu $< 10$ ms.
  - Mengisolasi struk duplikat langsung ke `SUSPICIOUS_HOLD` tanpa memanggil LLM Vision (menghemat 100% biaya token).
- **AI FinOps & Observability Cockpit (`GET /api/v1/ai/telemetry`, `AiTelemetryPanel.vue`)**:
  - Panel telemetri real-time melacak konsumsi token (Vision & SQL), estimasi biaya model dalam USD ($), rata-rata latensi inferensi vs SLA Fast-ACK (< 20ms), dan rasio karantina penipuan.
- **Enterprise Semantic Metrics Layer Text-to-SQL (`report_ai_service.go`)**:
  - Grounding formula bisnis baku (GMV, Net Revenue, Receivables, Quarantine Volume) ke dalam prompt sistem Gemini.

---

## [v1.1.0] - 2026-10-31 (Modern Mobile-First SaaS UI & Thermal Printing)

### Added
- **Modern `shadcn-vue` + `Vaul` Dynamic App Shell**:
  - Transformasi responsif: Desktop Sheet/Modal berubah otomatis menjadi *Swipeable Bottom Sheet* pada perangkat mobile (< 768px).
- **Thermal Waybill Printing A6 (`100x150mm`)**:
  - Dialog cetak label resi ekspedisi (JNE, SiCepat, J&T) berstandar barcode Code128 dan format CSS `@page { size: 100mm 150mm }`.
- **Command Palette (`Ctrl+K`)**:
  - Antarmuka pencarian cepat dan peluncur simulasi *race condition*, webhook deduplikasi, dan pergantian tema *light/dark*.

---

## [v1.0.0] - 2026-10-25 (Initial Core Release)

### Added
- **Omni-Channel OMS Engine**:
  - Integrasi multi-kanal: Web Checkout, Shopee, TikTok Shop, dan WhatsApp Live Chat parser.
- **3-State Inventory & Anti-Overselling**:
  - Penguncian baris PostgreSQL deterministik (`SELECT ... FOR UPDATE` + `WHERE available_stock >= $1`) dengan 3 status: `available_stock`, `reserved_stock`, dan `sold_stock`.
- **Secure Payment Switch & Double-Entry Ledger**:
  - Verifikasi tanda tangan *constant-time* `HMAC-SHA256` anti-*timing attack*.
  - Deduplikasi webhook bergaransi constraint unik PostgreSQL `UNIQUE(provider, external_event_id)` (< 50ms fast ACK).
  - Pembukuan akuntansi berpasangan (*Double-Entry Bookkeeping*) berstatus *append-only*.
- **3-Way Bank Settlement Reconciliation**:
  - Parser streaming file CSV mutasi rekening bank dan pencocokan 3-arah (*3-Way Matching*).
