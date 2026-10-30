<script setup lang="ts">
import { ref, computed } from 'vue'
import ResponsiveDialog from './ui/ResponsiveDialog.vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { formatIDR } from '../lib/utils'
import type { AIParseResponse } from '../types/oms'
import { Sparkles, AlertTriangle, CheckCircle2, Minus, Plus } from '@lucide/vue'

const store = useOmsStore()
const isOpen = ref(false)

const samplePresets = [
  {
    label: 'Preset 1 (Stok Kurang: 2 vs 1)',
    text: 'Kak mau pesan Kaos Oversize Hitam XL 2 pcs sama Topi Baseball Navy 1 pcs, kirim ke Budi Santoso 081299887766, Jl. Kaliurang Km 5 No 12 Sleman via JNE REG ya',
  },
  {
    label: 'Preset 2 (Valid: Hoodie + Totebag)',
    text: 'Min order Hoodie Fleece Misty Grey L 1 pcs dan Totebag Canvas Waterproof Hitam 2 pcs, atas nama Siti Rahmawati 085788991122, Jl. Dago No 84 Coblong Bandung pakai SICEPAT ya',
  },
  {
    label: 'Preset 3 (Live: Sneakers via J&T)',
    text: 'Fix ambil Sneakers Vulcanized Classic White 42 1 pasang + Topi Baseball Navy 1 pcs kirim ke Andi Pratama 081377665544, Jl. Sudirman No 19 Jakarta Selatan via J&T',
  },
]

const rawChat = ref<string>(samplePresets[0]!.text)
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
  isOpen.value = true
}

function closeModal() {
  isOpen.value = false
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

defineExpose({ openModal })
</script>

<template>
  <ResponsiveDialog
    v-model:open="isOpen"
    title="AI WhatsApp & Live Selling Order Capture"
    description="Ekstraksi chat tidak terstruktur ke katalog SKU PostgreSQL (< 3 detik SLA + Fallback NLP)"
    max-width-class="max-w-3xl"
  >
    <div class="space-y-4">
      <!-- Quick Chat Presets (Horizontal Scrollable on Mobile) -->
      <div class="space-y-1.5">
        <label class="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
          Preset Contoh Chat (Klik 1x):
        </label>
        <div class="flex gap-2 overflow-x-auto no-scrollbar pb-1">
          <button
            v-for="(preset, idx) in samplePresets"
            :key="idx"
            type="button"
            @click="applyPreset(preset.text)"
            class="shrink-0 px-3 py-1.5 rounded-xl bg-elevated hover:bg-muted border border-border text-[11px] font-medium text-foreground transition cursor-pointer"
          >
            {{ preset.label }}
          </button>
        </div>
      </div>

      <!-- Raw Textarea -->
      <div class="space-y-2">
        <label for="wa-raw-input" class="block text-xs font-semibold text-foreground">
          Teks Chat Pembeli (WhatsApp / TikTok Live):
        </label>
        <textarea
          id="wa-raw-input"
          v-model="rawChat"
          rows="3"
          class="w-full rounded-xl bg-elevated border border-border focus:ring-2 focus:ring-ring focus:outline-none p-3 text-xs text-foreground leading-relaxed"
          placeholder="Paste chat order WhatsApp / komentar TikTok Live di sini..."
        ></textarea>
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
          <span class="text-[11px] text-muted-foreground">
            Katalog aktif: <strong class="text-foreground">{{ store.activeMerchant?.name }}</strong>
          </span>
          <Button
            variant="emerald"
            :disabled="isParsing || !rawChat.trim()"
            class="w-full sm:w-auto"
            @click="handleExtract"
          >
            <Sparkles class="w-3.5 h-3.5" />
            <span>{{ isParsing ? 'Mengekstrak dengan AI...' : 'Extract with AI' }}</span>
          </Button>
        </div>
      </div>

      <!-- Extracted Draft Preview -->
      <div v-if="parsedDraft" class="space-y-4 pt-3 border-t border-border">
        <div class="flex flex-wrap items-center justify-between gap-2 bg-elevated px-3.5 py-2.5 rounded-xl border border-border text-xs">
          <div class="flex flex-wrap items-center gap-1.5">
            <Badge variant="indigo">Engine: {{ parsedDraft.parser_engine }}</Badge>
            <Badge variant="emerald">Latency: {{ parsedDraft.latency_ms }}ms (&lt; 3000ms)</Badge>
          </div>
          <span class="font-bold text-tone-emerald font-mono tabular-nums">
            Estimasi Total: {{ formatIDR(computedTotal) }}
          </span>
        </div>

        <!-- RED STOCK INSUFFICIENT WARNING BANNER -->
        <div
          v-if="hasStockShortage"
          role="alert"
          class="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/40 text-xs flex items-start gap-3"
        >
          <AlertTriangle class="w-4 h-4 text-tone-rose shrink-0 mt-0.5" />
          <div>
            <p class="font-bold text-tone-rose">
              Peringatan Stok Kurang Sebelum Order Disimpan!
            </p>
            <p class="text-secondary-foreground mt-0.5 leading-relaxed">
              Terdapat SKU dengan <code>available_stock</code> lebih kecil dari permintaan pembeli. Sesuaikan tombol <code>- / +</code> di bawah agar lolos PostgreSQL Row-Level Lock.
            </p>
          </div>
        </div>

        <!-- Editable Customer & Shipping Fields -->
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
          <div>
            <label class="block text-muted-foreground font-medium mb-1">Nama Penerima</label>
            <input
              v-model="parsedDraft.customer_name"
              type="text"
              class="w-full rounded-xl bg-elevated border border-border px-3 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div>
            <label class="block text-muted-foreground font-medium mb-1">No. WhatsApp</label>
            <input
              v-model="parsedDraft.customer_phone"
              type="text"
              class="w-full rounded-xl bg-elevated border border-border px-3 py-2 text-foreground font-mono focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div>
            <label class="block text-muted-foreground font-medium mb-1">Kurir Ekspedisi</label>
            <select
              v-model="parsedDraft.courier_code"
              class="w-full rounded-xl bg-elevated border border-border px-3 py-2 text-foreground font-semibold focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option value="JNE">JNE REG</option>
              <option value="JNT">J&amp;T EZ</option>
              <option value="SICEPAT">SiCepat REG</option>
            </select>
          </div>
          <div class="sm:col-span-3">
            <label class="block text-muted-foreground font-medium mb-1">Alamat Pengiriman Lengkap</label>
            <input
              v-model="parsedDraft.shipping_address"
              type="text"
              class="w-full rounded-xl bg-elevated border border-border px-3 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
        </div>

        <!-- Matched SKU Cards / Rows (Mobile-First Friendly) -->
        <div class="space-y-2">
          <div
            v-for="(item, idx) in parsedDraft.items"
            :key="item.sku_id"
            :class="item.qty > item.available_stock ? 'border-rose-500/45 bg-rose-500/6' : 'border-border bg-card'"
            class="rounded-xl border p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs"
          >
            <div class="min-w-0">
              <div class="font-semibold text-foreground">{{ item.matched_product_name }}</div>
              <div class="font-mono text-[11px] text-muted-foreground mt-0.5">
                {{ item.sku }} &bull; {{ formatIDR(item.unit_price) }} &bull;
                <span :class="item.qty > item.available_stock ? 'text-tone-rose font-bold' : 'text-tone-emerald font-semibold'">
                  Tersedia: {{ item.available_stock }} pcs
                </span>
              </div>
              <div v-if="item.qty > item.available_stock" class="text-[11px] font-semibold text-tone-rose mt-1">
                ⚠ {{ item.stock_warning || `Stok tersedia hanya ${item.available_stock} pcs` }}
              </div>
            </div>

            <div class="flex items-center justify-between sm:justify-end gap-4 shrink-0">
              <div class="inline-flex items-center gap-2 rounded-xl border border-border bg-elevated p-1">
                <button
                  type="button"
                  aria-label="Kurangi jumlah"
                  @click="adjustItemQty(idx, -1)"
                  class="h-8 w-8 rounded-lg bg-card hover:bg-muted border border-border flex items-center justify-center text-foreground cursor-pointer"
                >
                  <Minus class="w-3.5 h-3.5" />
                </button>
                <span class="font-mono font-bold w-6 text-center tabular-nums">{{ item.qty }}</span>
                <button
                  type="button"
                  aria-label="Tambah jumlah"
                  @click="adjustItemQty(idx, 1)"
                  class="h-8 w-8 rounded-lg bg-card hover:bg-muted border border-border flex items-center justify-center text-foreground cursor-pointer"
                >
                  <Plus class="w-3.5 h-3.5" />
                </button>
              </div>
              <div class="font-mono font-bold text-foreground tabular-nums min-w-[92px] text-right">
                {{ formatIDR(item.qty * item.unit_price) }}
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-col-reverse sm:flex-row items-stretch sm:items-center justify-end gap-2.5 pt-2">
          <Button variant="secondary" @click="closeModal">Batal</Button>
          <Button
            variant="indigo"
            size="lg"
            :disabled="isSubmitting || parsedDraft.items.length === 0"
            @click="handleCreateOrder"
          >
            <CheckCircle2 class="w-4 h-4" />
            <span>{{ isSubmitting ? 'Menyimpan & Mengunci Stok...' : 'Buat Order & Kunci Stok (PostgreSQL Tx)' }}</span>
          </Button>
        </div>
      </div>
    </div>
  </ResponsiveDialog>
</template>
