<script setup lang="ts">
/* biome-ignore-all assist/source/organizeImports: Vue SFC 按语义分组导入 */
/* biome-ignore-all lint/correctness/noUnusedImports: 模板中会使用导入内容 */
/* biome-ignore-all lint/correctness/noUnusedVariables: 模板中会使用变量 */
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDownTrayIcon, ArrowLeftIcon, PrinterIcon } from '@heroicons/vue/24/outline'
import BaseButton from '@/components/common/BaseButton.vue'
import { useWorkshopStore } from '@/stores/workshop'
import { downloadMarkdown, generateMarkdown, triggerPrint } from '@/utils/export'
import '@/assets/print.css'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const workshopStore = useWorkshopStore()

const workshopId = computed(() => String(route.params.id ?? ''))
const workshop = computed(() => workshopStore.currentWorkshop)
const nodes = computed(() => [...workshopStore.roadmapNodes].sort((a, b) => a.order - b.order))
const generatedAt = computed(() => {
  const formatter = new Intl.DateTimeFormat(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
  return formatter.format(new Date())
})

function nodeContent(content?: string): string {
  const trimmed = content?.trim()
  return trimmed && trimmed.length > 0 ? trimmed : t('export.notCompleted')
}

function downloadCurrentWorkshopMarkdown(): void {
  if (!workshop.value) {
    return
  }
  const markdown = generateMarkdown(workshop.value, nodes.value)
  const fileName = `${workshop.value.title}.md`
  downloadMarkdown(markdown, fileName)
}

function goBackToWorkshop(): void {
  if (!workshopId.value) {
    void router.push('/dashboard')
    return
  }
  void router.push(`/workshop/${workshopId.value}`)
}

onMounted(async () => {
  if (!workshopId.value) {
    return
  }
  await workshopStore.loadWorkshop(workshopId.value)
})
</script>

<template>
  <div class="min-h-screen bg-gray-50 px-4 py-6 dark:bg-gray-900 sm:px-6">
    <div class="mx-auto flex w-full max-w-3xl flex-col gap-4">
      <header class="no-print flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-gray-100">{{ t('export.title') }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('export.preview') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <BaseButton size="sm" variant="outline" @click="goBackToWorkshop">
            <span class="inline-flex items-center gap-1.5">
              <ArrowLeftIcon class="h-4 w-4" />
              {{ t('export.backToWorkshop') }}
            </span>
          </BaseButton>
          <BaseButton size="sm" variant="secondary" @click="downloadCurrentWorkshopMarkdown">
            <span class="inline-flex items-center gap-1.5">
              <ArrowDownTrayIcon class="h-4 w-4" />
              {{ t('export.downloadMarkdown') }}
            </span>
          </BaseButton>
          <BaseButton size="sm" @click="triggerPrint">
            <span class="inline-flex items-center gap-1.5">
              <PrinterIcon class="h-4 w-4" />
              {{ t('export.exportPdf') }}
            </span>
          </BaseButton>
        </div>
      </header>

      <article class="print-preview rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800 sm:p-8">
        <template v-if="workshop">
          <header class="mb-6 border-b border-gray-200 pb-4 dark:border-gray-600">
            <h2 class="text-2xl font-bold text-gray-900 dark:text-gray-100">{{ workshop.title }}</h2>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ workshop.description }}</p>
            <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">
              {{ t('export.generatedAt', { date: generatedAt }) }}
            </p>
          </header>

          <section class="space-y-6">
            <article v-for="node in nodes" :key="node.id" class="space-y-3">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ node.order }}. {{ node.title }}</h3>
              <blockquote class="border-l-4 border-indigo-200 pl-3 text-sm text-gray-600 dark:border-indigo-500/60 dark:text-gray-300">
                {{ node.description }}
              </blockquote>
              <p class="whitespace-pre-wrap text-sm leading-7 text-gray-800 dark:text-gray-100">{{ nodeContent(node.content) }}</p>
            </article>
          </section>
        </template>

        <p v-else class="text-sm text-gray-600 dark:text-gray-300">{{ t('export.noContent') }}</p>
      </article>
    </div>
  </div>
</template>
