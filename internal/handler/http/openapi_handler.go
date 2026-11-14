package http

import (
	"net/http"
)

// ServeOpenAPISpec serves the OpenAPI 3.0.3 JSON specification for Swagger UI and API consumers.
func ServeOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(openAPISpecJSON))
}

// ServeAPIDocs serves an interactive Swagger UI portal at /api/docs.
func ServeAPIDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerUIHTML))
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>API Documentation — AI-Powered Omni-Channel FinTech OMS & Secure Payment Gateway</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css" />
  <style>
    body { margin: 0; background: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .spg-topbar {
      background: #0f172a;
      color: #f8fafc;
      padding: 14px 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      border-bottom: 1px solid #1e293b;
    }
    .spg-topbar-left { display: flex; align-items: center; gap: 12px; }
    .spg-badge {
      background: #10b981;
      color: #052e16;
      font-weight: 700;
      font-size: 11px;
      padding: 3px 8px;
      border-radius: 999px;
      letter-spacing: 0.04em;
    }
    .spg-links a {
      color: #38bdf8;
      text-decoration: none;
      font-size: 13px;
      font-weight: 600;
      margin-left: 16px;
    }
    .spg-links a:hover { text-decoration: underline; }
  </style>
</head>
<body>
  <div class="spg-topbar">
    <div class="spg-topbar-left">
      <span class="spg-badge">OPENAPI 3.0.3</span>
      <strong>AI-Powered Omni-Channel FinTech OMS &amp; Secure Payment Gateway — Interactive API Reference</strong>
    </div>
    <div class="spg-links">
      <a href="/">← Kembali ke Aplikasi Utama</a>
      <a href="/api/openapi.json" target="_blank">Raw OpenAPI JSON</a>
      <a href="https://github.com/fransalwan/secure-payment-gateway" target="_blank">GitHub Repository</a>
    </div>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "/api/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        docExpansion: "list",
        defaultModelsExpandDepth: 1,
        tryItOutEnabled: true
      });
    };
  </script>
</body>
</html>`

const openAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "AI-Powered Omni-Channel FinTech OMS & Secure Payment Gateway API",
    "version": "1.4.0",
    "description": "Dokumentasi resmi REST API untuk **AI-Powered Omni-Channel FinTech OMS & Secure Payment Gateway** (Kandidat: **Frans Alwan Purba** — Midle Programmer PT. Jiwa Muda Cipta Indonesia `R260032`). Mencakup manajemen stok 3-status dengan penguncian PostgreSQL `SELECT ... FOR UPDATE`, verifikasi bukti bayar AI, buku besar akuntansi berpasangan (*Immutable Double-Entry Ledger*), enkripsi KYC `AES-256-GCM`, serta rekonsiliasi mutasi bank 3-arah.",
    "contact": {
      "name": "Frans Alwan Purba",
      "url": "https://github.com/fransalwan/secure-payment-gateway"
    }
  },
  "servers": [
    {
      "url": "/",
      "description": "Current Active Server (Production VPS / Local)"
    }
  ],
  "tags": [
    { "name": "1. Health & Telemetry", "description": "Healthcheck, queue status, dan telemetri FinOps AI" },
    { "name": "2. Omni-Channel OMS & Warehouse", "description": "Manajemen pesanan multi-channel, reservasi stok ACID 3-status, dan fraud override" },
    { "name": "3. Core Payment & Double-Entry Ledger", "description": "Akun Merchant/Escrow (AES-256-GCM KYC), Double-Entry Ledger, dan SHA-256 Audit Chain" },
    { "name": "4. Bank Reconciliation (3-Way Match)", "description": "Rekonsiliasi CSV rekening koran bank terhadap transaksi dan jurnal buku besar" },
    { "name": "5. Vision AI & Text-to-SQL BI", "description": "Verifikasi resi transfer Vision AI dan laporan analitik Text-to-SQL read-only" },
    { "name": "6. Concurrency & Resilience Simulators", "description": "Simulasi Flash Sale (10 Buyers vs 1 Stock), Duplicate Webhook, dan Reset Demo" }
  ],
  "paths": {
    "/health": {
      "get": {
        "tags": ["1. Health & Telemetry"],
        "summary": "Health Check & Queue Backend Status",
        "responses": {
          "200": { "description": "Service is healthy and connected to PostgreSQL & Redis" }
        }
      }
    },
    "/api/v1/dashboard/overview": {
      "get": {
        "tags": ["2. Omni-Channel OMS & Warehouse"],
        "summary": "Ambil seluruh data KPI, Inventory 3-Status, Order Inbox, & Queue Jobs",
        "responses": {
          "200": { "description": "Dashboard overview payload" }
        }
      }
    },
    "/api/v1/orders/ai-parse": {
      "post": {
        "tags": ["2. Omni-Channel OMS & Warehouse"],
        "summary": "Ekstraksi teks chat pesanan WhatsApp/Instagram menjadi JSON terstruktur",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "channel": { "type": "string", "example": "WHATSAPP" },
                  "raw_text": { "type": "string", "example": "Halo kak mau pesan Kopi Arabika Gayo 2 bungkus kirim ke Jl. Prapanca No. 6 Bantul" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Structured order draft" } }
      }
    },
    "/api/v1/orders": {
      "post": {
        "tags": ["2. Omni-Channel OMS & Warehouse"],
        "summary": "Buat pesanan baru & kunci stok atomik (SELECT ... FOR UPDATE)",
        "responses": { "201": { "description": "Order created and stock reserved" } }
      }
    },
    "/api/v1/orders/{id}/simulate-pay": {
      "post": {
        "tags": ["2. Omni-Channel OMS & Warehouse"],
        "summary": "Lunasi pesanan (PAID), potong stok Reserved -> Sold, & catat jurnal Double-Entry Ledger",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "example": "ORD-20261031-IG05" } }
        ],
        "responses": { "200": { "description": "Order marked PAID and ledger recorded" } }
      }
    },
    "/api/v1/orders/{id}/cancel": {
      "post": {
        "tags": ["2. Omni-Channel OMS & Warehouse"],
        "summary": "Batalkan pesanan / Reject Fraud Hold & kembalikan stok Reserved -> Available",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "example": "ORD-20261031-IG05" } }
        ],
        "responses": { "200": { "description": "Order cancelled and reserved stock released" } }
      }
    },
    "/api/v1/products/{id}/stock": {
      "post": {
        "tags": ["2. Omni-Channel OMS & Warehouse"],
        "summary": "Update jumlah stok fisik gudang (Available Stock)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "example": "SKU-KOPI-01" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": { "available_stock": { "type": "integer", "example": 25 } }
              }
            }
          }
        },
        "responses": { "200": { "description": "Stock updated" } }
      }
    },
    "/api/v1/accounts": {
      "get": {
        "tags": ["3. Core Payment & Double-Entry Ledger"],
        "summary": "Daftar Akun Escrow & Merchant beserta Ciphertext KYC (AES-256-GCM)",
        "responses": { "200": { "description": "List of accounts" } }
      }
    },
    "/api/v1/accounts/{id}": {
      "get": {
        "tags": ["3. Core Payment & Double-Entry Ledger"],
        "summary": "Dekripsi On-Demand data KYC (Rekening Bank & NPWP) menggunakan AES-256-GCM",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "example": "ACC-MERCH-8801" } }
        ],
        "responses": { "200": { "description": "Account detail with decrypted PII" } }
      }
    },
    "/api/v1/ledger": {
      "get": {
        "tags": ["3. Core Payment & Double-Entry Ledger"],
        "summary": "Daftar Jurnal Buku Besar Berpasangan (Immutable Double-Entry Ledger DEBIT/CREDIT)",
        "responses": { "200": { "description": "Ledger entries list" } }
      }
    },
    "/api/v1/audit-logs": {
      "get": {
        "tags": ["3. Core Payment & Double-Entry Ledger"],
        "summary": "Daftar Audit Log Berantai SHA-256 (Tamper-Evident Hash Chain)",
        "responses": { "200": { "description": "Audit log entries" } }
      }
    },
    "/api/v1/simulator/ledger-transfer": {
      "post": {
        "tags": ["3. Core Payment & Double-Entry Ledger"],
        "summary": "Simulasi Settlement Dana Escrow -> Merchant dengan Double-Entry Ledger & SHA-256 Audit Chain",
        "responses": { "200": { "description": "Settlement transfer executed atomically" } }
      }
    },
    "/api/v1/reconciliation/batches": {
      "get": {
        "tags": ["4. Bank Reconciliation (3-Way Match)"],
        "summary": "Daftar Batch Rekonsiliasi Mutasi Bank",
        "responses": { "200": { "description": "Reconciliation batches" } }
      }
    },
    "/api/v1/reconciliation/batches/{id}": {
      "get": {
        "tags": ["4. Bank Reconciliation (3-Way Match)"],
        "summary": "Rincian Baris Hasil Pencocokan 3-Arah (MATCHED, AMOUNT_MISMATCH, ORPHAN_EXTERNAL)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Batch detail items" } }
      }
    },
    "/api/v1/reconciliation/upload": {
      "post": {
        "tags": ["4. Bank Reconciliation (3-Way Match)"],
        "summary": "Upload CSV Mutasi Bank untuk Rekonsiliasi 3-Arah",
        "responses": { "200": { "description": "Reconciliation completed" } }
      }
    },
    "/api/v1/reports/ask-ai": {
      "post": {
        "tags": ["5. Vision AI & Text-to-SQL BI"],
        "summary": "Eksekusi Kueri Analitik Bahasa Alami (Text-to-SQL Read-Only dengan AST Guardrail)",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "question": { "type": "string", "example": "Tampilkan ringkasan total pendapatan berdasarkan channel penjualan" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "SQL query and result rows" } }
      }
    },
    "/api/v1/ai/telemetry": {
      "get": {
        "tags": ["1. Health & Telemetry"],
        "summary": "Ringkasan Telemetri AI FinOps, Latensi, & Outbox Reliability",
        "responses": { "200": { "description": "Telemetry summary" } }
      }
    },
    "/api/v1/simulator/race-condition": {
      "post": {
        "tags": ["6. Concurrency & Resilience Simulators"],
        "summary": "Simulasi Flash Sale: 10 Goroutine Konkuren Menembak 1 Stok Terakhir",
        "responses": { "200": { "description": "1 succeeded, 9 safely rejected (0 oversold)" } }
      }
    },
    "/api/v1/simulator/duplicate-webhook": {
      "post": {
        "tags": ["6. Concurrency & Resilience Simulators"],
        "summary": "Simulasi 5x Retry Webhook Pembayaran Duplikat (Redis Idempotency Guard)",
        "responses": { "200": { "description": "1 processed, 4 deduplicated" } }
      }
    },
    "/api/v1/simulator/reset-demo": {
      "post": {
        "tags": ["6. Concurrency & Resilience Simulators"],
        "summary": "Reset & Seed Ulang Seluruh Data Demo ke Kondisi Awal",
        "responses": { "200": { "description": "Demo state reset" } }
      }
    }
  }
}`
