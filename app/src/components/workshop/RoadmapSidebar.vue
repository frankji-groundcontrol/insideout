<script setup lang="ts">
import { computed } from 'vue'
import type { RoadmapNode } from '@/types'
import { CheckCircleIcon, LockClosedIcon, PlayCircleIcon, EllipsisHorizontalCircleIcon } from '@heroicons/vue/24/solid'

interface Props {
  nodes: RoadmapNode[]
  activeNodeId?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'select', id: string): void
}>()

const getStatusIcon = (status: RoadmapNode['status']) => {
  switch (status) {
    case 'completed': return CheckCircleIcon
    case 'in_progress': return PlayCircleIcon
    case 'locked': return LockClosedIcon
    default: return EllipsisHorizontalCircleIcon
  }
}

const getStatusColor = (status: RoadmapNode['status'], isActive: boolean) => {
  if (isActive) return 'text-white'
  switch (status) {
    case 'completed': return 'text-green-500'
    case 'in_progress': return 'text-indigo-500'
    case 'locked': return 'text-gray-300'
    default: return 'text-gray-400'
  }
}
</script>

<template>
  <div class="h-full bg-white border-r border-gray-200 flex flex-col w-80">
    <div class="p-6 border-b border-gray-100">
      <h3 class="font-bold text-gray-900 text-lg">🗺️ 路线图</h3>
      <p class="text-xs text-gray-500 mt-1">按部就班，完成挑战</p>
    </div>
    
    <div class="flex-grow overflow-y-auto p-4 space-y-2">
      <div 
        v-for="(node, index) in nodes" 
        :key="node.id"
        @click="node.status !== 'locked' && emit('select', node.id)"
        class="relative group"
      >
        <!-- Connector Line -->
        <div 
          v-if="index !== nodes.length - 1"
          class="absolute left-6 top-10 bottom-0 w-0.5 bg-gray-100 -ml-px group-last:hidden"
        ></div>

        <div 
          class="relative flex items-center p-3 rounded-lg transition-all duration-200 cursor-pointer"
          :class="[
            activeNodeId === node.id ? 'bg-indigo-600 text-white shadow-md' : 'hover:bg-gray-50 text-gray-700',
            node.status === 'locked' ? 'opacity-50 cursor-not-allowed' : ''
          ]"
        >
          <!-- Icon -->
          <div class="flex-shrink-0 mr-3">
            <component 
              :is="getStatusIcon(node.status)" 
              class="w-6 h-6"
              :class="getStatusColor(node.status, activeNodeId === node.id)"
            />
          </div>
          
          <!-- Text -->
          <div>
            <p class="text-sm font-medium">{{ node.title }}</p>
            <p 
              class="text-xs mt-0.5"
              :class="activeNodeId === node.id ? 'text-indigo-200' : 'text-gray-400'"
            >
              Step {{ node.order }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
