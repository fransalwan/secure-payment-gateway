<script setup lang="ts">
import { useOmsStore } from '../stores/omsStore'
import type { Product } from '../types/oms'

const store = useOmsStore()

function stockTotal(p: Product) {
  return Math.max(1, p.available_stock + p.reserved_stock + p.sold_stock)
}

function pct(val: number, total: number) {
  return `${Math.min(100, Math.round((val / total) * 100))}%`
}

function formatIDR(val: number) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val)
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-slate-900/80 border border-slate-800 overflow-hidden">
    <div class="p-4 sm:p-5 border-b border-slate-800 flex flex-wrap items-center justify-between gap-2">
      <div>
        <div class="flex items-center gap-2">
          <span class="px-2 py-0.5 rounded bg-indigo-500/15 text-indigo-300 font-mono text-[10px] font-bold">
            PANEL 2
          </span>
          <h2 id="inventory-monitor-caption" class="text-sm sm:text-base font-bold text-white">
            Real-Time 3-State Inventory Engine
          </h2>
        </div>
        <p class="text-xs text-slate-400 mt-0.5">
          Isolasi stok atomik PostgreSQL: <span class="text-emerald-400 font-semibold">Available</span> &bull;
          <span class="text-amber-400 font-semibold">Reserved (Checkout)</span> &bull;
          <span class="text-indigo-400 font-semibold">Sold (Paid)</span>
        </p>
      </div>
    </div>

    <div
      tabindex="0"
      role="region"
      aria-labelledby="inventory-monitor-caption"
      class="table-wrapper max-h-[420px] custom-scroll"
    >
      <table class="w-full text-xs text-left">
        <thead class="bg-slate-950 text-slate-400 uppercase text-[10px] border-b border-slate-800">
          <tr>
            <th scope="col" class="px-4 py-3 bg-slate-950">SKU &amp; Nama Produk</th>
            <th scope="col" class="px-3 py-3 bg-slate-950 text-center">Available</th>
            <th scope="col" class="px-3 py-3 bg-slate-950 text-center">Reserved</th>
            <th scope="col" class="px-3 py-3 bg-slate-950 text-center">Sold</th>
            <th scope="col" class="px-4 py-3 bg-slate-950 text-right">Kontrol Stok Demo</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/80">
          <tr v-for="prod in store.products" :key="prod.id" class="hover:bg-slate-800/35 transition">
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <span class="font-semibold text-white">{{ prod.name }}</span>
                <span class="px-1.5 py-0.2 rounded bg-slate-800 text-slate-400 font-mono text-[10px]">
                  v{{ prod.version }}
                </span>
              </div>
              <div class="flex items-center gap-2 mt-0.5">
                <span class="font-mono text-[11px] text-slate-400">{{ prod.sku }}</span>
                <span class="text-slate-600">&bull;</span>
                <span class="font-mono text-[11px] text-slate-300">{{ formatIDR(prod.price) }}</span>
              </div>
              <!-- 3-State Progress Bar -->
              <div class="mt-2 h-1.5 w-full rounded-full bg-slate-800 overflow-hidden flex">
                <div
                  class="bg-emerald-400 transition-all duration-300"
                  :style="{ width: pct(prod.available_stock, stockTotal(prod)) }"
                ></div>
                <div
                  class="bg-amber-400 transition-all duration-300"
                  :style="{ width: pct(prod.reserved_stock, stockTotal(prod)) }"
                ></div>
                <div
                  class="bg-indigo-400 transition-all duration-300"
                  :style="{ width: pct(prod.sold_stock, stockTotal(prod)) }"
                ></div>
              </div>
            </td>

            <td class="px-3 py-3 text-center font-mono">
              <span
                :class="
                  prod.available_stock === 0
                    ? 'bg-rose-500/20 text-rose-300 border-rose-500/40'
                    : prod.available_stock <= 2
                      ? 'bg-amber-500/20 text-amber-300 border-amber-500/40'
                      : 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
                "
                class="px-2.5 py-1 rounded-lg border font-extrabold text-xs inline-block min-w-[3rem]"
              >
                {{ prod.available_stock }}
              </span>
            </td>

            <td class="px-3 py-3 text-center font-mono">
              <span class="px-2.5 py-1 rounded-lg bg-amber-500/10 border border-amber-500/25 text-amber-300 font-bold">
                {{ prod.reserved_stock }}
              </span>
            </td>

            <td class="px-3 py-3 text-center font-mono">
              <span class="px-2.5 py-1 rounded-lg bg-indigo-500/10 border border-indigo-500/25 text-indigo-300 font-bold">
                {{ prod.sold_stock }}
              </span>
            </td>

            <td class="px-4 py-3 text-right">
              <div class="inline-flex items-center gap-1.5">
                <button
                  type="button"
                  @click="store.setProductStock(prod.id, 1)"
                  title="Set stok ke 1 pcs untuk demo Flash Sale Race Condition"
                  class="px-2 py-1 rounded-lg bg-amber-500/15 hover:bg-amber-500/25 border border-amber-500/30 text-amber-300 font-mono text-[10px] font-bold cursor-pointer"
                >
                  Set=1
                </button>
                <button
                  type="button"
                  @click="store.setProductStock(prod.id, prod.available_stock + 10)"
                  class="px-2 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 font-mono text-[10px] font-bold cursor-pointer"
                >
                  +10
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

