<script setup lang="ts">
import { ref } from 'vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import type { ChaosFailMode } from '../types/oms'
import { Activity, RotateCcw } from '@lucide/vue'

const store = useOmsStore()
const activeSubTab = ref<'jobs' | 'errors' | 'webhooks'>('jobs')

const chaosOptions: { mode: ChaosFailMode; label: string }[] = [
  { mode: 'OFF', label: 'Normal (200 OK)' },
  { mode: 'TRANSIENT_TWICE', label: 'Transient 429 (Recover #3)' },
  { mode: 'ALWAYS_429', label: 'Simulate 429 -> Force DLQ' },
]

function jobVariant(status: string): 'emerald' | 'amber' | 'rose' | 'sky' {
  switch (status) {
    case 'COMPLETED':
      return 'emerald'
    case 'RETRYING':
      return 'amber'
    case 'FAILED':
      return 'rose'
    default:
      return 'sky'
  }
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-card border border-border shadow-xs overflow-hidden">
    <!-- Header & 3rd-Party Failure Chaos Toggle -->
    <div class="p-4 sm:p-5 border-b border-border flex flex-col lg:flex-row lg:items-center justify-between gap-3">
      <div>
        <div class="flex items-center gap-2">
          <Badge variant="sky">
            <Activity class="w-3 h-3" />
            <span>REDIS QUEUE &amp; DLQ</span>
          </Badge>
          <h2 id="queue-monitor-heading" tabindex="-1" class="text-sm sm:text-base font-bold text-foreground focus:outline-none">
            Redis Background Queue, Exponential Backoff &amp; Dead-Letter Monitor
          </h2>
        </div>
        <p class="text-xs text-muted-foreground mt-0.5">
          Worker antrean Waybill Logistik, Sinkronisasi Stok Marketplace &amp; Notifikasi WA (<code>attempts: 4</code>, backoff <code>2s, 4s, 8s, 16s</code>)
        </p>
      </div>

      <!-- Horizontal scrollable chaos pills on mobile -->
      <div class="flex items-center gap-1.5 overflow-x-auto no-scrollbar bg-elevated p-1.5 rounded-xl border border-border">
        <span class="text-[10px] font-bold uppercase text-muted-foreground px-1.5 shrink-0">API Sim:</span>
        <button
          v-for="opt in chaosOptions"
          :key="opt.mode"
          type="button"
          @click="store.setChaosMode(opt.mode)"
          :class="
            store.queueMetrics.chaos_mode === opt.mode
              ? opt.mode === 'OFF'
                ? 'bg-emerald-600 text-white font-bold shadow-xs'
                : 'bg-rose-600 text-white font-bold shadow-xs'
              : 'text-muted-foreground hover:text-foreground'
          "
          class="shrink-0 px-2.5 py-1.5 rounded-lg text-[11px] transition cursor-pointer"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <!-- Queue KPI Counters -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 p-4 bg-elevated/35 border-b border-border text-center tabular-nums">
      <div class="p-3 rounded-xl bg-card border border-border">
        <div class="font-mono font-black text-lg text-tone-sky">{{ store.queueMetrics.active_count }}</div>
        <div class="text-[11px] text-muted-foreground">Active / Pending</div>
      </div>
      <div class="p-3 rounded-xl bg-card border border-border">
        <div class="font-mono font-black text-lg text-tone-amber">{{ store.queueMetrics.retrying_count }}</div>
        <div class="text-[11px] text-muted-foreground">Retrying (Exp. Backoff)</div>
      </div>
      <div class="p-3 rounded-xl bg-card border border-border">
        <div class="font-mono font-black text-lg text-tone-emerald">{{ store.queueMetrics.completed_count }}</div>
        <div class="text-[11px] text-muted-foreground">Completed Jobs</div>
      </div>
      <div class="p-3 rounded-xl bg-card border border-border">
        <div class="font-mono font-black text-lg text-tone-rose">{{ store.queueMetrics.failed_dlq_count }}</div>
        <div class="text-[11px] text-muted-foreground">Failed / DLQ</div>
      </div>
    </div>

    <!-- Sub-Navigation Tabs (Scrollable on Mobile) -->
    <div class="px-4 pt-2.5 bg-elevated/25 border-b border-border flex items-center gap-4 overflow-x-auto no-scrollbar text-xs">
      <button
        type="button"
        @click="activeSubTab = 'jobs'"
        :class="activeSubTab === 'jobs' ? 'border-emerald-500 text-tone-emerald font-bold' : 'border-transparent text-muted-foreground hover:text-foreground'"
        class="shrink-0 pb-2.5 border-b-2 transition cursor-pointer"
      >
        Queue Jobs ({{ store.queueMetrics.recent_jobs?.length || 0 }})
      </button>
      <button
        type="button"
        @click="activeSubTab = 'errors'"
        :class="activeSubTab === 'errors' ? 'border-rose-500 text-tone-rose font-bold' : 'border-transparent text-muted-foreground hover:text-foreground'"
        class="shrink-0 pb-2.5 border-b-2 transition cursor-pointer"
      >
        System Error Logs / DLQ ({{ store.errorLogs.length }})
      </button>
      <button
        type="button"
        @click="activeSubTab = 'webhooks'"
        :class="activeSubTab === 'webhooks' ? 'border-indigo-500 text-tone-indigo font-bold' : 'border-transparent text-muted-foreground hover:text-foreground'"
        class="shrink-0 pb-2.5 border-b-2 transition cursor-pointer"
      >
        Webhook Audit Trail ({{ store.webhookEvents.length }})
      </button>
    </div>

    <!-- Tab 1: Queue Jobs -->
    <div v-if="activeSubTab === 'jobs'" class="max-h-80 overflow-y-auto custom-scroll divide-y divide-border text-xs">
      <div v-if="!store.queueMetrics.recent_jobs?.length" class="p-8 text-center text-muted-foreground">
        Belum ada background job. Klik <strong class="text-foreground">Bayar (Webhook)</strong> pada salah satu order untuk memicu Waybill &amp; Stock Sync Worker!
      </div>
      <div
        v-for="job in store.queueMetrics.recent_jobs"
        :key="job.id"
        class="px-4 py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 hover:bg-elevated/40"
      >
        <div class="space-y-1 min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-mono font-bold text-foreground">{{ job.queue_name }}</span>
            <span class="font-mono text-[11px] text-muted-foreground">({{ job.order_number }})</span>
            <Badge :variant="jobVariant(job.status)" :class="job.status === 'RETRYING' ? 'animate-pulse' : ''">
              {{ job.status === 'FAILED' ? 'FAILED / DLQ' : job.status }}
            </Badge>
          </div>
          <div class="text-[11px] text-muted-foreground font-mono tabular-nums">
            Job ID: {{ job.id }} &bull; Attempts: {{ job.attempts }}/{{ job.max_attempts }}
          </div>
          <div v-if="job.last_error" class="text-[11px] text-tone-rose font-mono break-words">
            Error: {{ job.last_error }}
          </div>
        </div>

        <div v-if="job.status === 'FAILED'" class="shrink-0">
          <Button variant="emerald" size="sm" class="w-full sm:w-auto" @click="store.retryJob(job.id)">
            <RotateCcw class="w-3.5 h-3.5" />
            <span>Manual Retry</span>
          </Button>
        </div>
      </div>
    </div>

    <!-- Tab 2: System Error Logs -->
    <div v-else-if="activeSubTab === 'errors'" class="max-h-80 overflow-y-auto custom-scroll divide-y divide-border text-xs">
      <div v-if="store.errorLogs.length === 0" class="p-8 text-center text-muted-foreground">
        Tidak ada error DLQ di <code>system_error_logs</code>.
      </div>
      <div v-for="err in store.errorLogs" :key="err.id" class="px-4 py-3 space-y-1">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="rose" class="font-mono">{{ err.error_code }}</Badge>
            <span class="font-mono text-foreground font-semibold">{{ err.queue_name }}</span>
            <span class="font-mono text-muted-foreground text-[11px]">Ref: {{ err.reference_id }}</span>
          </div>
          <span
            :class="err.resolved ? 'text-tone-emerald' : 'text-tone-rose'"
            class="font-mono text-[11px] font-bold"
          >
            {{ err.resolved ? 'RESOLVED' : `FAILED (${err.attempts}x)` }}
          </span>
        </div>
        <div class="font-mono text-[11px] text-secondary-foreground break-words">{{ err.error_message }}</div>
      </div>
    </div>

    <!-- Tab 3: Webhook Audit Trail -->
    <div v-else class="max-h-80 overflow-y-auto custom-scroll divide-y divide-border text-xs">
      <div v-if="store.webhookEvents.length === 0" class="p-8 text-center text-muted-foreground">
        Belum ada log di <code>webhook_events</code>.
      </div>
      <div
        v-for="ev in store.webhookEvents"
        :key="ev.id"
        class="px-4 py-2.5 flex flex-wrap items-center justify-between gap-2 font-mono text-[11px]"
      >
        <div class="min-w-0">
          <span class="text-tone-indigo font-bold">{{ ev.provider }}</span>
          <span class="text-muted-foreground mx-1.5">/</span>
          <span class="text-foreground">{{ ev.external_event_id }}</span>
          <span class="text-muted-foreground ml-2">({{ ev.event_type }})</span>
        </div>
        <div class="flex items-center gap-2 tabular-nums">
          <span class="text-tone-emerald font-semibold">{{ ev.processing_ms }}ms</span>
          <Badge variant="default">{{ ev.status }}</Badge>
        </div>
      </div>
    </div>
  </section>
</template>
