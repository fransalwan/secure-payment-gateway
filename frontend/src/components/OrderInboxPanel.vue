<script setup lang="ts">
import { useClipboard } from '@vueuse/core'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { formatIDR, runViewTransition } from '../lib/utils'
import type { Order } from '../types/oms'
import {
  Sparkles,
  Printer,
  CreditCard,
  XCircle,
  Eye,
  Copy,
  Check,
  Search,
} from '@lucide/vue'
import { ref, computed } from 'vue'

const emit = defineEmits<{
  (e: 'openAiParser'): void
  (e: 'printWaybill', order: Order): void
  (e: 'inspectOrder', order: Order): void
}>()

const store = useOmsStore()
const { copy } = useClipboard()
const copiedAwb = ref<string>('')
const searchQuery = ref('')

const channels = [
  { value: '', label: 'Semua Kanal' },
  { value: 'WHATSAPP_LIVE', label: 'WhatsApp / Live' },
  { value: 'SHOPEE', label: 'Shopee' },
  { value: 'TIKTOK_SHOP', label: 'TikTok Shop' },
  { value: 'WEB', label: 'Web Checkout' },
]

const statuses = [
  { value: '', label: 'Semua Status' },
  { value: 'PENDING_PAYMENT', label: 'Pending Payment' },
  { value: 'PAID', label: 'Paid (Processing AWB)' },
  { value: 'READY_TO_SHIP', label: 'Ready to Ship' },
  { value: 'CANCELLED', label: 'Cancelled' },
]

const visibleOrders = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return store.orders
  return store.orders.filter(
    (o) =>
      o.order_number.toLowerCase().includes(q) ||
      o.customer_name.toLowerCase().includes(q) ||
      (o.waybill_number && o.waybill_number.toLowerCase().includes(q)),
  )
})

function setChannel(val: string) {
  runViewTransition(() => {
    store.channelFilter = val
    store.fetchOverview()
  })
}

function setStatus(val: string) {
  runViewTransition(() => {
    store.statusFilter = val
    store.fetchOverview()
  })
}

function channelVariant(ch: string): 'emerald' | 'orange' | 'pink' | 'sky' {
  switch (ch) {
    case 'WHATSAPP_LIVE':
      return 'emerald'
    case 'SHOPEE':
      return 'orange'
    case 'TIKTOK_SHOP':
      return 'pink'
    default:
      return 'sky'
  }
}

function statusVariant(st: string): 'emerald' | 'indigo' | 'amber' | 'rose' | 'default' {
  switch (st) {
    case 'READY_TO_SHIP':
      return 'emerald'
    case 'PAID':
      return 'indigo'
    case 'PENDING_PAYMENT':
      return 'amber'
    case 'CANCELLED':
      return 'rose'
    default:
      return 'default'
  }
}

async function copyAwb(awb: string) {
  await copy(awb)
  copiedAwb.value = awb
  setTimeout(() => {
    if (copiedAwb.value === awb) copiedAwb.value = ''
  }, 1800)
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-card border border-border shadow-xs overflow-hidden">
    <!-- Header -->
    <div class="p-4 sm:p-5 border-b border-border flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div>
        <div class="flex items-center gap-2">
          <Badge variant="emerald">OMS INBOX</Badge>
          <h2 id="unified-order-inbox-caption" tabindex="-1" class="text-sm sm:text-base font-bold text-foreground focus:outline-none">
            Unified Omni-Channel Order Inbox
          </h2>
        </div>
        <p class="text-xs text-muted-foreground mt-0.5">
          Konsolidasi order Web, Shopee, TikTok Shop &amp; WhatsApp Live dengan state machine satu arah
        </p>
      </div>

      <Button variant="emerald" class="w-full sm:w-auto" @click="emit('openAiParser')">
        <Sparkles class="w-3.5 h-3.5" />
        <span>+ AI WhatsApp Order Capture</span>
      </Button>
    </div>

    <!-- Mobile-First Swipeable Filter Chips & Search Toolbar -->
    <div class="px-4 sm:px-5 py-3 bg-elevated/50 border-b border-border flex flex-col lg:flex-row lg:items-center justify-between gap-2.5 text-xs">
      <!-- Horizontal scrollable chips on mobile -->
      <div class="flex items-center gap-1.5 overflow-x-auto no-scrollbar pb-0.5">
        <button
          v-for="ch in channels"
          :key="ch.value"
          type="button"
          @click="setChannel(ch.value)"
          :class="
            store.channelFilter === ch.value
              ? 'bg-foreground text-background font-bold border-foreground shadow-2xs'
              : 'bg-card text-muted-foreground border-border hover:text-foreground hover:bg-muted'
          "
          class="shrink-0 px-3 py-1.5 rounded-xl border text-[11px] transition cursor-pointer"
        >
          {{ ch.label }}
        </button>
      </div>

      <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
        <div class="relative flex-1 sm:w-52">
          <Search class="w-3.5 h-3.5 text-muted-foreground absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            v-model="searchQuery"
            type="search"
            aria-label="Cari nomor order, pembeli, atau AWB"
            placeholder="Cari order / pembeli / AWB..."
            class="w-full rounded-xl bg-card border border-border pl-8 pr-3 py-1.5 text-[11px] text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        <select
          id="order-status-filter"
          aria-label="Filter status order"
          :value="store.statusFilter"
          @change="setStatus(($event.target as HTMLSelectElement).value)"
          class="rounded-xl bg-card border border-border px-3 py-1.5 text-[11px] font-medium text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
        >
          <option v-for="st in statuses" :key="st.value" :value="st.value">{{ st.label }}</option>
        </select>
      </div>
    </div>

    <!-- Container-Query Responsive Data Table (Table >= 640px -> Stacked Seller Cards < 640px) -->
    <div
      tabindex="0"
      role="region"
      aria-labelledby="unified-order-inbox-caption"
      class="table-wrapper max-h-[520px] custom-scroll"
    >
      <table
        class="responsive-data-table text-xs text-left"
        style="--label-2: 'Pembeli & Kurir'; --label-3: 'Rincian SKU'; --label-4: 'Total'; --label-5: 'Status & Resi'"
      >
        <thead class="bg-elevated text-muted-foreground uppercase text-[10px] border-b border-border">
          <tr>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold">No. Order &amp; Kanal</th>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold">Pembeli &amp; Kurir</th>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold">Item SKU</th>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold text-right">Total</th>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold">Status &amp; Resi (Waybill)</th>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold text-right">Aksi Operasional</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="visibleOrders.length === 0">
            <td colspan="6" class="px-4 py-10 text-center text-muted-foreground">
              Tidak ada pesanan yang cocok dengan filter saat ini.
            </td>
          </tr>
          <tr
            v-for="ord in visibleOrders"
            :key="ord.id"
            class="data-row hover:bg-elevated/40 transition"
          >
            <!-- Sticky Row Header on Desktop & Card Header on Mobile -->
            <th scope="row" class="px-4 py-3 align-top text-left font-normal">
              <div class="flex items-center justify-between gap-2">
                <button
                  type="button"
                  @click="emit('inspectOrder', ord)"
                  class="font-mono font-bold text-foreground hover:text-tone-emerald underline-offset-4 hover:underline cursor-pointer text-left"
                >
                  {{ ord.order_number }}
                </button>
                <Badge :variant="channelVariant(ord.channel)">{{ ord.channel }}</Badge>
              </div>
            </th>

            <td class="px-4 py-3 align-top">
              <div>
                <div class="font-semibold text-foreground">{{ ord.customer_name }}</div>
                <div class="font-mono text-[11px] text-muted-foreground">{{ ord.customer_phone }}</div>
                <div class="text-[11px] text-tone-indigo font-semibold mt-0.5">
                  {{ ord.courier_code }} REG
                </div>
              </div>
            </td>

            <td class="px-4 py-3 align-top">
              <div class="space-y-1">
                <div
                  v-for="item in ord.items"
                  :key="item.id"
                  class="text-[11px] text-secondary-foreground"
                >
                  <span class="font-mono text-muted-foreground">{{ item.sku }}</span>
                  <span class="mx-1">&times;</span>
                  <strong class="font-mono text-foreground">{{ item.qty }}</strong>
                </div>
              </div>
            </td>

            <td class="px-4 py-3 align-top text-right font-mono font-bold text-foreground tabular-nums">
              {{ formatIDR(ord.total_amount) }}
            </td>

            <td class="px-4 py-3 align-top">
              <div class="flex flex-col items-end sm:items-start gap-1">
                <Badge :variant="statusVariant(ord.status)">{{ ord.status }}</Badge>
                <div v-if="ord.waybill_number" class="inline-flex items-center gap-1 font-mono text-[11px] text-tone-emerald font-semibold">
                  <span>{{ ord.waybill_number }}</span>
                  <button
                    type="button"
                    aria-label="Salin nomor resi"
                    @click="copyAwb(ord.waybill_number)"
                    class="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground cursor-pointer"
                  >
                    <Check v-if="copiedAwb === ord.waybill_number" class="w-3 h-3 text-tone-emerald" />
                    <Copy v-else class="w-3 h-3" />
                  </button>
                </div>
                <div v-else-if="ord.status === 'PAID'" class="text-[11px] text-tone-indigo animate-pulse">
                  Worker menerbitkan AWB...
                </div>
              </div>
            </td>

            <td class="px-4 py-3 align-top text-right">
              <div class="flex flex-wrap sm:inline-flex items-center justify-end gap-1.5 w-full sm:w-auto">
                <Button
                  v-if="ord.status === 'PENDING_PAYMENT'"
                  variant="indigo"
                  size="sm"
                  class="flex-1 sm:flex-none"
                  @click="store.simulatePayOrder(ord.id)"
                >
                  <CreditCard class="w-3.5 h-3.5" />
                  <span>Bayar (Webhook)</span>
                </Button>
                <Button
                  v-if="ord.status === 'PENDING_PAYMENT'"
                  variant="destructive"
                  size="sm"
                  class="flex-1 sm:flex-none"
                  @click="store.cancelOrder(ord.id)"
                >
                  <XCircle class="w-3.5 h-3.5" />
                  <span>Batal</span>
                </Button>
                <Button
                  v-if="ord.status === 'READY_TO_SHIP' || ord.waybill_number"
                  variant="emerald"
                  size="sm"
                  class="flex-1 sm:flex-none"
                  @click="emit('printWaybill', ord)"
                >
                  <Printer class="w-3.5 h-3.5" />
                  <span>Print Waybill A6</span>
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  aria-label="Lihat detail order"
                  @click="emit('inspectOrder', ord)"
                >
                  <Eye class="w-3.5 h-3.5" />
                  <span class="sm:hidden">Detail</span>
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
