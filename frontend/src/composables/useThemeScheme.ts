import { ref, computed, onMounted, onUnmounted } from 'vue'

export type ColorSchemePref = 'light dark' | 'light' | 'dark'

const storedScheme = ref<ColorSchemePref>('light dark')
const systemIsDark = ref<boolean>(false)

function readInitialScheme(): ColorSchemePref {
  if (typeof window === 'undefined') return 'light dark'
  try {
    const val = localStorage.getItem('color-scheme')
    if (val === 'light' || val === 'dark') return val
  } catch {
    // ignore storage restriction
  }
  return 'light dark'
}

function syncMetaTag(pref: ColorSchemePref) {
  if (typeof document === 'undefined') return
  const meta = document.querySelector('meta[name="color-scheme"]')
  if (meta) {
    meta.setAttribute('content', pref)
  }
}

export function useThemeScheme() {
  let mql: MediaQueryList | null = null

  const updateSystemMedia = (e: MediaQueryListEvent | MediaQueryList) => {
    // Never remove stored override when OS schedule coincides (per dark-mode-toggle)
    systemIsDark.value = e.matches
  }

  const onStorageSync = (e: StorageEvent) => {
    if (e.key === 'color-scheme') {
      const next: ColorSchemePref = e.newValue === 'light' || e.newValue === 'dark' ? e.newValue : 'light dark'
      storedScheme.value = next
      syncMetaTag(next)
    }
  }

  onMounted(() => {
    storedScheme.value = readInitialScheme()
    syncMetaTag(storedScheme.value)
    if (typeof window !== 'undefined' && window.matchMedia) {
      mql = window.matchMedia('(prefers-color-scheme: dark)')
      updateSystemMedia(mql)
      mql.addEventListener('change', updateSystemMedia)
      window.addEventListener('storage', onStorageSync)
    }
  })

  onUnmounted(() => {
    if (mql) mql.removeEventListener('change', updateSystemMedia)
    if (typeof window !== 'undefined') {
      window.removeEventListener('storage', onStorageSync)
    }
  })

  const resolvedTheme = computed<'light' | 'dark'>(() => {
    if (storedScheme.value === 'light') return 'light'
    if (storedScheme.value === 'dark') return 'dark'
    return systemIsDark.value ? 'dark' : 'light'
  })

  function setScheme(pref: ColorSchemePref) {
    storedScheme.value = pref
    syncMetaTag(pref)
    try {
      if (pref === 'light dark') {
        localStorage.removeItem('color-scheme')
      } else {
        localStorage.setItem('color-scheme', pref)
      }
    } catch {
      // ignore storage errors
    }
  }

  function toggleOppositeScheme() {
    const target: 'light' | 'dark' = resolvedTheme.value === 'dark' ? 'light' : 'dark'
    const currentSys: 'light' | 'dark' = systemIsDark.value ? 'dark' : 'light'
    if (target === currentSys) {
      // User is undoing override back to OS preference — remove stored key instead of pinning
      setScheme('light dark')
    } else {
      setScheme(target)
    }
  }

  return {
    storedScheme,
    systemIsDark,
    resolvedTheme,
    setScheme,
    toggleOppositeScheme,
  }
}

