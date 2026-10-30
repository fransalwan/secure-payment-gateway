<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { Zap, ShieldCheck, Lock } from '@lucide/vue'

const store = useOmsStore()
const selectedProductId = ref<string>('')

watch(
  () => store.products,
  (list) => {
    const first = list[0]
    if (first && (!selectedProductId.value || !list.some((p) => p.id === selectedProductId.value))) {
      selectedProductId.value = first.id
    }
  },
  { immediate: true },
)

const maxRaceLatency = computed(() => {
  const list = store.raceResult?.attempts || []
  if (!list.length) return 10
  return Math.max(1, ...list.map((a) => a.latency_ms))
})

function barWidth(ms: number) {
  return `${Math.max(8, Math.min(100, Math.round((ms / maxRaceLatency.value) * 100)))}%`
}

async function triggerRaceSim() {
  const targetId = selectedProductId.value || store.products[0]?.id || ''
  await store.runRaceConditionSim(targetId, 10, 1)
}

async function triggerWebhookSim() {
  await store.runDuplicateWebhookSim('', 5)
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-card border border-border shadow-xs overflow-hidden">
    <div class="p-4 sm:p-5 border-b border-border flex flex-wrap items-center justify-between gap-2">
      <div>
        <div class="flex items-center gap-2">
          <Badge variant="amber">
            <Zap class="w-3 h-3" />
            <span>CONCURRENCY &amp; WEBHOOK LAB</span>
          </Badge>
          <h2 id="stress-lab-heading" tabindex="-1" class="text-sm sm:text-base font-bold text-foreground focus:outline-none">
            Concurrency &amp; Idempotent Webhook Stress-Test Lab
          </h2>
        </div>
        <p class="text-xs text-muted-foreground mt-0.5">
          Pembuktian langsung PostgreSQL <code>SELECT FOR UPDATE</code> Anti-Overselling &amp; <code>HMAC-SHA256</code> Webhook Deduplication
        </p>
      </div>
    </div>

    <div class="p-4 sm:p-5 grid grid-cols-1 xl:grid-cols-2 gap-4">
      <!-- Left: 10 Concurrent Buyers for 1 Last Stock -->
      <div class="rounded-2xl bg-elevated/45 border border-border p-4 flex flex-col justify-between space-y-4">
        <div class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <h3 class="text-xs font-extrabold uppercase tracking-wider text-tone-amber flex items-center gap-1.5">
              <Lock class="w-3.5 h-3.5" />
              <span>1. Flash-Sale Anti-Overselling Test</span>
            </h3>
            <span class="font-mono text-[10px] text-muted-foreground">10 Goroutines // 1 Stok</span>
          </div>
          <p class="text-xs text-muted-foreground leading-relaxed">
            Mereset stok SKU terpilih menjadi <strong class="text-foreground">1 pcs</strong>, lalu menembakkan <strong class="text-foreground">10 request checkout paralel</strong> pada mikrodetik yang sama dari kanal Web, Shopee, TikTok &amp; WA.
          </p>

          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-1">
            <select
              v-model="selectedProductId"
              aria-label="Pilih SKU target simulasi race condition"
              class="flex-1 rounded-xl bg-card border border-border px-3 py-2 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option v-for="p in store.products" :key="p.id" :value="p.id">
                {{ p.sku }} — {{ p.name }} (Stok: {{ p.available_stock }})
              </option>
            </select>

            <Button
              variant="amber"
              :disabled="store.isSimulatingRace"
              class="w-full sm:w-auto shrink-0"
              @click="triggerRaceSim"
            >
              <Zap class="w-3.5 h-3.5" />
              <span>{{ store.isSimulatingRace ? 'Menembak 10 Buyer...' : 'Simulate 10 Concurrent Buyers' }}</span>
            </Button>
          </div>
        </div>

        <!-- Race Result Matrix + Microsecond Lock Waterfall -->
        <div v-if="store.raceResult" class="space-y-3 pt-3 border-t border-border">
          <div class="grid grid-cols-3 gap-2 text-center text-xs tabular-nums">
            <div class="p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/30">
              <div class="font-mono font-black text-base text-tone-emerald">{{ store.raceResult.success_count }}</div>
              <div class="text-[10px] font-medium text-muted-foreground">201 Created (Menang)</div>
            </div>
            <div class="p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/30">
              <div class="font-mono font-black text-base text-tone-rose">{{ store.raceResult.conflict_count }}</div>
              <div class="text-[10px] font-medium text-muted-foreground">409 Conflict (Ditolak)</div>
            </div>
            <div class="p-2.5 rounded-xl bg-indigo-500/10 border border-indigo-500/30">
              <div class="font-mono font-black text-base text-tone-indigo">
                {{ store.raceResult.final_available_stock }} pcs
              </div>
              <div class="text-[10px] font-medium text-muted-foreground">Sisa Stok (Anti-Minus)</div>
            </div>
          </div>

          <div class="max-h-56 overflow-y-auto custom-scroll rounded-xl bg-card border border-border divide-y divide-border text-[11px] font-mono">
            <div
              v-for="att in store.raceResult.attempts"
              :key="att.buyer_index"
              class="px-3 py-2 space-y-1"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2 min-w-0">
                  <Badge :variant="att.http_status === 201 ? 'emerald' : 'rose'">
                    {{ att.http_status }}
                  </Badge>
                  <span class="text-foreground font-semibold truncate">{{ att.buyer_name }}</span>
                  <span class="text-muted-foreground text-[10px]">[{{ att.channel }}]</span>
                </div>
                <span
                  :class="att.http_status === 201 ? 'text-tone-emerald font-bold' : 'text-muted-foreground'"
                  class="tabular-nums shrink-0"
                >
                  {{ att.order_number ? `WON -> ${att.order_number}` : 'Locked Out' }} ({{ att.latency_ms.toFixed(1) }}ms)
                </span>
              </div>
              <!-- Microsecond Lock Waterfall Bar -->
              <div class="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                <div
                  :class="att.http_status === 201 ? 'bg-emerald-500' : 'bg-rose-500/60'"
                  class="h-full rounded-full transition-all duration-500"
                  :style="{ width: barWidth(att.latency_ms) }"
                ></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: 5x Duplicate Payment Webhook -->
      <div class="rounded-2xl bg-elevated/45 border border-border p-4 flex flex-col justify-between space-y-4">
        <div class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <h3 class="text-xs font-extrabold uppercase tracking-wider text-tone-indigo flex items-center gap-1.5">
              <ShieldCheck class="w-3.5 h-3.5" />
              <span>2. 5x Duplicate Payment Webhook Test</span>
            </h3>
            <span class="font-mono text-[10px] text-muted-foreground">HMAC-SHA256 + Idempotency</span>
          </div>
          <p class="text-xs text-muted-foreground leading-relaxed">
            Menembakkan <strong class="text-foreground">5 webhook <code>payment.succeeded</code> identik</strong> dengan <code>X-Webhook-Id</code> yang sama ke <code>POST /api/v1/webhooks/payment</code> untuk membuktikan respons <code>&lt; 50ms</code> &amp; proteksi pemotongan stok ganda.
          </p>

          <div class="pt-1 flex justify-end">
            <Button
              variant="indigo"
              :disabled="store.isSimulatingWebhook"
              class="w-full sm:w-auto"
              @click="triggerWebhookSim"
            >
              <ShieldCheck class="w-3.5 h-3.5" />
              <span>{{ store.isSimulatingWebhook ? 'Menembak 5x Webhook...' : 'Simulate 5x Duplicate Payment Webhook' }}</span>
            </Button>
          </div>
        </div>

        <!-- Webhook Result Matrix -->
        <div v-if="store.webhookSimResult" class="space-y-3 pt-3 border-t border-border">
          <div class="grid grid-cols-3 gap-2 text-center text-xs tabular-nums">
            <div class="p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/30">
              <div class="font-mono font-black text-base text-tone-emerald">
                {{ store.webhookSimResult.processed_count }}
              </div>
              <div class="text-[10px] font-medium text-muted-foreground">PROCESSED (1x)</div>
            </div>
            <div class="p-2.5 rounded-xl bg-amber-500/10 border border-amber-500/30">
              <div class="font-mono font-black text-base text-tone-amber">
                {{ store.webhookSimResult.duplicate_ignored }}
              </div>
              <div class="text-[10px] font-medium text-muted-foreground">DUPLICATE_IGNORED</div>
            </div>
            <div class="p-2.5 rounded-xl bg-sky-500/10 border border-sky-500/30">
              <div class="font-mono font-black text-xs text-tone-sky truncate pt-1">
                {{ store.webhookSimResult.order_number }}
              </div>
              <div class="text-[10px] font-medium text-muted-foreground">{{ store.webhookSimResult.order_final_status }}</div>
            </div>
          </div>

          <div class="max-h-56 overflow-y-auto custom-scroll rounded-xl bg-card border border-border divide-y divide-border text-[11px] font-mono">
            <div
              v-for="w in store.webhookSimResult.attempts"
              :key="w.attempt_index"
              class="px-3 py-2 flex items-center justify-between gap-2"
            >
              <div class="flex items-center gap-2 min-w-0">
                <Badge variant="outline">#{{ w.attempt_index }}</Badge>
                <span
                  :class="w.status_result === 'accepted' ? 'text-tone-emerald font-bold' : 'text-tone-amber font-medium'"
                  class="truncate"
                >
                  {{ w.status_result === 'accepted' ? 'PROCESSED -> QUEUED' : 'IDEMPOTENT_SKIPPED (duplicate_ignored)' }}
                </span>
              </div>
              <span class="text-muted-foreground tabular-nums shrink-0">{{ w.latency_ms.toFixed(1) }}ms</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
