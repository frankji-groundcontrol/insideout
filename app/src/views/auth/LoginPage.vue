<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/common/BaseButton.vue'
import { UserIcon, LockClosedIcon } from '@heroicons/vue/24/solid'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()
const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMsg = ref('')
const { t } = useI18n()

const handleLogin = async () => {
  loading.value = true
  errorMsg.value = ''
  try {
    await userStore.login(username.value, password.value)
    router.push('/dashboard')
  } catch {
    errorMsg.value = t('login.error', 'Login failed')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 p-4 dark:bg-gray-900">
    <!-- 主卡片 -->
    <div
      class="flex min-h-[600px] w-full max-w-4xl flex-col overflow-hidden rounded-[2rem] bg-white shadow-2xl dark:bg-gray-800 md:flex-row"
    >
      <!-- 左侧品牌区 -->
      <div
        class="w-full md:w-[40%] bg-[#6C63FF] p-12 flex flex-col justify-center items-center text-white relative overflow-hidden"
      >
        <!-- 曲线背景 -->
        <div
          class="absolute top-0 right-0 w-full h-full bg-[#6C63FF] z-0"
          style="border-radius: 0 0 0 0"
        ></div>
        <!-- 装饰圆形 -->
        <div
          class="absolute -right-24 top-1/2 transform -translate-y-1/2 w-64 h-64 bg-white opacity-10 rounded-full"
        ></div>

        <div class="relative z-10 text-center">
          <h2 class="mb-4 text-3xl font-bold">{{ t('login.welcome') }}</h2>
          <p class="mb-8 text-blue-100">{{ t('login.noAccount') }}</p>
          <button
            class="px-8 py-2 border-2 border-white rounded-full text-white hover:bg-white hover:text-[#6C63FF] transition-colors duration-300 font-medium"
          >
            {{ t('login.register') }}
          </button>
        </div>
      </div>

      <!-- 右侧表单区 -->
      <div class="w-full md:w-[60%] p-12 flex flex-col justify-center relative">
        <div class="max-w-md mx-auto w-full">
          <h2 class="mb-12 text-center text-3xl font-bold text-gray-800 dark:text-gray-100">
            {{ t('login.title') }}
          </h2>

          <form @submit.prevent="handleLogin" class="space-y-6">
            <!-- 用户名输入 -->
            <div class="relative">
              <input
                v-model="username"
                type="text"
                :placeholder="t('login.username')"
                class="w-full rounded-lg bg-gray-100 py-3 pl-4 pr-10 outline-none transition-all focus:ring-2 focus:ring-[#6C63FF] dark:bg-gray-700 dark:text-gray-100"
              />
              <UserIcon class="absolute right-3 top-3.5 h-5 w-5 text-gray-400 dark:text-gray-300" />
            </div>

            <!-- 密码输入 -->
            <div class="relative">
              <input
                v-model="password"
                type="password"
                :placeholder="t('login.password')"
                class="w-full rounded-lg bg-gray-100 py-3 pl-4 pr-10 outline-none transition-all focus:ring-2 focus:ring-[#6C63FF] dark:bg-gray-700 dark:text-gray-100"
              />
              <LockClosedIcon
                class="absolute right-3 top-3.5 h-5 w-5 text-gray-400 dark:text-gray-300"
              />
            </div>

            <!-- 忘记密码 -->
            <div class="text-center">
              <a href="#" class="text-sm text-gray-500 hover:text-[#6C63FF] dark:text-gray-400">
                {{ t('login.forgotPassword') }}
              </a>
            </div>

            <!-- 登录按钮 -->
            <BaseButton
              type="submit"
              :loading="loading"
              class="w-full py-3 rounded-lg bg-[#6C63FF] hover:bg-[#5a52d5] text-white font-semibold shadow-lg shadow-indigo-200"
            >
              {{ t('login.loginButton') }}
            </BaseButton>

            <!-- 社交登录 -->
            <div class="mt-8">
              <p class="mb-4 text-center text-sm text-gray-400 dark:text-gray-500">
                {{ t('login.socialLogin') }}
              </p>
              <div class="flex justify-center space-x-4">
                <!-- 谷歌 -->
                <button
                  type="button"
                  class="rounded-lg border border-gray-200 p-2 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
                >
                  <img
                    src="https://www.svgrepo.com/show/475656/google-color.svg"
                    class="w-6 h-6"
                    alt="Google"
                  />
                </button>
                <!-- 脸书 -->
                <button
                  type="button"
                  class="rounded-lg border border-gray-200 p-2 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
                >
                  <img
                    src="https://www.svgrepo.com/show/475647/facebook-color.svg"
                    class="w-6 h-6"
                    alt="Facebook"
                  />
                </button>
                <!-- GitHub -->
                <button
                  type="button"
                  class="rounded-lg border border-gray-200 p-2 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
                >
                  <img
                    src="https://www.svgrepo.com/show/475654/github-color.svg"
                    class="w-6 h-6"
                    alt="Github"
                  />
                </button>
                <!-- 领英 -->
                <button
                  type="button"
                  class="rounded-lg border border-gray-200 p-2 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
                >
                  <img
                    src="https://www.svgrepo.com/show/475661/linkedin-color.svg"
                    class="w-6 h-6"
                    alt="LinkedIn"
                  />
                </button>
              </div>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 左侧面板圆角效果 */
.custom-curve {
  border-radius: 0;
}

@media (min-width: 768px) {
  .custom-curve {
    border-top-right-radius: 100px;
    border-bottom-right-radius: 100px;
    margin-right: -50px;
  }
  
  /* 确保蓝色面板内容在曲线背景之上 */
  .bg-\[\#6C63FF\] > div.relative {
    z-index: 20;
  }
}

@media (max-width: 767px) {
  /* 移动端使用底部曲线 */
  .custom-curve {
    border-bottom-left-radius: 50px;
    border-bottom-right-radius: 50px;
    margin-bottom: -30px;
  }
}
</style>
