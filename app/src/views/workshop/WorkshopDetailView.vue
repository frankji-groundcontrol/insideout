<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { services } from '@/services/registry'
import type { Workshop, RoadmapNode } from '@/types'
import RoadmapSidebar from '@/components/workshop/RoadmapSidebar.vue'
import TaskDetail from '@/components/workshop/TaskDetail.vue'
import { ArrowLeftIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const loading = ref(true)
const workshop = ref<Workshop | null>(null)
const nodes = ref<RoadmapNode[]>([])
const activeNodeId = ref<string>('')
const { t } = useI18n()

onMounted(async () => {
  const workshopId = route.params.id as string
  try {
    const [workshopData, nodesData] = await Promise.all([
      services.workshop.getWorkshop(workshopId),
      services.workshop.getRoadmap(workshopId)
    ])
    
    if (workshopData) workshop.value = workshopData
    nodes.value = nodesData
    
    // 默认选中第一个非锁定节点，或者第一个节点
    const firstActive = nodesData.find(n => n.status === 'in_progress') || nodesData[0]
    if (firstActive) activeNodeId.value = firstActive.id
    
  } catch (e) {
    console.error('Failed to load workshop detail', e)
  } finally {
    loading.value = false
  }
})

const activeNode = computed(() => nodes.value.find(n => n.id === activeNodeId.value))
</script>

<template>
  <div class="flex h-[calc(100vh-64px)] overflow-hidden bg-white dark:bg-gray-900">
    <!-- 侧边栏 -->
    <RoadmapSidebar 
      v-if="!loading"
      :nodes="nodes"
      :active-node-id="activeNodeId"
      @select="id => activeNodeId = id"
      class="flex-shrink-0"
    />

    <!-- 主内容区 -->
    <main class="flex-grow min-w-0">
      <div v-if="loading" class="h-full flex items-center justify-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
      </div>
      
      <TaskDetail 
        v-else-if="activeNode" 
        :node="activeNode" 
      />
      
      <div v-else class="flex h-full items-center justify-center text-gray-500 dark:text-gray-400">
        <div class="inline-flex items-center gap-2">
          <ArrowLeftIcon class="h-5 w-5" />
          <span>{{ t('workshop.selectTask') }}</span>
        </div>
      </div>
    </main>
  </div>
</template>
