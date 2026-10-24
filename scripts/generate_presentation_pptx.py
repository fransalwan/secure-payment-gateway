import sys
import os
from pptx import Presentation
from pptx.util import Inches, Pt
from pptx.dml.color import RGBColor
from pptx.enum.text import PP_ALIGN
from pptx.enum.shapes import MSO_SHAPE

def create_presentation(output_path="Frans_Alwan_Purba_Payment_Gateway_Demo.pptx"):
    prs = Presentation()
    # 16:9 Widescreen slides (13.333 x 7.5 inches)
    prs.slide_width = Inches(13.333)
    prs.slide_height = Inches(7.5)

    blank_slide_layout = prs.slide_layouts[6]

    # Color Palette
    BG_COLOR = RGBColor(11, 15, 25)         # Deep slate/navy #0B0F19
    CARD_BG = RGBColor(30, 41, 59)          # Card slate #1E293B
    CARD_BORDER = RGBColor(51, 65, 85)      # Card border #334155
    TEXT_WHITE = RGBColor(255, 255, 255)
    TEXT_MUTED = RGBColor(148, 163, 184)    # Slate 400 #94A3B8
    EMERALD = RGBColor(16, 185, 129)        # Emerald #10B981
    INDIGO = RGBColor(99, 102, 241)         # Indigo #6366F1
    AMBER = RGBColor(245, 158, 11)          # Amber #F59E0B
    ROSE = RGBColor(244, 63, 94)            # Rose #F43F5E

    def apply_background(slide):
        bg = slide.shapes.add_shape(MSO_SHAPE.RECTANGLE, 0, 0, prs.slide_width, prs.slide_height)
        bg.fill.solid()
        bg.fill.fore_color.rgb = BG_COLOR
        bg.line.fill.background()
        return bg

    def add_header(slide, tag, title, subtitle):
        # Tag pill
        tag_box = slide.shapes.add_textbox(Inches(0.8), Inches(0.5), Inches(11.7), Inches(0.4))
        tf = tag_box.text_frame
        tf.word_wrap = True
        p0 = tf.paragraphs[0]
        p0.text = tag.upper()
        p0.font.size = Pt(10)
        p0.font.bold = True
        p0.font.color.rgb = EMERALD

        # Title
        p1 = tf.add_paragraph()
        p1.text = title
        p1.font.size = Pt(22)
        p1.font.bold = True
        p1.font.color.rgb = TEXT_WHITE
        p1.space_before = Pt(4)

        # Subtitle
        if subtitle:
            p2 = tf.add_paragraph()
            p2.text = subtitle
            p2.font.size = Pt(11)
            p2.font.color.rgb = TEXT_MUTED
            p2.space_before = Pt(2)

    def add_card(slide, left, top, width, height, title, body_bullets, accent_color=EMERALD):
        card = slide.shapes.add_shape(MSO_SHAPE.ROUNDED_RECTANGLE, left, top, width, height)
        card.fill.solid()
        card.fill.fore_color.rgb = CARD_BG
        card.line.color.rgb = CARD_BORDER
        card.line.width = Pt(1.5)

        tf = card.text_frame
        tf.word_wrap = True
        tf.margin_left = Inches(0.25)
        tf.margin_right = Inches(0.25)
        tf.margin_top = Inches(0.2)
        tf.margin_bottom = Inches(0.2)

        p = tf.paragraphs[0]
        p.text = title
        p.font.size = Pt(13)
        p.font.bold = True
        p.font.color.rgb = accent_color

        for bullet in body_bullets:
            p_b = tf.add_paragraph()
            p_b.text = f"• {bullet}"
            p_b.font.size = Pt(10.5)
            p_b.font.color.rgb = TEXT_WHITE
            p_b.space_before = Pt(6)

    def set_speaker_notes(slide, notes_text):
        notes_slide = slide.notes_slide
        tf = notes_slide.notes_text_frame
        tf.text = notes_text

    # =========================================================================
    # SLIDE 1: COVER SLIDE
    # =========================================================================
    slide1 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide1)

    # Accent Top Line
    top_line = slide1.shapes.add_shape(MSO_SHAPE.RECTANGLE, Inches(0.8), Inches(1.2), Inches(3.5), Inches(0.06))
    top_line.fill.solid()
    top_line.fill.fore_color.rgb = EMERALD
    top_line.line.fill.background()

    # Title box
    tbox = slide1.shapes.add_textbox(Inches(0.8), Inches(1.4), Inches(11.7), Inches(3.2))
    tf1 = tbox.text_frame
    tf1.word_wrap = True

    p = tf1.paragraphs[0]
    p.text = "SECURE PAYMENT GATEWAY"
    p.font.size = Pt(36)
    p.font.bold = True
    p.font.color.rgb = TEXT_WHITE

    p2 = tf1.add_paragraph()
    p2.text = "High-Performance Double-Entry Ledger, Concurrency Control & 3-Way Reconciliation"
    p2.font.size = Pt(18)
    p2.font.color.rgb = EMERALD
    p2.space_before = Pt(8)

    p3 = tf1.add_paragraph()
    p3.text = "Arsitektur Financial Core Switch, Pencegah Deadlock Deterministik, dan Zero-Trust Audit Trail"
    p3.font.size = Pt(12)
    p3.font.color.rgb = TEXT_MUTED
    p3.space_before = Pt(10)

    # Presenter Card
    pcard = slide1.shapes.add_shape(MSO_SHAPE.ROUNDED_RECTANGLE, Inches(0.8), Inches(4.8), Inches(11.7), Inches(1.8))
    pcard.fill.solid()
    pcard.fill.fore_color.rgb = CARD_BG
    pcard.line.color.rgb = CARD_BORDER
    pcard.line.width = Pt(1.5)

    ptf = pcard.text_frame
    ptf.margin_left = Inches(0.4)
    ptf.margin_top = Inches(0.25)
    
    pp0 = ptf.paragraphs[0]
    pp0.text = "FRANS ALWAN PURBA"
    pp0.font.size = Pt(18)
    pp0.font.bold = True
    pp0.font.color.rgb = TEXT_WHITE

    pp1 = ptf.add_paragraph()
    pp1.text = "Software Engineer — Golang Backend & High-Availability Systems Focus"
    pp1.font.size = Pt(12)
    pp1.font.color.rgb = INDIGO
    pp1.space_before = Pt(3)

    pp2 = ptf.add_paragraph()
    pp2.text = "Master's Student in Computer Science (Artificial Intelligence) @ Universitas Gadjah Mada (UGM)"
    pp2.font.size = Pt(11)
    pp2.font.color.rgb = TEXT_MUTED
    pp2.space_before = Pt(4)

    set_speaker_notes(slide1, """[PANDUAN PEMBUKA / SPEAKER NOTES SLIDE 1]
"Selamat pagi/siang bapak/ibu tim interviewer. Terima kasih atas kesempatannya hari ini.
Perkenalkan, nama saya Frans Alwan Purba. Saya seorang Software Engineer yang berfokus pada Backend Engineering menggunakan Go, PostgreSQL, dan Redis, serta saat ini sedang menempuh studi Master di Ilmu Komputer Universitas Gadjah Mada.

Hari ini saya ingin mempresentasikan dan mendemokan salah satu portfolio inti yang saya bangun dari nol: Secure Payment Gateway.
Ini bukan sekadar API CRUD transfer biasa, melainkan implementasi menyeluruh dari sebuah financial core switch yang memecahkan 4 masalah paling krusial di fintech:
1. Double-Entry Bookkeeping untuk keterlacakan mutasi seimbang.
2. Pencegahan race condition dan circular deadlock pada transaksi konkuren tinggi.
3. Distributed Idempotency menggunakan Redis atomik.
4. Engine rekonsiliasi 3-arah otomatis untuk penyelarasan mutasi bank eksternal."
""")

    # =========================================================================
    # SLIDE 2: LATAR BELAKANG & FILOSOFI REKAYASA (WHY THIS PROJECT?)
    # =========================================================================
    slide2 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide2)
    add_header(slide2, "LATAR BELAKANG REKAYASA", "Mengapa Membangun Core Payment Engine Ini?", "Memecahkan problem fundamental yang menjadi momok sistem perbankan & payment switch skala tinggi.")

    add_card(slide2, Inches(0.8), Inches(2.0), Inches(3.6), Inches(4.8), 
             "1. Resiko Finansial Nyata", 
             [
                 "Di fintech, bug bukan sekadar crash aplikasi, melainkan hilangnya uang riil atau ketidaksesuaian saldo.",
                 "Kesalahan race condition pada saldo bisa mengakibatkan double-deduction atau saldo negatif nasabah.",
                 "Dibutuhkan arsitektur yang matematis dan bergaransi atomik."
             ], ROSE)

    add_card(slide2, Inches(4.8), Inches(2.0), Inches(3.6), Inches(4.8), 
             "2. Tantangan Rekonsiliasi Bank", 
             [
                 "Integrasi core banking eksternal sering mengalami network timeout atau delay webhook.",
                 "Bagaimana membuktikan apakah transfer yang timeout sebenarnya berhasil di bank atau tidak?",
                 "Diperlukan automated 3-way matching engine berbasis settlement CSV harian."
             ], AMBER)

    add_card(slide2, Inches(8.8), Inches(2.0), Inches(3.6), Inches(4.8), 
             "3. Keamanan Tingkat Nol (Zero-Trust)", 
             [
                 "Data sensitif nasabah (NIK/Kartu) wajib terenkripsi authenticated di level aplikasi (AES-256-GCM).",
                 "API dilindungi tanda tangan HMAC-SHA256 konstan-waktu untuk menolak tampering dan replay attacks.",
                 "Rantai audit tamper-evident menjamin tidak ada admin yang bisa mengubah histori ledger."
             ], EMERALD)

    set_speaker_notes(slide2, """[SPEAKER NOTES SLIDE 2]
"Ketika saya merancang proyek ini, motivasi utama saya adalah menjawab tantangan nyata yang selalu ditanyakan lead engineer di industri fintech:
1. 'Bagaimana sistem Anda menjamin uang tidak pernah berkurang dua kali saat koneksi mobile nasabah tidak stabil?'
2. 'Bagaimana gateway menangani kasus di mana bank mitra timeout, tapi ternyata transaksi nasabah sukses?'
3. 'Bagaimana arsitektur Anda menjamin database tidak mengalami kebuntuan (deadlock) saat ada 1.000 transaksi bersamaan?'

Proyek ini dibangun untuk menjawab ketiga pertanyaan tersebut secara tuntas melalui kombinasi arsitektur database, sistem terdistribusi, dan kriptografi aplikasi."
""")

    # =========================================================================
    # SLIDE 3: HIGH-LEVEL ARCHITECTURE & TECH STACK
    # =========================================================================
    slide3 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide3)
    add_header(slide3, "ARSITEKTUR & STACK TEKNOLOGI", "Arsitektur Modular, Clean Architecture & Ekosistem Go", "Struktur kode modular yang memisahkan domain, repository, service, dan HTTP handler secara ketat.")

    add_card(slide3, Inches(0.8), Inches(2.0), Inches(5.6), Inches(2.3),
             "Backend Core Engine (Go 1.22+)",
             [
                 "Go Chi Router v5 dengan alur middleware berkecepatan tinggi.",
                 "Structured logging terstandarisasi via Go 1.21+ slog (JSON output).",
                 "Non-blocking graceful shutdown untuk menguras in-flight transactions."
             ], INDIGO)

    add_card(slide3, Inches(6.8), Inches(2.0), Inches(5.6), Inches(2.3),
             "Persistence & ACID Guarantees",
             [
                 "PostgreSQL 16 dengan connection pool (pgxpool) teroptimasi.",
                 "Pessimistic row locking (SELECT FOR UPDATE) berurut deterministik.",
                 "Rantai hash SHA-256 tamper-evident pada tabel audit_logs."
             ], EMERALD)

    add_card(slide3, Inches(0.8), Inches(4.6), Inches(5.6), Inches(2.3),
             "Distributed Caching & Concurrency",
             [
                 "Redis 7 untuk distributed lock dan idempotency state machine.",
                 "Fallback in-memory idempotency jika koneksi Redis terganggu.",
                 "Transactional Outbox Pattern dengan PostgreSQL SKIP LOCKED."
             ], AMBER)

    add_card(slide3, Inches(6.8), Inches(4.6), Inches(5.6), Inches(2.3),
             "Operations & Observability",
             [
                 "GitHub Actions CI: pengujian otomatis dengan Go race detector (-race).",
                 "Multi-tenancy merchant context selector & live latency percentile (p95).",
                 "Ready deploy via Cloudflare Tunnel & ngrok untuk live remote demo."
             ], TEXT_WHITE)

    set_speaker_notes(slide3, """[SPEAKER NOTES SLIDE 3]
"Dari segi arsitektur, saya memilih ekosistem Go karena performa konkurensi goroutine yang ringan dan footprint memori yang efisien.
Kode diorganisasikan menggunakan Clean Architecture:
- Paket internal/domain: Menyimpan entitas murni tanpa dependensi eksternal.
- Paket internal/repository: PostgreSQL menggunakan pgxpool dan Redis untuk distributed locking.
- Paket pkg/crypto: Modul kriptografi mandiri untuk AES-256-GCM dan HMAC-SHA256.

Semua pengujian unit dan integrasi lolos pengujian race detector (-race) di GitHub Actions CI runner."
""")

    # =========================================================================
    # SLIDE 4: INVARIANT 1 — DOUBLE-ENTRY BOOKKEEPING
    # =========================================================================
    slide4 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide4)
    add_header(slide4, "PRINSIP FINANSIAL UTAMA #1", "Double-Entry Ledger Engine (Buku Besar Berpasangan)", "Uang tidak pernah diciptakan atau dimusnahkan secara sembarangan.")

    add_card(slide4, Inches(0.8), Inches(2.0), Inches(5.6), Inches(4.8),
             "❌ Flawed Approach (CRUD Biasa)",
             [
                 "Menggunakan query UPDATE langsung:",
                 "  UPDATE accounts SET balance = balance - 100",
                 "  UPDATE accounts SET balance = balance + 100",
                 "Resiko fatal:",
                 "- Jika query kedua gagal, uang lenyap tanpa jejak.",
                 "- Tidak ada histori mutasi audit (tidak terlacak siapa yang mengubah).",
                 "- Rentan anomali saldo negatif jika terjadi race condition."
             ], ROSE)

    add_card(slide4, Inches(6.8), Inches(2.0), Inches(5.6), Inches(4.8),
             "✅ Production Double-Entry Engine",
             [
                 "Setiap transaksi mutlak menghasilkan 2 entri jurnal berpasangan:",
                 "  1. DEBIT: Rekening Pengirim (Asset Outflow -)",
                 "  2. CREDIT: Rekening Penerima (Asset Inflow +)",
                 "Invariant Matematis Mutlak:",
                 "  Σ Debit - Σ Credit = IDR 0.00",
                 "Semua entri bersifat Append-Only (Immutable). Tidak ada UPDATE saldo sembarangan; saldo pasca-transaksi dihitung deterministik."
             ], EMERALD)

    set_speaker_notes(slide4, """[SPEAKER NOTES SLIDE 4]
"Salah satu kesalahan paling sering di portofolio backend pemula adalah melakukan UPDATE balance langsung di tabel akun.
Di industri core banking internasional, prinsip mutlak yang digunakan adalah Double-Entry Bookkeeping.
Setiap kali ada perpindahan dana:
- Akun pengirim dicatat sebagai DEBIT (outflow).
- Akun penerima dicatat sebagai CREDIT (inflow).
- Persamaan dasarnya: Total Debit harus selalu sama dengan Total Credit.
Di dashboard, saya menyediakan fitur 'T-Account Visualizer' sehingga interviewer bisa melihat langsung bahwa untuk setiap transaksi, mutasi kiri dan kanan selalu seimbang dengan selisih nol rupiah."
""")

    # =========================================================================
    # SLIDE 5: INVARIANT 2 — DETERMINISTIC LOCK ORDERING (ANTI-DEADLOCK)
    # =========================================================================
    slide5 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide5)
    add_header(slide5, "PRINSIP KONKURENSI TINGGI #2", "Deterministic Lock Ordering: Eliminasi Deadlock Matematis", "Mencegah fenomena Circular Wait pada database saat volume transaksi tinggi.")

    add_card(slide5, Inches(0.8), Inches(2.0), Inches(5.6), Inches(4.8),
             "Skenario Deadlock Klasik",
             [
                 "Transaksi 1 (User A transfer ke User B):",
                 "  1. Lock Row Akun A (Sukses)",
                 "  2. Mencoba Lock Row Akun B...",
                 "Bersamaan di thread lain:",
                 "Transaksi 2 (User B transfer ke User A):",
                 "  1. Lock Row Akun B (Sukses)",
                 "  2. Mencoba Lock Row Akun A...",
                 "Hasil: Database mengalami Circular Wait Deadlock. Transaksi hang atau di-abort paksa oleh PostgreSQL (error 40P01)!"
             ], ROSE)

    add_card(slide5, Inches(6.8), Inches(2.0), Inches(5.6), Inches(4.8),
             "Solusi: Pengurutan UUID Leksikografis",
             [
                 "Sebelum mengeksekusi SELECT FOR UPDATE, gateway membandingkan UUID kedua rekening:",
                 "  if sourceID < destID {",
                 "      lock(sourceID); lock(destID);",
                 "  } else {",
                 "      lock(destID); lock(sourceID);",
                 "  }",
                 "Hasil: Seluruh transaksi di sistem selalu mengakuisisi baris database dalam urutan yang identik.",
                 "Kondisi Circular Wait tereliminasi secara matematis ($O(1)$ overhead)!"
             ], EMERALD)

    set_speaker_notes(slide5, """[SPEAKER NOTES SLIDE 5]
"Problem konkurensi kedua yang saya selesaikan adalah circular wait deadlock.
Bayangkan dua transaksi terjadi di waktu yang sama: User A kirim uang ke B, dan User B kirim uang ke A. Jika tidak diatur, Transaksi 1 mengunci A lalu menunggu B, sementara Transaksi 2 mengunci B lalu menunggu A. Database akan deadlock.
Solusi yang saya implementasikan di `internal/repository/postgres/transfer_repo.go`:
Sebelum melakukan lock, aplikasi membandingkan string UUID kedua akun. Akun dengan UUID lebih kecil secara leksikografis SELALU dikunci terlebih dahulu, baru kemudian akun kedua.
Karena semua goroutine mengikuti aturan pengurutan global yang sama, circular wait mustahil terjadi. Deadlock dieliminasi secara matematis dengan overhead mendekati nol."
""")

    # =========================================================================
    # SLIDE 6: INVARIANT 3 — DISTRIBUTED IDEMPOTENCY STATE MACHINE
    # =========================================================================
    slide6 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide6)
    add_header(slide6, "SISTEM TERDISTRIBUSI #3", "Distributed Idempotency: Anti-Duplicate Deduction", "Menggunakan Redis atomic SETNX dengan 3-State Machine untuk menangkal spam & network retry.")

    add_card(slide6, Inches(0.8), Inches(2.0), Inches(3.6), Inches(4.8),
             "1. State: ACQUIRED",
             [
                 "Client mengirim request dengan header X-Idempotency-Key.",
                 "Gateway mengeksekusi atomic SETNX di Redis:",
                 "  SETNX idemp:key IN_PROGRESS EX 120",
                 "Jika kunci belum ada, request berhak melanjutkan ke eksekusi mutasi saldo."
             ], INDIGO)

    add_card(slide6, Inches(4.8), Inches(2.0), Inches(3.6), Inches(4.8),
             "2. State: IN_PROGRESS",
             [
                 "Jika ada request paralel lain yang masuk dengan key yang sama saat transaksi pertama masih berjalan:",
                 "Redis menolak akuisisi kunci.",
                 "Gateway seketika mengembalikan HTTP 409 Conflict.",
                 "Mencegah race condition dari tombol yang ditekan berkali-kali!"
             ], AMBER)

    add_card(slide6, Inches(8.8), Inches(2.0), Inches(3.6), Inches(4.8),
             "3. State: COMPLETED",
             [
                 "Setelah mutasi saldo sukses dicatat di database:",
                 "Payload respons JSON di-cache di Redis:",
                 "  SET idemp:key JSON EX 86400",
                 "Jika client melakukan retry jaringan karena timeout sebelumnya, gateway langsung mengembalikan respons dari cache (200 OK) tanpa memutasi saldo berulang!"
             ], EMERALD)

    set_speaker_notes(slide6, """[SPEAKER NOTES SLIDE 6]
"Fitur berikutnya adalah Distributed Idempotency.
Di aplikasi mobile, koneksi sering terputus sesaat setelah transfer dikirim. User sering menekan tombol transfer berkali-kali.
Sistem saya menggunakan 3-State Machine di Redis:
1. Saat request pertama masuk, aplikasi mengambil atomic lock via SETNX dengan status IN_PROGRESS.
2. Jika ada request kedua masuk bersamaan (misal user spam tombol), request kedua ditolak dengan HTTP 409 Conflict.
3. Setelah transaksi pertama berhasil dikomit ke database, status diubah menjadi COMPLETED beserta cache respons JSON-nya.
Jika user retry beberapa detik kemudian, sistem langsung mengembalikan respon sukses dari cache tanpa mendebit ulang rekening nasabah."
""")

    # =========================================================================
    # SLIDE 7: INVARIANT 4 — AUTOMATED 3-WAY BANK RECONCILIATION
    # =========================================================================
    slide7 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide7)
    add_header(slide7, "REKONSILIASI PERBANKAN #4", "Automated 3-Way Bank Reconciliation Engine", "Penyelarasan mutasi laporan settlement bank eksternal terhadap buku besar internal gateway.")

    add_card(slide7, Inches(0.8), Inches(2.0), Inches(3.6), Inches(4.8),
             "MATCHED (100% Cocok)",
             [
                 "Kondisi:",
                 "Reference ID cocok sempurna di Bank dan Gateway.",
                 "Nominal transaksi identik di kedua pihak.",
                 "Status Audit:",
                 "Otomatis ditandai MATCHED (Hijau). Siap untuk settlement buku kas tanpa intervensi manual."
             ], EMERALD)

    add_card(slide7, Inches(4.8), Inches(2.0), Inches(3.6), Inches(4.8),
             "DISCREPANCY_AMOUNT",
             [
                 "Kondisi:",
                 "Reference ID cocok, namun terdapat selisih nominal (misal: Bank catat Rp 120.000, Gateway catat Rp 100.000).",
                 "Penyebab:",
                 "Potongan biaya admin liar, selisih kurs valas, atau fee MDR.",
                 "Sistem menandai selisih nominal (+/- Rp X) untuk investigasi tim finance."
             ], AMBER)

    add_card(slide7, Inches(8.8), Inches(2.0), Inches(3.6), Inches(4.8),
             "MISSING_IN_GATEWAY",
             [
                 "Kondisi:",
                 "Transaksi sukses tercatat di mutasi Bank, tapi tidak ada di database Gateway.",
                 "Penyebab:",
                 "Gateway timeout atau callback webhook bank terputus.",
                 "Sistem mengisolasi anomali ini agar tim operasional bisa melakukan klaim manual atau auto-recovery."
             ], ROSE)

    set_speaker_notes(slide7, """[SPEAKER NOTES SLIDE 7]
"Pertanyaan krusial di setiap wawancara perbankan adalah: 'Bagaimana jika bank mencatat transaksi berhasil, tapi gateway kita timeout?'
Saya mengimplementasikan 3-Way Reconciliation Engine di `pkg/reconciliation` dan `internal/service/reconciliation_service.go`.
Sistem ini mem-parsing file settlement CSV dari bank (misal Bank Mandiri, BCA, atau Custom Bank), lalu mencocokkannya dengan entri ledger internal.
Hasil rekonsiliasi dikategorikan menjadi 3:
1. MATCHED: 100% cocok.
2. DISCREPANCY_AMOUNT: Ada selisih nominal (misal fee admin bank).
3. MISSING_IN_GATEWAY: Bank mencatat uang masuk tapi gateway belum mencatatnya.
Laporan ini bisa diekspor langsung ke file CSV untuk audit harian."
""")

    # =========================================================================
    # SLIDE 8: ZERO-TRUST AUDIT TRAIL & TAMPER DETECTION
    # =========================================================================
    slide8 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide8)
    add_header(slide8, "CYBERSECURITY & INTEGRITAS DATA", "Zero-Trust Security & Tamper-Evident Audit Chain", "Mencegah manipulasi data finansial bahkan dari level internal administrator database.")

    add_card(slide8, Inches(0.8), Inches(2.0), Inches(5.6), Inches(4.8),
             "Zero-Trust Payload & Timing Attacks",
             [
                 "AES-256-GCM Field Encryption:",
                 "  Data sensitif (NIK, Kartu) dienkripsi terotentikasi di memory sebelum disimpan ke tabel accounts.",
                 "Constant-Time HMAC-SHA256:",
                 "  Verifikasi tanda tangan menggunakan hmac.Equal untuk menangkal Side-Channel Timing Attacks.",
                 "Anti-Replay Window (±5 Menit):",
                 "  Request dengan timestamp di luar rentang waktu 5 menit seketika di-drop (HTTP 401)."
             ], INDIGO)

    add_card(slide8, Inches(6.8), Inches(2.0), Inches(5.6), Inches(4.8),
             "Cryptographic Audit Chain (SHA-256)",
             [
                 "Setiap aktivitas mutasi membentuk rantai blok:",
                 "  CurrentHash = SHA256(PreviousHash + Action + Payload + Timestamp)",
                 "Prinsip Blockchain / Git Commit History:",
                 "  Jika ada orang dalam yang mengubah satu angka di database secara ilegal, seluruh rantai hash setelahnya akan rusak (pecah)!",
                 "Fitur Live Demo:",
                 "  Tombol 'Simulate DB Tamper' di dashboard membuktikan deteksi fraud secara instan di depan interviewer."
             ], EMERALD)

    set_speaker_notes(slide8, """[SPEAKER NOTES SLIDE 8]
"Dari sisi cybersecurity, saya menerapkan arsitektur Zero-Trust:
1. Kerahasiaan data: Kolom identitas nasabah dienkripsi dengan AES-256-GCM. Mengapa GCM dan bukan CBC? Karena GCM memiliki tag otentikasi (AEAD) yang mendeteksi perubahan sekecil apa pun pada ciphertext.
2. Integritas request: Signature HMAC diverifikasi dengan perbandingan konstan-waktu (`crypto/subtle`) agar bebas dari timing attack.
3. Rantai audit tak terbantahkan (Tamper-Evident): Setiap log audit merantai hash blok sebelumnya. Di sesi demo nanti, saya akan mendemokan tombol 'Simulate DB Tamper' di mana sistem langsung mendeteksi modifikasi data ilegal dan membunyikan alarm visual."
""")

    # =========================================================================
    # SLIDE 9: RANCANGAN LIVE DEMO (DEMO WALKTHROUGH FLOW)
    # =========================================================================
    slide9 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide9)
    add_header(slide9, "PANDUAN LANGKAH DEMO", "Urutan Demo Interaktif di Depan Interviewer", "Langkah terstruktur 1-klik untuk mendemonstrasikan keunggulan arsitektur secara visual.")

    add_card(slide9, Inches(0.8), Inches(2.0), Inches(3.6), Inches(2.3),
             "1. Normal Atomic Transfer",
             [
                 "Tekan shortcut keyboard [1]",
                 "Lihat 5 tahap pipeline menyala hijau dari AUTH hingga AUDIT.",
                 "Roundtrip latensi tercatat < 15ms."
             ], EMERALD)

    add_card(slide9, Inches(4.8), Inches(2.0), Inches(3.6), Inches(2.3),
             "2. 5x Parallel Race Attack",
             [
                 "Tekan shortcut keyboard [3]",
                 "5 HTTP request paralel ditembakkan serentak via Promise.all().",
                 "Bukti: 1 request 200 OK, 4 request 409 Conflict."
             ], AMBER)

    add_card(slide9, Inches(8.8), Inches(2.0), Inches(3.6), Inches(2.3),
             "3. T-Account Ledger Proof",
             [
                 "Buka tab Double-Entry Ledger",
                 "Klik tombol '🔍 T-Account'",
                 "Tunjukkan bagan Debit vs Credit seimbang mutlak (ΣΔ = Rp 0.00)."
             ], INDIGO)

    add_card(slide9, Inches(0.8), Inches(4.6), Inches(3.6), Inches(2.3),
             "4. Tamper Simulation",
             [
                 "Buka tab Audit Chain (SHA-256)",
                 "Klik 'Simulate DB Tamper'",
                 "Sistem menyalakan alarm merah: 'CRITICAL INTEGRITY VIOLATION DETECTED'."
             ], ROSE)

    add_card(slide9, Inches(4.8), Inches(4.6), Inches(3.6), Inches(2.3),
             "5. 3-Way Bank Recon",
             [
                 "Buka tab Bank Reconciliation",
                 "Klik '1-Click Sample Recon'",
                 "Tunjukkan visual progress bar Matched vs Discrepancy, lalu klik 'Export CSV'."
             ], EMERALD)

    add_card(slide9, Inches(8.8), Inches(4.6), Inches(3.6), Inches(2.3),
             "6. Remote Tunnel Demo",
             [
                 "Jalankan 'make tunnel' (Cloudflare)",
                 "Bagikan link HTTPS publik ke chat room",
                 "Interviewer dapat membuka dashboard di laptop/ponsel mereka sendiri!"
             ], TEXT_WHITE)

    set_speaker_notes(slide9, """[SPEAKER NOTES SLIDE 9]
"Berikut adalah alur 6 langkah yang akan kita demokan secara langsung:
1. Normal Transfer: Memperlihatkan lifecycle pipeline yang melewati 5 tahap verifikasi finansial.
2. Concurrent Race Attack: Menembakkan 5 request paralel di milidetik yang sama membuktikan Redis SETNX memblokir 4 duplikasi dengan status 409 Conflict.
3. T-Account Visualizer: Memperlihatkan bahwa transaksi menghasilkan jurnal Debit dan Credit yang seimbang.
4. Tamper Simulation: Mensimulasikan manipulasi database, di mana verifikasi SHA-256 seketika mendeteksi pelanggaran integritas.
5. 3-Way Reconciliation: Menjalankan engine penyelarasan laporan bank dan mengekspor hasilnya ke CSV.
6. Public Tunnel: Memperlihatkan bahwa interviewer bisa mengakses portal ini langsung dari browser mereka melalui Cloudflare Tunnel."
""")

    # =========================================================================
    # SLIDE 10: CLOSING & Q&A
    # =========================================================================
    slide10 = prs.slides.add_slide(blank_slide_layout)
    apply_background(slide10)

    # Accent line
    top_line10 = slide10.shapes.add_shape(MSO_SHAPE.RECTANGLE, Inches(0.8), Inches(1.5), Inches(3.5), Inches(0.06))
    top_line10.fill.solid()
    top_line10.fill.fore_color.rgb = EMERALD
    top_line10.line.fill.background()

    tbox10 = slide10.shapes.add_textbox(Inches(0.8), Inches(1.8), Inches(11.7), Inches(4.5))
    tf10 = tbox10.text_frame
    tf10.word_wrap = True

    p0 = tf10.paragraphs[0]
    p0.text = "SIAP UNTUK LIVE CODE & DEEP ARCHITECTURE DIVE"
    p0.font.size = Pt(32)
    p0.font.bold = True
    p0.font.color.rgb = TEXT_WHITE

    p1 = tf10.add_paragraph()
    p1.text = "Rangkuman Metrik Utama Portfolio:"
    p1.font.size = Pt(14)
    p1.font.color.rgb = EMERALD
    p1.space_before = Pt(12)

    bullets = [
        "Throughput Skalabilitas: > 1.500 TPS dengan P95 Latency < 30ms.",
        "Zero Invariant Violations: Terbukti 0 saldo negatif dan bebas deadlock circular wait.",
        "Production-Grade Security: Zero-Trust AES-256-GCM, Constant-Time HMAC, SHA-256 Chain.",
        "Comprehensive CI Pipeline: Automated Go test & race condition detector (-race) on GitHub Actions."
    ]
    for b in bullets:
        pb = tf10.add_paragraph()
        pb.text = f"✓  {b}"
        pb.font.size = Pt(12)
        pb.font.color.rgb = TEXT_WHITE
        pb.space_before = Pt(6)

    p_contact = tf10.add_paragraph()
    p_contact.text = "Frans Alwan Purba | fransalwan55@gmail.com | github.com/fransalwan/secure-payment-gateway"
    p_contact.font.size = Pt(11)
    p_contact.font.color.rgb = TEXT_MUTED
    p_contact.space_before = Pt(20)

    set_speaker_notes(slide10, """[SPEAKER NOTES SLIDE 10]
"Sebagai penutup presentasi pengantar ini:
Portfolio ini saya bangun untuk membuktikan bahwa saya tidak hanya menguasai sintaks bahasa Go, tetapi benar-benar memahami standar rekayasa backend industri fintech skala tinggi — mulai dari atomisitas ACID, pengurutan kunci deterministik, penanganan idempotensi terdistribusi, hingga perlindungan kriptografi zero-trust.

Sekarang, mari kita beralih ke sesi live demo aplikasi di browser, atau kita bisa langsung meninjau arsitektur kode dan skenario pengujian unit yang bapak/ibu inginkan. Terima kasih!"
""")

    prs.save(output_path)
    print(f"[SUCCESS] Presentation generated: {output_path}")

if __name__ == "__main__":
    out_file = sys.argv[1] if len(sys.argv) > 1 else "Frans_Alwan_Purba_Payment_Gateway_Demo.pptx"
    create_presentation(out_file)
