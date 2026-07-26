<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import type { ProjectDetail } from '@/types'
import RoadmapCanvas from '@/components/roadmap/RoadmapCanvas.vue'
import { ChevronLeftIcon } from '@heroicons/vue/24/outline'

// Full-viewport roadmap canvas. Uses the chrome `canvas` layout (NavBar, no
// footer) and fills the remaining viewport; the project page embeds the same
// component in a fixed-height shell.
definePageMeta({ layout: 'canvas' })

const route = useRoute()
const projectId = route.params.id as string
const { t } = useI18n()

const project = ref<ProjectDetail | null>(null)

onMounted(async () => {
  try {
    project.value = await useServices().project.get(projectId)
  } catch {
    project.value = null
  }
})

const trail = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  { label: project.value?.title ?? '…', to: `/projects/${projectId}` },
  { label: t('roadmap.title') },
])
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <!-- Slim header: back to project + breadcrumb + title. -->
    <div class="flex flex-none items-center gap-3 border-b border-stroke-subtle bg-surface-raised px-4 py-2.5 sm:px-6">
      <NuxtLink
        :to="`/projects/${projectId}`"
        class="inline-flex items-center gap-1 rounded-control px-2 py-1 text-sm text-fg-muted transition-colors hover:bg-surface-sunken hover:text-fg-primary"
      >
        <ChevronLeftIcon class="h-4 w-4" />
        {{ project?.title ?? t('roadmap.title') }}
      </NuxtLink>
      <span class="select-none text-fg-muted/60" aria-hidden="true">/</span>
      <h1 class="font-serif text-base font-semibold text-fg-primary">{{ t('roadmap.title') }}</h1>
    </div>

    <!-- Canvas fills the rest. -->
    <div class="min-h-0 flex-1">
      <RoadmapCanvas :project-id="projectId" />
    </div>
  </div>
</template>
