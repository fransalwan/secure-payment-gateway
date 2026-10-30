<script setup lang="ts">
import { computed } from 'vue'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-xl text-xs font-semibold transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 cursor-pointer active:scale-[0.99]',
  {
    variants: {
      variant: {
        default: 'bg-foreground text-background hover:opacity-90 shadow-sm',
        emerald:
          'bg-emerald-600 text-white hover:bg-emerald-500 shadow-sm shadow-emerald-600/20 font-bold',
        indigo:
          'bg-indigo-600 text-white hover:bg-indigo-500 shadow-sm shadow-indigo-600/20 font-bold',
        amber:
          'bg-amber-500 text-zinc-950 hover:bg-amber-400 shadow-sm shadow-amber-500/20 font-bold',
        destructive:
          'bg-rose-500/12 text-tone-rose border border-rose-500/30 hover:bg-rose-500/20',
        outline:
          'border border-border bg-card hover:bg-elevated text-foreground',
        secondary:
          'bg-muted text-secondary-foreground hover:bg-elevated border border-border',
        ghost:
          'text-muted-foreground hover:bg-elevated hover:text-foreground',
      },
      size: {
        default: 'min-h-10 sm:min-h-9 px-3.5 py-2',
        sm: 'min-h-9 sm:min-h-8 px-2.5 py-1.5 text-[11px] rounded-lg',
        lg: 'min-h-11 px-5 py-2.5 text-sm',
        icon: 'h-9 w-9 p-0',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

type ButtonVariants = VariantProps<typeof buttonVariants>

const props = withDefaults(
  defineProps<{
    variant?: ButtonVariants['variant']
    size?: ButtonVariants['size']
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    class?: string
  }>(),
  {
    variant: 'default',
    size: 'default',
    type: 'button',
    disabled: false,
  },
)

const classes = computed(() => cn(buttonVariants({ variant: props.variant, size: props.size }), props.class))
</script>

<template>
  <button :type="type" :disabled="disabled" :class="classes">
    <slot />
  </button>
</template>
