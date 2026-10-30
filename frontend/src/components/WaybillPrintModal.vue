<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useOmsStore } from '../stores/omsStore'
import type { Order } from '../types/oms'

const store = useOmsStore()
const dialogRef = ref<HTMLDialogElement | null>(null)
const selectedOrder = ref<Order | null>(null)

function openWaybill(order: Order) {
  selectedOrder.value = order
  dialogRef.value?.showModal()
}

function triggerPrint() {
  window.print()
}

function formatIDR(val: number) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val)
}

onMounted(() => {
  const dialog = dialogRef.value
  if (dialog && !('closedBy' in HTMLDialogElement.prototype)) {
    dialog.addEventListener('click', (event: MouseEvent) => {
      if (event.target !== dialog) return
      const rect = dialog.getBoundingClientRect()
      const inside =
        rect.top <= event.clientY &&
        event.clientY <= rect.top + rect.height &&
        rect.left <= event.clientX &&
        event.clientX <= rect.left + rect.width
      if (!inside) dialog.close()
    })
  }
})

defineExpose({ openWaybill })
</script>

<template>
  <dialog
    ref="dialogRef"
    v-bind="{ closedby: 'any' } as any"
    aria-labelledby="waybill-dialog-title"
    class="w-full max-w-md rounded-2xl bg-slate-900 text-slate-100 border border-slate-700 shadow-2xl p-0 overflow-hidden"
  >
    <div class="px-5 py-3.5 bg-slate-950 border-b border-slate-800 flex items-center justify-between">
      <div>
        <h2 id="waybill-dialog-title" class="text-sm font-bold text-white">
          Thermal Shipping Label (Standar A6 — 100x150mm)
        </h2>
        <p class="text-[11px] text-slate-400">Siap cetak ke printer resi thermal ekspedisi</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          type="button"
          @click="triggerPrint"
          autofocus
          class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs cursor-pointer"
        >
          Cetak Resi A6
        </button>
        <form method="dialog">
          <button
            type="submit"
            class="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs cursor-pointer"
          >
            Tutup
          </button>
        </form>
      </div>
    </div>

    <div v-if="selectedOrder" class="p-5 bg-slate-900 flex justify-center">
      <!-- Printable A6 Sheet -->
      <div
        id="thermal-waybill-sheet"
        class="w-full bg-white text-slate-950 rounded-lg p-4 border-2 border-slate-900 space-y-3 text-xs"
      >
        <div class="flex items-center justify-between border-b-2 border-slate-900 pb-2">
          <div>
            <div class="text-lg font-black tracking-tight">{{ selectedOrder.courier_code }} REG</div>
            <div class="text-[10px] font-mono uppercase text-slate-600">
              Channel: {{ selectedOrder.channel }}
            </div>
          </div>
          <div class="text-right">
            <span class="px-2 py-0.5 bg-slate-950 text-white font-extrabold text-[10px] rounded uppercase">
              NON-COD (PAID)
            </span>
            <div class="text-[10px] font-mono mt-1">{{ selectedOrder.order_number }}</div>
          </div>
        </div>

        <!-- Simulated Vector Barcode -->
        <div class="py-2 text-center border-b-2 border-dashed border-slate-400">
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
        <div class="space-y-2 border-b border-slate-300 pb-2.5">
          <div>
            <div class="text-[10px] font-bold uppercase text-slate-500">Penerima:</div>
            <div class="font-extrabold text-sm">{{ selectedOrder.customer_name }}</div>
            <div class="font-mono text-xs">{{ selectedOrder.customer_phone }}</div>
            <div class="text-xs leading-snug mt-0.5">{{ selectedOrder.shipping_address }}</div>
          </div>
          <div class="pt-1 border-t border-slate-200 flex justify-between text-[11px]">
            <div>
              <span class="font-bold text-slate-500 uppercase">Pengirim:</span>
              <span class="font-bold ml-1">{{ store.activeMerchant?.name }}</span>
            </div>
            <span class="font-mono">{{ store.activeMerchant?.city }}</span>
          </div>
        </div>

        <!-- Packing Checklist -->
        <div>
          <div class="text-[10px] font-bold uppercase text-slate-500 mb-1">
            Daftar Barang Packing ({{ selectedOrder.items?.length || 0 }} SKU):
          </div>
          <table class="w-full text-[11px]">
            <tbody>
              <tr v-for="it in selectedOrder.items" :key="it.id" class="border-b border-slate-100">
                <td class="py-1 font-mono text-[10px]">{{ it.sku }}</td>
                <td class="py-1 font-medium">{{ it.product_name }}</td>
                <td class="py-1 text-right font-bold">x{{ it.qty }}</td>
              </tr>
            </tbody>
          </table>
          <div class="mt-2 flex justify-between items-center text-[11px] font-bold pt-1 border-t border-slate-900">
            <span>Nilai Pertanggungan</span>
            <span class="font-mono">{{ formatIDR(selectedOrder.total_amount) }}</span>
          </div>
        </div>
      </div>
    </div>
  </dialog>
</template>

