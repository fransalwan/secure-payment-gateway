# Secure Payment Gateway (`secure-payment-gateway`)

> **Production-Grade, High-Performance Payment Processing & Double-Entry Ledger Engine**  
> Built with Go, PostgreSQL, and Redis. Engineered with a zero-trust cybersecurity posture and high-throughput concurrency control.

[![Go Version](https://img.shields.io/badge/Go-1.27%2B-blue.svg)](https://golang.org)
[![Database](https://img.shields.io/badge/Database-PostgreSQL%2016-blue.svg)](https://www.postgresql.org/)
[![Cache & Lock](https://img.shields.io/badge/Cache%20%26%20Lock-Redis%207-red.svg)](https://redis.io/)
[![Security](https://img.shields.io/badge/Security-AES--256--GCM%20%7C%20HMAC--SHA256-green.svg)](#cybersecurity-architecture)

---

## 📌 Executive Summary

Dalam industri perbankan dan fintech, integritas finansial dan kecepatan transaksi adalah dua hal yang tidak bisa ditawar. Kesalahan sekecil apa pun dapat menyebabkan kerugian finansial (*double-deduction*), kebocoran data sensitif (*identity leakage*), atau kebuntuan sistem (*database deadlock*).

`secure-payment-gateway` adalah backend engine yang dirancang untuk memecahkan tantangan dunia nyata tersebut:
1. **Zero-Trust Cybersecurity**: Enkripsi data sensitif di level aplikasi (AES-256-GCM), autentikasi request berbasis tanda tangan kriptografis (HMAC-SHA256), mitigasi Replay Attack, serta rantai audit tamper-evident.
2. **Extreme Concurrency & Performance**: Arsitektur akuntansi berpasangan (*Double-Entry Bookkeeping*), algoritma pengurutan lock deterministik pencegah deadlock, dan *Distributed Idempotency Engine*.
3. **Resilient Architecture**: Pola *Transactional Outbox Pattern* dengan `SKIP LOCKED` untuk pengiriman event asinkron tanpa *dual-write anomaly*.

---

## 🏗️ Arsitektur Sistem

```
                      +-----------------------------+
                      |   Merchant / Client App     |
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
Setiap transaksi wajib membawa header:
* `X-Signature`: Tanda tangan hex `HMAC-SHA256(timestamp + "." + body, secret_key)`
* `X-Timestamp`: Unix timestamp saat request dibuat

Sistem memverifikasi signature menggunakan algoritma **Constant-Time Comparison** (`hmac.Equal`) untuk menggagalkan **Timing Attacks** (peretas mengukur selisih waktu respons mikrodetik untuk menebak byte signature).

### 2. Anti-Replay Attack Protection
Request yang dicegat oleh pihak ketiga di jaringan publik tidak dapat dikirim ulang berulang kali. Header `X-Timestamp` dievaluasi terhadap clock server dengan toleransi batas maksimal $\pm 300$ detik (5 menit).

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
// Selalu lock firstLockID, baru kemudian secondLockID
```
Hal ini menjamin seluruh proses di seluruh thread selalu meminta lock dalam urutan yang identik, secara matematis **mengeliminasi deadlock**.

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
```

---

## 🚀 Panduan Menjalankan & Pengujian

### 1. Prasyarat
* [Go 1.22+](https://golang.org)
* [Docker & Docker Compose](https://www.docker.com/) (atau PostgreSQL & Redis lokal)

### 2. Jalankan Database & Cache
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
Server akan aktif di `http://localhost:8080`.

### 5. Jalankan Background Outbox Worker
```bash
go run ./cmd/worker
```

---

## 🧪 Contoh Request API

### 1. Buat Rekening Baru
```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "account_number": "ACC-IDR-1001",
    "holder_name": "Budi Santoso",
    "identity_data": "NIK:3171012345670001;PAN:4111222233334444",
    "initial_balance": 10000000,
    "currency": "IDR"
  }'
```

### 2. Generate Signature untuk Transfer
Gunakan helper CLI bawaan untuk menghasilkan header keamanan yang valid:
```bash
go run ./scripts/generate_signature.go
```

Output:
```text
=== Generated Security Headers ===
X-Timestamp: 1776123456
X-Signature: 3c5e8f4...
X-Idempotency-Key: idemp-1776123456
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

Ketika ditanya mengenai proyek ini saat interview:
1. **Tentang Race Condition**: Jelaskan bagaimana pengurutan UUID deterministik memecahkan persoalan circular wait deadlock pada PostgreSQL `SELECT FOR UPDATE`.
2. **Tentang Data Loss**: Jelaskan peran Transactional Outbox Pattern yang mencatat event dalam satu transaksi ACID yang sama dengan mutasi saldo, mencegah masalah *Dual-Write*.
3. **Tentang Keamanan Finansial**: Jelaskan mengapa enkripsi AES-256-GCM dipilih dibanding CBC (karena integritas terotentikasi), dan pentingnya `hmac.Equal` untuk menolak side-channel timing attack.
