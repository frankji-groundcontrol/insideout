<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink, useRouter } from 'vue-router'
import BaseButton from '../common/BaseButton.vue'
import ThemeToggle from '../common/ThemeToggle.vue'
import LangToggle from '../common/LangToggle.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const router = useRouter()
const { t } = useI18n()

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <nav class="border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
    <div class="w-full px-4 sm:px-6 lg:px-8">
      <div class="flex justify-between h-16">
        <!-- 品牌区 -->
        <div class="flex">
          <RouterLink to="/" class="flex-shrink-0 flex items-center">
            <span class="text-xl font-bold text-indigo-600 dark:text-indigo-400">{{ t('nav.brand') }}</span>
          </RouterLink>
        </div>

        <!-- 右侧操作 -->
        <div class="flex items-center space-x-3">
          <LangToggle />
          <ThemeToggle />

          <template v-if="!userStore.isAuthenticated">
            <RouterLink
              to="/login"
              class="text-gray-500 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
            >
              {{ t('nav.login') }}
            </RouterLink>
            <RouterLink
              to="/login"
              class="inline-flex items-center justify-center rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              {{ t('nav.startJourney') }}
            </RouterLink>
          </template>

          <template v-else>
            <span class="text-gray-700 dark:text-gray-200">{{ t('nav.greeting', { name: userStore.user?.username ?? '' }) }}</span>
            <BaseButton variant="outline" size="sm" @click="handleLogout">
              {{ t('nav.logout') }}
            </BaseButton>
          </template>
        </div>
      </div>
    </div>
  </nav>
</template>
