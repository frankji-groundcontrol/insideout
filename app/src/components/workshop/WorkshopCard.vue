<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Workshop } from '@/types'
import { UserGroupIcon, ClockIcon, CheckCircleIcon, DocumentIcon } from '@heroicons/vue/24/outline'

interface Props {
  workshop: Workshop
}

const props = defineProps<Props>()

const statusConfig = computed(() => {
  switch (props.workshop.status) {
    case 'active':
      return { label: '进行中', color: 'text-green-600 bg-green-50', icon: ClockIcon }
    case 'completed':
      return { label: '已结束', color: 'text-gray-600 bg-gray-50', icon: CheckCircleIcon }
    case 'draft':
      return { label: '草稿', color: 'text-yellow-600 bg-yellow-50', icon: DocumentIcon }
    default:
      return { label: '未知', color: 'text-gray-400 bg-gray-50', icon: DocumentIcon }
  }
})
</script>

<template>
  <RouterLink 
    :to="`/workshop/${workshop.id}`"
    class="group bg-white rounded-xl shadow-sm hover:shadow-md border border-gray-100 transition-all duration-200 overflow-hidden cursor-pointer flex flex-col h-full block"
  >
    <!-- Cover Image (16:9) -->
    <div class="aspect-video w-full bg-gray-100 relative overflow-hidden">
      <img 
        v-if="workshop.cover_url" 
        :src="workshop.cover_url" 
        :alt="workshop.title"
        class="w-full h-full object-cover transform group-hover:scale-105 transition-transform duration-500"
      />
      <div v-else class="w-full h-full flex items-center justify-center text-gray-400">
        <span class="text-4xl">🌯</span>
      </div>
      
      <!-- Status Badge -->
      <div class="absolute top-3 right-3">
        <span 
          class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium backdrop-blur-sm bg-white/90 shadow-sm"
          :class="statusConfig.color.replace('bg-', 'text-')"
        >
          <component :is="statusConfig.icon" class="w-3 h-3 mr-1" />
          {{ statusConfig.label }}
        </span>
      </div>
    </div>

    <!-- Content -->
    <div class="p-5 flex-grow flex flex-col">
      <h3 class="text-lg font-bold text-gray-900 mb-2 line-clamp-1 group-hover:text-indigo-600 transition-colors">
        {{ workshop.title }}
      </h3>
      <p class="text-sm text-gray-500 line-clamp-2 mb-4 flex-grow">
        {{ workshop.description }}
      </p>

      <!-- Footer Info -->
      <div class="flex items-center justify-between text-xs text-gray-400 pt-4 border-t border-gray-50 mt-auto">
        <div class="flex items-center">
          <UserGroupIcon class="w-4 h-4 mr-1" />
          <span>{{ workshop.member_count }} 成员</span>
        </div>
        <div>
          ID: {{ workshop.code }}
        </div>
      </div>
    </div>
  </RouterLink>
</template>
