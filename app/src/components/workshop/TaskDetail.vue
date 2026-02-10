<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RoadmapNode } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import {
  ArrowUpTrayIcon,
  ChatBubbleLeftRightIcon,
  DocumentTextIcon,
  PaperAirplaneIcon,
} from '@heroicons/vue/24/outline'

interface Props {
  node: RoadmapNode
}

const props = defineProps<Props>()
const homework = ref('')
const { t } = useI18n()

const getStatusLabel = (status: RoadmapNode['status']) => {
  switch (status) {
    case 'in_progress':
      return t('workshop.status.inProgress')
    case 'completed':
      return t('workshop.status.completed')
    case 'pending':
      return t('workshop.status.pending')
    case 'locked':
      return t('workshop.status.locked')
    default:
      return status
  }
}

const handleSubmit = () => {
  // TODO: Submit homework
  console.log('Submit:', homework.value)
}
</script>

<template>
  <div class="flex h-full flex-col bg-gray-50 dark:bg-gray-900">
    <!-- 头部 -->
    <div class="border-b border-gray-200 bg-white px-8 py-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div class="mb-2 flex items-center space-x-2 text-sm text-gray-500 dark:text-gray-400">
        <span>{{ t('workshop.step') }} {{ node.order }}</span>
        <span>•</span>
        <span 
          class="capitalize"
          :class="{
            'text-green-600': node.status === 'completed',
            'text-indigo-600': node.status === 'in_progress',
            'text-gray-400': node.status === 'pending' || node.status === 'locked'
          }"
        >
          {{ getStatusLabel(node.status) }}
        </span>
      </div>
      <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">{{ node.title }}</h1>
    </div>

    <!-- 内容区 -->
    <div class="flex-grow p-8 overflow-y-auto">
      <div class="w-full space-y-8">
        <!-- 任务说明 -->
        <div class="rounded-xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 class="mb-4 flex items-center text-lg font-bold text-gray-900 dark:text-gray-100">
            <DocumentTextIcon class="mr-2 h-5 w-5 text-indigo-500 dark:text-indigo-400" />
            {{ t('workshop.taskDescription') }}
          </h2>
          <p class="leading-relaxed text-gray-600 dark:text-gray-400">{{ node.description }}</p>
        </div>

        <!-- 提交作业 -->
        <div class="rounded-xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 class="mb-4 flex items-center text-lg font-bold text-gray-900 dark:text-gray-100">
            <ArrowUpTrayIcon class="mr-2 h-5 w-5 text-indigo-500 dark:text-indigo-400" />
            {{ t('workshop.submitHomework') }}
          </h2>
          <div class="space-y-4">
            <textarea
              v-model="homework"
              rows="6"
              class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-4 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:border-gray-700 dark:bg-gray-700 dark:text-gray-100"
              :placeholder="t('workshop.textareaPlaceholder')"
            ></textarea>
            <div class="flex justify-end">
              <BaseButton @click="handleSubmit">
                <PaperAirplaneIcon class="w-4 h-4 mr-2" />
                {{ t('workshop.submitButton') }}
              </BaseButton>
            </div>
          </div>
        </div>

        <!-- 讨论区占位 -->
        <div class="rounded-xl border border-gray-100 bg-white p-6 opacity-75 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 class="mb-4 flex items-center text-lg font-bold text-gray-900 dark:text-gray-100">
            <ChatBubbleLeftRightIcon class="w-5 h-5 mr-2" />
            {{ t('workshop.discussion') }} ({{ t('workshop.comingSoon') }})
          </h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('workshop.discussionHint') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
