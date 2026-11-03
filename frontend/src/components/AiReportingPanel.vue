<script setup lang="ts">
import { ref } from 'vue'
import { toast } from 'vue-sonner'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import {
  Sparkles,
  Database,
  ShieldCheck,
  Clock,
  Terminal,
  Copy,
  Check,
  BarChart3,
  PieChart,
  Table as TableIcon,
  RefreshCw,
  HelpCircle,
} from '@lucide/vue'

const questionInput = ref('Tampilkan omset dan total order per channel penjualan')
const isLoading = ref(false)
const copied = ref(false)

interface ReportResult {
  question: string
  generated_sql: string
  explanation: string
  chart_type: string
  columns: string[]
  rows: Record<string, any>[]
  total_rows: number
  execution_ms: number
  sanitized_safe: boolean
  tokens_used?: number
}

const reportResult = ref<ReportResult | null>(null)

const presetQuestions = [
  'Tampilkan omset dan total order per channel penjualan',
  '10 produk SKU terlaris dan estimasi revenue',
  'Analisis pesanan yang dibatalkan dan lost GMV per channel',
  'Distribusi verifikasi Vision AI bukti transfer bank',
  'Ringkasan status seluruh pesanan toko',
]

function selectPreset(q: string) {
  questionInput.value = q
  runQuery()
}

async function runQuery() {
  const q = questionInput.value.trim()
  if (!q) {
    toast.error('Masukkan pertanyaan analitik terlebih dahulu')
    return
  }

  isLoading.value = true
  try {
    const resp = await fetch('/api/v1/reports/ask-ai', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question: q }),
    })

    const body = await resp.json()
    if (!resp.ok) {
      throw new Error(body.error?.details || body.message || 'Gagal memproses query')
    }

    reportResult.value = body.data
    toast.success('Query Text-to-SQL berhasil dieksekusi!')
  } catch (err: any) {
    toast.error(err.message || 'Gagal menghasilkan laporan analitik')
  } finally {
    isLoading.value = false
  }
}

function copySQL() {
  if (!reportResult.value?.generated_sql) return
  navigator.clipboard.writeText(reportResult.value.generated_sql)
  copied.value = true
  toast.success('SQL query disalin ke clipboard')
  setTimeout(() => {
    copied.value = false
  }, 2000)
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header Banner -->
    <div class="rounded-2xl border border-indigo-200/80 dark:border-indigo-950/60 bg-gradient-to-br from-indigo-50/60 via-white to-slate-50 dark:from-indigo-950/30 dark:via-slate-900 dark:to-slate-900 p-6 shadow-sm">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold bg-indigo-100 dark:bg-indigo-900/50 text-indigo-700 dark:text-indigo-300 mb-2">
            <Sparkles class="w-3.5 h-3.5" />
            ENTERPRISE TEXT-TO-SQL REPORTING ENGINE
          </div>
          <h2 class="text-xl md:text-2xl font-black text-slate-900 dark:text-slate-100 tracking-tight">
            Financial & Operations Natural Language Analytics
          </h2>
          <p class="text-sm text-slate-600 dark:text-slate-400 mt-1 max-w-3xl">
            Tanyakan pertanyaan bisnis dalam bahasa alami. Sistem mengonversinya menjadi query PostgreSQL berkecepatan tinggi dengan validasi AST anti-SQL Injection dan proteksi Read-Only session.
          </p>
        </div>
        <div class="flex items-center gap-2">
          <Badge variant="emerald" class="py-1">
            <ShieldCheck class="w-3.5 h-3.5 mr-1" />
            Strict AST Guardrail
          </Badge>
          <Badge variant="indigo" class="py-1">
            <Clock class="w-3.5 h-3.5 mr-1" />
            Timeout: 2500ms
          </Badge>
        </div>
      </div>

      <!-- Search Input Box -->
      <div class="mt-5 space-y-3">
        <div class="relative flex items-center">
          <input
            v-model="questionInput"
            type="text"
            placeholder="Ketik pertanyaan bisnis (misal: 'Berapa total GMV per channel?')..."
            class="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 pl-4 pr-32 py-3.5 text-sm text-slate-900 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 shadow-sm"
            @keydown.enter="runQuery"
          />
          <div class="absolute right-2">
            <Button
              variant="indigo"
              :disabled="isLoading"
              @click="runQuery"
            >
              <RefreshCw v-if="isLoading" class="w-4 h-4 mr-1.5 animate-spin" />
              <Sparkles v-else class="w-4 h-4 mr-1.5" />
              {{ isLoading ? 'Analisis...' : 'Generate SQL' }}
            </Button>
          </div>
        </div>

        <!-- Presets -->
        <div class="flex flex-wrap items-center gap-1.5 pt-1">
          <span class="text-xs font-semibold text-slate-400 flex items-center gap-1">
            <HelpCircle class="w-3.5 h-3.5" />
            Contoh Pertanyaan:
          </span>
          <button
            v-for="(preset, idx) in presetQuestions"
            :key="idx"
            type="button"
            class="text-xs px-2.5 py-1 rounded-lg bg-slate-100 hover:bg-indigo-50 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 hover:text-indigo-600 dark:hover:text-indigo-400 transition border border-slate-200 dark:border-slate-700/60"
            @click="selectPreset(preset)"
          >
            {{ preset }}
          </button>
        </div>
      </div>
    </div>

    <!-- Query Result Section -->
    <div v-if="reportResult" class="space-y-5 animate-in fade-in duration-300">
      <!-- Metadata Strip -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div class="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-3.5">
          <span class="text-slate-400 text-[11px] uppercase tracking-wider font-semibold block">Eksekusi Database</span>
          <div class="text-lg font-black text-slate-900 dark:text-slate-100 mt-0.5">
            {{ reportResult.execution_ms }} ms
          </div>
        </div>
        <div class="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-3.5">
          <span class="text-slate-400 text-[11px] uppercase tracking-wider font-semibold block">Baris Data</span>
          <div class="text-lg font-black text-slate-900 dark:text-slate-100 mt-0.5">
            {{ reportResult.total_rows }} baris
          </div>
        </div>
        <div class="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-3.5">
          <span class="text-slate-400 text-[11px] uppercase tracking-wider font-semibold block">Rekomendasi Visualisasi</span>
          <div class="text-sm font-bold text-indigo-600 dark:text-indigo-400 mt-1 flex items-center gap-1.5 uppercase">
            <BarChart3 v-if="reportResult.chart_type === 'bar'" class="w-4 h-4" />
            <PieChart v-else-if="reportResult.chart_type === 'pie'" class="w-4 h-4" />
            <TableIcon v-else class="w-4 h-4" />
            {{ reportResult.chart_type }} CHART
          </div>
        </div>
        <div class="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-3.5">
          <span class="text-slate-400 text-[11px] uppercase tracking-wider font-semibold block">Keamanan Kueri</span>
          <div class="text-sm font-bold text-emerald-600 dark:text-emerald-400 mt-1 flex items-center gap-1">
            <ShieldCheck class="w-4 h-4" />
            READ-ONLY ENFORCED
          </div>
        </div>
      </div>

      <!-- Generated SQL & Explanation Box -->
      <div class="rounded-xl border border-slate-800 bg-slate-950 p-4 text-slate-200 shadow-md">
        <div class="flex items-center justify-between pb-2 mb-2 border-b border-slate-800">
          <div class="flex items-center gap-2">
            <Terminal class="w-4 h-4 text-cyan-400" />
            <span class="text-xs font-mono font-bold text-cyan-400">POSTGRESQL 16 GENERATED QUERY</span>
          </div>
          <button
            type="button"
            class="text-xs text-slate-400 hover:text-white transition flex items-center gap-1 px-2 py-0.5 rounded bg-slate-800"
            @click="copySQL"
          >
            <Check v-if="copied" class="w-3.5 h-3.5 text-emerald-400" />
            <Copy v-else class="w-3.5 h-3.5" />
            {{ copied ? 'Disalin' : 'Salin SQL' }}
          </button>
        </div>
        <pre class="font-mono text-xs overflow-x-auto text-emerald-300 py-1 leading-relaxed">{{ reportResult.generated_sql }}</pre>
        <div v-if="reportResult.explanation" class="mt-3 pt-2.5 border-t border-slate-800/80 text-xs text-slate-400 flex items-start gap-2">
          <Database class="w-3.5 h-3.5 shrink-0 mt-0.5 text-indigo-400" />
          <span>{{ reportResult.explanation }}</span>
        </div>
      </div>

      <!-- Tabular Data Display -->
      <div class="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 overflow-hidden shadow-sm">
        <div class="px-5 py-3 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <span class="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-300">
            Hasil Eksekusi Kueri
          </span>
          <span class="text-xs font-mono text-slate-400">
            Total {{ reportResult.rows.length }} Data
          </span>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead class="bg-slate-50 dark:bg-slate-800/60 text-slate-500 uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th
                  v-for="col in reportResult.columns"
                  :key="col"
                  class="px-4 py-3 font-bold"
                >
                  {{ col }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
              <tr
                v-for="(row, rIdx) in reportResult.rows"
                :key="rIdx"
                class="hover:bg-slate-50/70 dark:hover:bg-slate-800/40 transition"
              >
                <td
                  v-for="col in reportResult.columns"
                  :key="col"
                  class="px-4 py-3 font-mono text-slate-800 dark:text-slate-200 whitespace-nowrap"
                >
                  {{ row[col] !== null && row[col] !== undefined ? row[col] : '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>
