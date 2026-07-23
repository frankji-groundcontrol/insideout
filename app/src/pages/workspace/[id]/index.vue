<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { timeAgo } from '@/composables/useTimeAgo'
import type { Workspace, Project } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { PlusIcon, Cog6ToothIcon, LightBulbIcon, FolderIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const workspaceId = route.params.id as string
const { t } = useI18n()

const loading = ref(true)
const notFound = ref(false)
const workspace = ref<Workspace | null>(null)
const projects = ref<Project[]>([])

const showCreate = ref(false)
const newTitle = ref('')
const newDescription = ref('')
const submitting = ref(false)

const statusFilter = ref<Project['status'] | 'all'>('all')
const filteredProjects = computed(() =>
  statusFilter.value === 'all' ? projects.value : projects.value.filter((p) => p.status === statusFilter.value),
)

const breadcrumb = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  { label: workspace.value?.title ?? '…' },
])

function isStale(p: Project): boolean {
  if (!p.latestUpdateAt) return true
  const days = (Date.now() - new Date(p.latestUpdateAt).getTime()) / 86_400_000
  return days > 14
}

async function load() {
  loading.value = true
  notFound.value = false
  try {
    const { workspace: ws, project } = useServices()
    const [wsData, projectData] = await Promise.all([ws.get(workspaceId), project.list(workspaceId)])
    workspace.value = wsData
    projects.value = projectData
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)

function openCreate() {
  newTitle.value = ''
  newDescription.value = ''
  showCreate.value = true
}

async function handleCreate() {
  if (!newTitle.value.trim()) return
  submitting.value = true
  try {
    await useServices().project.create(workspaceId, newTitle.value.trim(), newDescription.value)
    newTitle.value = ''
    newDescription.value = ''
    showCreate.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

const statusOrder: Project['status'][] = ['planning', 'active', 'paused', 'done', 'archived']
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <div v-if="loading" class="space-y-6">
      <div class="h-9 w-64 animate-pulse rounded-card bg-surface-sunken" />
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="i in 3" :key="i" class="h-32 animate-pulse rounded-card bg-surface-sunken" />
      </div>
    </div>

    <BaseEmptyState v-else-if="notFound" :title="t('workspace.notFound')">
      <template #icon><FolderIcon class="h-6 w-6" /></template>
      <BaseButton to="/dashboard">{{ t('nav.dashboard') }}</BaseButton>
    </BaseEmptyState>

    <template v-else-if="workspace">
      <PageHeader
        :trail="breadcrumb"
        :title="workspace.title"
        :subtitle="`${t('workspace.inviteCode', { code: workspace.code })} · ${t('workspace.members', { count: workspace.memberCount })}`"
      >
        <template #actions>
          <BaseButton variant="outline" :to="`/workspace/${workspaceId}/ideas`">
            <LightBulbIcon class="-ml-1 mr-2 h-5 w-5" />
            {{ t('workspace.ideas') }}
          </BaseButton>
          <BaseButton variant="outline" :to="`/workspace/${workspaceId}/settings`">
            <Cog6ToothIcon class="-ml-1 mr-2 h-5 w-5" />
            {{ t('workspace.settings') }}
          </BaseButton>
          <BaseButton @click="openCreate">
            <PlusIcon class="-ml-1 mr-2 h-5 w-5" />
            {{ t('project.newProject') }}
          </BaseButton>
        </template>
      </PageHeader>

      <div class="mb-5 flex flex-wrap gap-2">
        <button
          class="rounded-pill px-3 py-1.5 text-sm transition-colors"
          :class="statusFilter === 'all' ? 'bg-btn text-btn-fg' : 'bg-surface-sunken text-fg-secondary hover:text-fg-primary'"
          @click="statusFilter = 'all'"
        >
          {{ t('workspace.board') }}
        </button>
        <button
          v-for="s in statusOrder"
          :key="s"
          class="rounded-pill px-3 py-1.5 text-sm transition-colors"
          :class="statusFilter === s ? 'bg-btn text-btn-fg' : 'bg-surface-sunken text-fg-secondary hover:text-fg-primary'"
          @click="statusFilter = s"
        >
          {{ t(`project.status.${s}`) }}
        </button>
      </div>

      <BaseEmptyState v-if="filteredProjects.length === 0" :title="t('project.empty')">
        <template #icon><FolderIcon class="h-6 w-6" /></template>
        <BaseButton @click="openCreate">{{ t('project.newProject') }}</BaseButton>
      </BaseEmptyState>

      <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <NuxtLink v-for="p in filteredProjects" :key="p.id" :to="`/projects/${p.id}`">
          <BaseCard interactive>
            <div class="mb-2 flex items-center justify-between gap-3">
              <h3 class="font-semibold text-fg-primary">{{ p.title }}</h3>
              <BaseBadge>{{ t(`project.status.${p.status}`) }}</BaseBadge>
            </div>
            <p class="mb-3 line-clamp-2 min-h-[2.5rem] text-sm text-fg-muted">{{ p.description }}</p>
            <p class="text-xs" :class="isStale(p) ? 'font-medium text-status-warn-fg' : 'text-fg-muted'">
              <template v-if="p.latestUpdateContent">
                {{ p.latestUpdateContent }} — {{ t('workspace.lastUpdated', { when: timeAgo(p.latestUpdateAt) }) }}
              </template>
              <template v-else>{{ t('workspace.noUpdateYet') }}</template>
            </p>
          </BaseCard>
        </NuxtLink>
      </div>
    </template>

    <!-- New project -->
    <BaseModal :open="showCreate" :title="t('project.newProject')" @close="showCreate = false">
      <form id="create-project-form" class="space-y-4" @submit.prevent="handleCreate">
        <BaseInput v-model="newTitle" :label="t('project.namePlaceholder')" required />
        <BaseInput v-model="newDescription" :label="t('project.descPlaceholder')" />
      </form>
      <template #footer>
        <BaseButton variant="outline" @click="showCreate = false">{{ t('common.cancel') }}</BaseButton>
        <BaseButton type="submit" form="create-project-form" :loading="submitting">
          {{ t('project.create') }}
        </BaseButton>
      </template>
    </BaseModal>
  </div>
</template>
