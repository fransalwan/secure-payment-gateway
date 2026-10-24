# Secure Payment Gateway (`secure-payment-gateway`)

> **Production-Grade, High-Performance Payment Processing & Double-Entry Ledger Engine**  
> Built with Go, PostgreSQL, and Redis. Engineered with a zero-trust cybersecurity posture, high-throughput concurrency control, and an interactive web dashboard.

[![Go Version](https://img.shields.io/badge/Go-1.27%2B-blue.svg)](https://golang.org)
[![Database](https://img.shields.io/badge/Database-PostgreSQL%2016-blue.svg)](https://www.postgresql.org/)
[![Cache & Lock](https://img.shields.io/badge/Cache%20%26%20Lock-Redis%207-red.svg)](https://redis.io/)
[![Security](https://img.shields.io/badge/Security-AES--256--GCM%20%7C%20HMAC--SHA256-green.svg)](#cybersecurity-architecture)
[![Dashboard UI](https://img.shields.io/badge/UI-Interactive%20Fintech%20Dashboard-emerald.svg)](#-interactive-fintech-dashboard--security-playground)

---

## 📌 Executive Summary

Dalam industri perbankan dan fintech, integritas finansial dan kecepatan transaksi adalah dua hal yang tidak bisa ditawar. Kesalahan sekecil apa pun dapat menyebabkan kerugian finansial (*double-deduction*), kebocoran data sensitif (*identity leakage*), atau kebuntuan sistem (*database deadlock*).

`secure-payment-gateway` adalah backend engine yang dirancang untuk memecahkan tantangan dunia nyata tersebut:
1. **Zero-Trust Cybersecurity**: Enkripsi data sensitif di level aplikasi (AES-256-GCM), autentikasi request berbasis tanda tangan kriptografis (HMAC-SHA256), mitigasi Replay Attack, serta rantai audit tamper-evident.
2. **Extreme Concurrency & Performance**: Arsitektur akuntansi berpasangan (*Double-Entry Bookkeeping*), algoritma pengurutan lock deterministik pencegah deadlock, dan *Distributed Idempotency Engine*.
3. **Resilient Architecture**: Pola *Transactional Outbox Pattern* dengan `SKIP LOCKED` untuk pengiriman event asinkron tanpa *dual-write anomaly*.
4. **Interactive Playground UI**: Web Dashboard siap demo untuk memvisualisasikan buku besar (*double-entry ledger*), audit chain, serta simulasi serangan siber secara interaktif langsung dari browser.

---

## 🖥️ Interactive Fintech Dashboard & Security Playground

Aplikasi dilengkapi dengan dashboard visual interaktif yang dapat diakses langsung di browser saat server berjalan:

📍 **URL Dashboard**: `http://localhost:8080/` atau `http://localhost:8080/dashboard`

### Fitur Unggulan Dashboard:
* **Live Payment & Transfer Simulator**: Melakukan transfer dana riil dengan kalkulasi otomatis `X-Signature` (HMAC-SHA256), `X-Timestamp`, dan `X-Idempotency-Key`.
* **Security & Attack Playground Controls**:
  * **Simulasi MITM (Tamper Signature)**: Memanipulasi tanda tangan payload untuk mengamati penolakan otomatis oleh middleware keamanan (`401 Unauthorized`).
  * **Simulasi Replay Attack**: Mengirimkan request dengan timestamp basi (> 5 menit) untuk menguji penolakan anti-replay.
  * **Uji Idempotensi**: Menembak request berulang dengan key yang sama untuk memverifikasi pencegahan *double-deduct*.
* **Field-Level AES-256-GCM Inspector**: Form pendaftaran rekening nasabah yang mengenkripsi data identitas (NIK / Card Number) ke bentuk ciphertext hex sebelum disimpan ke database.
* **Double-Entry Ledger Audit Trail**: Visualisasi tabel mutasi seimbang $\sum \text{Debit} = \sum \text{Credit}$ dengan badge warna realtime dan saldo pasca-transaksi (*balance_after*).
* **Tamper-Evident Audit Chain Explorer**: Visualisasi rantai blok SHA-256 (`Previous Hash` $\rightarrow$ `Current Hash`) yang membuktikan rekaman transaksi tidak dimanipulasi dari dalam database.

---

## 🏗️ Arsitektur Sistem

```
                      +-----------------------------+
                      |   Merchant / Client / Web UI|
                      +--------------+--------------+
                                     |  (HTTPS + HMAC-SHA256 Signature)
                                     v
               +--------------------------------------------+
               |        API Gateway / Chi HTTP Router       |
               |  - Anti-Replay Timestamp Validator         |
               |  - Constant-Time HMAC Signature Check      |
               |  - Panic Recovery & Structured Logging     |
               +---------------------+----------------------+
                                     |
                                     v
               +--------------------------------------------+
               |          Transfer & Ledger Service         |
               |  - Idempotency State Machine (Lock/Cache)  |
               |  - AES-256-GCM Field-Level Decryption      |
               +----------+-----------------------+---------+
                          |                       |
        (Distributed Lock)|                       | (Async Batch)
                          v                       v
               +--------------------+   +-------------------+
               |    Redis Store     |   |   Outbox Worker   |
               | - Idempotency Lock |   | - Reliable Queue  |
               | - Response Cache   |   | - Webhook Dispatch|
               +--------------------+   +---------+---------+
                          |                       |
                          +-----------+-----------+
                                      |
                                      v
               +--------------------------------------------+
               |             PostgreSQL Engine              |
               |  - Deterministic Row Locking (No Deadlock) |
               |  - Double-Entry Ledger (DEBIT & CREDIT)    |
               |  - Hash-Chained Tamper-Evident Audit Logs  |
               +--------------------------------------------+
```

---

## 🛡️ Cybersecurity Highlights

### 1. Constant-Time HMAC-SHA256 Request Verification
Setiap transaksi transfer wajib membawa header:
* `X-Signature`: Tanda tangan hex `HMAC-SHA256(timestamp + "." + body, secret_key)`
* `X-Timestamp`: Unix timestamp saat request dibuat

Sistem memverifikasi signature menggunakan algoritma **Constant-Time Comparison** (`hmac.Equal`) untuk menggagalkan **Timing Attacks** (peretas mengukur selisih waktu respons mikrodetik untuk menebak byte signature).

### 2. Anti-Replay Attack Protection
Request yang dicegat oleh pihak ketiga di jaringan publik tidak dapat dikirim ulang berulang kali. Header `X-Timestamp` dievaluasi terhadap clock server dengan batas toleransi maksimal $\pm 300$ detik (5 menit).

### 3. Field-Level Authenticated Encryption (AES-256-GCM)
Nomor identitas (NIK / Card Number) dienkripsi sebelum masuk ke database menggunakan **AES-256-GCM**. Nonce 12-byte digenerasi secara acak untuk setiap enkripsi, menghasilkan ciphertext unik bahkan untuk data identitas yang identik. Jika ada manipulasi langsung pada baris database, tag otentikasi GCM akan gagal mendekripsi.

### 4. Tamper-Evident Hash-Chained Audit Logs
Setiap mutasi saldo dicatat dalam tabel `audit_logs` yang bersifat append-only:
$$\text{CurrentHash} = \text{SHA256}(\text{PreviousHash} + ":" + \text{Payload})$$
Hal ini membentuk rantai kriptografis mirip struktur blockchain mini yang membuktikan catatan audit tidak pernah disisipkan atau diubah secara retrospektif.

---

## ⚡ Concurrency & Performance Engine

### 1. Deterministic Lock Ordering (Eliminasi Deadlock)
Dalam sistem transfer, skenario di mana Akun A mentransfer ke Akun B bersamaan dengan Akun B mentransfer ke Akun A dapat memicu **Database Deadlock** jika kedua transaksi mengunci baris dengan urutan terbalik:
* Transaksi 1: Lock Akun A $\rightarrow$ menunggu Lock Akun B
* Transaksi 2: Lock Akun B $\rightarrow$ menunggu Lock Akun A

**Solusi yang diimplementasikan:**
Sebelum menjalankan `SELECT FOR UPDATE`, ID kedua akun dibandingkan secara leksikografis:
```go
firstLockID := req.SourceAccountID
secondLockID := req.DestinationAccountID
if strings.Compare(firstLockID.String(), secondLockID.String()) > 0 {
    firstLockID, secondLockID = secondLockID, firstLockID
}
// Selalu lock firstLockID terlebih dahulu, baru kemudian secondLockID
```
Hal ini menjamin seluruh proses di seluruh thread selalu meminta lock dalam urutan yang identik, secara matematis **mengeliminasi circular wait deadlock**.

### 2. Distributed Idempotency State Machine
Mencegah terjadinya saldo terpotong dua kali (*Double Deduction*):
1. Request masuk dengan header `X-Idempotency-Key: <UUID>`.
2. Melakukan atomic `SETNX` ke Redis dengan status `PROCESSING`.
3. Jika request kedua datang saat status masih `PROCESSING`, tolak dengan `409 Conflict`.
4. Saat transaksi berhasil di-commit di PostgreSQL, cache response disimpan di Redis. Request ulangan berikutnya langsung mengembalikan response tanpa menyentuh saldo database.

### 3. Double-Entry Bookkeeping
Uang tidak pernah diubah secara sembarangan. Setiap transaksi selalu menghasilkan dua baris entri di `ledger_entries`:
* **DEBIT** pada rekening pengirim.
* **CREDIT** pada rekening penerima.
* Nilai mutasi selalu berimbang ($\sum \text{Debit} - \sum \text{Credit} = 0$).

---

## 📡 Daftar REST API Endpoints

| Method | Endpoint | Keterangan | Autentikasi / Header |
|---|---|---|---|
| `GET` | `/` atau `/dashboard` | Interactive Fintech Dashboard Web UI | Publik |
| `GET` | `/health` | Health Check & Latency Probe | Publik |
| `GET` | `/api/v1/accounts` | List semua rekening terdaftar | Publik |
| `POST` | `/api/v1/accounts` | Buat rekening baru (Enkripsi AES-256-GCM) | Publik |
| `GET` | `/api/v1/accounts/{id}` | Detail saldo & rekening | Publik |
| `POST` | `/api/v1/transfers` | Eksekusi transfer saldo atomik berpasangan | `X-Signature`, `X-Timestamp`, `X-Idempotency-Key` |
| `GET` | `/api/v1/ledger` | Riwayat mutasi Double-Entry Ledger | Publik |
| `GET` | `/api/v1/audit-logs` | Rantai Audit Tamper-Evident SHA-256 | Publik |
| `POST` | `/api/v1/security/sign` | Helper pembuat signature untuk testing UI | Publik |
| `POST` | `/api/v1/reconciliation/upload` | Upload & proses rekonsiliasi CSV settlement bank | Publik |
| `GET` | `/api/v1/reconciliation/batches` | Daftar batch riwayat rekonsiliasi | Publik |
| `GET` | `/api/v1/reconciliation/batches/{id}` | Laporan detail selisih & kecocokan batch | Publik |
| `POST` | `/api/v1/reconciliation/sample` | Simulasi 1-klik rekonsiliasi bank untuk demo UI | Publik |

---

## 🗄️ Database Schema

```sql
accounts (
    id UUID PRIMARY KEY,
    account_number VARCHAR(32) UNIQUE,
    holder_name VARCHAR(128),
    encrypted_identity TEXT, -- AES-256-GCM
    balance BIGINT NOT NULL CHECK (balance >= 0),
    currency VARCHAR(3),
    version INT
)

transactions (
    id UUID PRIMARY KEY,
    reference_id VARCHAR(64) UNIQUE,
    idempotency_key VARCHAR(128) UNIQUE,
    source_account_id UUID REFERENCES accounts(id),
    destination_account_id UUID REFERENCES accounts(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    status VARCHAR(20) -- PENDING, SETTLED, FAILED
)

ledger_entries (
    id UUID PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id),
    account_id UUID REFERENCES accounts(id),
    entry_type VARCHAR(6), -- DEBIT / CREDIT
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL
)

audit_logs (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50),
    action VARCHAR(50),
    payload JSONB,
    previous_hash VARCHAR(64),
    current_hash VARCHAR(64)
)

outbox_events (
    id UUID PRIMARY KEY,
    event_type VARCHAR(50),
    payload JSONB,
    status VARCHAR(20) -- PENDING, PROCESSED, FAILED
)

reconciliation_batches (
    id UUID PRIMARY KEY,
    batch_number VARCHAR(64) UNIQUE,
    bank_name VARCHAR(64),
    total_records INT,
    matched_count INT,
    discrepancy_count INT,
    missing_in_gateway_count INT,
    status VARCHAR(20) -- PROCESSING, COMPLETED, FAILED
)

reconciliation_items (
    id UUID PRIMARY KEY,
    batch_id UUID REFERENCES reconciliation_batches(id),
    reference_id VARCHAR(64),
    bank_amount BIGINT,
    gateway_amount BIGINT,
    difference_amount BIGINT,
    match_status VARCHAR(30) -- MATCHED, DISCREPANCY_AMOUNT, MISSING_IN_GATEWAY
)
```

---

## 🚀 Panduan Menjalankan & Pengujian

### 1. Prasyarat
* [Go 1.22+](https://golang.org)
* [Docker & Docker Compose](https://www.docker.com/) (atau PostgreSQL & Redis lokal)

### 2. Jalankan Database & Cache (Docker)
```bash
docker compose up -d
```

### 3. Jalankan Automated Test Suite
```bash
go test -v -race ./...
```

### 4. Jalankan API Server
```bash
go run ./cmd/api
```
Server akan aktif di `http://localhost:8080`. Buka browser untuk melihat dashboard interaktif.

### 5. Jalankan Background Outbox Worker (Terminal Terpisah)
```bash
go run ./cmd/worker
```

---

## 📊 Benchmark & Performance Targets (k6)

Jalankan script stress-testing:
```bash
k6 run scripts/k6_load_test.js
```

**Target SLO Performa:**
* **Throughput**: $> 1.500$ TPS
* **Latency p95**: $< 45$ ms
* **Latency p99**: $< 90$ ms
* **Zero Negative Balance Invariant**: Terbukti $0$ saldo negatif bahkan saat ribuan request konkuren bersaing pada satu akun.

---

## 💼 Poin Diskusi Saat Technical Interview

Ketika ditanya mengenai proyek ini saat wawancara teknis:
1. **Tentang Race Condition**: Jelaskan bagaimana pengurutan UUID deterministik memecahkan persoalan circular wait deadlock pada PostgreSQL `SELECT FOR UPDATE`.
2. **Tentang Data Loss**: Jelaskan peran Transactional Outbox Pattern yang mencatat event dalam satu transaksi ACID yang sama dengan mutasi saldo, mencegah masalah *Dual-Write*.
3. **Tentang Keamanan Finansial**: Jelaskan mengapa enkripsi AES-256-GCM dipilih dibanding CBC (karena integritas terotentikasi), dan pentingnya `hmac.Equal` untuk menolak side-channel timing attack.
