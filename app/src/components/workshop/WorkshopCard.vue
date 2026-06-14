<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Workshop } from '@/types'
import {
  UserGroupIcon,
  ClockIcon,
  CheckCircleIcon,
  DocumentIcon,
  PhotoIcon,
} from '@heroicons/vue/24/outline'

interface Props {
  workshop: Workshop
}

const props = defineProps<Props>()
const { t } = useI18n()

const statusConfig = computed(() => {
  switch (props.workshop.status) {
    case 'active':
      return { label: t('workshop.status.active'), color: 'text-green-700 bg-green-50 dark:text-green-300 dark:bg-green-900/40', icon: ClockIcon }
    case 'completed':
      return { label: t('workshop.status.completed'), color: 'text-gray-600 bg-gray-50 dark:text-gray-300 dark:bg-gray-700', icon: CheckCircleIcon }
    case 'draft':
      return { label: t('workshop.status.draft'), color: 'text-yellow-700 bg-yellow-50 dark:text-yellow-300 dark:bg-yellow-900/40', icon: DocumentIcon }
    default:
      return { label: t('workshop.status.unknown'), color: 'text-gray-400 bg-gray-50 dark:text-gray-300 dark:bg-gray-700', icon: DocumentIcon }
  }
})
</script>

<template>
  <NuxtLink
    :to="`/workshop/${workshop.id}`"
    class="group block h-full cursor-pointer overflow-hidden rounded-xl border border-gray-100 bg-white shadow-sm transition-all duration-200 hover:shadow-md dark:border-gray-700 dark:bg-gray-800"
  >
    <!-- Cover Image (16:9) -->
    <div class="relative aspect-video w-full overflow-hidden bg-gray-100 dark:bg-gray-700">
      <img 
        v-if="workshop.cover_url" 
        :src="workshop.cover_url" 
        :alt="workshop.title"
        class="w-full h-full object-cover transform group-hover:scale-105 transition-transform duration-500"
      />
      <div v-else class="flex h-full w-full items-center justify-center text-gray-400 dark:text-gray-300">
        <PhotoIcon class="h-12 w-12" />
      </div>
      
      <!-- Status Badge -->
      <div class="absolute top-3 right-3">
        <span 
          class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium shadow-sm backdrop-blur-sm"
          :class="statusConfig.color"
        >
          <component :is="statusConfig.icon" class="w-3 h-3 mr-1" />
          {{ statusConfig.label }}
        </span>
      </div>
    </div>

    <!-- Content -->
    <div class="p-5 flex-grow flex flex-col">
      <h3 class="mb-2 line-clamp-1 text-lg font-bold text-gray-900 transition-colors group-hover:text-indigo-600 dark:text-gray-100 dark:group-hover:text-indigo-300">
        {{ workshop.title }}
      </h3>
      <p class="mb-4 flex-grow line-clamp-2 text-sm text-gray-500 dark:text-gray-400">
        {{ workshop.description }}
      </p>

      <!-- Footer Info -->
      <div class="mt-auto flex items-center justify-between border-t border-gray-100 pt-4 text-xs text-gray-400 dark:border-gray-700 dark:text-gray-400">
        <div class="flex items-center">
          <UserGroupIcon class="w-4 h-4 mr-1" />
          <span>{{ t('workshop.members', { count: workshop.member_count }) }}</span>
        </div>
        <div>
          {{ t('workshop.codeLabel') }}: {{ workshop.code }}
        </div>
      </div>
    </div>
  </NuxtLink>
</template>
