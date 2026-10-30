# Production Deployment Guide (Cloud VPS + Docker Compose + Cloudflare)

Dokumen ini menjelaskan cara men-deploy **Omni-Channel Mini E-Commerce OMS & Secure Payment Gateway** secara *always-on* dengan arsitektur multi-container terpisah (**API Receiver**, **Standalone Queue Worker**, **PostgreSQL 16**, **Redis 7**, dan **Caddy Auto-HTTPS**) menggunakan **IDCloudHost Cloud VPS NVMe** (mendukung pembayaran lokal instan via **QRIS / Virtual Account / E-Wallet**) serta domain kustom di belakang **Cloudflare DNS & WAF**.

---

## 1. Topologi Produksi (`docker-compose.prod.yml`)

```text
Internet (HTTPS :443 / HTTP/3 QUIC)
   │
   ▼
[Cloudflare DNS + Proxy + WAF Rate Limit]
   │
   ▼
[caddy:2-alpine]  (:80, :443 -> Automatic TLS Let's Encrypt / ZeroSSL)
   │
   ▼ (Internal Bridge Network: oms_internal)
[oms-api]         (/app/spg-api, ENABLE_EMBEDDED_WORKER=false)
   ├──► [postgres:16-alpine] (SELECT ... FOR UPDATE Row Locks + Double-Entry Ledger)
   └──► [redis:7-alpine]     (Idempotency SET NX + Persistent Job Queue)
          ▲
          │
[oms-worker]      (/app/spg-worker, Independent Background Worker)
   ├──► Memproses Redis Queue (Exponential Backoff: 2s, 4s, 8s, 16s + DLQ)
   └──► Memproses PostgreSQL Transactional Outbox
```

**Keamanan Jaringan Internal:**
- Port `5432` (PostgreSQL), `6379` (Redis), dan `8080` (Go HTTP API) **tidak diekspos ke internet publik**; ketiganya hanya dapat diakses di dalam jaringan privat Docker `oms_internal`.
- Hanya reverse proxy `caddy` yang membuka port `80` dan `443`.
- Binary aplikasi berjalan sebagai *non-root user* (`spg`, `UID 10001`) di atas `alpine:3.21`.

---

## 2. Provisioning Server & Domain via QRIS (IDCloudHost)

1. Buka [console.idcloudhost.com](https://console.idcloudhost.com) dan lakukan **Top-Up Saldo** menggunakan **QRIS** (estimasi Rp 55.000 untuk 1 bulan penuh, atau sesuai durasi pemakaian per jam).
2. Buat **Cloud VPS (Compute NVMe)**:
   - **OS / App Catalog:** Pilih **Ubuntu 24.04 LTS** (atau template **Docker** di *App Catalog*).
   - **Spesifikasi Minimum:** `1 vCPU / 1 GB RAM / 20 GB NVMe` (total penggunaan memori 5 container hanya `~180 MB`).
   - **Location:** **Jakarta / Indonesia** (latensi domestik `~3–8 ms`).
   - Buka port **HTTP (`80`)**, **HTTPS (`443`)**, dan **SSH (`22`)** pada pengaturan Firewall / Security Group.
3. *(Sekalian Domain Resmi)* Daftarkan domain seperti `namakamu.my.id` (~Rp 15.000–25.000/tahun via **QRIS** di portal domain IDCloudHost / DomaiNesia / Niagahoster).

---

## 3. Deploy 1-Perintah di VPS Ubuntu

Login ke VPS melalui terminal PowerShell / SSH:

```bash
ssh root@<IP_PUBLIK_VPS>
```

Clone repository dan jalankan skrip *bootstrap* produksi otomatis:

```bash
git clone https://github.com/fransalwan/secure-payment-gateway.git
cd secure-payment-gateway
chmod +x scripts/deploy-vps.sh

# Opsi A — Test langsung via IP Publik sebelum domain aktif:
./scripts/deploy-vps.sh

# Opsi B — Langsung dengan subdomain produksi (setelah DNS A-record diarahkan ke IP VPS):
./scripts/deploy-vps.sh oms.namakamu.my.id
```

**Apa yang dilakukan `scripts/deploy-vps.sh` secara otomatis?**
1. Menginstal Docker Engine & Docker Compose plugin resmi jika belum terpasang di VPS.
2. Membuat file `.env.prod` (`chmod 600`) berisi *cryptographic secrets* acak (`AES_SECRET_KEY` presisi 32 karakter ASCII untuk `AES-256-GCM`, password acak PostgreSQL & Redis, serta kunci `HMAC_SECRET_KEY` & `WEBHOOK_SECRET`).
3. Menjalankan *multi-stage build* (`Dockerfile`) dan menyalakan kelima container dengan pengecekan `healthcheck` berurutan (`postgres` & `redis` &rarr; `oms-api` &rarr; `oms-worker` & `caddy`).

---

## 4. Konfigurasi Custom Domain & Cloudflare WAF (Gratis Tanpa Kartu Kredit)

1. Buat akun gratis di [dash.cloudflare.com](https://dash.cloudflare.com) &rarr; **Add a Domain** &rarr; masukkan domain `.my.id` milikmu (pilih paket **Free Rp 0**).
2. Ganti *Nameservers* di tempat pembelian domain dengan 2 *Nameservers* dari Cloudflare.
3. Di menu **DNS &rarr; Records** Cloudflare, tambahkan record:
   - **Type:** `A`
   - **Name:** `oms` *(menghasilkan `oms.namakamu.my.id`)*
   - **IPv4 address:** `<IP_PUBLIK_VPS>`
   - **Proxy status:** **Proxied (Orange Cloud)** *(atau DNS Only saat inisialisasi sertifikat pertama kali)*
4. Di menu **SSL/TLS &rarr; Overview** Cloudflare, ubah mode enkripsi menjadi **Full (Strict)**.
5. *(Rekomendasi Proteksi Webhook)* Di menu **Security &rarr; WAF &rarr; Rate limiting rules**, buat 1 aturan gratis:
   - **If incoming requests match:** `URI Path` *starts with* `/api/v1/webhooks/`
   - **Rate:** misal maksimal `60 requests / 10 seconds` per IP &rarr; **Block**.
6. Jika sebelumnya deploy menggunakan IP (`:80`), aktifkan domain HTTPS di VPS cukup dengan satu perintah:
   ```bash
   ./scripts/deploy-vps.sh oms.namakamu.my.id
   ```

---

## 5. Operasional Harian & Zero-Rebuild Secret Preservation

| Kebutuhan Operasional | Perintah di Folder Repo VPS |
| :--- | :--- |
| **Cek status & kesehatan 5 container** | `docker compose --env-file .env.prod -f docker-compose.prod.yml ps` |
| **Pantau log real-time API & Worker** | `make prod-logs` |
| **Tarik update kode terbaru dari GitHub** | `git pull origin main && ./scripts/deploy-vps.sh` |
| **Aktifkan Gemini Live API (Opsional)** | Edit `GEMINI_API_KEY=...` di `.env.prod`, lalu jalankan `./scripts/deploy-vps.sh` |
| **Verifikasi endpoint kesehatan** | `curl -i https://oms.namakamu.my.id/health` |

---

## 6. (Opsional) Auto-Deploy dari GitHub Actions setiap `git push main`

Jika ingin pembaruan otomatis (*Continuous Deployment*) setiap kali kode lolos *unit/integration test* di GitHub:
1. Tambahkan 3 *Repository Secrets* di GitHub (**Settings &rarr; Secrets and variables &rarr; Actions**):
   - `VPS_HOST`: IP publik VPS IDCloudHost
   - `VPS_USER`: `root` (atau user Linux dengan grup `docker`)
   - `VPS_SSH_KEY`: Private SSH key untuk login ke VPS
2. Tambahkan job `deploy` setelah job `test` di `.github/workflows/ci.yml` yang menjalankan:
   ```bash
   cd ~/secure-payment-gateway && git pull origin main && ./scripts/deploy-vps.sh
   ```

