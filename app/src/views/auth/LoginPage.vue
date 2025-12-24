<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/common/BaseButton.vue'
import { UserIcon, LockClosedIcon } from '@heroicons/vue/24/solid'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)

const handleLogin = async () => {
  loading.value = true
  // 模拟登录延迟
  setTimeout(() => {
    loading.value = false
    router.push('/')
  }, 1000)
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 p-4">
    <!-- Main Card -->
    <div
      class="bg-white w-full max-w-4xl rounded-[2rem] shadow-2xl overflow-hidden flex flex-col md:flex-row min-h-[600px]"
    >
      <!-- Left Side (Blue Area) -->
      <div
        class="w-full md:w-[40%] bg-[#6C63FF] p-12 flex flex-col justify-center items-center text-white relative overflow-hidden"
      >
        <!-- The curved shape effect -->
        <div
          class="absolute top-0 right-0 w-full h-full bg-[#6C63FF] z-0"
          style="border-radius: 0 0 0 0"
        ></div>
        <!-- Decorative Circle -->
        <div
          class="absolute -right-24 top-1/2 transform -translate-y-1/2 w-64 h-64 bg-white opacity-10 rounded-full"
        ></div>

        <div class="relative z-10 text-center">
          <h2 class="text-3xl font-bold mb-4">Hello, Welcome!</h2>
          <p class="mb-8 text-blue-100">Don't have an account?</p>
          <button
            class="px-8 py-2 border-2 border-white rounded-full text-white hover:bg-white hover:text-[#6C63FF] transition-colors duration-300 font-medium"
          >
            Register
          </button>
        </div>
      </div>

      <!-- Right Side (Form Area) -->
      <div class="w-full md:w-[60%] p-12 flex flex-col justify-center relative">
        <div class="max-w-md mx-auto w-full">
          <h2 class="text-3xl font-bold text-gray-800 mb-12 text-center">
            Login
          </h2>

          <form @submit.prevent="handleLogin" class="space-y-6">
            <!-- Username Input -->
            <div class="relative">
              <input
                v-model="username"
                type="text"
                placeholder="Username"
                class="w-full bg-gray-100 rounded-lg py-3 px-4 pl-4 pr-10 outline-none focus:ring-2 focus:ring-[#6C63FF] transition-all"
              />
              <UserIcon class="h-5 w-5 text-gray-400 absolute right-3 top-3.5" />
            </div>

            <!-- Password Input -->
            <div class="relative">
              <input
                v-model="password"
                type="password"
                placeholder="Password"
                class="w-full bg-gray-100 rounded-lg py-3 px-4 pl-4 pr-10 outline-none focus:ring-2 focus:ring-[#6C63FF] transition-all"
              />
              <LockClosedIcon
                class="h-5 w-5 text-gray-400 absolute right-3 top-3.5"
              />
            </div>

            <!-- Forgot Password -->
            <div class="text-center">
              <a href="#" class="text-sm text-gray-500 hover:text-[#6C63FF]">
                Forgot password?
              </a>
            </div>

            <!-- Login Button -->
            <BaseButton
              type="submit"
              :loading="loading"
              class="w-full py-3 rounded-lg bg-[#6C63FF] hover:bg-[#5a52d5] text-white font-semibold shadow-lg shadow-indigo-200"
            >
              Login
            </BaseButton>

            <!-- Social Login -->
            <div class="mt-8">
              <p class="text-center text-gray-400 text-sm mb-4">
                or login with social platforms
              </p>
              <div class="flex justify-center space-x-4">
                <!-- Google -->
                <button
                  type="button"
                  class="p-2 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  <img
                    src="https://www.svgrepo.com/show/475656/google-color.svg"
                    class="w-6 h-6"
                    alt="Google"
                  />
                </button>
                <!-- Facebook -->
                <button
                  type="button"
                  class="p-2 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  <img
                    src="https://www.svgrepo.com/show/475647/facebook-color.svg"
                    class="w-6 h-6"
                    alt="Facebook"
                  />
                </button>
                <!-- Github -->
                <button
                  type="button"
                  class="p-2 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  <img
                    src="https://www.svgrepo.com/show/475654/github-color.svg"
                    class="w-6 h-6"
                    alt="Github"
                  />
                </button>
                <!-- LinkedIn -->
                <button
                  type="button"
                  class="p-2 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
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
/* Custom rounded corner for the left panel to match the design */
.custom-curve {
  border-radius: 0;
}

@media (min-width: 768px) {
  .custom-curve {
    border-top-right-radius: 100px;
    border-bottom-right-radius: 100px;
    margin-right: -50px;
  }
  
  /* Make sure the blue panel content is above the curve background */
  .bg-\[\#6C63FF\] > div.relative {
    z-index: 20;
  }
}

@media (max-width: 767px) {
  /* On mobile, use a wave or bottom curve */
  .custom-curve {
    border-bottom-left-radius: 50px;
    border-bottom-right-radius: 50px;
    margin-bottom: -30px;
  }
}
</style>
