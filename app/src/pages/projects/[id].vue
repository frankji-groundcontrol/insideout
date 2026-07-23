<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import type { ProjectDetail, ProjectUpdateKind } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import RoadmapTree from '@/components/roadmap/RoadmapTree.vue'
import GithubSync from '@/components/project/GithubSync.vue'
import { FolderIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const projectId = route.params.id as string
const { t } = useI18n()

const loading = ref(true)
const notFound = ref(false)
const project = ref<ProjectDetail | null>(null)
const updateKind = ref<ProjectUpdateKind>('progress')
const updateContent = ref('')
const submitting = ref(false)

const breadcrumb = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  { label: t('workspace.board'), to: project.value ? `/workspace/${project.value.workspaceId}` : '/dashboard' },
  { label: project.value?.title ?? '…' },
])

async function load() {
  loading.value = true
  notFound.value = false
  try {
    project.value = await useServices().project.get(projectId)
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function handlePostUpdate() {
  if (!updateContent.value.trim()) return
  submitting.value = true
  try {
    await useServices().project.addUpdate(projectId, updateKind.value, updateContent.value.trim())
    updateContent.value = ''
    await load()
  } finally {
    submitting.value = false
  }
}

const kindOrder: ProjectUpdateKind[] = ['progress', 'blocker', 'note']
const kindTone: Record<ProjectUpdateKind, 'success' | 'danger' | 'neutral'> = {
  progress: 'success',
  blocker: 'danger',
  note: 'neutral',
}
// Tailwind's content scanner statically greps for whole class-name tokens,
// so a template literal like `bg-status-${tone}-bg` never matches and the
// utility silently never gets generated — every combination must appear
// here as a complete literal string instead.
const kindActiveClasses: Record<ProjectUpdateKind, string> = {
  progress: 'bg-status-success-bg text-status-success-fg',
  blocker: 'bg-status-danger-bg text-status-danger-fg',
  note: 'bg-status-neutral-bg text-status-neutral-fg',
}
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <div v-if="loading" class="space-y-6">
      <div class="h-9 w-64 animate-pulse rounded-card bg-surface-sunken" />
      <div class="h-64 animate-pulse rounded-card bg-surface-sunken" />
    </div>

    <BaseEmptyState v-else-if="notFound || !project" :title="t('project.notFound')">
      <template #icon><FolderIcon class="h-6 w-6" /></template>
      <BaseButton to="/dashboard">{{ t('nav.dashboard') }}</BaseButton>
    </BaseEmptyState>

    <template v-else>
      <PageHeader :trail="breadcrumb" :title="project.title" :subtitle="project.description">
        <template #actions>
          <BaseBadge>{{ t(`project.status.${project.status}`) }}</BaseBadge>
        </template>
      </PageHeader>

      <div class="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <!-- Roadmap: the primary stage -->
        <section class="lg:col-span-2" aria-labelledby="roadmap-heading">
          <h2 id="roadmap-heading" class="mb-4 font-serif text-lg font-semibold text-fg-primary">
            {{ t('roadmap.title') }}
          </h2>
          <RoadmapTree :project-id="projectId" />
        </section>

        <!-- Activity rail -->
        <aside class="space-y-6" aria-labelledby="activity-heading">
          <h2 id="activity-heading" class="font-serif text-lg font-semibold text-fg-primary">
            {{ t('project.activity') }}
          </h2>

          <GithubSync :project-id="projectId" :repo-url="project.repoUrl" :owner-id="project.ownerId" @synced="load" />

          <BaseCard>
            <form class="space-y-3" @submit.prevent="handlePostUpdate">
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="k in kindOrder"
                  :key="k"
                  type="button"
                  class="rounded-pill px-3 py-1 text-sm"
                  :class="updateKind === k ? kindActiveClasses[k] : 'bg-surface-sunken text-fg-muted'"
                  @click="updateKind = k"
                >
                  {{ t(`project.updateKind.${k}`) }}
                </button>
              </div>
              <BaseInput v-model="updateContent" :label="t('project.updatePlaceholder')" required />
              <BaseButton type="submit" :loading="submitting" size="sm">{{ t('project.post') }}</BaseButton>
            </form>
          </BaseCard>

          <div>
            <ol v-if="project.updates.length" class="space-y-4 border-l border-stroke-subtle pl-4">
              <li v-for="u in project.updates" :key="u.id">
                <div class="mb-1 flex items-center gap-2">
                  <BaseBadge :tone="kindTone[u.kind]">{{ t(`project.updateKind.${u.kind}`) }}</BaseBadge>
                  <span class="text-xs text-fg-muted">{{ new Date(u.createdAt).toLocaleString() }}</span>
                </div>
                <p class="text-sm text-fg-secondary">{{ u.content }}</p>
              </li>
            </ol>
            <p v-else class="text-sm text-fg-muted">{{ t('project.empty') }}</p>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>
