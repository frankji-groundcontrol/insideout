<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RoadmapNode } from '@/types'
import { MapIcon } from '@heroicons/vue/24/outline'
import { CheckCircleIcon, LockClosedIcon, PlayCircleIcon, EllipsisHorizontalCircleIcon } from '@heroicons/vue/24/solid'

interface Props {
  nodes: RoadmapNode[]
  activeNodeId?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'select', id: string): void
}>()
const { t } = useI18n()

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
  <div class="flex h-full w-80 flex-col border-r border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
    <div class="border-b border-gray-100 p-6 dark:border-gray-700">
      <h3 class="flex items-center text-lg font-bold text-gray-900 dark:text-gray-100">
        <MapIcon class="mr-2 h-5 w-5 text-indigo-500 dark:text-indigo-400" />
        {{ t('workshop.roadmap') }}
      </h3>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('workshop.roadmapDesc') }}</p>
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
          class="absolute bottom-0 left-6 top-10 -ml-px w-0.5 bg-gray-100 group-last:hidden dark:bg-gray-700"
        ></div>

        <div 
          class="relative flex items-center p-3 rounded-lg transition-all duration-200 cursor-pointer"
          :class="[
            activeNodeId === node.id ? 'bg-indigo-600 text-white shadow-md' : 'text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-700',
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
                :class="activeNodeId === node.id ? 'text-indigo-200' : 'text-gray-400 dark:text-gray-400'"
            >
              {{ t('workshop.step') }} {{ node.order }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
