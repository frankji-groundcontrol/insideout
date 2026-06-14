<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import ThemeToggle from '../common/ThemeToggle.vue'
import LangToggle from '../common/LangToggle.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const { t } = useI18n()

const handleLogout = async () => {
  await userStore.logout()
  await navigateTo('/login')
}
</script>

<template>
  <nav class="border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900">
    <div class="w-full px-4 sm:px-6 lg:px-8">
      <div class="flex justify-between h-16">
        <!-- 品牌区 -->
        <div class="flex">
          <NuxtLink to="/" class="flex-shrink-0 flex items-center">
            <span class="text-xl font-bold text-indigo-600 dark:text-indigo-400">{{ t('nav.brand') }}</span>
          </NuxtLink>
        </div>

        <!-- 右侧操作 -->
        <div class="flex items-center space-x-3">
          <LangToggle />
          <ThemeToggle />

          <template v-if="!userStore.isAuthenticated">
            <NuxtLink
              to="/login"
              class="text-gray-500 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
            >
              {{ t('nav.login') }}
            </NuxtLink>
            <NuxtLink
              to="/login"
              class="inline-flex items-center justify-center rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              {{ t('nav.startJourney') }}
            </NuxtLink>
          </template>

          <template v-else>
            <NuxtLink
              to="/profile"
              class="inline-flex items-center gap-2 rounded-lg px-2 py-1 text-gray-700 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-gray-200 dark:hover:bg-gray-800 dark:hover:text-white"
            >
              <img
                v-if="userStore.user?.avatar_url"
                :src="userStore.user.avatar_url"
                :alt="userStore.user?.username ?? ''"
                class="h-7 w-7 rounded-full object-cover"
              />
              <span
                v-else
                class="inline-flex h-7 w-7 items-center justify-center rounded-full bg-indigo-100 text-xs font-semibold text-indigo-700 dark:bg-indigo-900 dark:text-indigo-200"
              >
                {{ (userStore.user?.username?.charAt(0) ?? 'U').toUpperCase() }}
              </span>
              <span class="text-sm font-medium">{{ userStore.user?.username ?? '' }}</span>
            </NuxtLink>
            <BaseButton variant="outline" size="sm" @click="handleLogout">
              {{ t('nav.logout') }}
            </BaseButton>
          </template>
        </div>
      </div>
    </div>
  </nav>
</template>
