<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { formatIDR } from '../lib/utils'
import { Shield, FileSpreadsheet, Plus } from '@lucide/vue'

const store = useOmsStore()

const holderName = ref('')
const identityNumber = ref('')
const initialBalance = ref(2500000)

const bankName = ref('BCA')
const sampleCSV = ref(
  'reference_id,amount,status,timestamp\nREF-DEMO-1001,150000,SUCCESS,2026-10-31T08:00:00Z\nREF-MISSING-999,275000,SUCCESS,2026-10-31T08:15:00Z',
)
const isCreatingAccount = ref(false)
const isReconciling = ref(false)

async function handleCreateAccount() {
  if (!holderName.value.trim() || !identityNumber.value.trim()) return
  isCreatingAccount.value = true
  try {
    const res = await fetch('/api/v1/accounts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        account_number: `ACC-${Math.floor(100000 + Math.random() * 900000)}`,
        holder_name: holderName.value.trim(),
        identity_number: identityNumber.value.trim(),
        initial_balance: Number(initialBalance.value) || 0,
        currency: 'IDR',
      }),
    })
    const json = await res.json()
    if (res.ok && json.success) {
      store.notify('success', 'Akun Vault Dibuat (AES-256-GCM)', 'NIK dienkripsi sebelum masuk PostgreSQL')
      holderName.value = ''
      identityNumber.value = ''
      await store.fetchCoreGatewayData()
    } else {
      store.notify('error', 'Gagal membuat akun', json.error?.details || json.message)
    }
  } finally {
    isCreatingAccount.value = false
  }
}

async function handleUploadRecon() {
  isReconciling.value = true
  try {
    const blob = new Blob([sampleCSV.value], { type: 'text/csv' })
    const form = new FormData()
    form.append('bank_name', bankName.value)
    form.append('file', blob, 'settlement.csv')

    const res = await fetch('/api/v1/reconciliation/upload', {
      method: 'POST',
      body: form,
    })
    const json = await res.json()
    if (res.ok && json.success) {
      store.notify('success', `Batch Rekonsiliasi ${json.data.batch_number} Selesai`, '3-Way Matching berhasil diproses')
      await store.fetchCoreGatewayData()
    } else {
      store.notify('error', 'Gagal rekonsiliasi CSV', json.error?.details || json.message)
    }
  } finally {
    isReconciling.value = false
  }
}

onMounted(() => {
  store.fetchCoreGatewayData()
})
</script>

<template>
  <div class="space-y-5">
    <!-- Accounts & AES-256-GCM Encryption + Recon CSV -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      <section class="oms-panel rounded-2xl bg-card border border-border shadow-xs p-4 sm:p-5 space-y-4">
        <div>
          <Badge variant="emerald">
            <Shield class="w-3 h-3" />
            <span>AES-256-GCM VAULT</span>
          </Badge>
          <h2 id="core-ledger-heading" tabindex="-1" class="text-sm sm:text-base font-bold text-foreground mt-1.5 focus:outline-none">
            Merchant Settlement Accounts &amp; Field-Level Encryption
          </h2>
          <p class="text-xs text-muted-foreground mt-0.5">
            Identitas sensitif (NIK/NPWP) dienkripsi dengan AEAD AES-256-GCM sebelum disimpan ke PostgreSQL.
          </p>
        </div>

        <form @submit.prevent="handleCreateAccount" class="grid grid-cols-1 sm:grid-cols-3 gap-2.5 text-xs">
          <input
            v-model="holderName"
            type="text"
            placeholder="Nama Pemilik Akun"
            class="rounded-xl bg-elevated border border-border px-3 py-2 text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <input
            v-model="identityNumber"
            type="text"
            placeholder="NIK / NPWP (Dienkripsi)"
            class="rounded-xl bg-elevated border border-border px-3 py-2 text-foreground font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <Button type="submit" variant="emerald" :disabled="isCreatingAccount">
            <Plus class="w-3.5 h-3.5" />
            <span>Buat Akun Vault</span>
          </Button>
        </form>

        <div class="max-h-56 overflow-y-auto custom-scroll divide-y divide-border text-xs">
          <div v-if="store.gatewayAccounts.length === 0" class="py-6 text-center text-muted-foreground">
            Belum ada akun settlement terdaftar. Buat akun baru di atas.
          </div>
          <div
            v-for="acc in store.gatewayAccounts"
            :key="acc.id"
            class="py-2.5 flex items-center justify-between gap-2"
          >
            <div>
              <div class="font-semibold text-foreground">{{ acc.holder_name }}</div>
              <div class="font-mono text-[11px] text-muted-foreground">{{ acc.account_number }} &bull; v{{ acc.version }}</div>
            </div>
            <div class="font-mono font-bold text-tone-emerald tabular-nums">{{ formatIDR(acc.balance) }}</div>
          </div>
        </div>
      </section>

      <!-- 3-Way Bank CSV Reconciliation -->
      <section class="oms-panel rounded-2xl bg-card border border-border shadow-xs p-4 sm:p-5 space-y-4">
        <div>
          <Badge variant="indigo">
            <FileSpreadsheet class="w-3 h-3" />
            <span>3-WAY SETTLEMENT RECON</span>
          </Badge>
          <h2 class="text-sm sm:text-base font-bold text-foreground mt-1.5">
            Streaming Bank Settlement CSV Matcher
          </h2>
          <p class="text-xs text-muted-foreground mt-0.5">
            Mencocokkan laporan CSV mutasi bank (BCA/Mandiri/BRI) dengan buku besar internal gateway.
          </p>
        </div>

        <div class="space-y-2.5 text-xs">
          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
            <select
              v-model="bankName"
              aria-label="Pilih bank settlement"
              class="rounded-xl bg-elevated border border-border px-3 py-2 text-foreground font-bold focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option value="BCA">BCA Corporate</option>
              <option value="MANDIRI">Bank Mandiri</option>
              <option value="BRI">BRI Institutional</option>
            </select>
            <Button variant="indigo" :disabled="isReconciling" @click="handleUploadRecon">
              <span>{{ isReconciling ? 'Memproses CSV...' : 'Run 3-Way CSV Reconciliation' }}</span>
            </Button>
          </div>
          <textarea
            v-model="sampleCSV"
            rows="3"
            aria-label="Data CSV settlement bank"
            class="w-full rounded-xl bg-elevated border border-border p-3 font-mono text-[11px] text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          ></textarea>
        </div>

        <div class="max-h-40 overflow-y-auto custom-scroll divide-y divide-border text-xs">
          <div
            v-for="b in store.reconBatches"
            :key="b.id"
            class="py-2 flex flex-wrap items-center justify-between gap-2 font-mono text-[11px]"
          >
            <div>
              <span class="text-foreground font-bold">{{ b.batch_number }}</span>
              <span class="text-muted-foreground ml-2">({{ b.bank_name }})</span>
            </div>
            <div class="flex items-center gap-2 tabular-nums">
              <Badge variant="emerald">Matched: {{ b.matched_count }}</Badge>
              <Badge variant="amber">Missing: {{ b.missing_in_gateway_count }}</Badge>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Immutable Double-Entry Ledger -->
    <section class="oms-panel rounded-2xl bg-card border border-border shadow-xs overflow-hidden">
      <div class="p-4 border-b border-border">
        <h3 id="double-entry-ledger-caption" class="text-sm font-bold text-foreground">
          Immutable Double-Entry Bookkeeping Ledger (DEBIT / CREDIT)
        </h3>
      </div>
      <div
        tabindex="0"
        role="region"
        aria-labelledby="double-entry-ledger-caption"
        class="table-wrapper max-h-64 custom-scroll"
      >
        <table
          class="responsive-data-table text-xs text-left"
          style="--label-2: 'Entry Type'; --label-3: 'Amount'; --label-4: 'Balance After'"
        >
          <thead class="bg-elevated text-muted-foreground uppercase text-[10px] border-b border-border">
            <tr>
              <th scope="col" class="px-4 py-2.5 bg-elevated">Transaction ID</th>
              <th scope="col" class="px-4 py-2.5 bg-elevated">Entry Type</th>
              <th scope="col" class="px-4 py-2.5 bg-elevated text-right">Amount</th>
              <th scope="col" class="px-4 py-2.5 bg-elevated text-right">Balance After</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border font-mono text-[11px]">
            <tr v-if="store.ledgerEntries.length === 0">
              <td colspan="4" class="px-4 py-6 text-center text-muted-foreground font-sans">
                Belum ada mutasi ledger transfer internal.
              </td>
            </tr>
            <tr v-for="le in store.ledgerEntries" :key="le.id" class="data-row">
              <th scope="row" class="px-4 py-2.5 text-left font-mono text-secondary-foreground">
                {{ le.transaction_id }}
              </th>
              <td class="px-4 py-2.5">
                <Badge :variant="le.entry_type === 'CREDIT' ? 'emerald' : 'rose'">
                  {{ le.entry_type }}
                </Badge>
              </td>
              <td class="px-4 py-2.5 text-right text-foreground font-semibold tabular-nums">
                {{ formatIDR(le.amount) }}
              </td>
              <td class="px-4 py-2.5 text-right text-secondary-foreground tabular-nums">
                {{ formatIDR(le.balance_after) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
