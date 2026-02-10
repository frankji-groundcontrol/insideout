<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RoadmapNode } from '@/types'
import { CheckCircleIcon, LockClosedIcon, PlayIcon } from '@heroicons/vue/24/solid'
import { StopCircleIcon as CircleIcon } from '@heroicons/vue/24/outline'

interface Props {
  node: RoadmapNode
  isActive: boolean
  showConnector?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showConnector: false,
})

const emit = defineEmits<{
  (e: 'select', id: string): void
}>()

const { t } = useI18n()

const isLocked = computed(() => props.node.status === 'locked')

const statusIcon = computed(() => {
  switch (props.node.status) {
    case 'locked':
      return LockClosedIcon
    case 'in_progress':
      return PlayIcon
    case 'completed':
      return CheckCircleIcon
    default:
      return CircleIcon
  }
})

const itemClass = computed(() => {
  if (props.isActive) {
    return 'bg-indigo-600 text-white shadow-md ring-1 ring-indigo-400/40'
  }

  if (isLocked.value) {
    return 'cursor-not-allowed border border-gray-200/80 bg-gray-50 text-gray-400 opacity-70 dark:border-gray-700 dark:bg-gray-800/60 dark:text-gray-500'
  }

  return 'cursor-pointer border border-transparent bg-white text-gray-700 hover:border-indigo-200 hover:bg-indigo-50/60 dark:bg-gray-800 dark:text-gray-200 dark:hover:border-indigo-500/40 dark:hover:bg-indigo-900/20'
})

const iconClass = computed(() => {
  if (props.isActive) {
    return 'text-white'
  }

  switch (props.node.status) {
    case 'locked':
      return 'text-gray-400 dark:text-gray-500'
    case 'pending':
      return 'text-blue-500 dark:text-blue-400'
    case 'in_progress':
      return 'text-indigo-500 dark:text-indigo-400'
    case 'completed':
      return 'text-green-500 dark:text-green-400'
    default:
      return 'text-gray-400 dark:text-gray-500'
  }
})

const stepTextClass = computed(() =>
  props.isActive ? 'text-indigo-100' : 'text-gray-400 dark:text-gray-500',
)

const connectorClass = computed(() => {
  if (props.node.status === 'completed') {
    return 'bg-green-300/80 dark:bg-green-500/50'
  }
  return 'bg-gray-200 dark:bg-gray-700'
})

function handleSelect() {
  if (isLocked.value) {
    return
  }
  emit('select', props.node.id)
}
</script>

<template>
  <div class="relative">
    <div
      v-if="showConnector"
      class="pointer-events-none absolute left-6 top-10 h-[calc(100%-2.5rem)] w-px"
      :class="connectorClass"
    />

    <button
      type="button"
      class="relative flex w-full items-center gap-3 rounded-xl p-3 text-left transition-all duration-200"
      :class="itemClass"
      :disabled="isLocked"
      @click="handleSelect"
    >
      <component :is="statusIcon" class="h-6 w-6 shrink-0" :class="iconClass" />

      <div class="min-w-0">
        <p class="truncate text-sm font-semibold">{{ node.title }}</p>
        <p class="mt-0.5 text-xs" :class="stepTextClass">
          {{ t('workshop.step') }} {{ node.order }} · {{ t(`workshop.status.${node.status === 'in_progress' ? 'inProgress' : node.status}`) }}
        </p>
      </div>
    </button>
  </div>
</template>
