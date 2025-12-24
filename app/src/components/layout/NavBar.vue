<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import BaseButton from '../common/BaseButton.vue'

// 模拟登录状态，后面会接 Pinia
const isLoggedIn = ref(false)

const toggleLogin = () => {
  isLoggedIn.value = !isLoggedIn.value
}
</script>

<template>
  <nav class="bg-white border-b border-gray-200">
    <div class="w-full px-4 sm:px-6 lg:px-8">
      <div class="flex justify-between h-16">
        <!-- Logo -->
        <div class="flex">
          <RouterLink to="/" class="flex-shrink-0 flex items-center">
            <span class="text-xl font-bold text-indigo-600">🌯 卷了么</span>
          </RouterLink>
        </div>

        <!-- Right Side -->
        <div class="flex items-center space-x-4">
          <template v-if="!isLoggedIn">
            <RouterLink to="/login" class="text-gray-500 hover:text-gray-900">
              登录
            </RouterLink>
            <BaseButton size="sm" @click="toggleLogin">开始旅程</BaseButton>
          </template>

          <template v-else>
            <span class="text-gray-700">你好, 坏胖胖</span>
            <BaseButton variant="outline" size="sm" @click="toggleLogin">
              退出
            </BaseButton>
          </template>
        </div>
      </div>
    </div>
  </nav>
</template>
