<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import type { Idea, Workspace } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { LightBulbIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const workspaceId = route.params.id as string
const { t } = useI18n()

const loading = ref(true)
const workspace = ref<Workspace | null>(null)
const ideas = ref<Idea[]>([])
const title = ref('')
const content = ref('')
const submitting = ref(false)
const convertingId = ref<string | null>(null)
const dropTarget = ref<Idea | null>(null)
const dropping = ref(false)

const breadcrumb = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  { label: workspace.value?.title ?? '…', to: `/workspace/${workspaceId}` },
  { label: t('idea.title') },
])

async function load() {
  loading.value = true
  try {
    const { workspace: ws, idea } = useServices()
    const [wsData, ideaData] = await Promise.all([ws.get(workspaceId), idea.list(workspaceId)])
    workspace.value = wsData
    ideas.value = ideaData
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function handleCapture() {
  if (!title.value.trim()) return
  submitting.value = true
  try {
    const idea = await useServices().idea.create(workspaceId, title.value.trim(), content.value)
    ideas.value = [idea, ...ideas.value]
    title.value = ''
    content.value = ''
  } finally {
    submitting.value = false
  }
}

function askDrop(idea: Idea) {
  dropTarget.value = idea
}

async function confirmDrop() {
  if (!dropTarget.value) return
  dropping.value = true
  try {
    await useServices().idea.drop(dropTarget.value.id)
    ideas.value = ideas.value.filter((i) => i.id !== dropTarget.value?.id)
    dropTarget.value = null
  } finally {
    dropping.value = false
  }
}

async function handleConvert(id: string) {
  convertingId.value = id
  try {
    const { prdId } = await useServices().idea.convert(id)
    await navigateTo(`/prd/${prdId}`)
  } finally {
    convertingId.value = null
  }
}

const statusTone: Record<Idea['status'], 'neutral' | 'info' | 'success'> = {
  inbox: 'neutral',
  refining: 'info',
  converted: 'success',
  dropped: 'neutral',
}
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <PageHeader :trail="breadcrumb" :title="t('idea.title')" :subtitle="t('idea.inboxHint')" />

    <BaseCard class="mb-8 max-w-2xl">
      <form class="space-y-3" @submit.prevent="handleCapture">
        <BaseInput v-model="title" :label="t('idea.titlePlaceholder')" required />
        <BaseInput v-model="content" :label="t('idea.contentPlaceholder')" />
        <BaseButton type="submit" :loading="submitting">{{ t('idea.capture') }}</BaseButton>
      </form>
    </BaseCard>

    <div v-if="loading" class="space-y-3">
      <div v-for="i in 3" :key="i" class="h-20 animate-pulse rounded-card bg-surface-sunken" />
    </div>

    <BaseEmptyState v-else-if="ideas.length === 0" :title="t('idea.empty')">
      <template #icon><LightBulbIcon class="h-6 w-6" /></template>
    </BaseEmptyState>

    <div v-else class="space-y-3">
      <BaseCard v-for="idea in ideas" :key="idea.id" class="flex items-start justify-between gap-4">
        <div class="min-w-0 flex-1">
          <div class="mb-1 flex items-center gap-2">
            <h3 class="truncate font-medium text-fg-primary">{{ idea.title }}</h3>
            <BaseBadge class="shrink-0" :tone="statusTone[idea.status]">
              {{ t(`idea.status.${idea.status}`) }}
            </BaseBadge>
          </div>
          <p class="line-clamp-2 text-sm text-fg-muted">{{ idea.content }}</p>
        </div>
        <div v-if="idea.status !== 'converted' && idea.status !== 'dropped'" class="flex shrink-0 gap-2">
          <BaseButton size="sm" :loading="convertingId === idea.id" @click="handleConvert(idea.id)">
            {{ t('idea.convert') }}
          </BaseButton>
          <BaseButton size="sm" variant="outline" @click="askDrop(idea)">{{ t('idea.drop') }}</BaseButton>
        </div>
        <NuxtLink v-else-if="idea.prdId" :to="`/prd/${idea.prdId}`" class="shrink-0 text-sm text-seal hover:underline">
          {{ t('prd.title') }} →
        </NuxtLink>
      </BaseCard>
    </div>

    <!-- Drop confirm -->
    <BaseModal :open="dropTarget !== null" :title="t('idea.drop')" size="sm" @close="dropTarget = null">
      <p class="text-sm text-fg-secondary">
        {{ t('idea.dropConfirm') }}
        <span v-if="dropTarget" class="mt-2 block font-medium text-fg-primary">“{{ dropTarget.title }}”</span>
      </p>
      <template #footer>
        <BaseButton variant="outline" @click="dropTarget = null">{{ t('common.cancel') }}</BaseButton>
        <BaseButton variant="danger" :loading="dropping" @click="confirmDrop">{{ t('idea.drop') }}</BaseButton>
      </template>
    </BaseModal>
  </div>
</template>
