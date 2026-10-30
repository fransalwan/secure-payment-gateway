<script setup lang="ts">
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useOmsStore } from '../stores/omsStore'
import { formatIDR } from '../lib/utils'
import type { Product } from '../types/oms'
import { Boxes, Flame, Plus } from '@lucide/vue'

const store = useOmsStore()

function stockTotal(p: Product) {
  return Math.max(1, p.available_stock + p.reserved_stock + p.sold_stock)
}

function pct(val: number, total: number) {
  return `${Math.min(100, Math.round((val / total) * 100))}%`
}
</script>

<template>
  <section class="oms-panel rounded-2xl bg-card border border-border shadow-xs overflow-hidden">
    <div class="p-4 sm:p-5 border-b border-border flex flex-wrap items-center justify-between gap-2">
      <div>
        <div class="flex items-center gap-2">
          <Badge variant="indigo">
            <Boxes class="w-3 h-3" />
            <span>3-STATE STOCK</span>
          </Badge>
          <h2 id="inventory-monitor-caption" tabindex="-1" class="text-sm sm:text-base font-bold text-foreground focus:outline-none">
            Real-Time 3-State Inventory Engine
          </h2>
        </div>
        <p class="text-xs text-muted-foreground mt-0.5">
          Isolasi stok atomik PostgreSQL: <span class="text-tone-emerald font-semibold">Available</span> &bull;
          <span class="text-tone-amber font-semibold">Reserved (Checkout)</span> &bull;
          <span class="text-tone-indigo font-semibold">Sold (Paid)</span>
        </p>
      </div>
    </div>

    <div
      tabindex="0"
      role="region"
      aria-labelledby="inventory-monitor-caption"
      class="table-wrapper max-h-[500px] custom-scroll"
    >
      <table
        class="responsive-data-table text-xs text-left"
        style="--label-2: 'Available Stock'; --label-3: 'Reserved (Lock)'; --label-4: 'Sold (Paid)'"
      >
        <thead class="bg-elevated text-muted-foreground uppercase text-[10px] border-b border-border">
          <tr>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold">SKU &amp; Nama Produk</th>
            <th scope="col" class="px-3 py-3 bg-elevated font-semibold text-center">Available</th>
            <th scope="col" class="px-3 py-3 bg-elevated font-semibold text-center">Reserved</th>
            <th scope="col" class="px-3 py-3 bg-elevated font-semibold text-center">Sold</th>
            <th scope="col" class="px-4 py-3 bg-elevated font-semibold text-right">Kontrol Stok</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr
            v-for="prod in store.products"
            :key="prod.id"
            class="data-row hover:bg-elevated/40 transition"
          >
            <th scope="row" class="px-4 py-3 text-left font-normal">
              <div class="flex items-center justify-between sm:justify-start gap-2">
                <span class="font-semibold text-foreground">{{ prod.name }}</span>
                <Badge variant="outline" class="font-mono">v{{ prod.version }}</Badge>
              </div>
              <div class="flex items-center gap-2 mt-0.5">
                <span class="font-mono text-[11px] text-muted-foreground">{{ prod.sku }}</span>
                <span class="text-muted-foreground">&bull;</span>
                <span class="font-mono text-[11px] font-medium text-secondary-foreground tabular-nums">
                  {{ formatIDR(prod.price) }}
                </span>
              </div>
              <!-- 3-State Segmented Bar -->
              <div class="mt-2 h-1.5 w-full rounded-full bg-muted overflow-hidden flex">
                <div
                  class="bg-emerald-500 transition-all duration-300"
                  :style="{ width: pct(prod.available_stock, stockTotal(prod)) }"
                ></div>
                <div
                  class="bg-amber-500 transition-all duration-300"
                  :style="{ width: pct(prod.reserved_stock, stockTotal(prod)) }"
                ></div>
                <div
                  class="bg-indigo-500 transition-all duration-300"
                  :style="{ width: pct(prod.sold_stock, stockTotal(prod)) }"
                ></div>
              </div>
            </th>

            <td class="px-3 py-3 text-center font-mono tabular-nums">
              <Badge
                :variant="
                  prod.available_stock === 0
                    ? 'rose'
                    : prod.available_stock <= 2
                      ? 'amber'
                      : 'emerald'
                "
                class="px-2.5 py-1 text-xs font-extrabold min-w-[3rem] justify-center"
              >
                {{ prod.available_stock }}
              </Badge>
            </td>

            <td class="px-3 py-3 text-center font-mono tabular-nums">
              <Badge variant="amber" class="px-2.5 py-1 text-xs font-bold min-w-[2.5rem] justify-center">
                {{ prod.reserved_stock }}
              </Badge>
            </td>

            <td class="px-3 py-3 text-center font-mono tabular-nums">
              <Badge variant="indigo" class="px-2.5 py-1 text-xs font-bold min-w-[2.5rem] justify-center">
                {{ prod.sold_stock }}
              </Badge>
            </td>

            <td class="px-4 py-3 text-right">
              <div class="flex sm:inline-flex items-center justify-end gap-1.5 w-full sm:w-auto">
                <Button
                  variant="outline"
                  size="sm"
                  title="Set stok ke 1 pcs untuk demo Flash Sale Race Condition"
                  class="flex-1 sm:flex-none text-tone-amber border-amber-500/35 hover:bg-amber-500/10 font-mono"
                  @click="store.setProductStock(prod.id, 1)"
                >
                  <Flame class="w-3.5 h-3.5" />
                  <span>Set=1</span>
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  class="flex-1 sm:flex-none font-mono"
                  @click="store.setProductStock(prod.id, prod.available_stock + 10)"
                >
                  <Plus class="w-3.5 h-3.5" />
                  <span>10</span>
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
