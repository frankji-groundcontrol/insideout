<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { PRD_SECTION_KEYS, type Prd, type PrdRevision } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { ClockIcon, DocumentTextIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const prdId = route.params.id as string
const { t } = useI18n()

const loading = ref(true)
const notFound = ref(false)
const prd = ref<Prd | null>(null)
const revisions = ref<PrdRevision[]>([])
const selectedId = ref<string | null>(null)

const selected = computed<PrdRevision | null>(
  () => revisions.value.find((r) => r.id === selectedId.value) ?? revisions.value[0] ?? null,
)

const breadcrumb = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  { label: prd.value?.title ?? '…', to: `/prd/${prdId}` },
  { label: t('prd.revisionsTitle') },
])

async function load() {
  loading.value = true
  notFound.value = false
  try {
    const svc = useServices().prd
    const [prdData, revData] = await Promise.all([svc.get(prdId), svc.listRevisions(prdId)])
    prd.value = prdData
    revisions.value = revData
    selectedId.value = revData[0]?.id ?? null
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <div v-if="loading" class="space-y-6">
      <div class="h-9 w-64 animate-pulse rounded-card bg-surface-sunken" />
      <div class="h-64 animate-pulse rounded-card bg-surface-sunken" />
    </div>

    <BaseEmptyState v-else-if="notFound" :title="t('prd.notFound')">
      <template #icon><DocumentTextIcon class="h-6 w-6" /></template>
      <BaseButton to="/dashboard">{{ t('nav.dashboard') }}</BaseButton>
    </BaseEmptyState>

    <template v-else-if="prd">
      <PageHeader :trail="breadcrumb" :title="t('prd.revisionsTitle')" :subtitle="prd.title">
        <template #actions>
          <BaseButton variant="outline" :to="`/prd/${prdId}`">{{ t('prd.backToPrd') }}</BaseButton>
        </template>
      </PageHeader>

      <BaseEmptyState v-if="revisions.length === 0" :title="t('prd.emptyRevisions')">
        <template #icon><ClockIcon class="h-6 w-6" /></template>
        <BaseButton :to="`/prd/${prdId}`">{{ t('prd.backToPrd') }}</BaseButton>
      </BaseEmptyState>

      <div v-else class="grid grid-cols-1 gap-8 lg:grid-cols-4">
        <!-- Revision list -->
        <aside class="lg:col-span-1">
          <ol class="space-y-2">
            <li v-for="r in revisions" :key="r.id">
              <button
                type="button"
                class="w-full rounded-card border px-4 py-3 text-left transition-colors"
                :class="selected?.id === r.id
                  ? 'border-seal bg-surface-raised'
                  : 'border-stroke-subtle bg-surface-raised hover:border-stroke-strong'"
                @click="selectedId = r.id"
              >
                <div class="flex items-center justify-between gap-2">
                  <span class="font-medium text-fg-primary">{{ t('prd.revisionN', { n: r.revision }) }}</span>
                  <BaseBadge v-if="r.revision === prd.currentRevision" tone="info">{{ t('prd.snapshotLabel') }}</BaseBadge>
                </div>
                <p v-if="r.note" class="mt-1 line-clamp-1 text-xs text-fg-muted">{{ r.note }}</p>
                <p class="mt-1 text-xs text-fg-muted">{{ new Date(r.createdAt).toLocaleString() }}</p>
              </button>
            </li>
          </ol>
        </aside>

        <!-- Read-only snapshot -->
        <section v-if="selected" class="space-y-4 lg:col-span-3" :aria-label="t('prd.revisionN', { n: selected.revision })">
          <BaseCard v-for="key in PRD_SECTION_KEYS" :key="key">
            <h3 class="mb-2 font-serif text-base font-semibold text-fg-primary">{{ t(`prd.sections.${key}`) }}</h3>
            <p v-if="selected.sections[key]" class="whitespace-pre-wrap text-sm text-fg-secondary">
              {{ selected.sections[key] }}
            </p>
            <p v-else class="text-sm italic text-fg-muted">—</p>
          </BaseCard>
        </section>
      </div>
    </template>
  </div>
</template>
