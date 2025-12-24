<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { mockApi } from '@/services/mock'
import type { Workshop, UserProfile } from '@/types'
import WorkshopCard from '@/components/workshop/WorkshopCard.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import { PlusIcon, UserPlusIcon } from '@heroicons/vue/24/outline'

const router = useRouter()
const loading = ref(true)
const user = ref<UserProfile | null>(null)
const workshops = ref<Workshop[]>([])

// 获取问候语
const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return '早上好'
  if (hour < 18) return '下午好'
  return '晚上好'
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
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <!-- Header Section -->
    <div class="md:flex md:items-center md:justify-between mb-8">
      <div class="flex-1 min-w-0">
        <h2 class="text-2xl font-bold leading-7 text-gray-900 sm:text-3xl sm:truncate">
          👋 {{ greeting }}，{{ user?.username || '朋友' }}！
        </h2>
        <p class="mt-1 text-sm text-gray-500">
          今天也要充满 Vibe 地写代码吗？
        </p>
      </div>
      <div class="mt-4 flex md:mt-0 md:ml-4 space-x-3">
        <BaseButton variant="outline" @click="handleJoin">
          <UserPlusIcon class="w-5 h-5 mr-2 -ml-1" />
          加入工作坊
        </BaseButton>
        <BaseButton @click="handleCreate">
          <PlusIcon class="w-5 h-5 mr-2 -ml-1" />
          创建新工作坊
        </BaseButton>
      </div>
    </div>

    <!-- Skeleton Loading -->
    <div v-if="loading" class="animate-pulse space-y-8">
      <div class="h-8 bg-gray-200 rounded w-1/4 mb-4"></div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        <div v-for="i in 3" :key="i" class="h-64 bg-gray-200 rounded-xl"></div>
      </div>
    </div>

    <div v-else class="space-y-12">
      <!-- Joined Workshops -->
      <section>
        <h3 class="text-lg font-medium leading-6 text-gray-900 mb-4 flex items-center">
          <span class="mr-2">📂</span> 我参与的 ({{ joinedWorkshops.length }})
        </h3>
        
        <div v-if="joinedWorkshops.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-[repeat(auto-fit,minmax(400px,1fr))] gap-6">
          <WorkshopCard 
            v-for="workshop in joinedWorkshops" 
            :key="workshop.id" 
            :workshop="workshop" 
          />
        </div>
        
        <!-- Empty State for Joined -->
        <div v-else class="text-center py-12 bg-white rounded-xl border border-dashed border-gray-300">
          <p class="text-gray-500">你还没有加入任何组织，快去输入邀请码吧！</p>
        </div>
      </section>

      <!-- Managed Workshops -->
      <section>
        <h3 class="text-lg font-medium leading-6 text-gray-900 mb-4 flex items-center">
          <span class="mr-2">👑</span> 我管理的 ({{ managedWorkshops.length }})
        </h3>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <WorkshopCard 
            v-for="workshop in managedWorkshops" 
            :key="workshop.id" 
            :workshop="workshop" 
          />
          
          <!-- Create New Placeholder Card -->
          <button 
            @click="handleCreate"
            class="group flex flex-col items-center justify-center h-full min-h-[280px] rounded-xl border-2 border-dashed border-gray-300 hover:border-indigo-500 hover:bg-indigo-50 transition-all duration-200 text-center p-6"
          >
            <div class="w-12 h-12 rounded-full bg-gray-100 group-hover:bg-indigo-100 flex items-center justify-center mb-4 transition-colors">
              <PlusIcon class="w-6 h-6 text-gray-400 group-hover:text-indigo-600" />
            </div>
            <h3 class="text-sm font-medium text-gray-900 group-hover:text-indigo-700">创建新工作坊</h3>
            <p class="mt-1 text-xs text-gray-500">发起一个新的 Vibe Coding 活动</p>
          </button>
        </div>
      </section>
    </div>
  </div>
</template>
