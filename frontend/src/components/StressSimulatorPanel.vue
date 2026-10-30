<script setup lang="ts">
import { ref, watch } from 'vue'
import { useOmsStore } from '../stores/omsStore'

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

async function triggerRaceSim() {
  const targetId = selectedProductId.value || store.products[0]?.id || ''
  await store.runRaceConditionSim(targetId, 10, 1)
}

async function triggerWebhookSim() {
  await store.runDuplicateWebhookSim('', 5)
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-slate-900/80 border border-slate-800 overflow-hidden">
    <div class="p-4 sm:p-5 border-b border-slate-800 flex flex-wrap items-center justify-between gap-2">
      <div>
        <div class="flex items-center gap-2">
          <span class="px-2 py-0.5 rounded bg-amber-500/20 text-amber-300 font-mono text-[10px] font-bold">
            PANEL 3 &bull; LIVE DEMO SIMULATOR
          </span>
          <h2 class="text-sm sm:text-base font-bold text-white">
            Concurrency &amp; Idempotent Webhook Stress-Test Lab
          </h2>
        </div>
        <p class="text-xs text-slate-400 mt-0.5">
          Pembuktian langsung PostgreSQL <code>SELECT FOR UPDATE</code> Anti-Overselling &amp; <code>HMAC-SHA256</code> Webhook Deduplication
        </p>
      </div>
    </div>

    <div class="p-4 sm:p-5 grid grid-cols-1 lg:grid-cols-2 gap-4">
      <!-- Left: 10 Concurrent Buyers for 1 Last Stock -->
      <div class="rounded-xl bg-slate-950/80 border border-slate-800 p-4 flex flex-col justify-between space-y-3">
        <div class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <h3 class="text-xs font-extrabold uppercase tracking-wider text-amber-400">
              1. Flash-Sale Anti-Overselling Test
            </h3>
            <span class="font-mono text-[10px] text-slate-400">10 Goroutines // 1 Stok Terakhir</span>
          </div>
          <p class="text-xs text-slate-400 leading-relaxed">
            Mereset stok SKU terpilih menjadi <strong>1 pcs</strong>, lalu menembakkan <strong>10 request checkout paralel</strong> pada mikrodetik yang sama dari kanal Web, Shopee, TikTok &amp; WA.
          </p>

          <div class="flex flex-wrap items-center gap-2 pt-1">
            <select
              v-model="selectedProductId"
              aria-label="Pilih SKU target simulasi race condition"
              class="flex-1 min-w-[180px] rounded-xl bg-slate-900 border border-slate-700 px-3 py-2 text-xs text-slate-200"
            >
              <option v-for="p in store.products" :key="p.id" :value="p.id">
                {{ p.sku }} — {{ p.name }} (Stok: {{ p.available_stock }})
              </option>
            </select>

            <button
              type="button"
              :disabled="store.isSimulatingRace"
              @click="triggerRaceSim"
              class="px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 disabled:opacity-50 text-slate-950 font-extrabold text-xs transition cursor-pointer shrink-0"
            >
              {{ store.isSimulatingRace ? 'Menembak 10 Buyer...' : 'Simulate 10 Concurrent Buyers' }}
            </button>
          </div>
        </div>

        <!-- Race Result Matrix -->
        <div v-if="store.raceResult" class="space-y-2.5 pt-2 border-t border-slate-800">
          <div class="grid grid-cols-3 gap-2 text-center text-xs">
            <div class="p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/30">
              <div class="font-mono font-black text-base text-emerald-400">{{ store.raceResult.success_count }}</div>
              <div class="text-[10px] text-emerald-200">201 Created (Menang)</div>
            </div>
            <div class="p-2 rounded-lg bg-rose-500/10 border border-rose-500/30">
              <div class="font-mono font-black text-base text-rose-400">{{ store.raceResult.conflict_count }}</div>
              <div class="text-[10px] text-rose-200">409 Conflict (Ditolak)</div>
            </div>
            <div class="p-2 rounded-lg bg-indigo-500/10 border border-indigo-500/30">
              <div class="font-mono font-black text-base text-indigo-300">
                {{ store.raceResult.final_available_stock }} pcs
              </div>
              <div class="text-[10px] text-indigo-200">Sisa Stok (Anti-Minus)</div>
            </div>
          </div>

          <div class="max-h-44 overflow-y-auto custom-scroll rounded-lg bg-slate-900 border border-slate-800 divide-y divide-slate-800/70 text-[11px] font-mono">
            <div
              v-for="att in store.raceResult.attempts"
              :key="att.buyer_index"
              class="px-3 py-1.5 flex items-center justify-between gap-2"
            >
              <div class="flex items-center gap-2">
                <span
                  :class="att.http_status === 201 ? 'bg-emerald-500 text-slate-950' : 'bg-rose-500/20 text-rose-300'"
                  class="px-1.5 py-0.2 rounded font-bold text-[10px]"
                >
                  {{ att.http_status }}
                </span>
                <span class="text-slate-200">{{ att.buyer_name }}</span>
                <span class="text-slate-500 text-[10px]">[{{ att.channel }}]</span>
              </div>
              <span :class="att.http_status === 201 ? 'text-emerald-400 font-bold' : 'text-slate-400'" class="truncate">
                {{ att.order_number ? `WON -> ${att.order_number}` : 'Locked Out (Insufficient Stock)' }}
                ({{ att.latency_ms.toFixed(1) }}ms)
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: 5x Duplicate Payment Webhook -->
      <div class="rounded-xl bg-slate-950/80 border border-slate-800 p-4 flex flex-col justify-between space-y-3">
        <div class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <h3 class="text-xs font-extrabold uppercase tracking-wider text-indigo-400">
              2. 5x Duplicate Payment Webhook Test
            </h3>
            <span class="font-mono text-[10px] text-slate-400">HMAC-SHA256 + Idempotency Key</span>
          </div>
          <p class="text-xs text-slate-400 leading-relaxed">
            Menembakkan <strong>5 webhook <code>payment.succeeded</code> identik</strong> dengan <code>X-Webhook-Id</code> yang sama ke <code>POST /api/v1/webhooks/payment</code> untuk membuktikan respons <code>&lt; 50ms</code> &amp; proteksi pemotongan stok ganda.
          </p>

          <div class="pt-1 flex justify-end">
            <button
              type="button"
              :disabled="store.isSimulatingWebhook"
              @click="triggerWebhookSim"
              class="w-full sm:w-auto px-4 py-2 rounded-xl bg-indigo-500 hover:bg-indigo-400 disabled:opacity-50 text-white font-extrabold text-xs transition cursor-pointer"
            >
              {{ store.isSimulatingWebhook ? 'Menembak 5x Webhook...' : 'Simulate 5x Duplicate Payment Webhook' }}
            </button>
          </div>
        </div>

        <!-- Webhook Result Matrix -->
        <div v-if="store.webhookSimResult" class="space-y-2.5 pt-2 border-t border-slate-800">
          <div class="grid grid-cols-3 gap-2 text-center text-xs">
            <div class="p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/30">
              <div class="font-mono font-black text-base text-emerald-400">
                {{ store.webhookSimResult.processed_count }}
              </div>
              <div class="text-[10px] text-emerald-200">PROCESSED (1x)</div>
            </div>
            <div class="p-2 rounded-lg bg-amber-500/10 border border-amber-500/30">
              <div class="font-mono font-black text-base text-amber-300">
                {{ store.webhookSimResult.duplicate_ignored }}
              </div>
              <div class="text-[10px] text-amber-200">DUPLICATE_IGNORED</div>
            </div>
            <div class="p-2 rounded-lg bg-sky-500/10 border border-sky-500/30">
              <div class="font-mono font-black text-xs text-sky-300 truncate pt-1">
                {{ store.webhookSimResult.order_number }}
              </div>
              <div class="text-[10px] text-sky-200">{{ store.webhookSimResult.order_final_status }}</div>
            </div>
          </div>

          <div class="max-h-44 overflow-y-auto custom-scroll rounded-lg bg-slate-900 border border-slate-800 divide-y divide-slate-800/70 text-[11px] font-mono">
            <div
              v-for="w in store.webhookSimResult.attempts"
              :key="w.attempt_index"
              class="px-3 py-1.5 flex items-center justify-between gap-2"
            >
              <div class="flex items-center gap-2">
                <span class="px-1.5 py-0.2 rounded bg-slate-800 text-slate-300 text-[10px]">
                  #{{ w.attempt_index }}
                </span>
                <span
                  :class="w.status_result === 'accepted' ? 'text-emerald-400 font-bold' : 'text-amber-300'"
                >
                  {{ w.status_result === 'accepted' ? 'PROCESSED -> QUEUED' : 'IDEMPOTENT_SKIPPED (duplicate_ignored)' }}
                </span>
              </div>
              <span class="text-slate-400">{{ w.latency_ms.toFixed(1) }}ms</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

