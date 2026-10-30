<script setup lang="ts">
import { computed } from 'vue'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const badgeVariants = cva(
  'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[10px] font-semibold tracking-tight transition-colors',
  {
    variants: {
      variant: {
        default: 'bg-muted text-secondary-foreground border-border',
        emerald: 'bg-emerald-500/12 text-tone-emerald border-emerald-500/30',
        amber: 'bg-amber-500/15 text-tone-amber border-amber-500/35',
        rose: 'bg-rose-500/12 text-tone-rose border-rose-500/30',
        indigo: 'bg-indigo-500/12 text-tone-indigo border-indigo-500/30',
        sky: 'bg-sky-500/12 text-tone-sky border-sky-500/30',
        orange: 'bg-orange-500/12 text-tone-orange border-orange-500/30',
        pink: 'bg-pink-500/12 text-tone-pink border-pink-500/30',
        outline: 'bg-transparent text-muted-foreground border-border',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
)

type BadgeVariants = VariantProps<typeof badgeVariants>

const props = defineProps<{
  variant?: BadgeVariants['variant']
  class?: string
}>()

const classes = computed(() => cn(badgeVariants({ variant: props.variant }), props.class))
</script>

<template>
  <span :class="classes">
    <slot />
  </span>
</template>
