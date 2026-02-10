<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { mockApi } from '@/services/mock'
import type { Workshop, UserProfile } from '@/types'
import WorkshopCard from '@/components/workshop/WorkshopCard.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import { HandRaisedIcon, FolderIcon, PlusIcon, StarIcon, UserPlusIcon } from '@heroicons/vue/24/outline'

const loading = ref(true)
const user = ref<UserProfile | null>(null)
const workshops = ref<Workshop[]>([])
const { t } = useI18n()

// 获取问候语
const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return t('dashboard.greeting.morning')
  if (hour < 18) return t('dashboard.greeting.afternoon')
  return t('dashboard.greeting.evening')
})

// 分类工作坊
const joinedWorkshops = computed(() => workshops.value.filter(w => w.is_joined))
const managedWorkshops = computed(() => workshops.value.filter(w => w.creator_id === user.value?.id))

onMounted(async () => {
  try {
    const [userData, workshopData] = await Promise.all([
      mockApi.auth.getCurrentUser(),
      mockApi.workshop.list()
    ])
    user.value = userData
    workshops.value = workshopData
  } catch (e) {
    console.error('Failed to load dashboard data', e)
  } finally {
    loading.value = false
  }
})

const handleCreate = () => {
  // TODO: 跳转到创建页
  console.log('Create workshop')
}

const handleJoin = () => {
  // TODO: 弹出加入弹窗
  console.log('Join workshop')
}
</script>

<template>
  <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
    <!-- 顶部区域 -->
    <div class="md:flex md:items-center md:justify-between mb-8">
      <div class="flex-1 min-w-0">
        <h2 class="text-2xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:text-3xl sm:truncate">
          <span class="inline-flex items-center gap-2">
            <HandRaisedIcon class="h-6 w-6 text-indigo-500 dark:text-indigo-400" />
            <span>{{ greeting }}，{{ user?.username || t('dashboard.friend') }}！</span>
          </span>
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('dashboard.vibeMessage') }}
        </p>
      </div>
      <div class="mt-4 flex md:mt-0 md:ml-4 space-x-3">
        <BaseButton variant="outline" @click="handleJoin">
          <UserPlusIcon class="w-5 h-5 mr-2 -ml-1" />
          {{ t('dashboard.joinWorkshop') }}
        </BaseButton>
        <BaseButton @click="handleCreate">
          <PlusIcon class="w-5 h-5 mr-2 -ml-1" />
          {{ t('dashboard.createWorkshop') }}
        </BaseButton>
      </div>
    </div>

    <!-- 骨架屏 -->
    <div v-if="loading" class="animate-pulse space-y-8">
      <div class="mb-4 h-8 w-1/4 rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        <div v-for="i in 3" :key="i" class="h-64 rounded-xl bg-gray-200 dark:bg-gray-700"></div>
      </div>
    </div>

    <div v-else class="space-y-12">
      <!-- 我参与的 -->
      <section>
        <h3 class="mb-4 flex items-center text-lg font-medium leading-6 text-gray-900 dark:text-gray-100">
          <FolderIcon class="mr-2 h-5 w-5 text-indigo-500 dark:text-indigo-400" />
          {{ t('dashboard.joinedTitle') }} ({{ joinedWorkshops.length }})
        </h3>
        
        <div v-if="joinedWorkshops.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-[repeat(auto-fit,minmax(400px,1fr))] gap-6">
          <WorkshopCard 
            v-for="workshop in joinedWorkshops" 
            :key="workshop.id" 
            :workshop="workshop" 
          />
        </div>
        
        <!-- 我参与的空态 -->
        <div v-else class="rounded-xl border border-dashed border-gray-300 bg-white py-12 text-center dark:border-gray-700 dark:bg-gray-800">
          <p class="text-gray-500 dark:text-gray-400">{{ t('dashboard.emptyJoined') }}</p>
        </div>
      </section>

      <!-- 我管理的 -->
      <section>
        <h3 class="mb-4 flex items-center text-lg font-medium leading-6 text-gray-900 dark:text-gray-100">
          <StarIcon class="mr-2 h-5 w-5 text-amber-500 dark:text-amber-400" />
          {{ t('dashboard.managedTitle') }} ({{ managedWorkshops.length }})
        </h3>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <WorkshopCard 
            v-for="workshop in managedWorkshops" 
            :key="workshop.id" 
            :workshop="workshop" 
          />
          
          <!-- 创建工作坊占位卡 -->
          <button 
            @click="handleCreate"
            class="group flex h-full min-h-[280px] flex-col items-center justify-center rounded-xl border-2 border-dashed border-gray-300 p-6 text-center transition-all duration-200 hover:border-indigo-500 hover:bg-indigo-50 dark:border-gray-700 dark:hover:border-indigo-400 dark:hover:bg-gray-700"
          >
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 transition-colors group-hover:bg-indigo-100 dark:bg-gray-700 dark:group-hover:bg-gray-600">
              <PlusIcon class="h-6 w-6 text-gray-400 group-hover:text-indigo-600 dark:text-gray-300 dark:group-hover:text-indigo-400" />
            </div>
            <h3 class="text-sm font-medium text-gray-900 group-hover:text-indigo-700 dark:text-gray-100 dark:group-hover:text-indigo-300">{{ t('dashboard.createNewTitle') }}</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.createNewDesc') }}</p>
          </button>
        </div>
      </section>
    </div>
  </div>
</template>
