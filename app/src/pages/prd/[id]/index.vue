<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { useUserStore } from '@/stores/user'
import { PRD_SECTION_KEYS, RoadmapReplaceConflictError, type Prd, type Workspace } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PrdStatusBadge from '@/components/common/PrdStatusBadge.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import CoachPanel from '@/components/prd/CoachPanel.vue'
import { SparklesIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const prdId = route.params.id as string
const { t } = useI18n()
const userStore = useUserStore()

const loading = ref(true)
const notFound = ref(false)
const prd = ref<Prd | null>(null)
const workspace = ref<Workspace | null>(null)
const savingTitle = ref(false)
const titleDraft = ref('')

const isAuthor = computed(() => prd.value && userStore.user && prd.value.authorId === userStore.user.id)
const isAdmin = computed(() => workspace.value?.myRole === 'admin')

const breadcrumb = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  ...(workspace.value
    ? [{ label: workspace.value.title, to: `/workspace/${workspace.value.id}` }]
    : []),
  { label: prd.value?.title ?? '' },
])

async function load() {
  loading.value = true
  notFound.value = false
  try {
    const { prd: prdSvc, workspace: wsSvc } = useServices()
    prd.value = await prdSvc.get(prdId)
    titleDraft.value = prd.value.title
    workspace.value = await wsSvc.get(prd.value.workspaceId)
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

async function reloadPrd() {
  prd.value = await useServices().prd.get(prdId)
}

onMounted(load)

// Coach sidebar open state, persisted across sessions. Default: docked open on
// desktop, closed (drawer) on mobile so it doesn't cover the editor.
const COACH_KEY = 'insideout.coach.open'
const coachOpen = ref(true)
onMounted(() => {
  if (!import.meta.client) return
  const saved = localStorage.getItem(COACH_KEY)
  if (saved !== null) coachOpen.value = saved === '1'
  else if (window.matchMedia('(max-width: 1023px)').matches) coachOpen.value = false
})
watch(coachOpen, (v) => {
  if (import.meta.client) localStorage.setItem(COACH_KEY, v ? '1' : '0')
})

async function saveSection(key: (typeof PRD_SECTION_KEYS)[number], value: string) {
  if (!prd.value) return
  prd.value = await useServices().prd.updateSections(prdId, prd.value.title, { [key]: value })
}

async function saveTitle() {
  if (!prd.value || !titleDraft.value.trim() || titleDraft.value.trim() === prd.value.title) return
  savingTitle.value = true
  try {
    prd.value = await useServices().prd.updateSections(prdId, titleDraft.value.trim(), {})
  } finally {
    savingTitle.value = false
  }
}

async function setStatus(status: 'reviewing' | 'approved' | 'rejected' | 'draft') {
  if (!prd.value) return
  prd.value = await useServices().prd.updateStatus(prdId, status)
}

const snapshotting = ref(false)
async function snapshotRevision() {
  snapshotting.value = true
  try {
    await useServices().prd.createRevision(prdId)
  } finally {
    snapshotting.value = false
  }
}

const building = ref(false)
const buildError = ref('')
async function buildMVP() {
  building.value = true
  buildError.value = ''
  try {
    // First attempt carries no expectedCount. If the live roadmap is non-empty
    // the API 409s with its node count; confirm, then retry with that count.
    const res = await useServices().prd.build(prdId)
    await navigateTo(`/projects/${res.projectId}`)
  } catch (e) {
    if (e instanceof RoadmapReplaceConflictError) {
      if (window.confirm(t('prd.buildReplaceConfirm', { count: e.liveCount }))) {
        try {
          const res = await useServices().prd.build(prdId, e.liveCount)
          await navigateTo(`/projects/${res.projectId}`)
        } catch (e2) {
          buildError.value = (e2 as Error).message
        }
      }
    } else {
      buildError.value = (e as Error).message
    }
  } finally {
    building.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="w-full px-4 py-8 lg:px-8">
    <div class="h-8 w-1/3 animate-pulse rounded bg-surface-sunken" />
    <div class="mt-6 space-y-4">
      <div v-for="i in 3" :key="i" class="h-28 animate-pulse rounded-card bg-surface-sunken" />
    </div>
  </div>

  <div v-else-if="notFound" class="px-4 py-16 lg:px-8">
    <BaseEmptyState :title="t('prd.notFound')">
      <BaseButton to="/dashboard" variant="outline">{{ t('nav.dashboard') }}</BaseButton>
    </BaseEmptyState>
  </div>

  <div v-else-if="prd" class="flex w-full items-start">
    <!-- Editor column -->
    <div class="min-w-0 flex-1 px-4 py-6 lg:px-8">
      <PageHeader :trail="breadcrumb">
        <template #title>
          <div class="flex flex-wrap items-center gap-3">
            <input
              v-model="titleDraft"
              :aria-label="t('prd.title')"
              class="min-w-0 flex-1 rounded-control border border-transparent bg-transparent px-2 py-1 font-serif text-3xl font-semibold tracking-tight text-fg-primary transition-colors hover:border-stroke-subtle focus:border-stroke-focus focus:bg-surface-raised focus:outline-none"
              @blur="saveTitle"
              @keyup.enter="($event.target as HTMLInputElement).blur()"
            />
            <PrdStatusBadge :status="prd.status" />
          </div>
        </template>

        <template #actions>
          <BaseButton v-if="isAuthor && prd.status === 'draft'" size="sm" @click="setStatus('reviewing')">
            {{ t('prd.submitForReview') }}
          </BaseButton>
          <BaseButton v-if="isAuthor && prd.status === 'rejected'" size="sm" @click="setStatus('draft')">
            {{ t('prd.resubmit') }}
          </BaseButton>
          <template v-if="isAdmin && prd.status === 'reviewing'">
            <BaseButton size="sm" @click="setStatus('approved')">{{ t('prd.approve') }}</BaseButton>
            <BaseButton size="sm" variant="danger" @click="setStatus('rejected')">{{ t('prd.reject') }}</BaseButton>
          </template>
          <BaseButton size="sm" class="!bg-seal !text-carve" :loading="building" @click="buildMVP">
            <SparklesIcon class="-ml-0.5 mr-1.5 h-4 w-4" />
            {{ building ? t('prd.buildingMVP') : t('prd.buildMVP') }}
          </BaseButton>
        </template>
      </PageHeader>

      <!-- Document utilities -->
      <div class="mb-6 flex flex-wrap items-center gap-2">
        <BaseButton size="sm" variant="outline" :loading="snapshotting" @click="snapshotRevision">
          {{ t('prd.snapshotRevision') }}
        </BaseButton>
        <BaseButton size="sm" variant="outline" :to="`/prd/${prdId}/revisions`">
          {{ t('prd.revisionHistory') }}
        </BaseButton>
        <BaseButton size="sm" variant="outline" :to="`/prd/${prdId}/export`">
          {{ t('prd.exportMarkdown') }}
        </BaseButton>
        <p v-if="buildError" class="w-full text-sm text-fg-danger">{{ buildError }}</p>
      </div>

      <!-- Sections -->
      <div class="space-y-5">
        <BaseCard v-for="key in PRD_SECTION_KEYS" :key="key">
          <h3 class="mb-2 text-sm font-semibold text-fg-secondary">{{ t(`prd.sections.${key}`) }}</h3>
          <textarea
            :value="prd.sections[key]"
            rows="4"
            class="w-full resize-y rounded-control border border-stroke-subtle bg-surface-sunken p-3 text-sm leading-relaxed text-fg-primary focus:border-stroke-focus focus:outline-none focus:ring-1 focus:ring-stroke-focus"
            @blur="saveSection(key, ($event.target as HTMLTextAreaElement).value)"
          />
        </BaseCard>
      </div>
    </div>

    <!-- Detachable coach sidebar -->
    <CoachPanel v-model:open="coachOpen" :prd-id="prdId" @prd-updated="reloadPrd" />
  </div>
</template>
