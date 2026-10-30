<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useOmsStore } from '../stores/omsStore'

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
      store.notify('success', 'Akun Vault Dibuat (AES-256-GCM)', `NIK dienkripsi sebelum masuk PostgreSQL`)
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

function formatIDR(val: number) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val)
}

onMounted(() => {
  store.fetchCoreGatewayData()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Accounts & AES-256-GCM Encryption + Recon CSV -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      <section class="oms-panel rounded-2xl bg-slate-900/80 border border-slate-800 p-5 space-y-4">
        <div>
          <span class="px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-400 font-mono text-[10px] font-bold">
            AES-256-GCM VAULT
          </span>
          <h2 class="text-base font-bold text-white mt-1">
            Merchant Settlement Accounts &amp; Field-Level Encryption
          </h2>
          <p class="text-xs text-slate-400">
            Identitas sensitif (NIK/NPWP) dienkripsi dengan AEAD AES-256-GCM sebelum disimpan ke PostgreSQL.
          </p>
        </div>

        <form @submit.prevent="handleCreateAccount" class="grid grid-cols-1 sm:grid-cols-3 gap-2.5 text-xs">
          <input
            v-model="holderName"
            type="text"
            placeholder="Nama Pemilik Akun"
            class="rounded-xl bg-slate-950 border border-slate-700 px-3 py-2 text-slate-100"
          />
          <input
            v-model="identityNumber"
            type="text"
            placeholder="NIK / NPWP (Dienkripsi)"
            class="rounded-xl bg-slate-950 border border-slate-700 px-3 py-2 text-slate-100 font-mono"
          />
          <button
            type="submit"
            :disabled="isCreatingAccount"
            class="rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-extrabold px-4 py-2 cursor-pointer"
          >
            + Buat Akun Vault
          </button>
        </form>

        <div class="max-h-56 overflow-y-auto custom-scroll divide-y divide-slate-800 text-xs">
          <div v-if="store.gatewayAccounts.length === 0" class="py-6 text-center text-slate-500">
            Belum ada akun settlement terdaftar. Buat akun baru di atas.
          </div>
          <div
            v-for="acc in store.gatewayAccounts"
            :key="acc.id"
            class="py-2.5 flex items-center justify-between gap-2"
          >
            <div>
              <div class="font-semibold text-white">{{ acc.holder_name }}</div>
              <div class="font-mono text-[11px] text-slate-400">{{ acc.account_number }} &bull; v{{ acc.version }}</div>
            </div>
            <div class="font-mono font-bold text-emerald-400">{{ formatIDR(acc.balance) }}</div>
          </div>
        </div>
      </section>

      <!-- 3-Way Bank CSV Reconciliation -->
      <section class="oms-panel rounded-2xl bg-slate-900/80 border border-slate-800 p-5 space-y-4">
        <div>
          <span class="px-2 py-0.5 rounded bg-indigo-500/15 text-indigo-300 font-mono text-[10px] font-bold">
            3-WAY SETTLEMENT RECONCILIATION
          </span>
          <h2 class="text-base font-bold text-white mt-1">
            Streaming Bank Settlement CSV Matcher
          </h2>
          <p class="text-xs text-slate-400">
            Mencocokkan laporan CSV mutasi bank (BCA/Mandiri/BRI) dengan buku besar internal gateway.
          </p>
        </div>

        <div class="space-y-2 text-xs">
          <div class="flex items-center gap-2">
            <select
              v-model="bankName"
              class="rounded-xl bg-slate-950 border border-slate-700 px-3 py-1.5 text-slate-200 font-bold"
            >
              <option value="BCA">BCA Corporate</option>
              <option value="MANDIRI">Bank Mandiri</option>
              <option value="BRI">BRI Institutional</option>
            </select>
            <button
              type="button"
              :disabled="isReconciling"
              @click="handleUploadRecon"
              class="px-4 py-1.5 rounded-xl bg-indigo-500 hover:bg-indigo-400 text-white font-bold cursor-pointer"
            >
              {{ isReconciling ? 'Memproses CSV...' : 'Run 3-Way CSV Reconciliation' }}
            </button>
          </div>
          <textarea
            v-model="sampleCSV"
            rows="3"
            class="w-full rounded-xl bg-slate-950 border border-slate-700 p-2.5 font-mono text-[11px] text-slate-200"
          ></textarea>
        </div>

        <div class="max-h-40 overflow-y-auto custom-scroll divide-y divide-slate-800 text-xs">
          <div
            v-for="b in store.reconBatches"
            :key="b.id"
            class="py-2 flex items-center justify-between font-mono text-[11px]"
          >
            <div>
              <span class="text-white font-bold">{{ b.batch_number }}</span>
              <span class="text-slate-400 ml-2">({{ b.bank_name }})</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-emerald-400">Matched: {{ b.matched_count }}</span>
              <span class="text-amber-400">Missing: {{ b.missing_in_gateway_count }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Immutable Double-Entry Ledger -->
    <section class="oms-panel rounded-2xl bg-slate-900/80 border border-slate-800 overflow-hidden">
      <div class="p-4 border-b border-slate-800">
        <h3 class="text-sm font-bold text-white">Immutable Double-Entry Bookkeeping Ledger (DEBIT / CREDIT)</h3>
      </div>
      <div class="max-h-64 overflow-y-auto custom-scroll">
        <table class="w-full text-xs text-left">
          <thead class="bg-slate-950 text-slate-400 uppercase text-[10px]">
            <tr>
              <th class="px-4 py-2.5">Transaction ID</th>
              <th class="px-4 py-2.5">Entry Type</th>
              <th class="px-4 py-2.5 text-right">Amount</th>
              <th class="px-4 py-2.5 text-right">Balance After</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800 font-mono text-[11px]">
            <tr v-if="store.ledgerEntries.length === 0">
              <td colspan="4" class="px-4 py-6 text-center text-slate-500 font-sans">
                Belum ada mutasi ledger transfer internal.
              </td>
            </tr>
            <tr v-for="le in store.ledgerEntries" :key="le.id">
              <td class="px-4 py-2 text-slate-300">{{ le.transaction_id }}</td>
              <td class="px-4 py-2">
                <span
                  :class="le.entry_type === 'CREDIT' ? 'text-emerald-400' : 'text-rose-400'"
                  class="font-bold"
                >
                  {{ le.entry_type }}
                </span>
              </td>
              <td class="px-4 py-2 text-right text-white">{{ formatIDR(le.amount) }}</td>
              <td class="px-4 py-2 text-right text-slate-300">{{ formatIDR(le.balance_after) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

