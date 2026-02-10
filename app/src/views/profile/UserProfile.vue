<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { UserCircleIcon } from '@heroicons/vue/24/outline'
import BaseButton from '@/components/common/BaseButton.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const { t } = useI18n()

const username = ref('')
const bio = ref('')
const keywordsText = ref('')
const avatarPreview = ref('')
const errorMsg = ref('')
const successMsg = ref('')

const user = computed(() => userStore.user)
const avatarDisplay = computed(() => avatarPreview.value || user.value?.avatar_url || '')

watch(
  user,
  (currentUser) => {
    username.value = currentUser?.username ?? ''
    bio.value = currentUser?.bio ?? ''
    keywordsText.value = (currentUser?.keywords ?? []).join(',')
  },
  { immediate: true }
)

const handleAvatarChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) {
    return
  }
  avatarPreview.value = URL.createObjectURL(file)
}

const handleSave = () => {
  const trimmedUsername = username.value.trim()
  if (!trimmedUsername) {
    errorMsg.value = t('profile.usernameRequired')
    successMsg.value = ''
    return
  }

  const keywords = keywordsText.value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)

  userStore.updateProfile({
    username: trimmedUsername,
    bio: bio.value.trim(),
    keywords,
  })

  errorMsg.value = ''
  successMsg.value = t('profile.saveSuccess')
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 p-4 dark:bg-gray-900">
    <div class="w-full max-w-3xl rounded-3xl bg-white p-6 shadow-2xl dark:bg-gray-800 sm:p-8">
      <h1 class="mb-6 text-2xl font-bold text-gray-900 dark:text-gray-100">{{ t('profile.title') }}</h1>

      <form data-testid="profile-form" class="space-y-6" @submit.prevent="handleSave">
        <div class="rounded-2xl border border-gray-200 p-4 dark:border-gray-700 sm:p-5">
          <p class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('profile.avatar') }}</p>
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div class="h-20 w-20 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700">
              <img
                v-if="avatarDisplay"
                :src="avatarDisplay"
                :alt="t('profile.avatar')"
                class="h-full w-full object-cover"
              />
              <UserCircleIcon v-else class="h-full w-full text-gray-400 dark:text-gray-300" />
            </div>
            <label
              class="inline-flex cursor-pointer items-center rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-600 dark:text-gray-100 dark:hover:bg-gray-700"
            >
              {{ t('profile.changeAvatar') }}
              <input
                data-testid="profile-avatar-input"
                type="file"
                accept="image/*"
                class="sr-only"
                @change="handleAvatarChange"
              />
            </label>
          </div>
        </div>

        <div>
          <label
            for="profile-username"
            class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {{ t('profile.username') }}
          </label>
          <input
            id="profile-username"
            data-testid="profile-username"
            v-model="username"
            type="text"
            class="block w-full rounded-md border border-gray-300 bg-gray-100 px-3 py-2 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:border-gray-700 dark:bg-gray-700 dark:text-gray-100"
          />
        </div>

        <div>
          <label for="profile-email" class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('profile.email') }}
          </label>
          <input
            id="profile-email"
            data-testid="profile-email"
            :value="user?.email ?? ''"
            type="email"
            readonly
            class="block w-full rounded-md border border-gray-300 bg-gray-200 px-3 py-2 text-gray-600 shadow-sm sm:text-sm dark:border-gray-700 dark:bg-gray-700 dark:text-gray-300"
          />
        </div>

        <div>
          <label for="profile-bio" class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('profile.bio') }}
          </label>
          <textarea
            id="profile-bio"
            data-testid="profile-bio"
            v-model="bio"
            :placeholder="t('profile.bioPlaceholder')"
            maxlength="200"
            rows="4"
            class="block w-full rounded-md border border-gray-300 bg-gray-100 px-3 py-2 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:border-gray-700 dark:bg-gray-700 dark:text-gray-100"
          ></textarea>
        </div>

        <div>
          <label
            for="profile-keywords"
            class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {{ t('profile.keywords') }}
          </label>
          <input
            id="profile-keywords"
            data-testid="profile-keywords"
            v-model="keywordsText"
            type="text"
            :placeholder="t('profile.keywordsPlaceholder')"
            class="block w-full rounded-md border border-gray-300 bg-gray-100 px-3 py-2 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:border-gray-700 dark:bg-gray-700 dark:text-gray-100"
          />
        </div>

        <p v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400">{{ errorMsg }}</p>
        <p v-if="successMsg" class="text-sm text-green-600 dark:text-green-400">{{ successMsg }}</p>

        <BaseButton type="submit" class="w-full sm:w-auto">{{ t('profile.save') }}</BaseButton>
      </form>
    </div>
  </div>
</template>
