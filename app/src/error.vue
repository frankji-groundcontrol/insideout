<script setup lang="ts">
import type { NuxtError } from '#app'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'

const props = defineProps<{
  error: NuxtError
}>()
const { t } = useI18n()
const userStore = useUserStore()

const statusCode = computed(() => props.error?.statusCode ?? 500)
const is404 = computed(() => statusCode.value === 404)

const title = computed(() => (is404.value ? t('error.notFoundTitle') : t('error.genericTitle')))
const body = computed(() => (is404.value ? t('error.notFoundBody') : t('error.genericBody')))

// 返回首页并清除错误状态（已登录用户回工作台，访客回首页）
const homePath = computed(() => (userStore.isAuthenticated ? '/dashboard' : '/'))
const handleClear = () => clearError({ redirect: homePath.value })
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center bg-surface-base px-4 py-16 text-center font-sans">
    <div class="flex h-24 w-24 items-center justify-center rounded-card border-2 border-seal bg-surface-raised shadow-card">
      <span class="font-serif text-4xl font-bold leading-none text-seal">{{ statusCode }}</span>
    </div>
    <h1 class="mt-8 font-serif text-2xl font-semibold text-fg-primary">{{ title }}</h1>
    <p class="mt-2 max-w-md text-sm leading-relaxed text-fg-muted">{{ body }}</p>
    <button
      type="button"
      class="mt-8 inline-flex items-center justify-center rounded-control bg-btn px-4 py-2 text-sm font-medium text-btn-fg transition-colors hover:opacity-90"
      @click="handleClear"
    >
      {{ t('error.backHome') }}
    </button>
  </div>
</template>
