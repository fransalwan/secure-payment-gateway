<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useOmsStore } from './stores/omsStore'
import OrderInboxPanel from './components/OrderInboxPanel.vue'
import InventoryMonitorPanel from './components/InventoryMonitorPanel.vue'
import StressSimulatorPanel from './components/StressSimulatorPanel.vue'
import QueueMonitorPanel from './components/QueueMonitorPanel.vue'
import AiOrderParserModal from './components/AiOrderParserModal.vue'
import WaybillPrintModal from './components/WaybillPrintModal.vue'
import CorePaymentLedgerView from './components/CorePaymentLedgerView.vue'
import type { Order } from './types/oms'

const store = useOmsStore()
const activeWorkspace = ref<'oms' | 'ledger'>('oms')
const aiModalRef = ref<InstanceType<typeof AiOrderParserModal> | null>(null)
const waybillModalRef = ref<InstanceType<typeof WaybillPrintModal> | null>(null)

let pollTimer: ReturnType<typeof setInterval> | null = null

function openAiModal() {
  aiModalRef.value?.openModal()
}

function openWaybillModal(order: Order) {
  waybillModalRef.value?.openWaybill(order)
}

function onMerchantSwitch(merchantId: string) {
  store.activeMerchantId = merchantId
  store.fetchOverview()
}

function formatIDR(val: number) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val)
}

onMounted(() => {
  store.fetchOverview()
  pollTimer = setInterval(() => {
    if (activeWorkspace.value === 'oms') {
      store.fetchOverview(true)
    }
  }, 2500)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-slate-950 text-slate-100">
    <!-- Toast Stack -->
    <div class="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm pointer-events-none">
      <div
        v-for="t in store.toasts"
        :key="t.id"
        :class="{
          'border-emerald-500/50 bg-slate-900/95 text-emerald-100': t.type === 'success',
          'border-rose-500/50 bg-slate-900/95 text-rose-100': t.type === 'error',
          'border-amber-500/50 bg-slate-900/95 text-amber-100': t.type === 'warning',
          'border-indigo-500/50 bg-slate-900/95 text-indigo-100': t.type === 'info',
        }"
        class="p-3.5 rounded-xl border shadow-2xl backdrop-blur pointer-events-auto text-xs"
      >
        <div class="font-bold">{{ t.title }}</div>
        <div v-if="t.detail" class="text-[11px] text-slate-300 mt-0.5">{{ t.detail }}</div>
      </div>
    </div>

    <!-- Top Operational Navbar -->
    <header class="sticky top-0 z-30 border-b border-slate-800 bg-slate-900/90 backdrop-blur">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3 flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-emerald-500 to-teal-400 flex items-center justify-center text-slate-950 font-black text-sm shadow-lg shadow-emerald-500/20">
            OMS
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h1 class="text-base sm:text-lg font-extrabold tracking-tight text-white">
                Omni-Channel E-Commerce OMS &amp; Payment Gateway
              </h1>
              <span class="hidden sm:inline-block px-2 py-0.5 rounded-full bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-mono text-[10px] font-bold">
                Vue 3 + Go + PG16 + Redis
              </span>
            </div>
            <p class="text-xs text-slate-400">
              AI Social Commerce Capture &bull; Row-Level Stock Lock &bull; Idempotent Webhook &bull; Waybill Queue
            </p>
          </div>
        </div>

        <!-- Multi-Merchant Selector & Telemetry -->
        <div class="flex flex-wrap items-center gap-2.5 text-xs">
          <div class="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-950 border border-slate-800">
            <span class="text-slate-400 text-[11px]">Merchant:</span>
            <select
              aria-label="Pilih tenant merchant aktif"
              :value="store.activeMerchantId"
              @change="onMerchantSwitch(($event.target as HTMLSelectElement).value)"
              class="bg-transparent text-indigo-300 font-bold focus:outline-none cursor-pointer"
            >
              <option
                v-for="m in store.merchants"
                :key="m.id"
                :value="m.id"
                class="bg-slate-900 text-slate-100"
              >
                {{ m.name }} ({{ m.code }})
              </option>
            </select>
          </div>

          <div class="hidden md:flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-950 border border-slate-800 font-mono text-[11px]">
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
            <span class="text-emerald-400 font-bold">{{ store.lastApiLatencyMs }}ms</span>
            <span class="text-slate-600">|</span>
            <span class="text-slate-400">{{ store.queueMetrics.queue_backend }}</span>
          </div>

          <button
            type="button"
            @click="store.fetchOverview()"
            class="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-700 font-semibold cursor-pointer"
          >
            Sync
          </button>
        </div>
      </div>

      <!-- 3-Minute Interview Demo Presenter Shortcut Bar -->
      <div class="bg-slate-950/90 border-t border-slate-800/80 px-4 sm:px-6 lg:px-8 py-2">
        <div class="max-w-7xl mx-auto flex flex-wrap items-center justify-between gap-2 text-xs">
          <div class="flex items-center gap-2">
            <span class="font-extrabold text-amber-400 uppercase tracking-wider text-[11px]">
              3-Min Live Demo Flow:
            </span>
            <button
              type="button"
              @click="activeWorkspace = 'oms'; openAiModal()"
              class="px-2.5 py-1 rounded-lg bg-emerald-500/15 hover:bg-emerald-500/25 border border-emerald-500/30 text-emerald-300 font-semibold text-[11px] cursor-pointer"
            >
              1. AI WhatsApp Order Capture
            </button>
            <button
              type="button"
              @click="activeWorkspace = 'oms'; store.runRaceConditionSim(store.products[0]?.id || '', 10, 1)"
              class="px-2.5 py-1 rounded-lg bg-amber-500/15 hover:bg-amber-500/25 border border-amber-500/30 text-amber-300 font-semibold text-[11px] cursor-pointer"
            >
              2. 10 Buyers vs 1 Last Stock
            </button>
            <button
              type="button"
              @click="activeWorkspace = 'oms'; store.runDuplicateWebhookSim('', 5)"
              class="px-2.5 py-1 rounded-lg bg-indigo-500/15 hover:bg-indigo-500/25 border border-indigo-500/30 text-indigo-300 font-semibold text-[11px] cursor-pointer"
            >
              3. 5x Duplicate Payment Webhook
            </button>
          </div>

          <!-- Workspace Switcher -->
          <div class="inline-flex rounded-xl bg-slate-900 p-0.5 border border-slate-800">
            <button
              type="button"
              @click="activeWorkspace = 'oms'"
              :class="activeWorkspace === 'oms' ? 'bg-emerald-500 text-slate-950 font-extrabold' : 'text-slate-400 hover:text-white'"
              class="px-3 py-1 rounded-lg text-[11px] transition cursor-pointer"
            >
              Omni-Channel OMS (4 Panels)
            </button>
            <button
              type="button"
              @click="activeWorkspace = 'ledger'"
              :class="activeWorkspace === 'ledger' ? 'bg-indigo-500 text-white font-extrabold' : 'text-slate-400 hover:text-white'"
              class="px-3 py-1 rounded-lg text-[11px] transition cursor-pointer"
            >
              Core Payment Switch &amp; Ledger
            </button>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
      <template v-if="activeWorkspace === 'oms'">
        <!-- KPI Summary Strip -->
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-3.5">
          <div class="rounded-2xl bg-slate-900/80 border border-slate-800 p-4">
            <div class="text-[11px] text-slate-400 font-medium">Total Unified Orders</div>
            <div class="text-xl font-black font-mono text-white mt-1">{{ store.orders.length }}</div>
            <div class="text-[10px] text-emerald-400 mt-0.5">Web, Shopee, TikTok &amp; WA Live</div>
          </div>
          <div class="rounded-2xl bg-slate-900/80 border border-slate-800 p-4">
            <div class="text-[11px] text-slate-400 font-medium">Active Gross Merchandise Value</div>
            <div class="text-xl font-black font-mono text-emerald-400 mt-1">{{ formatIDR(store.totalGMV) }}</div>
            <div class="text-[10px] text-slate-400 mt-0.5">Multi-tenant isolated ledger</div>
          </div>
          <div class="rounded-2xl bg-slate-900/80 border border-slate-800 p-4">
            <div class="text-[11px] text-slate-400 font-medium">Low-Stock Flash SKUs (&le; 2 pcs)</div>
            <div class="text-xl font-black font-mono text-amber-400 mt-1">{{ store.lowStockCount }} SKU</div>
            <div class="text-[10px] text-slate-400 mt-0.5">Protected by SELECT FOR UPDATE</div>
          </div>
          <div class="rounded-2xl bg-slate-900/80 border border-slate-800 p-4">
            <div class="text-[11px] text-slate-400 font-medium">Background Worker Jobs</div>
            <div class="text-xl font-black font-mono text-indigo-300 mt-1">
              {{ store.queueMetrics.completed_count }} Done / {{ store.queueMetrics.failed_dlq_count }} DLQ
            </div>
            <div class="text-[10px] text-slate-400 mt-0.5">Waybill AWB + Marketplace Sync</div>
          </div>
        </div>

        <!-- Panel 1 & Panel 2 -->
        <div class="grid grid-cols-1 xl:grid-cols-12 gap-5">
          <div class="xl:col-span-7">
            <OrderInboxPanel @open-ai-parser="openAiModal" @print-waybill="openWaybillModal" />
          </div>
          <div class="xl:col-span-5">
            <InventoryMonitorPanel />
          </div>
        </div>

        <!-- Panel 3: Concurrency & Duplicate Webhook Simulator -->
        <StressSimulatorPanel />

        <!-- Panel 4: Background Queue, Exponential Backoff & DLQ Monitor -->
        <QueueMonitorPanel />
      </template>

      <template v-else>
        <CorePaymentLedgerView />
      </template>
    </main>

    <!-- Native Accessible Modals -->
    <AiOrderParserModal ref="aiModalRef" />
    <WaybillPrintModal ref="waybillModalRef" />
  </div>
</template>

