export type OrderStatus =
  | 'PENDING_PAYMENT'
  | 'PAID'
  | 'READY_TO_SHIP'
  | 'SHIPPED'
  | 'COMPLETED'
  | 'CANCELLED'

export type OrderChannel = 'WEB' | 'WHATSAPP_LIVE' | 'SHOPEE' | 'TIKTOK_SHOP'

export type ChaosFailMode = 'OFF' | 'TRANSIENT_TWICE' | 'ALWAYS_429'

export interface Merchant {
  id: string
  code: string
  name: string
  city: string
  phone: string
}

export interface Product {
  id: string
  merchant_id: string
  sku: string
  name: string
  price: number
  available_stock: number
  reserved_stock: number
  sold_stock: number
  version: number
  updated_at: string
}

export interface OrderItem {
  id: string
  order_id: string
  product_id: string
  sku: string
  product_name: string
  qty: number
  unit_price: number
  subtotal: number
}

export interface Order {
  id: string
  merchant_id: string
  order_number: string
  channel: OrderChannel
  external_order_id?: string
  customer_name: string
  customer_phone: string
  shipping_address: string
  courier_code: string
  status: OrderStatus
  total_amount: number
  waybill_number?: string
  payment_url?: string
  payment_expires_at: string
  paid_at?: string
  created_at: string
  items?: OrderItem[]
}

export interface AIParsedItem {
  sku_id: string
  sku: string
  matched_product_name: string
  qty: number
  unit_price: number
  subtotal: number
  available_stock: number
  stock_sufficient: boolean
  stock_warning?: string
}

export interface AIParseResponse {
  customer_name: string
  customer_phone: string
  shipping_address: string
  courier_code: string
  items: AIParsedItem[]
  estimated_total: number
  has_insufficient_stock: boolean
  parser_engine: string
  latency_ms: number
}

export interface WebhookEvent {
  id: string
  provider: string
  external_event_id: string
  event_type: string
  status: string
  retry_count: number
  processing_ms: number
  error_message?: string
  created_at: string
}

export interface QueueJob {
  id: string
  queue_name: string
  order_id: string
  order_number: string
  status: 'PENDING' | 'ACTIVE' | 'RETRYING' | 'COMPLETED' | 'FAILED'
  attempts: number
  max_attempts: number
  last_error?: string
  next_retry_at?: string
  created_at: string
  updated_at: string
}

export interface QueueMetrics {
  active_count: number
  retrying_count: number
  completed_count: number
  failed_dlq_count: number
  chaos_mode: ChaosFailMode
  queue_backend: string
  recent_jobs: QueueJob[]
}

export interface SystemErrorLog {
  id: string
  service_name: string
  queue_name: string
  job_id: string
  reference_id?: string
  error_code: string
  error_message: string
  attempts: number
  resolved: boolean
  created_at: string
}

export interface RaceBuyerAttempt {
  buyer_index: number
  buyer_name: string
  channel: string
  http_status: number
  outcome: string
  order_number?: string
  reason: string
  latency_ms: number
}

export interface RaceSimulationResult {
  product_sku: string
  product_name: string
  initial_stock: number
  concurrent_buyers: number
  success_count: number
  conflict_count: number
  final_available_stock: number
  final_reserved_stock: number
  zero_oversell_proven: boolean
  total_duration_ms: number
  attempts: RaceBuyerAttempt[]
}

export interface WebhookAttempt {
  attempt_index: number
  webhook_id: string
  http_status: number
  status_result: string
  latency_ms: number
  signature_used: string
}

export interface DuplicateWebhookSimResult {
  order_id: string
  order_number: string
  external_event_id: string
  total_sent: number
  processed_count: number
  duplicate_ignored: number
  idempotency_proven: boolean
  order_final_status: OrderStatus
  attempts: WebhookAttempt[]
}

export interface DashboardOverview {
  merchants: Merchant[]
  active_merchant?: Merchant
  products: Product[]
  orders: Order[]
  webhook_events: WebhookEvent[]
  queue_metrics: QueueMetrics
  error_logs: SystemErrorLog[]
}

// Core Payment Gateway Ledger Types
export interface GatewayAccount {
  id: string
  account_number: string
  holder_name: string
  identity_number?: string
  balance: number
  currency: string
  version: number
  created_at: string
}

export interface LedgerEntry {
  id: string
  transaction_id: string
  account_id: string
  entry_type: 'DEBIT' | 'CREDIT'
  amount: number
  balance_after: number
  created_at: string
}

export interface ReconciliationBatch {
  id: string
  batch_number: string
  bank_name: string
  total_records: number
  matched_count: number
  discrepancy_count: number
  missing_in_gateway_count: number
  total_bank_amount: number
  total_gateway_amount: number
  status: string
  created_at: string
}

