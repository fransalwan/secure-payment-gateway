<script setup lang="ts">
import { useOmsStore } from '../stores/omsStore'
import type { Order } from '../types/oms'

const emit = defineEmits<{
  (e: 'openAiParser'): void
  (e: 'printWaybill', order: Order): void
}>()

const store = useOmsStore()

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

function setChannel(val: string) {
  store.channelFilter = val
  store.fetchOverview()
}

function setStatus(val: string) {
  store.statusFilter = val
  store.fetchOverview()
}

function channelBadgeClass(ch: string) {
  switch (ch) {
    case 'WHATSAPP_LIVE':
      return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
    case 'SHOPEE':
      return 'bg-orange-500/15 text-orange-300 border-orange-500/30'
    case 'TIKTOK_SHOP':
      return 'bg-pink-500/15 text-pink-300 border-pink-500/30'
    default:
      return 'bg-sky-500/15 text-sky-300 border-sky-500/30'
  }
}

function statusBadgeClass(st: string) {
  switch (st) {
    case 'READY_TO_SHIP':
      return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40'
    case 'PAID':
      return 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40'
    case 'PENDING_PAYMENT':
      return 'bg-amber-500/20 text-amber-300 border-amber-500/40'
    case 'CANCELLED':
      return 'bg-rose-500/15 text-rose-300 border-rose-500/30'
    default:
      return 'bg-slate-800 text-slate-300 border-slate-700'
  }
}

function formatIDR(val: number) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val)
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-slate-900/80 border border-slate-800 overflow-hidden">
    <!-- Header -->
    <div class="p-4 sm:p-5 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3">
      <div>
        <div class="flex items-center gap-2">
          <span class="px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-400 font-mono text-[10px] font-bold">
            PANEL 1
          </span>
          <h2 id="unified-order-inbox-caption" class="text-sm sm:text-base font-bold text-white">
            Unified Omni-Channel Order Inbox
          </h2>
        </div>
        <p class="text-xs text-slate-400 mt-0.5">
          Konsolidasi order Web, Shopee, TikTok Shop &amp; WhatsApp Live dengan state machine satu arah
        </p>
      </div>

      <button
        type="button"
        @click="emit('openAiParser')"
        class="px-3.5 py-2 rounded-xl bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 font-extrabold text-xs shadow-lg shadow-emerald-500/15 transition cursor-pointer flex items-center gap-2"
      >
        <span>+ Paste WA / Live Chat (AI Parse)</span>
      </button>
    </div>

    <!-- Filters Toolbar -->
    <div class="px-4 sm:px-5 py-2.5 bg-slate-950/60 border-b border-slate-800/80 flex flex-wrap items-center justify-between gap-2 text-xs">
      <div class="flex flex-wrap items-center gap-1.5">
        <button
          v-for="ch in channels"
          :key="ch.value"
          type="button"
          @click="setChannel(ch.value)"
          :class="
            store.channelFilter === ch.value
              ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40 font-bold'
              : 'bg-slate-900 text-slate-400 border-slate-800 hover:text-slate-200'
          "
          class="px-2.5 py-1 rounded-lg border text-[11px] transition cursor-pointer"
        >
          {{ ch.label }}
        </button>
      </div>

      <div class="flex items-center gap-2">
        <label for="order-status-filter" class="text-[11px] text-slate-400">Status:</label>
        <select
          id="order-status-filter"
          :value="store.statusFilter"
          @change="setStatus(($event.target as HTMLSelectElement).value)"
          class="rounded-lg bg-slate-900 border border-slate-700 px-2.5 py-1 text-[11px] text-slate-200"
        >
          <option v-for="st in statuses" :key="st.value" :value="st.value">{{ st.label }}</option>
        </select>
      </div>
    </div>

    <!-- Accessible Data Table -->
    <div
      tabindex="0"
      role="region"
      aria-labelledby="unified-order-inbox-caption"
      class="table-wrapper max-h-[420px] custom-scroll"
    >
      <table class="w-full text-xs text-left">
        <thead class="bg-slate-950 text-slate-400 uppercase text-[10px] border-b border-slate-800">
          <tr>
            <th scope="col" class="px-4 py-3 bg-slate-950">No. Order &amp; Kanal</th>
            <th scope="col" class="px-4 py-3 bg-slate-950">Pembeli &amp; Kurir</th>
            <th scope="col" class="px-4 py-3 bg-slate-950">Item SKU</th>
            <th scope="col" class="px-4 py-3 bg-slate-950 text-right">Total</th>
            <th scope="col" class="px-4 py-3 bg-slate-950">Status &amp; Resi (Waybill)</th>
            <th scope="col" class="px-4 py-3 bg-slate-950 text-right">Aksi Operasional</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/80">
          <tr v-if="store.orders.length === 0">
            <td colspan="6" class="px-4 py-8 text-center text-slate-500">
              Belum ada pesanan untuk filter ini.
            </td>
          </tr>
          <tr
            v-for="ord in store.orders"
            :key="ord.id"
            class="hover:bg-slate-800/35 transition"
          >
            <td class="px-4 py-3 align-top">
              <div class="font-mono font-bold text-white">{{ ord.order_number }}</div>
              <span
                :class="channelBadgeClass(ord.channel)"
                class="inline-block mt-1 px-2 py-0.5 rounded border text-[10px] font-semibold"
              >
                {{ ord.channel }}
              </span>
            </td>
            <td class="px-4 py-3 align-top">
              <div class="font-semibold text-slate-200">{{ ord.customer_name }}</div>
              <div class="font-mono text-[11px] text-slate-400">{{ ord.customer_phone }}</div>
              <div class="text-[11px] text-indigo-300 font-semibold mt-0.5">
                Kurir: {{ ord.courier_code }}
              </div>
            </td>
            <td class="px-4 py-3 align-top">
              <div
                v-for="it in ord.items"
                :key="it.id"
                class="text-[11px] text-slate-300 leading-relaxed"
              >
                <span class="font-mono text-emerald-400 font-bold">{{ it.qty }}x</span>
                {{ it.product_name }}
              </div>
            </td>
            <td class="px-4 py-3 align-top text-right font-mono font-bold text-white">
              {{ formatIDR(ord.total_amount) }}
            </td>
            <td class="px-4 py-3 align-top">
              <span
                :class="statusBadgeClass(ord.status)"
                class="inline-block px-2 py-0.5 rounded border text-[10px] font-bold"
              >
                {{ ord.status }}
              </span>
              <div v-if="ord.waybill_number" class="mt-1 font-mono text-[11px] text-emerald-400 font-semibold">
                AWB: {{ ord.waybill_number }}
              </div>
              <div v-else-if="ord.status === 'PAID'" class="mt-1 text-[10px] text-indigo-300 animate-pulse">
                Worker sedang request AWB...
              </div>
            </td>
            <td class="px-4 py-3 align-top text-right">
              <div class="inline-flex flex-wrap justify-end gap-1.5">
                <button
                  v-if="ord.status === 'PENDING_PAYMENT'"
                  type="button"
                  @click="store.simulatePayOrder(ord.id)"
                  class="px-2.5 py-1 rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white font-bold text-[11px] cursor-pointer"
                >
                  Bayar (Webhook)
                </button>
                <button
                  v-if="ord.status === 'PENDING_PAYMENT'"
                  type="button"
                  @click="store.cancelOrder(ord.id)"
                  class="px-2 py-1 rounded-lg bg-slate-800 hover:bg-rose-950 hover:text-rose-300 border border-slate-700 text-slate-300 text-[11px] cursor-pointer"
                >
                  Batal
                </button>
                <button
                  v-if="ord.waybill_number"
                  type="button"
                  @click="emit('printWaybill', ord)"
                  class="px-2.5 py-1 rounded-lg bg-emerald-500/20 hover:bg-emerald-500/30 border border-emerald-500/40 text-emerald-300 font-bold text-[11px] cursor-pointer"
                >
                  Print Waybill A6
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

