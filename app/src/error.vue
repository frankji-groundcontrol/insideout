<script setup lang="ts">
import type { NuxtError } from '#app'

const props = defineProps<{
  error: NuxtError
}>()

// 返回首页并清除错误状态
const handleClear = () => clearError({ redirect: '/' })

const statusCode = computed(() => props.error?.statusCode ?? 500)
const message = computed(() => props.error?.message || '出错了')
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center gap-4 bg-gray-50 px-4 text-center text-gray-900 font-sans dark:bg-gray-900 dark:text-gray-100">
    <p class="text-6xl font-bold text-indigo-600 dark:text-indigo-400">{{ statusCode }}</p>
    <p class="text-lg text-gray-600 dark:text-gray-300">{{ message }}</p>
    <button
      type="button"
      class="inline-flex items-center justify-center rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
      @click="handleClear"
    >
      返回首页
    </button>
  </div>
</template>
