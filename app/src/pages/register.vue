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
const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMsg = ref('')

async function handleRegister() {
  loading.value = true
  errorMsg.value = ''
  try {
    await userStore.register(email.value, password.value, username.value)
    await navigateTo('/dashboard')
  } catch {
    errorMsg.value = t('register.error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthDoor @close="navigateTo('/')">
    <h2 class="mb-6 text-center font-serif text-2xl font-bold text-fg-primary">
      {{ t('register.title') }}
    </h2>

    <form class="space-y-4" @submit.prevent="handleRegister">
      <BaseInput v-model="email" type="email" :label="t('register.email')" required />
      <BaseInput v-model="username" type="text" :label="t('register.username')" required />
      <div>
        <BaseInput v-model="password" type="password" :label="t('register.password')" required />
        <p class="mt-1 text-xs text-fg-muted">{{ t('register.passwordHint') }}</p>
      </div>

      <p v-if="errorMsg" class="text-sm text-fg-danger">{{ errorMsg }}</p>

      <BaseButton type="submit" block :loading="loading">{{ t('register.submit') }}</BaseButton>
    </form>

    <p class="mt-6 text-center text-sm text-fg-muted">
      {{ t('register.haveAccount') }}
      <NuxtLink to="/login" class="font-medium text-seal hover:underline">
        {{ t('register.login') }}
      </NuxtLink>
    </p>
  </AuthDoor>
</template>
