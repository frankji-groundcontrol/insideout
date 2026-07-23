<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { UserCircleIcon } from '@heroicons/vue/24/outline'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const { t } = useI18n()

const username = ref('')
const bio = ref('')
const keywordsText = ref('')
const avatarPreview = ref('')
const errorMsg = ref('')
const successMsg = ref('')
const saving = ref(false)

const user = computed(() => userStore.user)
const avatarDisplay = computed(() => avatarPreview.value || user.value?.avatarUrl || '')

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

const handleSave = async () => {
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

  saving.value = true
  try {
    await userStore.updateProfile({
      username: trimmedUsername,
      bio: bio.value.trim(),
      keywords,
    })
    errorMsg.value = ''
    successMsg.value = t('profile.saveSuccess')
  } catch {
    errorMsg.value = t('profile.usernameRequired')
    successMsg.value = ''
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <PageHeader :title="t('profile.title')" />

    <div class="max-w-2xl">
      <form data-testid="profile-form" class="space-y-6" @submit.prevent="handleSave">
        <!-- Avatar -->
        <div class="rounded-card border border-stroke-subtle bg-surface-raised p-5">
          <p class="mb-3 text-sm font-medium text-fg-secondary">{{ t('profile.avatar') }}</p>
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div class="h-20 w-20 overflow-hidden rounded-full border border-stroke-subtle bg-surface-sunken">
              <img
                v-if="avatarDisplay"
                :src="avatarDisplay"
                :alt="t('profile.avatar')"
                class="h-full w-full object-cover"
              />
              <UserCircleIcon v-else class="h-full w-full p-3 text-fg-muted" />
            </div>
            <div class="flex flex-col gap-1.5">
              <label
                class="inline-flex w-fit cursor-pointer items-center rounded-control border border-stroke-subtle px-3 py-2 text-sm font-medium text-fg-secondary transition-colors hover:bg-surface-sunken"
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
              <p class="text-xs text-fg-muted">{{ t('profile.avatarUploadComingSoon') }}</p>
            </div>
          </div>
        </div>

        <!-- Fields -->
        <div class="space-y-5 rounded-card border border-stroke-subtle bg-surface-raised p-5">
          <BaseInput
            v-model="username"
            data-testid="profile-username"
            :label="t('profile.username')"
            required
          />

          <BaseInput
            :model-value="user?.email ?? ''"
            data-testid="profile-email"
            :label="t('profile.email')"
            type="email"
            readonly
            class="text-fg-muted"
          />

          <div>
            <label for="profile-bio" class="mb-1 block text-sm font-medium text-fg-secondary">
              {{ t('profile.bio') }}
            </label>
            <textarea
              id="profile-bio"
              data-testid="profile-bio"
              v-model="bio"
              :placeholder="t('profile.bioPlaceholder')"
              maxlength="200"
              rows="4"
              class="block w-full rounded-control border border-stroke-subtle bg-surface-sunken px-3 py-2 text-fg-primary shadow-sm focus:border-stroke-focus focus:ring-stroke-focus sm:text-sm"
            ></textarea>
          </div>

          <BaseInput
            v-model="keywordsText"
            data-testid="profile-keywords"
            :label="t('profile.keywords')"
            :placeholder="t('profile.keywordsPlaceholder')"
          />
        </div>

        <p v-if="errorMsg" class="text-sm text-fg-danger">{{ errorMsg }}</p>
        <p v-if="successMsg" class="text-sm text-status-success-fg">{{ successMsg }}</p>

        <BaseButton type="submit" :loading="saving" class="w-full sm:w-auto">
          {{ t('profile.save') }}
        </BaseButton>
      </form>
    </div>
  </div>
</template>
