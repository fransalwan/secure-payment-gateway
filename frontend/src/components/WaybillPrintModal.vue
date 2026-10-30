<script setup lang="ts">
import { ref } from 'vue'
import ResponsiveDialog from './ui/ResponsiveDialog.vue'
import Button from './ui/Button.vue'
import { useOmsStore } from '../stores/omsStore'
import { formatIDR } from '../lib/utils'
import type { Order } from '../types/oms'
import { Printer } from '@lucide/vue'

const store = useOmsStore()
const isOpen = ref(false)
const selectedOrder = ref<Order | null>(null)

function openWaybill(order: Order) {
  selectedOrder.value = order
  isOpen.value = true
}

function triggerPrint() {
  window.print()
}

defineExpose({ openWaybill })
</script>

<template>
  <ResponsiveDialog
    v-model:open="isOpen"
    title="Thermal Shipping Label (A6 — 100x150mm)"
    description="Siap cetak ke printer resi thermal ekspedisi gudang"
    max-width-class="max-w-md"
  >
    <template #header-actions>
      <Button variant="emerald" size="sm" @click="triggerPrint">
        <Printer class="w-3.5 h-3.5" />
        <span>Cetak Resi A6</span>
      </Button>
    </template>

    <div v-if="selectedOrder" class="flex justify-center">
      <!-- Printable A6 Sheet (Always light paper contrast) -->
      <div
        id="thermal-waybill-sheet"
        class="w-full bg-white text-zinc-950 rounded-xl p-4 border-2 border-zinc-900 space-y-3 text-xs shadow-md"
      >
        <div class="flex items-center justify-between border-b-2 border-zinc-900 pb-2">
          <div>
            <div class="text-lg font-black tracking-tight">{{ selectedOrder.courier_code }} REG</div>
            <div class="text-[10px] font-mono uppercase text-zinc-600">
              Channel: {{ selectedOrder.channel }}
            </div>
          </div>
          <div class="text-right">
            <span class="px-2 py-0.5 bg-zinc-950 text-white font-extrabold text-[10px] rounded uppercase">
              NON-COD (PAID)
            </span>
            <div class="text-[10px] font-mono mt-1">{{ selectedOrder.order_number }}</div>
          </div>
        </div>

        <!-- Simulated Vector Barcode -->
        <div class="py-2 text-center border-b-2 border-dashed border-zinc-400">
          <svg class="w-full h-12 mx-auto" viewBox="0 0 260 44" aria-hidden="true">
            <rect x="6" y="2" width="3" height="40" fill="#000" />
            <rect x="12" y="2" width="1.5" height="40" fill="#000" />
            <rect x="17" y="2" width="4" height="40" fill="#000" />
            <rect x="25" y="2" width="2" height="40" fill="#000" />
            <rect x="30" y="2" width="1.5" height="40" fill="#000" />
            <rect x="35" y="2" width="5" height="40" fill="#000" />
            <rect x="44" y="2" width="2" height="40" fill="#000" />
            <rect x="50" y="2" width="3.5" height="40" fill="#000" />
            <rect x="57" y="2" width="1.5" height="40" fill="#000" />
            <rect x="62" y="2" width="4" height="40" fill="#000" />
            <rect x="70" y="2" width="2" height="40" fill="#000" />
            <rect x="76" y="2" width="4.5" height="40" fill="#000" />
            <rect x="84" y="2" width="1.5" height="40" fill="#000" />
            <rect x="89" y="2" width="3" height="40" fill="#000" />
            <rect x="96" y="2" width="2" height="40" fill="#000" />
            <rect x="102" y="2" width="5" height="40" fill="#000" />
            <rect x="111" y="2" width="1.5" height="40" fill="#000" />
            <rect x="116" y="2" width="3.5" height="40" fill="#000" />
            <rect x="124" y="2" width="2" height="40" fill="#000" />
            <rect x="129" y="2" width="4" height="40" fill="#000" />
            <rect x="137" y="2" width="1.5" height="40" fill="#000" />
            <rect x="142" y="2" width="4.5" height="40" fill="#000" />
            <rect x="150" y="2" width="2" height="40" fill="#000" />
            <rect x="156" y="2" width="3" height="40" fill="#000" />
            <rect x="163" y="2" width="5" height="40" fill="#000" />
            <rect x="172" y="2" width="1.5" height="40" fill="#000" />
            <rect x="177" y="2" width="3" height="40" fill="#000" />
            <rect x="184" y="2" width="2" height="40" fill="#000" />
            <rect x="190" y="2" width="4" height="40" fill="#000" />
            <rect x="198" y="2" width="1.5" height="40" fill="#000" />
            <rect x="203" y="2" width="5" height="40" fill="#000" />
            <rect x="212" y="2" width="2.5" height="40" fill="#000" />
            <rect x="218" y="2" width="3.5" height="40" fill="#000" />
            <rect x="226" y="2" width="1.5" height="40" fill="#000" />
            <rect x="231" y="2" width="4" height="40" fill="#000" />
            <rect x="239" y="2" width="2" height="40" fill="#000" />
            <rect x="245" y="2" width="4.5" height="40" fill="#000" />
            <rect x="252" y="2" width="2" height="40" fill="#000" />
          </svg>
          <div class="font-mono font-black text-sm tracking-widest mt-1">
            {{ selectedOrder.waybill_number || 'AWB-PENDING' }}
          </div>
        </div>

        <!-- Recipient & Sender -->
        <div class="space-y-2 border-b border-zinc-300 pb-2.5">
          <div>
            <div class="text-[10px] font-bold uppercase text-zinc-500">Penerima:</div>
            <div class="font-extrabold text-sm">{{ selectedOrder.customer_name }}</div>
            <div class="font-mono text-xs">{{ selectedOrder.customer_phone }}</div>
            <div class="text-xs leading-snug mt-0.5">{{ selectedOrder.shipping_address }}</div>
          </div>
          <div class="pt-1 border-t border-zinc-200 flex justify-between text-[11px]">
            <div>
              <span class="font-bold text-zinc-500 uppercase">Pengirim:</span>
              <span class="font-bold ml-1">{{ store.activeMerchant?.name }}</span>
            </div>
            <span class="font-mono">{{ store.activeMerchant?.city }}</span>
          </div>
        </div>

        <!-- Packing Checklist -->
        <div>
          <div class="text-[10px] font-bold uppercase text-zinc-500 mb-1">
            Daftar Barang Packing ({{ selectedOrder.items?.length || 0 }} SKU):
          </div>
          <table class="w-full text-[11px]">
            <tbody>
              <tr v-for="it in selectedOrder.items" :key="it.id" class="border-b border-zinc-100">
                <td class="py-1 font-mono text-[10px]">{{ it.sku }}</td>
                <td class="py-1 font-medium">{{ it.product_name }}</td>
                <td class="py-1 text-right font-bold">x{{ it.qty }}</td>
              </tr>
            </tbody>
          </table>
          <div class="mt-2 flex justify-between items-center text-[11px] font-bold pt-1 border-t border-zinc-900 tabular-nums">
            <span>Nilai Pertanggungan</span>
            <span class="font-mono">{{ formatIDR(selectedOrder.total_amount) }}</span>
          </div>
        </div>
      </div>
    </div>
  </ResponsiveDialog>
</template>
