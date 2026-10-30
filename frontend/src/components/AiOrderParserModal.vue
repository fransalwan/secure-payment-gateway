<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useOmsStore } from '../stores/omsStore'
import type { AIParseResponse } from '../types/oms'

const store = useOmsStore()
const dialogRef = ref<HTMLDialogElement | null>(null)

const samplePresets = [
  {
    label: 'Preset 1 (Stok Kurang Warning — 2 pcs vs 1 stok)',
    text: 'Kak mau pesan Kaos Oversize Hitam XL 2 pcs sama Topi Baseball Navy 1 pcs, kirim ke Budi Santoso 081299887766, Jl. Kaliurang Km 5 No 12 Sleman via JNE REG ya',
  },
  {
    label: 'Preset 2 (Order Valid — Hoodie + Totebag via SiCepat)',
    text: 'Min order Hoodie Fleece Misty Grey L 1 pcs dan Totebag Canvas Waterproof Hitam 2 pcs, atas nama Siti Rahmawati 085788991122, Jl. Dago No 84 Coblong Bandung pakai SICEPAT ya',
  },
  {
    label: 'Preset 3 (FB/TikTok Live — Sneakers Putih via J&T)',
    text: 'Fix ambil Sneakers Vulcanized Classic White 42 1 pasang + Topi Baseball Navy 1 pcs kirim ke Andi Pratama 081377665544, Jl. Sudirman No 19 Jakarta Selatan via J&T',
  },
]

const rawChat = ref<string>(samplePresets[0].text)
const parsedDraft = ref<AIParseResponse | null>(null)
const isParsing = ref(false)
const isSubmitting = ref(false)

const computedTotal = computed(() => {
  if (!parsedDraft.value) return 0
  return parsedDraft.value.items.reduce((sum, it) => sum + it.qty * it.unit_price, 0)
})

const hasStockShortage = computed(() => {
  if (!parsedDraft.value) return false
  return parsedDraft.value.items.some((it) => it.qty > it.available_stock)
})

function openModal() {
  dialogRef.value?.showModal()
}

function closeModal() {
  dialogRef.value?.close()
}

function applyPreset(text: string) {
  rawChat.value = text
  parsedDraft.value = null
}

function adjustItemQty(index: number, delta: number) {
  if (!parsedDraft.value) return
  const item = parsedDraft.value.items[index]
  if (!item) return
  const next = Math.max(1, item.qty + delta)
  item.qty = next
  item.subtotal = next * item.unit_price
  item.stock_sufficient = item.available_stock >= next
  item.stock_warning = item.stock_sufficient
    ? ''
    : `Stok kurang! Tersedia ${item.available_stock} pcs, diminta ${next} pcs`
}

async function handleExtract() {
  if (!rawChat.value.trim()) return
  isParsing.value = true
  try {
    parsedDraft.value = await store.parseSocialChat(rawChat.value)
  } finally {
    isParsing.value = false
  }
}

async function handleCreateOrder() {
  if (!parsedDraft.value || parsedDraft.value.items.length === 0) return
  isSubmitting.value = true
  try {
    const created = await store.createOrder({
      channel: 'WHATSAPP_LIVE',
      customer_name: parsedDraft.value.customer_name,
      customer_phone: parsedDraft.value.customer_phone,
      shipping_address: parsedDraft.value.shipping_address,
      courier_code: parsedDraft.value.courier_code,
      items: parsedDraft.value.items.map((it) => ({
        product_id: it.sku_id,
        qty: it.qty,
      })),
    })
    if (created) {
      parsedDraft.value = null
      closeModal()
    }
  } finally {
    isSubmitting.value = false
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
  const dialog = dialogRef.value
  // Fallback for browsers without declarative <dialog closedby="any"> support
  if (dialog && !('closedBy' in HTMLDialogElement.prototype)) {
    dialog.addEventListener('click', (event: MouseEvent) => {
      if (event.target !== dialog) return
      const rect = dialog.getBoundingClientRect()
      const isDialogContent =
        rect.top <= event.clientY &&
        event.clientY <= rect.top + rect.height &&
        rect.left <= event.clientX &&
        event.clientX <= rect.left + rect.width
      if (!isDialogContent) dialog.close()
    })
  }
})

defineExpose({ openModal })
</script>

<template>
  <dialog
    ref="dialogRef"
    v-bind="{ closedby: 'any' } as any"
    aria-labelledby="ai-parser-dialog-title"
    class="w-full max-w-3xl rounded-2xl bg-slate-900 text-slate-100 border border-slate-700/80 shadow-2xl p-0 overflow-hidden"
  >
    <div class="px-6 py-4 bg-slate-950 border-b border-slate-800 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-xl bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400 font-bold text-sm">
          AI
        </div>
        <div>
          <h2 id="ai-parser-dialog-title" class="text-base font-bold text-white">
            AI WhatsApp &amp; Live Selling Order Capture
          </h2>
          <p class="text-xs text-slate-400">
            Ekstraksi chat tidak terstruktur ke katalog SKU PostgreSQL (&lt; 3 detik SLA + Fallback Manual)
          </p>
        </div>
      </div>
      <form method="dialog">
        <button
          type="submit"
          aria-label="Tutup modal AI Parser"
          class="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold cursor-pointer"
        >
          Esc / Tutup
        </button>
      </form>
    </div>

    <div class="p-6 space-y-4 max-h-[80vh] overflow-y-auto custom-scroll">
      <!-- Quick Chat Presets -->
      <div class="space-y-1.5">
        <label class="text-xs font-semibold text-slate-400 uppercase tracking-wider">
          Contoh Chat WhatsApp / Live Selling (Klik untuk isi cepat):
        </label>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="(preset, idx) in samplePresets"
            :key="idx"
            type="button"
            @click="applyPreset(preset.text)"
            class="px-2.5 py-1 rounded-lg bg-slate-800/90 hover:bg-slate-700 border border-slate-700 text-[11px] text-slate-200 transition cursor-pointer"
          >
            {{ preset.label }}
          </button>
        </div>
      </div>

      <!-- Raw Textarea -->
      <div>
        <label for="wa-raw-input" class="block text-xs font-semibold text-slate-300 mb-1.5">
          Teks Chat Mentah dari Pembeli:
        </label>
        <textarea
          id="wa-raw-input"
          v-model="rawChat"
          autofocus
          rows="3"
          class="w-full rounded-xl bg-slate-950 border border-slate-700 focus:border-emerald-500 focus:outline-none p-3 text-xs text-slate-100 leading-relaxed"
          placeholder="Paste chat order WhatsApp / komentar TikTok Live di sini..."
        ></textarea>
        <div class="mt-2.5 flex items-center justify-between">
          <span class="text-[11px] text-slate-400">
            Mencocokkan otomatis dengan katalog SKU milik <strong class="text-indigo-300">{{ store.activeMerchant?.name }}</strong>
          </span>
          <button
            type="button"
            :disabled="isParsing || !rawChat.trim()"
            @click="handleExtract"
            class="px-4 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-bold text-xs transition cursor-pointer flex items-center gap-2"
          >
            <span>{{ isParsing ? 'Mengekstrak dengan AI...' : 'Extract with AI' }}</span>
          </button>
        </div>
      </div>

      <!-- Extracted Draft Preview -->
      <div v-if="parsedDraft" class="space-y-4 pt-3 border-t border-slate-800">
        <div class="flex flex-wrap items-center justify-between gap-2 bg-slate-950/80 px-3.5 py-2 rounded-xl border border-slate-800 text-xs">
          <div class="flex items-center gap-2">
            <span class="px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300 font-mono text-[11px]">
              Engine: {{ parsedDraft.parser_engine }}
            </span>
            <span class="px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 font-mono text-[11px]">
              Latency: {{ parsedDraft.latency_ms }}ms (&lt; 3000ms SLA)
            </span>
          </div>
          <span class="font-bold text-emerald-400 font-mono">
            Estimasi Total: {{ formatIDR(computedTotal) }}
          </span>
        </div>

        <!-- RED STOCK INSUFFICIENT WARNING BANNER (FR-01 AC) -->
        <div
          v-if="hasStockShortage"
          role="alert"
          class="p-3.5 rounded-xl bg-rose-950/80 border border-rose-500/60 text-rose-200 text-xs flex items-start gap-3"
        >
          <span class="px-2 py-0.5 rounded bg-rose-500 text-slate-950 font-extrabold text-[10px] uppercase shrink-0 mt-0.5">
            Stok Kurang!
          </span>
          <div>
            <p class="font-bold text-rose-100">
              Peringatan Inventori Sebelum Order Dibuat:
            </p>
            <p class="text-rose-200/90 mt-0.5">
              Satu atau lebih SKU memiliki <code>available_stock</code> lebih kecil dari jumlah yang diminta pembeli. Kurangi <code>qty</code> di bawah agar tidak ditolak <code>409 Conflict</code> oleh PostgreSQL Row-Level Lock.
            </p>
          </div>
        </div>

        <!-- Editable Customer & Shipping Fields (Manual Fallback Ready) -->
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
          <div>
            <label class="block text-slate-400 mb-1">Nama Penerima</label>
            <input
              v-model="parsedDraft.customer_name"
              type="text"
              class="w-full rounded-lg bg-slate-950 border border-slate-700 px-3 py-1.5 text-slate-100"
            />
          </div>
          <div>
            <label class="block text-slate-400 mb-1">No. WhatsApp</label>
            <input
              v-model="parsedDraft.customer_phone"
              type="text"
              class="w-full rounded-lg bg-slate-950 border border-slate-700 px-3 py-1.5 text-slate-100 font-mono"
            />
          </div>
          <div>
            <label class="block text-slate-400 mb-1">Kurir Ekspedisi</label>
            <select
              v-model="parsedDraft.courier_code"
              class="w-full rounded-lg bg-slate-950 border border-slate-700 px-3 py-1.5 text-slate-100 font-semibold"
            >
              <option value="JNE">JNE REG</option>
              <option value="JNT">J&amp;T EZ</option>
              <option value="SICEPAT">SiCepat REG</option>
            </select>
          </div>
          <div class="sm:col-span-3">
            <label class="block text-slate-400 mb-1">Alamat Pengiriman Lengkap</label>
            <input
              v-model="parsedDraft.shipping_address"
              type="text"
              class="w-full rounded-lg bg-slate-950 border border-slate-700 px-3 py-1.5 text-slate-100"
            />
          </div>
        </div>

        <!-- Matched SKU Items Table -->
        <div class="rounded-xl border border-slate-800 overflow-hidden">
          <table class="w-full text-xs text-left">
            <thead class="bg-slate-950 text-slate-400 uppercase text-[10px]">
              <tr>
                <th class="px-3 py-2">SKU &amp; Produk</th>
                <th class="px-3 py-2 text-right">Harga</th>
                <th class="px-3 py-2 text-center">Stok Tersedia</th>
                <th class="px-3 py-2 text-center">Qty Order</th>
                <th class="px-3 py-2 text-right">Subtotal</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800 bg-slate-900/60">
              <tr
                v-for="(item, idx) in parsedDraft.items"
                :key="item.sku_id"
                :class="item.qty > item.available_stock ? 'bg-rose-950/35' : ''"
              >
                <td class="px-3 py-2.5">
                  <div class="font-semibold text-white">{{ item.matched_product_name }}</div>
                  <div class="font-mono text-[10px] text-slate-400">{{ item.sku }}</div>
                  <div v-if="item.qty > item.available_stock" class="text-[11px] font-semibold text-rose-400 mt-0.5">
                    ⚠ {{ item.stock_warning || `Stok tersedia hanya ${item.available_stock} pcs` }}
                  </div>
                </td>
                <td class="px-3 py-2.5 text-right font-mono text-slate-300">
                  {{ formatIDR(item.unit_price) }}
                </td>
                <td class="px-3 py-2.5 text-center font-mono">
                  <span
                    :class="item.qty > item.available_stock ? 'bg-rose-500/20 text-rose-300 border-rose-500/40' : 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'"
                    class="px-2 py-0.5 rounded border font-bold"
                  >
                    {{ item.available_stock }} pcs
                  </span>
                </td>
                <td class="px-3 py-2.5 text-center">
                  <div class="inline-flex items-center gap-1.5">
                    <button
                      type="button"
                      @click="adjustItemQty(idx, -1)"
                      class="w-6 h-6 rounded bg-slate-800 hover:bg-slate-700 text-white font-bold cursor-pointer"
                    >
                      -
                    </button>
                    <span class="font-mono font-bold w-6 text-center">{{ item.qty }}</span>
                    <button
                      type="button"
                      @click="adjustItemQty(idx, 1)"
                      class="w-6 h-6 rounded bg-slate-800 hover:bg-slate-700 text-white font-bold cursor-pointer"
                    >
                      +
                    </button>
                  </div>
                </td>
                <td class="px-3 py-2.5 text-right font-mono font-semibold text-white">
                  {{ formatIDR(item.qty * item.unit_price) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="flex items-center justify-end gap-2.5 pt-2">
          <button
            type="button"
            @click="closeModal"
            class="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold cursor-pointer"
          >
            Batal
          </button>
          <button
            type="button"
            :disabled="isSubmitting || parsedDraft.items.length === 0"
            @click="handleCreateOrder"
            class="px-4 py-2 rounded-xl bg-indigo-500 hover:bg-indigo-400 disabled:opacity-50 text-white font-bold text-xs transition cursor-pointer"
          >
            {{ isSubmitting ? 'Menyimpan & Mengunci Stok...' : 'Buat Order & Kunci Stok (PostgreSQL Tx)' }}
          </button>
        </div>
      </div>
    </div>
  </dialog>
</template>

