import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { nextTick } from 'vue'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatIDR(val: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val)
}

/**
 * Progressive-enhancement SPA View Transition with mandatory accessibility focus routing
 * (per modern-web-guidance: same-document-transitions).
 */
export function runViewTransition(updateFn: () => void, focusHeadingId?: string) {
  const doc = document as Document & {
    startViewTransition?: (cb: () => Promise<void> | void) => { finished: Promise<void> }
  }

  const routeFocus = () => {
    if (!focusHeadingId) return
    nextTick(() => {
      document.getElementById(focusHeadingId)?.focus({ preventScroll: true })
    })
  }

  if (typeof document === 'undefined' || !doc.startViewTransition) {
    updateFn()
    routeFocus()
    return
  }

  const transition = doc.startViewTransition(async () => {
    updateFn()
    await nextTick()
  })

  transition.finished.finally(() => {
    routeFocus()
  })
}

