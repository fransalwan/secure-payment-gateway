<script setup lang="ts">
import { ref } from 'vue'
import { toast } from 'vue-sonner'
import ResponsiveDialog from './ui/ResponsiveDialog.vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { formatIDR } from '../lib/utils'
import {
  UploadCloud,
  CheckCircle2,
  AlertTriangle,
  ShieldAlert,
  ShieldCheck,
  FileText,
  Clock,
  Sparkles,
  RefreshCw,
} from '@lucide/vue'

const store = useOmsStore()

const isOpen = ref(false)
const selectedOrderID = ref('')
const selectedFile = ref<File | null>(null)
const previewURL = ref<string | null>(null)
const isSubmitting = ref(false)
const isPolling = ref(false)

interface VerificationResult {
  id?: string
  proof_id?: string
  order_id: string
  order_number: string
  status: string
  bank_name?: string
  sender_name?: string
  amount?: number
  reference_number?: string
  tamper_detected?: boolean
  confidence_score?: number
  rejection_reason?: string
  latency_ms?: number
}

const lastResult = ref<VerificationResult | null>(null)

function openModal(preselectedOrderID?: string) {
  isOpen.value = true
  lastResult.value = null
  selectedFile.value = null
  previewURL.value = null

  if (preselectedOrderID) {
    selectedOrderID.value = preselectedOrderID
  } else {
    // Pick the first pending or suspicious order if available
    const pending = store.orders.find(
      (o) => o.status === 'PENDING_PAYMENT' || o.status === 'SUSPICIOUS_HOLD',
    )
    selectedOrderID.value = pending ? pending.id : (store.orders[0]?.id ?? '')
  }
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files[0]) {
    setFile(target.files[0])
  }
}

function setFile(file: File) {
  selectedFile.value = file
  previewURL.value = URL.createObjectURL(file)
  lastResult.value = null
}

// 1-Click Simulation Presets for Live Interview Demo
function applyPreset(type: 'authentic' | 'tampered' | 'mismatch' | 'duplicate') {
  const currentOrder = store.orders.find((o) => o.id === selectedOrderID.value)
  const targetAmount = currentOrder ? currentOrder.total_amount : 378000

  let content = ''
  let filename = ''

  if (type === 'authentic') {
    content = `TRANSFER RECEIPT BCA Rp ${targetAmount} REF-AUTO-${Date.now()}`
    filename = 'bca_transfer_slip_authentic.jpg'
  } else if (type === 'tampered') {
    content = `SIMULATE_TAMPER fake_photoshop_receipt font_glitch Rp ${targetAmount}`
    filename = 'photoshop_slip_tampered.jpg'
  } else if (type === 'duplicate') {
    content = `DUPLICATE_RECEIPT_PHASH_SIMULATION_PAYLOAD_REF_9999`
    filename = 'reused_duplicate_slip.jpg'
  } else {
    content = `SIMULATE_AMOUNT_MISMATCH Rp 50.000 underpaid_slip`
    filename = 'underpaid_transfer_slip.jpg'
  }

  const blob = new Blob([content], { type: 'image/jpeg' })
  const mockFile = new File([blob], filename, { type: 'image/jpeg' })
  setFile(mockFile)
}

async function uploadAndVerify() {
  if (!selectedOrderID.value) {
    toast.error('Pilih order target terlebih dahulu')
    return
  }
  if (!selectedFile.value) {
    toast.error('Unggah file struk transfer atau gunakan preset')
    return
  }

  isSubmitting.value = true
  lastResult.value = null

  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    formData.append('order_id', selectedOrderID.value)

    const resp = await fetch('/api/v1/payments/proof', {
      method: 'POST',
      body: formData,
    })

    const data = await resp.json()
    if (!resp.ok) {
      throw new Error(data.message || 'Upload gagal')
    }

    toast.success('Struk berhasil diunggah! Memulai verifikasi Vision AI...')
    const proofID = data.data.proof_id

    // Fast ACK received, now poll for verification outcome
    await pollProofResult(proofID)
  } catch (err: any) {
    toast.error(err.message || 'Gagal mengunggah struk transfer')
  } finally {
    isSubmitting.value = false
  }
}

async function pollProofResult(proofID: string) {
  isPolling.value = true
  let attempts = 0
  const maxAttempts = 15

  const interval = setInterval(async () => {
    attempts++
    try {
      const resp = await fetch(`/api/v1/payments/proofs/${proofID}`)
      if (resp.ok) {
        const body = await resp.json()
        const proof = body.data
        if (proof && proof.status !== 'PENDING') {
          clearInterval(interval)
          isPolling.value = false
          lastResult.value = proof
          await store.fetchOverview() // Refresh orders in store

          if (proof.status === 'VERIFIED') {
            toast.success('Bukti transfer ASLI & TERVERIFIKASI! Order dialihkan ke status PAID.')
          } else if (proof.status === 'SUSPICIOUS_HOLD') {
            toast.warning('Peringatan: Bukti transfer ditandai SUSPICIOUS HOLD untuk review manual.')
          }
          return
        }
      }
    } catch (e) {
      // Keep polling
    }

    if (attempts >= maxAttempts) {
      clearInterval(interval)
      isPolling.value = false
      toast.info('Verifikasi sedang diproses oleh background worker.')
    }
  }, 500)
}

defineExpose({
  openModal,
})
</script>

<template>
  <ResponsiveDialog
    v-model:open="isOpen"
    title="Multimodal Vision AI Bank Slip Validator"
    description="Validasi otomatis struk transfer bank (BCA, Mandiri, BRI) via Vision AI asinkron dengan deteksi manipulasi & pencocokan nominal"
    content-class="max-w-2xl"
  >
    <div class="space-y-5 py-2">
      <!-- Target Order Selector -->
      <div>
        <label class="block text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1.5">
          Pilih Order Target
        </label>
        <select
          v-model="selectedOrderID"
          class="w-full rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        >
          <option
            v-for="ord in store.orders"
            :key="ord.id"
            :value="ord.id"
          >
            {{ ord.order_number }} · {{ ord.customer_name }} · {{ formatIDR(ord.total_amount) }} [{{ ord.status }}]
          </option>
        </select>
      </div>

      <!-- Presets for Fast Live Demo -->
      <div class="rounded-xl border border-indigo-100 dark:border-indigo-950/60 bg-indigo-50/50 dark:bg-indigo-950/20 p-3.5">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-bold text-indigo-900 dark:text-indigo-300 flex items-center gap-1.5">
            <Sparkles class="w-3.5 h-3.5" />
            Preset Skenario Live Demo (1-Click)
          </span>
          <span class="text-[11px] text-indigo-700 dark:text-indigo-400">Pilih skenario uji</span>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
          <button
            type="button"
            class="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-emerald-600 text-white hover:bg-emerald-700 transition flex items-center justify-center gap-1.5 shadow-sm"
            @click="applyPreset('authentic')"
          >
            <ShieldCheck class="w-3.5 h-3.5" />
            1. Asli (Match)
          </button>
          <button
            type="button"
            class="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-rose-600 text-white hover:bg-rose-700 transition flex items-center justify-center gap-1.5 shadow-sm"
            @click="applyPreset('tampered')"
          >
            <ShieldAlert class="w-3.5 h-3.5" />
            2. Editan (Tamper)
          </button>
          <button
            type="button"
            class="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-amber-600 text-white hover:bg-amber-700 transition flex items-center justify-center gap-1.5 shadow-sm"
            @click="applyPreset('mismatch')"
          >
            <AlertTriangle class="w-3.5 h-3.5" />
            3. Kurang Bayar
          </button>
          <button
            type="button"
            class="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-purple-600 text-white hover:bg-purple-700 transition flex items-center justify-center gap-1.5 shadow-sm"
            @click="applyPreset('duplicate')"
          >
            <RefreshCw class="w-3.5 h-3.5" />
            4. Duplikat (pHash)
          </button>
        </div>
      </div>

      <!-- Upload Dropzone / Preview -->
      <div
        class="border-2 border-dashed rounded-xl p-5 text-center transition flex flex-col items-center justify-center cursor-pointer"
        :class="selectedFile ? 'border-indigo-500 bg-indigo-50/20 dark:bg-indigo-950/10' : 'border-slate-300 dark:border-slate-700 hover:border-indigo-400'"
        @click="($refs.fileInput as HTMLInputElement)?.click()"
      >
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          class="hidden"
          @change="handleFileSelect"
        />

        <div v-if="selectedFile" class="space-y-2">
          <div class="w-10 h-10 rounded-full bg-indigo-100 dark:bg-indigo-900/40 text-indigo-600 dark:text-indigo-400 flex items-center justify-center mx-auto">
            <FileText class="w-5 h-5" />
          </div>
          <div class="text-sm font-semibold text-slate-800 dark:text-slate-200">
            {{ selectedFile.name }}
          </div>
          <div class="text-xs text-slate-500">
            Ukuran: {{ (selectedFile.size / 1024).toFixed(1) }} KB · Klik untuk mengganti
          </div>
        </div>

        <div v-else class="space-y-2">
          <div class="w-10 h-10 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-500 flex items-center justify-center mx-auto">
            <UploadCloud class="w-5 h-5" />
          </div>
          <div class="text-sm font-medium text-slate-700 dark:text-slate-300">
            Tarik & Lepas gambar struk transfer di sini atau klik untuk browse
          </div>
          <div class="text-xs text-slate-400">
            Mendukung file JPEG, PNG, WebP (maks 6MB)
          </div>
        </div>
      </div>

      <!-- Action Button -->
      <div class="flex items-center justify-between pt-1">
        <div class="text-xs text-slate-500 flex items-center gap-1.5">
          <Clock class="w-3.5 h-3.5 text-indigo-500" />
          Fast ACK &lt; 15ms via RabbitMQ Pipeline
        </div>
        <Button
          variant="indigo"
          :disabled="isSubmitting || !selectedFile || !selectedOrderID"
          @click="uploadAndVerify"
        >
          <RefreshCw v-if="isSubmitting || isPolling" class="w-4 h-4 mr-2 animate-spin" />
          <Sparkles v-else class="w-4 h-4 mr-2" />
          {{ isSubmitting ? 'Mengunggah...' : isPolling ? 'Memverifikasi via AI...' : 'Upload & Ekstrak Struk' }}
        </Button>
      </div>

      <!-- Result Card (After Vision AI completes) -->
      <div
        v-if="lastResult"
        class="rounded-xl border p-4 space-y-3 transition-all"
        :class="lastResult.status === 'VERIFIED'
          ? 'border-emerald-200 bg-emerald-50/40 dark:border-emerald-950 dark:bg-emerald-950/20'
          : 'border-amber-200 bg-amber-50/40 dark:border-amber-950 dark:bg-amber-950/20'"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Badge :variant="lastResult.status === 'VERIFIED' ? 'emerald' : 'amber'">
              {{ lastResult.status === 'VERIFIED' ? 'VERIFIED (PAID)' : 'SUSPICIOUS HOLD' }}
            </Badge>
            <span class="text-xs font-semibold text-slate-700 dark:text-slate-300">
              Confidence: {{ ((lastResult.confidence_score || 0) * 100).toFixed(1) }}%
            </span>
          </div>
          <span class="text-[11px] font-mono text-slate-400">
            Proof ID: {{ (lastResult.id || lastResult.proof_id)?.substring(0, 8) }}
          </span>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
          <div class="rounded-lg bg-white/70 dark:bg-slate-900/60 p-2 border border-slate-200/60 dark:border-slate-800">
            <span class="text-slate-400 block text-[10px] uppercase">Bank</span>
            <span class="font-bold text-slate-800 dark:text-slate-200">{{ lastResult.bank_name || '-' }}</span>
          </div>
          <div class="rounded-lg bg-white/70 dark:bg-slate-900/60 p-2 border border-slate-200/60 dark:border-slate-800">
            <span class="text-slate-400 block text-[10px] uppercase">Pengirim</span>
            <span class="font-bold text-slate-800 dark:text-slate-200 truncate block">{{ lastResult.sender_name || '-' }}</span>
          </div>
          <div class="rounded-lg bg-white/70 dark:bg-slate-900/60 p-2 border border-slate-200/60 dark:border-slate-800">
            <span class="text-slate-400 block text-[10px] uppercase">Nominal Struk</span>
            <span class="font-bold text-slate-800 dark:text-slate-200">{{ formatIDR(lastResult.amount || 0) }}</span>
          </div>
          <div class="rounded-lg bg-white/70 dark:bg-slate-900/60 p-2 border border-slate-200/60 dark:border-slate-800">
            <span class="text-slate-400 block text-[10px] uppercase">Manipulasi</span>
            <span
              class="font-bold"
              :class="lastResult.tamper_detected ? 'text-rose-600' : 'text-emerald-600'"
            >
              {{ lastResult.tamper_detected ? 'TERDETEKSI FAKE' : 'ASLI (BERSIH)' }}
            </span>
          </div>
        </div>

        <div v-if="lastResult.rejection_reason" class="text-xs text-rose-700 dark:text-rose-400 font-medium bg-rose-50 dark:bg-rose-950/40 p-2.5 rounded-lg border border-rose-200 dark:border-rose-900 flex items-start gap-2">
          <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
          <span>{{ lastResult.rejection_reason }}</span>
        </div>
      </div>
    </div>
  </ResponsiveDialog>
</template>
