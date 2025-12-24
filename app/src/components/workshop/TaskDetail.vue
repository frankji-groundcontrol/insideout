<script setup lang="ts">
import { ref } from 'vue'
import type { RoadmapNode } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import { PaperAirplaneIcon, ChatBubbleLeftRightIcon } from '@heroicons/vue/24/outline'

interface Props {
  node: RoadmapNode
}

const props = defineProps<Props>()
const homework = ref('')

const handleSubmit = () => {
  // TODO: Submit homework
  console.log('Submit:', homework.value)
}
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <!-- Header -->
    <div class="bg-white px-8 py-6 border-b border-gray-200 shadow-sm">
      <div class="flex items-center space-x-2 text-sm text-gray-500 mb-2">
        <span>Step {{ node.order }}</span>
        <span>•</span>
        <span 
          class="capitalize"
          :class="{
            'text-green-600': node.status === 'completed',
            'text-indigo-600': node.status === 'in_progress',
            'text-gray-400': node.status === 'pending'
          }"
        >
          {{ node.status === 'in_progress' ? '进行中' : node.status }}
        </span>
      </div>
      <h1 class="text-2xl font-bold text-gray-900">{{ node.title }}</h1>
    </div>

    <!-- Content Area -->
    <div class="flex-grow p-8 overflow-y-auto">
      <div class="w-full space-y-8">
        <!-- Task Description -->
        <div class="bg-white rounded-xl shadow-sm p-6 border border-gray-100">
          <h2 class="text-lg font-bold text-gray-900 mb-4">📝 任务说明</h2>
          <p class="text-gray-600 leading-relaxed">{{ node.description }}</p>
        </div>

        <!-- Submission Area -->
        <div class="bg-white rounded-xl shadow-sm p-6 border border-gray-100">
          <h2 class="text-lg font-bold text-gray-900 mb-4">📤 提交作业</h2>
          <div class="space-y-4">
            <textarea
              v-model="homework"
              rows="6"
              class="block w-full rounded-lg border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-4 border bg-gray-50"
              placeholder="在这里写下你的思考，或者粘贴你的代码链接..."
            ></textarea>
            <div class="flex justify-end">
              <BaseButton @click="handleSubmit">
                <PaperAirplaneIcon class="w-4 h-4 mr-2" />
                提交作业
              </BaseButton>
            </div>
          </div>
        </div>

        <!-- Discussion Placeholder -->
        <div class="bg-white rounded-xl shadow-sm p-6 border border-gray-100 opacity-75">
          <h2 class="text-lg font-bold text-gray-900 mb-4 flex items-center">
            <ChatBubbleLeftRightIcon class="w-5 h-5 mr-2" />
            讨论区 (Coming Soon)
          </h2>
          <p class="text-gray-500 text-sm">大家可以在这里交流心得...</p>
        </div>
      </div>
    </div>
  </div>
</template>
