<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from 'reka-ui'
import {
  DrawerRoot,
  DrawerPortal,
  DrawerOverlay,
  DrawerContent,
  DrawerTitle,
  DrawerDescription,
  DrawerClose,
} from 'vaul-vue'
import { X } from '@lucide/vue'
import { cn } from '../../lib/utils'

withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    maxWidthClass?: string
    sideSheetOnDesktop?: boolean
  }>(),
  {
    description: '',
    maxWidthClass: 'max-w-2xl',
    sideSheetOnDesktop: false,
  },
)

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
}>()

const isDesktop = useMediaQuery('(min-width: 768px)')
</script>

<template>
  <!-- Desktop (>= 768px): Accessible Reka UI Dialog or Right Slide-Over Sheet -->
  <DialogRoot v-if="isDesktop" :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs transition-opacity"
      />
      <DialogContent
        :class="
          cn(
            sideSheetOnDesktop
              ? 'fixed inset-y-0 right-0 z-50 flex h-dvh w-full max-w-lg flex-col border-l border-border bg-card text-foreground shadow-2xl focus:outline-none'
              : cn(
                  'fixed left-1/2 top-1/2 z-50 flex max-h-[88dvh] w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2 flex-col rounded-2xl border border-border bg-card text-foreground shadow-2xl focus:outline-none overflow-hidden',
                  maxWidthClass,
                ),
          )
        "
      >
        <div class="flex items-start justify-between gap-3 border-b border-border bg-elevated/60 px-5 py-4 shrink-0">
          <div class="min-w-0">
            <DialogTitle class="text-sm sm:text-base font-bold text-foreground truncate">
              {{ title }}
            </DialogTitle>
            <DialogDescription v-if="description" class="text-xs text-muted-foreground mt-0.5">
              {{ description }}
            </DialogDescription>
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <slot name="header-actions" />
            <DialogClose
              aria-label="Tutup dialog"
              class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-card text-muted-foreground hover:bg-muted hover:text-foreground cursor-pointer"
            >
              <X class="w-4 h-4" />
            </DialogClose>
          </div>
        </div>

        <div class="flex-1 overflow-y-auto custom-scroll p-5">
          <slot />
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>

  <!-- Mobile (< 768px): Native-Feel Swipeable Bottom Sheet (Vaul Vue + Dynamic Viewport dvh) -->
  <DrawerRoot v-else :open="open" @update:open="emit('update:open', $event)">
    <DrawerPortal>
      <DrawerOverlay class="fixed inset-0 z-50 bg-black/65 backdrop-blur-xs" />
      <DrawerContent
        class="fixed inset-x-0 bottom-0 z-50 mt-10 flex max-h-[92dvh] flex-col rounded-t-3xl border-t border-border bg-card text-foreground shadow-2xl focus:outline-none"
      >
        <!-- Drag Handle Pill -->
        <div class="mx-auto mt-2.5 mb-1 h-1.5 w-12 shrink-0 rounded-full bg-border" />

        <div class="flex items-start justify-between gap-3 border-b border-border px-4 py-3 shrink-0">
          <div class="min-w-0">
            <DrawerTitle class="text-sm font-bold text-foreground">
              {{ title }}
            </DrawerTitle>
            <DrawerDescription v-if="description" class="text-[11px] text-muted-foreground mt-0.5">
              {{ description }}
            </DrawerDescription>
          </div>
          <div class="flex items-center gap-1.5 shrink-0">
            <slot name="header-actions" />
            <DrawerClose
              aria-label="Tutup panel bawah"
              class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-elevated text-muted-foreground"
            >
              <X class="w-4 h-4" />
            </DrawerClose>
          </div>
        </div>

        <div class="flex-1 overflow-y-auto custom-scroll p-4 pb-safe">
          <slot />
        </div>
      </DrawerContent>
    </DrawerPortal>
  </DrawerRoot>
</template>

