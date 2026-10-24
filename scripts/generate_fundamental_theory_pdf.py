import os
import sys
from reportlab.lib.pagesizes import A4
from reportlab.lib import colors
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
            self.drawString(36, A4[1] - 28, "Financial Engineering & Core Switch Systems — Panduan Teori Fundamental Lengkap")
            self.drawRightString(A4[0] - 36, A4[1] - 28, "Frans Alwan Purba | UGM")
            self.setStrokeColor(colors.HexColor("#e2e8f0"))
            self.setLineWidth(0.5)
            self.line(36, A4[1] - 32, A4[0] - 36, A4[1] - 32)

        # Footer (All pages)
        page_text = f"Halaman {self._pageNumber} dari {page_count}"
        self.drawRightString(A4[0] - 36, 25, page_text)
        self.drawString(36, 25, "RAHASIA / CONFIDENTIAL — Panduan Wawancara Teknis Backend Fintech Skala Tinggi")
        self.setStrokeColor(colors.HexColor("#e2e8f0"))
        self.setLineWidth(0.5)
        self.line(36, 35, A4[0] - 36, 35)

        self.restoreState()

def build_pdf(filename="Secure_Payment_Gateway_Fundamental_Engineering_Manual.pdf"):
    doc = SimpleDocTemplate(
        filename,
        pagesize=A4,
        leftMargin=36,
        rightMargin=36,
        topMargin=42,
        bottomMargin=42
    )

    styles = getSampleStyleSheet()

    # Typography Styles
    title_style = ParagraphStyle(
        'DocTitle',
        parent=styles['Heading1'],
        fontName='Helvetica-Bold',
        fontSize=20,
        leading=24,
        textColor=colors.HexColor('#0f172a'),
        spaceAfter=4
    )

    subtitle_style = ParagraphStyle(
        'DocSubTitle',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=10,
        leading=14,
        textColor=colors.HexColor('#059669'),
        spaceAfter=12
    )

    h1_style = ParagraphStyle(
        'SectionH1',
        parent=styles['Heading1'],
        fontName='Helvetica-Bold',
        fontSize=13,
        leading=17,
        textColor=colors.HexColor('#0f172a'),
        spaceBefore=14,
        spaceAfter=6,
        keepWithNext=True
    )

    h2_style = ParagraphStyle(
        'SectionH2',
        parent=styles['Heading2'],
        fontName='Helvetica-Bold',
        fontSize=10.5,
        leading=14,
        textColor=colors.HexColor('#1e293b'),
        spaceBefore=10,
        spaceAfter=4,
        keepWithNext=True
    )

    body_style = ParagraphStyle(
        'BodyDark',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9,
        leading=13,
        textColor=colors.HexColor('#334155'),
        spaceAfter=5
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
        fontSize=8.5,
        leading=12.5,
        textColor=colors.HexColor('#0f172a')
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

    # =========================================================================
    # HEADER BANNER
    # =========================================================================
    banner_data = [
        [
            Paragraph("<b>FINANCIAL ENGINEERING &amp; CORE SWITCH SYSTEMS</b><br/><font size=9 color='#64748b'>BUKU PANDUAN TEORI FUNDAMENTAL UNTUK SENIOR BACKEND INTERVIEW</font><br/><font size=8.5 color='#0f172a'><b>Kandidat: Frans Alwan Purba</b> | M.CS Student (AI) Universitas Gadjah Mada</font>", title_style),
            Paragraph("<b>KATEGORI KOMPETENSI</b><br/><font size=8 color='#059669'>Core Banking Ledger &bull; Concurrency &bull; Distributed Systems &bull; Cryptography &bull; 3-Way Reconciliation</font><br/><font size=7.5 color='#64748b'>Repo: github.com/fransalwan/secure-payment-gateway</font>", ParagraphStyle('RightSub', parent=subtitle_style, alignment=2))
        ]
    ]
    banner_table = Table(banner_data, colWidths=[320, 200])
    banner_table.setStyle(TableStyle([
        ('VALIGN', (0, 0), (-1, -1), 'TOP'),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 4),
    ]))
    story.append(banner_table)
    story.append(HRFlowable(width="100%", thickness=2, color=colors.HexColor("#059669"), spaceBefore=4, spaceAfter=10))

    # =========================================================================
    # INTRODUCTORY NOTE
    # =========================================================================
    intro_p = (
        "<b>Catatan Strategis untuk Frans Alwan Purba:</b> Dokumen ini dirancang khusus sebagai panduan penguasaan teori tingkat tinggi. "
        "Saat diwawancarai oleh VP of Engineering atau Principal Architect fintech, pertanyaan yang diajukan bukan lagi tentang 'bagaimana sintaksis Go bekerja', "
        "melainkan <b>mengapa Anda memilih arsitektur tersebut, apa invariant matematis yang dipertahankan, dan apa konsekuensinya pada saat beban transaksi mencapai 10.000 TPS</b>. "
        "Gunakan analogi dan hukum fundamental di bawah ini saat bernarasi."
    )
    intro_table = Table([[Paragraph(intro_p, callout_style)]], colWidths=[520])
    intro_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, -1), colors.HexColor("#f8fafc")),
        ('BOX', (0, 0), (-1, -1), 1, colors.HexColor("#cbd5e1")),
        ('LEFTPADDING', (0, 0), (-1, -1), 10),
        ('RIGHTPADDING', (0, 0), (-1, -1), 10),
        ('TOPPADDING', (0, 0), (-1, -1), 8),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 8),
    ]))
    story.append(intro_table)
    story.append(Spacer(1, 10))

    # =========================================================================
    # BAB 1: TEORI DASAR AKUNTANSI PERBANKAN & DOUBLE-ENTRY BOOKKEEPING
    # =========================================================================
    story.append(Paragraph("BAB 1: MATEMATIKA AKUNTANSI &amp; PRINSIP DOUBLE-ENTRY BOOKKEEPING", h1_style))
    
    story.append(Paragraph("1.1 Saldo Hanyalah Proyeksi (*Balances are Derived Projections, Not Facts*)", h2_style))
    story.append(Paragraph(
        "Di sistem CRUD amatir, uang disimpan sebagai kolom tunggal: <code>accounts.balance</code>. Ketika ada transfer, developer menjalankan "
        "<code>UPDATE accounts SET balance = balance - 100</code>. Ini adalah **dosa terbesar rekayasa finansial** karena melanggar prinsip *Auditability* dan *Immutability*. "
        "Jika ada anomali atau data corrupt, tidak ada cara untuk membuktikan dari mana hilangnya angka tersebut. "
        "Dalam core banking profesional (Stripe, Twilio, Bank Mandiri), **fakta historis adalah mutasi (ledger entries)**, sedangkan angka saldo hanyalah *cache terhitung* atau proyeksi fungsi:",
        body_style
    ))
    story.append(Paragraph(
        "<b>Persamaan Akuntansi Fundamental (Luca Pacioli, 1494):</b><br/>"
        "$$\\text{Assets} = \\text{Liabilities} + \\text{Equity}$$<br/>"
        "Di dalam sistem payment switch kita, invariant mutlak yang dipertahankan adalah:<br/>"
        "$$\\sum_{i=1}^{N} \\text{Debit}_i - \\sum_{i=1}^{N} \\text{Credit}_i = 0.00$$",
        callout_style
    ))
    story.append(Spacer(1, 4))

    story.append(Paragraph("1.2 Bahaya Floating-Point (IEEE 754) vs Fixed-Point Sub-Units", h2_style))
    story.append(Paragraph(
        "Mengapa tipe data <code>float32</code> atau <code>float64</code> haram digunakan untuk uang? Komputer merepresentasikan bilangan pecahan biner berbasis standar IEEE 754. "
        "Angka desimal sederhana seperti <code>0.1 + 0.2</code> menghasilkan <code>0.30000000000000004</code>. Dalam volume 1.000.000 transaksi harian, akumulasi galat pembulatan (*rounding drift*) "
        "dapat menyebabkan kebocoran kas jutaan rupiah. Di sistem ini, seluruh nilai uang direpresentasikan sebagai bilangan bulat (*integer*) dalam denominasi unit terkecil (Rupiah atau Cents), dan hanya diformat ke desimal saat dirender di antarmuka.",
        body_style
    ))

    # =========================================================================
    # BAB 2: KONKURENSI TINGGI & ELIMINASI DEADLOCK DETERMINISTIK
    # =========================================================================
    story.append(Paragraph("BAB 2: TEORI KONKURENSI DATABASE &amp; ELIMINASI DEADLOCK DETERMINISTIK", h1_style))

    story.append(Paragraph("2.1 PostgreSQL MVCC &amp; Row-Level Locking (SELECT FOR UPDATE)", h2_style))
    story.append(Paragraph(
        "PostgreSQL menggunakan MVCC (*Multi-Version Concurrency Control*). Secara default (isolasi *Read Committed*), pembacaan data tidak pernah memblokir penulisan. "
        "Namun, untuk transfer finansial, operasi *Read-Modify-Write* rentan mengalami anomali **Lost Update** jika dua transfer bersaing membaca saldo akun yang sama secara bersamaan. "
        "Oleh karena itu, gateway menggunakan **Pessimistic Row-Level Locking** via <code>SELECT ... FOR UPDATE</code>. Kunci eksklusif (*X-Lock*) diambil pada baris akun yang terlibat hingga transaksi ACID selesai di-commit.",
        body_style
    ))

    story.append(Paragraph("2.2 Empat Kondisi Coffman Deadlock &amp; Solusi Pengurutan Deterministik", h2_style))
    story.append(Paragraph(
        "Menurut Edward G. Coffman Jr. (1971), sebuah sistem konkurensi hanya dapat mengalami **deadlock** jika keempat kondisi berikut terpenuhi sekaligus:<br/>"
        "1. <b>Mutual Exclusion</b>: Baris database hanya dapat di-lock oleh satu transaksi pada satu waktu.<br/>"
        "2. <b>Hold and Wait</b>: Transaksi memegang lock akun pertama sambil menunggu lock akun kedua.<br/>"
        "3. <b>No Preemption</b>: Database tidak bisa merebut paksa lock yang sedang dipegang oleh transaksi lain sebelum commit.<br/>"
        "4. <b>Circular Wait</b>: Transaksi T1 menunggu resource yang dipegang T2, sementara T2 menunggu resource yang dipegang T1 (siklus lingkaran).",
        body_style
    ))

    deadlock_box = (
        "<b>Pembuktian Matematis Eliminasi Deadlock (Breaking Circular Wait):</b><br/>"
        "Dari keempat kondisi di atas, kondisi 1, 2, dan 3 adalah keharusan mutlak dalam transaksi ACID perbankan. "
        "Maka satu-satunya cara mengeliminasi deadlock adalah <b>memecahkan kondisi ke-4 (Circular Wait)</b>.<br/>"
        "Gateway membandingkan UUID kedua akun secara leksikografis sebelum query dijalankan:<br/>"
        "$$\\text{Order}(\\text{ID}_A, \\text{ID}_B) = \\min(\\text{ID}_A, \\text{ID}_B) \\rightarrow \\max(\\text{ID}_A, \\text{ID}_B)$$<br/>"
        "Karena seluruh transaksi di seluruh goroutine selalu meminta lock dalam urutan global yang sama, graf ketergantungan resource (*Resource Allocation Graph*) "
        "dijamin berupa <b>Directed Acyclic Graph (DAG)</b> yang secara matematis mustahil memiliki siklus (*cycle*). "
        "Deadlock 100% tereliminasi dengan kompleksitas waktu $O(1)$ tanpa beban memory overhead!"
    )
    deadlock_table = Table([[Paragraph(deadlock_box, callout_style)]], colWidths=[520])
    deadlock_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, -1), colors.HexColor("#f0fdf4")),
        ('BOX', (0, 0), (-1, -1), 1, colors.HexColor("#86efac")),
        ('LEFTPADDING', (0, 0), (-1, -1), 10),
        ('RIGHTPADDING', (0, 0), (-1, -1), 10),
        ('TOPPADDING', (0, 0), (-1, -1), 8),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 8),
    ]))
    story.append(deadlock_table)
    story.append(Spacer(1, 10))

    # =========================================================================
    # BAB 3: SISTEM TERDISTRIBUSI & DISTRIBUTED IDEMPOTENCY
    # =========================================================================
    story.append(Paragraph("BAB 3: SISTEM TERDISTRIBUSI &amp; DISTRIBUTED IDEMPOTENCY", h1_style))

    story.append(Paragraph("3.1 The Fallacies of Distributed Computing &amp; At-Least-Once Delivery", h2_style))
    story.append(Paragraph(
        "Peter Deutsch merumuskan hukum mendasar sistem terdistribusi: *Koneksi jaringan tidak pernah 100% andal*. "
        "Ketika aplikasi mobile nasabah menembak request transfer, ada 3 kemungkinan saat terjadi network timeout:<br/>"
        "a. Request tidak pernah sampai ke gateway.<br/>"
        "b. Request sampai dan berhasil diproses, namun respons balik terputus di jaringan seluler.<br/>"
        "c. Request masih antre di intermediate proxy.<br/>"
        "Karena smartphone client pasti melakukan **retry otomatis** (*At-Least-Once Delivery*), backend wajib bersifat **Idempotent**.",
        body_style
    ))

    story.append(Paragraph("3.2 3-State Machine Idempotensi Berbasis Redis Atomic SETNX", h2_style))
    story.append(Paragraph(
        "Idempotensi tidak boleh hanya berupa query pengecekan sederhana (misal: <code>SELECT id FROM transfers WHERE idemp_key = ?</code>) "
        "karena query SELECT rentan *Time-Of-Check to Time-Of-Use (TOCTOU)* race condition. Jika 5 request masuk bersamaan dalam milidetik yang sama, "
        "kelima request akan melihat data belum ada di database dan kelimanya akan mendebit saldo nasabah (bencana race condition).",
        body_style
    ))

    state_rows = [
        [
            Paragraph("<b>State</b>", ParagraphStyle('Th1', parent=body_bold, textColor=colors.white)),
            Paragraph("<b>Mekanisme Eksekusi di Redis &amp; DB</b>", ParagraphStyle('Th2', parent=body_bold, textColor=colors.white)),
            Paragraph("<b>Respon HTTP ke Client</b>", ParagraphStyle('Th3', parent=body_bold, textColor=colors.white))
        ],
        [
            Paragraph("<b>ACQUIRED</b>", body_style),
            Paragraph("Eksekusi atomik <code>SET key IN_PROGRESS NX EX 120</code>. Kunci berhasil diambil pertama kali oleh goroutine pemenang.", body_style),
            Paragraph("Request diizinkan melanjutkan eksekusi ACID transfer ke database.", body_style)
        ],
        [
            Paragraph("<b>IN_PROGRESS</b>", body_style),
            Paragraph("Request duplikat masuk saat transaksi pertama masih berjalan di database. Operasi <code>SETNX</code> gagal karena kunci sudah ada.", body_style),
            Paragraph("<b>HTTP 409 Conflict</b> (Pesan: Transaksi sedang diproses, tolak duplikasi).", body_style)
        ],
        [
            Paragraph("<b>COMPLETED</b>", body_style),
            Paragraph("Transaksi pertama sukses dikomit ke database. Payload respons disimpan di Redis: <code>SET key json_data EX 86400</code>.", body_style),
            Paragraph("<b>HTTP 200 OK (Cached)</b>. Mengembalikan respons instan tanpa mutasi saldo berulang.", body_style)
        ]
    ]
    state_table = Table(state_rows, colWidths=[100, 260, 160])
    state_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, 0), colors.HexColor("#0f172a")),
        ('ALIGN', (0, 0), (-1, -1), 'LEFT'),
        ('VALIGN', (0, 0), (-1, -1), 'TOP'),
        ('GRID', (0, 0), (-1, -1), 0.5, colors.HexColor("#cbd5e1")),
        ('ROWBACKGROUNDS', (0, 1), (-1, -1), [colors.white, colors.HexColor("#f8fafc")]),
        ('TOPPADDING', (0, 0), (-1, -1), 5),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 5),
        ('LEFTPADDING', (0, 0), (-1, -1), 6),
        ('RIGHTPADDING', (0, 0), (-1, -1), 6),
    ]))
    story.append(state_table)
    story.append(Spacer(1, 10))

    # =========================================================================
    # BAB 4: CYBERSECURITY & KRIPTOGRAFI APLIKASI FINANSIAL
    # =========================================================================
    story.append(Paragraph("BAB 4: CYBERSECURITY &amp; KRIPTOGRAFI APLIKASI FINANSIAL", h1_style))

    story.append(Paragraph("4.1 Mengapa AES-256-GCM (AEAD) dan Bukan AES-CBC?", h2_style))
    story.append(Paragraph(
        "Banyak developer mengira enkripsi simetris apa pun sudah cukup aman. Mengapa kita memilih mode **Galois/Counter Mode (GCM)**? "
        "Dalam mode CBC (*Cipher Block Chaining*), algoritma hanya memberikan kerahasiaan (*confidentiality*), bukan integritas (*authenticity*). "
        "CBC rentan terhadap serangan **Padding Oracle Attack** dan modifikasi ciphertext (*bit-flipping attack*). "
        "AES-256-GCM adalah cipher **AEAD (Authenticated Encryption with Associated Data)**. Selain menghasilkan ciphertext, GCM menghasilkan "
        "**16-byte Authentication Tag**. Jika satu bit ciphertext diubah oleh hacker di database, proses dekripsi seketika gagal melempar error otentikasi tanpa membocorkan plaintext.",
        body_style
    ))

    story.append(Paragraph("4.2 Constant-Time Comparison (hmac.Equal) vs Timing Attacks", h2_style))
    story.append(Paragraph(
        "Fungsi perbandingan string biasa (misal: <code>str1 == str2</code>) di sebagian besar bahasa pemrograman menggunakan optimasi *early-exit*: "
        "fungsi langsung berhenti pada karakter pertama yang berbeda. Hacker dapat mengukur durasi respons server dalam satuan nanodetik (*Side-Channel Timing Attack*) "
        "untuk menebak tanda tangan HMAC byte demi byte. Gateway kita menggunakan <code>crypto/subtle.ConstantTimeCompare</code> (via <code>hmac.Equal</code>), "
        "yang selalu mengecek seluruh panjang byte dengan durasi konstan $O(N)$ tanpa peduli di mana letak perbedaan karakternya.",
        body_style
    ))

    story.append(Paragraph("4.3 Rantai Hash Audit Tamper-Evident (Cryptographic Proof)", h2_style))
    story.append(Paragraph(
        "Bagaimana membuktikan bahwa data ledger tidak dimanipulasi oleh administrator database atau disusupi ransomware? "
        "Setiap baris di tabel <code>audit_logs</code> membentuk rantai hash SHA-256 kriptografis:<br/>"
        "$$H_i = \\text{SHA-256}(H_{i-1} \\parallel \\text{Action}_i \\parallel \\text{Payload}_i \\parallel \\text{Timestamp}_i)$$<br/>"
        "Jika ada yang mengubah data pada baris $k$, maka saat rekonsiliasi audit dijalankan, $H_k$ tidak lagi cocok dengan $H_{k+1}$, "
        "sehingga sistem langsung mengetahui secara presisi di baris mana pelanggaran integritas terjadi (mirip mekanisme block header pada blockchain).",
        body_style
    ))
    story.append(Spacer(1, 10))

    # =========================================================================
    # BAB 5: REKONSILIASI PERBANKAN 3-ARAH (3-WAY MATCHING ENGINE)
    # =========================================================================
    story.append(Paragraph("BAB 5: TEORI REKONSILIASI PERBANKAN 3-ARAH (3-WAY MATCHING ENGINE)", h1_style))
    story.append(Paragraph(
        "Dalam ekosistem integrasi payment switch dengan perbankan nasional (BI-FAST, RTGS, Switching Mandiri/BCA/BRI), "
        "selisih mutasi (*settlement discrepancy*) adalah hal yang tak terhindarkan setiap pergantian hari (*cut-off time*). "
        "Engine rekonsiliasi kita membandingkan 3 sumber data: <b>Order Database Merchant</b> vs <b>Internal Double-Entry Ledger</b> vs <b>External Bank Settlement Statement (CSV/MT940)</b>.",
        body_style
    ))

    recon_data = [
        [
            Paragraph("<b>Klasifikasi Anomali</b>", ParagraphStyle('Th1', parent=body_bold, textColor=colors.white)),
            Paragraph("<b>Penyebab di Dunia Nyata</b>", ParagraphStyle('Th2', parent=body_bold, textColor=colors.white)),
            Paragraph("<b>Resolusi Operasional Finansial</b>", ParagraphStyle('Th3', parent=body_bold, textColor=colors.white))
        ],
        [
            Paragraph("<b>100% MATCHED</b>", body_style),
            Paragraph("Reference ID cocok sempurna dan nominal identik di Bank dan Gateway.", body_style),
            Paragraph("Otomatis di-clearing ke rekening merchant tanpa campur tangan manual.", body_style)
        ],
        [
            Paragraph("<b>DISCREPANCY_AMOUNT</b>", body_style),
            Paragraph("Transaksi cocok di kedua pihak, tapi terdapat selisih nominal (misal potongan MDR, fee admin bank liar, atau selisih kurs valas).", body_style),
            Paragraph("Sistem mencatat selisih varians (+/- Rp X) ke akun penampungan (*Discrepancy Suspense GL*) untuk diselesaikan tim akuntansi.", body_style)
        ],
        [
            Paragraph("<b>MISSING_IN_GATEWAY</b>", body_style),
            Paragraph("Uang terpotong di rekening bank eksternal, tetapi gateway tidak memiliki catatan transaksi (akibat network drop saat callback).", body_style),
            Paragraph("Sistem mengisolasi ID bank tersebut agar tim ops dapat melakukan klaim balik atau membuat mutasi susulan manual.", body_style)
        ]
    ]
    recon_table = Table(recon_data, colWidths=[120, 220, 180])
    recon_table.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, 0), colors.HexColor("#0f172a")),
        ('ALIGN', (0, 0), (-1, -1), 'LEFT'),
        ('VALIGN', (0, 0), (-1, -1), 'TOP'),
        ('GRID', (0, 0), (-1, -1), 0.5, colors.HexColor("#cbd5e1")),
        ('ROWBACKGROUNDS', (0, 1), (-1, -1), [colors.white, colors.HexColor("#f8fafc")]),
        ('TOPPADDING', (0, 0), (-1, -1), 5),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 5),
        ('LEFTPADDING', (0, 0), (-1, -1), 6),
        ('RIGHTPADDING', (0, 0), (-1, -1), 6),
    ]))
    story.append(recon_table)
    story.append(Spacer(1, 10))

    # =========================================================================
    # BAB 6: OBSERVABILITY, SLA LATENSI & PERFORMANCE
    # =========================================================================
    story.append(Paragraph("BAB 6: OBSERVABILITY, DISTRIBUSI LATENSI (P95/P99) &amp; DATABASE POOLING", h1_style))
    story.append(Paragraph(
        "<b>Mengapa Rata-rata (*Arithmetic Mean*) Latensi Adalah Kebohongan dalam SLA Fintech?</b><br/>"
        "Jika 99 transaksi selesai dalam 5ms, namun 1 transaksi tertahan lock database selama 10.000ms (10 detik):<br/>"
        "$$\\text{Mean} = \\frac{99 \\times 5 + 10000}{100} = 104.95\\text{ ms}$$<br/>"
        "Angka rata-rata 104ms tampak wajar, padahal nasabah ke-100 mengalami kegagalan fatal (*timeout*). "
        "Di industri core banking, metrik evaluasi yang diwajibkan adalah **P95** ($95\\%$ transaksi selesai di bawah $X$ ms) dan **P99** ($99\\%$ transaksi selesai di bawah $Y$ ms). "
        "Gateway kita memantau rolling persentil ini secara dinamis melalui antarmuka telemetry.",
        body_style
    ))

    # =========================================================================
    # BAB 7: 10 PERTANYAAN MEMATIKAN TECH LEAD & JAWABAN EKSPLISIT
    # =========================================================================
    story.append(Paragraph("BAB 7: 10 PERTANYAAN PALING MEMATIKAN DARI TECH LEAD FINTECH &amp; CARA MENJAWABNYA", h1_style))

    qas = [
        (
            "Q1: Apa yang terjadi jika database PostgreSQL mati persis di tengah eksekusi transfer?",
            "Jawab: Transaksi kita dibungkus di dalam <code>BEGIN ... COMMIT</code> blok ACID murni. Jika koneksi atau database mati sebelum COMMIT berhasil dijalankan, PostgreSQL Write-Ahead Logging (WAL) akan me-rollback seluruh operasi mutasi secara atomik. Saldo pengirim tidak akan pernah terpotong separuh jalan."
        ),
        (
            "Q2: Kenapa tidak memakai isolasi SERIALIZABLE saja daripada pusing mengurutkan lock UUID di level aplikasi?",
            "Jawab: Isolasi SERIALIZABLE pada PostgreSQL bekerja berbasis SSI (Serializable Snapshot Isolation) yang mendeteksi konflik rw-antidependency. Jika terjadi konflik transaksi bersamaan, PostgreSQL melempar error 40001 (Serialization Failure) dan mewajibkan aplikasi melakukan retry loop. Di bawah beban transaksi tinggi, serializable retry loop memicu thundering herd problem dan membebani CPU database. Pengurutan UUID deterministik di level aplikasi mengeliminasi deadlock sejak awal tanpa perlu retry loop sama sekali."
        ),
        (
            "Q3: Bagaimana jika Redis mati total? Apakah gateway langsung berhenti memproses transfer?",
            "Jawab: Tidak. Repositori idempotensi kita dirancang dengan pattern graceful degradation: jika Redis tidak dapat dihubungi, sistem otomatis beralih ke in-memory sync.Map idempotency engine dengan proteksi mutex di level proses Go, sambil terus mencatat alert di structured logger."
        ),
        (
            "Q4: Mengapa tidak mengirim webhook langsung dari dalam transaksi database transfer?",
            "Jawab: Itu adalah anti-pattern 'Dual-Write'. Melakukan HTTP call ke pihak ketiga di dalam database transaction dapat menahan koneksi database terlalu lama (connection starvation) jika server pihak ketiga lambat. Selain itu, jika transaksi DB gagal setelah webhook terkirim, kita tidak bisa membatalkan webhook tersebut. Solusinya adalah Transactional Outbox Pattern: event disimpan di tabel outbox dalam transaksi yang sama, lalu di-dispatch oleh worker asinkron via SELECT FOR UPDATE SKIP LOCKED."
        ),
        (
            "Q5: Bagaimana jika server Anda dan server merchant memiliki selisih jam (Clock Drift)?",
            "Jawab: Kami menerapkan toleransi jendela waktu ±5 menit (300 detik) pada validasi X-Timestamp. Ini memberikan ruang yang cukup untuk clock skew wajar antara NTP server merchant dan gateway, namun cukup ketat untuk menolak Replay Attack dari rekaman trafik lampau."
        )
    ]

    for q, a in qas:
        story.append(Paragraph(f"<b>{q}</b>", h2_style))
        story.append(Paragraph(a, body_style))
        story.append(Spacer(1, 3))

    # Build Document
    doc.build(story, canvasmaker=NumberedCanvas)
    print(f"[SUCCESS] Fundamental Engineering Manual built: {filename}")

if __name__ == '__main__':
    out_file = "Secure_Payment_Gateway_Fundamental_Engineering_Manual.pdf"
    if len(sys.argv) > 1:
        out_file = sys.argv[1]
    build_pdf(out_file)

