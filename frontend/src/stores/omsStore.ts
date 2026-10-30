import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { toast } from 'vue-sonner'
import type {
  Merchant,
  Product,
  Order,
  WebhookEvent,
  QueueMetrics,
  SystemErrorLog,
  AIParseResponse,
  RaceSimulationResult,
  DuplicateWebhookSimResult,
  ChaosFailMode,
  GatewayAccount,
  LedgerEntry,
  ReconciliationBatch,
} from '../types/oms'

export interface ToastMessage {
  id: number
  type: 'success' | 'error' | 'info' | 'warning'
  title: string
  detail?: string
}

export const useOmsStore = defineStore('oms', () => {
  // Multi-Tenant & Filters
  const merchants = ref<Merchant[]>([])
  const activeMerchantId = ref<string>('11111111-1111-1111-1111-111111111111')
  const channelFilter = ref<string>('')
  const statusFilter = ref<string>('')

  // OMS Core State
  const products = ref<Product[]>([])
  const orders = ref<Order[]>([])
  const webhookEvents = ref<WebhookEvent[]>([])
  const errorLogs = ref<SystemErrorLog[]>([])
  const queueMetrics = ref<QueueMetrics>({
    active_count: 0,
    retrying_count: 0,
    completed_count: 0,
    failed_dlq_count: 0,
    chaos_mode: 'OFF',
    queue_backend: 'redis-streams-zset',
    recent_jobs: [],
  })

  // Stress Simulators State
  const raceResult = ref<RaceSimulationResult | null>(null)
  const webhookSimResult = ref<DuplicateWebhookSimResult | null>(null)
  const isSimulatingRace = ref(false)
  const isSimulatingWebhook = ref(false)
  const isLoading = ref(false)
  const lastApiLatencyMs = ref<number>(3)

  // Core Payment Gateway State (Accounts, Double-Entry Ledger, 3-Way Recon)
  const gatewayAccounts = ref<GatewayAccount[]>([])
  const ledgerEntries = ref<LedgerEntry[]>([])
  const reconBatches = ref<ReconciliationBatch[]>([])

  // Toast Notifications
  const toasts = ref<ToastMessage[]>([])
  let toastSeq = 1

  function notify(type: ToastMessage['type'], title: string, detail?: string) {
    const id = toastSeq++
    toasts.value.push({ id, type, title, detail })
    setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id)
    }, 4500)

    const opts = detail ? { description: detail } : undefined
    if (type === 'success') toast.success(title, opts)
    else if (type === 'error') toast.error(title, opts)
    else if (type === 'warning') toast.warning(title, opts)
    else toast.info(title, opts)
  }

  const activeMerchant = computed(() =>
    merchants.value.find((m) => m.id === activeMerchantId.value) ?? merchants.value[0] ?? null,
  )

  const totalGMV = computed(() =>
    orders.value
      .filter((o) => o.status !== 'CANCELLED')
      .reduce((acc, o) => acc + o.total_amount, 0),
  )

  const lowStockCount = computed(() => products.value.filter((p) => p.available_stock <= 2).length)

  async function fetchOverview(silent = false) {
    if (!silent) isLoading.value = true
    const t0 = performance.now()
    try {
      const params = new URLSearchParams()
      if (activeMerchantId.value) params.set('merchant_id', activeMerchantId.value)
      if (channelFilter.value) params.set('channel', channelFilter.value)
      if (statusFilter.value) params.set('status', statusFilter.value)

      const res = await fetch(`/api/v1/dashboard/overview?${params.toString()}`)
      const json = await res.json()
      lastApiLatencyMs.value = Math.max(1, Math.round(performance.now() - t0))

      if (json.success && json.data) {
        const d = json.data
        merchants.value = d.merchants || []
        if (d.active_merchant?.id) {
          activeMerchantId.value = d.active_merchant.id
        }
        products.value = d.products || []
        orders.value = d.orders || []
        webhookEvents.value = d.webhook_events || []
        queueMetrics.value = d.queue_metrics || queueMetrics.value
        errorLogs.value = d.error_logs || []
      }
    } catch (err: unknown) {
      if (!silent) {
        notify('error', 'Gagal memuat dashboard OMS', err instanceof Error ? err.message : String(err))
      }
    } finally {
      if (!silent) isLoading.value = false
    }
  }

  async function parseSocialChat(rawText: string): Promise<AIParseResponse | null> {
    const res = await fetch('/api/v1/orders/ai-parse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        merchant_id: activeMerchantId.value,
        raw_text: rawText,
      }),
    })
    const json = await res.json()
    if (!res.ok || !json.success) {
      notify('error', 'AI Order Parser gagal', json.error?.details || json.message)
      return null
    }
    return json.data as AIParseResponse
  }

  async function createOrder(payload: {
    channel: string
    customer_name: string
    customer_phone: string
    shipping_address: string
    courier_code: string
    items: { product_id: string; qty: number }[]
  }): Promise<Order | null> {
    const res = await fetch('/api/v1/orders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        merchant_id: activeMerchantId.value,
        ...payload,
      }),
    })
    const json = await res.json()
    if (!res.ok || !json.success) {
      notify(
        res.status === 409 ? 'warning' : 'error',
        res.status === 409 ? '409 Conflict — Stok Tidak Cukup!' : 'Gagal Membuat Order',
        json.error?.details || json.message,
      )
      await fetchOverview(true)
      return null
    }
    notify('success', `Order ${json.data.order_number} Dibuat!`, 'Stok berhasil dikunci di reserved_stock')
    await fetchOverview(true)
    return json.data as Order
  }

  async function simulatePayOrder(orderId: string) {
    const res = await fetch(`/api/v1/orders/${orderId}/simulate-pay`, { method: 'POST' })
    const json = await res.json()
    if (!res.ok || !json.success) {
      notify('error', 'Gagal memproses webhook pembayaran', json.error?.details || json.message)
      return
    }
    notify(
      'success',
      'Webhook Pembayaran Terverifikasi (HMAC-SHA256)',
      `ACK dalam ${json.data?.processing_ms ?? 2}ms — Worker memproses Waybill di Redis Queue`,
    )
    setTimeout(() => fetchOverview(true), 350)
  }

  async function cancelOrder(orderId: string) {
    const res = await fetch(`/api/v1/orders/${orderId}/cancel`, { method: 'POST' })
    const json = await res.json()
    if (!res.ok || !json.success) {
      notify('error', 'Gagal membatalkan order', json.error?.details || json.message)
      return
    }
    notify('info', 'Order Dibatalkan', 'Reserved stock dikembalikan ke available_stock secara atomik')
    await fetchOverview(true)
  }

  async function setProductStock(productId: string, availableStock: number) {
    const res = await fetch(`/api/v1/products/${productId}/stock`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ available_stock: availableStock }),
    })
    const json = await res.json()
    if (res.ok && json.success) {
      notify('info', `Stok ${json.data.sku} Diperbarui`, `Available Stock diset ke ${availableStock} pcs`)
      await fetchOverview(true)
    }
  }

  async function runRaceConditionSim(productId: string, buyers = 10, resetStockTo = 1) {
    isSimulatingRace.value = true
    try {
      const res = await fetch('/api/v1/simulator/race-condition', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          merchant_id: activeMerchantId.value,
          product_id: productId,
          concurrent_buyers: buyers,
          reset_stock_to: resetStockTo,
        }),
      })
      const json = await res.json()
      if (res.ok && json.success) {
        raceResult.value = json.data as RaceSimulationResult
        notify(
          'success',
          `Anti-Overselling Terbukti! (${json.data.success_count} Menang / ${json.data.conflict_count} Ditolak)`,
          `SKU ${json.data.product_sku}: Available Stock = ${json.data.final_available_stock} (Tidak Minus)`,
        )
        await fetchOverview(true)
      }
    } finally {
      isSimulatingRace.value = false
    }
  }

  async function runDuplicateWebhookSim(orderId = '', count = 5) {
    isSimulatingWebhook.value = true
    try {
      const res = await fetch('/api/v1/simulator/duplicate-webhook', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          merchant_id: activeMerchantId.value,
          order_id: orderId,
          duplicate_count: count,
        }),
      })
      const json = await res.json()
      if (res.ok && json.success) {
        webhookSimResult.value = json.data as DuplicateWebhookSimResult
        notify(
          'success',
          `Idempotency Terbukti! (1 Diproses, ${json.data.duplicate_ignored} Diabaikan)`,
          `Order ${json.data.order_number} aman dari pemotongan stok ganda`,
        )
        setTimeout(() => fetchOverview(true), 300)
      }
    } finally {
      isSimulatingWebhook.value = false
    }
  }

  async function setChaosMode(mode: ChaosFailMode) {
    const res = await fetch('/api/v1/simulator/chaos-mode', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode }),
    })
    const json = await res.json()
    if (res.ok && json.success) {
      queueMetrics.value.chaos_mode = mode
      const labels: Record<ChaosFailMode, string> = {
        OFF: 'Normal (3rd-Party API 200 OK)',
        TRANSIENT_TWICE: 'Simulasi HTTP 429 Sementara (Auto-Recover Retry #3)',
        ALWAYS_429: 'Simulasi HTTP 429 Permanen (Masuk Dead-Letter Queue)',
      }
      notify('warning', 'Mode Simulasi API Pihak Ketiga Diubah', labels[mode])
      await fetchOverview(true)
    }
  }

  async function retryJob(jobId: string) {
    const res = await fetch(`/api/v1/queue/jobs/${jobId}/retry`, { method: 'POST' })
    const json = await res.json()
    if (res.ok && json.success) {
      notify('info', `Manual Retry Job ${jobId}`, 'Job dikembalikan ke Redis Queue untuk diproses ulang')
      setTimeout(() => fetchOverview(true), 350)
    }
  }

  // Core Payment Gateway & Double-Entry Ledger Fetchers
  async function fetchCoreGatewayData() {
    try {
      const [accRes, ledRes, recRes] = await Promise.all([
        fetch('/api/v1/accounts'),
        fetch('/api/v1/ledger'),
        fetch('/api/v1/reconciliation/batches'),
      ])
      const [accJson, ledJson, recJson] = await Promise.all([
        accRes.json(),
        ledRes.json(),
        recRes.json(),
      ])
      if (accJson.success) gatewayAccounts.value = accJson.data || []
      if (ledJson.success) ledgerEntries.value = ledJson.data || []
      if (recJson.success) reconBatches.value = recJson.data || []
    } catch {
      // ignore background error
    }
  }

  return {
    merchants,
    activeMerchantId,
    activeMerchant,
    channelFilter,
    statusFilter,
    products,
    orders,
    webhookEvents,
    errorLogs,
    queueMetrics,
    raceResult,
    webhookSimResult,
    isSimulatingRace,
    isSimulatingWebhook,
    isLoading,
    lastApiLatencyMs,
    totalGMV,
    lowStockCount,
    gatewayAccounts,
    ledgerEntries,
    reconBatches,
    toasts,
    notify,
    fetchOverview,
    parseSocialChat,
    createOrder,
    simulatePayOrder,
    cancelOrder,
    setProductStock,
    runRaceConditionSim,
    runDuplicateWebhookSim,
    setChaosMode,
    retryJob,
    fetchCoreGatewayData,
  }
})

