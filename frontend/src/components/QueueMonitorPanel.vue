<script setup lang="ts">
import { ref } from 'vue'
import { useOmsStore } from '../stores/omsStore'
import type { ChaosFailMode } from '../types/oms'

const store = useOmsStore()
const activeSubTab = ref<'jobs' | 'errors' | 'webhooks'>('jobs')

const chaosOptions: { mode: ChaosFailMode; label: string }[] = [
  { mode: 'OFF', label: 'Normal (200 OK)' },
  { mode: 'TRANSIENT_TWICE', label: 'Transient 429 (Auto-Recover @ Retry #3)' },
  { mode: 'ALWAYS_429', label: 'Simulate 429 Rate Limit -> Force DLQ' },
]

function jobBadge(status: string) {
  switch (status) {
    case 'COMPLETED':
      return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
    case 'RETRYING':
      return 'bg-amber-500/20 text-amber-300 border-amber-500/40 animate-pulse'
    case 'FAILED':
      return 'bg-rose-500/20 text-rose-300 border-rose-500/40'
    default:
      return 'bg-sky-500/20 text-sky-300 border-sky-500/40'
  }
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-slate-900/80 border border-slate-800 overflow-hidden">
    <!-- Header & 3rd-Party Failure Chaos Toggle -->
    <div class="p-4 sm:p-5 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3">
      <div>
        <div class="flex items-center gap-2">
          <span class="px-2 py-0.5 rounded bg-sky-500/15 text-sky-300 font-mono text-[10px] font-bold">
            PANEL 4
          </span>
          <h2 class="text-sm sm:text-base font-bold text-white">
            Redis Background Queue, Exponential Backoff &amp; Dead-Letter Monitor
          </h2>
        </div>
        <p class="text-xs text-slate-400 mt-0.5">
          Worker antrean Waybill Logistik, Sinkronisasi Stok Marketplace &amp; Notifikasi WA (<code>attempts: 4</code>, backoff <code>2s, 4s, 8s, 16s</code>)
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-1.5 bg-slate-950 p-1.5 rounded-xl border border-slate-800">
        <span class="text-[10px] font-bold uppercase text-slate-400 px-2">3rd-Party API Sim:</span>
        <button
          v-for="opt in chaosOptions"
          :key="opt.mode"
          type="button"
          @click="store.setChaosMode(opt.mode)"
          :class="
            store.queueMetrics.chaos_mode === opt.mode
              ? opt.mode === 'OFF'
                ? 'bg-emerald-500 text-slate-950 font-extrabold'
                : 'bg-rose-500 text-white font-extrabold'
              : 'text-slate-400 hover:text-slate-200'
          "
          class="px-2.5 py-1 rounded-lg text-[11px] transition cursor-pointer"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <!-- Queue KPI Counters -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 p-4 bg-slate-950/50 border-b border-slate-800 text-center">
      <div class="p-2.5 rounded-xl bg-slate-900 border border-slate-800">
        <div class="font-mono font-black text-lg text-sky-400">{{ store.queueMetrics.active_count }}</div>
        <div class="text-[11px] text-slate-400">Active / Pending</div>
      </div>
      <div class="p-2.5 rounded-xl bg-slate-900 border border-slate-800">
        <div class="font-mono font-black text-lg text-amber-400">{{ store.queueMetrics.retrying_count }}</div>
        <div class="text-[11px] text-slate-400">Retrying (Exp. Backoff)</div>
      </div>
      <div class="p-2.5 rounded-xl bg-slate-900 border border-slate-800">
        <div class="font-mono font-black text-lg text-emerald-400">{{ store.queueMetrics.completed_count }}</div>
        <div class="text-[11px] text-slate-400">Completed Jobs</div>
      </div>
      <div class="p-2.5 rounded-xl bg-slate-900 border border-slate-800">
        <div class="font-mono font-black text-lg text-rose-400">{{ store.queueMetrics.failed_dlq_count }}</div>
        <div class="text-[11px] text-slate-400">Failed / Dead-Letter (DLQ)</div>
      </div>
    </div>

    <!-- Sub-Navigation Tabs -->
    <div class="px-4 pt-2.5 bg-slate-950/40 border-b border-slate-800 flex items-center gap-3 text-xs">
      <button
        type="button"
        @click="activeSubTab = 'jobs'"
        :class="activeSubTab === 'jobs' ? 'border-emerald-400 text-emerald-400 font-bold' : 'border-transparent text-slate-400'"
        class="pb-2.5 border-b-2 transition cursor-pointer"
      >
        Queue Jobs ({{ store.queueMetrics.recent_jobs?.length || 0 }})
      </button>
      <button
        type="button"
        @click="activeSubTab = 'errors'"
        :class="activeSubTab === 'errors' ? 'border-rose-400 text-rose-400 font-bold' : 'border-transparent text-slate-400'"
        class="pb-2.5 border-b-2 transition cursor-pointer"
      >
        System Error Logs / DLQ ({{ store.errorLogs.length }})
      </button>
      <button
        type="button"
        @click="activeSubTab = 'webhooks'"
        :class="activeSubTab === 'webhooks' ? 'border-indigo-400 text-indigo-400 font-bold' : 'border-transparent text-slate-400'"
        class="pb-2.5 border-b-2 transition cursor-pointer"
      >
        Webhook Audit Trail ({{ store.webhookEvents.length }})
      </button>
    </div>

    <!-- Tab 1: Queue Jobs -->
    <div v-if="activeSubTab === 'jobs'" class="max-h-72 overflow-y-auto custom-scroll divide-y divide-slate-800/70 text-xs">
      <div v-if="!store.queueMetrics.recent_jobs?.length" class="p-6 text-center text-slate-500">
        Belum ada background job. Klik <strong>Bayar (Webhook)</strong> pada salah satu order untuk memicu Waybill &amp; Stock Sync Worker!
      </div>
      <div
        v-for="job in store.queueMetrics.recent_jobs"
        :key="job.id"
        class="px-4 py-3 flex flex-wrap items-center justify-between gap-2 hover:bg-slate-800/30"
      >
        <div class="space-y-0.5">
          <div class="flex items-center gap-2">
            <span class="font-mono font-bold text-white">{{ job.queue_name }}</span>
            <span class="font-mono text-[11px] text-slate-400">({{ job.order_number }})</span>
            <span :class="jobBadge(job.status)" class="px-2 py-0.5 rounded border text-[10px] font-bold">
              {{ job.status === 'FAILED' ? 'FAILED / DLQ' : job.status }}
            </span>
          </div>
          <div class="text-[11px] text-slate-400 font-mono">
            Job ID: {{ job.id }} &bull; Attempts: {{ job.attempts }}/{{ job.max_attempts }}
          </div>
          <div v-if="job.last_error" class="text-[11px] text-rose-400 font-mono">
            Error: {{ job.last_error }}
          </div>
        </div>

        <div v-if="job.status === 'FAILED'">
          <button
            type="button"
            @click="store.retryJob(job.id)"
            class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-extrabold text-[11px] cursor-pointer"
          >
            Manual Retry
          </button>
        </div>
      </div>
    </div>

    <!-- Tab 2: System Error Logs -->
    <div v-else-if="activeSubTab === 'errors'" class="max-h-72 overflow-y-auto custom-scroll divide-y divide-slate-800/70 text-xs">
      <div v-if="store.errorLogs.length === 0" class="p-6 text-center text-slate-500">
        Tidak ada error DLQ di <code>system_error_logs</code>.
      </div>
      <div v-for="err in store.errorLogs" :key="err.id" class="px-4 py-3 space-y-1">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 font-mono text-[10px] font-bold">
              {{ err.error_code }}
            </span>
            <span class="font-mono text-white font-semibold">{{ err.queue_name }}</span>
            <span class="font-mono text-slate-400 text-[11px]">Ref: {{ err.reference_id }}</span>
          </div>
          <span
            :class="err.resolved ? 'text-emerald-400' : 'text-rose-400'"
            class="font-mono text-[11px] font-bold"
          >
            {{ err.resolved ? 'RESOLVED' : `FAILED (${err.attempts}x)` }}
          </span>
        </div>
        <div class="font-mono text-[11px] text-slate-300">{{ err.error_message }}</div>
      </div>
    </div>

    <!-- Tab 3: Webhook Audit Trail -->
    <div v-else class="max-h-72 overflow-y-auto custom-scroll divide-y divide-slate-800/70 text-xs">
      <div v-if="store.webhookEvents.length === 0" class="p-6 text-center text-slate-500">
        Belum ada log di <code>webhook_events</code>.
      </div>
      <div
        v-for="ev in store.webhookEvents"
        :key="ev.id"
        class="px-4 py-2.5 flex items-center justify-between gap-2 font-mono text-[11px]"
      >
        <div>
          <span class="text-indigo-300 font-bold">{{ ev.provider }}</span>
          <span class="text-slate-500 mx-1.5">/</span>
          <span class="text-white">{{ ev.external_event_id }}</span>
          <span class="text-slate-400 ml-2">({{ ev.event_type }})</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-emerald-400">{{ ev.processing_ms }}ms</span>
          <span class="px-2 py-0.5 rounded bg-slate-800 text-slate-200 text-[10px] font-bold">
            {{ ev.status }}
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

