import os
import sys
from reportlab.lib.pagesizes import A4
from reportlab.lib import colors
from reportlab.lib.units import inch
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.platypus import (
    SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, PageBreak, KeepTogether, HRFlowable
)
from reportlab.pdfgen import canvas

class NumberedCanvas(canvas.Canvas):
    def __init__(self, *args, **kwargs):
        super(NumberedCanvas, self).__init__(*args, **kwargs)
        self._saved_page_states = []

    def showPage(self):
        self._saved_page_states.append(dict(self.__dict__))
        self._startPage()

    def save(self):
        num_pages = len(self._saved_page_states)
        for state in self._saved_page_states:
            self.__dict__.update(state)
            self.draw_page_decorations(num_pages)
            super(NumberedCanvas, self).showPage()
        super(NumberedCanvas, self).save()

    def draw_page_decorations(self, page_count):
        self.saveState()
        self.setFont("Helvetica", 8)
        self.setFillColor(colors.HexColor("#64748b"))

        # Header (Pages > 1)
        if self._pageNumber > 1:
            self.drawString(36, A4[1] - 28, "Secure Payment Gateway — Technical Interview & Narrative Playbook")
            self.setStrokeColor(colors.HexColor("#e2e8f0"))
            self.setLineWidth(0.5)
            self.line(36, A4[1] - 32, A4[0] - 36, A4[1] - 32)

        # Footer (All pages)
        page_text = f"Halaman {self._pageNumber} dari {page_count}"
        self.drawRightString(A4[0] - 36, 25, page_text)
        self.drawString(36, 25, "CONFIDENTIAL — Panduan Wawancara Portfolio Backend Engineer Fintech")
        self.setStrokeColor(colors.HexColor("#e2e8f0"))
        self.setLineWidth(0.5)
        self.line(36, 35, A4[0] - 36, 35)

        self.restoreState()

def build_pdf(filename="Secure_Payment_Gateway_Interview_Playbook.pdf"):
    doc = SimpleDocTemplate(
        filename,
        pagesize=A4,
        leftMargin=36,
        rightMargin=36,
        topMargin=42,
        bottomMargin=42
    )

    styles = getSampleStyleSheet()

    # Custom typography styles
    title_style = ParagraphStyle(
        'DocTitle',
        parent=styles['Heading1'],
        fontName='Helvetica-Bold',
        fontSize=22,
        leading=26,
        textColor=colors.HexColor('#0f172a'),
        spaceAfter=6
    )

    subtitle_style = ParagraphStyle(
        'DocSubTitle',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=11,
        leading=15,
        textColor=colors.HexColor('#059669'),
        spaceAfter=15
    )

    h1_style = ParagraphStyle(
        'SectionH1',
        parent=styles['Heading1'],
        fontName='Helvetica-Bold',
        fontSize=14,
        leading=18,
        textColor=colors.HexColor('#0f172a'),
        spaceBefore=14,
        spaceAfter=8,
        keepWithNext=True
    )

    h2_style = ParagraphStyle(
        'SectionH2',
        parent=styles['Heading2'],
        fontName='Helvetica-Bold',
        fontSize=11,
        leading=15,
        textColor=colors.HexColor('#1e293b'),
        spaceBefore=10,
        spaceAfter=4,
        keepWithNext=True
    )

    body_style = ParagraphStyle(
        'BodyDark',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9.5,
        leading=13.5,
        textColor=colors.HexColor('#334155'),
        spaceAfter=6
    )

    body_bold = ParagraphStyle(
        'BodyBold',
        parent=body_style,
        fontName='Helvetica-Bold'
    )

    callout_style = ParagraphStyle(
        'CalloutText',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9,
        leading=13,
        textColor=colors.HexColor('#0f172a')
    )

    q_style = ParagraphStyle(
        'QuestionStyle',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=10,
        leading=14,
        textColor=colors.HexColor('#047857')
    )

    ans_style = ParagraphStyle(
        'AnswerStyle',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9,
        leading=13,
        textColor=colors.HexColor('#1e293b'),
        spaceAfter=6
    )

    code_style = ParagraphStyle(
        'CodeSnippet',
        parent=styles['Normal'],
        fontName='Courier',
        fontSize=8,
        leading=10.5,
        textColor=colors.HexColor('#0f172a')
    )

    story = []

    # Title Banner Block
    banner_data = [
        [
            Paragraph("<b>SECURE PAYMENT GATEWAY</b><br/><font size=10 color='#64748b'>CORE FINANCIAL SWITCH &amp; DOUBLE-ENTRY SETTLEMENT ENGINE</font><br/><font size=8.5 color='#0f172a'><b>Kandidat: Frans Alwan Purba</b> | Software Engineer (Golang Backend) | M.CS (AI) Universitas Gadjah Mada</font>", title_style),
            Paragraph("<b>PORTFOLIO PLAYBOOK &amp; INTERVIEW CHEATSHEET</b><br/><font size=8 color='#059669'>Panduan Strategis Wawancara Tech Lead / VP of Engineering Fintech</font><br/><font size=7.5 color='#64748b'>fransalwan55@gmail.com &bull; github.com/fransalwan</font>", ParagraphStyle('RightSub', parent=subtitle_style, alignment=2))
        ]
    ]
    banner_table = Table(banner_data, colWidths=[310, 210])
    banner_table.setStyle(TableStyle([
        ('VALIGN', (0, 0), (-1, -1), 'TOP'),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 4),
    ]))
    story.append(banner_table)
    story.append(HRFlowable(width="100%", thickness=2, color=colors.HexColor("#059669"), spaceBefore=4, spaceAfter=12))

    # SECTION 1: ELEVATOR PITCH
    story.append(Paragraph("1. ELEVATOR PITCH: CARA MEMPERKENALKAN DIRI & PROYEK INI", h1_style))
    story.append(Paragraph(
        "Kunci utama saat interviewer berkata <i>'Ceritakan tentang diri kamu dan proyek paling berkesan yang pernah kamu bangun'</i> adalah <b>menghubungkan latar belakang akademis & profesional dengan solusi engineering riil</b>. Hindari berbicara seperti membacakan tutorial.",
        body_style
    ))

    # Elevator Pitch Box
    pitch_30s = (
        "<b>Perkenalan Diri & Elevator Pitch 30 Detik (Opening):</b><br/>"
        "<i>\"Nama saya <b>Frans Alwan Purba</b>. Saya seorang Software Engineer yang berfokus pada Go Backend & High-Availability Systems, serta saat ini menempuh studi Master Ilmu Komputer (AI) di Universitas Gadjah Mada (UGM).<br/>"
        "Salah satu portfolio utama yang saya bangun adalah <b>secure-payment-gateway</b>, sebuah financial core switch & settlement engine berbasis Go, PostgreSQL, dan Redis. "
        "Fokus utama saya adalah memecahkan tantangan krusial fintech: <b>Integritas Finansial</b> (Double-Entry Bookkeeping dan penguncian deterministik anti-deadlock), <b>Ketahanan Konkurensi Tinggi</b> (Distributed Idempotency Redis menolak race condition), serta <b>Penyelarasan Bank Otomatis</b> (Automated 3-Way Reconciliation).\"</i>"
    )
    pitch_table = Table([[Paragraph(pitch_30s, callout_style)]], colWidths=[520])
    pitch_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, -1), colors.HexColor("#ecfdf5")),
        ('BOX', (0, 0), (-1, -1), 1, colors.HexColor("#a7f3d0")),
        ('LEFTPADDING', (0, 0), (-1, -1), 10),
        ('RIGHTPADDING', (0, 0), (-1, -1), 10),
        ('TOPPADDING', (0, 0), (-1, -1), 8),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 8),
    ]))
    story.append(pitch_table)
    story.append(Spacer(1, 8))

    pitch_2m = (
        "<b>Versi 2 Menit (Deep Dive untuk Tech Lead / Principal Engineer):</b><br/>"
        "<i>\"Banyak kandidat backend hanya melakukan operasi CRUD sederhana seperti UPDATE balance = balance - 100. Namun di industri payment switch, pola tersebut sangat berbahaya karena rentan race condition dan tidak memiliki jejak audit.<br/>"
        "Di secure-payment-gateway, saya merancang 4 pilar arsitektur perbankan:<br/>"
        "1. <b>Double-Entry Accounting</b>: Setiap transaksi wajib menghasilkan sepasang entri DEBIT dan CREDIT yang mutlak seimbang (&Sigma;Debit = &Sigma;Credit). Saldo riil akun terlindungi invariant matematis non-negatif.<br/>"
        "2. <b>Deterministic Lock Ordering</b>: Mengeliminasi circular wait deadlock pada PostgreSQL SELECT FOR UPDATE secara matematis dengan membandingkan UUID kedua akun secara leksikografis.<br/>"
        "3. <b>Distributed Idempotency State Machine</b>: Menggunakan Redis atomic SETNX (ACQUIRED, IN_PROGRESS, COMPLETED). Ketika diuji dengan 5 request paralel di milidetik yang sama, 1 mutasi sukses dan 4 request duplikat diblokir instan via 409 Conflict.<br/>"
        "4. <b>Automated 3-Way Reconciliation</b>: Membandingkan settlement CSV bank eksternal terhadap mutasi ledger internal untuk mendeteksi MATCHED, DISCREPANCY_AMOUNT, dan MISSING_IN_GATEWAY secara presisi.\"</i>"
    )
    pitch2_table = Table([[Paragraph(pitch_2m, callout_style)]], colWidths=[520])
    pitch2_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, -1), colors.HexColor("#f8fafc")),
        ('BOX', (0, 0), (-1, -1), 1, colors.HexColor("#cbd5e1")),
        ('LEFTPADDING', (0, 0), (-1, -1), 10),
        ('RIGHTPADDING', (0, 0), (-1, -1), 10),
        ('TOPPADDING', (0, 0), (-1, -1), 8),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 8),
    ]))
    story.append(pitch2_table)
    story.append(Spacer(1, 14))

    # SECTION 2: 5 KEPUTUSAN ARSITEKTUR BESERTA "THE WHY"
    story.append(Paragraph("2. ARSITEKTUR TEKNIS & KEPUTUSAN ENGINEERING (THE 'WHY')", h1_style))
    story.append(Paragraph("Tech Lead fintech akan selalu menguji <b>kenapa</b> Anda memilih solusi tertentu dibanding alternatif yang lebih populer/mudah:", body_style))

    arch_rows = [
        [
            Paragraph("<b>Tantangan Teknis</b>", ParagraphStyle('Th1', parent=body_bold, textColor=colors.white)),
            Paragraph("<b>Solusi yang Diterapkan</b>", ParagraphStyle('Th2', parent=body_bold, textColor=colors.white)),
            Paragraph("<b>Alasan Teknis & Mengapa Bukan yang Lain</b>", ParagraphStyle('Th3', parent=body_bold, textColor=colors.white))
        ],
        [
            Paragraph("<b>Integritas Saldo & Mutasi</b>", body_style),
            Paragraph("Double-Entry Bookkeeping (Debit & Credit)", body_style),
            Paragraph("Mengubah saldo via <code>UPDATE balance</code> tidak memiliki jejak audit dan rawan 'uang hilang'. Dengan double-entry, setiap rupiah berpindah wajib tercatat di <code>ledger_entries</code> berpasangan. Kehilangan dana langsung terdeteksi jika total debit &ne; kredit.", body_style)
        ],
        [
            Paragraph("<b>Concurrency & Deadlock</b>", body_style),
            Paragraph("Deterministic Lock Ordering via UUID", body_style),
            Paragraph("Jika Transaksi 1 lock Akun A lalu B, sementara Transaksi 2 lock Akun B lalu A, akan terjadi deadlock database. Dengan selalu mengurutkan akun (min UUID dulu, lalu max UUID), circular wait deadlock dieliminasi secara matematis.", body_style)
        ],
        [
            Paragraph("<b>Mencegah Double-Deduct</b>", body_style),
            Paragraph("Distributed Idempotency (Redis SETNX)", body_style),
            Paragraph("Menyimpan status <code>PROCESSING</code> ber-TTL. Request konkuren ditolak dengan 409 Conflict. Request sukses meng-cache respons utuh sehingga pengulangan retry network langsung mengembalikan response tanpa menyentuh saldo lagi.", body_style)
        ],
        [
            Paragraph("<b>Pencegahan MITM & Timing Attacks</b>", body_style),
            Paragraph("HMAC-SHA256 + <code>hmac.Equal</code>", body_style),
            Paragraph("Perbandingan string biasa (<code>==</code>) rentan kebocoran waktu mikrodetik (timing attack). <code>hmac.Equal</code> berjalan konstan waktu. Ditambah toleransi timestamp &plusmn;5 menit untuk menangkal Replay Attack.", body_style)
        ],
        [
            Paragraph("<b>Reliable Webhooks</b>", body_style),
            Paragraph("Transactional Outbox Pattern (SKIP LOCKED)", body_style),
            Paragraph("Memanggil broker (Kafka/RabbitMQ) langsung di handler rentan dual-write (DB commit sukses tapi broker down). Outbox pattern menyimpan event di tabel DB yang sama dalam 1 transaksi ACID, lalu worker memprosesnya aman.", body_style)
        ],
        [
            Paragraph("<b>Penyelarasan Bank (Rekonsiliasi)</b>", body_style),
            Paragraph("Automated 3-Way Matching Engine", body_style),
            Paragraph("Membandingkan laporan settlement eksternal perbankan (CSV) terhadap mutasi ledger internal secara otomatis untuk mendeteksi selisih nominal (discrepancy), transaksi hilang di gateway, atau transaksi hilang di bank.", body_style)
        ]
    ]

    arch_table = Table(arch_rows, colWidths=[110, 140, 270])
    arch_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, 0), colors.HexColor("#0f172a")),
        ('ALIGN', (0, 0), (-1, -1), 'LEFT'),
        ('VALIGN', (0, 0), (-1, -1), 'TOP'),
        ('GRID', (0, 0), (-1, -1), 0.5, colors.HexColor("#cbd5e1")),
        ('ROWBACKGROUNDS', (0, 1), (-1, -1), [colors.white, colors.HexColor("#f8fafc")]),
        ('TOPPADDING', (0, 0), (-1, -1), 6),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 6),
        ('LEFTPADDING', (0, 0), (-1, -1), 6),
        ('RIGHTPADDING', (0, 0), (-1, -1), 6),
    ]))
    story.append(arch_table)
    story.append(Spacer(1, 14))

    # SECTION 3: TOP 10 INTERVIEW DRILLS & BULLETPROOF ANSWERS
    story.append(Paragraph("3. 10 PERTANYAAN WAWANCARA FINTECH PALING MENJEBAK & JAWABANNYA", h1_style))
    story.append(Paragraph("Hafalkan dan pahami intisari dari 10 pertanyaan teknis ini sebelum masuk ke sesi interview:", body_style))

    drills = [
        (
            "Q1. Bagaimana sistem Anda memastikan saldo akun tidak pernah minus saat ditimpa banyak request simultan?",
            "Saya menerapkan 3 lapis perlindungan: Pertama, di level skema PostgreSQL terdapat database constraint <code>CHECK (balance >= 0)</code>. Kedua, di level aplikasi kami menggunakan Pessimistic Row-Level Locking (<code>SELECT ... FOR UPDATE</code>) sehingga hanya satu transaksi yang dapat memvalidasi dan memotong saldo pada satu waktu. Ketiga, mutasi dicatat secara atomik bersama entri ledger di dalam transaksi ACID."
        ),
        (
            "Q2. Apa itu circular wait deadlock pada PostgreSQL dan bagaimana kode Anda mencegahnya?",
            "Deadlock terjadi jika dua transaksi mengunci resource dengan urutan berlawanan. Contoh: Transaksi X mengunci Akun A dan ingin mengunci B, sedangkan Transaksi Y mengunci Akun B dan ingin mengunci A. Database akan mendeteksi deadlock dan me-rollback salah satunya. Solusi saya: sebelum mengeksekusi <code>SELECT FOR UPDATE</code>, kedua UUID akun diurutkan secara deterministik (secara leksikografis). Sehingga proses mana pun selalu mengunci akun bernilai lebih kecil terlebih dahulu, menghilangkan kondisi circular wait secara absolut."
        ),
        (
            "Q3. Mengapa menggunakan AES-256-GCM dan bukan AES-CBC untuk enkripsi nomor identitas/kartu?",
            "AES-CBC hanya menyediakan kerahasiaan (*confidentiality*), tetapi tidak menyediakan otentikasi data (*integrity*). Tanpa MAC terintegrasi, ciphertext rentan terhadap manipulasi bit-flipping attack atau padding oracle attack. Sebaliknya, AES-256-GCM adalah Authenticated Encryption with Associated Data (AEAD) yang menyertakan authentication tag. Jika seseorang memanipulasi 1 bit saja di database, proses dekripsi akan langsung gagal dan melempar error."
        ),
        (
            "Q4. Apa bahaya membandingkan tanda tangan HMAC menggunakan operator string biasa (==)?",
            "Operator perbandingan biasa (<code>a == b</code>) berhenti mengevaluasi begitu menemukan karakter pertama yang berbeda (*early return*). Peretas dapat mengukur perbedaan waktu respons dalam skala nanodetik untuk menebak byte demi byte signature (Timing Attack). Di Go, saya menggunakan fungsi <code>crypto/subtle.ConstantTimeCompare</code> atau <code>hmac.Equal</code> yang membutuhkan waktu komputasi identik berapa pun karakter yang cocok."
        ),
        (
            "Q5. Jelaskan bagaimana Idempotency State Machine Anda menangani skenario network drop!",
            "Ketika klien mengirim request dengan <code>X-Idempotency-Key</code>, gateway melakukan <code>SETNX</code> ke Redis dengan nilai <code>PROCESSING</code>. Jika network putus saat transfer sedang berjalan dan klien mengirim ulang, gateway membaca status <code>PROCESSING</code> dan merespons dengan <code>409 Conflict</code> (transaksi sedang berjalan, jangan digandakan). Setelah transaksi DB sukses commit, nilai di Redis di-update menjadi <code>COMPLETED</code> beserta serialisasi JSON responsnya. Request retry ketiga langsung menerima respons sukses yang di-cache tanpa mengeksekusi transfer baru."
        ),
        (
            "Q6. Apa masalah yang diselesaikan oleh Transactional Outbox Pattern?",
            "Ini menyelesaikan masalah <i>Dual-Write Anomaly</i>. Jika kita menyimpan transaksi ke database lalu langsung mempublikasikan event ke Kafka atau webhook merchant, salah satu operasi bisa gagal (misal DB commit sukses, tapi broker down, atau sebaliknya). Dengan Outbox Pattern, event dicatat ke tabel <code>outbox_events</code> dalam 1 transaksi ACID database yang sama dengan mutasi ledger. Background worker kemudian membaca event dengan kueri <code>FOR UPDATE SKIP LOCKED</code> untuk meneruskannya secara andal."
        ),
        (
            "Q7. Mengapa audit log Anda menggunakan hash-chaining seperti blockchain mini?",
            "Di industri keuangan yang diatur ketat (regulasi OJK / BI / PCI-DSS), audit trail harus bersifat <i>tamper-evident</i>. Jika peretas atau bahkan database administrator nakal mengedit atau menyisipkan baris di tabel <code>audit_logs</code>, hash pada baris tersebut dan baris-baris berikutnya akan menjadi tidak valid (karena <code>CurrentHash = SHA256(PreviousHash + Payload)</code>). Integritas seluruh riwayat audit dapat diverifikasi secara matematis kapan saja."
        ),
        (
            "Q8. Mengapa memilih Go (Golang) dan Chi router dibanding framework berat?",
            "Go adalah standar industri fintech masa kini (digunakan di GoTo Financial, Xendit, DANA, Flip) karena konkurensi goroutine-nya yang ultra-ringan (overhead memori ~2KB per goroutine) dan garbage collector dengan latensi sub-milidetik. Chi router dipilih karena 100% kompatibel dengan standard library <code>net/http</code>, zero-allocation context routing, dan tidak mengaburkan kode dengan 'magic abstraction' yang sulit di-profiling."
        ),
        (
            "Q9. Bagaimana strategi penanganan database failure atau Redis downtime?",
            "Pada sistem ini, Redis digunakan untuk caching respons idempotensi dan distributed locking. Namun jika Redis tiba-tiba unreachable, kode saya tidak panik ataupun crash; melainkan mengaktifkan fallback in-memory sync lock secara otomatis dengan structured warning log. Sementara database PostgreSQL dikonfigurasi dengan connection pool (<code>pgxpool</code>) yang mengatur min/max connection, connection lifetime, dan automatic health check ping."
        ),
        (
            "Q10. Berapa performa yang Anda capai saat stress testing dan bagaimana membuktikannya?",
            "Menggunakan tool k6, saya mensimulasikan ratusan Virtual Users mengirim transaksi transfer simultan. Hasilnya menghasilkan latensi p95 di bawah 45ms dan p99 di bawah 90ms. Yang paling krusial, pada skenario ekstrim di mana ratusan user bersaing menguras saldo 1 rekening yang sama, konsistensi saldo terbukti 100% akurat dengan 0 overdraft violation."
        ),
        (
            "Q11. Mengapa gateway tidak menyediakan endpoint untuk men-generate signature?",
            "Menyediakan endpoint signing di server payment gateway adalah kerentanan keamanan (*security backdoor*) yang fatal. Di dunia nyata, tanda tangan HMAC-SHA256 wajib di-generate secara independen di sisi merchant menggunakan Merchant Secret Key mereka (via SDK atau Web Crypto). Gateway berfungsi murni sebagai verifikator zero-trust tanpa pernah mengekspos endpoint penandatanganan payload publik."
        )
    ]

    for q_text, a_text in drills:
        drill_box = [
            [Paragraph(q_text, q_style)],
            [Spacer(1, 2)],
            [Paragraph(a_text, ans_style)]
        ]
        t = Table(drill_box, colWidths=[520])
        t.setStyle(TableStyle([
            ('BACKGROUND', (0, 0), (-1, -1), colors.HexColor("#f8fafc")),
            ('BOX', (0, 0), (-1, -1), 0.5, colors.HexColor("#e2e8f0")),
            ('LEFTPADDING', (0, 0), (-1, -1), 8),
            ('RIGHTPADDING', (0, 0), (-1, -1), 8),
            ('TOPPADDING', (0, 0), (-1, -1), 6),
            ('BOTTOMPADDING', (0, 0), (-1, -1), 6),
        ]))
        story.append(t)
        story.append(Spacer(1, 6))

    story.append(Spacer(1, 8))

    # SECTION 4: LIVE DEMO PLAYBOOK (5 MENIT DEMO DI DEPAN INTERVIEWER)
    story.append(Paragraph("4. LIVE DEMO PLAYBOOK: LANGKAH MEMUKAU SAAT LIVE CODING / INTERVIEW", h1_style))
    story.append(Paragraph("Jika interviewer meminta melihat aplikasi berjalan, buka <code>http://localhost:8080/</code> (atau via public tunnel <code>make tunnel</code>) dan gunakan keyboard shortcut angka untuk presentasi cepat:", body_style))

    steps_data = [
        [
            Paragraph("<b>Langkah</b>", body_bold),
            Paragraph("<b>Tindakan / Shortcut</b>", body_bold),
            Paragraph("<b>Narasi yang Diucapkan</b>", body_bold)
        ],
        [
            Paragraph("<b>Step 1</b>", body_style),
            Paragraph("Buka Dashboard atau jalankan <code>make tunnel</code>.", body_style),
            Paragraph("<i>\"Di sini kita bisa melihat Enterprise Operations Console yang memantau core switch secara real-time, lengkap dengan metrik p95 latency dan multi-tenant merchant switcher.\"</i>", body_style)
        ],
        [
            Paragraph("<b>Step 2</b>", body_style),
            Paragraph("Tekan keyboard <b>[1]</b> (Normal Transfer).", body_style),
            Paragraph("<i>\"Saat saya picu transfer, transaksi melewati 5 tahap State Machine: dari verifikasi HMAC, Redis dedup, ACID row lock, mutasi Double-Entry, hingga audit chaining. Semua tahap menyala hijau dalam waktu < 15ms.\"</i>", body_style)
        ],
        [
            Paragraph("<b>Step 3</b>", body_style),
            Paragraph("Tekan keyboard <b>[3]</b> (5x Concurrent Race Attack).", body_style),
            Paragraph("<i>\"Sekarang saya simulasikan serangan race condition: 5 request HTTP ditembakkan serentak di milidetik yang sama via Promise.all(). Perhatikan hasilnya: Request #1 berhasil (200 OK), sementara Request #2 s/d #5 seketika diblokir oleh Redis SETNX dengan status 409 Conflict. Saldo tidak pernah terdebit dua kali.\"</i>", body_style)
        ],
        [
            Paragraph("<b>Step 4</b>", body_style),
            Paragraph("Buka tab Double-Entry & klik tombol <b>🔍 T-Account</b>.", body_style),
            Paragraph("<i>\"Setiap mutasi tidak pernah mengubah saldo secara sembarangan, melainkan membentuk bagan T-Account berpasangan: DEBIT pada pengirim dan CREDIT pada penerima dengan selisih mutlak IDR 0.00.\"</i>", body_style)
        ],
        [
            Paragraph("<b>Step 5</b>", body_style),
            Paragraph("Buka tab Audit Chain & klik <b>Simulate DB Tamper</b>.", body_style),
            Paragraph("<i>\"Ini membuktikan Zero-Trust Audit Trail: ketika saya simulasikan manipulasi ilegal pada payload di database, sistem seketika mendeteksi ketidaksesuaian SHA-256 dan menyalakan alarm pelanggaran integritas data.\"</i>", body_style)
        ],
        [
            Paragraph("<b>Step 6</b>", body_style),
            Paragraph("Buka tab Bank Recon, klik <b>1-Click Recon</b> lalu <b>Export CSV</b>.", body_style),
            Paragraph("<i>\"Di sini engine 3-way matching menyelaraskan laporan mutasi bank eksternal terhadap ledger internal, memisahkan transaksi 100% MATCHED, selisih biaya admin (DISCREPANCY), dan transaksi tertunda (MISSING_IN_GATEWAY) yang siap diekspor ke CSV.\"</i>", body_style)
        ]
    ]

    steps_table = Table(steps_data, colWidths=[60, 160, 300])
    steps_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, 0), colors.HexColor("#0f172a")),
        ('TEXTCOLOR', (0, 0), (-1, 0), colors.white),
        ('ALIGN', (0, 0), (-1, -1), 'LEFT'),
        ('VALIGN', (0, 0), (-1, -1), 'TOP'),
        ('GRID', (0, 0), (-1, -1), 0.5, colors.HexColor("#cbd5e1")),
        ('ROWBACKGROUNDS', (0, 1), (-1, -1), [colors.white, colors.HexColor("#f8fafc")]),
        ('TOPPADDING', (0, 0), (-1, -1), 5),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 5),
        ('LEFTPADDING', (0, 0), (-1, -1), 6),
        ('RIGHTPADDING', (0, 0), (-1, -1), 6),
    ]))
    story.append(steps_table)
    story.append(Spacer(1, 14))

    # SECTION 5: CV BULLETS & LINKEDIN PITCH
    story.append(Paragraph("5. BULLET POINTS CV & LINKEDIN (SIAP PAKAI)", h1_style))
    cv_box = (
        "<b>Bahasa Inggris (Rekomendasi Utama untuk CV Internasional / Unicorn Fintech):</b><br/>"
        "&bull; <b>Architected an ACID-compliant core payment engine</b> in Go utilizing a double-entry bookkeeping ledger (&Sigma;Debit = &Sigma;Credit), achieving sub-45ms p95 latency under high-concurrency benchmarks.<br/>"
        "&bull; <b>Prevented race conditions & database deadlocks</b> by enforcing deterministic UUID lock ordering with PostgreSQL row-level locks (<code>SELECT FOR UPDATE</code>) and a distributed Redis idempotency state machine.<br/>"
        "&bull; <b>Hardened API security posture</b> with field-level authenticated AES-256-GCM encryption for customer identities, constant-time HMAC-SHA256 signature verification against timing attacks, and anti-replay timestamps.<br/>"
        "&bull; <b>Engineered reliable asynchronous event processing</b> using the Transactional Outbox Pattern with <code>FOR UPDATE SKIP LOCKED</code>, eliminating dual-write inconsistencies.<br/><br/>"
        "<b>Bahasa Indonesia:</b><br/>"
        "&bull; Merancang backend payment gateway & buku besar berpasangan (*double-entry ledger*) berbasis Go dan PostgreSQL yang menjamin konsistensi saldo secara atomik dengan latensi p95 &lt; 45ms.<br/>"
        "&bull; Mengeliminasi deadlock konkurensi melalui algoritma pengurutan lock deterministik dan mencegah pemotongan ganda (*double-deduct*) via *distributed idempotency* berbasis Redis.<br/>"
        "&bull; Mengamankan sistem dengan enkripsi AES-256-GCM, verifikasi tanda tangan HMAC-SHA256 konstan-waktu, serta audit log *tamper-evident* berbasis rantai hash SHA-256."
    )
    cv_table = Table([[Paragraph(cv_box, callout_style)]], colWidths=[520])
    cv_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, -1), colors.HexColor("#f1f5f9")),
        ('BOX', (0, 0), (-1, -1), 1, colors.HexColor("#94a3b8")),
        ('LEFTPADDING', (0, 0), (-1, -1), 10),
        ('RIGHTPADDING', (0, 0), (-1, -1), 10),
        ('TOPPADDING', (0, 0), (-1, -1), 8),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 8),
    ]))
    story.append(cv_table)

    # Build Document
    doc.build(story, canvasmaker=NumberedCanvas)
    print(f"PDF successfully built: {filename}")

if __name__ == '__main__':
    output_filename = "Secure_Payment_Gateway_Interview_Playbook.pdf"
    if len(sys.argv) > 1:
        output_filename = sys.argv[1]
    build_pdf(output_filename)

