<script setup lang="ts">
export interface Breadcrumb {
  label: string
  to?: string
}

interface Props {
  title?: string
  subtitle?: string
  trail?: Breadcrumb[]
}
withDefaults(defineProps<Props>(), { title: undefined, subtitle: undefined, trail: undefined })
</script>

<template>
  <header class="mb-8">
    <nav v-if="trail && trail.length" aria-label="Breadcrumb" class="mb-4">
      <ol class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
        <li v-for="(crumb, i) in trail" :key="i" class="flex items-center gap-x-2">
          <NuxtLink
            v-if="crumb.to && i < trail.length - 1"
            :to="crumb.to"
            class="text-fg-muted transition-colors hover:text-fg-primary"
          >
            {{ crumb.label }}
          </NuxtLink>
          <span
            v-else
            class="text-fg-secondary"
            :aria-current="i === trail.length - 1 ? 'page' : undefined"
          >
            {{ crumb.label }}
          </span>
          <span v-if="i < trail.length - 1" class="select-none text-fg-muted/60" aria-hidden="true">/</span>
        </li>
      </ol>
    </nav>

    <div class="flex flex-wrap items-end justify-between gap-x-6 gap-y-4">
      <div class="min-w-0">
        <slot name="title">
          <h1 class="font-serif text-3xl font-semibold tracking-tight text-fg-primary">{{ title }}</h1>
        </slot>
        <p v-if="subtitle" class="mt-2 text-sm text-fg-muted">{{ subtitle }}</p>
      </div>
      <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2.5">
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>
