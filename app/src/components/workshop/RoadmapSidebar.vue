<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RoadmapNode } from '@/types'
import { MapIcon } from '@heroicons/vue/24/outline'
import RoadmapItem from '@/components/workshop/RoadmapItem.vue'

interface Props {
  nodes: RoadmapNode[]
  activeNodeId?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'select', id: string): void
}>()
const { t } = useI18n()
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
    
    <div class="flex-grow space-y-2 overflow-y-auto p-4">
      <RoadmapItem
        v-for="(node, index) in nodes"
        :key="node.id"
        :node="node"
        :is-active="activeNodeId === node.id"
        :show-connector="index !== nodes.length - 1"
        @select="emit('select', $event)"
      />
    </div>
  </div>
</template>
