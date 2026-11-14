import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { toast } from 'vue-sonner'

export type DemoRole = 'MERCHANT_ADMIN' | 'FINANCE_AUDITOR'

export interface DemoUser {
  id: string
  email: string
  name: string
  role: DemoRole
  roleTitle: string
  badgeVariant: 'emerald' | 'indigo'
  merchantName: string
  merchantCode: string
  permissions: string[]
  description: string
}

export const PRESET_ACCOUNTS: Record<DemoRole, DemoUser> = {
  MERCHANT_ADMIN: {
    id: 'usr-demo-01',
    email: 'demo@merchant.id',
    name: 'Frans Alwan (Admin Toko)',
    role: 'MERCHANT_ADMIN',
    roleTitle: 'Merchant Admin & Gudang (Operasional)',
    badgeVariant: 'emerald',
    merchantName: 'PT Toko Maju Omni-Retail',
    merchantCode: 'MID-8801',
    permissions: [
      'OMS Order Capture & Fulfillment',
      '3-State Inventory & Stock Mutasi',
      'Thermal Waybill A6 Printing',
      'AI WhatsApp Live Capture',
      'Multimodal Vision Slip AI & Anti-Fraud',
    ],
    description: 'Akses penuh operasional kasir, gudang, integrasi marketplace, dan validasi bukti bayar.',
  },
  FINANCE_AUDITOR: {
    id: 'usr-demo-02',
    email: 'finance@merchant.id',
    name: 'Budi Santoso (Head of Finance)',
    role: 'FINANCE_AUDITOR',
    roleTitle: 'Finance, Bendahara & Auditor (ERP)',
    badgeVariant: 'indigo',
    merchantName: 'PT Toko Maju Omni-Retail',
    merchantCode: 'MID-8801',
    permissions: [
      'Core Payment Switch & Double-Entry Ledger',
      '3-Way Bank Mutation Reconciliation (CSV)',
      'Natural Language Text-to-SQL Financial BI',
      'AI FinOps & API Token Cost Cockpit',
      'Audit Trail Hash-Chain SHA-256',
    ],
    description: 'Akses bendahara & akuntansi, audit mutasi rekening bank, dan analitik eksekutif (e-Bendahara / ERP).',
  },
}

export const DEMO_PASSWORD = 'DemoJMC2026!'

export const useAuthStore = defineStore('auth', () => {
  const storedRole = (localStorage.getItem('spg_demo_role') as DemoRole) || 'MERCHANT_ADMIN'
  const currentUser = ref<DemoUser>(PRESET_ACCOUNTS[storedRole] || PRESET_ACCOUNTS.MERCHANT_ADMIN)
  const isAuthModalOpen = ref(false)
  const isAuthenticated = ref(true)

  const isFinance = computed(() => currentUser.value.role === 'FINANCE_AUDITOR')
  const isAdmin = computed(() => currentUser.value.role === 'MERCHANT_ADMIN')

  function openAuthModal() {
    isAuthModalOpen.value = true
  }

  function closeAuthModal() {
    isAuthModalOpen.value = false
  }

  function selectRole(role: DemoRole) {
    currentUser.value = PRESET_ACCOUNTS[role]
    localStorage.setItem('spg_demo_role', role)
    isAuthenticated.value = true
    isAuthModalOpen.value = false
    toast.success(`Beralih Akun: ${currentUser.value.name}`, {
      description: `Role aktif: ${currentUser.value.roleTitle} (${currentUser.value.merchantName})`,
    })
  }

  function loginWithCredentials(emailInput: string, passwordInput: string): boolean {
    const cleanEmail = emailInput.trim().toLowerCase()
    const cleanPass = passwordInput.trim()

    if (cleanPass !== DEMO_PASSWORD) {
      toast.error('Password Tidak Cocok', {
        description: `Gunakan password demo pengujian: ${DEMO_PASSWORD}`,
      })
      return false
    }

    if (cleanEmail === 'finance@merchant.id' || cleanEmail.includes('finance') || cleanEmail.includes('bendahara')) {
      selectRole('FINANCE_AUDITOR')
      return true
    } else if (cleanEmail === 'demo@merchant.id' || cleanEmail.includes('demo') || cleanEmail.includes('admin')) {
      selectRole('MERCHANT_ADMIN')
      return true
    } else {
      // Default to Admin
      selectRole('MERCHANT_ADMIN')
      return true
    }
  }

  function logout() {
    isAuthenticated.value = false
    isAuthModalOpen.value = true
    toast.info('Sesi Ditutup', {
      description: 'Silakan pilih atau masukkan akun demo pengujian JMC untuk melanjutkan.',
    })
  }

  return {
    currentUser,
    isAuthModalOpen,
    isAuthenticated,
    isFinance,
    isAdmin,
    openAuthModal,
    closeAuthModal,
    selectRole,
    loginWithCredentials,
    logout,
  }
})
