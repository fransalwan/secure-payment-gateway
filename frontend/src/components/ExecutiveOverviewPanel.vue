<script setup lang="ts">
import { computed } from 'vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { useAuthStore } from '../stores/authStore'
import { formatIDR } from '../lib/utils'
import type { Order } from '../types/oms'
import {
  ShieldCheck,
  Database,
  Cpu,
  Landmark,
  AlertTriangle,
  ArrowRight,
  Sparkles,
  Zap,
  Eye,
  Lock,
  CheckCircle2,
  Flame,
  Camera,
  Activity,
} from '@lucide/vue'

const emit = defineEmits<{
  (e: 'navigate', view: 'orders' | 'inventory' | 'lab' | 'queues' | 'ledger' | 'reports' | 'telemetry'): void
  (e: 'openAiParser'): void
  (e: 'openSlipValidator', orderId?: string): void
  (e: 'inspectOrder', order: Order): void
}>()

const store = useOmsStore()
const authStore = useAuthStore()

const suspiciousOrders = computed(() =>
  store.orders.filter((o) => o.status === 'SUSPICIOUS_HOLD'),
)

const pendingOrders = computed(() =>
  store.orders.filter((o) => o.status === 'PENDING_PAYMENT'),
)

const criticalProducts = computed(() =>
  store.products.filter((p) => p.available_stock <= 5),
)

const topPriorityOrders = computed(() => {
  const sorted = [...store.orders].sort((a, b) => {
    const score = (st: string) =>
      st === 'SUSPICIOUS_HOLD' ? 0 : st === 'PENDING_PAYMENT' ? 1 : st === 'PAID' ? 2 : 3
    return score(a.status) - score(b.status)
  })
  return sorted.slice(0, 5)
})

function statusVariant(st: string): 'emerald' | 'indigo' | 'amber' | 'rose' | 'default' {
  switch (st) {
    case 'READY_TO_SHIP':
      return 'emerald'
    case 'PAID':
      return 'indigo'
    case 'PENDING_PAYMENT':
      return 'amber'
    case 'SUSPICIOUS_HOLD':
    case 'CANCELLED':
      return 'rose'
    default:
      return 'default'
  }
}
</script>

<template>
  <div class="space-y-5">
    <!-- 1. INTERACTIVE 4-STAGE SYSTEM ARCHITECTURE PIPELINE -->
    <section class="rounded-2xl bg-card border border-border shadow-xs overflow-hidden">
      <div class="p-4 sm:p-5 border-b border-border flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
        <div>
          <div class="flex items-center gap-2">
            <Badge variant="indigo">SYSTEM ARCHITECTURE</Badge>
            <h2
              id="executive-overview-heading"
              tabindex="-1"
              class="text-sm sm:text-base font-bold text-foreground focus:outline-none"
            >
              End-to-End Zero-Trust AI &amp; ACID Settlement Pipeline
            </h2>
          </div>
          <p class="text-xs text-muted-foreground mt-0.5">
            Klik tahapan arsitektur di bawah untuk menguji langsung komponen mikroservis, guardrails AI, dan database lock.
          </p>
        </div>
        <div class="flex items-center gap-2">
          <Badge variant="emerald" class="font-mono text-[10px]">
            Zero Dual-Write (Outbox Pattern)
          </Badge>
          <Badge variant="outline" class="font-mono text-[10px]">
            Role: {{ authStore.currentUser.role }}
          </Badge>
        </div>
      </div>

      <div class="p-4 sm:p-5 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3.5">
        <!-- Stage 1 -->
        <div class="rounded-xl border border-border bg-elevated/40 p-3.5 flex flex-col justify-between gap-3 hover:border-emerald-500/50 transition">
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-mono text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-emerald-500/15 text-tone-emerald">
                Stage 01 // Ingress
              </span>
              <ShieldCheck class="w-4 h-4 text-tone-emerald" />
            </div>
            <h3 class="text-xs font-extrabold text-foreground">
              Fast-ACK HTTP, HMAC &amp; 64-bit dHash Gate
            </h3>
            <p class="text-[11px] text-muted-foreground leading-relaxed">
              Validasi tanda tangan <code class="text-foreground">HMAC-SHA256</code> (replay window &plusmn;5m) &amp; deduplikasi gambar struk <code class="text-foreground">Perceptual dHash</code> di memori (&lt;15ms) sebelum memanggil LLM.
            </p>
          </div>
          <div class="pt-2 border-t border-border/60 flex items-center justify-between gap-2">
            <button
              type="button"
              @click="emit('openAiParser')"
              class="text-[11px] font-bold text-tone-emerald hover:underline inline-flex items-center gap-1 cursor-pointer"
            >
              <Sparkles class="w-3 h-3" />
              <span>AI WA Parser</span>
            </button>
            <button
              type="button"
              @click="emit('openSlipValidator')"
              class="text-[11px] font-bold text-tone-indigo hover:underline inline-flex items-center gap-1 cursor-pointer"
            >
              <Camera class="w-3 h-3" />
              <span>Uji Slip dHash &rarr;</span>
            </button>
          </div>
        </div>

        <!-- Stage 2 -->
        <div class="rounded-xl border border-border bg-elevated/40 p-3.5 flex flex-col justify-between gap-3 hover:border-amber-500/50 transition">
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-mono text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-amber-500/15 text-tone-amber">
                Stage 02 // ACID DB
              </span>
              <Database class="w-4 h-4 text-tone-amber" />
            </div>
            <h3 class="text-xs font-extrabold text-foreground">
              PostgreSQL 16 Pessimistic Lock &amp; Outbox
            </h3>
            <p class="text-[11px] text-muted-foreground leading-relaxed">
              Isolasi stok 3-State (<code class="text-foreground">Available &rarr; Reserved &rarr; Sold</code>) via <code class="text-foreground">SELECT ... FOR UPDATE</code> berurutan + <code class="text-foreground">outbox_events</code> atomik.
            </p>
          </div>
          <div class="pt-2 border-t border-border/60 flex items-center justify-between gap-2">
            <button
              type="button"
              @click="emit('navigate', 'inventory')"
              class="text-[11px] font-bold text-foreground hover:text-tone-amber hover:underline cursor-pointer"
            >
              3-State Stock ({{ store.products.length }})
            </button>
            <button
              type="button"
              @click="emit('navigate', 'lab')"
              class="text-[11px] font-bold text-tone-amber hover:underline inline-flex items-center gap-1 cursor-pointer"
            >
              <Zap class="w-3 h-3" />
              <span>Race Test &rarr;</span>
            </button>
          </div>
        </div>

        <!-- Stage 3 -->
        <div class="rounded-xl border border-border bg-elevated/40 p-3.5 flex flex-col justify-between gap-3 hover:border-indigo-500/50 transition">
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-mono text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-indigo-500/15 text-tone-indigo">
                Stage 03 // Async Broker
              </span>
              <Cpu class="w-4 h-4 text-tone-indigo" />
            </div>
            <h3 class="text-xs font-extrabold text-foreground">
              RabbitMQ &amp; Redis Workers + Self-Healing DLQ
            </h3>
            <p class="text-[11px] text-muted-foreground leading-relaxed">
              Outbox relay daemon mempublikasikan job ke worker pool dengan <code class="text-foreground">Exponential Backoff</code> otomatis &amp; Dead-Letter Queue recovery.
            </p>
          </div>
          <div class="pt-2 border-t border-border/60 flex items-center justify-between gap-2">
            <span class="font-mono text-[10px] text-muted-foreground">
              OK: <strong class="text-tone-emerald">{{ store.queueMetrics.completed_count }}</strong> |
              DLQ: <strong :class="store.queueMetrics.failed_dlq_count > 0 ? 'text-tone-rose' : 'text-foreground'">{{ store.queueMetrics.failed_dlq_count }}</strong>
            </span>
            <button
              type="button"
              @click="emit('navigate', 'queues')"
              class="text-[11px] font-bold text-tone-indigo hover:underline inline-flex items-center gap-1 cursor-pointer"
            >
              <Activity class="w-3 h-3" />
              <span>Queue &amp; DLQ &rarr;</span>
            </button>
          </div>
        </div>

        <!-- Stage 4 -->
        <div class="rounded-xl border border-border bg-elevated/40 p-3.5 flex flex-col justify-between gap-3 hover:border-purple-500/50 transition">
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-mono text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-purple-500/15 text-purple-400">
                Stage 04 // AI &amp; Ledger
              </span>
              <Landmark class="w-4 h-4 text-purple-400" />
            </div>
            <h3 class="text-xs font-extrabold text-foreground">
              Double-Entry Ledger, AES-256-GCM &amp; Text-to-SQL
            </h3>
            <p class="text-[11px] text-muted-foreground leading-relaxed">
              Setiap settlement mencatat jurnal <code class="text-foreground">DEBIT/CREDIT</code> berpasangan, hash-chain audit log <code class="text-foreground">SHA-256</code>, &amp; analitik SQL <code class="text-foreground">READ ONLY</code>.
            </p>
          </div>
          <div class="pt-2 border-t border-border/60 flex items-center justify-between gap-2">
            <button
              type="button"
              @click="emit('navigate', 'reports')"
              class="text-[11px] font-bold text-purple-400 hover:underline inline-flex items-center gap-1 cursor-pointer"
            >
              <Sparkles class="w-3 h-3" />
              <span>Text-to-SQL</span>
            </button>
            <button
              type="button"
              @click="emit('navigate', 'ledger')"
              class="text-[11px] font-bold text-tone-emerald hover:underline inline-flex items-center gap-1 cursor-pointer"
            >
              <Lock class="w-3 h-3" />
              <span>Ledger &amp; KYC &rarr;</span>
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- 2. EXECUTIVE WATCHLIST: PRIORITY ORDERS & ACTIONABLE RISK ALERTS -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <!-- Left 7 Cols: Priority & Recent Orders Snapshot -->
      <section class="lg:col-span-7 rounded-2xl bg-card border border-border shadow-xs overflow-hidden flex flex-col justify-between">
        <div>
          <div class="p-4 border-b border-border flex items-center justify-between gap-2">
            <div>
              <div class="flex items-center gap-2">
                <Badge variant="emerald">LIVE SNAPSHOT</Badge>
                <h3 class="text-sm font-bold text-foreground">
                  Prioritas &amp; Transaksi Terbaru (Top 5)
                </h3>
              </div>
              <p class="text-[11px] text-muted-foreground mt-0.5">
                Menampilkan transaksi yang membutuhkan atensi (Fraud Hold / Pending) terlebih dahulu.
              </p>
            </div>
            <Button variant="outline" size="sm" class="text-xs font-semibold" @click="emit('navigate', 'orders')">
              <span>Buka Inbox Lengkap</span>
              <ArrowRight class="w-3.5 h-3.5 ml-1" />
            </Button>
          </div>

          <div class="divide-y divide-border">
            <div
              v-for="ord in topPriorityOrders"
              :key="ord.id"
              class="p-3.5 hover:bg-elevated/40 transition flex flex-col sm:flex-row sm:items-center justify-between gap-2.5"
            >
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <button
                    type="button"
                    @click="emit('inspectOrder', ord)"
                    class="font-mono text-xs font-extrabold text-foreground hover:text-tone-emerald hover:underline cursor-pointer"
                  >
                    {{ ord.order_number }}
                  </button>
                  <Badge :variant="statusVariant(ord.status)">{{ ord.status }}</Badge>
                  <span class="text-[11px] text-muted-foreground font-mono">{{ ord.channel }}</span>
                </div>
                <div class="text-xs text-secondary-foreground mt-1 truncate">
                  <strong>{{ ord.customer_name }}</strong> &bull;
                  <span class="text-muted-foreground">{{ ord.items?.[0]?.product_name || 'Multi-SKU Order' }}</span>
                </div>
              </div>

              <div class="flex items-center justify-between sm:justify-end gap-2.5 shrink-0">
                <div class="text-right font-mono">
                  <div class="text-xs font-extrabold text-foreground">{{ formatIDR(ord.total_amount) }}</div>
                  <div class="text-[10px] text-muted-foreground">{{ ord.courier_code }} REG</div>
                </div>
                <Button
                  v-if="ord.status === 'SUSPICIOUS_HOLD'"
                  variant="destructive"
                  size="sm"
                  class="text-[11px]"
                  @click="emit('inspectOrder', ord)"
                >
                  <AlertTriangle class="w-3.5 h-3.5" />
                  <span>Review Fraud</span>
                </Button>
                <Button
                  v-else
                  variant="secondary"
                  size="sm"
                  class="text-[11px]"
                  @click="emit('inspectOrder', ord)"
                >
                  <Eye class="w-3.5 h-3.5" />
                  <span>Detail</span>
                </Button>
              </div>
            </div>
          </div>
        </div>

        <div class="px-4 py-2.5 bg-elevated/40 border-t border-border flex items-center justify-between text-[11px] text-muted-foreground">
          <span>Total Pesanan Aktif: <strong class="text-foreground">{{ store.orders.length }}</strong> ({{ pendingOrders.length }} menunggu pembayaran)</span>
          <button
            type="button"
            @click="emit('navigate', 'orders')"
            class="font-bold text-tone-emerald hover:underline cursor-pointer"
          >
            Kelola Semua Order &rarr;
          </button>
        </div>
      </section>

      <!-- Right 5 Cols: Fraud Hold & Critical Inventory Watchlist -->
      <div class="lg:col-span-5 space-y-4">
        <!-- Fraud & Anomaly Hold Card (Human-in-the-Loop) -->
        <section
          class="rounded-2xl border p-4 shadow-xs transition"
          :class="
            suspiciousOrders.length > 0
              ? 'bg-rose-950/15 border-rose-500/40'
              : 'bg-card border-border'
          "
        >
          <div class="flex items-start justify-between gap-2 mb-2.5">
            <div class="flex items-center gap-2">
              <AlertTriangle
                class="w-4 h-4 shrink-0"
                :class="suspiciousOrders.length > 0 ? 'text-tone-rose' : 'text-tone-emerald'"
              />
              <h3 class="text-xs sm:text-sm font-extrabold text-foreground">
                Human-in-the-Loop Fraud Hold Queue
              </h3>
            </div>
            <Badge :variant="suspiciousOrders.length > 0 ? 'rose' : 'emerald'">
              {{ suspiciousOrders.length }} Hold
            </Badge>
          </div>

          <div v-if="suspiciousOrders.length > 0" class="space-y-2.5">
            <p class="text-[11px] text-muted-foreground leading-relaxed">
              Vision AI menahan pesanan berikut karena terdeteksi anomali manipulasi struk atau ketidaksesuaian nominal. Stok tetap dikunci di <code class="text-foreground">Reserved</code> hingga Admin/Finance mengambil keputusan:
            </p>
            <div
              v-for="sOrd in suspiciousOrders"
              :key="sOrd.id"
              class="p-3 rounded-xl bg-card border border-rose-500/30 flex items-center justify-between gap-2"
            >
              <div class="min-w-0">
                <div class="font-mono text-xs font-bold text-foreground truncate">
                  {{ sOrd.order_number }} &bull; {{ sOrd.customer_name }}
                </div>
                <div class="text-[11px] text-tone-rose font-medium">
                  Tagihan: {{ formatIDR(sOrd.total_amount) }} (Suspect Tampered Slip)
                </div>
              </div>
              <Button variant="destructive" size="sm" class="shrink-0 text-[11px]" @click="emit('inspectOrder', sOrd)">
                <span>Putuskan &rarr;</span>
              </Button>
            </div>
          </div>

          <div v-else class="flex items-center justify-between gap-2 pt-1">
            <div class="flex items-center gap-2 text-xs text-tone-emerald font-semibold">
              <CheckCircle2 class="w-4 h-4" />
              <span>Zero Fraud Hold — Semua struk transfer terverifikasi bersih.</span>
            </div>
            <Button variant="outline" size="sm" class="text-[11px]" @click="emit('openSlipValidator')">
              <span>Simulasi Slip Palsu</span>
            </Button>
          </div>
        </section>

        <!-- Critical Inventory Watchlist -->
        <section class="rounded-2xl bg-card border border-border p-4 shadow-xs space-y-3">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <Flame class="w-4 h-4 text-tone-amber" />
              <h3 class="text-xs sm:text-sm font-extrabold text-foreground">
                Watchlist Stok Kritis (&le; 5 Pcs)
              </h3>
            </div>
            <button
              type="button"
              @click="emit('navigate', 'inventory')"
              class="text-[11px] font-bold text-tone-indigo hover:underline cursor-pointer"
            >
              Lihat Semua SKU &rarr;
            </button>
          </div>

          <div class="space-y-2">
            <div
              v-for="prod in criticalProducts"
              :key="prod.id"
              class="p-2.5 rounded-xl bg-elevated/60 border border-border flex items-center justify-between gap-2 text-xs"
            >
              <div class="min-w-0">
                <div class="font-semibold text-foreground truncate">{{ prod.name }}</div>
                <div class="font-mono text-[10px] text-muted-foreground">
                  {{ prod.sku }} &bull; Reserved: {{ prod.reserved_stock }} &bull; Sold: {{ prod.sold_stock }}
                </div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <Badge :variant="prod.available_stock === 0 ? 'rose' : 'amber'" class="font-mono">
                  Sisa {{ prod.available_stock }}
                </Badge>
                <Button
                  variant="outline"
                  size="sm"
                  class="text-[11px] px-2 py-1"
                  @click="emit('navigate', 'lab')"
                >
                  <Zap class="w-3 h-3 text-tone-amber" />
                  <span>Uji Lock</span>
                </Button>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>
