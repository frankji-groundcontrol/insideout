<script setup lang="ts">
import { ref } from 'vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'

const demoInput = ref('')
const errorInput = ref('invalid-email')
</script>

<template>
  <div class="max-w-7xl mx-auto py-12 px-4 sm:px-6 lg:px-8">
    <div class="text-center mb-12">
      <h1 class="text-4xl font-extrabold text-gray-900 sm:text-5xl sm:tracking-tight lg:text-6xl">
        组件展示厅 🎨
      </h1>
      <p class="mt-5 max-w-xl mx-auto text-xl text-gray-500">
        这里展示了我们刚刚造好的基础零件。它们是构建整个应用的基石。
      </p>
    </div>

    <!-- 按钮展示区 -->
    <div class="bg-white shadow overflow-hidden sm:rounded-lg mb-8">
      <div class="px-4 py-5 sm:px-6 border-b border-gray-200">
        <h3 class="text-lg leading-6 font-medium text-gray-900">
          1. 通用按钮 (BaseButton)
        </h3>
        <p class="mt-1 max-w-2xl text-sm text-gray-500">
          支持多种颜色、尺寸和状态。
        </p>
      </div>
      <div class="px-4 py-5 sm:p-6 space-y-6">
        <!-- 颜色变体 -->
        <div>
          <h4 class="text-sm font-medium text-gray-500 mb-4 uppercase tracking-wider">颜色变体 (Variants)</h4>
          <div class="flex flex-wrap gap-4">
            <BaseButton variant="primary">Primary (主要)</BaseButton>
            <BaseButton variant="secondary">Secondary (次要)</BaseButton>
            <BaseButton variant="danger">Danger (危险)</BaseButton>
            <BaseButton variant="outline">Outline (描边)</BaseButton>
          </div>
        </div>

        <!-- 尺寸 -->
        <div>
          <h4 class="text-sm font-medium text-gray-500 mb-4 uppercase tracking-wider">尺寸 (Sizes)</h4>
          <div class="flex flex-wrap items-center gap-4">
            <BaseButton size="sm">Small (小)</BaseButton>
            <BaseButton size="md">Medium (中)</BaseButton>
            <BaseButton size="lg">Large (大)</BaseButton>
          </div>
        </div>

        <!-- 状态 -->
        <div>
          <h4 class="text-sm font-medium text-gray-500 mb-4 uppercase tracking-wider">状态 (States)</h4>
          <div class="flex flex-wrap gap-4">
            <BaseButton loading>Loading (加载中)</BaseButton>
            <BaseButton disabled>Disabled (禁用)</BaseButton>
          </div>
        </div>
      </div>
    </div>

    <!-- 输入框展示区 -->
    <div class="bg-white shadow overflow-hidden sm:rounded-lg">
      <div class="px-4 py-5 sm:px-6 border-b border-gray-200">
        <h3 class="text-lg leading-6 font-medium text-gray-900">
          2. 通用输入框 (BaseInput)
        </h3>
        <p class="mt-1 max-w-2xl text-sm text-gray-500">
          支持标签、占位符和错误提示。
        </p>
      </div>
      <div class="px-4 py-5 sm:p-6 space-y-6">
        <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-2">
          
          <!-- 正常输入框 -->
          <div>
            <BaseInput
              v-model="demoInput"
              label="用户名 (正常状态)"
              placeholder="请输入你的昵称"
            />
            <p class="mt-2 text-sm text-gray-500">
              当前输入: {{ demoInput || '(空)' }}
            </p>
          </div>

          <!-- 必填 & 报错 -->
          <div>
            <BaseInput
              v-model="errorInput"
              label="邮箱 (报错状态)"
              required
              error="请输入有效的邮箱地址"
            />
          </div>

        </div>
      </div>
    </div>

  </div>
</template>
