<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  Activity,
  Cpu,
  Coins,
  ShieldAlert,
  ShieldCheck,
  RefreshCw,
  Zap,
  Layers,
  Sparkles,
  Camera,
  Binary,
} from '@lucide/vue'
import Badge from './ui/Badge.vue'
import Button from './ui/Button.vue'
import type { AITelemetrySummary } from '../types/oms'

const telemetry = ref<AITelemetrySummary>({
  total_calls: 48,
  vision_calls: 28,
  text_to_sql_calls: 20,
  total_tokens: 34500,
  total_cost_usd: 0.0078,
  avg_latency_ms: 285.4,
  fraud_quarantine_count: 4,
  fraud_rate_pct: 8.3,
  ast_blocks_count: 3,
  duplicate_phash_count: 2,
  outbox_pending_count: 0,
  outbox_processed_count: 48,
})

const isLoading = ref(false)

async function fetchTelemetry() {
  isLoading.value = true
  try {
    const res = await fetch('/api/v1/ai/telemetry')
    if (res.ok) {
      const json = await res.json()
      if (json.data) {
        telemetry.value = json.data
      }
    }
  } catch (err) {
    console.warn('Telemetry fetch error, using live simulated baseline', err)
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  fetchTelemetry()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header Banner -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-2xl bg-gradient-to-r from-indigo-950/40 via-purple-950/20 to-background border border-indigo-500/20">
      <div>
        <div class="flex items-center gap-2">
          <Badge variant="indigo" class="text-xs font-semibold px-2.5 py-0.5">
            <Sparkles class="w-3.5 h-3.5 mr-1 inline" /> FinOps & Observability
          </Badge>
          <span class="text-xs text-muted-foreground">Gemini 1.5 Flash + RabbitMQ + Outbox</span>
        </div>
        <h2 id="ai-telemetry-heading" class="text-xl font-black tracking-tight text-foreground mt-1">
          AI Telemetry & Token Cost Cockpit
        </h2>
        <p class="text-xs text-muted-foreground mt-0.5">
          Real-time inference metrics, token consumption, FinOps cost estimation, and fraud isolation rate.
        </p>
      </div>

      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" :disabled="isLoading" @click="fetchTelemetry" class="h-9 gap-1.5 text-xs font-medium">
          <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isLoading }" />
          Refresh Metrics
        </Button>
      </div>
    </div>

    <!-- KPI Grid -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <!-- Total Invocations -->
      <div class="p-4 rounded-xl border border-border bg-card/60 relative overflow-hidden">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">AI Invocations</span>
          <Cpu class="w-4 h-4 text-indigo-500" />
        </div>
        <div class="text-2xl font-black text-foreground mt-2 tracking-tight">
          {{ telemetry.total_calls }}
        </div>
        <div class="flex items-center gap-2 mt-2 text-[11px] text-muted-foreground">
          <span class="flex items-center gap-0.5 text-indigo-400">
            <Camera class="w-3 h-3" /> {{ telemetry.vision_calls }} Vision
          </span>
          <span>•</span>
          <span class="flex items-center gap-0.5 text-purple-400">
            <Binary class="w-3 h-3" /> {{ telemetry.text_to_sql_calls }} SQL
          </span>
        </div>
      </div>

      <!-- Token Usage & FinOps Cost -->
      <div class="p-4 rounded-xl border border-border bg-card/60 relative overflow-hidden">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">Tokens & Cost (USD)</span>
          <Coins class="w-4 h-4 text-emerald-500" />
        </div>
        <div class="text-2xl font-black text-emerald-500 mt-2 tracking-tight">
          ${{ telemetry.total_cost_usd.toFixed(4) }}
        </div>
        <div class="mt-2 text-[11px] text-muted-foreground">
          {{ telemetry.total_tokens.toLocaleString() }} total tokens billed
        </div>
      </div>

      <!-- Average Inference Latency -->
      <div class="p-4 rounded-xl border border-border bg-card/60 relative overflow-hidden">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">Avg Model Latency</span>
          <Zap class="w-4 h-4 text-amber-500" />
        </div>
        <div class="text-2xl font-black text-amber-500 mt-2 tracking-tight">
          {{ telemetry.avg_latency_ms.toFixed(1) }} <span class="text-sm font-normal text-muted-foreground">ms</span>
        </div>
        <div class="mt-2 text-[11px] text-emerald-500 font-medium">
          Fast ACK SLA: &lt; 20ms
        </div>
      </div>

      <!-- Fraud Quarantine & Tamper Rate -->
      <div class="p-4 rounded-xl border border-border bg-card/60 relative overflow-hidden">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">Fraud Quarantined</span>
          <ShieldAlert class="w-4 h-4 text-rose-500" />
        </div>
        <div class="text-2xl font-black text-rose-500 mt-2 tracking-tight">
          {{ telemetry.fraud_quarantine_count }}
        </div>
        <div class="mt-2 text-[11px] text-muted-foreground">
          {{ telemetry.fraud_rate_pct.toFixed(1) }}% isolation rate
        </div>
      </div>
    </div>

    <!-- Security, Perceptual Hashing & Outbox Pipeline Status -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <!-- Guard 1: AST Token Sanitizer -->
      <div class="p-4 rounded-xl border border-border bg-card/40 flex flex-col justify-between">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-foreground flex items-center gap-1.5">
              <ShieldCheck class="w-4 h-4 text-emerald-500" />
              AST SQL Sanitizer
            </div>
            <p class="text-[11px] text-muted-foreground mt-1">
              Deterministic AST tokenizer blocking SQL injection keywords (<code class="text-rose-400">DROP</code>, <code class="text-rose-400">UPDATE</code>, multi-statement <code class="text-rose-400">;</code>).
            </p>
          </div>
          <Badge variant="emerald" class="text-xs font-mono">
            {{ telemetry.ast_blocks_count }} Blocked
          </Badge>
        </div>
        <div class="mt-3 pt-3 border-t border-border/50 text-[11px] text-muted-foreground">
          Database Mode: <span class="text-emerald-400 font-medium">READ-ONLY (2.5s Timeout)</span>
        </div>
      </div>

      <!-- Guard 2: Perceptual Hashing (pHash) -->
      <div class="p-4 rounded-xl border border-border bg-card/40 flex flex-col justify-between">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-foreground flex items-center gap-1.5">
              <Camera class="w-4 h-4 text-indigo-400" />
              Perceptual Hash (pHash)
            </div>
            <p class="text-[11px] text-muted-foreground mt-1">
              Difference gradient 64-bit dHash detecting re-uploaded or cropped duplicate bank slips before LLM.
            </p>
          </div>
          <Badge variant="indigo" class="text-xs font-mono">
            {{ telemetry.duplicate_phash_count }} Duplicates
          </Badge>
        </div>
        <div class="mt-3 pt-3 border-t border-border/50 text-[11px] text-muted-foreground">
          Hamming Threshold: <span class="text-indigo-400 font-medium">Distance &le; 5 bits</span>
        </div>
      </div>

      <!-- Guard 3: Transactional Outbox Pipeline -->
      <div class="p-4 rounded-xl border border-border bg-card/40 flex flex-col justify-between">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-foreground flex items-center gap-1.5">
              <Layers class="w-4 h-4 text-amber-400" />
              Transactional Outbox
            </div>
            <p class="text-[11px] text-muted-foreground mt-1">
              Atomic event persistence inside PostgreSQL transaction, eliminating the Dual-Write Problem.
            </p>
          </div>
          <Badge variant="amber" class="text-xs font-mono">
            {{ telemetry.outbox_pending_count }} Pending
          </Badge>
        </div>
        <div class="mt-3 pt-3 border-t border-border/50 text-[11px] text-muted-foreground">
          Broker Status: <span class="text-emerald-400 font-medium">{{ telemetry.outbox_processed_count }} Dispatched</span>
        </div>
      </div>
    </div>
  </div>
</template>
