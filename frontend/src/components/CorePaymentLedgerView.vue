<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { formatIDR } from '../lib/utils'
import {
  Landmark,
  FileSpreadsheet,
  Plus,
  RefreshCw,
  Lock,
  Unlock,
  Zap,
  ShieldCheck,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Hash,
} from '@lucide/vue'

interface Account {
  id: string
  account_number: string
  holder_name: string
  ciphertext_preview?: string
  decrypted_identity?: string
  balance: number
  currency: string
  version: number
}

interface LedgerEntry {
  id: string
  transaction_id: string
  account_id: string
  entry_type: 'DEBIT' | 'CREDIT'
  amount: number
  balance_after: number
  created_at: string
}

interface AuditLogEntry {
  id: number
  entity_type: string
  entity_id: string
  action: string
  previous_hash: string
  current_hash: string
  created_at: string
}

interface ReconItem {
  id: string
  reference_id: string
  bank_amount: number
  system_amount?: number | null
  status: 'MATCHED' | 'AMOUNT_MISMATCH' | 'MISSING_IN_SYSTEM'
  notes: string
}

const accounts = ref<Account[]>([])
const ledger = ref<LedgerEntry[]>([])
const auditLogs = ref<AuditLogEntry[]>([])
const decryptedMap = ref<Record<string, string>>({})
const decryptingId = ref<string | null>(null)
const activeLedgerTab = ref<'entries' | 'audit'>('entries')

const accNum = ref('')
const holder = ref('')
const identity = ref('')
const balance = ref(2500000)

const simAmount = ref(275000)
const isSimulatingTransfer = ref(false)

const PRESET_MATCHED_CSV = `reference_id,amount,currency,settled_at
REF-BCA-2026-1001,324000,IDR,2026-10-31T10:00:00Z
REF-BCA-2026-1002,394000,IDR,2026-10-31T10:05:00Z
REF-BCA-2026-1003,249000,IDR,2026-10-31T10:10:00Z`

const PRESET_ANOMALY_CSV = `reference_id,amount,currency,settled_at
REF-BCA-2026-1001,324000,IDR,2026-10-31T10:00:00Z
REF-BCA-2026-1002,389000,IDR,2026-10-31T10:05:00Z
REF-BCA-2026-9999,550000,IDR,2026-10-31T10:15:00Z`

const activePreset = ref<'matched' | 'anomaly'>('matched')
const csvText = ref(PRESET_MATCHED_CSV)
const reconSummary = ref<any | null>(null)
const reconItems = ref<ReconItem[]>([])
const isReconciling = ref(false)

const accountNameById = computed(() => {
  const map: Record<string, { number: string; name: string }> = {}
  for (const a of accounts.value) {
    map[a.id] = { number: a.account_number, name: a.holder_name }
  }
  return map
})

async function loadCoreData() {
  try {
    const [aRes, lRes, audRes] = await Promise.all([
      fetch('/api/v1/accounts'),
      fetch('/api/v1/ledger'),
      fetch('/api/v1/audit-logs'),
    ])
    const aJson = await aRes.json()
    const lJson = await lRes.json()
    const audJson = await audRes.json()
    accounts.value = aJson.data || []
    ledger.value = lJson.data || []
    auditLogs.value = audJson.data || []
  } catch {
    // Non-blocking
  }
}

async function inspectDecryptedKYC(accountId: string) {
  if (decryptedMap.value[accountId]) {
    delete decryptedMap.value[accountId]
    return
  }
  decryptingId.value = accountId
  try {
    const res = await fetch(`/api/v1/accounts/${accountId}`)
    const json = await res.json()
    if (!res.ok) throw new Error(json.message || 'Gagal mendekripsi identitas')
    const plain = json.data?.decrypted_identity || 'Terdekripsi via AES-256-GCM'
    decryptedMap.value[accountId] = plain
    toast.success('Identitas KYC berhasil didekripsi di memori (AES-256-GCM)')
  } catch (e: any) {
    toast.error('Gagal mendekripsi KYC', { description: e.message })
  } finally {
    decryptingId.value = null
  }
}

async function createAccount() {
  if (!accNum.value.trim() || !holder.value.trim() || !identity.value.trim()) {
    toast.warning('Lengkapi nomor akun, nama pemilik, dan identitas KYC')
    return
  }
  try {
    const res = await fetch('/api/v1/accounts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        account_number: accNum.value.trim(),
        holder_name: holder.value.trim(),
        identity_data: identity.value.trim(),
        initial_balance: Number(balance.value) || 0,
        currency: 'IDR',
      }),
    })
    const json = await res.json()
    if (!res.ok) throw new Error(json.error?.details || json.message || 'Gagal membuat akun')
    toast.success(`Akun ${json.data.account_number} berhasil dienkripsi (AES-256-GCM)`)
    accNum.value = ''
    holder.value = ''
    identity.value = ''
    await loadCoreData()
  } catch (e: any) {
    toast.error('Gagal membuat akun', { description: e.message })
  }
}

async function simulateDoubleEntryTransfer() {
  isSimulatingTransfer.value = true
  try {
    const res = await fetch('/api/v1/simulator/ledger-transfer', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        amount: Number(simAmount.value) || 275000,
        currency: 'IDR',
      }),
    })
    const json = await res.json()
    if (!res.ok) throw new Error(json.error?.details || json.message || 'Simulasi transfer gagal')

    toast.success(`Double-Entry Settlement (${json.data.reference_id}) Berhasil!`, {
      description: `DEBIT Escrow & CREDIT Merchant sebesar ${formatIDR(json.data.amount)} tercatat atomik beserta Hash-Chain Audit Log.`,
    })
    await loadCoreData()
  } catch (e: any) {
    toast.error('Gagal menjalankan simulasi transfer', { description: e.message })
  } finally {
    isSimulatingTransfer.value = false
  }
}

function selectCsvPreset(mode: 'matched' | 'anomaly') {
  activePreset.value = mode
  csvText.value = mode === 'matched' ? PRESET_MATCHED_CSV : PRESET_ANOMALY_CSV
}

async function runReconciliation() {
  isReconciling.value = true
  try {
    const blob = new Blob([csvText.value.trim()], { type: 'text/csv' })
    const form = new FormData()
    form.append('file', blob, `bca-settlement-${activePreset.value}.csv`)
    const res = await fetch('/api/v1/reconciliation/upload', { method: 'POST', body: form })
    const json = await res.json()
    if (!res.ok) throw new Error(json.error?.details || json.message || 'Rekonsiliasi gagal')
    reconSummary.value = json.data

    // Fetch itemized reconciliation detail
    if (json.data?.id) {
      const detailRes = await fetch(`/api/v1/reconciliation/batches/${json.data.id}`)
      if (detailRes.ok) {
        const detailJson = await detailRes.json()
        reconItems.value = detailJson.data?.items || []
      }
    }

    toast.success('Rekonsiliasi CSV 3-Way selesai diproses', {
      description: `Matched: ${json.data.matched_records} | Discrepancy: ${json.data.mismatched_records} | Missing: ${json.data.missing_in_system}`,
    })
  } catch (e: any) {
    toast.error('Rekonsiliasi gagal', { description: e.message })
  } finally {
    isReconciling.value = false
  }
}

onMounted(async () => {
  await loadCoreData()
  await runReconciliation()
})
</script>

<template>
  <div class="space-y-5">
    <!-- TOP BANNER: ATOMIC DOUBLE-ENTRY SETTLEMENT SIMULATOR -->
    <section class="rounded-2xl border border-indigo-500/30 bg-gradient-to-r from-indigo-950/40 via-slate-900/40 to-emerald-950/30 p-4 sm:p-5 shadow-xs">
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div class="space-y-1">
          <div class="flex items-center gap-2 flex-wrap">
            <Badge variant="indigo">ZERO-TRUST FINTECH CORE</Badge>
            <Badge variant="emerald" class="font-mono">Deterministic Lock Ordering</Badge>
            <Badge variant="outline" class="font-mono">AES-256-GCM + SHA-256 Hash Chain</Badge>
          </div>
          <h2 id="core-ledger-heading" tabindex="-1" class="text-base sm:text-lg font-extrabold text-foreground focus:outline-none">
            Double-Entry Payment Ledger, KYC Encryption Vault &amp; 3-Way Bank Reconciliation
          </h2>
          <p class="text-xs text-muted-foreground max-w-3xl">
            Setiap pencairan dana mengeksekusi penguncian baris deterministik (<code class="text-foreground">SELECT ... FOR UPDATE</code> berurutan berdasarkan UUID untuk mencegah deadlock), mencatat pasangan jurnal <code class="text-foreground">DEBIT</code> &amp; <code class="text-foreground">CREDIT</code> yang seimbang, serta merangkai hash kriptografis <code class="text-foreground">SHA-256</code>.
          </p>
        </div>

        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 shrink-0 bg-card/90 p-3 rounded-xl border border-border">
          <div>
            <label for="sim-transfer-amount" class="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
              Nominal Settlement (IDR)
            </label>
            <input
              id="sim-transfer-amount"
              v-model.number="simAmount"
              type="number"
              step="25000"
              min="10000"
              class="mt-0.5 w-full sm:w-36 rounded-lg bg-elevated border border-border px-2.5 py-1.5 text-xs font-mono font-bold text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <Button
            variant="emerald"
            size="sm"
            class="sm:mt-4 font-semibold"
            :disabled="isSimulatingTransfer"
            @click="simulateDoubleEntryTransfer"
          >
            <Zap class="w-3.5 h-3.5" />
            <span>{{ isSimulatingTransfer ? 'Memproses...' : 'Simulasi Settlement (Double-Entry)' }}</span>
          </Button>
        </div>
      </div>
    </section>

    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <!-- LEFT 5 COLS: AES-256-GCM MERCHANT VAULT -->
      <section class="lg:col-span-5 rounded-2xl bg-card border border-border p-4 sm:p-5 space-y-4 shadow-xs">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Badge variant="indigo">
              <Landmark class="w-3 h-3" />
              <span>AES-256-GCM VAULT</span>
            </Badge>
            <h3 class="text-sm font-bold text-foreground">
              Merchant &amp; Escrow Accounts
            </h3>
          </div>
          <Button variant="ghost" size="sm" @click="loadCoreData">
            <RefreshCw class="w-3.5 h-3.5" />
          </Button>
        </div>

        <!-- Existing Seeded Accounts with On-Demand AES-256-GCM Decryption -->
        <div class="space-y-2.5 max-h-80 overflow-y-auto custom-scroll pr-1">
          <div
            v-for="a in accounts"
            :key="a.id"
            class="p-3 rounded-xl bg-elevated/70 border border-border space-y-2 text-xs"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <div class="flex items-center gap-1.5">
                  <span class="font-mono font-extrabold text-foreground">{{ a.account_number }}</span>
                  <Badge variant="outline" class="font-mono text-[9px]">v{{ a.version }}</Badge>
                </div>
                <div class="text-[11px] text-secondary-foreground font-medium truncate">
                  {{ a.holder_name }}
                </div>
              </div>
              <div class="text-right font-mono font-extrabold text-tone-emerald tabular-nums shrink-0">
                {{ formatIDR(a.balance) }}
              </div>
            </div>

            <!-- Ciphertext vs Decrypted KYC Box -->
            <div class="pt-2 border-t border-border/70 flex items-center justify-between gap-2">
              <div class="min-w-0 flex-1">
                <div v-if="decryptedMap[a.id]" class="text-[11px] font-mono text-tone-emerald font-semibold break-all">
                  🔓 Plaintext KYC: {{ decryptedMap[a.id] }}
                </div>
                <div v-else class="text-[10px] font-mono text-muted-foreground truncate" :title="a.ciphertext_preview">
                  🔒 Ciphertext: {{ a.ciphertext_preview || 'AES-256-GCM Encrypted' }}
                </div>
              </div>
              <button
                type="button"
                @click="inspectDecryptedKYC(a.id)"
                class="shrink-0 inline-flex items-center gap-1 px-2 py-1 rounded-lg border border-border bg-card hover:bg-muted text-[10px] font-bold text-foreground transition cursor-pointer"
              >
                <Unlock v-if="decryptedMap[a.id]" class="w-3 h-3 text-tone-amber" />
                <Lock v-else class="w-3 h-3 text-tone-indigo" />
                <span>{{ decryptingId === a.id ? '...' : decryptedMap[a.id] ? 'Sembunyikan' : 'Dekripsi KYC' }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Register New Encrypted Account Form -->
        <div class="pt-3 border-t border-border space-y-2.5">
          <div class="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
            + Registrasi Akun Baru (Enkripsi Otomatis AES-256-GCM)
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
            <div>
              <input
                v-model="accNum"
                placeholder="No. Akun (ACC-MERCH-8803)"
                class="w-full rounded-xl bg-elevated border border-border px-3 py-1.5 text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <input
                v-model="holder"
                placeholder="Nama Pemilik / PT"
                class="w-full rounded-xl bg-elevated border border-border px-3 py-1.5 text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <input
                v-model="identity"
                placeholder="NPWP / NIK (Dienkripsi AES-GCM)"
                class="w-full rounded-xl bg-elevated border border-border px-3 py-1.5 text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <input
                v-model.number="balance"
                type="number"
                placeholder="Saldo Awal (IDR)"
                class="w-full rounded-xl bg-elevated border border-border px-3 py-1.5 text-foreground font-mono focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
          </div>

          <Button variant="indigo" size="sm" class="w-full" @click="createAccount">
            <Plus class="w-3.5 h-3.5" />
            <span>Enkripsi Identitas &amp; Buat Akun</span>
          </Button>
        </div>
      </section>

      <!-- RIGHT 7 COLS: DOUBLE-ENTRY LEDGER & SHA-256 AUDIT LOG -->
      <section class="lg:col-span-7 rounded-2xl bg-card border border-border p-4 sm:p-5 space-y-4 shadow-xs flex flex-col justify-between">
        <div class="space-y-3">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div>
              <div class="flex items-center gap-2">
                <Badge variant="emerald">IMMUTABLE JOURNAL</Badge>
                <h3 class="text-sm font-bold text-foreground">
                  Double-Entry Ledger &amp; Tamper-Evident Audit Chain
                </h3>
              </div>
              <p class="text-[11px] text-muted-foreground mt-0.5">
                Setiap transaksi menghasilkan 2 baris jurnal berpasangan (<code class="text-foreground">DEBIT</code> = <code class="text-foreground">CREDIT</code>) dan hash SHA-256 berantai.
              </p>
            </div>

            <div class="inline-flex rounded-xl bg-elevated p-1 border border-border text-xs shrink-0">
              <button
                type="button"
                @click="activeLedgerTab = 'entries'"
                :class="activeLedgerTab === 'entries' ? 'bg-card text-foreground font-bold shadow-2xs' : 'text-muted-foreground'"
                class="px-2.5 py-1 rounded-lg transition cursor-pointer"
              >
                Jurnal DEBIT/CREDIT ({{ ledger.length }})
              </button>
              <button
                type="button"
                @click="activeLedgerTab = 'audit'"
                :class="activeLedgerTab === 'audit' ? 'bg-card text-foreground font-bold shadow-2xs' : 'text-muted-foreground'"
                class="px-2.5 py-1 rounded-lg transition cursor-pointer inline-flex items-center gap-1"
              >
                <Hash class="w-3 h-3 text-tone-indigo" />
                <span>SHA-256 Chain ({{ auditLogs.length }})</span>
              </button>
            </div>
          </div>

          <!-- TAB 1: DOUBLE-ENTRY LEDGER TABLE -->
          <div v-if="activeLedgerTab === 'entries'" class="max-h-96 overflow-y-auto custom-scroll border border-border rounded-xl">
            <table class="w-full text-xs text-left">
              <thead class="bg-elevated text-muted-foreground uppercase text-[10px] border-b border-border sticky top-0">
                <tr>
                  <th scope="col" class="px-3 py-2.5">Tipe Jurnal</th>
                  <th scope="col" class="px-3 py-2.5">Akun Ledger</th>
                  <th scope="col" class="px-3 py-2.5 text-right">Nominal Mutasi</th>
                  <th scope="col" class="px-3 py-2.5 text-right">Saldo Akhir</th>
                  <th scope="col" class="px-3 py-2.5 text-right">Tx ID</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border font-mono">
                <tr v-if="ledger.length === 0">
                  <td colspan="5" class="px-3 py-8 text-center text-muted-foreground font-sans">
                    Belum ada entri jurnal. Klik tombol "Simulasi Settlement" di atas.
                  </td>
                </tr>
                <tr v-for="e in ledger" :key="e.id" class="hover:bg-elevated/40">
                  <td class="px-3 py-2">
                    <Badge :variant="e.entry_type === 'CREDIT' ? 'emerald' : 'amber'">
                      {{ e.entry_type }}
                    </Badge>
                  </td>
                  <td class="px-3 py-2">
                    <div class="font-bold text-foreground text-[11px]">
                      {{ accountNameById[e.account_id]?.number || e.account_id.slice(0, 10) }}
                    </div>
                    <div class="font-sans text-[10px] text-muted-foreground truncate max-w-[180px]">
                      {{ accountNameById[e.account_id]?.name || 'Ledger Account' }}
                    </div>
                  </td>
                  <td
                    class="px-3 py-2 text-right font-bold tabular-nums"
                    :class="e.entry_type === 'CREDIT' ? 'text-tone-emerald' : 'text-tone-amber'"
                  >
                    {{ e.entry_type === 'CREDIT' ? '+' : '-' }}{{ formatIDR(e.amount) }}
                  </td>
                  <td class="px-3 py-2 text-right text-secondary-foreground tabular-nums">
                    {{ formatIDR(e.balance_after) }}
                  </td>
                  <td class="px-3 py-2 text-right text-[10px] text-muted-foreground">
                    {{ e.transaction_id.slice(0, 8) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- TAB 2: SHA-256 TAMPER-EVIDENT AUDIT HASH CHAIN -->
          <div v-else class="max-h-96 overflow-y-auto custom-scroll space-y-2 pr-1">
            <div
              v-for="log in auditLogs"
              :key="log.id"
              class="p-3 rounded-xl bg-elevated/60 border border-border text-xs space-y-1.5 font-mono"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <Badge variant="indigo">Block #{{ log.id }}</Badge>
                  <span class="font-bold text-foreground">{{ log.entity_type }} // {{ log.action }}</span>
                </div>
                <span class="text-[10px] text-muted-foreground">{{ new Date(log.created_at).toLocaleTimeString('id-ID') }}</span>
              </div>
              <div class="text-[10px] text-muted-foreground truncate">
                Prev Hash: <span class="text-secondary-foreground">{{ log.previous_hash }}</span>
              </div>
              <div class="text-[10px] text-tone-emerald truncate font-semibold">
                Curr Hash: {{ log.current_hash }}
              </div>
            </div>
          </div>
        </div>

        <div class="pt-2 border-t border-border flex items-center justify-between text-[11px] text-muted-foreground">
          <span class="inline-flex items-center gap-1.5">
            <ShieldCheck class="w-3.5 h-3.5 text-tone-emerald" />
            <span>Invariant Check: Total <code class="text-foreground">SUM(DEBIT) == SUM(CREDIT)</code> terjamin atomik</span>
          </span>
          <span class="font-mono text-[10px]">ISO-Level: READ COMMITTED + Row Lock</span>
        </div>
      </section>
    </div>

    <!-- BOTTOM FULL-WIDTH: 3-WAY BANK STATEMENT CSV RECONCILIATION ENGINE -->
    <section class="rounded-2xl bg-card border border-border p-4 sm:p-5 space-y-4 shadow-xs">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <div class="flex items-center gap-2">
            <Badge variant="emerald">
              <FileSpreadsheet class="w-3 h-3" />
              <span>3-WAY RECONCILIATION</span>
            </Badge>
            <h3 class="text-sm sm:text-base font-bold text-foreground">
              Automated Bank Statement CSV Settlement Matcher
            </h3>
          </div>
          <p class="text-xs text-muted-foreground mt-0.5">
            Mencocokkan mutasi rekening koran bank (CSV) terhadap tabel <code class="text-foreground">transactions</code> internal dan mendeteksi selisih biaya MDR (<code class="text-foreground">AMOUNT_MISMATCH</code>) maupun transaksi siluman (<code class="text-foreground">MISSING_IN_SYSTEM</code>).
          </p>
        </div>

        <!-- 2 Realistic Presets -->
        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            @click="selectCsvPreset('matched'); runReconciliation()"
            :class="
              activePreset === 'matched'
                ? 'bg-emerald-600 text-white border-emerald-600'
                : 'bg-elevated text-foreground border-border hover:bg-muted'
            "
            class="px-3 py-1.5 rounded-xl border text-xs font-semibold transition cursor-pointer inline-flex items-center gap-1.5"
          >
            <CheckCircle2 class="w-3.5 h-3.5" />
            <span>Preset 1: 100% Matched (3/3)</span>
          </button>
          <button
            type="button"
            @click="selectCsvPreset('anomaly'); runReconciliation()"
            :class="
              activePreset === 'anomaly'
                ? 'bg-amber-600 text-white border-amber-600'
                : 'bg-elevated text-foreground border-border hover:bg-muted'
            "
            class="px-3 py-1.5 rounded-xl border text-xs font-semibold transition cursor-pointer inline-flex items-center gap-1.5"
          >
            <AlertTriangle class="w-3.5 h-3.5" />
            <span>Preset 2: Simulasi Selisih MDR &amp; Ghost Tx</span>
          </button>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-12 gap-4 items-start">
        <!-- Left 5 Cols: CSV Input -->
        <div class="lg:col-span-5 space-y-2.5">
          <label for="recon-csv" class="block text-[11px] font-semibold text-muted-foreground">
            Payload Mutasi CSV Bank (BCA / Mandiri Corporate Statement):
          </label>
          <textarea
            id="recon-csv"
            v-model="csvText"
            rows="5"
            class="w-full rounded-xl bg-elevated border border-border p-3 font-mono text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          ></textarea>
          <Button variant="emerald" size="sm" class="w-full" :disabled="isReconciling" @click="runReconciliation">
            <FileSpreadsheet class="w-3.5 h-3.5" />
            <span>{{ isReconciling ? 'Memproses Rekonsiliasi...' : 'Jalankan 3-Way Reconciliation Engine' }}</span>
          </Button>
        </div>

        <!-- Right 7 Cols: Reconciliation Summary & Itemized Audit Table -->
        <div class="lg:col-span-7 space-y-3">
          <div v-if="reconSummary" class="grid grid-cols-3 gap-2.5 font-mono text-xs">
            <div class="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/30">
              <div class="text-[10px] text-muted-foreground uppercase font-sans font-bold">Matched (Valid)</div>
              <div class="text-xl font-extrabold text-tone-emerald mt-0.5">
                {{ reconSummary.matched_records }}
              </div>
            </div>
            <div class="p-3 rounded-xl bg-amber-500/10 border border-amber-500/30">
              <div class="text-[10px] text-muted-foreground uppercase font-sans font-bold">Amount Mismatch</div>
              <div class="text-xl font-extrabold text-tone-amber mt-0.5">
                {{ reconSummary.mismatched_records }}
              </div>
            </div>
            <div class="p-3 rounded-xl bg-rose-500/10 border border-rose-500/30">
              <div class="text-[10px] text-muted-foreground uppercase font-sans font-bold">Missing in Ledger</div>
              <div class="text-xl font-extrabold text-tone-rose mt-0.5">
                {{ reconSummary.missing_in_system }}
              </div>
            </div>
          </div>

          <div v-if="reconItems.length > 0" class="border border-border rounded-xl overflow-hidden">
            <table class="w-full text-xs text-left">
              <thead class="bg-elevated text-muted-foreground uppercase text-[10px] border-b border-border">
                <tr>
                  <th class="px-3 py-2">Reference ID</th>
                  <th class="px-3 py-2 text-right">Mutasi Bank</th>
                  <th class="px-3 py-2 text-right">Ledger Internal</th>
                  <th class="px-3 py-2">Status &amp; Temuan Audit</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr v-for="item in reconItems" :key="item.id" class="hover:bg-elevated/40">
                  <td class="px-3 py-2 font-mono font-bold text-foreground">
                    {{ item.reference_id }}
                  </td>
                  <td class="px-3 py-2 text-right font-mono tabular-nums text-foreground">
                    {{ formatIDR(item.bank_amount) }}
                  </td>
                  <td class="px-3 py-2 text-right font-mono tabular-nums text-secondary-foreground">
                    {{ item.system_amount != null ? formatIDR(item.system_amount) : 'TIDAK ADA' }}
                  </td>
                  <td class="px-3 py-2">
                    <div class="flex items-center gap-1.5">
                      <CheckCircle2 v-if="item.status === 'MATCHED'" class="w-3.5 h-3.5 text-tone-emerald shrink-0" />
                      <AlertTriangle v-else-if="item.status === 'AMOUNT_MISMATCH'" class="w-3.5 h-3.5 text-tone-amber shrink-0" />
                      <XCircle v-else class="w-3.5 h-3.5 text-tone-rose shrink-0" />
                      <Badge
                        :variant="
                          item.status === 'MATCHED'
                            ? 'emerald'
                            : item.status === 'AMOUNT_MISMATCH'
                              ? 'amber'
                              : 'rose'
                        "
                      >
                        {{ item.status }}
                      </Badge>
                    </div>
                    <div class="text-[10px] text-muted-foreground mt-0.5">
                      {{ item.notes }}
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
