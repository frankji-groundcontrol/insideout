<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'primary' | 'secondary' | 'danger' | 'outline'
  size?: 'sm' | 'md' | 'lg'
  block?: boolean
  loading?: boolean
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
  // When set, render as a single styled NuxtLink instead of a <button> — so
  // call-to-action links stay one interactive element (never <a><button>).
  to?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  block: false,
  loading: false,
  disabled: false,
  type: 'button',
  to: undefined
})

const classes = computed(() => {
  const base =
    'inline-flex items-center justify-center rounded-control font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-stroke-focus disabled:opacity-50 disabled:cursor-not-allowed'

  // Primary is INK (bg-btn/text-btn-fg, inverts to paper on dark) — the
  // vermilion seal is reserved for accents/status, not the primary action.
  const variants = {
    primary: 'bg-btn text-btn-fg hover:opacity-90',
    secondary: 'bg-surface-sunken text-fg-primary hover:bg-surface-raised',
    danger: 'bg-fg-danger text-fg-inverse hover:opacity-90',
    outline: 'border border-stroke-subtle bg-transparent text-fg-secondary hover:bg-surface-sunken',
  }

  const sizes = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg'
  }

  return [
    base,
    variants[props.variant],
    sizes[props.size],
    props.block ? 'w-full' : ''
  ]
})
</script>

<template>
  <NuxtLink v-if="to" :to="to" :class="classes">
    <slot />
  </NuxtLink>
  <button v-else :type="type" :class="classes" :disabled="disabled || loading">
    <svg
      v-if="loading"
      class="animate-spin -ml-1 mr-2 h-4 w-4"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      ></circle>
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
    <slot />
  </button>
</template>
