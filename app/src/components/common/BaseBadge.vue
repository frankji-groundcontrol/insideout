<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  tone?: 'neutral' | 'info' | 'warn' | 'success' | 'danger'
}
const props = withDefaults(defineProps<Props>(), { tone: 'neutral' })

const toneClasses: Record<NonNullable<Props['tone']>, string> = {
  neutral: 'bg-status-neutral-bg text-status-neutral-fg',
  info: 'bg-status-info-bg text-status-info-fg',
  warn: 'bg-status-warn-bg text-status-warn-fg',
  success: 'bg-status-success-bg text-status-success-fg',
  danger: 'bg-status-danger-bg text-status-danger-fg',
}

const classes = computed(() => toneClasses[props.tone])
</script>

<template>
  <span class="inline-flex items-center rounded-pill px-2.5 py-0.5 text-xs font-medium" :class="classes">
    <slot />
  </span>
</template>
