<script setup lang="ts">
interface Props {
  title: string
  hint?: string
}
withDefaults(defineProps<Props>(), { hint: undefined })
</script>

<template>
  <div
    class="flex flex-col items-center justify-center rounded-card border border-dashed border-stroke-strong bg-surface-raised px-6 py-14 text-center"
  >
    <div
      v-if="$slots.icon"
      class="mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-stroke-subtle bg-surface-sunken text-fg-muted"
    >
      <slot name="icon" />
    </div>
    <h3 class="font-serif text-lg font-semibold text-fg-primary">{{ title }}</h3>
    <p v-if="hint" class="mt-1.5 max-w-sm text-sm text-fg-muted">{{ hint }}</p>
    <div v-if="$slots.default" class="mt-5">
      <slot />
    </div>
  </div>
</template>
