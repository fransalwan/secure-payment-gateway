<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Toaster } from 'vue-sonner'
import { useOmsStore } from './stores/omsStore'
import { useThemeScheme } from './composables/useThemeScheme'
import { formatIDR, runViewTransition } from './lib/utils'
import Button from './components/ui/Button.vue'
import Badge from './components/ui/Badge.vue'
import OrderInboxPanel from './components/OrderInboxPanel.vue'
import InventoryMonitorPanel from './components/InventoryMonitorPanel.vue'
import StressSimulatorPanel from './components/StressSimulatorPanel.vue'
import QueueMonitorPanel from './components/QueueMonitorPanel.vue'
import AiOrderParserModal from './components/AiOrderParserModal.vue'
import WaybillPrintModal from './components/WaybillPrintModal.vue'
import CorePaymentLedgerView from './components/CorePaymentLedgerView.vue'
import CommandPaletteModal, { type NavSection } from './components/CommandPaletteModal.vue'
import OrderInspectorSheet from './components/OrderInspectorSheet.vue'
import type { Order } from './types/oms'
import {
  LayoutGrid,
  ShoppingBag,
  Boxes,
  Zap,
  Activity,
  Landmark,
  Sparkles,
  Command,
  RefreshCw,
  Sun,
  Moon,
  ShieldCheck,
  Store,
} from '@lucide/vue'

type WorkspaceView = 'overview' | NavSection

const store = useOmsStore()
const { resolvedTheme, toggleOppositeScheme } = useThemeScheme()

const activeView = ref<WorkspaceView>('overview')
const isCommandOpen = ref(false)
const inspectedOrder = ref<Order | null>(null)
const isInspectorOpen = ref(false)

const aiModalRef = ref<InstanceType<typeof AiOrderParserModal> | null>(null)
const waybillModalRef = ref<InstanceType<typeof WaybillPrintModal> | null>(null)

let pollTimer: ReturnType<typeof setInterval> | null = null

const headingIdByView: Record<WorkspaceView, string> = {
  overview: 'unified-order-inbox-caption',
  orders: 'unified-order-inbox-caption',
  inventory: 'inventory-monitor-caption',
  lab: 'stress-lab-heading',
  queues: 'queue-monitor-heading',
  ledger: 'core-ledger-heading',
}

const navItems = computed(() => [
  {
    id: 'overview' as WorkspaceView,
    label: 'Command Overview',
    shortLabel: 'Home',
    icon: LayoutGrid,
    badge: null as string | null,
    badgeVariant: 'default' as const,
  },
  {
    id: 'orders' as WorkspaceView,
    label: 'Order Inbox',
    shortLabel: 'Orders',
    icon: ShoppingBag,
    badge: String(store.orders.length),
    badgeVariant: 'emerald' as const,
  },
  {
    id: 'inventory' as WorkspaceView,
    label: '3-State Inventory',
    shortLabel: 'Stock',
    icon: Boxes,
    badge: store.lowStockCount > 0 ? `${store.lowStockCount} Low` : null,
    badgeVariant: 'amber' as const,
  },
  {
    id: 'lab' as WorkspaceView,
    label: 'Concurrency Lab',
    shortLabel: 'Lab',
    icon: Zap,
    badge: 'Sim',
    badgeVariant: 'amber' as const,
  },
  {
    id: 'queues' as WorkspaceView,
    label: 'Redis Queue & DLQ',
    shortLabel: 'Queues',
    icon: Activity,
    badge:
      store.queueMetrics.failed_dlq_count > 0
        ? `${store.queueMetrics.failed_dlq_count} DLQ`
        : null,
    badgeVariant: 'rose' as const,
  },
  {
    id: 'ledger' as WorkspaceView,
    label: 'Payment Ledger',
    shortLabel: 'Ledger',
    icon: Landmark,
    badge: null,
    badgeVariant: 'indigo' as const,
  },
])

const activeViewTitle = computed(
  () => navItems.value.find((n) => n.id === activeView.value)?.label ?? 'Command Overview',
)

function switchView(next: WorkspaceView) {
  if (activeView.value === next) return
  runViewTransition(() => {
    activeView.value = next
  }, headingIdByView[next])
}

function openAiModal() {
  aiModalRef.value?.openModal()
}

function openWaybillModal(order: Order) {
  isInspectorOpen.value = false
  waybillModalRef.value?.openWaybill(order)
}

function openOrderInspector(order: Order) {
  inspectedOrder.value = order
  isInspectorOpen.value = true
}

function onMerchantSwitch(merchantId: string) {
  runViewTransition(() => {
    store.activeMerchantId = merchantId
    store.fetchOverview()
  })
}

function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    isCommandOpen.value = !isCommandOpen.value
  }
}

onMounted(() => {
  store.fetchOverview()
  window.addEventListener('keydown', onGlobalKeydown)
  pollTimer = setInterval(() => {
    if (activeView.value !== 'ledger') {
      store.fetchOverview(true)
    }
  }, 2500)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div class="min-h-dvh flex bg-background text-foreground">
    <!-- Modern Stacked Swipeable Toast Notifications (vue-sonner) -->
    <Toaster position="top-right" :theme="resolvedTheme" rich-colors close-button />

    <!-- DESKTOP SAAS SIDEBAR (>= 1024px) -->
    <aside class="hidden lg:flex w-64 shrink-0 flex-col border-r border-border bg-sidebar sticky top-0 h-dvh">
      <!-- Brand Header -->
      <div class="p-4 border-b border-border flex items-center gap-3">
        <div class="w-9 h-9 rounded-xl bg-emerald-600 text-white flex items-center justify-center font-black text-xs shadow-sm shadow-emerald-600/25 shrink-0">
          OMS
        </div>
        <div class="min-w-0">
          <div class="text-sm font-extrabold tracking-tight text-foreground truncate">
            Omni-Channel OMS
          </div>
          <div class="text-[11px] text-muted-foreground truncate">
            Payment Switch &amp; Stock Engine
          </div>
        </div>
      </div>

      <!-- Multi-Tenant Merchant Switcher -->
      <div class="p-3 border-b border-border">
        <label for="sidebar-merchant-select" class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5 px-1">
          <Store class="w-3 h-3" />
          <span>Active Merchant Tenant</span>
        </label>
        <select
          id="sidebar-merchant-select"
          :value="store.activeMerchantId"
          @change="onMerchantSwitch(($event.target as HTMLSelectElement).value)"
          class="w-full rounded-xl bg-elevated border border-border px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-ring cursor-pointer"
        >
          <option v-for="m in store.merchants" :key="m.id" :value="m.id">
            {{ m.name }} ({{ m.code }})
          </option>
        </select>
      </div>

      <!-- Primary Workspace Navigation -->
      <nav aria-label="Menu utama aplikasi" class="flex-1 overflow-y-auto custom-scroll p-3 space-y-1">
        <div class="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          Operational Modules
        </div>
        <button
          v-for="item in navItems"
          :key="item.id"
          type="button"
          @click="switchView(item.id)"
          :class="
            activeView === item.id
              ? 'bg-elevated text-foreground font-bold border-border shadow-2xs'
              : 'text-muted-foreground hover:text-foreground hover:bg-elevated/60 border-transparent'
          "
          class="w-full flex items-center justify-between gap-2 px-3 py-2 rounded-xl border text-xs transition cursor-pointer"
        >
          <div class="flex items-center gap-2.5 min-w-0">
            <component :is="item.icon" class="w-4 h-4 shrink-0" />
            <span class="truncate">{{ item.label }}</span>
          </div>
          <Badge v-if="item.badge" :variant="item.badgeVariant" class="tabular-nums">
            {{ item.badge }}
          </Badge>
        </button>
      </nav>

      <!-- Bottom Sandbox Quick-Launch Card (Replaces Prototype Banner Cleanly) -->
      <div class="p-3 border-t border-border space-y-2 bg-elevated/35">
        <div class="flex items-center justify-between px-1">
          <span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Sandbox Quick-Run
          </span>
          <button
            type="button"
            @click="isCommandOpen = true"
            class="font-mono text-[10px] text-tone-emerald hover:underline cursor-pointer"
          >
            Ctrl+K
          </button>
        </div>

        <div class="grid grid-cols-1 gap-1.5">
          <button
            type="button"
            @click="openAiModal"
            class="flex items-center gap-2 rounded-lg border border-border bg-card hover:bg-muted px-2.5 py-1.5 text-[11px] font-semibold text-foreground transition cursor-pointer text-left"
          >
            <Sparkles class="w-3.5 h-3.5 text-tone-emerald shrink-0" />
            <span class="truncate">1. AI WhatsApp Capture</span>
          </button>
          <button
            type="button"
            @click="switchView('lab'); store.runRaceConditionSim(store.products[0]?.id || '', 10, 1)"
            class="flex items-center gap-2 rounded-lg border border-border bg-card hover:bg-muted px-2.5 py-1.5 text-[11px] font-semibold text-foreground transition cursor-pointer text-left"
          >
            <Zap class="w-3.5 h-3.5 text-tone-amber shrink-0" />
            <span class="truncate">2. 10 Buyers vs 1 Stock</span>
          </button>
          <button
            type="button"
            @click="switchView('lab'); store.runDuplicateWebhookSim('', 5)"
            class="flex items-center gap-2 rounded-lg border border-border bg-card hover:bg-muted px-2.5 py-1.5 text-[11px] font-semibold text-foreground transition cursor-pointer text-left"
          >
            <ShieldCheck class="w-3.5 h-3.5 text-tone-indigo shrink-0" />
            <span class="truncate">3. 5x Duplicate Webhook</span>
          </button>
        </div>
      </div>
    </aside>

    <!-- MAIN CONTENT SHELL -->
    <div class="flex-1 min-w-0 flex flex-col">
      <!-- Sticky Top Bar (Compact Mobile-First + Desktop Command Bar) -->
      <header class="sticky top-0 z-30 border-b border-border bg-card/85 backdrop-blur-md">
        <div class="max-w-7xl mx-auto px-3.5 sm:px-6 py-2.5 flex items-center justify-between gap-2.5">
          <!-- Left: Mobile Logo / Desktop Breadcrumb -->
          <div class="flex items-center gap-2.5 min-w-0">
            <div class="lg:hidden w-8 h-8 rounded-xl bg-emerald-600 text-white flex items-center justify-center font-black text-[11px] shrink-0">
              OMS
            </div>
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <h1 class="text-sm sm:text-base font-extrabold tracking-tight text-foreground truncate">
                  {{ activeViewTitle }}
                </h1>
                <Badge variant="emerald" class="hidden md:inline-flex font-mono">
                  Go + Vue 3 + PG16 + Redis
                </Badge>
              </div>
              <!-- Mobile Merchant Selector Inline -->
              <div class="lg:hidden flex items-center gap-1 mt-0.5">
                <select
                  aria-label="Pilih tenant merchant aktif"
                  :value="store.activeMerchantId"
                  @change="onMerchantSwitch(($event.target as HTMLSelectElement).value)"
                  class="bg-transparent text-[11px] font-semibold text-muted-foreground focus:outline-none max-w-[190px] truncate"
                >
                  <option v-for="m in store.merchants" :key="m.id" :value="m.id">
                    {{ m.name }} ({{ m.code }})
                  </option>
                </select>
              </div>
            </div>
          </div>

          <!-- Right: Search/Command Launcher, Live Latency, Theme, Sync -->
          <div class="flex items-center gap-1.5 sm:gap-2 shrink-0">
            <!-- Command & Simulation Palette Trigger -->
            <button
              type="button"
              @click="isCommandOpen = true"
              class="inline-flex items-center gap-2 rounded-xl border border-border bg-elevated hover:bg-muted px-2.5 sm:px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition cursor-pointer"
            >
              <Command class="w-3.5 h-3.5 text-tone-emerald" />
              <span class="hidden sm:inline font-medium">Simulasi &amp; Menu</span>
              <kbd class="hidden md:inline-block rounded bg-card px-1.5 py-0.2 font-mono text-[10px] border border-border">
                Ctrl K
              </kbd>
            </button>

            <!-- Live Backend Telemetry -->
            <div class="hidden xl:flex items-center gap-2 px-3 py-1.5 rounded-xl bg-elevated border border-border font-mono text-[11px] tabular-nums">
              <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
              <span class="text-tone-emerald font-bold">{{ store.lastApiLatencyMs }}ms</span>
              <span class="text-muted-foreground">|</span>
              <span class="text-muted-foreground">{{ store.queueMetrics.queue_backend }}</span>
            </div>

            <!-- Two-State Opposite-of-Rendered Theme Button (per dark-mode-toggle) -->
            <Button
              variant="outline"
              size="icon"
              :aria-label="resolvedTheme === 'dark' ? 'Beralih ke tema terang' : 'Beralih ke tema gelap'"
              @click="toggleOppositeScheme"
            >
              <Sun v-if="resolvedTheme === 'dark'" class="w-4 h-4 text-amber-400" />
              <Moon v-else class="w-4 h-4 text-secondary-foreground" />
            </Button>

            <Button
              variant="outline"
              size="icon"
              aria-label="Sinkronisasi data sekarang"
              @click="store.fetchOverview()"
            >
              <RefreshCw class="w-3.5 h-3.5" :class="store.isLoading ? 'animate-spin' : ''" />
            </Button>

            <Button variant="emerald" size="sm" class="hidden sm:inline-flex" @click="openAiModal">
              <Sparkles class="w-3.5 h-3.5" />
              <span>+ AI Order</span>
            </Button>
          </div>
        </div>
      </header>

      <!-- Main Workspace Container (Extra bottom padding on mobile for fixed Bottom Tab Bar) -->
      <main class="flex-1 w-full max-w-7xl mx-auto px-3.5 sm:px-6 py-4 sm:py-6 pb-24 lg:pb-8 space-y-5">
        <!-- Bento KPI Summary Strip (2x2 on Mobile -> 4 Columns on Desktop) -->
        <div v-if="activeView !== 'ledger'" class="grid grid-cols-2 lg:grid-cols-4 gap-3 tabular-nums">
          <div class="p-3.5 sm:p-4 rounded-2xl bg-card border border-border shadow-2xs">
            <div class="text-[10px] sm:text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
              Active Orders
            </div>
            <div class="text-lg sm:text-2xl font-extrabold text-foreground font-mono mt-1">
              {{ store.orders.length }}
            </div>
            <div class="text-[11px] text-tone-emerald font-medium mt-0.5 truncate">
              GMV: {{ formatIDR(store.totalGMV) }}
            </div>
          </div>

          <div class="p-3.5 sm:p-4 rounded-2xl bg-card border border-border shadow-2xs">
            <div class="text-[10px] sm:text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
              3-State Catalog
            </div>
            <div class="text-lg sm:text-2xl font-extrabold text-foreground font-mono mt-1">
              {{ store.products.length }} <span class="text-xs font-normal text-muted-foreground">SKU</span>
            </div>
            <div class="text-[11px] text-tone-amber font-medium mt-0.5 truncate">
              {{ store.lowStockCount }} SKU stok kritis (&le; 2)
            </div>
          </div>

          <div class="p-3.5 sm:p-4 rounded-2xl bg-card border border-border shadow-2xs">
            <div class="text-[10px] sm:text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
              Webhook Guard
            </div>
            <div class="text-lg sm:text-2xl font-extrabold text-tone-indigo font-mono mt-1">
              {{ store.webhookEvents.length }}
            </div>
            <div class="text-[11px] text-muted-foreground mt-0.5 truncate">
              HMAC-SHA256 &lt; 50ms ACK
            </div>
          </div>

          <div class="p-3.5 sm:p-4 rounded-2xl bg-card border border-border shadow-2xs">
            <div class="text-[10px] sm:text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
              Worker &amp; DLQ
            </div>
            <div class="flex items-baseline gap-2 mt-1 font-mono">
              <span class="text-lg sm:text-2xl font-extrabold text-tone-emerald">
                {{ store.queueMetrics.completed_count }}
              </span>
              <span class="text-xs text-muted-foreground">ok /</span>
              <span
                :class="store.queueMetrics.failed_dlq_count > 0 ? 'text-tone-rose' : 'text-muted-foreground'"
                class="text-sm sm:text-base font-bold"
              >
                {{ store.queueMetrics.failed_dlq_count }} DLQ
              </span>
            </div>
            <div class="text-[11px] text-muted-foreground mt-0.5 truncate">
              API Mode: {{ store.queueMetrics.chaos_mode }}
            </div>
          </div>
        </div>

        <!-- VIEW: OVERVIEW (All 4 OMS Modules) -->
        <template v-if="activeView === 'overview'">
          <OrderInboxPanel
            @open-ai-parser="openAiModal"
            @print-waybill="openWaybillModal"
            @inspect-order="openOrderInspector"
          />
          <InventoryMonitorPanel />
          <StressSimulatorPanel />
          <QueueMonitorPanel />
        </template>

        <!-- VIEW: FOCUSED SINGLE MODULES (Super Clean on Mobile Tabs!) -->
        <OrderInboxPanel
          v-else-if="activeView === 'orders'"
          @open-ai-parser="openAiModal"
          @print-waybill="openWaybillModal"
          @inspect-order="openOrderInspector"
        />

        <InventoryMonitorPanel v-else-if="activeView === 'inventory'" />

        <StressSimulatorPanel v-else-if="activeView === 'lab'" />

        <QueueMonitorPanel v-else-if="activeView === 'queues'" />

        <CorePaymentLedgerView v-else-if="activeView === 'ledger'" />
      </main>
    </div>

    <!-- MOBILE FLOATING ACTION BUTTON (Thumb-Zone Quick AI Order Capture) -->
    <button
      type="button"
      aria-label="Tambah order via AI WhatsApp Parser"
      @click="openAiModal"
      class="sm:hidden fixed right-4 bottom-20 z-30 inline-flex items-center gap-2 rounded-full bg-emerald-600 text-white px-4 py-3 text-xs font-extrabold shadow-lg shadow-emerald-600/30 active:scale-95 transition"
    >
      <Sparkles class="w-4 h-4" />
      <span>+ AI WA</span>
    </button>

    <!-- MOBILE FIXED BOTTOM NAVIGATION BAR (< 1024px) -->
    <nav
      aria-label="Navigasi bawah mobile"
      class="lg:hidden fixed inset-x-0 bottom-0 z-30 border-t border-border bg-card/95 backdrop-blur-md pb-safe"
    >
      <div class="grid grid-cols-5 px-1 pt-1.5">
        <button
          v-for="item in navItems.slice(1)"
          :key="item.id"
          type="button"
          @click="switchView(item.id)"
          :class="activeView === item.id ? 'text-tone-emerald font-bold' : 'text-muted-foreground'"
          class="relative flex flex-col items-center justify-center gap-0.5 py-1.5 text-[10px] transition cursor-pointer"
        >
          <component :is="item.icon" class="w-4 h-4" />
          <span>{{ item.shortLabel }}</span>
          <span
            v-if="item.id === 'queues' && store.queueMetrics.failed_dlq_count > 0"
            class="absolute top-1 right-3 h-2 w-2 rounded-full bg-rose-500"
          />
        </button>
      </div>
    </nav>

    <!-- Adaptive Modals & Drawers (Desktop Dialog / Sheet -> Mobile Swipeable Vaul Drawer) -->
    <AiOrderParserModal ref="aiModalRef" />
    <WaybillPrintModal ref="waybillModalRef" />
    <CommandPaletteModal
      v-model:open="isCommandOpen"
      @navigate="(sec) => switchView(sec)"
      @open-ai-parser="openAiModal"
    />
    <OrderInspectorSheet
      v-model:open="isInspectorOpen"
      :order="inspectedOrder"
      @print-waybill="openWaybillModal"
    />
  </div>
</template>
