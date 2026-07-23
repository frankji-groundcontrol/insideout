<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import type { Prd } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { DocumentTextIcon, ClipboardDocumentIcon, CheckIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const prdId = route.params.id as string
const { t } = useI18n()

const loading = ref(true)
const notFound = ref(false)
const prd = ref<Prd | null>(null)
const markdown = ref('')
const copied = ref(false)
let copyTimer: ReturnType<typeof setTimeout> | null = null

const breadcrumb = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  { label: prd.value?.title ?? t('prd.title'), to: `/prd/${prdId}` },
  { label: t('prd.exportMarkdown') },
])

onMounted(async () => {
  try {
    const svc = useServices()
    const [prdData, { content }] = await Promise.all([
      svc.prd.get(prdId),
      svc.export.download(prdId, 'markdown'),
    ])
    prd.value = prdData
    markdown.value = content
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  if (copyTimer) clearTimeout(copyTimer)
})

async function copyMarkdown() {
  try {
    await navigator.clipboard.writeText(markdown.value)
    copied.value = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => (copied.value = false), 2000)
  } catch {
    copied.value = false
  }
}

function downloadMarkdown() {
  const blob = new Blob([markdown.value], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'prd.md'
  a.click()
  URL.revokeObjectURL(url)
}

async function printPdf() {
  const { content } = await useServices().export.download(prdId, 'print')
  const win = window.open('', '_blank')
  if (!win) return
  win.document.write(content)
  win.document.close()
  win.focus()
  win.print()
}
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <div v-if="loading" class="mx-auto max-w-3xl">
      <div class="h-64 animate-pulse rounded-card bg-surface-sunken" />
    </div>

    <BaseEmptyState v-else-if="notFound" :title="t('prd.notFound')">
      <template #icon><DocumentTextIcon class="h-6 w-6" /></template>
      <BaseButton to="/dashboard">{{ t('nav.dashboard') }}</BaseButton>
    </BaseEmptyState>

    <div v-else class="mx-auto max-w-3xl">
      <PageHeader :trail="breadcrumb" :title="t('prd.exportMarkdown')" :subtitle="prd?.title">
        <template #actions>
          <BaseButton variant="outline" @click="copyMarkdown">
            <component :is="copied ? CheckIcon : ClipboardDocumentIcon" class="-ml-1 mr-2 h-5 w-5" />
            {{ copied ? t('common.copied') : t('prd.copyMarkdown') }}
          </BaseButton>
          <BaseButton variant="outline" @click="downloadMarkdown">{{ t('prd.exportMarkdown') }}</BaseButton>
          <BaseButton @click="printPdf">{{ t('prd.exportPrint') }}</BaseButton>
        </template>
      </PageHeader>

      <pre class="whitespace-pre-wrap rounded-card border border-stroke-subtle bg-surface-raised p-6 font-sans text-sm leading-relaxed text-fg-secondary">{{ markdown }}</pre>
    </div>
  </div>
</template>
