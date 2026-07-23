<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { useUserStore } from '@/stores/user'
import type { Workspace } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { PlusIcon, UserPlusIcon, RectangleGroupIcon } from '@heroicons/vue/24/outline'

const { t } = useI18n()
const userStore = useUserStore()

const loading = ref(true)
const workspaces = ref<Workspace[]>([])

const showCreate = ref(false)
const showJoin = ref(false)
const createTitle = ref('')
const createDescription = ref('')
const joinCode = ref('')
const formError = ref('')
const submitting = ref(false)

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return t('dashboard.greeting.morning')
  if (hour < 18) return t('dashboard.greeting.afternoon')
  return t('dashboard.greeting.evening')
})

async function loadWorkspaces() {
  loading.value = true
  try {
    workspaces.value = await useServices().workspace.list()
  } finally {
    loading.value = false
  }
}

onMounted(loadWorkspaces)

function openCreate() {
  formError.value = ''
  showCreate.value = true
}
function openJoin() {
  formError.value = ''
  showJoin.value = true
}

async function handleCreate() {
  if (!createTitle.value.trim()) {
    formError.value = t('workspace.createNamePlaceholder')
    return
  }
  submitting.value = true
  formError.value = ''
  try {
    const ws = await useServices().workspace.create(createTitle.value.trim(), createDescription.value)
    showCreate.value = false
    createTitle.value = ''
    createDescription.value = ''
    await navigateTo(`/workspace/${ws.id}`)
  } catch {
    formError.value = t('workspace.createTitle')
  } finally {
    submitting.value = false
  }
}

async function handleJoin() {
  if (!/^\d{6}$/.test(joinCode.value.trim())) {
    formError.value = t('workspace.joinCodePlaceholder')
    return
  }
  submitting.value = true
  formError.value = ''
  try {
    const ws = await useServices().workspace.join(joinCode.value.trim())
    showJoin.value = false
    joinCode.value = ''
    await navigateTo(`/workspace/${ws.id}`)
  } catch {
    formError.value = t('workspace.joinCodePlaceholder')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <PageHeader
      :title="`${greeting}, ${userStore.user?.username || t('dashboard.friend')}`"
      :subtitle="t('workspace.joinedTitle')"
    >
      <template #actions>
        <BaseButton variant="outline" @click="openJoin">
          <UserPlusIcon class="-ml-1 mr-2 h-5 w-5" />
          {{ t('workspace.join') }}
        </BaseButton>
        <BaseButton @click="openCreate">
          <PlusIcon class="-ml-1 mr-2 h-5 w-5" />
          {{ t('workspace.createTitle') }}
        </BaseButton>
      </template>
    </PageHeader>

    <div v-if="loading" class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="i in 3" :key="i" class="h-40 animate-pulse rounded-card bg-surface-sunken" />
    </div>

    <BaseEmptyState v-else-if="workspaces.length === 0" :title="t('workspace.empty')">
      <template #icon><RectangleGroupIcon class="h-6 w-6" /></template>
      <div class="flex justify-center gap-3">
        <BaseButton variant="outline" @click="openJoin">{{ t('workspace.join') }}</BaseButton>
        <BaseButton @click="openCreate">{{ t('workspace.createTitle') }}</BaseButton>
      </div>
    </BaseEmptyState>

    <div v-else class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink v-for="ws in workspaces" :key="ws.id" :to="`/workspace/${ws.id}`">
        <BaseCard interactive>
          <div class="flex items-start justify-between gap-3">
            <h3 class="text-lg font-semibold text-fg-primary">{{ ws.title }}</h3>
            <BaseBadge v-if="ws.myRole === 'admin'" tone="warn">{{ t('workspace.role.admin') }}</BaseBadge>
          </div>
          <p class="mt-1 line-clamp-2 min-h-[2.5rem] text-sm text-fg-muted">{{ ws.description }}</p>
          <div class="mt-4 flex items-center justify-between text-xs text-fg-muted">
            <span>{{ t('workspace.members', { count: ws.memberCount }) }}</span>
            <span class="tracking-widest">{{ ws.code }}</span>
          </div>
        </BaseCard>
      </NuxtLink>
    </div>

    <!-- Create workspace -->
    <BaseModal :open="showCreate" :title="t('workspace.createTitle')" @close="showCreate = false">
      <form id="create-ws-form" class="space-y-4" @submit.prevent="handleCreate">
        <BaseInput v-model="createTitle" :label="t('workspace.createNamePlaceholder')" required />
        <BaseInput v-model="createDescription" :label="t('workspace.createDescPlaceholder')" />
        <p v-if="formError" class="text-sm text-fg-danger">{{ formError }}</p>
      </form>
      <template #footer>
        <BaseButton variant="outline" @click="showCreate = false">{{ t('common.cancel') }}</BaseButton>
        <BaseButton type="submit" form="create-ws-form" :loading="submitting">{{ t('workspace.create') }}</BaseButton>
      </template>
    </BaseModal>

    <!-- Join workspace -->
    <BaseModal :open="showJoin" :title="t('workspace.joinTitle')" size="sm" @close="showJoin = false">
      <form id="join-ws-form" class="space-y-4" @submit.prevent="handleJoin">
        <BaseInput v-model="joinCode" :label="t('workspace.joinCodePlaceholder')" required />
        <p v-if="formError" class="text-sm text-fg-danger">{{ formError }}</p>
      </form>
      <template #footer>
        <BaseButton variant="outline" @click="showJoin = false">{{ t('common.cancel') }}</BaseButton>
        <BaseButton type="submit" form="join-ws-form" :loading="submitting">{{ t('workspace.join') }}</BaseButton>
      </template>
    </BaseModal>
  </div>
</template>
