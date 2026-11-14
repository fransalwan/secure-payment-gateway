<script setup lang="ts">
import { ref } from 'vue'
import ResponsiveDialog from './ui/ResponsiveDialog.vue'
import Button from './ui/Button.vue'
import Badge from './ui/Badge.vue'
import { useAuthStore, DEMO_PASSWORD, type DemoRole } from '../stores/authStore'
import {
  UserCheck,
  ShieldCheck,
  KeyRound,
  Store,
  Landmark,
  Sparkles,
  ArrowRight,
  Boxes,
  Coins,
  CheckCircle2,
} from '@lucide/vue'

const authStore = useAuthStore()

const inputEmail = ref(authStore.currentUser.email)
const inputPassword = ref(DEMO_PASSWORD)

function onManualSubmit() {
  authStore.loginWithCredentials(inputEmail.value, inputPassword.value)
}

function quickSelect(role: DemoRole) {
  if (role === 'MERCHANT_ADMIN') {
    inputEmail.value = 'demo@merchant.id'
    inputPassword.value = DEMO_PASSWORD
  } else {
    inputEmail.value = 'finance@merchant.id'
    inputPassword.value = DEMO_PASSWORD
  }
  authStore.selectRole(role)
}
</script>

<template>
  <ResponsiveDialog
    :open="authStore.isAuthModalOpen"
    title="Pilih Akun Demo & Role Pengujian"
    description="Kredensial Resmi Pengujian Teknis PT. Jiwa Muda Cipta Indonesia (JMC)"
    max-width="max-w-2xl"
    @update:open="authStore.isAuthModalOpen = $event"
  >
    <div class="space-y-5 py-1">
      <!-- JMC Technical Assessment Header Banner -->
      <div class="rounded-xl border border-indigo-500/30 bg-gradient-to-r from-indigo-950/40 via-purple-950/30 to-background p-3.5 flex items-start gap-3">
        <div class="w-9 h-9 rounded-lg bg-indigo-500/20 text-indigo-400 flex items-center justify-center shrink-0 mt-0.5 font-bold">
          <ShieldCheck class="w-5 h-5 text-indigo-400" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-xs font-bold text-foreground">JMC IT Consultant — Assessment Sandbox Gate</span>
            <Badge variant="indigo" class="text-[10px]">Posisi: Midle Programmer</Badge>
          </div>
          <p class="text-[11px] text-muted-foreground mt-0.5 leading-relaxed">
            Sesuai petunjuk tes teknis JMC, gunakan salah satu dari 2 akun demo berikut untuk menguji alur kerja <strong>ERP Warehouse &amp; OMS</strong> atau <strong>ERP Financial Accounting &amp; e-Bendahara</strong>.
          </p>
        </div>
      </div>

      <!-- 2 One-Click Preset Account Cards -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <!-- Akun 1: Merchant Admin & Gudang -->
        <div
          class="rounded-xl border p-3.5 transition cursor-pointer relative flex flex-col justify-between"
          :class="
            authStore.isAdmin
              ? 'border-emerald-500/60 bg-emerald-950/20 shadow-xs ring-1 ring-emerald-500/30'
              : 'border-border bg-card hover:border-emerald-500/40 hover:bg-muted/40'
          "
          @click="quickSelect('MERCHANT_ADMIN')"
        >
          <div>
            <div class="flex items-center justify-between gap-1 mb-2">
              <div class="flex items-center gap-2">
                <div class="w-7 h-7 rounded-md bg-emerald-500/10 text-emerald-400 flex items-center justify-center font-bold text-xs">
                  <Store class="w-4 h-4" />
                </div>
                <div>
                  <div class="text-xs font-bold text-foreground">Admin Operasional &amp; Gudang</div>
                  <div class="text-[10px] text-muted-foreground">Toko Maju Omni-Retail (MID-8801)</div>
                </div>
              </div>
              <Badge variant="emerald" class="text-[10px]">OMS Admin</Badge>
            </div>

            <div class="bg-elevated/70 rounded-lg p-2 font-mono text-[11px] space-y-0.5 border border-border/60 mb-2.5">
              <div class="text-foreground"><span class="text-muted-foreground">User:</span> demo@merchant.id</div>
              <div class="text-foreground"><span class="text-muted-foreground">Pass:</span> DemoJMC2026!</div>
            </div>

            <div class="text-[11px] text-muted-foreground mb-3 leading-relaxed">
              Fokus: Manajemen stok 3-status, order capture, cetak resi thermal A6, dan validasi slip bukti bayar bank.
            </div>

            <div class="flex flex-wrap gap-1 mb-3">
              <span class="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md bg-muted text-foreground border border-border">
                <Boxes class="w-3 h-3 text-emerald-400" /> 3-State Stock
              </span>
              <span class="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md bg-muted text-foreground border border-border">
                <Sparkles class="w-3 h-3 text-emerald-400" /> Slip AI
              </span>
            </div>
          </div>

          <Button
            type="button"
            size="sm"
            variant="emerald"
            class="w-full text-xs font-semibold justify-between"
            @click.stop="quickSelect('MERCHANT_ADMIN')"
          >
            <span>Masuk: Admin Gudang</span>
            <ArrowRight class="w-3.5 h-3.5" />
          </Button>
        </div>

        <!-- Akun 2: Finance & Bendahara -->
        <div
          class="rounded-xl border p-3.5 transition cursor-pointer relative flex flex-col justify-between"
          :class="
            authStore.isFinance
              ? 'border-indigo-500/60 bg-indigo-950/20 shadow-xs ring-1 ring-indigo-500/30'
              : 'border-border bg-card hover:border-indigo-500/40 hover:bg-muted/40'
          "
          @click="quickSelect('FINANCE_AUDITOR')"
        >
          <div>
            <div class="flex items-center justify-between gap-1 mb-2">
              <div class="flex items-center gap-2">
                <div class="w-7 h-7 rounded-md bg-indigo-500/10 text-indigo-400 flex items-center justify-center font-bold text-xs">
                  <Landmark class="w-4 h-4" />
                </div>
                <div>
                  <div class="text-xs font-bold text-foreground">Finance &amp; Bendahara (ERP)</div>
                  <div class="text-[10px] text-muted-foreground">Settlement &amp; Accounting Hub</div>
                </div>
              </div>
              <Badge variant="indigo" class="text-[10px]">Finance</Badge>
            </div>

            <div class="bg-elevated/70 rounded-lg p-2 font-mono text-[11px] space-y-0.5 border border-border/60 mb-2.5">
              <div class="text-foreground"><span class="text-muted-foreground">User:</span> finance@merchant.id</div>
              <div class="text-foreground"><span class="text-muted-foreground">Pass:</span> DemoJMC2026!</div>
            </div>

            <div class="text-[11px] text-muted-foreground mb-3 leading-relaxed">
              Fokus: Buku besar berpasangan (Double-Entry Ledger), rekonsiliasi CSV mutasi bank, dan Text-to-SQL BI.
            </div>

            <div class="flex flex-wrap gap-1 mb-3">
              <span class="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md bg-muted text-foreground border border-border">
                <Landmark class="w-3 h-3 text-indigo-400" /> Ledger
              </span>
              <span class="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md bg-muted text-foreground border border-border">
                <Coins class="w-3 h-3 text-indigo-400" /> FinOps
              </span>
            </div>
          </div>

          <Button
            type="button"
            size="sm"
            variant="indigo"
            class="w-full text-xs font-semibold justify-between"
            @click.stop="quickSelect('FINANCE_AUDITOR')"
          >
            <span>Masuk: Finance &amp; Auditor</span>
            <ArrowRight class="w-3.5 h-3.5" />
          </Button>
        </div>
      </div>

      <!-- Manual Input Fallback Form (Untuk Menguji Form Input Kredensial Asli) -->
      <form @submit.prevent="onManualSubmit" class="rounded-xl border border-border bg-card p-3.5 space-y-3">
        <div class="text-xs font-bold text-foreground flex items-center gap-2">
          <KeyRound class="w-3.5 h-3.5 text-muted-foreground" />
          <span>Atau Masukkan Kredensial Manual:</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label class="block text-[11px] font-semibold text-muted-foreground mb-1">
              Email / Username Demo
            </label>
            <input
              v-model="inputEmail"
              type="email"
              required
              class="w-full rounded-lg bg-elevated border border-border px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-ring font-mono"
              placeholder="demo@merchant.id"
            />
          </div>

          <div>
            <label class="block text-[11px] font-semibold text-muted-foreground mb-1">
              Password Demo
            </label>
            <input
              v-model="inputPassword"
              type="text"
              required
              class="w-full rounded-lg bg-elevated border border-border px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-ring font-mono"
              placeholder="DemoJMC2026!"
            />
          </div>
        </div>

        <div class="flex items-center justify-between pt-1">
          <span class="text-[11px] text-muted-foreground">
            Akun Aktif Sekarang: <strong class="text-foreground">{{ authStore.currentUser.email }}</strong> ({{ authStore.currentUser.roleTitle }})
          </span>
          <div class="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              @click="authStore.closeAuthModal"
            >
              Tutup
            </Button>
            <Button
              type="submit"
              variant="default"
              size="sm"
            >
              <UserCheck class="w-3.5 h-3.5 mr-1" />
              <span>Verifikasi Masuk</span>
            </Button>
          </div>
        </div>
      </form>
    </div>
  </ResponsiveDialog>
</template>
