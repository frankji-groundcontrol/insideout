<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AuthDoor from '@/components/auth/AuthDoor.vue'
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
  <AuthDoor :greeting="t('login.welcome')" @close="navigateTo('/')">
    <form :aria-label="t('login.title')" class="space-y-4" @submit.prevent="handleLogin">
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
  </AuthDoor>
</template>
