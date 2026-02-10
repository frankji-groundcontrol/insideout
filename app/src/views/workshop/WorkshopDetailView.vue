<script setup lang="ts">
/* biome-ignore-all assist/source/organizeImports: Vue SFC 按语义分组导入 */
/* biome-ignore-all lint/correctness/noUnusedImports: 模板中会使用导入内容 */
/* biome-ignore-all lint/correctness/noUnusedVariables: 模板中会使用变量 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { ArrowLeftIcon } from '@heroicons/vue/24/outline'
import BaseButton from '@/components/common/BaseButton.vue'
import AiSidebar from '@/components/workshop/AiSidebar.vue'
import RoadmapSidebar from '@/components/workshop/RoadmapSidebar.vue'
import { useEditorStore } from '@/stores/editor'
import { useUserStore } from '@/stores/user'
import { useWorkshopStore } from '@/stores/workshop'
import TaskEditor from './TaskEditor.vue'

type MobileTab = 'roadmap' | 'editor' | 'ai'

const route = useRoute()
const { t } = useI18n()
const workshopStore = useWorkshopStore()
const editorStore = useEditorStore()
const userStore = useUserStore()

const mobileTab = ref<MobileTab>('editor')
const isMobile = ref(false)
const mediaQuery = window.matchMedia('(max-width: 767px)')
const isSwitchingNode = ref(false)

const workshopId = computed(() => String(route.params.id ?? ''))
const nodes = computed(() => workshopStore.roadmapNodes)
const activeNodeId = computed(() => workshopStore.currentNodeId)
const activeNode = computed(() => workshopStore.currentNode)
const loading = computed(() => workshopStore.loading)
const userId = computed(() => userStore.user?.id ?? 'guest')
const activeDraftKey = computed(() => {
  if (!activeNodeId.value) {
    return ''
  }
  return createDraftKey(activeNodeId.value)
})

function createDraftKey(nodeId: string): string {
  return `draft:${userId.value}:${workshopId.value}:${nodeId}:v1`
}

function updateViewportState(event: MediaQueryList | MediaQueryListEvent) {
  isMobile.value = event.matches
}

async function switchNodeWithDraft(nodeId: string) {
  if (!nodeId || nodeId === activeNodeId.value || isSwitchingNode.value) {
    return
  }

  isSwitchingNode.value = true
  try {
    // 切换节点前先落盘，避免当前编辑内容丢失。
    await editorStore.flush()
    workshopStore.selectNode(nodeId)
  } finally {
    isSwitchingNode.value = false
  }
}

async function handleCompleteNode() {
  const nodeId = activeNodeId.value
  if (!nodeId) {
    return
  }

  await editorStore.flush()
  await workshopStore.completeNode(nodeId)

  // 若 store 尚未切到下一节点，则兜底激活下一个 pending 节点。
  if (workshopStore.currentNodeId === nodeId) {
    const nextPending = workshopStore.roadmapNodes.find((node) => node.status === 'pending')
    if (nextPending) {
      workshopStore.selectNode(nextPending.id)
    }
  }
}

watch(
  () => workshopStore.currentNodeId,
  async (nodeId, previousNodeId) => {
    if (!nodeId || nodeId === previousNodeId) {
      return
    }
    await editorStore.loadDraft(createDraftKey(nodeId))
  },
)

watch(
  () => route.params.id,
  async (value) => {
    const nextWorkshopId = String(value ?? '')
    if (!nextWorkshopId) {
      return
    }

    await workshopStore.loadWorkshop(nextWorkshopId)
    if (workshopStore.currentNodeId) {
      await editorStore.loadDraft(createDraftKey(workshopStore.currentNodeId))
    }
  },
)

onMounted(async () => {
  updateViewportState(mediaQuery)
  mediaQuery.addEventListener('change', updateViewportState)

  if (!workshopId.value) {
    return
  }

  await workshopStore.loadWorkshop(workshopId.value)
  if (workshopStore.currentNodeId) {
    await editorStore.loadDraft(createDraftKey(workshopStore.currentNodeId))
  }
})

onBeforeUnmount(async () => {
  mediaQuery.removeEventListener('change', updateViewportState)
  await editorStore.flush()
})
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
              <BaseButton size="sm" @click="handleCompleteNode">
                {{ t('workshop.submitButton') }}
              </BaseButton>
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
            <BaseButton size="sm" @click="handleCompleteNode">
              {{ t('workshop.submitButton') }}
            </BaseButton>
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
