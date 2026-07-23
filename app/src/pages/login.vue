<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import { useUserStore } from '@/stores/user'

definePageMeta({ public: true, layout: 'empty' })

const userStore = useUserStore()
const { t } = useI18n()

const email = ref('')
const password = ref('')
const loading = ref(false)
const errorMsg = ref('')

async function handleLogin() {
  loading.value = true
  errorMsg.value = ''
  try {
    await userStore.login(email.value, password.value)
    await navigateTo('/dashboard')
  } catch {
    errorMsg.value = t('login.error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-surface-base p-4">
    <div class="w-full max-w-sm">
      <!-- Ink & Seal door: vermilion seal stamp + serif wordmark -->
      <div class="mb-6 flex flex-col items-center text-center">
        <div class="mb-4 flex h-14 w-14 items-center justify-center rounded-card border-2 border-seal bg-surface-raised shadow-card">
          <span class="font-serif text-2xl font-bold text-seal">{{ t('nav.brand').charAt(0) }}</span>
        </div>
        <p class="font-serif text-xl font-semibold text-fg-primary">{{ t('nav.brand') }}</p>
        <p class="mt-1 text-sm text-fg-muted">{{ t('login.welcome') }}</p>
      </div>

      <div class="rounded-hero border border-stroke-subtle bg-surface-raised p-8 shadow-modal">
        <h1 class="mb-6 text-center font-serif text-2xl font-bold text-fg-primary">
          {{ t('login.title') }}
        </h1>

        <form class="space-y-4" @submit.prevent="handleLogin">
          <BaseInput v-model="email" type="email" :label="t('login.username')" required />
          <BaseInput v-model="password" type="password" :label="t('login.password')" required />

          <p v-if="errorMsg" class="text-sm text-fg-danger">{{ errorMsg }}</p>

          <BaseButton type="submit" block :loading="loading">{{ t('login.loginButton') }}</BaseButton>
        </form>

        <p class="mt-6 text-center text-sm text-fg-muted">
          {{ t('login.noAccount') }}
          <NuxtLink to="/register" class="font-medium text-seal hover:underline">
            {{ t('login.register') }}
          </NuxtLink>
        </p>
      </div>
    </div>
  </div>
</template>
