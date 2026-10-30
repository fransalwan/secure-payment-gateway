<script setup lang="ts">
import { computed } from 'vue'
import { useClipboard } from '@vueuse/core'
import ResponsiveDialog from './ui/ResponsiveDialog.vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { formatIDR } from '../lib/utils'
import type { Order } from '../types/oms'
import {
  Copy,
  Check,
  Printer,
  CreditCard,
  XCircle,
  Truck,
  Phone,
  MapPin,
  PackageCheck,
} from '@lucide/vue'

const props = defineProps<{
  open: boolean
  order: Order | null
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
  (e: 'printWaybill', order: Order): void
}>()

const store = useOmsStore()
const { copy, copied } = useClipboard()

const lifecycleSteps = computed(() => {
  const st = props.order?.status || 'PENDING_PAYMENT'
  if (st === 'CANCELLED') {
    return [
      { key: 'PENDING_PAYMENT', label: 'Order Dibuat', done: true },
      { key: 'CANCELLED', label: 'Dibatalkan (Stok Kembali)', done: true, error: true },
    ]
  }
  const orderRank: Record<string, number> = {
    PENDING_PAYMENT: 1,
    PAID: 2,
    READY_TO_SHIP: 3,
    SHIPPED: 4,
    COMPLETED: 5,
  }
  const rank = orderRank[st] ?? 1
  return [
    { key: 'PENDING_PAYMENT', label: 'Stok Dikunci', done: rank >= 1, error: false },
    { key: 'PAID', label: 'Paid (HMAC)', done: rank >= 2, error: false },
    { key: 'READY_TO_SHIP', label: 'AWB Terbit', done: rank >= 3, error: false },
    { key: 'SHIPPED', label: 'Dikirim', done: rank >= 4, error: false },
  ]
})

async function handleSimulatePay() {
  if (!props.order) return
  await store.simulatePayOrder(props.order.id)
  emit('update:open', false)
}

async function handleCancel() {
  if (!props.order) return
  await store.cancelOrder(props.order.id)
  emit('update:open', false)
}
</script>

<template>
  <ResponsiveDialog
    :open="open"
    :title="order ? `Detail Pesanan ${order.order_number}` : 'Detail Pesanan'"
    description="Inspeksi state machine satu arah, rincian SKU terkunci, dan nomor resi ekspedisi"
    :side-sheet-on-desktop="true"
    @update:open="emit('update:open', $event)"
  >
    <div v-if="order" class="space-y-5 text-xs">
      <!-- Top Summary Badges -->
      <div class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-elevated/60 p-3.5">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <span class="font-mono font-bold text-sm text-foreground">{{ order.order_number }}</span>
            <button
              type="button"
              @click="copy(order.order_number)"
              class="inline-flex items-center gap-1 rounded-md border border-border bg-card px-1.5 py-0.5 text-[10px] text-muted-foreground hover:text-foreground cursor-pointer"
            >
              <Check v-if="copied" class="w-3 h-3 text-tone-emerald" />
              <Copy v-else class="w-3 h-3" />
              <span>{{ copied ? 'Tersalin' : 'Copy' }}</span>
            </button>
          </div>
          <div class="text-[11px] text-muted-foreground">
            Kanal: <strong class="text-foreground">{{ order.channel }}</strong> &bull; Kurir:
            <strong class="text-foreground">{{ order.courier_code }}</strong>
          </div>
        </div>
        <div class="text-right">
          <div class="font-mono font-extrabold text-base text-tone-emerald tabular-nums">
            {{ formatIDR(order.total_amount) }}
          </div>
          <Badge :variant="order.status === 'READY_TO_SHIP' ? 'emerald' : order.status === 'CANCELLED' ? 'rose' : 'amber'">
            {{ order.status }}
          </Badge>
        </div>
      </div>

      <!-- One-Way State Machine Stepper -->
      <div class="rounded-xl border border-border bg-card p-3.5 space-y-2.5">
        <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          One-Way Order State Machine
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
          <div
            v-for="(step, idx) in lifecycleSteps"
            :key="step.key"
            :class="
              step.error
                ? 'border-rose-500/40 bg-rose-500/10 text-tone-rose'
                : step.done
                  ? 'border-emerald-500/40 bg-emerald-500/10 text-tone-emerald'
                  : 'border-border bg-elevated/40 text-muted-foreground'
            "
            class="rounded-lg border p-2 flex items-center gap-2"
          >
            <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-card border border-border font-mono text-[10px] font-bold">
              {{ idx + 1 }}
            </span>
            <span class="font-semibold text-[11px] leading-tight">{{ step.label }}</span>
          </div>
        </div>
      </div>

      <!-- Waybill AWB Card -->
      <div v-if="order.waybill_number" class="rounded-xl border border-emerald-500/30 bg-emerald-500/8 p-3.5 flex items-center justify-between gap-2">
        <div class="flex items-center gap-2.5 min-w-0">
          <Truck class="w-4 h-4 text-tone-emerald shrink-0" />
          <div class="min-w-0">
            <div class="text-[10px] uppercase font-bold text-muted-foreground">Nomor Resi Ekspedisi (AWB)</div>
            <div class="font-mono font-bold text-sm text-foreground truncate">{{ order.waybill_number }}</div>
          </div>
        </div>
        <Button variant="emerald" size="sm" @click="emit('printWaybill', order)">
          <Printer class="w-3.5 h-3.5" />
          <span>Print A6</span>
        </Button>
      </div>

      <!-- Customer & Shipping Info -->
      <div class="rounded-xl border border-border bg-card p-3.5 space-y-2.5">
        <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          Informasi Penerima &amp; Pengiriman
        </div>
        <div class="font-bold text-sm text-foreground">{{ order.customer_name }}</div>
        <div class="flex items-center gap-2 text-muted-foreground font-mono">
          <Phone class="w-3.5 h-3.5 shrink-0" />
          <span>{{ order.customer_phone }}</span>
        </div>
        <div class="flex items-start gap-2 text-secondary-foreground leading-relaxed">
          <MapPin class="w-3.5 h-3.5 shrink-0 mt-0.5 text-muted-foreground" />
          <span>{{ order.shipping_address }}</span>
        </div>
      </div>

      <!-- Locked SKU Items -->
      <div class="rounded-xl border border-border bg-card overflow-hidden">
        <div class="px-3.5 py-2.5 bg-elevated/60 border-b border-border flex items-center gap-2">
          <PackageCheck class="w-3.5 h-3.5 text-muted-foreground" />
          <span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Rincian Item SKU ({{ order.items?.length || 0 }})
          </span>
        </div>
        <div class="divide-y divide-border">
          <div
            v-for="item in order.items"
            :key="item.id"
            class="px-3.5 py-2.5 flex items-center justify-between gap-2"
          >
            <div class="min-w-0">
              <div class="font-semibold text-foreground truncate">{{ item.product_name }}</div>
              <div class="font-mono text-[10px] text-muted-foreground">
                {{ item.sku }} &bull; {{ formatIDR(item.unit_price) }} &times; {{ item.qty }}
              </div>
            </div>
            <div class="font-mono font-bold text-foreground tabular-nums shrink-0">
              {{ formatIDR(item.subtotal) }}
            </div>
          </div>
        </div>
      </div>

      <!-- Primary Operational Actions (Thumb-Friendly Full Width on Mobile) -->
      <div v-if="order.status === 'PENDING_PAYMENT'" class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 pt-2">
        <Button variant="indigo" size="lg" class="w-full" @click="handleSimulatePay">
          <CreditCard class="w-4 h-4" />
          <span>Simulasi Bayar (HMAC Webhook)</span>
        </Button>
        <Button variant="destructive" size="lg" class="w-full" @click="handleCancel">
          <XCircle class="w-4 h-4" />
          <span>Batalkan &amp; Lepas Stok</span>
        </Button>
      </div>
    </div>
  </ResponsiveDialog>
</template>

