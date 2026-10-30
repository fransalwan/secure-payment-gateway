<script setup lang="ts">
import { ref, computed } from 'vue'
import ResponsiveDialog from './ui/ResponsiveDialog.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { useThemeScheme, type ColorSchemePref } from '../composables/useThemeScheme'
import {
  Search,
  Sparkles,
  Zap,
  ShieldCheck,
  AlertTriangle,
  CheckCircle2,
  ShoppingBag,
  Boxes,
  Activity,
  Landmark,
  Sun,
  Moon,
  Monitor,
  ArrowRight,
} from '@lucide/vue'

export type NavSection = 'orders' | 'inventory' | 'lab' | 'queues' | 'ledger'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
  (e: 'navigate', section: NavSection): void
  (e: 'openAiParser'): void
}>()

const store = useOmsStore()
const { storedScheme, systemIsDark, setScheme } = useThemeScheme()
const query = ref('')

interface ActionItem {
  id: string
  group: 'Simulasi Live & AI' | 'Navigasi Modul'
  title: string
  subtitle: string
  badge?: string
  badgeVariant?: 'emerald' | 'amber' | 'indigo' | 'rose' | 'sky'
  icon: unknown
  run: () => void
}

const actions = computed<ActionItem[]>(() => [
  {
    id: 'sim-ai',
    group: 'Simulasi Live & AI',
    title: '1. AI WhatsApp & Live Selling Order Capture',
    subtitle: 'Ekstrak chat WA tidak terstruktur ke katalog SKU PostgreSQL (< 3s SLA)',
    badge: 'AI + NLP',
    badgeVariant: 'emerald',
    icon: Sparkles,
    run: () => {
      emit('update:open', false)
      emit('openAiParser')
    },
  },
  {
    id: 'sim-race',
    group: 'Simulasi Live & AI',
    title: '2. Simulate 10 Concurrent Buyers vs 1 Last Stock',
    subtitle: 'Uji PostgreSQL SELECT FOR UPDATE anti-overselling lintas kanal',
    badge: 'Race Lock',
    badgeVariant: 'amber',
    icon: Zap,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'lab')
      store.runRaceConditionSim(store.products[0]?.id || '', 10, 1)
    },
  },
  {
    id: 'sim-webhook',
    group: 'Simulasi Live & AI',
    title: '3. Simulate 5x Duplicate Payment Webhook',
    subtitle: 'Uji HMAC-SHA256 constant-time & deduplikasi atomik webhook_events',
    badge: 'Idempotent',
    badgeVariant: 'indigo',
    icon: ShieldCheck,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'lab')
      store.runDuplicateWebhookSim('', 5)
    },
  },
  {
    id: 'sim-chaos-429',
    group: 'Simulasi Live & AI',
    title: '4. Aktifkan Simulasi HTTP 429 Rate Limit -> Force DLQ',
    subtitle: 'Paksa API logistik gagal 4x untuk menguji Exponential Backoff & DLQ',
    badge: 'Chaos 429',
    badgeVariant: 'rose',
    icon: AlertTriangle,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'queues')
      store.setChaosMode('ALWAYS_429')
    },
  },
  {
    id: 'sim-chaos-off',
    group: 'Simulasi Live & AI',
    title: '5. Pulihkan API Pihak Ketiga ke Normal (200 OK)',
    subtitle: 'Matikan chaos mode agar Manual Retry di DLQ berhasil menerbitkan AWB',
    badge: 'Healthy',
    badgeVariant: 'emerald',
    icon: CheckCircle2,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'queues')
      store.setChaosMode('OFF')
    },
  },
  {
    id: 'nav-orders',
    group: 'Navigasi Modul',
    title: 'Unified Omni-Channel Order Inbox',
    subtitle: 'Kelola pesanan Web, Shopee, TikTok Shop & WhatsApp Live',
    icon: ShoppingBag,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'orders')
    },
  },
  {
    id: 'nav-inventory',
    group: 'Navigasi Modul',
    title: 'Real-Time 3-State Inventory Engine',
    subtitle: 'Pantau stok Available, Reserved, dan Sold per SKU',
    icon: Boxes,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'inventory')
    },
  },
  {
    id: 'nav-queues',
    group: 'Navigasi Modul',
    title: 'Redis Queue, Retry Backoff & System Error Logs (DLQ)',
    subtitle: 'Monitor worker asinkron, webhook audit trail, dan manual retry',
    icon: Activity,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'queues')
    },
  },
  {
    id: 'nav-ledger',
    group: 'Navigasi Modul',
    title: 'Core Payment Switch, AES Vault & 3-Way CSV Recon',
    subtitle: 'Double-entry ledger akuntansi & pencocokan mutasi bank',
    icon: Landmark,
    run: () => {
      emit('update:open', false)
      emit('navigate', 'ledger')
    },
  },
])

const filteredActions = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return actions.value
  return actions.value.filter(
    (a) => a.title.toLowerCase().includes(q) || a.subtitle.toLowerCase().includes(q),
  )
})

const simItems = computed(() => filteredActions.value.filter((i) => i.group === 'Simulasi Live & AI'))
const navItems = computed(() => filteredActions.value.filter((i) => i.group === 'Navigasi Modul'))

const schemeOptions = computed<{ value: ColorSchemePref; label: string; icon: unknown }[]>(() => [
  {
    value: 'light dark',
    label: `System (${systemIsDark.value ? 'Dark' : 'Light'})`,
    icon: Monitor,
  },
  {
    value: 'light',
    label: 'Light',
    icon: Sun,
  },
  {
    value: 'dark',
    label: 'Dark',
    icon: Moon,
  },
])
</script>

<template>
  <ResponsiveDialog
    :open="open"
    title="Command & Simulation Launcher"
    description="Akses cepat modul operasional, skenario stress-test konkurensi, dan preferensi tampilan"
    max-width-class="max-w-xl"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-4">
      <!-- Search Bar -->
      <div class="relative">
        <Search class="w-4 h-4 text-muted-foreground absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
        <input
          v-model="query"
          type="search"
          autofocus
          placeholder="Cari aksi simulasi, menu OMS, atau ketik 'race', 'webhook', 'resi'..."
          class="w-full rounded-xl border border-border bg-elevated pl-10 pr-3.5 py-2.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
        />
      </div>

      <!-- Group 1: Live Engineering Simulations -->
      <div v-if="simItems.length > 0" class="space-y-1.5">
        <div class="px-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          Skenario Demo &amp; Simulasi Produksi
        </div>
        <div class="space-y-1">
          <button
            v-for="item in simItems"
            :key="item.id"
            type="button"
            @click="item.run()"
            class="w-full flex items-center justify-between gap-3 rounded-xl border border-transparent hover:border-border hover:bg-elevated p-2.5 text-left transition cursor-pointer group"
          >
            <div class="flex items-start gap-3 min-w-0">
              <div class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-muted border border-border text-foreground">
                <component :is="item.icon" class="w-4 h-4" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-xs font-semibold text-foreground">{{ item.title }}</span>
                  <Badge v-if="item.badge" :variant="item.badgeVariant">{{ item.badge }}</Badge>
                </div>
                <p class="text-[11px] text-muted-foreground truncate mt-0.5">{{ item.subtitle }}</p>
              </div>
            </div>
            <ArrowRight class="w-4 h-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition shrink-0" />
          </button>
        </div>
      </div>

      <!-- Group 2: Module Navigation -->
      <div v-if="navItems.length > 0" class="space-y-1.5">
        <div class="px-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          Navigasi Workspace
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
          <button
            v-for="item in navItems"
            :key="item.id"
            type="button"
            @click="item.run()"
            class="flex items-center gap-2.5 rounded-xl border border-border/60 bg-elevated/40 hover:bg-elevated p-2.5 text-left transition cursor-pointer"
          >
            <component :is="item.icon" class="w-4 h-4 text-muted-foreground shrink-0" />
            <div class="min-w-0">
              <div class="text-xs font-semibold text-foreground truncate">{{ item.title }}</div>
              <div class="text-[10px] text-muted-foreground truncate">{{ item.subtitle }}</div>
            </div>
          </button>
        </div>
      </div>

      <!-- Group 3: 3-State Color Scheme Override (per modern-web-guidance: dark-mode-toggle) -->
      <div class="pt-3 border-t border-border flex flex-wrap items-center justify-between gap-2">
        <span class="text-xs font-medium text-muted-foreground">Skema Tampilan (Color Scheme):</span>
        <div class="inline-flex rounded-xl border border-border bg-elevated p-1 gap-1">
          <button
            v-for="opt in schemeOptions"
            :key="opt.value"
            type="button"
            @click="setScheme(opt.value)"
            :class="
              storedScheme === opt.value
                ? 'bg-card text-foreground shadow-xs font-bold border border-border'
                : 'text-muted-foreground hover:text-foreground border border-transparent'
            "
            class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-[11px] transition cursor-pointer"
          >
            <component :is="opt.icon" class="w-3.5 h-3.5" />
            <span>{{ opt.label }}</span>
          </button>
        </div>
      </div>
    </div>
  </ResponsiveDialog>
</template>

