<script setup lang="ts">
import { useI18n } from 'vue-i18n'

// A build-instruction "1:1 call-out" box: a small, honest peek at the real
// artifact each step produces, drawn in the product's own design language —
// skeleton bars + real status labels, no fabricated marketing copy. The corner
// "1:1" tag echoes the brick-instruction scale call-out.
interface Props {
  kind: 'idea' | 'prd' | 'roadmap' | 'shipped'
}
defineProps<Props>()
const { t } = useI18n()
</script>

<template>
  <div class="relative rounded-card border border-stroke-subtle bg-surface-raised p-5 shadow-card">
    <span class="absolute right-3 top-3 select-none text-[10px] font-medium tracking-wide text-fg-muted">1:1</span>

    <!-- idea: a captured inbox card -->
    <div v-if="kind === 'idea'" class="space-y-3">
      <div class="flex items-center gap-2">
        <span class="h-2 w-2 rounded-full bg-seal"></span>
        <div class="h-2.5 w-36 rounded bg-fg-primary/75"></div>
      </div>
      <div class="h-2 w-11/12 rounded bg-surface-sunken"></div>
      <div class="h-2 w-7/12 rounded bg-surface-sunken"></div>
      <span class="inline-block rounded bg-status-neutral-bg px-1.5 py-0.5 text-[10px] font-medium text-status-neutral-fg">
        {{ t('idea.status.inbox') }}
      </span>
    </div>

    <!-- prd: the coach interviewing, a section filling in -->
    <div v-else-if="kind === 'prd'" class="space-y-3">
      <div class="flex">
        <div class="max-w-[80%] rounded-xl rounded-tl-sm bg-surface-sunken px-3 py-2">
          <div class="h-2 w-40 rounded bg-stroke-subtle"></div>
        </div>
      </div>
      <div class="flex justify-end">
        <div class="max-w-[70%] rounded-xl rounded-tr-sm bg-btn px-3 py-2">
          <div class="h-2 w-24 rounded bg-btn-fg/70"></div>
        </div>
      </div>
      <div class="rounded-lg border border-stroke-subtle p-3">
        <div class="mb-2 h-2.5 w-24 rounded bg-fg-primary/80"></div>
        <div class="space-y-1.5">
          <div class="h-1.5 w-full rounded bg-surface-sunken"></div>
          <div class="h-1.5 w-10/12 rounded bg-surface-sunken"></div>
          <div class="h-1.5 w-6/12 rounded bg-seal/70"></div>
        </div>
      </div>
    </div>

    <!-- roadmap: a branched tree with live node statuses -->
    <div v-else-if="kind === 'roadmap'" class="space-y-2.5">
      <div class="flex items-center gap-2.5">
        <span class="h-2.5 w-2.5 shrink-0 rounded-full bg-status-success-fg"></span>
        <div class="h-2 w-32 rounded bg-surface-sunken"></div>
        <span class="ml-auto rounded bg-status-success-bg px-1.5 py-0.5 text-[10px] font-medium text-status-success-fg">
          {{ t('roadmap.status.done') }}
        </span>
      </div>
      <div class="ml-4 flex items-center gap-2.5 border-l border-stroke-subtle pl-3">
        <span class="h-2.5 w-2.5 shrink-0 rounded-full bg-seal"></span>
        <div class="h-2 w-24 rounded bg-surface-sunken"></div>
        <span class="ml-auto rounded bg-status-info-bg px-1.5 py-0.5 text-[10px] font-medium text-status-info-fg">
          {{ t('roadmap.status.in_progress') }}
        </span>
      </div>
      <div class="ml-4 flex items-center gap-2.5 border-l border-stroke-subtle pl-3">
        <span class="h-2.5 w-2.5 shrink-0 rounded-full bg-status-neutral-fg"></span>
        <div class="h-2 w-20 rounded bg-surface-sunken"></div>
        <span class="ml-auto rounded bg-status-neutral-bg px-1.5 py-0.5 text-[10px] font-medium text-status-neutral-fg">
          {{ t('roadmap.status.locked') }}
        </span>
      </div>
    </div>

    <!-- shipped: GitHub commits landing as seals on the timeline -->
    <div v-else class="relative space-y-4 pl-1">
      <div class="absolute bottom-2 left-[7px] top-2 w-px bg-stroke-subtle"></div>
      <div v-for="c in ['a1b2c3d', 'e4f5g6h', 'i7j8k9l']" :key="c" class="relative flex items-center gap-3">
        <span class="relative z-10 h-3.5 w-3.5 shrink-0 rounded-[4px] bg-seal"></span>
        <div class="min-w-0 flex-1">
          <div class="h-2 w-40 max-w-full rounded bg-surface-sunken"></div>
          <div class="mt-1.5 h-1.5 w-16 rounded bg-surface-sunken/70"></div>
        </div>
        <span class="font-mono text-[11px] text-fg-muted">{{ c }}</span>
      </div>
      <p class="relative pt-1 text-xs text-fg-muted">{{ t('github.synced', { count: 12 }) }}</p>
    </div>
  </div>
</template>
