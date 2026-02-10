<script setup lang="ts">
/* biome-ignore-all assist/source/organizeImports: Vue SFC 按语义分组导入 */
/* biome-ignore-all lint/correctness/noUnusedImports: 模板中会使用导入内容 */
/* biome-ignore-all lint/correctness/noUnusedVariables: 模板中会使用变量 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeftIcon } from '@heroicons/vue/24/outline'
import AiSidebar from '@/components/workshop/AiSidebar.vue'
import RoadmapSidebar from '@/components/workshop/RoadmapSidebar.vue'
import { useWorkshopStore } from '@/stores/workshop'
import { useUserStore } from '@/stores/user'
import WorkshopActionBar from './components/WorkshopActionBar.vue'
import { useWorkshopDraftKey } from './composables/useWorkshopDraftKey'
import { useWorkshopDetailSession } from './composables/useWorkshopDetailSession'
import { useWorkshopViewport } from './composables/useWorkshopViewport'
import TaskEditor from '@/views/workshop/TaskEditor.vue'

type MobileTab = 'roadmap' | 'editor' | 'ai'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const workshopStore = useWorkshopStore()

const mobileTab = ref<MobileTab>('editor')
const activeNodeId = computed(() => workshopStore.currentNodeId)
const workshopId = computed(() => String(route.params.id ?? ''))
const userId = computed(() => userStore.user?.id ?? 'guest')

const { isMobile } = useWorkshopViewport()

const { activeDraftKey, createDraftKey } = useWorkshopDraftKey({
  userId,
  workshopId,
  activeNodeId,
})

const useWorkshopDetailSessionInstance = useWorkshopDetailSession({
  workshopId,
  activeNodeId,
  createDraftKey,
  router,
})

const { nodes, activeNode, loading, switchNodeWithDraft, handleCompleteNode, goToExportPage } =
  useWorkshopDetailSessionInstance
</script>

<template>
  <div class="h-[calc(100vh-64px)] overflow-hidden bg-gray-50 dark:bg-gray-900">
    <div v-if="loading" class="flex h-full items-center justify-center">
      <div class="h-12 w-12 animate-spin rounded-full border-b-2 border-indigo-600"></div>
    </div>

    <div
      v-else-if="activeNode"
      class="h-full min-h-0"
    >
      <section v-if="isMobile" class="flex h-full min-h-0 flex-col bg-gray-50 dark:bg-gray-900">
        <nav class="border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
          <div class="grid grid-cols-3 px-2 py-2">
            <button
              type="button"
              class="rounded-lg px-2 py-2 text-sm font-medium transition"
              :class="mobileTab === 'roadmap'
                ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-200'
                : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700/70'"
              @click="mobileTab = 'roadmap'"
            >
              {{ t('workshop.tabs.roadmap') }}
            </button>
            <button
              type="button"
              class="rounded-lg px-2 py-2 text-sm font-medium transition"
              :class="mobileTab === 'editor'
                ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-200'
                : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700/70'"
              @click="mobileTab = 'editor'"
            >
              {{ t('workshop.tabs.editor') }}
            </button>
            <button
              type="button"
              class="rounded-lg px-2 py-2 text-sm font-medium transition"
              :class="mobileTab === 'ai'
                ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-200'
                : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700/70'"
              @click="mobileTab = 'ai'"
            >
              {{ t('workshop.tabs.ai') }}
            </button>
          </div>
        </nav>

        <div class="flex-1 min-h-0">
          <div v-show="mobileTab === 'roadmap'" class="h-full min-h-0">
            <RoadmapSidebar
              :nodes="nodes"
              :active-node-id="activeNodeId"
              class="h-full w-full border-r-0"
              @select="switchNodeWithDraft"
            />
          </div>

          <div v-show="mobileTab === 'editor'" class="flex h-full min-h-0 flex-col overflow-y-auto bg-gray-50 px-4 py-4 dark:bg-gray-900">
            <div class="mb-3 flex items-center justify-between">
              <h2 class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ activeNode.title }}</h2>
              <WorkshopActionBar
                :export-text="t('export.title')"
                :submit-text="t('workshop.submitButton')"
                @export="goToExportPage"
                @submit="handleCompleteNode"
              />
            </div>
            <KeepAlive>
              <TaskEditor :draft-key="activeDraftKey" />
            </KeepAlive>
          </div>

          <div v-show="mobileTab === 'ai'" class="h-full min-h-0 border-t border-gray-200 dark:border-gray-700">
            <AiSidebar :node-id="activeNodeId" class="h-full" />
          </div>
        </div>
      </section>

      <section v-else class="flex h-full min-h-0 bg-gray-50 dark:bg-gray-900">
        <RoadmapSidebar
          :nodes="nodes"
          :active-node-id="activeNodeId"
          class="h-full w-72 flex-shrink-0 xl:w-80"
          @select="switchNodeWithDraft"
        />

        <main class="flex min-w-0 flex-1 flex-col overflow-y-auto bg-gray-50 px-6 py-6 dark:bg-gray-900">
          <header class="mb-4 flex items-center justify-between">
            <div>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('workshop.step') }} {{ activeNode.order }}</p>
              <h1 class="text-xl font-semibold text-gray-900 dark:text-gray-100">{{ activeNode.title }}</h1>
            </div>
            <WorkshopActionBar
              :export-text="t('export.title')"
              :submit-text="t('workshop.submitButton')"
              @export="goToExportPage"
              @submit="handleCompleteNode"
            />
          </header>

          <TaskEditor :draft-key="activeDraftKey" />
        </main>

        <aside class="h-full w-80 flex-shrink-0 border-l border-gray-200 dark:border-gray-700">
          <AiSidebar :node-id="activeNodeId" class="h-full" />
        </aside>
      </section>
    </div>

    <div v-else class="flex h-full items-center justify-center text-gray-500 dark:text-gray-400">
      <div class="inline-flex items-center gap-2">
        <ArrowLeftIcon class="h-5 w-5" />
        <span>{{ t('workshop.selectTask') }}</span>
      </div>
    </div>
  </div>
</template>
