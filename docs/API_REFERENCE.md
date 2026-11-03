# REST API Contract Reference (`v1`)

Seluruh endpoint di bawah disajikan pada base URL `http://localhost:8080` dan mengembalikan amplop JSON standar:

```json
{
  "success": true,
  "message": "Human-readable description",
  "data": {},
  "error": {
    "code": "ERROR_CODE",
    "details": "Diagnostic message (only present when success=false)"
  }
}
```

---

## 1. Omni-Channel OMS & Inventory Endpoints

### `GET /api/v1/dashboard/overview`
Mengambil seluruh data operasional multi-merchant dalam satu panggilan teragregasi.

* **Query Parameters (Opsional)**:
  * `merchant_id`: UUID merchant atau kode merchant (`MID-8801`, `MID-8802`, `MID-8803`).
  * `channel`: `WEB` | `WHATSAPP_LIVE` | `SHOPEE` | `TIKTOK_SHOP`
  * `status`: `PENDING_PAYMENT` | `PAID` | `READY_TO_SHIP` | `SHIPPED` | `COMPLETED` | `CANCELLED`

---

### `POST /api/v1/orders/ai-parse`
Mengekstrak teks chat WhatsApp / komentar Live Selling menjadi draft order terstruktur dan memvalidasi kecukupan stok SKU secara langsung.

* **Request Body**:
```json
{
  "merchant_id": "11111111-1111-1111-1111-111111111111",
  "raw_text": "Kak mau pesan Kaos Oversize Hitam XL 2 pcs sama Topi Baseball Navy 1 pcs, kirim ke Budi Santoso 081299887766, Jl. Kaliurang Km 5 No 12 Sleman via JNE REG ya"
}
```
* **Response (`200 OK`)**:
```json
{
  "success": true,
  "message": "WhatsApp/Live chat extracted into structured order draft",
  "data": {
    "customer_name": "Budi Santoso",
    "customer_phone": "081299887766",
    "shipping_address": "Jl. Kaliurang Km 5 No 12 Sleman",
    "courier_code": "JNE",
    "estimated_total": 333000,
    "has_insufficient_stock": true,
    "parser_engine": "hybrid-nlp-fallback",
    "latency_ms": 2,
    "items": [
      {
        "sku_id": "a1111111-1111-1111-1111-111111111101",
        "sku": "SKU-KAOS-BLK-XL",
        "matched_product_name": "Kaos Oversize Hitam XL",
        "qty": 2,
        "unit_price": 129000,
        "subtotal": 258000,
        "available_stock": 1,
        "stock_sufficient": false,
        "stock_warning": "Stok kurang! Tersedia 1 pcs, diminta 2 pcs"
      }
    ]
  }
}
```

---

### `POST /api/v1/orders`
Membuat order baru dan menahan stok (`available_stock` &rarr; `reserved_stock`) menggunakan PostgreSQL `SELECT ... FOR UPDATE`.

* **Request Body**:
```json
{
  "merchant_id": "11111111-1111-1111-1111-111111111111",
  "channel": "WHATSAPP_LIVE",
  "customer_name": "Budi Santoso",
  "customer_phone": "081299887766",
  "shipping_address": "Jl. Kaliurang Km 5 No 12 Sleman",
  "courier_code": "JNE",
  "items": [
    {
      "product_id": "a1111111-1111-1111-1111-111111111102",
      "qty": 1
    }
  ]
}
```
* **Status Codes**:
  * `201 Created`: Stok berhasil dikunci dan order dibuat (`PENDING_PAYMENT`).
  * `409 Conflict` (`INSUFFICIENT_STOCK`): Stok tersedia tidak mencukupi.

---

### `POST /api/v1/orders/{id}/cancel`
Membatalkan order berstatus `PENDING_PAYMENT` dan mengembalikan `reserved_stock` ke `available_stock`.

---

### `POST /api/v1/products/{id}/stock`
Mengatur nilai `available_stock` produk (digunakan untuk administrasi gudang atau persiapan simulasi).

```json
{
  "available_stock": 1
}
```

---

## 2. Secure Payment Webhook & Queue Control

### `POST /api/v1/webhooks/payment`
Endpoint penerima notifikasi pembayaran dengan verifikasi tanda tangan kriptografis dan jaminan idempotency.

* **Required Headers**:
  * `X-Webhook-Id`: ID unik event dari penyedia pembayaran (contoh: `evt-pay-99182`).
  * `X-Webhook-Timestamp`: Unix epoch detik (ditolak jika selisih `> 300` detik).
  * `X-Webhook-Signature`: Hex `HMAC-SHA256(timestamp + "." + raw_body, WEBHOOK_SECRET)`.
* **Request Body**:
```json
{
  "provider": "PAYMENT_GW",
  "event_type": "payment.succeeded",
  "order_id": "b1111111-1111-1111-1111-111111111102",
  "order_number": "ORD-20261031-SHP2",
  "amount": 258000
}
```
* **Status Codes**:
  * `200 OK` (`{"status": "accepted", "idempotent": false}`): Event baru diterima dan didorong ke antrean Redis.
  * `200 OK` (`{"status": "duplicate_ignored", "idempotent": true}`): Event duplikat diabaikan dengan aman.
  * `401 Unauthorized`: Signature tidak cocok atau timestamp kedaluwarsa.

---

### `POST /api/v1/simulator/race-condition`
Menjalankan uji konkurensi paralel (`N` goroutine serentak) memperebutkan stok terbatas pada satu SKU.

```json
{
  "merchant_id": "11111111-1111-1111-1111-111111111111",
  "product_id": "a1111111-1111-1111-1111-111111111101",
  "concurrent_buyers": 10,
  "reset_stock_to": 1
}
```

### `POST /api/v1/simulator/duplicate-webhook`
Menembakkan beberapa webhook identik dengan `X-Webhook-Id` yang sama untuk memverifikasi deduplikasi idempotent.

```json
{
  "merchant_id": "11111111-1111-1111-1111-111111111111",
  "duplicate_count": 5
}
```

### `POST /api/v1/simulator/chaos-mode`
Mengubah mode simulasi kegagalan API pihak ketiga pada worker logistik dan sinkronisasi marketplace.

```json
{
  "mode": "OFF"
}
```
*(Nilai valid: `"OFF"`, `"TRANSIENT_TWICE"`, `"ALWAYS_429"`)*

### `POST /api/v1/queue/jobs/{id}/retry`
Mereset penghitung kegagalan job berstatus `FAILED` (DLQ) dan memasukkannya kembali ke antrean aktif.

---

## 3. Core Payment Switch & Reconciliation Endpoints

| Method | Path | Header Khusus | Deskripsi |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/accounts` | — | Daftar akun settlement beserta saldo dan versi optimistik. |
| `POST` | `/api/v1/accounts` | — | Registrasi akun baru dengan enkripsi identitas `AES-256-GCM`. |
| `GET` | `/api/v1/accounts/{id}` | — | Detail akun beserta dekripsi identitas terotorisasi. |
| `POST` | `/api/v1/transfers` | `X-Timestamp`, `X-Signature`, `X-Idempotency-Key` | Eksekusi transfer saldo atomik *Double-Entry* (`DEBIT` & `CREDIT`). |
| `GET` | `/api/v1/ledger` | — | Daftar entri buku besar berpasangan yang bersifat *append-only*. |
| `GET` | `/api/v1/audit-logs` | — | Riwayat rantai audit kriptografis `SHA-256`. |
| `POST` | `/api/v1/reconciliation/upload` | `multipart/form-data` (`bank_name`, `file`) | Unggah CSV laporan settlement bank untuk *3-Way Matching*. |
| `GET` | `/api/v1/reconciliation/batches` | — | Daftar riwayat batch rekonsiliasi settlement. |
| `GET` | `/api/v1/reconciliation/batches/{id}` | — | Rincian item kecocokan & selisih nominal per batch. |

---

## 4. AI-Powered FinTech Workflows (Vision Slip & Text-to-SQL)

### `POST /api/v1/payments/proof`
Mengunggah file struk transfer bank (multipart/form-data atau JSON payload dengan `image_base64`). Merespons seketika dengan `202 Accepted` (< 20ms) dan menerbitkan event ke RabbitMQ Exchange `oms.events` (routing key `payment.verify`).

* **Request Form / Body**:
  * `order_id` (string, UUID, wajib): ID pesanan yang dibayar.
  * `slip` (file binary multipart, opsional jika `image_base64` diisi): Gambar format `image/jpeg` atau `image/png`.
  * `image_base64` (string text, opsional): Representasi Base64 dari gambar struk atau representasi teks transfer.
* **Response `202 Accepted`**:
```json
{
  "code": 202,
  "message": "Payment slip accepted for asynchronous AI verification",
  "data": {
    "proof_id": "c1111111-1111-1111-1111-111111111199",
    "order_id": "b1111111-1111-1111-1111-111111111101",
    "status": "PROCESSING",
    "trace_id": "tr-abc12345"
  }
}
```

---

### `GET /api/v1/payments/proof/{id}`
Mengambil detail hasil verifikasi bukti transfer Vision AI setelah dikonsumsi oleh RabbitMQ worker.

* **Response `200 OK`**:
```json
{
  "code": 200,
  "data": {
    "id": "c1111111-1111-1111-1111-111111111199",
    "order_id": "b1111111-1111-1111-1111-111111111101",
    "order_number": "ORD-20261104-BCA1",
    "status": "VERIFIED",
    "bank_name": "BCA",
    "sender_name": "EKO PRASETYO",
    "amount": 378000,
    "reference_number": "TRX-982137912",
    "tamper_detected": false,
    "confidence_score": 0.99,
    "extracted_text": "TRANSFER BERHASIL ...",
    "created_at": "2026-11-04T00:15:20Z",
    "verified_at": "2026-11-04T00:15:21Z"
  }
}
```
*(Status: `"PENDING"`, `"PROCESSING"`, `"VERIFIED"`, `"FAILED"`, `"FRAUD_DETECTED"`)*

---

### `GET /api/v1/payments/proof`
Mengambil daftar seluruh riwayat bukti transfer bank merchant.

---

### `POST /api/v1/reports/ask-ai`
Eksekusi query laporan analitik bisnis berbasis bahasa alami (Natural Language Text-to-SQL) dengan AST token sanitizer, read-only PostgreSQL transaction, dan rekomendasi chart visual.

* **Request Body**:
```json
{
  "question": "Berapa total GMV dan jumlah pesanan per status sepanjang minggu ini?",
  "merchant_id": "11111111-1111-1111-1111-111111111111"
}
```
* **Response `200 OK`**:
```json
{
  "code": 200,
  "data": {
    "question": "Berapa total GMV dan jumlah pesanan per status sepanjang minggu ini?",
    "generated_sql": "SELECT status, COUNT(*) AS total_orders, COALESCE(SUM(total_amount), 0) AS gmv FROM orders GROUP BY status ORDER BY gmv DESC LIMIT 100;",
    "explanation": "Query ini menghitung akumulasi total pendapatan (GMV) dan volume pesanan dikelompokkan berdasarkan status.",
    "is_safe": true,
    "execution_time_ms": 14,
    "row_count": 3,
    "columns": ["status", "total_orders", "gmv"],
    "rows": [
      {"status": "PAID", "total_orders": 8, "gmv": 3024000},
      {"status": "READY_TO_SHIP", "total_orders": 4, "gmv": 1512000},
      {"status": "PENDING_PAYMENT", "total_orders": 2, "gmv": 756000}
    ],
    "chart_type": "bar",
    "chart_config": {
      "x_axis": "status",
      "y_axis": "gmv"
    }
  }
}
```

---

### `GET /api/v1/ai/telemetry`
Mengambil metrik observabilitas AI, FinOps token usage, pHash duplicate detection, dan kesehatan Transactional Outbox.

* **Response `200 OK`**:
```json
{
  "code": 200,
  "data": {
    "total_calls": 48,
    "vision_calls": 28,
    "text_to_sql_calls": 20,
    "total_tokens": 34500,
    "total_cost_usd": 0.0078,
    "avg_latency_ms": 285.4,
    "fraud_quarantine_count": 4,
    "fraud_rate_pct": 8.3,
    "ast_blocks_count": 3,
    "duplicate_phash_count": 2,
    "outbox_pending_count": 0,
    "outbox_processed_count": 48
  }
}
```
