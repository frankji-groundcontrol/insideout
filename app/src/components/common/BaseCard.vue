<script setup lang="ts">
interface Props {
  /** Renders as a clickable/hoverable surface (e.g. a link card). */
  interactive?: boolean
}
withDefaults(defineProps<Props>(), { interactive: false })
</script>

<template>
  <div
    class="rounded-card border border-stroke-subtle bg-surface-raised p-5 shadow-card"
    :class="interactive && 'transition-shadow hover:shadow-popover'"
  >
    <slot />
  </div>
</template>
