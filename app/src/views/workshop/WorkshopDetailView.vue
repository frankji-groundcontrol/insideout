<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { mockApi } from '@/services/mock'
import type { Workshop, RoadmapNode } from '@/types'
import RoadmapSidebar from '@/components/workshop/RoadmapSidebar.vue'
import TaskDetail from '@/components/workshop/TaskDetail.vue'

const route = useRoute()
const loading = ref(true)
const workshop = ref<Workshop | null>(null)
const nodes = ref<RoadmapNode[]>([])
const activeNodeId = ref<string>('')

onMounted(async () => {
  const workshopId = route.params.id as string
  try {
    const [workshopData, nodesData] = await Promise.all([
      mockApi.workshop.getById(workshopId),
      mockApi.workshop.getRoadmap(workshopId)
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
  <div class="h-[calc(100vh-64px)] flex overflow-hidden bg-white">
    <!-- Sidebar -->
    <RoadmapSidebar 
      v-if="!loading"
      :nodes="nodes"
      :active-node-id="activeNodeId"
      @select="id => activeNodeId = id"
      class="flex-shrink-0"
    />

    <!-- Main Content -->
    <main class="flex-grow min-w-0">
      <div v-if="loading" class="h-full flex items-center justify-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
      </div>
      
      <TaskDetail 
        v-else-if="activeNode" 
        :node="activeNode" 
      />
      
      <div v-else class="h-full flex items-center justify-center text-gray-500">
        选择左侧的一个任务开始吧 👈
      </div>
    </main>
  </div>
</template>
